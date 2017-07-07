package kfmt

import (
	"io"
	"unsafe"
)

// maxBufSize defines the buffer size for formatting numbers.
const maxBufSize = 32

var (
	errMissingArg   = []byte("(MISSING)")
	errWrongArgType = []byte("%!(WRONGTYPE)")
	errNoVerb       = []byte("%!(NOVERB)")
	errExtraArg     = []byte("%!(EXTRA)")
	trueValue       = []byte("true")
	falseValue      = []byte("false")

	numFmtBuf = []byte("012345678901234567890123456789012")

	// singleByte is used as a shared buffer for passing single characters
	// to doWrite.
	singleByte = []byte(" ")

	// earlyPrintBuffer is a ring buffer that stores Printf output before the
	// console and TTYs are initialized.
	earlyPrintBuffer ringBuffer

	// outputSink is a io.Writer where Printf will send its output. If set
	// to nil, then the output will be redirected to the earlyPrintBuffer.
	outputSink io.Writer
)

// SetOutputSink sets the default target for calls to Printf to w and copies
// any data accumulated in the earlyPrintBuffer to itt .
func SetOutputSink(w io.Writer) {
	outputSink = w
	if w != nil {
		io.Copy(w, &earlyPrintBuffer)
	}
}

// Printf provides a minimal Printf implementation that can be safely  used
// before the Go runtime has been properly initialized. This implementation
// does not allocate any memory.
//
// Similar to fmt.Printf, this version of printf supports the following subset
// of formatting verbs:
//
// Strings:
//		%s the uninterpreted bytes of the string or byte slice
//
// Integers:
//              %o base 8
//              %d base 10
//              %x base 16, with lower-case letters for a-f
//
// Booleans:
//              %t "true" or "false"
//
// Width is specified by an optional decimal number immediately preceding the verb.
// If absent, the width is whatever is necessary to represent the value.
//
// String values with length less than the specified width will be left-padded with
// spaces. Integer values formatted as base-10 will also be left-padded with spaces.
// Finally, integer values formatted as base-16 will be left-padded with zeroes.
//
// Printf supports all built-in string and integer types but assumes that the
// Go itables have not been initialized yet so it will not check whether its
// arguments support io.Stringer if they don't match one of the supported tupes.
//
// This function does not provide support for printing pointers (%p) as this
// requires importing the reflect package. By importing reflect, the go compiler
// starts generating calls to runtime.convT2E (which calls runtime.newobject)
// when assembling the argument slice which obviously will crash the kernel since
// memory management is not yet available.
//
// The output of Printf is written to the currently active TTY. If no TTY is
// available, then the output is buffered into a ring-buffer and can be
// retrieved by a call to FlushRingBuffer.
func Printf(format string, args ...interface{}) {
	Fprintf(outputSink, format, args...)
}

// Fprintf behaves exactly like Printf but it writes the formatted output to
// the specified io.Writer.
func Fprintf(w io.Writer, format string, args ...interface{}) {
	var (
		nextCh                       byte
		nextArgIndex                 int
		blockStart, blockEnd, padLen int
		fmtLen                       = len(format)
	)

	for blockEnd < fmtLen {
		nextCh = format[blockEnd]
		if nextCh != '%' {
			blockEnd++
			continue
		}

		if blockStart < blockEnd {
			// passing format[blockStart:blockEnd] to doWrite triggers a
			// memory allocation so we need to do this one byte at a time.
			for i := blockStart; i < blockEnd; i++ {
				singleByte[0] = format[i]
				doWrite(w, singleByte)
			}
		}

		// Scan til we hit the format character
		padLen = 0
		blockEnd++
	parseFmt:
		for ; blockEnd < fmtLen; blockEnd++ {
			nextCh = format[blockEnd]
			switch {
			case nextCh == '%':
				singleByte[0] = '%'
				doWrite(w, singleByte)
				break parseFmt
			case nextCh >= '0' && nextCh <= '9':
				padLen = (padLen * 10) + int(nextCh-'0')
				continue
			case nextCh == 'd' || nextCh == 'x' || nextCh == 'o' || nextCh == 's' || nextCh == 't':
				// Run out of args to print
				if nextArgIndex >= len(args) {
					doWrite(w, errMissingArg)
					break parseFmt
				}

				switch nextCh {
				case 'o':
					fmtInt(w, args[nextArgIndex], 8, padLen)
				case 'd':
					fmtInt(w, args[nextArgIndex], 10, padLen)
				case 'x':
					fmtInt(w, args[nextArgIndex], 16, padLen)
				case 's':
					fmtString(w, args[nextArgIndex], padLen)
				case 't':
					fmtBool(w, args[nextArgIndex])
				}

				nextArgIndex++
				break parseFmt
			}

			// reached end of formatting string without finding a verb
			doWrite(w, errNoVerb)
		}
		blockStart, blockEnd = blockEnd+1, blockEnd+1
	}

	if blockStart != blockEnd {
		// passing format[blockStart:blockEnd] to doWrite triggers a
		// memory allocation so we need to do this one byte at a time.
		for i := blockStart; i < blockEnd; i++ {
			singleByte[0] = format[i]
			doWrite(w, singleByte)
		}
	}

	// Check for unused args
	for ; nextArgIndex < len(args); nextArgIndex++ {
		doWrite(w, errExtraArg)
	}
}

