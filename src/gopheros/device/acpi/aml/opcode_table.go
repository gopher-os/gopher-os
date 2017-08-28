package aml

const (
	badOpcode   = 0xff
	extOpPrefix = 0x5b

	// Regular opcode list
	opZero             = opcode(0x00)
	opOne              = opcode(0x01)
	opAlias            = opcode(0x06)
	opName             = opcode(0x08)
	opBytePrefix       = opcode(0x0a)
	opWordPrefix       = opcode(0x0b)
	opDwordPrefix      = opcode(0x0c)
	opStringPrefix     = opcode(0x0d)
	opQwordPrefix      = opcode(0x0e)
	opScope            = opcode(0x10)
	opBuffer           = opcode(0x11)
	opPackage          = opcode(0x12)
	opVarPackage       = opcode(0x13)
	opMethod           = opcode(0x14)
	opExternal         = opcode(0x15)
	opLocal0           = opcode(0x60)
	opLocal1           = opcode(0x61)
	opLocal2           = opcode(0x62)
	opLocal3           = opcode(0x63)
	opLocal4           = opcode(0x64)
	opLocal5           = opcode(0x65)
	opLocal6           = opcode(0x66)
	opLocal7           = opcode(0x67)
	opArg0             = opcode(0x68)
	opArg1             = opcode(0x69)
	opArg2             = opcode(0x6a)
	opArg3             = opcode(0x6b)
	opArg4             = opcode(0x6c)
	opArg5             = opcode(0x6d)
	opArg6             = opcode(0x6e)
	opStore            = opcode(0x70)
	opRefOf            = opcode(0x71)
	opAdd              = opcode(0x72)
	opConcat           = opcode(0x73)
	opSubtract         = opcode(0x74)
	opIncrement        = opcode(0x75)
	opDecrement        = opcode(0x76)
	opMultiply         = opcode(0x77)
	opDivide           = opcode(0x78)
	opShiftLeft        = opcode(0x79)
	opShiftRight       = opcode(0x7a)
	opAnd              = opcode(0x7b)
	opNand             = opcode(0x7c)
	opOr               = opcode(0x7d)
	opNor              = opcode(0x7e)
	opXor              = opcode(0x7f)
	opNot              = opcode(0x80)
	opFindSetLeftBit   = opcode(0x81)
	opFindSetRightBit  = opcode(0x82)
	opDerefOf          = opcode(0x83)
	opConcatRes        = opcode(0x84)
	opMod              = opcode(0x85)
	opNotify           = opcode(0x86)
	opSizeOf           = opcode(0x87)
	opIndex            = opcode(0x88)
	opMatch            = opcode(0x89)
	opCreateDWordField = opcode(0x8a)
	opCreateWordField  = opcode(0x8b)
	opCreateByteField  = opcode(0x8c)
	opCreateBitField   = opcode(0x8d)
	opObjectType       = opcode(0x8e)
	opCreateQWordField = opcode(0x8f)
	opLand             = opcode(0x90)
	opLor              = opcode(0x91)
	opLnot             = opcode(0x92)
	opLEqual           = opcode(0x93)
	opLGreater         = opcode(0x94)
	opLLess            = opcode(0x95)
	opToBuffer         = opcode(0x96)
	opToDecimalString  = opcode(0x97)
	opToHexString      = opcode(0x98)
	opToInteger        = opcode(0x99)
	opToString         = opcode(0x9c)
	opCopyObject       = opcode(0x9d)
	opMid              = opcode(0x9e)
	opContinue         = opcode(0x9f)
	opIf               = opcode(0xa0)
	opElse             = opcode(0xa1)
	opWhile            = opcode(0xa2)
	opNoop             = opcode(0xa3)
	opReturn           = opcode(0xa4)
	opBreak            = opcode(0xa5)
	opBreakPoint       = opcode(0xcc)
	opOnes             = opcode(0xff)
	// Extended opcodes
	opMutex       = opcode(0xff + 0x01)
	opEvent       = opcode(0xff + 0x02)
	opCondRefOf   = opcode(0xff + 0x12)
	opCreateField = opcode(0xff + 0x13)
	opLoadTable   = opcode(0xff + 0x1f)
	opLoad        = opcode(0xff + 0x20)
	opStall       = opcode(0xff + 0x21)
	opSleep       = opcode(0xff + 0x22)
	opAcquire     = opcode(0xff + 0x23)
	opSignal      = opcode(0xff + 0x24)
	opWait        = opcode(0xff + 0x25)
	opReset       = opcode(0xff + 0x26)
	opRelease     = opcode(0xff + 0x27)
	opFromBCD     = opcode(0xff + 0x28)
	opToBCD       = opcode(0xff + 0x29)
	opUnload      = opcode(0xff + 0x2a)
	opRevision    = opcode(0xff + 0x30)
	opDebug       = opcode(0xff + 0x31)
	opFatal       = opcode(0xff + 0x32)
	opTimer       = opcode(0xff + 0x33)
	opOpRegion    = opcode(0xff + 0x80)
	opField       = opcode(0xff + 0x81)
	opDevice      = opcode(0xff + 0x82)
	opProcessor   = opcode(0xff + 0x83)
	opPowerRes    = opcode(0xff + 0x84)
	opThermalZone = opcode(0xff + 0x85)
	opIndexField  = opcode(0xff + 0x86)
	opBankField   = opcode(0xff + 0x87)
	opDataRegion  = opcode(0xff + 0x88)
)

