package aml

import (
	"reflect"
	"testing"
)

func TestVMLoad(t *testing.T) {
	// Use a pointer to ensure that when we dereference an objRef we get
	// back the same pointer
	uniqueVal := &execContext{}
	aRef := &objRef{isArgRef: false, ref: 42}

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
		// reference handling
		{
			nil,
			&objRef{isArgRef: true, ref: uniqueVal},
			uniqueVal,
			nil,
		},
		{
			nil,
			aRef,
			aRef,
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

func TestVMStore(t *testing.T) {
	t.Run("errors", func(t *testing.T) {
		ctx := new(execContext)

		if err := vmStore(ctx, nil, 42); err != errNilStoreOperands {
			t.Fatal("expected to get errNilStoreOperands")
		}

		if err := vmStore(ctx, "foo", nil); err != errNilStoreOperands {
			t.Fatal("expected to get errNilStoreOperands")
		}

		if err := vmStore(ctx, 42, "not-an-entity"); err != errInvalidStoreDestination {
			t.Fatal("expected to get errInvalidStoreDestination")
		}

		if err := vmStore(ctx, &unnamedEntity{}, &unnamedEntity{op: opArg0}); err != errCopyFailed {
			t.Fatal("expected to get errCopyFailed")
		}

		// Storing to fields, bufferFields & named objects is not yet supported
		expErr := &Error{message: "vmStore: unsupported opcode: Buffer"}
		if err := vmStore(ctx, uint64(42), &scopeEntity{op: opBuffer, name: "BUF0"}); err == nil || err.Error() != expErr.Error() {
			t.Fatalf("expected to get error: %v; got %v", expErr, err)
		}

	})

	t.Run("store to local arg", func(t *testing.T) {
		ctx := &execContext{
			localArg: [maxLocalArgs]interface{}{
				"foo",
				uint64(42),
				10,
			},
		}

		if err := vmStore(ctx, uint64(123), &unnamedEntity{op: opLocal1}); err != nil {
			t.Fatal(err)
		}

		expArgs := [maxLocalArgs]interface{}{
			"foo",
			uint64(123),
			10,
		}
		if !reflect.DeepEqual(ctx.localArg, expArgs) {
			t.Fatalf("expected local args to be %v; got %v", expArgs, ctx.localArg)
		}
	})

	t.Run("store to method arg", func(t *testing.T) {
		ref := &objRef{isArgRef: true, ref: "foo"}
		ctx := &execContext{
			methodArg: [maxMethodArgs]interface{}{
				"foo",
				uint64(42),
				ref,
			},
		}

		if err := vmStore(ctx, uint64(123), &unnamedEntity{op: opArg1}); err != nil {
			t.Fatal(err)
		}

		if err := vmStore(ctx, "bar", &unnamedEntity{op: opArg2}); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ctx.methodArg[0], "foo") {
			t.Fatal("expected methodArg[0] not to be modified")
		}

		if exp := uint64(123); !reflect.DeepEqual(ctx.methodArg[1], exp) {
			t.Fatalf("expected methodArg[1] to be set to: %v; got %v", exp, ctx.methodArg[1])
		}

		if ctx.methodArg[2] != ref {
			t.Fatal("expected the reference instance in methodArg[2] not to be modified")
		}

		if exp := "bar"; ref.ref != exp {
			t.Fatalf("expected the referenced value by methodArg[2] to be set to: %v; got %v", exp, ctx.methodArg[2])
		}
	})

	t.Run("store to Debug object or a constant", func(t *testing.T) {
		constant := &constEntity{val: "foo"}
		if err := vmStore(nil, 42, constant); err != nil {
			t.Fatal(err)
		}

		if exp := "foo"; constant.val != exp {
			t.Fatalf("expected storing to constant to be a no-op; constant value changed from %v to %v", exp, constant.val)
		}

		if err := vmStore(nil, 42, &unnamedEntity{op: opDebug}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestVMCondStore(t *testing.T) {
	specs := []struct {
		ctx      *execContext
		args     []interface{}
		argIndex int
		val      interface{}
	}{
		// Not enough args to get target
		{
			nil,
			[]interface{}{"foo"},
			2,
			"bar",
		},
		// Target is nil
		{
			nil,
			[]interface{}{nil, "foo"},
			0,
			"bar",
		},
		// Target is a constant with a nil value
		{
			nil,
			[]interface{}{
				&constEntity{},
			},
			0,
			"bar",
		},
		// Valid target
		{
			&execContext{},
			[]interface{}{
				&unnamedEntity{op: opLocal0},
			},
			0,
			"bar",
		},
	}

	for specIndex, spec := range specs {
		if err := vmCondStore(spec.ctx, spec.val, &unnamedEntity{args: spec.args}, spec.argIndex); err != nil {
			t.Errorf("[spec %d] error: %v", specIndex, spec)
		}
	}
}
