package aml

import (
	"reflect"
	"testing"
)

func TestValueTypeToString(t *testing.T) {
	specs := map[valueType]string{
		valueTypeBuffer:        "Buffer",
		valueTypeBufferField:   "BufferField",
		valueTypeDDBHandle:     "DDBHandle",
		valueTypeDebugObject:   "DebugObject",
		valueTypeDevice:        "Device",
		valueTypeEvent:         "Event",
		valueTypeFieldUnit:     "FieldUnit",
		valueTypeInteger:       "Integer",
		valueTypeMethod:        "Method",
		valueTypeMutex:         "Mutex",
		valueTypeObjectRef:     "ObjectRef",
		valueTypeRegion:        "Region",
		valueTypePackage:       "Package",
		valueTypeString:        "String",
		valueTypePowerResource: "PowerResource",
		valueTypeProcessor:     "Processor",
		valueTypeRawDataBuffer: "RawDataBuffer",
		valueTypeThermalZone:   "ThermalZone",
		valueTypeUninitialized: "Uninitialized",
	}

	for vt, exp := range specs {
		if got := vt.String(); got != exp {
			t.Errorf("expected valueType %d string representation to be %q; got %q", vt, exp, got)
		}
	}
}

func TestVMTypeOf(t *testing.T) {
	specs := []struct {
		ctx     *execContext
		in      interface{}
		expType valueType
	}{
		{
			nil,
			&constEntity{val: uint64(42)},
			valueTypeInteger,
		},
		{
			nil,
			&constEntity{val: "some string"},
			valueTypeString,
		},
		{
			nil,
			&Device{},
			valueTypeDevice,
		},
		{
			nil,
			&Method{},
			valueTypeMethod,
		},
		{
			nil,
			&bufferEntity{},
			valueTypeBuffer,
		},
		{
			nil,
			&bufferFieldEntity{},
			valueTypeBufferField,
		},
		{
			nil,
			&fieldUnitEntity{},
			valueTypeFieldUnit,
		},
		{
			nil,
			&indexFieldEntity{},
			valueTypeFieldUnit,
		},
		{
			nil,
			&regionEntity{},
			valueTypeRegion,
		},
		{
			nil,
			&objRef{},
			valueTypeObjectRef,
		},
		{
			nil,
			&eventEntity{},
			valueTypeEvent,
		},
		{
			nil,
			&mutexEntity{},
			valueTypeMutex,
		},
		{
			nil,
			&unnamedEntity{op: opPackage},
			valueTypePackage,
		},
		{
			nil,
			&unnamedEntity{op: opPowerRes},
			valueTypePowerResource,
		},
		{
			nil,
			&unnamedEntity{op: opProcessor},
			valueTypeProcessor,
		},
		{
			nil,
			&unnamedEntity{op: opThermalZone},
			valueTypeThermalZone,
		},
		{
			&execContext{
				localArg: [maxLocalArgs]interface{}{
					uint64(42),
				},
			},
			&unnamedEntity{op: opLocal0},
			valueTypeInteger,
		},
		{
			&execContext{
				methodArg: [maxMethodArgs]interface{}{
					uint64(42),
					"foo",
				},
			},
			&unnamedEntity{op: opArg1},
			valueTypeString,
		},
		{
			nil,
			"foo",
			valueTypeString,
		},
		{
			nil,
			uint64(42),
			valueTypeInteger,
		},
		{
			nil,
			[]byte("some data"),
			valueTypeRawDataBuffer,
		},
		{
			nil,
			&unnamedEntity{op: opAdd},
			valueTypeUninitialized,
		},
		{
			nil,
			int64(0xbadf00d),
			valueTypeUninitialized,
		},
	}

	for specIndex, spec := range specs {
		if got := vmTypeOf(spec.ctx, spec.in); got != spec.expType {
			t.Errorf("[spec %d] expected to get value type %s; got %s", specIndex, spec.expType, got)
		}
	}
}

func TestVMToIntArg(t *testing.T) {
	ctx := &execContext{
		vm: &VM{sizeOfIntInBits: 64},
	}

	specs := []struct {
		ent      Entity
		argIndex int
		expVal   uint64
		expErr   *Error
	}{
		{
			&unnamedEntity{
				args: []interface{}{uint64(42)},
			},
			0,
			42,
			nil,
		},
		{
			&unnamedEntity{
				args: []interface{}{""},
			},
			0,
			0,
			errConversionFailed,
		},
		{
			&unnamedEntity{},
			0,
			0,
			errArgIndexOutOfBounds,
		},
	}

	for specIndex, spec := range specs {
		got, err := vmToIntArg(ctx, spec.ent, spec.argIndex)
		switch {
		case !reflect.DeepEqual(spec.expErr, err):
			t.Errorf("[spec %d] expected error: %v; got: %v", specIndex, spec.expErr, err)
		case got != spec.expVal:
			t.Errorf("[spec %d] expected to get value %v; got %v", specIndex, spec.expVal, got)
		}
	}
}