// The opcode table contains all opcode-related information that the parser knows.
// This table is modeled after a similar table used in the acpica implementation.
var opcodeTable = []opcodeInfo{
	/*0x00*/ {opZero, "Zero", objTypeInteger, opFlagConstant, makeArg0()},
	/*0x01*/ {opOne, "One", objTypeInteger, opFlagConstant, makeArg0()},
	/*0x02*/ {opAlias, "Alias", objTypeLocalAlias, opFlagNamed, makeArg2(opArgNameString, opArgNameString)},
	/*0x03*/ {opName, "Name", objTypeAny, opFlagNamed, makeArg2(opArgNameString, opArgDataRefObj)},
	/*0x04*/ {opBytePrefix, "Byte", objTypeInteger, opFlagConstant, makeArg1(opArgByteData)},
	/*0x05*/ {opWordPrefix, "Word", objTypeInteger, opFlagConstant, makeArg1(opArgWord)},
	/*0x06*/ {opDwordPrefix, "Dword", objTypeInteger, opFlagConstant, makeArg1(opArgDword)},
	/*0x07*/ {opStringPrefix, "String", objTypeString, opFlagConstant, makeArg1(opArgString)},
	/*0x08*/ {opQwordPrefix, "Qword", objTypeInteger, opFlagConstant, makeArg1(opArgQword)},
	/*0x09*/ {opScope, "Scope", objTypeLocalScope, opFlagNamed, makeArg2(opArgNameString, opArgTermList)},
	/*0x0a*/ {opBuffer, "Buffer", objTypeBuffer, opFlagHasPkgLen, makeArg2(opArgTermObj, opArgByteList)},
	/*0x0b*/ {opPackage, "Package", objTypePackage, opFlagNone, makeArg2(opArgByteData, opArgTermList)},
	/*0x0c*/ {opVarPackage, "VarPackage", objTypePackage, opFlagNone, makeArg2(opArgByteData, opArgTermList)},
	/*0x0d*/ {opMethod, "Method", objTypeMethod, opFlagNamed | opFlagScoped, makeArg3(opArgNameString, opArgByteData, opArgTermList)},
	/*0x0e*/ {opExternal, "External", objTypeAny, opFlagNamed, makeArg3(opArgNameString, opArgByteData, opArgByteData)},
	/*0x0f*/ {opLocal0, "Local0", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x10*/ {opLocal1, "Local1", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x11*/ {opLocal2, "Local2", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x12*/ {opLocal3, "Local3", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x13*/ {opLocal4, "Local4", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0120*/ {opLocal5, "Local5", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x15*/ {opLocal6, "Local6", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x16*/ {opLocal7, "Local7", objTypeLocalVariable, opFlagExecutable, makeArg0()},
	/*0x17*/ {opArg0, "Arg0", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x18*/ {opArg1, "Arg1", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x19*/ {opArg2, "Arg2", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x1a*/ {opArg3, "Arg3", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x1b*/ {opArg4, "Arg4", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x1c*/ {opArg5, "Arg5", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x1d*/ {opArg6, "Arg6", objTypeMethodArgument, opFlagExecutable, makeArg0()},
	/*0x1e*/ {opStore, "Store", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgSuperName)},
	/*0x1f*/ {opRefOf, "RefOf", objTypeAny, opFlagReference | opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x20*/ {opAdd, "Add", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x21*/ {opConcat, "Concat", objTypeAny, opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x22*/ {opSubtract, "Subtract", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x23*/ {opIncrement, "Increment", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x24*/ {opDecrement, "Decrement", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x25*/ {opMultiply, "Multiply", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x26*/ {opDivide, "Divide", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg4(opArgTermObj, opArgTermObj, opArgTarget, opArgTarget)},
	/*0x27*/ {opShiftLeft, "ShiftLeft", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x28*/ {opShiftRight, "ShiftRight", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x29*/ {opAnd, "And", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x2a*/ {opNand, "Nand", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x2b*/ {opOr, "Or", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x2c*/ {opNor, "Nor", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x2d*/ {opXor, "Xor", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x2e*/ {opNot, "Not", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x2f*/ {opFindSetLeftBit, "FindSetLeftBit", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x30*/ {opFindSetRightBit, "FindSetRightBit", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x31*/ {opDerefOf, "DerefOf", objTypeAny, opFlagExecutable, makeArg1(opArgTermObj)},
	/*0x32*/ {opConcatRes, "ConcatRes", objTypeAny, opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x33*/ {opMod, "Mod", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x34*/ {opNotify, "Notify", objTypeAny, opFlagExecutable, makeArg2(opArgSuperName, opArgTermObj)},
	/*0x35*/ {opSizeOf, "SizeOf", objTypeAny, opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x36*/ {opIndex, "Index", objTypeAny, opFlagExecutable, makeArg3(opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x37*/ {opMatch, "Match", objTypeAny, opFlagExecutable, makeArg6(opArgTermObj, opArgByteData, opArgTermObj, opArgByteData, opArgTermObj, opArgTermObj)},
	/*0x38*/ {opCreateDWordField, "CreateDWordField", objTypeBufferField, opFlagNamed | opFlagCreate, makeArg3(opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x39*/ {opCreateWordField, "CreateWordField", objTypeBufferField, opFlagNamed | opFlagCreate, makeArg3(opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x3a*/ {opCreateByteField, "CreateByteField", objTypeBufferField, opFlagNamed | opFlagCreate, makeArg3(opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x3b*/ {opCreateBitField, "CreateBitField", objTypeBufferField, opFlagNamed | opFlagCreate, makeArg3(opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x3c*/ {opObjectType, "ObjectType", objTypeAny, opFlagNone, makeArg1(opArgSuperName)},
	/*0x3d*/ {opCreateQWordField, "CreateQWordField", objTypeBufferField, opFlagNamed | opFlagCreate, makeArg3(opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x3e*/ {opLand, "Land", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTermObj)},
	/*0x3f*/ {opLor, "Lor", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTermObj)},
	/*0x40*/ {opLnot, "Lnot", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg1(opArgTermObj)},
	/*0x41*/ {opLEqual, "LEqual", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTermObj)},
	/*0x42*/ {opLGreater, "LGreater", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTermObj)},
	/*0x43*/ {opLLess, "LLess", objTypeAny, opFlagArithmetic | opFlagExecutable, makeArg2(opArgTermObj, opArgTermObj)},
	/*0x44*/ {opToBuffer, "ToBuffer", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x45*/ {opToDecimalString, "ToDecimalString", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x46*/ {opToHexString, "ToHexString", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x47*/ {opToInteger, "ToInteger", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x48*/ {opToString, "ToString", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x49*/ {opCopyObject, "CopyObject", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgSimpleName)},
	/*0x4a*/ {opMid, "Mid", objTypeAny, opFlagExecutable, makeArg4(opArgTermObj, opArgTermObj, opArgTermObj, opArgTarget)},
	/*0x4b*/ {opContinue, "Continue", objTypeAny, opFlagExecutable, makeArg0()},
	/*0x4c*/ {opIf, "If", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTermList)},
	/*0x4d*/ {opElse, "Else", objTypeAny, opFlagExecutable | opFlagScoped, makeArg1(opArgTermList)},
	/*0x4e*/ {opWhile, "While", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTermList)},
	/*0x4f*/ {opNoop, "Noop", objTypeAny, opFlagNoOp, makeArg0()},
	/*0x50*/ {opReturn, "Return", objTypeAny, opFlagReturn, makeArg1(opArgTermObj)},
	/*0x51*/ {opBreak, "Break", objTypeAny, opFlagExecutable, makeArg0()},
	/*0x52*/ {opBreakPoint, "BreakPoint", objTypeAny, opFlagNoOp, makeArg0()},
	/*0x53*/ {opOnes, "Ones", objTypeInteger, opFlagConstant, makeArg0()},
	/*0x54*/ {opMutex, "Mutex", objTypeMutex, opFlagNamed, makeArg2(opArgNameString, opArgByteData)},
	/*0x55*/ {opEvent, "Event", objTypeEvent, opFlagNamed, makeArg1(opArgNameString)},
	/*0x56*/ {opCondRefOf, "CondRefOf", objTypeAny, opFlagExecutable, makeArg2(opArgSuperName, opArgSuperName)},
	/*0x57*/ {opCreateField, "CreateField", objTypeBufferField, opFlagExecutable, makeArg4(opArgTermObj, opArgTermObj, opArgTermObj, opArgNameString)},
	/*0x58*/ {opLoadTable, "LoadTable", objTypeAny, opFlagExecutable, makeArg7(opArgTermObj, opArgTermObj, opArgTermObj, opArgTermObj, opArgTermObj, opArgTermObj, opArgTermObj)},
	/*0x59*/ {opLoad, "Load", objTypeAny, opFlagExecutable, makeArg2(opArgNameString, opArgSuperName)},
	/*0x5a*/ {opStall, "Stall", objTypeAny, opFlagExecutable, makeArg1(opArgTermObj)},
	/*0x5b*/ {opSleep, "Sleep", objTypeAny, opFlagExecutable, makeArg1(opArgTermObj)},
	/*0x5c*/ {opAcquire, "Acquire", objTypeAny, opFlagExecutable, makeArg2(opArgNameString, opArgSuperName)},
	/*0x5d*/ {opSignal, "Signal", objTypeAny, opFlagExecutable, makeArg1(opArgTermObj)},
	/*0x5e*/ {opWait, "Wait", objTypeAny, opFlagExecutable, makeArg2(opArgSuperName, opArgTermObj)},
	/*0x5f*/ {opReset, "Reset", objTypeAny, opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x60*/ {opRelease, "Release", objTypeAny, opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x61*/ {opFromBCD, "FromBCD", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x62*/ {opToBCD, "ToBCD", objTypeAny, opFlagExecutable, makeArg2(opArgTermObj, opArgTarget)},
	/*0x63*/ {opUnload, "Unload", objTypeAny, opFlagExecutable, makeArg1(opArgSuperName)},
	/*0x64*/ {opRevision, "Revision", objTypeInteger, opFlagConstant | opFlagExecutable, makeArg0()},
	/*0x65*/ {opDebug, "Debug", objTypeLocalReference, opFlagExecutable, makeArg0()},
	/*0x66*/ {opFatal, "Fatal", objTypeAny, opFlagExecutable, makeArg3(opArgByteData, opArgDword, opArgTermObj)},
	/*0x67*/ {opTimer, "Timer", objTypeAny, opFlagNone, makeArg0()},
	/*0x68*/ {opOpRegion, "OpRegion", objTypeRegion, opFlagNamed, makeArg4(opArgNameString, opArgByteData, opArgTermObj, opArgTermObj)},
	/*0x69*/ {opField, "Field", objTypeAny, opFlagNone, makeArg3(opArgNameString, opArgByteData, opArgFieldList)},
	/*0x6a*/ {opDevice, "Device", objTypeDevice, opFlagNamed | opFlagScoped, makeArg2(opArgNameString, opArgTermList)},
	/*0x6b*/ {opProcessor, "Processor", objTypeProcessor, opFlagNamed | opFlagScoped, makeArg5(opArgNameString, opArgByteData, opArgDword, opArgByteData, opArgTermList)},
	/*0x6c*/ {opPowerRes, "PowerRes", objTypePower, opFlagNamed | opFlagScoped, makeArg4(opArgNameString, opArgByteData, opArgWord, opArgTermList)},
	/*0x6d*/ {opThermalZone, "ThermalZone", objTypeThermal, opFlagNamed | opFlagScoped, makeArg2(opArgNameString, opArgTermList)},
	/*0x6e*/ {opIndexField, "IndexField", objTypeAny, opFlagNone, makeArg4(opArgNameString, opArgNameString, opArgByteData, opArgFieldList)},
	/*0x6f*/ {opBankField, "BankField", objTypeLocalBankField, opFlagNamed, makeArg5(opArgNameString, opArgNameString, opArgTermObj, opArgByteData, opArgFieldList)},
	/*0x70*/ {opDataRegion, "DataRegion", objTypeLocalRegionField, opFlagNamed, makeArg4(opArgNameString, opArgTermObj, opArgTermObj, opArgTermObj)},
}

// opcodeMap maps an AML opcode to an entry in the opcode table. Entries with
// the value 0xff indicate an invalid/unsupported opcode.
var opcodeMap = [256]uint8{
	/*              0     1     2     3     4     5     6     7*/
	/*0x00 - 0x07*/ 0x00, 0x01, 0xff, 0xff, 0xff, 0xff, 0x02, 0xff,
	/*0x08 - 0x0f*/ 0x03, 0xff, 0x04, 0x05, 0x06, 0x07, 0x08, 0xff,
	/*0x10 - 0x17*/ 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0xff, 0xff, 0xff,
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
	/*0x88 - 0x8f*/ 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0xff, 0x3d,
	/*0x90 - 0x97*/ 0x3e, 0x3f, 0x40, 0x41, 0x42, 0x43, 0x44, 0x45,
	/*0x98 - 0x9f*/ 0x46, 0x47, 0x48, 0x49, 0x4a, 0xff, 0x4a, 0x4b,
	/*0xa0 - 0xa7*/ 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0x51, 0xff, 0xff,
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
	/*0xf8 - 0xff*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x53,
}

// extendedOpcodeMap maps an AML extended opcode (extOpPrefix + code) to an
// entry in the opcode table. Entries with the value 0xff indicate an
// invalid/unsupported opcode.
var extendedOpcodeMap = [256]uint8{
	/*              0     1     2     3     4     5     6     7*/
	/*0x00 - 0x07*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x08 - 0x0f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x10 - 0x17*/ 0xff, 0xff, 0x56, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x18 - 0x1f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x20 - 0x27*/ 0xff, 0x5a, 0x5b, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x28 - 0x2f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x30 - 0x37*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x38 - 0x3f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x40 - 0x47*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x48 - 0x4f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x50 - 0x57*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x58 - 0x5f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x60 - 0x67*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x68 - 0x6f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x70 - 0x77*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x78 - 0x7f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	/*0x80 - 0x87*/ 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0xff,
	/*0x88 - 0x8f*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
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
	/*0xf8 - 0xff*/ 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x53,
}
