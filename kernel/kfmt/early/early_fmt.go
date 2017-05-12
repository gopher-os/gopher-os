package early

import "github.com/achilleasa/gopher-os/kernel/hal"

var (
	errMissingArg   = []byte("(MISSING)")
	errWrongArgType = []byte("%!(WRONGTYPE)")
	errNoVerb       = []byte("%!(NOVERB)")
	errExtraArg     = []byte("%!(EXTRA)")
	padding         = byte(' ')
	trueValue       = []byte("true")
	falseValue      = []byte("false")
)

// Printf provides a minimal Printf implementation that can be used before the
// Go runtime has been properly initialized. This version of printf does not
// allocate any memory and uses hal.ActiveTerminal for its output.
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
func Printf(format string, args ...interface{}) {
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
			for i := blockStart; i < blockEnd; i++ {
				hal.ActiveTerminal.WriteByte(format[i])
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
				hal.ActiveTerminal.Write([]byte{'%'})
				break parseFmt
			case nextCh >= '0' && nextCh <= '9':
				padLen = (padLen * 10) + int(nextCh-'0')
				continue
			case nextCh == 'd' || nextCh == 'x' || nextCh == 'o' || nextCh == 's' || nextCh == 't':
				// Run out of args to print
				if nextArgIndex >= len(args) {
					hal.ActiveTerminal.Write(errMissingArg)
					break parseFmt
				}

				switch nextCh {
				case 'o':
					fmtInt(args[nextArgIndex], 8, padLen)
				case 'd':
					fmtInt(args[nextArgIndex], 10, padLen)
				case 'x':
					fmtInt(args[nextArgIndex], 16, padLen)
				case 's':
					fmtString(args[nextArgIndex], padLen)
				case 't':
					fmtBool(args[nextArgIndex])
				}

				nextArgIndex++
				break parseFmt
			}

			// reached end of formatting string without finding a verb
			hal.ActiveTerminal.Write(errNoVerb)
		}
		blockStart, blockEnd = blockEnd+1, blockEnd+1
	}

	if blockStart != blockEnd {
		for i := blockStart; i < blockEnd; i++ {
			hal.ActiveTerminal.WriteByte(format[i])
		}
	}

	// Check for unused args
	for ; nextArgIndex < len(args); nextArgIndex++ {
		hal.ActiveTerminal.Write(errExtraArg)
	}
}

// fmtBool prints a formatted version of boolean value v using hal.ActiveTerminal
// for its output.
func fmtBool(v interface{}) {
	switch bVal := v.(type) {
	case bool:
		switch bVal {
		case true:
			hal.ActiveTerminal.Write(trueValue)
		case false:
			hal.ActiveTerminal.Write(falseValue)
		}
	default:
		hal.ActiveTerminal.Write(errWrongArgType)
		return
	}
}

// fmtString prints a formatted version of string or []byte value v, applying the
// padding specified by padLen. This function uses hal.ActiveTerminal for its
// output.
func fmtString(v interface{}, padLen int) {
	switch castedVal := v.(type) {
	case string:
		fmtRepeat(padding, padLen-len(castedVal))
		for i := 0; i < len(castedVal); i++ {
			hal.ActiveTerminal.WriteByte(castedVal[i])
		}
	case []byte:
		fmtRepeat(padding, padLen-len(castedVal))
		hal.ActiveTerminal.Write(castedVal)
	default:
		hal.ActiveTerminal.Write(errWrongArgType)
	}
}

// fmtRepeat writes count bytes with value ch to the hal.ActiveTerminal.
func fmtRepeat(ch byte, count int) {
	for i := 0; i < count; i++ {
		hal.ActiveTerminal.WriteByte(ch)
	}
}

// fmtInt prints out a formatted version of v in the requested base, applying the
// padding specified by padLen. This function uses hal.ActiveTerminal for its
// output, supports all built-in signed and unsigned integer types and supports
// base 8, 10 and 16 output.
func fmtInt(v interface{}, base, padLen int) {
	var (
		sval             int64
		uval             uint64
		divider          uint64
		remainder        uint64
		buf              [20]byte
		padCh            byte
		left, right, end int
	)

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
		hal.ActiveTerminal.Write(errWrongArgType)
		return
	}

	// Handle signs
	if sval < 0 {
		uval = uint64(-sval)
	} else if sval > 0 {
		uval = uint64(sval)
	}

	for {
		remainder = uval % divider
		if remainder < 10 {
			buf[right] = byte(remainder) + '0'
		} else {
			// map values from 10 to 15 -> a-f
			buf[right] = byte(remainder-10) + 'a'
		}

		right++

		uval /= divider
		if uval == 0 {
			break
		}
	}

	// Apply padding if required
	for ; right-left < padLen; right++ {
		buf[right] = padCh
	}

	// Apply hex prefix
	if base == 16 {
		buf[right] = 'x'
		buf[right+1] = '0'
		right += 2
	}

	// Apply negative sign to the rightmost blank character (if using enough padding);
	// otherwise append the sign as a new char
	if sval < 0 {
		for end = right - 1; buf[end] == ' '; end-- {
		}

		if end == right-1 {
			right++
		}

		buf[end+1] = '-'
	}

	// Reverse in place
	end = right
	for right = right - 1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}

	hal.ActiveTerminal.Write(buf[0:end])
}
