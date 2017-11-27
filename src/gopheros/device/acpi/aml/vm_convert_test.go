package aml

import (
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
