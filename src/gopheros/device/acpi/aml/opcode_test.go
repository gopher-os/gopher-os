package aml

import "testing"

func TestOpcodeToString(t *testing.T) {
	if exp, got := "Acquire", opAcquire.String(); got != exp {
		t.Fatalf("expected opAcquire.toString() to return %q; got %q", exp, got)
	}

	if exp, got := "unknown", opcode(0xffff).String(); got != exp {
		t.Fatalf("expected opcode.String() to return %q; got %q", exp, got)
	}
}

func TestOpcodeIsX(t *testing.T) {
	specs := []struct {
		op     opcode
		testFn func(opcode) bool
		want   bool
	}{
		// opIsLocalArg
		{opLocal0, opIsLocalArg, true},
		{opLocal1, opIsLocalArg, true},
		{opLocal2, opIsLocalArg, true},
		{opLocal3, opIsLocalArg, true},
		{opLocal4, opIsLocalArg, true},
		{opLocal5, opIsLocalArg, true},
		{opLocal6, opIsLocalArg, true},
		{opLocal7, opIsLocalArg, true},
		{opArg0, opIsLocalArg, false},
		{opDivide, opIsLocalArg, false},
		// opIsMethodArg
		{opArg0, opIsMethodArg, true},
		{opArg1, opIsMethodArg, true},
		{opArg2, opIsMethodArg, true},
		{opArg3, opIsMethodArg, true},
		{opArg4, opIsMethodArg, true},
		{opArg5, opIsMethodArg, true},
		{opArg6, opIsMethodArg, true},
		{opLocal7, opIsMethodArg, false},
		{opIf, opIsMethodArg, false},
		// opIsArg
		{opLocal5, opIsArg, true},
		{opArg1, opIsArg, true},
		{opDivide, opIsArg, false},
		// opIsType2
		{opAcquire, opIsType2, true},
		{opAdd, opIsType2, true},
		{opAnd, opIsType2, true},
		{opBuffer, opIsType2, true},
		{opConcat, opIsType2, true},
		{opConcatRes, opIsType2, true},
		{opCondRefOf, opIsType2, true},
		{opCopyObject, opIsType2, true},
		{opDecrement, opIsType2, true},
		{opDerefOf, opIsType2, true},
		{opDivide, opIsType2, true},
		{opFindSetLeftBit, opIsType2, true},
		{opFindSetRightBit, opIsType2, true},
		{opFromBCD, opIsType2, true},
		{opIncrement, opIsType2, true},
		{opIndex, opIsType2, true},
		{opLand, opIsType2, true},
		{opLEqual, opIsType2, true},
		{opLGreater, opIsType2, true},
		{opLLess, opIsType2, true},
		{opMid, opIsType2, true},
		{opLnot, opIsType2, true},
		{opLoadTable, opIsType2, true},
		{opLor, opIsType2, true},
		{opMatch, opIsType2, true},
		{opMod, opIsType2, true},
		{opMultiply, opIsType2, true},
		{opNand, opIsType2, true},
		{opNor, opIsType2, true},
		{opNot, opIsType2, true},
		{opObjectType, opIsType2, true},
		{opOr, opIsType2, true},
		{opPackage, opIsType2, true},
		{opVarPackage, opIsType2, true},
		{opRefOf, opIsType2, true},
		{opShiftLeft, opIsType2, true},
		{opShiftRight, opIsType2, true},
		{opSizeOf, opIsType2, true},
		{opStore, opIsType2, true},
		{opSubtract, opIsType2, true},
		{opTimer, opIsType2, true},
		{opToBCD, opIsType2, true},
		{opToBuffer, opIsType2, true},
		{opToDecimalString, opIsType2, true},
		{opToHexString, opIsType2, true},
		{opToInteger, opIsType2, true},
		{opToString, opIsType2, true},
		{opWait, opIsType2, true},
		{opXor, opIsType2, true},
		{opBytePrefix, opIsType2, false},
		// opIsDataObject
		{opBytePrefix, opIsDataObject, true},
		{opWordPrefix, opIsDataObject, true},
		{opDwordPrefix, opIsDataObject, true},
		{opQwordPrefix, opIsDataObject, true},
		{opStringPrefix, opIsDataObject, true},
		{opZero, opIsDataObject, true},
		{opOne, opIsDataObject, true},
		{opOnes, opIsDataObject, true},
		{opRevision, opIsDataObject, true},
		{opBuffer, opIsDataObject, true},
		{opPackage, opIsDataObject, true},
		{opVarPackage, opIsDataObject, true},
		{opLor, opIsDataObject, false},
		// opIsBufferField
		{opCreateField, opIsBufferField, true},
		{opCreateBitField, opIsBufferField, true},
		{opCreateByteField, opIsBufferField, true},
		{opCreateWordField, opIsBufferField, true},
		{opCreateDWordField, opIsBufferField, true},
		{opCreateQWordField, opIsBufferField, true},
		{opRevision, opIsBufferField, false},
	}

	for specIndex, spec := range specs {
		if got := spec.testFn(spec.op); got != spec.want {
			t.Errorf("[spec %d] opcode %q: expected to get %t; got %t", specIndex, spec.op, spec.want, got)
		}
	}
}

func TestOpArgFlagToString(t *testing.T) {
	specs := map[opArgFlag]string{
		opArgTermList:   "opArgTermList",
		opArgTermObj:    "opArgTermObj",
		opArgByteList:   "opArgByteList",
		opArgPackage:    "opArgPackage",
		opArgString:     "opArgString",
		opArgByteData:   "opArgByteData",
		opArgWord:       "opArgWord",
		opArgDword:      "opArgDword",
		opArgQword:      "opArgQword",
		opArgNameString: "opArgNameString",
		opArgSuperName:  "opArgSuperName",
		opArgSimpleName: "opArgSimpleName",
		opArgDataRefObj: "opArgDataRefObj",
		opArgTarget:     "opArgTarget",
		opArgFieldList:  "opArgFieldList",
		opArgFlag(0xff): "",
	}

	for flag, want := range specs {
		if got := flag.String(); got != want {
			t.Errorf("expected %q; got %q", want, got)
		}
	}
}

// TestFindUnmappedOpcodes is a helper test that pinpoints opcodes that have
// not yet been mapped via an opcode table. This test will be removed once all
// opcodes are supported.
func TestFindUnmappedOpcodes(t *testing.T) {
	//t.SkipNow()
	for opIndex, opRef := range opcodeMap {
		if opRef != badOpcode {
			continue
		}

		for tabIndex, info := range opcodeTable {
			if uint16(info.op) == uint16(opIndex) {
				t.Errorf("set opcodeMap[0x%02x] = 0x%02x // %s\n", opIndex, tabIndex, info.op.String())
				break
			}
		}
	}

	for opIndex, opRef := range extendedOpcodeMap {
		// 0xff (opOnes) is defined in opcodeTable
		if opRef != badOpcode || opIndex == 0 {
			continue
		}

		opIndex += 0xff
		for tabIndex, info := range opcodeTable {
			if uint16(info.op) == uint16(opIndex) {
				t.Errorf("set extendedOpcodeMap[0x%02x] = 0x%02x // %s\n", opIndex-0xff, tabIndex, info.op.String())
				break
			}
		}
	}
}
