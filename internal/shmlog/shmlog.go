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
	fd int
}

func NewSharedMem() (*Logger, error) {
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
	buf, err := unix.Mmap(l.fd, 0, int(_BUF_SIZE), unix.PROT_WRITE|unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	h := (*_ring_header)(unsafe.Pointer(&buf[0]))
	return &Reader{buf: buf, buf_d: buf[_BUF_HEAD_SIZE:], buf_head: &h.head, buf_tail: &h.tail}, nil
}

func (r *Reader) Read(b []byte) (n int, err error) {
	head := atomic.LoadUint32(r.buf_head)
	tail := atomic.LoadUint32(r.buf_tail)

	for {
		if head == tail {
			// buffer empty
			futex.Wait(unsafe.Pointer(r.buf_head), head)
			continue
		}
		break
	}

	for n = 0; n < len(b) || head-tail > 0; n++ {
		b[n] = r.buf_d[tail&_BUF_DATA_MASK]
		atomic.AddUint32(r.buf_tail, 1)
		head = atomic.LoadUint32(r.buf_head)
		tail = atomic.LoadUint32(r.buf_tail)
	}

	futex.Wake(unsafe.Pointer(r.buf_head))
	return n, nil
}

type Writer struct {
	buf      []byte
	buf_head *uint32
	buf_tail *uint32
	buf_d    []byte // buffer payload
}

func (l *Logger) NewWriter() (io.Writer, error) {
	buf, err := unix.Mmap(l.fd, 0, int(_BUF_SIZE), unix.PROT_WRITE|unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	h := (*_ring_header)(unsafe.Pointer(&buf[0]))
	return &Writer{buf: buf, buf_d: buf[_BUF_HEAD_SIZE:], buf_head: &h.head, buf_tail: &h.tail}, nil
}

func (w *Writer) Write(p []byte) (n int, err error) {
	head := atomic.LoadUint32(w.buf_head)
	tail := atomic.LoadUint32(w.buf_tail)

	for {
		if head+1 == tail {
			futex.Wait(unsafe.Pointer(w.buf_tail), tail)
			continue
		}
		break
	}

	for n = 0; n < len(p) || head+1 != tail; n++ {
		w.buf[head&_BUF_DATA_MASK] = p[n]

		atomic.AddUint32(w.buf_head, 1)
		head = atomic.LoadUint32(w.buf_head)
		tail = atomic.LoadUint32(w.buf_tail)
	}
	futex.Wake(unsafe.Pointer(w.buf_head))

	return n, err
}
