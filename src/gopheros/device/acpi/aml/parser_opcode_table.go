package aml

// List of AML opcodes.
const (
	// Regular opcode list
	pOpZero             = uint16(0x00)
	pOpOne              = uint16(0x01)
	pOpAlias            = uint16(0x06)
	pOpName             = uint16(0x08)
	pOpBytePrefix       = uint16(0x0a)
	pOpWordPrefix       = uint16(0x0b)
	pOpDwordPrefix      = uint16(0x0c)
	pOpStringPrefix     = uint16(0x0d)
	pOpQwordPrefix      = uint16(0x0e)
	pOpScope            = uint16(0x10)
	pOpBuffer           = uint16(0x11)
	pOpPackage          = uint16(0x12)
	pOpVarPackage       = uint16(0x13)
	pOpMethod           = uint16(0x14)
	pOpExternal         = uint16(0x15)
	pOpLocal0           = uint16(0x60)
	pOpLocal1           = uint16(0x61)
	pOpLocal2           = uint16(0x62)
	pOpLocal3           = uint16(0x63)
	pOpLocal4           = uint16(0x64)
	pOpLocal5           = uint16(0x65)
	pOpLocal6           = uint16(0x66)
	pOpLocal7           = uint16(0x67)
	pOpArg0             = uint16(0x68)
	pOpArg1             = uint16(0x69)
	pOpArg2             = uint16(0x6a)
	pOpArg3             = uint16(0x6b)
	pOpArg4             = uint16(0x6c)
	pOpArg5             = uint16(0x6d)
	pOpArg6             = uint16(0x6e)
	pOpStore            = uint16(0x70)
	pOpRefOf            = uint16(0x71)
	pOpAdd              = uint16(0x72)
	pOpConcat           = uint16(0x73)
	pOpSubtract         = uint16(0x74)
	pOpIncrement        = uint16(0x75)
	pOpDecrement        = uint16(0x76)
	pOpMultiply         = uint16(0x77)
	pOpDivide           = uint16(0x78)
	pOpShiftLeft        = uint16(0x79)
	pOpShiftRight       = uint16(0x7a)
	pOpAnd              = uint16(0x7b)
	pOpNand             = uint16(0x7c)
	pOpOr               = uint16(0x7d)
	pOpNor              = uint16(0x7e)
	pOpXor              = uint16(0x7f)
	pOpNot              = uint16(0x80)
	pOpFindSetLeftBit   = uint16(0x81)
	pOpFindSetRightBit  = uint16(0x82)
	pOpDerefOf          = uint16(0x83)
	pOpConcatRes        = uint16(0x84)
	pOpMod              = uint16(0x85)
	pOpNotify           = uint16(0x86)
	pOpSizeOf           = uint16(0x87)
	pOpIndex            = uint16(0x88)
	pOpMatch            = uint16(0x89)
	pOpCreateDWordField = uint16(0x8a)
	pOpCreateWordField  = uint16(0x8b)
	pOpCreateByteField  = uint16(0x8c)
	pOpCreateBitField   = uint16(0x8d)
	pOpObjectType       = uint16(0x8e)
	pOpCreateQWordField = uint16(0x8f)
	pOpLand             = uint16(0x90)
	pOpLor              = uint16(0x91)
	pOpLnot             = uint16(0x92)
	pOpLEqual           = uint16(0x93)
	pOpLGreater         = uint16(0x94)
	pOpLLess            = uint16(0x95)
	pOpToBuffer         = uint16(0x96)
	pOpToDecimalString  = uint16(0x97)
	pOpToHexString      = uint16(0x98)
	pOpToInteger        = uint16(0x99)
	pOpToString         = uint16(0x9c)
	pOpCopyObject       = uint16(0x9d)
	pOpMid              = uint16(0x9e)
	pOpContinue         = uint16(0x9f)
	pOpIf               = uint16(0xa0)
	pOpElse             = uint16(0xa1)
	pOpWhile            = uint16(0xa2)
	pOpNoop             = uint16(0xa3)
	pOpReturn           = uint16(0xa4)
	pOpBreak            = uint16(0xa5)
	pOpBreakPoint       = uint16(0xcc)
	pOpOnes             = uint16(0xff)
	// Extended opcodes
	pOpMutex       = uint16(0xff + 0x01)
	pOpEvent       = uint16(0xff + 0x02)
	pOpCondRefOf   = uint16(0xff + 0x12)
	pOpCreateField = uint16(0xff + 0x13)
	pOpLoadTable   = uint16(0xff + 0x1f)
	pOpLoad        = uint16(0xff + 0x20)
	pOpStall       = uint16(0xff + 0x21)
	pOpSleep       = uint16(0xff + 0x22)
	pOpAcquire     = uint16(0xff + 0x23)
	pOpSignal      = uint16(0xff + 0x24)
	pOpWait        = uint16(0xff + 0x25)
	pOpReset       = uint16(0xff + 0x26)
	pOpRelease     = uint16(0xff + 0x27)
	pOpFromBCD     = uint16(0xff + 0x28)
	pOpToBCD       = uint16(0xff + 0x29)
	pOpUnload      = uint16(0xff + 0x2a)
	pOpRevision    = uint16(0xff + 0x30)
	pOpDebug       = uint16(0xff + 0x31)
	pOpFatal       = uint16(0xff + 0x32)
	pOpTimer       = uint16(0xff + 0x33)
	pOpOpRegion    = uint16(0xff + 0x80)
	pOpField       = uint16(0xff + 0x81)
	pOpDevice      = uint16(0xff + 0x82)
	pOpProcessor   = uint16(0xff + 0x83)
	pOpPowerRes    = uint16(0xff + 0x84)
	pOpThermalZone = uint16(0xff + 0x85)
	pOpIndexField  = uint16(0xff + 0x86)
	pOpBankField   = uint16(0xff + 0x87)
	pOpDataRegion  = uint16(0xff + 0x88)
	// Special internal opcodes which are not part of the spec; these are
	// for internal use by the AML parser.
	pOpIntScopeBlock           = uint16(0xff + 0xf7)
	pOpIntByteList             = uint16(0xff + 0xf8)
	pOpIntConnection           = uint16(0xff + 0xf9)
	pOpIntNamedField           = uint16(0xff + 0xfa)
	pOpIntResolvedNamePath     = uint16(0xff + 0xfb)
	pOpIntNamePath             = uint16(0xff + 0xfc)
	pOpIntNamePathOrMethodCall = uint16(0xff + 0xfd)
	pOpIntMethodCall           = uint16(0xff + 0xfe)
	// Sentinel value to indicate freed objects
	pOpIntFreedObject = uint16(0xff + 0xff)
)

