package kernel

import (
	_ "unsafe" // required for go:linkname

	"github.com/achilleasa/gopher-os/kernel/hal"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
)

// Kmain is the only Go symbol that is visible (exported) from the rt0 initialization
// code. This function is invoked by the rt0 assembly code after setting up the GDT
// and setting up a a minimal g0 struct that allows Go code using the 4K stack
// allocated by the assembly code.
//
// The rt0 code passes the address of the multiboot info payload provided by the
// bootloader.
//
// Kmain is not expected to return. If it does, the rt0 code will halt the CPU.
//
//go:noinline
func Kmain(multibootInfoPtr uintptr) {
	multiboot.SetInfoPtr(multibootInfoPtr)

	// Initialize and clear the terminal
	hal.InitTerminal()
	hal.ActiveTerminal.Clear()
	early.Printf("Starting gopher-os\n")

	// Prevent Kmain from returning
	for {
	}
}
