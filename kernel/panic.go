package kernel

import (
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
)

var (
	// cpuHaltFn is mocked by tests and is automatically inlined by the compiler.
	cpuHaltFn = cpu.Halt
)

// Panic outputs the supplied error (if not nil) to the console and halts the
// CPU. Calls to Panic never return.
func Panic(err *Error) {
	early.Printf("\n-----------------------------------\n")
	if err != nil {
		early.Printf("[%s] unrecoverable error: %s\n", err.Module, err.Message)
	}
	early.Printf("*** kernel panic: system halted ***")
	early.Printf("\n-----------------------------------\n")

	cpuHaltFn()
}
