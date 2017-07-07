package kfmt

import (
	"bytes"
	"errors"
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"testing"
)

func TestPanic(t *testing.T) {
	defer func() {
		cpuHaltFn = cpu.Halt
		SetOutputSink(nil)
	}()

	var buf bytes.Buffer
	SetOutputSink(&buf)

	var cpuHaltCalled bool
	cpuHaltFn = func() {
		cpuHaltCalled = true
	}

	t.Run("with *kernel.Error", func(t *testing.T) {
		cpuHaltCalled = false
		buf.Reset()
		err := &kernel.Error{Module: "test", Message: "panic test"}

		Panic(err)

		exp := "\n-----------------------------------\n[test] unrecoverable error: panic test\n*** kernel panic: system halted ***\n-----------------------------------\n"

		if got := buf.String(); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})

	t.Run("with error", func(t *testing.T) {
		cpuHaltCalled = false
		buf.Reset()
		err := errors.New("go error")

		Panic(err)

		exp := "\n-----------------------------------\n[rt] unrecoverable error: go error\n*** kernel panic: system halted ***\n-----------------------------------\n"

		if got := buf.String(); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})

	t.Run("with string", func(t *testing.T) {
		cpuHaltCalled = false
		buf.Reset()
		err := "string error"

		Panic(err)

		exp := "\n-----------------------------------\n[rt] unrecoverable error: string error\n*** kernel panic: system halted ***\n-----------------------------------\n"

		if got := buf.String(); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})

	t.Run("without error", func(t *testing.T) {
		cpuHaltCalled = false
		buf.Reset()

		Panic(nil)

		exp := "\n-----------------------------------\n*** kernel panic: system halted ***\n-----------------------------------\n"

		if got := buf.String(); got != exp {
			t.Fatalf("expected to get:\n%q\ngot:\n%q", exp, got)
		}

		if !cpuHaltCalled {
			t.Fatal("expected cpu.Halt() to be called by Panic")
		}
	})
}
