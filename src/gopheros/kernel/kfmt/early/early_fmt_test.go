package early

import (
	"bytes"
	"gopheros/kernel/driver/tty"
	"gopheros/kernel/driver/video/console"
	"gopheros/kernel/hal"
	"testing"
	"unsafe"
)

func TestPrintf(t *testing.T) {
	origTerm := hal.ActiveTerminal
	defer func() {
		hal.ActiveTerminal = origTerm
	}()

	// mute vet warnings about malformed printf formatting strings
	printfn := Printf

	ega := &console.Ega{}
	fb := make([]uint8, 160*25)
	ega.Init(80, 25, uintptr(unsafe.Pointer(&fb[0])))

	vt := &tty.Vt{}
	vt.AttachTo(ega)
	hal.ActiveTerminal = vt

	specs := []struct {
		fn        func()
		expOutput string
	}{
		{
			func() { printfn("no args") },
			"no args",
		},
		// bool values
		{
			func() { printfn("%t", true) },
			"true",
		},
		{
			func() { printfn("%41t", false) },
			"false",
		},
		// strings and byte slices
		{
			func() { printfn("%s arg", "STRING") },
			"STRING arg",
		},
		{
			func() { printfn("%s arg", []byte("BYTE SLICE")) },
			"BYTE SLICE arg",
		},
		{
			func() { printfn("'%4s' arg with padding", "ABC") },
			"' ABC' arg with padding",
		},
		{
			func() { printfn("'%4s' arg longer than padding", "ABCDE") },
			"'ABCDE' arg longer than padding",
		},
		// uints
		{
			func() { printfn("uint arg: %d", uint8(10)) },
			"uint arg: 10",
		},
		{
			func() { printfn("uint arg: %o", uint16(0777)) },
			"uint arg: 777",
		},
		{
			func() { printfn("uint arg: 0x%x", uint32(0xbadf00d)) },
			"uint arg: 0xbadf00d",
		},
		{
			func() { printfn("uint arg with padding: '%10d'", uint64(123)) },
			"uint arg with padding: '       123'",
		},
		{
			func() { printfn("uint arg with padding: '%4o'", uint64(0777)) },
			"uint arg with padding: '0777'",
		},
		{
			func() { printfn("uint arg with padding: '0x%10x'", uint64(0xbadf00d)) },
			"uint arg with padding: '0x000badf00d'",
		},
		{
			func() { printfn("uint arg longer than padding: '0x%5x'", int64(0xbadf00d)) },
			"uint arg longer than padding: '0xbadf00d'",
		},
		// pointers
		{
			func() { printfn("uintptr 0x%x", uintptr(0xb8000)) },
			"uintptr 0xb8000",
		},
		// ints

		{
			func() { printfn("int arg: %d", int8(-10)) },
			"int arg: -10",
		},
		{
			func() { printfn("int arg: %o", int16(0777)) },
			"int arg: 777",
		},
		{
			func() { printfn("int arg: %x", int32(-0xbadf00d)) },
			"int arg: -badf00d",
		},
		{
			func() { printfn("int arg with padding: '%10d'", int64(-12345678)) },
			"int arg with padding: ' -12345678'",
		},
		{
			func() { printfn("int arg with padding: '%10d'", int64(-123456789)) },
			"int arg with padding: '-123456789'",
		},
		{
			func() { printfn("int arg with padding: '%10d'", int64(-1234567890)) },
			"int arg with padding: '-1234567890'",
		},
		{
			func() { printfn("int arg longer than padding: '%5x'", int(-0xbadf00d)) },
			"int arg longer than padding: '-badf00d'",
		},
		// multiple arguments
		{
			func() { printfn("%%%s%d%t", "foo", 123, true) },
			`%foo123true`,
		},
		// errors
		{
			func() { printfn("more args", "foo", "bar", "baz") },
			`more args%!(EXTRA)%!(EXTRA)%!(EXTRA)`,
		},
		{
			func() { printfn("missing args %s") },
			`missing args (MISSING)`,
		},
		{
			func() { printfn("bad verb %Q") },
			`bad verb %!(NOVERB)`,
		},
		{
			func() { printfn("not bool %t", "foo") },
			`not bool %!(WRONGTYPE)`,
		},
		{
			func() { printfn("not int %d", "foo") },
			`not int %!(WRONGTYPE)`,
		},
		{
			func() { printfn("not string %s", 123) },
			`not string %!(WRONGTYPE)`,
		},
	}

	for specIndex, spec := range specs {
		for index := 0; index < len(fb); index++ {
			fb[index] = 0
		}
		vt.SetPosition(0, 0)

		spec.fn()

		var buf bytes.Buffer
		for index := 0; ; index += 2 {
			if fb[index] == 0 {
				break
			}

			buf.WriteByte(fb[index])
		}

		if got := buf.String(); got != spec.expOutput {
			t.Errorf("[spec %d] expected to get %q; got %q", specIndex, spec.expOutput, got)
		}
	}
}
