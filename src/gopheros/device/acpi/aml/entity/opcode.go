package entity

// AMLOpcode describes an AML opcode. While AML supports 256 opcodes, some of
// them are specified using a combination of an extension prefix and a code. To
// map each opcode into a single unique value the parser uses an uint16
// representation of the opcode values.
type AMLOpcode uint16

const (
	// Regular opcode list
	OpZero             = AMLOpcode(0x00)
	OpOne              = AMLOpcode(0x01)
	OpAlias            = AMLOpcode(0x06)
	OpName             = AMLOpcode(0x08)
	OpBytePrefix       = AMLOpcode(0x0a)
	OpWordPrefix       = AMLOpcode(0x0b)
	OpDwordPrefix      = AMLOpcode(0x0c)
	OpStringPrefix     = AMLOpcode(0x0d)
	OpQwordPrefix      = AMLOpcode(0x0e)
	OpScope            = AMLOpcode(0x10)
	OpBuffer           = AMLOpcode(0x11)
	OpPackage          = AMLOpcode(0x12)
	OpVarPackage       = AMLOpcode(0x13)
	OpMethod           = AMLOpcode(0x14)
	OpExternal         = AMLOpcode(0x15)
	OpLocal0           = AMLOpcode(0x60)
	OpLocal1           = AMLOpcode(0x61)
	OpLocal2           = AMLOpcode(0x62)
	OpLocal3           = AMLOpcode(0x63)
	OpLocal4           = AMLOpcode(0x64)
	OpLocal5           = AMLOpcode(0x65)
	OpLocal6           = AMLOpcode(0x66)
	OpLocal7           = AMLOpcode(0x67)
	OpArg0             = AMLOpcode(0x68)
	OpArg1             = AMLOpcode(0x69)
	OpArg2             = AMLOpcode(0x6a)
	OpArg3             = AMLOpcode(0x6b)
	OpArg4             = AMLOpcode(0x6c)
	OpArg5             = AMLOpcode(0x6d)
	OpArg6             = AMLOpcode(0x6e)
	OpStore            = AMLOpcode(0x70)
	OpRefOf            = AMLOpcode(0x71)
	OpAdd              = AMLOpcode(0x72)
	OpConcat           = AMLOpcode(0x73)
	OpSubtract         = AMLOpcode(0x74)
	OpIncrement        = AMLOpcode(0x75)
	OpDecrement        = AMLOpcode(0x76)
	OpMultiply         = AMLOpcode(0x77)
	OpDivide           = AMLOpcode(0x78)
	OpShiftLeft        = AMLOpcode(0x79)
	OpShiftRight       = AMLOpcode(0x7a)
	OpAnd              = AMLOpcode(0x7b)
	OpNand             = AMLOpcode(0x7c)
	OpOr               = AMLOpcode(0x7d)
	OpNor              = AMLOpcode(0x7e)
	OpXor              = AMLOpcode(0x7f)
	OpNot              = AMLOpcode(0x80)
	OpFindSetLeftBit   = AMLOpcode(0x81)
	OpFindSetRightBit  = AMLOpcode(0x82)
	OpDerefOf          = AMLOpcode(0x83)
	OpConcatRes        = AMLOpcode(0x84)
	OpMod              = AMLOpcode(0x85)
	OpNotify           = AMLOpcode(0x86)
	OpSizeOf           = AMLOpcode(0x87)
	OpIndex            = AMLOpcode(0x88)
	OpMatch            = AMLOpcode(0x89)
	OpCreateDWordField = AMLOpcode(0x8a)
	OpCreateWordField  = AMLOpcode(0x8b)
	OpCreateByteField  = AMLOpcode(0x8c)
	OpCreateBitField   = AMLOpcode(0x8d)
	OpObjectType       = AMLOpcode(0x8e)
	OpCreateQWordField = AMLOpcode(0x8f)
	OpLand             = AMLOpcode(0x90)
	OpLor              = AMLOpcode(0x91)
	OpLnot             = AMLOpcode(0x92)
	OpLEqual           = AMLOpcode(0x93)
	OpLGreater         = AMLOpcode(0x94)
	OpLLess            = AMLOpcode(0x95)
	OpToBuffer         = AMLOpcode(0x96)
	OpToDecimalString  = AMLOpcode(0x97)
	OpToHexString      = AMLOpcode(0x98)
	OpToInteger        = AMLOpcode(0x99)
	OpToString         = AMLOpcode(0x9c)
	OpCopyObject       = AMLOpcode(0x9d)
	OpMid              = AMLOpcode(0x9e)
	OpContinue         = AMLOpcode(0x9f)
	OpIf               = AMLOpcode(0xa0)
	OpElse             = AMLOpcode(0xa1)
	OpWhile            = AMLOpcode(0xa2)
	OpNoop             = AMLOpcode(0xa3)
	OpReturn           = AMLOpcode(0xa4)
	OpBreak            = AMLOpcode(0xa5)
	OpBreakPoint       = AMLOpcode(0xcc)
	OpOnes             = AMLOpcode(0xff)
	// Extended opcodes
	OpMutex       = AMLOpcode(0xff + 0x01)
	OpEvent       = AMLOpcode(0xff + 0x02)
	OpCondRefOf   = AMLOpcode(0xff + 0x12)
	OpCreateField = AMLOpcode(0xff + 0x13)
	OpLoadTable   = AMLOpcode(0xff + 0x1f)
	OpLoad        = AMLOpcode(0xff + 0x20)
	OpStall       = AMLOpcode(0xff + 0x21)
	OpSleep       = AMLOpcode(0xff + 0x22)
	OpAcquire     = AMLOpcode(0xff + 0x23)
	OpSignal      = AMLOpcode(0xff + 0x24)
	OpWait        = AMLOpcode(0xff + 0x25)
	OpReset       = AMLOpcode(0xff + 0x26)
	OpRelease     = AMLOpcode(0xff + 0x27)
	OpFromBCD     = AMLOpcode(0xff + 0x28)
	OpToBCD       = AMLOpcode(0xff + 0x29)
	OpUnload      = AMLOpcode(0xff + 0x2a)
	OpRevision    = AMLOpcode(0xff + 0x30)
	OpDebug       = AMLOpcode(0xff + 0x31)
	OpFatal       = AMLOpcode(0xff + 0x32)
	OpTimer       = AMLOpcode(0xff + 0x33)
	OpOpRegion    = AMLOpcode(0xff + 0x80)
	OpField       = AMLOpcode(0xff + 0x81)
	OpDevice      = AMLOpcode(0xff + 0x82)
	OpProcessor   = AMLOpcode(0xff + 0x83)
	OpPowerRes    = AMLOpcode(0xff + 0x84)
	OpThermalZone = AMLOpcode(0xff + 0x85)
	OpIndexField  = AMLOpcode(0xff + 0x86)
	OpBankField   = AMLOpcode(0xff + 0x87)
	OpDataRegion  = AMLOpcode(0xff + 0x88)
)

