package mm

const (
	// PointerShift is equal to log2(unsafe.Sizeof(uintptr)). The pointer
	// size for this architecture is defined as (1 << PointerShift).
	PointerShift = uintptr(3)

	// PageShift is equal to log2(PageSize). This constant is used when
	// we need to convert a physical address to a page number (shift right by PageShift)
	// and vice-versa.
	PageShift = uintptr(12)

	// PageSize defines the system's page size in bytes.
	PageSize = uintptr(1 << PageShift)
)