// pOpIsLocalArg returns true if this opcode represents any of the supported local
// function args 0 to 7.
func pOpIsLocalArg(op uint16) bool {
	return op >= pOpLocal0 && op <= pOpLocal7
}

// pOpIsMethodArg returns true if this opcode represents any of the supported
// input function args 0 to 6.
func pOpIsMethodArg(op uint16) bool {
	return op >= pOpArg0 && op <= pOpArg6
}

// pOpIsArg returns true if this opcode is either a local or a method arg.
func pOpIsArg(op uint16) bool {
	return pOpIsLocalArg(op) || pOpIsMethodArg(op)
}

// pOpIsType2 returns true if this is a Type2Opcode.
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
func pOpIsType2(op uint16) bool {
	switch op {
	case pOpAcquire, pOpAdd, pOpAnd, pOpBuffer, pOpConcat,
		pOpConcatRes, pOpCondRefOf, pOpCopyObject, pOpDecrement,
		pOpDerefOf, pOpDivide, pOpFindSetLeftBit, pOpFindSetRightBit,
		pOpFromBCD, pOpIncrement, pOpIndex, pOpLand, pOpLEqual,
		pOpLGreater, pOpLLess, pOpMid,
		pOpLnot, pOpLoadTable, pOpLor, pOpMatch, pOpMod,
		pOpMultiply, pOpNand, pOpNor, pOpNot, pOpObjectType, pOpOr,
		pOpPackage, pOpVarPackage, pOpRefOf, pOpShiftLeft, pOpShiftRight,
		pOpSizeOf, pOpStore, pOpSubtract, pOpTimer, pOpToBCD, pOpToBuffer,
		pOpToDecimalString, pOpToHexString, pOpToInteger, pOpToString,
		pOpWait, pOpXor:
		return true
	default:
		return false
	}
}

