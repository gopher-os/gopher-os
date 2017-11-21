package aml

import (
	"os"
	"reflect"
	"testing"
)

func TestArithmeticExpressions(t *testing.T) {
	specs := []struct {
		method string
		input  interface{}
		exp    uint64
	}{
		{`\AR00`, uint64(10), uint64(15)},
		{`\ARI0`, uint64(10), uint64(15)},
		{`\AR01`, uint64(6), uint64(1)},
		{`\ARI1`, uint64(6), uint64(1)},
		{`\AR02`, uint64(3), uint64(24)},
		{`\ARI2`, uint64(3), uint64(24)},
		{`\AR03`, uint64(42), uint64(41)},
		{`\ARI3`, uint64(42), uint64(41)},
		{`\AR04`, uint64(42), uint64(43)},
		{`\ARI4`, uint64(42), uint64(43)},
		{`\AR05`, uint64(100), uint64(0)},
		{`\ARI5`, uint64(100), uint64(0)},
		{`\AR06`, uint64(100), uint64(10)},
		{`\ARI6`, uint64(100), uint64(10)},
		{`\AR06`, uint64(101), uint64(11)},
		{`\ARI6`, uint64(101), uint64(10)},
	}

	resolver := &mockResolver{
		tableFiles: []string{"vm-testsuite-DSDT.aml"},
	}

	vm := NewVM(os.Stderr, resolver)
	if err := vm.Init(); err != nil {
		t.Fatal(err)
	}

	for specIndex, spec := range specs {
		m := vm.Lookup(spec.method)
		if m == nil {
			t.Errorf("error looking up method: %q", spec.method)
			continue
		}

		method := m.(*Method)

		ctx := &execContext{
			methodArg: [maxMethodArgs]interface{}{spec.input},
			vm:        vm,
		}

		if err := vm.execBlock(ctx, method); err != nil {
			t.Errorf("[spec %02d] %s: invocation failed: %v\n", specIndex, spec.method, err)
			continue
		}

		if !reflect.DeepEqual(ctx.retVal, spec.exp) {
			t.Errorf("[spec %02d] %s: expected %d; got %v\n", specIndex, spec.method, spec.exp, ctx.retVal)
		}
	}
}

func TestArithmeticExpressionErrors(t *testing.T) {
	t.Run("arg handling errors", func(t *testing.T) {
		specs := []opHandler{
			vmOpAdd,
			vmOpSubtract,
			vmOpIncrement,
			vmOpDecrement,
			vmOpMultiply,
			vmOpDivide,
			vmOpMod,
		}

		for specIndex, handler := range specs {
			if err := handler(nil, new(unnamedEntity)); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}
		}
	})

	t.Run("division by zero errors", func(t *testing.T) {
		specs := []opHandler{
			vmOpDivide,
			vmOpMod,
		}

		ent := &unnamedEntity{
			args: []interface{}{
				&constEntity{val: uint64(1)},
				&constEntity{val: uint64(0)},
			},
		}
		for specIndex, handler := range specs {
			if err := handler(nil, ent); err != errDivideByZero {
				t.Errorf("[spec %d] expected opHandler to return errDivideByZero; got %v", specIndex, err)
			}
		}
	})

	t.Run("secondary value store errors", func(t *testing.T) {
		specs := []opHandler{
			vmOpIncrement,
			vmOpDecrement,
			vmOpDivide,
			vmOpMod,
		}

		ctx := new(execContext)
		ent := &unnamedEntity{
			args: []interface{}{
				uint64(64),
				&constEntity{val: uint64(4)},
				"foo", // error: store target must be an AML entity
				"bar", // error: store target must be an AML entity
			},
		}
		for specIndex, handler := range specs {
			if err := handler(ctx, ent); err != errInvalidStoreDestination {
				t.Errorf("[spec %d] expected opHandler to return errInvalidStoreDestination; got %v", specIndex, err)
			}
		}
	})
}
