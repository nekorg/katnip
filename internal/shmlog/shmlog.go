package shmlog

import (
	"io"
	"sync/atomic"
	"unsafe"

	"github.com/codelif/katnip/internal/futex"
	"golang.org/x/sys/unix"
)

type _ring_header struct {
	head uint32
	tail uint32
}

const _BUF_HEAD_SIZE = unsafe.Sizeof(_ring_header{})
const _BUF_DATA_SIZE = 1 << 20 // should be less than 2^32 (since head/tail are 32bit ints)
const _BUF_SIZE = _BUF_HEAD_SIZE + _BUF_DATA_SIZE
const _BUF_DATA_MASK = _BUF_DATA_SIZE - 1

type Logger struct {
	Fd int
}

func New() (*Logger, error) {
	fd, err := unix.MemfdCreate("log", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, err
	}

	err = unix.Ftruncate(fd, int64(_BUF_SIZE))
	if err != nil {
		unix.Close(fd)
		return nil, err
	}

	return &Logger{fd}, nil
}

type Reader struct {
	buf      []byte
	buf_head *uint32
	buf_tail *uint32
	buf_d    []byte // buffer payload
}

func (l *Logger) NewReader() (io.Reader, error) {
	buf, err := unix.Mmap(l.Fd, 0, int(_BUF_SIZE), unix.PROT_WRITE|unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	h := (*_ring_header)(unsafe.Pointer(&buf[0]))
	return &Reader{buf: buf, buf_d: buf[_BUF_HEAD_SIZE:], buf_head: &h.head, buf_tail: &h.tail}, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
    for n == 0 {
        head := atomic.LoadUint32(r.buf_head)
        tail := atomic.LoadUint32(r.buf_tail)

        avail := head-tail
        if avail == 0 {
            futex.Wait(unsafe.Pointer(r.buf_head), head)
            continue
        }

        chunk := int(avail)

        wrap := int(_BUF_DATA_SIZE - (tail & _BUF_DATA_MASK))
        if chunk > wrap {
            chunk = wrap
        }
        remain := len(p) - n
        if chunk > remain {
            chunk = remain
        }

        start := int(tail & _BUF_DATA_MASK)
        copy(p[n:n+chunk], r.buf_d[start:start+chunk])

        atomic.AddUint32(r.buf_tail, uint32(chunk))
        futex.Wake(unsafe.Pointer(r.buf_tail))
        n += chunk
    }

    return n, nil
}

type Writer struct {
	buf      []byte
	buf_head *uint32
	buf_tail *uint32
	buf_d    []byte // buffer payload
}

func (l *Logger) NewWriter() (io.Writer, error) {
	buf, err := unix.Mmap(l.Fd, 0, int(_BUF_SIZE), unix.PROT_WRITE|unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	h := (*_ring_header)(unsafe.Pointer(&buf[0]))
	return &Writer{buf: buf, buf_d: buf[_BUF_HEAD_SIZE:], buf_head: &h.head, buf_tail: &h.tail}, nil
}

func (w *Writer) Write(p []byte) (n int, err error) {
    total := len(p)

    for n < total {
        head := atomic.LoadUint32(w.buf_head)
        tail := atomic.LoadUint32(w.buf_tail)

        free := _BUF_DATA_SIZE - (head - tail)
        if free == 0 {
            futex.Wait(unsafe.Pointer(w.buf_tail), tail)
            continue
        }

        chunk := int(free)
        endOfBuf := int(_BUF_DATA_SIZE - (head&_BUF_DATA_MASK))
        if chunk > endOfBuf {
            chunk = endOfBuf
        }
        if chunk > total-n {
            chunk = total - n
        }

        dst := w.buf_d[head&_BUF_DATA_MASK : (head&_BUF_DATA_MASK)+uint32(chunk)]
        copy(dst, p[n:n+chunk])
        atomic.AddUint32(w.buf_head, uint32(chunk))
        futex.Wake(unsafe.Pointer(w.buf_head))
        n += chunk
    }
    return n, nil
}
