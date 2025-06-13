// package futex implements wrappers for futex system
// call. Wait and Wake are implemented with single FUTEX_WAIT or 
// FUTEX_WAKE operations. Other operation and optimizations
// like FUTEX_PRIVATE_FLAG are not implemented.
// Also, Wake only wakes one waiter.
package futex

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	FUTEX_WAIT = 0
	FUTEX_WAKE = 1
)

func Wait(addr unsafe.Pointer, val uint32) error {
	_, _, e := unix.Syscall6(unix.SYS_FUTEX,
		uintptr(addr),
		uintptr(FUTEX_WAIT),
		uintptr(val),      
		0, 0, 0)          
	if e != 0 {
		return e
	}
	return nil
}

// wakes only one waiter
func Wake(addr unsafe.Pointer) error {
	const N_WAITERS = 1
	_, _, e := unix.Syscall6(unix.SYS_FUTEX,
		uintptr(addr),
		uintptr(FUTEX_WAKE),
		N_WAITERS, 0, 0, 0)
	return e
}
