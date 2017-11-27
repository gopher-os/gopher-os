package aml

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
