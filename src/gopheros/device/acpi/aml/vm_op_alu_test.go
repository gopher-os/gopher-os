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

func TestBitwiseExpressions(t *testing.T) {
	specs := []struct {
		method string
		input  interface{}
		exp    uint64
	}{
		{`\BI00`, uint64(100), uint64(800)},
		{`\BI01`, uint64(100), uint64(25)},
		{`\BI02`, uint64(7), uint64(5)},
		{`\BI03`, uint64(32), uint64(40)},
		{`\BI04`, uint64(8), uint64(0)},
		{`\BI04`, uint64(9), uint64(1)},
		{`\BI05`, uint64(7), uint64(0xfffffffffffffff8)},
		{`\BI05`, uint64(12), uint64(0xfffffffffffffff0)},
		{`\BI06`, uint64(32), uint64(48)},
		{`\BI07`, uint64(0xffffffff), uint64(0xffffffff00000000)},
		{`\BI08`, uint64(1 << 63), uint64(1)},
		{`\BI08`, uint64(1), uint64(64)},
		{`\BI08`, uint64(0), uint64(0)},
		{`\BI09`, uint64(1 << 2), uint64(3)},
		{`\BI09`, uint64(1 << 63), uint64(64)},
		{`\BI09`, uint64(0), uint64(0)},
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

func TestBitwiseExpressionErrors(t *testing.T) {
	t.Run("arg handling errors", func(t *testing.T) {
		specs := []opHandler{
			vmOpShiftLeft,
			vmOpShiftRight,
			vmOpBitwiseAnd,
			vmOpBitwiseOr,
			vmOpBitwiseNand,
			vmOpBitwiseNor,
			vmOpBitwiseXor,
			vmOpBitwiseNot,
			vmOpFindSetLeftBit,
			vmOpFindSetRightBit,
		}

		for specIndex, handler := range specs {
			if err := handler(nil, new(unnamedEntity)); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}
		}
	})
}

func TestLogicExpressions(t *testing.T) {
	bufOp := unnamedEntity{op: opBuffer}

	specs := []struct {
		method         string
		inputA, inputB interface{}
		exp            interface{}
	}{
		{`\LO00`, uint64(10), uint64(10), uint64(1)},
		{`\LO00`, uint64(5), uint64(0), uint64(0)},
		{`\LO00`, "lorem", "lorem", uint64(1)},
		{`\LO00`, "ipsum", "DOLOR", uint64(0)},
		{
			`\LO00`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte{'!'}},
			&bufferEntity{unnamedEntity: bufOp, data: []byte{'!'}},
			uint64(1),
		},
		{
			`\LO00`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte("LOREM")},
			&bufferEntity{unnamedEntity: bufOp, data: []byte("lorem")},
			uint64(0),
		},
		//
		{`\LO01`, uint64(10), uint64(10), uint64(0)},
		{`\LO01`, uint64(5), uint64(0), uint64(1)},
		{`\LO01`, uint64(30), uint64(150), uint64(0)},
		{`\LO01`, "lorem", "lore", uint64(1)},
		{`\LO01`, "abc", "abd", uint64(0)},
		{
			`\LO01`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte("lore0")},
			&bufferEntity{unnamedEntity: bufOp, data: []byte("lore1")},
			uint64(0),
		},
		{
			`\LO01`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte("1000")},
			&bufferEntity{unnamedEntity: bufOp, data: []byte("0111")},
			uint64(1),
		},
		//
		{`\LO02`, uint64(10), uint64(10), uint64(1)},
		{`\LO02`, uint64(50), uint64(49), uint64(1)},
		{`\LO02`, uint64(49), uint64(50), uint64(0)},
		//
		{`\LO03`, uint64(10), uint64(10), uint64(0)},
		{`\LO03`, uint64(0), uint64(10), uint64(1)},
		//
		{`\LO04`, uint64(10), uint64(10), uint64(0)},
		{`\LO04`, uint64(5), uint64(0), uint64(0)},
		{`\LO04`, uint64(30), uint64(150), uint64(1)},
		{`\LO04`, "123", "321", uint64(1)},
		{`\LO04`, "ab", "abc", uint64(1)},
		{
			`\LO04`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte("lore000")},
			&bufferEntity{unnamedEntity: bufOp, data: []byte("lore1")},
			uint64(0),
		},
		{
			`\LO04`,
			&bufferEntity{unnamedEntity: bufOp, data: []byte("1000")},
			&bufferEntity{unnamedEntity: bufOp, data: []byte("0111+1")},
			uint64(1),
		},
		//
		{`\LO05`, uint64(10), uint64(10), uint64(1)},
		{`\LO05`, uint64(50), uint64(49), uint64(0)},
		{`\LO05`, uint64(49), uint64(50), uint64(1)},
		//
		{`\LO06`, true, false, uint64(0)},
		{`\LO06`, false, true, uint64(0)},
		{`\LO06`, true, true, uint64(1)},
		{`\LO06`, false, false, uint64(0)},
		{`\LO06`, "AA", "0", uint64(0)},
		{`\LO06`, "0", "F00", uint64(0)},
		//
		{`\LO07`, true, false, uint64(1)},
		{`\LO07`, false, true, uint64(1)},
		{`\LO07`, true, true, uint64(1)},
		{`\LO07`, false, false, uint64(0)},
		{`\LO07`, "AA", "0", uint64(1)},
		{`\LO07`, "0", "F00", uint64(1)},
		{`\LO07`, "f00", "c0ffee", uint64(1)},
		{`\LO07`, "0", "0", uint64(0)},
		//
		{`\LO08`, true, nil, uint64(0)},
		{`\LO08`, false, nil, uint64(1)},
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
			methodArg: [maxMethodArgs]interface{}{spec.inputA, spec.inputB},
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

