package mem

import "github.com/achilleasa/gopher-os/kernel/errors"

var (
	ErrOutOfMemory = errors.KernelError("out of memory")
)
