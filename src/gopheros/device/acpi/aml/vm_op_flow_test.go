package aml

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestFlowExpressionErrors(t *testing.T) {
	t.Run("opReturn errors", func(t *testing.T) {
		// opReturn expects an argument to evaluate as the return value
		if err := vmOpReturn(nil, new(unnamedEntity)); err != errArgIndexOutOfBounds {
			t.Errorf("expected to get errArgIndexOutOfBounds; got %v", err)
		}
	})
}

func TestVMFlowChanges(t *testing.T) {
	resolver := &mockResolver{
		tableFiles: []string{"vm-testsuite-DSDT.aml"},
	}

	vm := NewVM(os.Stderr, resolver)
	if err := vm.Init(); err != nil {
		t.Fatal(err)
	}

	specs := []struct {
		method         string
		inputA, inputB interface{}
		exp            interface{}
	}{
		{`\FL00`, uint64(0), "sequential", uint64(8)},
		{`\FL00`, uint64(42), "sequential", uint64(16)},
		{`\FL00`, uint64(100), "sequential", uint64(32)},
		{`\FL00`, uint64(999), "break", uint64(1)},
		{`\FL00`, uint64(42), "continue", uint64(0)},
		{`\FL00`, uint64(42), "return", uint64(0xbadf00d)},
	}

	for specIndex, spec := range specs {
		m := vm.Lookup(spec.method)
		if m == nil {
			t.Errorf("error looking up method: %q", spec.method)
			continue
		}

		method := m.(*Method)

		ctx := &execContext{
			methodArg: [maxMethodArgs]interface{}{spec.inputA, spec.inputB},
			vm:        vm,
		}

		if err := execBlock(ctx, method); err != nil {
			t.Errorf("[spec %02d] %s: invocation failed: %v\n", specIndex, spec.method, err)
			continue
		}

		if !reflect.DeepEqual(ctx.retVal, spec.exp) {
			t.Errorf("[spec %02d] %s: expected %d; got %v\n", specIndex, spec.method, spec.exp, ctx.retVal)
		}
	}
}

func TestVMFlowOpErrors(t *testing.T) {
	op0Err := &Error{message: "something went wrong with op 0"}
	op1Err := &Error{message: "something went wrong with op 1"}
	op2Err := &Error{message: "something went wrong with op 2"}

	vm := &VM{sizeOfIntInBits: 64}
	vm.populateJumpTable()
	vm.jumpTable[0] = func(_ *execContext, ent Entity) *Error { return op0Err }
	vm.jumpTable[1] = func(_ *execContext, ent Entity) *Error { return op1Err }
	vm.jumpTable[2] = func(_ *execContext, ent Entity) *Error { return op2Err }

	specs := []struct {
		handler opHandler
		entArgs []interface{}
		expErr  *Error
	}{
		// opWhile tests
		{
			vmOpWhile,
			[]interface{}{"args < 2"},
			errArgIndexOutOfBounds,
		},
		{
			vmOpWhile,
			[]interface{}{
				"foo",
				"not a scoped ent",
			},
			errWhileBodyNotScopedEntity,
		},
		{
			vmOpWhile,
			[]interface{}{
				&unnamedEntity{op: 0},
				&scopeEntity{},
			},
			op0Err,
		},
		{
			vmOpWhile,
			[]interface{}{
				uint64(1),
				// raise an error while exeuting the body of the while statement
				&scopeEntity{
					children: []Entity{
						&unnamedEntity{op: 1},
					},
				},
			},
			op1Err,
		},
		// opIf tests
		{
			vmOpIf,
			[]interface{}{"args < 2"},
			errArgIndexOutOfBounds,
		},
		{
			vmOpIf,
			[]interface{}{"args", ">", "3", "!!!"},
			errArgIndexOutOfBounds,
		},
		{
			vmOpIf,
			[]interface{}{
				"foo",
				"if body not a scoped ent",
			},
			errIfBodyNotScopedEntity,
		},
		{
			vmOpIf,
			[]interface{}{
				"foo",
				&scopeEntity{},
				"else body not a scoped ent",
			},
			errElseBodyNotScopedEntity,
		},
		{
			vmOpIf,
			[]interface{}{
				&unnamedEntity{op: 0},
				&scopeEntity{},
			},
			op0Err,
		},
		{
			vmOpIf,
			[]interface{}{
				uint64(1),
				// raise an error while executing the If body
				&scopeEntity{
					children: []Entity{
						&unnamedEntity{op: 1},
					},
				},
			},
			op1Err,
		},
		{
			vmOpIf,
			[]interface{}{
				uint64(0),
				&scopeEntity{},
				// raise an error while exeuting the Else body
				&scopeEntity{
					children: []Entity{
						&unnamedEntity{op: 2},
					},
				},
			},
			op2Err,
		},
		{
			vmOpFatal,
			[]interface{}{
				&scopeEntity{},
				uint64(42),
				uint64(128),
			},
			op0Err,
		},
		{
			vmOpFatal,
			[]interface{}{
				uint64(42),
				&scopeEntity{},
				uint64(128),
			},
			op0Err,
		},
	}

	ctx := &execContext{vm: vm}
	for specIndex, spec := range specs {
		ent := &unnamedEntity{args: spec.entArgs}
		if err := spec.handler(ctx, ent); err == nil || err.Error() != spec.expErr.Error() {
			t.Errorf("[spec %d] expected error: %s; got %v", specIndex, spec.expErr.Error(), err)
		}
	}
}