func TestLogicExpressionErrors(t *testing.T) {
	specs := []struct {
		handler  opHandler
		argCount int
	}{
		{vmOpLogicalNot, 1},
		{vmOpLogicalAnd, 2},
		{vmOpLogicalOr, 2},
		{vmOpLogicalEqual, 2},
		{vmOpLogicalLess, 2},
		{vmOpLogicalGreater, 2},
	}

	t.Run("arg handling errors", func(t *testing.T) {
		for specIndex, spec := range specs {
			if err := spec.handler(nil, new(unnamedEntity)); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}
		}
	})

	t.Run("arg loading errors", func(t *testing.T) {
		ent := &unnamedEntity{
			args: make([]interface{}, 2),
		}

		vm := &VM{sizeOfIntInBits: 64}
		vm.populateJumpTable()
		ctx := &execContext{vm: vm}

		for specIndex, spec := range specs {
			ent.args[0] = &Device{}
			ent.args[1] = uint64(9)
			if err := spec.handler(ctx, ent); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}

			if spec.argCount < 2 {
				continue
			}
			ent.args[0] = uint64(123)
			ent.args[1] = &Device{}
			if err := spec.handler(ctx, ent); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}
		}
	})

	t.Run("2nd arg conversion error", func(t *testing.T) {
		ent := &unnamedEntity{
			args: make([]interface{}, 2),
		}

		vm := &VM{sizeOfIntInBits: 64}
		vm.populateJumpTable()
		ctx := &execContext{vm: vm}

		for specIndex, spec := range specs {
			if spec.argCount < 2 {
				continue
			}

			ent.args[0] = uint64(30)
			ent.args[1] = int64(9)
			if err := spec.handler(ctx, ent); err == nil {
				t.Errorf("[spec %d] expected opHandler to return an error", specIndex)
			}
		}
	})

	t.Run("unsupported comparison error", func(t *testing.T) {
		specs := []opHandler{
			vmOpLogicalEqual,
			vmOpLogicalLess,
			vmOpLogicalGreater,
		}

		ent := &unnamedEntity{
			args: make([]interface{}, 2),
		}

		vm := &VM{sizeOfIntInBits: 64}
		vm.populateJumpTable()
		ctx := &execContext{vm: vm}

		for specIndex, handler := range specs {
			ent.args[0] = int64(1)
			ent.args[1] = int64(9)
			if err := handler(ctx, ent); err != errInvalidComparisonType {
				t.Errorf("[spec %d] expected opHandler to return errInvalidComparisonType; got %v", specIndex, err)
			}
		}
	})
}
