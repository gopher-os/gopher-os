package kmain

import (
	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm/allocator"
)

// Kmain is the only Go symbol that is visible (exported) from the rt0 initialization
// code. This function is invoked by the rt0 assembly code after setting up the GDT
// and setting up a a minimal g0 struct that allows Go code using the 4K stack
// allocated by the assembly code.
//
// The rt0 code passes the address of the multiboot info payload provided by the
// bootloader as well as the physical addresses for the kernel start/end.
//
// Kmain is not expected to return. If it does, the rt0 code will halt the CPU.
//
//go:noinline
func Kmain(multibootInfoPtr, kernelStart, kernelEnd uintptr) {
	multiboot.SetInfoPtr(multibootInfoPtr)

	hal.InitTerminal()
	hal.ActiveTerminal.Clear()

	if err := allocator.Init(kernelStart, kernelEnd); err != nil {
		early.Printf("[%s] error: %s\n", err.Module, err.Message)
	}
}