// pOpIsDataObject returns true if this opcode is part of a DataObject definition
//
// Grammar:
// DataObject := ComputationalData | DefPackage | DefVarPackage
// ComputationalData := ByteConst | WordConst | DWordConst | QWordConst | String | ConstObj | RevisionOp | DefBuffer
// ConstObj := ZeroOp | OneOp | OnesOp
func pOpIsDataObject(op uint16) bool {
	switch op {
	case pOpBytePrefix, pOpWordPrefix, pOpDwordPrefix, pOpQwordPrefix, pOpStringPrefix,
		pOpZero, pOpOne, pOpOnes, pOpRevision, pOpBuffer, pOpPackage, pOpVarPackage:
		return true
	default:
		return false
	}
}

const (
	badOpcode   = 0xff
	extOpPrefix = 0x5b
)

// pOpFlag specifies a list of OR-able flags that describe the object
// type/attributes generated by a particular opcode.
type pOpFlag uint8

const (
	pOpFlagNamed = 1 << iota
	pOpFlagConstant
	pOpFlagReference
	pOpFlagCreate
	pOpFlagExecutable
	pOpFlagScoped
	pOpFlagDeferParsing
)

// pOpArgTypeList encodes up to 7 opArgFlag values in a uint64 value.
type pOpArgTypeList uint64

// argCount returns the number of encoded args in the given arg type list.
func (fl pOpArgTypeList) argCount() (count uint8) {
	// Each argument is specified using 8 bits with 0x0 indicating the end of the
	// argument list
	for ; fl&0xf != 0; fl, count = fl>>8, count+1 {
	}

	return count
}

// arg returns the arg type for argument "num" where num is the 0-based index
// of the argument to return. The allowed values for num are 0-6.
func (fl pOpArgTypeList) arg(num uint8) pArgType {
	return pArgType((fl >> (num * 8)) & 0xf)
}

// pArgType represents the type of an argument expected by a particular opcode.
type pArgType uint8

// The list of supported opArgFlag values.
const (
	_ pArgType = iota
	pArgTypeTermList
	pArgTypeTermArg
	pArgTypeByteList
	pArgTypeString
	pArgTypeByteData
	pArgTypeWordData
	pArgTypeDwordData
	pArgTypeQwordData
	pArgTypeNameString
	pArgTypeSuperName
	pArgTypeSimpleName
	pArgTypeDataRefObj
	pArgTypeTarget
	pArgTypeFieldList
	pArgTypePkgLen
)

func makeArg0() pOpArgTypeList              { return 0 }
func makeArg1(arg0 pArgType) pOpArgTypeList { return pOpArgTypeList(arg0) }
func makeArg2(arg0, arg1 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}
func makeArg3(arg0, arg1, arg2 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg2)<<16 | pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}
func makeArg4(arg0, arg1, arg2, arg3 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg3)<<24 | pOpArgTypeList(arg2)<<16 | pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}
func makeArg5(arg0, arg1, arg2, arg3, arg4 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg4)<<32 | pOpArgTypeList(arg3)<<24 | pOpArgTypeList(arg2)<<16 | pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}
func makeArg6(arg0, arg1, arg2, arg3, arg4, arg5 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg5)<<40 | pOpArgTypeList(arg4)<<32 | pOpArgTypeList(arg3)<<24 | pOpArgTypeList(arg2)<<16 | pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}
func makeArg7(arg0, arg1, arg2, arg3, arg4, arg5, arg6 pArgType) pOpArgTypeList {
	return pOpArgTypeList(arg6)<<48 | pOpArgTypeList(arg5)<<40 | pOpArgTypeList(arg4)<<32 | pOpArgTypeList(arg3)<<24 | pOpArgTypeList(arg2)<<16 | pOpArgTypeList(arg1)<<8 | pOpArgTypeList(arg0)
}

