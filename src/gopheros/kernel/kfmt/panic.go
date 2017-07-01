package kfmt

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
)

var (
	// cpuHaltFn is mocked by tests and is automatically inlined by the compiler.
	cpuHaltFn = cpu.Halt

	errRuntimePanic = &kernel.Error{Module: "rt", Message: "unknown cause"}
)

// Panic outputs the supplied error (if not nil) to the console and halts the
// CPU. Calls to Panic never return. Panic also works as a redirection target
// for calls to panic() (resolved via runtime.gopanic)
//go:redirect-from runtime.gopanic
func Panic(e interface{}) {
	var err *kernel.Error

	switch t := e.(type) {
	case *kernel.Error:
		err = t
	case string:
		panicString(t)
		return
	case error:
		errRuntimePanic.Message = t.Error()
		err = errRuntimePanic
	}

	Printf("\n-----------------------------------\n")
	if err != nil {
		Printf("[%s] unrecoverable error: %s\n", err.Module, err.Message)
	}
	Printf("*** kernel panic: system halted ***")
	Printf("\n-----------------------------------\n")

	cpuHaltFn()
}

// panicString serves as a redirect target for runtime.throw
//go:redirect-from runtime.throw
func panicString(msg string) {
	errRuntimePanic.Message = msg
	Panic(errRuntimePanic)
}