// fmtBool prints a formatted version of boolean value v.
func fmtBool(w io.Writer, v interface{}) {
	switch bVal := v.(type) {
	case bool:
		switch bVal {
		case true:
			doWrite(w, trueValue)
		case false:
			doWrite(w, falseValue)
		}
	default:
		doWrite(w, errWrongArgType)
		return
	}
}

// fmtString prints a formatted version of string or []byte value v, applying
// the padding specified by padLen.
func fmtString(w io.Writer, v interface{}, padLen int) {
	switch castedVal := v.(type) {
	case string:
		fmtRepeat(w, ' ', padLen-len(castedVal))
		// converting the string to a byte slice triggers a memory allocation
		// so we need to do this one byte at a time.
		for i := 0; i < len(castedVal); i++ {
			singleByte[0] = castedVal[i]
			doWrite(w, singleByte)
		}
	case []byte:
		fmtRepeat(w, ' ', padLen-len(castedVal))
		doWrite(w, castedVal)
	default:
		doWrite(w, errWrongArgType)
	}
}

// fmtRepeat writes count bytes with value ch.
func fmtRepeat(w io.Writer, ch byte, count int) {
	singleByte[0] = ch
	for i := 0; i < count; i++ {
		doWrite(w, singleByte)
	}
}

// fmtInt prints out a formatted version of v in the requested base, applying
// the padding specified by padLen. This function supports all built-in signed
// and unsigned integer types and base 8, 10 and 16 output.
func fmtInt(w io.Writer, v interface{}, base, padLen int) {
	var (
		sval             int64
		uval             uint64
		divider          uint64
		remainder        uint64
		padCh            byte
		left, right, end int
	)

	if padLen >= maxBufSize {
		padLen = maxBufSize - 1
	}

	switch base {
	case 8:
		divider = 8
		padCh = '0'
	case 10:
		divider = 10
		padCh = ' '
	case 16:
		divider = 16
		padCh = '0'
	}

	switch v.(type) {
	case uint8:
		uval = uint64(v.(uint8))
	case uint16:
		uval = uint64(v.(uint16))
	case uint32:
		uval = uint64(v.(uint32))
	case uint64:
		uval = v.(uint64)
	case uintptr:
		uval = uint64(v.(uintptr))
	case int8:
		sval = int64(v.(int8))
	case int16:
		sval = int64(v.(int16))
	case int32:
		sval = int64(v.(int32))
	case int64:
		sval = v.(int64)
	case int:
		sval = int64(v.(int))
	default:
		doWrite(w, errWrongArgType)
		return
	}

	// Handle signs
	if sval < 0 {
		uval = uint64(-sval)
	} else if sval > 0 {
		uval = uint64(sval)
	}

	for right < maxBufSize {
		remainder = uval % divider
		if remainder < 10 {
			numFmtBuf[right] = byte(remainder) + '0'
		} else {
			// map values from 10 to 15 -> a-f
			numFmtBuf[right] = byte(remainder-10) + 'a'
		}

		right++

		uval /= divider
		if uval == 0 {
			break
		}
	}

	// Apply padding if required
	for ; right-left < padLen; right++ {
		numFmtBuf[right] = padCh
	}

	// Apply negative sign to the rightmost blank character (if using enough padding);
	// otherwise append the sign as a new char
	if sval < 0 {
		for end = right - 1; numFmtBuf[end] == ' '; end-- {
		}

		if end == right-1 {
			right++
		}

		numFmtBuf[end+1] = '-'
	}

	// Reverse in place
	end = right
	for right = right - 1; left < right; left, right = left+1, right-1 {
		numFmtBuf[left], numFmtBuf[right] = numFmtBuf[right], numFmtBuf[left]
	}

	doWrite(w, numFmtBuf[0:end])
}

// doWrite is a proxy that uses the runtime.noescape hack to hide p from the
// compiler's escape analysis. Without this hack, the compiler cannot properly
// detect that p does not escape (due to the call to the yet unknown outputSink
// io.Writer) and plays it safe by flagging it as escaping. This causes all
// calls to Printf to call runtime.convT2E which triggers a memory allocation
// causing the kernel to crash if a call to Printf is made before the Go
// allocator is initialized.
func doWrite(w io.Writer, p []byte) {
	doRealWrite(w, noEscape(unsafe.Pointer(&p)))
}

func doRealWrite(w io.Writer, bufPtr unsafe.Pointer) {
	p := *(*[]byte)(bufPtr)
	if w != nil {
		w.Write(p)
	} else {
		earlyPrintBuffer.Write(p)
	}
}

// noEscape hides a pointer from escape analysis. This function is copied over
// from runtime/stubs.go
//go:nosplit
func noEscape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}
