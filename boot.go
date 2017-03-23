package main

import "github.com/achilleasa/gopher-os/kernel"

// main is the only Go symbol that is visible (exported) from the rt0 initialization
// code. This function works as a trampoline for calling the actual kernel entrypoint
// (kernel.Kmain) and its intentionally defined to prevent the Go compiler from
// optimizing away the actual kernel code as its not aware of the presence of the
// rt0 code.
//
// The main function is invoked by the rt0 assembly code after setting up the GDT
// and setting up a a minimal g0 struct that allows Go code using the 4K stack
// allocated by the assembly code.
//
// main is not expected to return. If it does, the rt0 code will halt the CPU.
func main() {
	kernel.Kmain()
}
