package mem

import (
	"reflect"
	"unsafe"
)

// Memset sets size bytes at the given address to the supplied value. The implementation
// is based on bytes.Repeat; instead of using a for loop, this function uses
// log2(size) copy calls which should give us a speed boost as page addresses
// are always aligned.
func Memset(addr uintptr, value byte, size Size) {
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
	for index := Size(1); index < size; index *= 2 {
		copy(target[index:], target[:index])
	}
}