// pOpcodeInfo contains all known information about an opcode,
// its argument count and types as well as the type of object
// represented by it.
type pOpcodeInfo struct {
	op     uint16
	opName string

	flags    pOpFlag
	argFlags pOpArgTypeList
}

// The opcode table contains all opcode-related information that the parser knows.
// This table is modeled after a similar table used in the acpica implementation.
var pOpcodeTable = []pOpcodeInfo{
	/*0x00*/ {pOpZero, "Zero", pOpFlagConstant, makeArg0()},
	/*0x01*/ {pOpOne, "One", pOpFlagConstant, makeArg0()},
	/*0x02*/ {pOpAlias, "Alias", pOpFlagNamed, makeArg2(pArgTypeNameString, pArgTypeNameString)},
	/*0x03*/ {pOpName, "Name", pOpFlagNamed, makeArg2(pArgTypeNameString, pArgTypeDataRefObj)},
	/*0x04*/ {pOpBytePrefix, "BytePrefix", pOpFlagConstant, makeArg1(pArgTypeByteData)},
	/*0x05*/ {pOpWordPrefix, "WordPrefix", pOpFlagConstant, makeArg1(pArgTypeWordData)},
	/*0x06*/ {pOpDwordPrefix, "DwordPrefix", pOpFlagConstant, makeArg1(pArgTypeDwordData)},
	/*0x07*/ {pOpStringPrefix, "StringPrefix", pOpFlagConstant, makeArg1(pArgTypeString)},
	/*0x08*/ {pOpQwordPrefix, "QwordPrefix", pOpFlagConstant, makeArg1(pArgTypeQwordData)},
	/*0x09*/ {pOpScope, "Scope", 0, makeArg3(pArgTypePkgLen, pArgTypeNameString, pArgTypeTermList)},
	/*0x0a*/ {pOpBuffer, "Buffer", pOpFlagDeferParsing | pOpFlagCreate, makeArg3(pArgTypePkgLen, pArgTypeTermArg, pArgTypeByteList)},
	/*0x0b*/ {pOpPackage, "Package", pOpFlagCreate, makeArg3(pArgTypePkgLen, pArgTypeByteData, pArgTypeTermList)},
	/*0x0c*/ {pOpVarPackage, "VarPackage", pOpFlagCreate, makeArg3(pArgTypePkgLen, pArgTypeByteData, pArgTypeTermList)},
	/*0x0d*/ {pOpMethod, "Method", pOpFlagNamed | pOpFlagScoped, makeArg4(pArgTypePkgLen, pArgTypeNameString, pArgTypeByteData, pArgTypeTermList)},
	/*0x0e*/ {pOpExternal, "External", pOpFlagNamed, makeArg3(pArgTypeNameString, pArgTypeByteData, pArgTypeByteData)},
	/*0x0f*/ {pOpLocal0, "Local0", pOpFlagExecutable, makeArg0()},
	/*0x10*/ {pOpLocal1, "Local1", pOpFlagExecutable, makeArg0()},
	/*0x11*/ {pOpLocal2, "Local2", pOpFlagExecutable, makeArg0()},
	/*0x12*/ {pOpLocal3, "Local3", pOpFlagExecutable, makeArg0()},
	/*0x13*/ {pOpLocal4, "Local4", pOpFlagExecutable, makeArg0()},
	/*0120*/ {pOpLocal5, "Local5", pOpFlagExecutable, makeArg0()},
	/*0x15*/ {pOpLocal6, "Local6", pOpFlagExecutable, makeArg0()},
	/*0x16*/ {pOpLocal7, "Local7", pOpFlagExecutable, makeArg0()},
	/*0x17*/ {pOpArg0, "Arg0", pOpFlagExecutable, makeArg0()},
	/*0x18*/ {pOpArg1, "Arg1", pOpFlagExecutable, makeArg0()},
	/*0x19*/ {pOpArg2, "Arg2", pOpFlagExecutable, makeArg0()},
	/*0x1a*/ {pOpArg3, "Arg3", pOpFlagExecutable, makeArg0()},
	/*0x1b*/ {pOpArg4, "Arg4", pOpFlagExecutable, makeArg0()},
	/*0x1c*/ {pOpArg5, "Arg5", pOpFlagExecutable, makeArg0()},
	/*0x1d*/ {pOpArg6, "Arg6", pOpFlagExecutable, makeArg0()},
	/*0x1e*/ {pOpStore, "Store", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeSuperName)},
	/*0x1f*/ {pOpRefOf, "RefOf", pOpFlagReference | pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x20*/ {pOpAdd, "Add", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x21*/ {pOpConcat, "Concat", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x22*/ {pOpSubtract, "Subtract", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x23*/ {pOpIncrement, "Increment", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x24*/ {pOpDecrement, "Decrement", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x25*/ {pOpMultiply, "Multiply", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x26*/ {pOpDivide, "Divide", pOpFlagExecutable, makeArg4(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget, pArgTypeTarget)},
	/*0x27*/ {pOpShiftLeft, "ShiftLeft", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x28*/ {pOpShiftRight, "ShiftRight", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x29*/ {pOpAnd, "And", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x2a*/ {pOpNand, "Nand", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x2b*/ {pOpOr, "Or", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x2c*/ {pOpNor, "Nor", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x2d*/ {pOpXor, "Xor", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x2e*/ {pOpNot, "Not", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x2f*/ {pOpFindSetLeftBit, "FindSetLeftBit", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x30*/ {pOpFindSetRightBit, "FindSetRightBit", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x31*/ {pOpDerefOf, "DerefOf", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x32*/ {pOpConcatRes, "ConcatRes", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x33*/ {pOpMod, "Mod", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x34*/ {pOpNotify, "Notify", pOpFlagExecutable, makeArg2(pArgTypeSuperName, pArgTypeTermArg)},
	/*0x35*/ {pOpSizeOf, "SizeOf", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x36*/ {pOpIndex, "Index", pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x37*/ {pOpMatch, "Match", pOpFlagExecutable, makeArg6(pArgTypeTermArg, pArgTypeByteData, pArgTypeTermArg, pArgTypeByteData, pArgTypeTermArg, pArgTypeTermArg)},
	/*0x38*/ {pOpCreateDWordField, "CreateDWordField", pOpFlagCreate | pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x39*/ {pOpCreateWordField, "CreateWordField", pOpFlagCreate | pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x3a*/ {pOpCreateByteField, "CreateByteField", pOpFlagCreate | pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x3b*/ {pOpCreateBitField, "CreateBitField", pOpFlagCreate | pOpFlagExecutable, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x3c*/ {pOpObjectType, "ObjectType", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x3d*/ {pOpCreateQWordField, "CreateQWordField", pOpFlagCreate, makeArg3(pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x3e*/ {pOpLand, "Land", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTermArg)},
	/*0x3f*/ {pOpLor, "Lor", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTermArg)},
	/*0x40*/ {pOpLnot, "Lnot", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x41*/ {pOpLEqual, "LEqual", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTermArg)},
	/*0x42*/ {pOpLGreater, "LGreater", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTermArg)},
	/*0x43*/ {pOpLLess, "LLess", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTermArg)},
	/*0x44*/ {pOpToBuffer, "ToBuffer", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x45*/ {pOpToDecimalString, "ToDecimalString", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x46*/ {pOpToHexString, "ToHexString", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x47*/ {pOpToInteger, "ToInteger", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x48*/ {pOpToString, "ToString", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x49*/ {pOpCopyObject, "CopyObject", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeSimpleName)},
	/*0x4a*/ {pOpMid, "Mid", pOpFlagExecutable, makeArg4(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTarget)},
	/*0x4b*/ {pOpContinue, "Continue", pOpFlagExecutable, makeArg0()},
	/*0x4c*/ {pOpIf, "If", pOpFlagExecutable | pOpFlagScoped, makeArg3(pArgTypePkgLen, pArgTypeTermArg, pArgTypeTermList)},
	/*0x4d*/ {pOpElse, "Else", pOpFlagExecutable | pOpFlagScoped, makeArg2(pArgTypePkgLen, pArgTypeTermList)},
	/*0x4e*/ {pOpWhile, "While", pOpFlagDeferParsing | pOpFlagExecutable | pOpFlagScoped, makeArg3(pArgTypePkgLen, pArgTypeTermArg, pArgTypeTermList)},
	/*0x4f*/ {pOpNoop, "Noop", pOpFlagExecutable, makeArg0()},
	/*0x50*/ {pOpReturn, "Return", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x51*/ {pOpBreak, "Break", pOpFlagExecutable, makeArg0()},
	/*0x52*/ {pOpBreakPoint, "BreakPoint", pOpFlagExecutable, makeArg0()},
	/*0x53*/ {pOpOnes, "Ones", pOpFlagConstant, makeArg0()},
	/*0x54*/ {pOpMutex, "Mutex", pOpFlagNamed, makeArg2(pArgTypeNameString, pArgTypeByteData)},
	/*0x55*/ {pOpEvent, "Event", pOpFlagNamed, makeArg1(pArgTypeNameString)},
	/*0x56*/ {pOpCondRefOf, "CondRefOf", pOpFlagExecutable, makeArg2(pArgTypeSuperName, pArgTypeSuperName)},
	/*0x57*/ {pOpCreateField, "CreateField", pOpFlagExecutable, makeArg4(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeNameString)},
	/*0x58*/ {pOpLoadTable, "LoadTable", pOpFlagExecutable, makeArg7(pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg)},
	/*0x59*/ {pOpLoad, "Load", pOpFlagExecutable, makeArg2(pArgTypeNameString, pArgTypeSuperName)},
	/*0x5a*/ {pOpStall, "Stall", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x5b*/ {pOpSleep, "Sleep", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x5c*/ {pOpAcquire, "Acquire", pOpFlagExecutable, makeArg2(pArgTypeSuperName, pArgTypeWordData)},
	/*0x5d*/ {pOpSignal, "Signal", pOpFlagExecutable, makeArg1(pArgTypeTermArg)},
	/*0x5e*/ {pOpWait, "Wait", pOpFlagExecutable, makeArg2(pArgTypeSuperName, pArgTypeTermArg)},
	/*0x5f*/ {pOpReset, "Reset", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x60*/ {pOpRelease, "Release", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x61*/ {pOpFromBCD, "FromBCD", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x62*/ {pOpToBCD, "ToBCD", pOpFlagExecutable, makeArg2(pArgTypeTermArg, pArgTypeTarget)},
	/*0x63*/ {pOpUnload, "Unload", pOpFlagExecutable, makeArg1(pArgTypeSuperName)},
	/*0x64*/ {pOpRevision, "Revision", pOpFlagConstant | pOpFlagExecutable, makeArg0()},
	/*0x65*/ {pOpDebug, "Debug", pOpFlagExecutable, makeArg0()},
	/*0x66*/ {pOpFatal, "Fatal", pOpFlagExecutable, makeArg3(pArgTypeByteData, pArgTypeDwordData, pArgTypeTermArg)},
	/*0x67*/ {pOpTimer, "Timer", pOpFlagExecutable, makeArg0()},
	/*0x68*/ {pOpOpRegion, "OpRegion", pOpFlagNamed, makeArg4(pArgTypeNameString, pArgTypeByteData, pArgTypeTermArg, pArgTypeTermArg)},
	/*0x69*/ {pOpField, "Field", pOpFlagCreate, makeArg4(pArgTypePkgLen, pArgTypeNameString, pArgTypeByteData, pArgTypeFieldList)},
	/*0x6a*/ {pOpDevice, "Device", pOpFlagNamed | pOpFlagScoped, makeArg3(pArgTypePkgLen, pArgTypeNameString, pArgTypeTermList)},
	/*0x6b*/ {pOpProcessor, "Processor", pOpFlagNamed | pOpFlagScoped, makeArg6(pArgTypePkgLen, pArgTypeNameString, pArgTypeByteData, pArgTypeDwordData, pArgTypeByteData, pArgTypeTermList)},
	/*0x6c*/ {pOpPowerRes, "PowerRes", pOpFlagNamed | pOpFlagScoped, makeArg5(pArgTypePkgLen, pArgTypeNameString, pArgTypeByteData, pArgTypeWordData, pArgTypeTermList)},
	/*0x6d*/ {pOpThermalZone, "ThermalZone", pOpFlagNamed | pOpFlagScoped, makeArg3(pArgTypePkgLen, pArgTypeNameString, pArgTypeTermList)},
	/*0x6e*/ {pOpIndexField, "IndexField", pOpFlagCreate | pOpFlagNamed, makeArg5(pArgTypePkgLen, pArgTypeNameString, pArgTypeNameString, pArgTypeByteData, pArgTypeFieldList)},
	/*0x6f*/ {pOpBankField, "BankField", pOpFlagDeferParsing | pOpFlagCreate | pOpFlagNamed, makeArg6(pArgTypePkgLen, pArgTypeNameString, pArgTypeNameString, pArgTypeTermArg, pArgTypeByteData, pArgTypeFieldList)},
	/*0x70*/ {pOpDataRegion, "DataRegion", pOpFlagCreate | pOpFlagNamed, makeArg4(pArgTypeNameString, pArgTypeTermArg, pArgTypeTermArg, pArgTypeTermArg)},
	// Special internal opcodes
	/*0xf7*/ {pOpIntScopeBlock, "ScopeBlock", pOpFlagCreate | pOpFlagNamed, makeArg1(pArgTypeTermList)},
	/*0xf8*/ {pOpIntByteList, "ByteList", pOpFlagCreate, makeArg0()},
	/*0xf9*/ {pOpIntConnection, "Connection", pOpFlagCreate, makeArg0()},
	/*0xfa*/ {pOpIntNamedField, "NamedField", pOpFlagCreate, makeArg0()},
	/*0xfb*/ {pOpIntResolvedNamePath, "ResolvedNamePath", pOpFlagCreate, makeArg0()},
	/*0xfc*/ {pOpIntNamePath, "NamePath", pOpFlagCreate, makeArg0()},
	/*0xfd*/ {pOpIntNamePathOrMethodCall, "NamePath or MethodCall", pOpFlagCreate, makeArg0()},
	/*0xfe*/ {pOpIntMethodCall, "MethodCall", pOpFlagCreate, makeArg0()},
}

// opcodeMap maps an AML opcode to an entry in the opcode table. Entries with
// the value 0xff indicate an invalid/unsupported opcode.
var opcodeMap = [256]uint8{
	/*              0     1     2     3     4     5     6     7*/
	/*0x00 - 0x07*/ 0x00, 0x01, 0xff, 0xff, 0xff, 0xff, 0x02, 0xff,
	/*0x08 - 0x0f*/ 0x03, 0xff, 0x04, 0x05, 0x06, 0x07, 0x08, 0xff,
	/*0x10 - 0x17*/ 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0xff, 0xff,
	/*0x18 - 0x1f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x20 - 0x27*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x28 - 0x2f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x30 - 0x37*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x38 - 0x3f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x40 - 0x47*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x48 - 0x4f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x50 - 0x57*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x58 - 0x5f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x60 - 0x67*/ 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16,
	/*0x68 - 0x6f*/ 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0xff,
	/*0x70 - 0x77*/ 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25,
	/*0x78 - 0x7f*/ 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d,
	/*0x80 - 0x87*/ 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35,
	/*0x88 - 0x8f*/ 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d,
	/*0x90 - 0x97*/ 0x3e, 0x3f, 0x40, 0x41, 0x42, 0x43, 0x44, 0x45,
	/*0x98 - 0x9f*/ 0x46, 0x47, 0xff, 0xff, 0x48, 0x49, 0x4a, 0x4b,
	/*0xa0 - 0xa7*/ 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0x51, 0xff, 0xff,
	/*0xa8 - 0xaf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xb0 - 0xb7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xb8 - 0xbf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xc0 - 0xc7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xc8 - 0xcf*/ 0xff, 0xff, 0xff, 0xff, 0x52, 0xff, 0xff, 0xff,
	/*0xd0 - 0xd7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xd8 - 0xdf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xe0 - 0xe7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xe8 - 0xef*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xf0 - 0xf7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xf8 - 0xff*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x53,
}

// extendedOpcodeMap maps an AML extended opcode (extOpPrefix + code) to an
// entry in the opcode table. Entries with the value 0xff indicate an
// invalid/unsupported opcode.
var extendedOpcodeMap = [256]uint8{
	/*              0     1     2     3     4     5     6     7*/
	/*0x00 - 0x07*/ 0xff, 0x54, 0x55, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x08 - 0x0f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x10 - 0x17*/ 0xff, 0xff, 0x56, 0x57, 0xff, 0xff, 0xff, 0xff,
	/*0x18 - 0x1f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x58,
	/*0x20 - 0x27*/ 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
	/*0x28 - 0x2f*/ 0x61, 0x62, 0x63, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x30 - 0x37*/ 0x64, 0x65, 0x66, 0x67, 0xff, 0xff, 0xff, 0xff,
	/*0x38 - 0x3f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x40 - 0x47*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x48 - 0x4f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x50 - 0x57*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x58 - 0x5f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x60 - 0x67*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x68 - 0x6f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x70 - 0x77*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x78 - 0x7f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x80 - 0x87*/ 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f,
	/*0x88 - 0x8f*/ 0x70, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x90 - 0x97*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x98 - 0x9f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xa0 - 0xa7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xa8 - 0xaf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xb0 - 0xb7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xb8 - 0xbf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xc0 - 0xc7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xc8 - 0xcf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xd0 - 0xd7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xd8 - 0xdf*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xe0 - 0xe7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xe8 - 0xef*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xf0 - 0xf7*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0xf8 - 0xff*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// pOpcodeName returns the name of an opcode as a string.
func pOpcodeName(opcode uint16) string {
	index := pOpcodeTableIndex(opcode, true)
	if index == badOpcode {
		return "unknown"
	}

	return pOpcodeTable[index].opName
}

// pOpcodeTableIndex returns the opcode table index for opcode or badOpcode(0xff)
// if opcode does not map to a valid opcode table entry.
func pOpcodeTableIndex(opcode uint16, allowInternalOp bool) uint8 {
	if opcode <= 0xff {
		return opcodeMap[opcode]
	}

	index := extendedOpcodeMap[opcode-0xff]

	// internal opcodes do not have entries in the extendedOpcodeMap. They get
	// allocated descending opcode values starting from (0xff + 0xfe). They do
	// however have entries in the opcodeTable. To calculate their index we use
	// the following formula: len(opcodeTable) - 1 - (0x1fd - intOpcode) or the
	// equivalent: len(opcodeTable) + (intOpcode - 0x1fe)
	// with
	if index == badOpcode && allowInternalOp {
		index = uint8(len(pOpcodeTable) + int(opcode) - 0x1fe)
	}

	return index
}
