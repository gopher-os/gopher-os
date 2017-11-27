package aml

import "strconv"

// valueType represents the data types that the AML interpreter can process.
type valueType uint8

// The list of supporte AML value types (see p. 876 of ACPI 6.2 standard)
const (
	valueTypeUninitialized valueType = iota
	valueTypeBuffer
	valueTypeBufferField
	valueTypeDDBHandle
	valueTypeDebugObject
	valueTypeDevice
	valueTypeEvent
	valueTypeFieldUnit
	valueTypeInteger // we also treat constants as integers
	valueTypeMethod
	valueTypeMutex
	valueTypeObjectRef
	valueTypeRegion
	valueTypePackage
	valueTypeString
	valueTypePowerResource
	valueTypeProcessor
	valueTypeRawDataBuffer
	valueTypeThermalZone
)

// String implements fmt.Stringer for valueType.
func (vt valueType) String() string {
	switch vt {
	case valueTypeBuffer:
		return "Buffer"
	case valueTypeBufferField:
		return "BufferField"
	case valueTypeDDBHandle:
		return "DDBHandle"
	case valueTypeDebugObject:
		return "DebugObject"
	case valueTypeDevice:
		return "Device"
	case valueTypeEvent:
		return "Event"
	case valueTypeFieldUnit:
		return "FieldUnit"
	case valueTypeInteger:
		return "Integer"
	case valueTypeMethod:
		return "Method"
	case valueTypeMutex:
		return "Mutex"
	case valueTypeObjectRef:
		return "ObjectRef"
	case valueTypeRegion:
		return "Region"
	case valueTypePackage:
		return "Package"
	case valueTypeString:
		return "String"
	case valueTypePowerResource:
		return "PowerResource"
	case valueTypeProcessor:
		return "Processor"
	case valueTypeRawDataBuffer:
		return "RawDataBuffer"
	case valueTypeThermalZone:
		return "ThermalZone"
	default:
		return "Uninitialized"
	}
}

// vmToIntArg attempts to convert the entity argument at position argIndex to
// a uint64 value.
func vmToIntArg(ctx *execContext, ent Entity, argIndex int) (uint64, *Error) {
	args := ent.getArgs()
	if len(args) <= argIndex {
		return 0, errArgIndexOutOfBounds
	}

	argVal, err := vmConvert(ctx, args[argIndex], valueTypeInteger)
	if err != nil {
		return 0, err
	}

	return argVal.(uint64), nil
}

// vmToIntArgs2 attempts to convert the entity arguments at positions argIndex1
// and argIndex2 to uint64 values.
func vmToIntArgs2(ctx *execContext, ent Entity, argIndex1, argIndex2 int) (uint64, uint64, *Error) {
	args := ent.getArgs()
	if len(args) <= argIndex1 || len(args) <= argIndex2 {
		return 0, 0, errArgIndexOutOfBounds
	}

	argVal1, err := vmConvert(ctx, args[argIndex1], valueTypeInteger)
	if err != nil {
		return 0, 0, err
	}

	argVal2, err := vmConvert(ctx, args[argIndex2], valueTypeInteger)
	if err != nil {
		return 0, 0, err
	}

	return argVal1.(uint64), argVal2.(uint64), nil

}

// vmConvert attempts to convert the input argument to the specified type. If
// the conversion is not possible then vmConvert returns back an error.
func vmConvert(ctx *execContext, arg interface{}, toType valueType) (interface{}, *Error) {
	argVal, err := vmLoad(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Conversion not required; we can just read the value directly
	argType := vmTypeOf(ctx, argVal)
	if argType == toType {
		return argVal, nil
	}

	switch argType {
	case valueTypeString:
		argAsStr := argVal.(string)
		switch toType {
		case valueTypeInteger:
			// According to the spec: If no integer object exists, a new integer is created. The
			// integer is initialized to the value zero and the ASCII string is interpreted as a
			// hexadecimal constant. Each string character is interpreted as a hexadecimal value
			// (‘0’- ‘9’, ‘A’-‘F’, ‘a’-‘f’), starting with the first character as the most significant
			// digit, and ending with the first non-hexadecimal character, end-of-string, or
			// when the size of an integer is reached (8 characters for 32-bit integers and 16
			// characters for 64-bit integers). Note: the first non-hex character terminates the
			// conversion without error, and a “0x” prefix is not allowed. Conversion of a null
			// (zero-length) string to an integer is not allowed.
			if len(argAsStr) == 0 {
				return nil, errConversionFailed
			}

			var res = uint64(0)
			for i := 0; i < len(argAsStr) && i < ctx.vm.sizeOfIntInBits>>2; i++ {
				ch := argAsStr[i]
				if ch >= '0' && ch <= '9' {
					res = res<<4 | uint64(ch-'0')
				} else if ch >= 'a' && ch <= 'f' {
					res = res<<4 | uint64(ch-'a'+10)
				} else if ch >= 'A' && ch <= 'F' {
					res = res<<4 | uint64(ch-'A'+10)
				} else {
					// non-hex character; we should stop and return without an error
					break
				}
			}

			return res, nil
		}
	case valueTypeInteger:
		argAsInt := argVal.(uint64)
		switch toType {
		case valueTypeString:
			// Integers are formatted as hex strings without a 0x prefix
			return strconv.FormatUint(argAsInt, 16), nil
		}
	}

	return nil, errConversionFailed
}

// vmTypeOf returns the type of data stored inside the supplied argument.
func vmTypeOf(ctx *execContext, arg interface{}) valueType {
	// Some objects (e.g args, constEntity contents) may require to perform
	// more than one pass to figure out their type
	for {
		switch typ := arg.(type) {
		case *constEntity:
			// check the value stored inside
			arg = typ.val
		case *Device:
			return valueTypeDevice
		case *Method:
			return valueTypeMethod
		case *bufferEntity:
			return valueTypeBuffer
		case *bufferFieldEntity:
			return valueTypeBufferField
		case *fieldUnitEntity, *indexFieldEntity:
			return valueTypeFieldUnit
		case *regionEntity:
			return valueTypeRegion
		case *objRef:
			return valueTypeObjectRef
		case *eventEntity:
			return valueTypeEvent
		case *mutexEntity:
			return valueTypeMutex
		case Entity:
			op := typ.getOpcode()

			switch op {
			case opPackage:
				return valueTypePackage
			case opPowerRes:
				return valueTypePowerResource
			case opProcessor:
				return valueTypeProcessor
			case opThermalZone:
				return valueTypeThermalZone
			}

			// Check if this a local or method arg; if so we need to
			// fetch the arg and check its type
			if op >= opLocal0 && op <= opLocal7 {
				arg = ctx.localArg[op-opLocal0]
			} else if op >= opArg0 && op <= opArg6 {
				arg = ctx.methodArg[op-opArg0]
			} else {
				return valueTypeUninitialized
			}
		case string:
			return valueTypeString
		case uint64, bool:
			return valueTypeInteger
		case []byte:
			return valueTypeRawDataBuffer
		default:
			return valueTypeUninitialized
		}
	}
}
