// +build amd64

package mem

const (
	// PageShift is equal to log2(PageSize). This constant is used when
	// we need to convert a physical address to a page number (shift right by PageShift)
	// and vice-versa.
	PageShift = 12

	// PageSize defines the system's page size in bytes.
	PageSize = Size(1 << PageShift)
)