// OpIsLocalArg returns true if this opcode represents any of the supported local
// function args 0 to 7.
func OpIsLocalArg(op AMLOpcode) bool {
	return op >= OpLocal0 && op <= OpLocal7
}

// OpIsMethodArg returns true if this opcode represents any of the supported
// input function args 0 to 6.
func OpIsMethodArg(op AMLOpcode) bool {
	return op >= OpArg0 && op <= OpArg6
}

// OpIsArg returns true if this opcode is either a local or a method arg.
func OpIsArg(op AMLOpcode) bool {
	return OpIsLocalArg(op) || OpIsMethodArg(op)
}

// OpIsDataObject returns true if this opcode is part of a DataObject definition
//
// Grammar:
// DataObject := ComputationalData | DefPackage | DefVarPackage
// ComputationalData := ByteConst | WordConst | DWordConst | QWordConst | String | ConstObj | RevisionOp | DefBuffer
// ConstObj := ZeroOp | OneOp | OnesOp
func OpIsDataObject(op AMLOpcode) bool {
	switch op {
	case OpBytePrefix, OpWordPrefix, OpDwordPrefix, OpQwordPrefix, OpStringPrefix,
		OpZero, OpOne, OpOnes, OpRevision, OpBuffer, OpPackage, OpVarPackage:
		return true
	default:
		return false
	}
}

// OpIsType2 returns true if this is a Type2Opcode.
//
// Grammar:
// Type2Opcode := DefAcquire | DefAdd | DefAnd | DefBuffer | DefConcat |
//  DefConcatRes | DefCondRefOf | DefCopyObject | DefDecrement |
//  DefDerefOf | DefDivide | DefFindSetLeftBit | DefFindSetRightBit |
//  DefFromBCD | DefIncrement | DefIndex | DefLAnd | DefLEqual |
//  DefLGreater | DefLGreaterEqual | DefLLess | DefLLessEqual | DefMid |
//  DefLNot | DefLNotEqual | DefLoadTable | DefLOr | DefMatch | DefMod |
//  DefMultiply | DefNAnd | DefNOr | DefNot | DefObjectType | DefOr |
//  DefPackage | DefVarPackage | DefRefOf | DefShiftLeft | DefShiftRight |
//  DefSizeOf | DefStore | DefSubtract | DefTimer | DefToBCD | DefToBuffer |
//  DefToDecimalString | DefToHexString | DefToInteger | DefToString |
//  DefWait | DefXOr
func OpIsType2(op AMLOpcode) bool {
	switch op {
	case OpAcquire, OpAdd, OpAnd, OpBuffer, OpConcat,
		OpConcatRes, OpCondRefOf, OpCopyObject, OpDecrement,
		OpDerefOf, OpDivide, OpFindSetLeftBit, OpFindSetRightBit,
		OpFromBCD, OpIncrement, OpIndex, OpLand, OpLEqual,
		OpLGreater, OpLLess, OpMid,
		OpLnot, OpLoadTable, OpLor, OpMatch, OpMod,
		OpMultiply, OpNand, OpNor, OpNot, OpObjectType, OpOr,
		OpPackage, OpVarPackage, OpRefOf, OpShiftLeft, OpShiftRight,
		OpSizeOf, OpStore, OpSubtract, OpTimer, OpToBCD, OpToBuffer,
		OpToDecimalString, OpToHexString, OpToInteger, OpToString,
		OpWait, OpXor:
		return true
	default:
		return false
	}
}

// OpIsBufferField returens true if this opcode describes a
// buffer field creation operation.
func OpIsBufferField(op AMLOpcode) bool {
	switch op {
	case OpCreateField, OpCreateBitField, OpCreateByteField, OpCreateWordField, OpCreateDWordField, OpCreateQWordField:
		return true
	default:
		return false
	}
}