func TestVMToIntArgs2(t *testing.T) {
	ctx := &execContext{
		vm: &VM{sizeOfIntInBits: 64},
	}

	specs := []struct {
		ent      Entity
		argIndex [2]int
		expVal   [2]uint64
		expErr   *Error
	}{
		{
			&unnamedEntity{
				args: []interface{}{uint64(42), uint64(999)},
			},
			[2]int{0, 1},
			[2]uint64{42, 999},
			nil,
		},
		{
			&unnamedEntity{
				args: []interface{}{"", uint64(999)},
			},
			[2]int{0, 1},
			[2]uint64{0, 0},
			errConversionFailed,
		},
		{
			&unnamedEntity{
				args: []interface{}{uint64(123), ""},
			},
			[2]int{0, 1},
			[2]uint64{0, 0},
			errConversionFailed,
		},
		{
			&unnamedEntity{},
			[2]int{128, 0},
			[2]uint64{0, 0},
			errArgIndexOutOfBounds,
		},
		{
			&unnamedEntity{args: []interface{}{uint64(42)}},
			[2]int{0, 128},
			[2]uint64{0, 0},
			errArgIndexOutOfBounds,
		},
	}

	for specIndex, spec := range specs {
		got1, got2, err := vmToIntArgs2(ctx, spec.ent, 0, 1)
		switch {
		case !reflect.DeepEqual(spec.expErr, err):
			t.Errorf("[spec %d] expected error: %v; got: %v", specIndex, spec.expErr, err)
		case got1 != spec.expVal[0] || got2 != spec.expVal[1]:
			t.Errorf("[spec %d] expected to get values [%v, %v] ; got [%v, %v]", specIndex,
				spec.expVal[0], spec.expVal[1],
				got1, got2,
			)
		}
	}
}

func TestVMConvert(t *testing.T) {
	vm := NewVM(nil, nil)
	vm.populateJumpTable()

	vm.jumpTable[0] = func(_ *execContext, ent Entity) *Error {
		return &Error{message: "something went wrong"}
	}

	specs := []struct {
		ctx    *execContext
		in     interface{}
		toType valueType
		expVal interface{}
		expErr *Error
	}{
		// No conversion required
		{
			nil,
			"foo",
			valueTypeString,
			"foo",
			nil,
		},
		// string -> int (32-bit mode)
		{
			&execContext{
				vm: &VM{sizeOfIntInBits: 32},
			},
			"bAdF00D9",
			valueTypeInteger,
			uint64(0xbadf00d9),
			nil,
		},
		// string -> int (64-bit mode)
		{
			&execContext{
				vm: &VM{sizeOfIntInBits: 64},
			},
			"feedfaceDEADC0DE-ignored-data",
			valueTypeInteger,
			uint64(0xfeedfacedeadc0de),
			nil,
		},
		// string -> int (64-bit mode) ; stop at first non-hex char
		{
			&execContext{
				vm: &VM{sizeOfIntInBits: 64},
			},
			"feedGARBAGE",
			valueTypeInteger,
			uint64(0xfeed),
			nil,
		},
		// string -> int; empty string should trigger an error
		{
			&execContext{
				vm: &VM{sizeOfIntInBits: 64},
			},
			"",
			valueTypeInteger,
			nil,
			errConversionFailed,
		},
		// int -> string
		{
			nil,
			uint64(0xfeedfacedeadc0de),
			valueTypeString,
			"feedfacedeadc0de",
			nil,
		},
		// conversion to unsupported type
		{
			nil,
			uint64(42),
			valueTypeDevice,
			nil,
			errConversionFailed,
		},
		{
			&execContext{vm: vm},
			&unnamedEntity{op: 0}, // uses our patched jumpTable[0] that always errors
			valueTypeString,
			nil,
			&Error{message: "vmLoad: something went wrong"},
		},
	}

	for specIndex, spec := range specs {
		got, err := vmConvert(spec.ctx, spec.in, spec.toType)
		switch {
		case !reflect.DeepEqual(spec.expErr, err):
			t.Errorf("[spec %d] expected error: %v; got: %v", specIndex, spec.expErr, err)
		case got != spec.expVal:
			t.Errorf("[spec %d] expected to get value %v (type: %v); got %v (type %v)", specIndex,
				spec.expVal, reflect.TypeOf(spec.expVal),
				got, reflect.TypeOf(got),
			)
		}
	}
}
