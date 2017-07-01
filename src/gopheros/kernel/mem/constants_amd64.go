// +build amd64

package mem

const (
	// PointerShift is equal to log2(unsafe.Sizeof(uintptr)). The pointer
	// size for this architecture is defined as (1 << PointerShift).
	PointerShift = 3

	// PageShift is equal to log2(PageSize). This constant is used when
	// we need to convert a physical address to a page number (shift right by PageShift)
	// and vice-versa.
	PageShift = 12

	// PageSize defines the system's page size in bytes.
	PageSize = Size(1 << PageShift)
)
