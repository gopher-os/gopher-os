package aml

import (
	"reflect"
	"testing"
)

func TestVMLoad(t *testing.T) {
	specs := []struct {
		ctx    *execContext
		argIn  interface{}
		valOut interface{}
		err    *Error
	}{
		{
			nil,
			&constEntity{val: uint64(123)},
			uint64(123),
			nil,
		},
		{
			nil,
			&constEntity{val: "a string"},
			"a string",
			nil,
		},
		{
			nil,
			&constEntity{
				val: &constEntity{val: true},
			},
			uint64(1),
			nil,
		},
		{
			nil,
			&constEntity{
				val: &constEntity{val: false},
			},
			uint64(0),
			nil,
		},
		{
			&execContext{
				localArg: [maxLocalArgs]interface{}{"foo"},
			},
			&unnamedEntity{op: opLocal0},
			"foo",
			nil,
		},
		{
			&execContext{
				methodArg: [maxMethodArgs]interface{}{"bar", "foo"},
			},
			&unnamedEntity{op: opArg1},
			"foo",
			nil,
		},
		// Unsupported reads
		{
			nil,
			&unnamedEntity{op: opBuffer},
			nil,
			&Error{message: "readArg: unsupported entity type: Buffer"},
		},
	}

	for specIndex, spec := range specs {
		got, err := vmLoad(spec.ctx, spec.argIn)
		switch {
		case !reflect.DeepEqual(spec.err, err):
			t.Errorf("[spec %d] expected error: %v; got: %v", specIndex, spec.err, err)
		case got != spec.valOut:
			t.Errorf("[spec %d] expected to get value %v (type: %v); got %v (type %v)", specIndex,
				spec.valOut, reflect.TypeOf(spec.valOut),
				got, reflect.TypeOf(got),
			)
		}

	}
}