func TestVMNestedMethodCalls(t *testing.T) {
	resolver := &mockResolver{
		tableFiles: []string{"vm-testsuite-DSDT.aml"},
	}

	vm := NewVM(ioutil.Discard, resolver)
	if err := vm.Init(); err != nil {
		t.Fatal(err)
	}

	t.Run("nested call success", func(t *testing.T) {
		inv := &methodInvocationEntity{
			unnamedEntity: unnamedEntity{
				args: []interface{}{uint64(10)},
			},
			methodName: `\NST0`,
		}

		ctx := &execContext{vm: vm}
		if err := vmOpMethodInvocation(ctx, inv); err != nil {
			t.Fatal(err)
		}

		if exp := uint64(52); !reflect.DeepEqual(ctx.retVal, exp) {
			t.Fatalf("expected return value to be: %v; got: %v", exp, ctx.retVal)
		}
	})

	t.Run("undefined method", func(t *testing.T) {
		inv := &methodInvocationEntity{methodName: `UNDEFINED`}

		ctx := &execContext{vm: vm}
		expErr := "call to undefined method: UNDEFINED"
		err := vmOpMethodInvocation(ctx, inv)
		if err == nil || err.Error() != expErr {
			t.Fatalf("expected error: %s; got %v", expErr, err)
		}

		// Since we are invoking the method directly instead of within an execBlock
		// call, the error stack trace will not be populated
		if exp, got := "No stack trace available", err.StackTrace(); got != exp {
			t.Fatalf("expected error.StackTrace() to return:\n%s\ngot:\n%s", exp, got)
		}
	})

	t.Run("method arg load error", func(t *testing.T) {
		op0Err := &Error{message: "something went wrong with op 0"}
		vm.jumpTable[0] = func(_ *execContext, ent Entity) *Error { return op0Err }

		inv := &methodInvocationEntity{
			unnamedEntity: unnamedEntity{
				args: []interface{}{
					&unnamedEntity{}, // vmLoad will invoke jumpTable[0] which always returns an error
				},
			},
			methodName: `\NST0`,
		}

		ctx := &execContext{vm: vm}
		if err := vmOpMethodInvocation(ctx, inv); err != op0Err {
			t.Fatalf("expected error: %s; got %v", op0Err, err)
		}
	})

	t.Run("method raises fatal error", func(t *testing.T) {
		inv := &methodInvocationEntity{
			unnamedEntity: unnamedEntity{args: []interface{}{uint64(0x42)}},
			methodName:    `\NST2`,
		}

		ctx := &execContext{vm: vm}
		err := vmOpMethodInvocation(ctx, inv)
		expErr := "fatal OEM-defined error (type: 0xde, code: 0xad, arg: 0xc0de)"
		if err == nil || err.Error() != expErr {
			t.Fatalf("expected to get error: %s; got %v", expErr, err)
		}

		expTrace := `Stack trace:
[000] [DSDT] [NST2():0x2] opcode: Store
[001] [DSDT] [NST3():0x1] opcode: Add
[002] [DSDT] [NST4():0x8] opcode: If
[003] [DSDT] [NST4():0x9] opcode: Fatal
`

		if got := err.StackTrace(); got != expTrace {
			t.Fatalf("expected error.StackTrace() to return:\n%s\ngot:\n%s", expTrace, got)
		}
	})
}
