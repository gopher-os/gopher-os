package aml

import "testing"

func TestOpcodeIsX(t *testing.T) {
	specs := []struct {
		op     uint16
		testFn func(uint16) bool
		want   bool
	}{
		// OpIsLocalArg
		{pOpLocal0, pOpIsLocalArg, true},
		{pOpLocal1, pOpIsLocalArg, true},
		{pOpLocal2, pOpIsLocalArg, true},
		{pOpLocal3, pOpIsLocalArg, true},
		{pOpLocal4, pOpIsLocalArg, true},
		{pOpLocal5, pOpIsLocalArg, true},
		{pOpLocal6, pOpIsLocalArg, true},
		{pOpLocal7, pOpIsLocalArg, true},
		{pOpArg0, pOpIsLocalArg, false},
		{pOpDivide, pOpIsLocalArg, false},
		// OpIsMethodArg
		{pOpArg0, pOpIsMethodArg, true},
		{pOpArg1, pOpIsMethodArg, true},
		{pOpArg2, pOpIsMethodArg, true},
		{pOpArg3, pOpIsMethodArg, true},
		{pOpArg4, pOpIsMethodArg, true},
		{pOpArg5, pOpIsMethodArg, true},
		{pOpArg6, pOpIsMethodArg, true},
		{pOpLocal7, pOpIsMethodArg, false},
		{pOpIf, pOpIsMethodArg, false},
		// OpIsArg
		{pOpLocal5, pOpIsArg, true},
		{pOpArg1, pOpIsArg, true},
		{pOpDivide, pOpIsArg, false},
		// OpIsType2
		{pOpAcquire, pOpIsType2, true},
		{pOpAdd, pOpIsType2, true},
		{pOpAnd, pOpIsType2, true},
		{pOpBuffer, pOpIsType2, true},
		{pOpConcat, pOpIsType2, true},
		{pOpConcatRes, pOpIsType2, true},
		{pOpCondRefOf, pOpIsType2, true},
		{pOpCopyObject, pOpIsType2, true},
		{pOpDecrement, pOpIsType2, true},
		{pOpDerefOf, pOpIsType2, true},
		{pOpDivide, pOpIsType2, true},
		{pOpFindSetLeftBit, pOpIsType2, true},
		{pOpFindSetRightBit, pOpIsType2, true},
		{pOpFromBCD, pOpIsType2, true},
		{pOpIncrement, pOpIsType2, true},
		{pOpIndex, pOpIsType2, true},
		{pOpLand, pOpIsType2, true},
		{pOpLEqual, pOpIsType2, true},
		{pOpLGreater, pOpIsType2, true},
		{pOpLLess, pOpIsType2, true},
		{pOpMid, pOpIsType2, true},
		{pOpLnot, pOpIsType2, true},
		{pOpLoadTable, pOpIsType2, true},
		{pOpLor, pOpIsType2, true},
		{pOpMatch, pOpIsType2, true},
		{pOpMod, pOpIsType2, true},
		{pOpMultiply, pOpIsType2, true},
		{pOpNand, pOpIsType2, true},
		{pOpNor, pOpIsType2, true},
		{pOpNot, pOpIsType2, true},
		{pOpObjectType, pOpIsType2, true},
		{pOpOr, pOpIsType2, true},
		{pOpPackage, pOpIsType2, true},
		{pOpVarPackage, pOpIsType2, true},
		{pOpRefOf, pOpIsType2, true},
		{pOpShiftLeft, pOpIsType2, true},
		{pOpShiftRight, pOpIsType2, true},
		{pOpSizeOf, pOpIsType2, true},
		{pOpStore, pOpIsType2, true},
		{pOpSubtract, pOpIsType2, true},
		{pOpTimer, pOpIsType2, true},
		{pOpToBCD, pOpIsType2, true},
		{pOpToBuffer, pOpIsType2, true},
		{pOpToDecimalString, pOpIsType2, true},
		{pOpToHexString, pOpIsType2, true},
		{pOpToInteger, pOpIsType2, true},
		{pOpToString, pOpIsType2, true},
		{pOpWait, pOpIsType2, true},
		{pOpXor, pOpIsType2, true},
		{pOpBytePrefix, pOpIsType2, false},
		// OpIsDataObject
		{pOpBytePrefix, pOpIsDataObject, true},
		{pOpWordPrefix, pOpIsDataObject, true},
		{pOpDwordPrefix, pOpIsDataObject, true},
		{pOpQwordPrefix, pOpIsDataObject, true},
		{pOpStringPrefix, pOpIsDataObject, true},
		{pOpZero, pOpIsDataObject, true},
		{pOpOne, pOpIsDataObject, true},
		{pOpOnes, pOpIsDataObject, true},
		{pOpRevision, pOpIsDataObject, true},
		{pOpBuffer, pOpIsDataObject, true},
		{pOpPackage, pOpIsDataObject, true},
		{pOpVarPackage, pOpIsDataObject, true},
		{pOpLor, pOpIsDataObject, false},
	}

	for specIndex, spec := range specs {
		if got := spec.testFn(spec.op); got != spec.want {
			t.Errorf("[spec %d] opcode %q: expected to get %t; got %t", specIndex, spec.op, spec.want, got)
		}
	}
}

func TestOpcodeName(t *testing.T) {
	for specIndex, spec := range pOpcodeTable {
		if got := pOpcodeName(spec.op); got != spec.opName {
			t.Errorf("[spec %d] expected OpcodeName(0x%x) to return %q; got %q", specIndex, spec.op, spec.opName, got)
		}
	}

	if exp, got := "unknown", pOpcodeName(0xf8); got != exp {
		t.Fatalf("expected OpcodeName to return %q for unknown opcode; got %q", exp, got)
	}
}

func TestOpcodeMap(t *testing.T) {
	freqs := make(map[uint8]int)
	for _, tableIndex := range opcodeMap {
		if tableIndex == badOpcode {
			continue
		}
		freqs[tableIndex]++
	}

	for _, tableIndex := range extendedOpcodeMap {
		if tableIndex == badOpcode {
			continue
		}
		freqs[tableIndex]++
	}

	for tableIndex, freq := range freqs {
		if freq > 1 {
			t.Errorf("[index 0x%x] found %d duplicate entries in opcodeMap/extendedOpcodeMap for %s", tableIndex, freq, pOpcodeTable[tableIndex].opName)
		}
	}
}
