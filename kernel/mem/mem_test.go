package mem

import (
	"testing"
	"unsafe"
)

func TestMemset(t *testing.T) {
	// memset with a 0 size should be a no-op
	Memset(uintptr(0), 0x00, 0)

	for pageCount := uint32(1); pageCount <= 10; pageCount++ {
		buf := make([]byte, PageSize<<pageCount)
		for i := 0; i < len(buf); i++ {
			buf[i] = 0xFE
		}

		addr := uintptr(unsafe.Pointer(&buf[0]))
		Memset(addr, 0x00, Size(len(buf)))

		for i := 0; i < len(buf); i++ {
			if got := buf[i]; got != 0x00 {
				t.Errorf("[block with %d pages] expected byte: %d to be 0x00; got 0x%x", pageCount, i, got)
			}
		}
	}
}

func TestMemcopy(t *testing.T) {
	// memcopy with a 0 size should be a no-op
	Memcopy(uintptr(0), uintptr(0), 0)

	var (
		src = make([]byte, PageSize)
		dst = make([]byte, PageSize)
	)
	for i := 0; i < len(src); i++ {
		src[i] = byte(i % 256)
	}

	Memcopy(
		uintptr(unsafe.Pointer(&src[0])),
		uintptr(unsafe.Pointer(&dst[0])),
		PageSize,
	)

	for i := 0; i < len(src); i++ {
		if got := dst[i]; got != src[i] {
			t.Errorf("value mismatch between src and dst at index %d", i)
		}
	}
}
