package kernel

import (
	"reflect"
	"unsafe"
)

// Memset sets size bytes at the given address to the supplied value. The implementation
// is based on bytes.Repeat; instead of using a for loop, this function uses
// log2(size) copy calls which should give us a speed boost as page addresses
// are always aligned.
func Memset(addr uintptr, value byte, size uintptr) {
	if size == 0 {
		return
	}

	// overlay a slice on top of this address region
	target := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(size),
		Cap:  int(size),
		Data: addr,
	}))

	// Set first element and make log2(size) optimized copies
	target[0] = value
	for index := uintptr(1); index < size; index *= 2 {
		copy(target[index:], target[:index])
	}
}

// Memcopy copies size bytes from src to dst.
func Memcopy(src, dst uintptr, size uintptr) {
	if size == 0 {
		return
	}

	srcSlice := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(size),
		Cap:  int(size),
		Data: src,
	}))
	dstSlice := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(size),
		Cap:  int(size),
		Data: dst,
	}))

	copy(dstSlice, srcSlice)
}
