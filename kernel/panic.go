package kernel

import (
	"github.com/achilleasa/gopher-os/kernel/cpu"
	"github.com/achilleasa/gopher-os/kernel/kfmt/early"
)

var (
	// cpuHaltFn is mocked by tests and is automatically inlined by the compiler.
	cpuHaltFn = cpu.Halt

	errRuntimePanic = &Error{Module: "rt", Message: "unknown cause"}
)

// Panic outputs the supplied error (if not nil) to the console and halts the
// CPU. Calls to Panic never return. Panic also works as a redirection target
// for calls to panic() (resolved via runtime.gopanic)
//go:redirect-from runtime.gopanic
func Panic(e interface{}) {
	var err *Error

	switch t := e.(type) {
	case *Error:
		err = t
	case string:
		panicString(t)
		return
	case error:
		errRuntimePanic.Message = t.Error()
		err = errRuntimePanic
	}

	early.Printf("\n-----------------------------------\n")
	if err != nil {
		early.Printf("[%s] unrecoverable error: %s\n", err.Module, err.Message)
	}
	early.Printf("*** kernel panic: system halted ***")
	early.Printf("\n-----------------------------------\n")

	cpuHaltFn()
}

// panicString serves as a redirect target for runtime.throw
//go:redirect-from runtime.throw
func panicString(msg string) {
	errRuntimePanic.Message = msg
	Panic(errRuntimePanic)
}
