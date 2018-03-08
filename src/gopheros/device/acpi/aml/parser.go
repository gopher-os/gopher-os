package aml

import (
	"gopheros/device/acpi/table"
	"gopheros/kernel"
	"gopheros/kernel/kfmt"
	"io"
	"reflect"
	"unsafe"
)

var (
	errParsingAML = &kernel.Error{Module: "acpi_aml_parser", Message: "could not parse AML bytecode"}
)

type parseResult uint8
type parseMode uint8

const (
	parseResultFailed parseResult = iota
	parseResultOk
	parseResultShortCircuit
	parseResultRequireExtraPass

	parseModeSkipAmbiguousBlocks parseMode = iota
	parseModeAllBlocks

	// The maximum number of mergeScopeDirectives - relocateNamedObjects passes
	// attempted by the parser to fully resolve static objects.
	maxResolvePasses = 5
)

// Parser implements a parser for ACPI Machine Language (AML) bytecode.
type Parser struct {
	r         amlStreamReader
	errWriter io.Writer

	tableName   string
	tableHandle uint8

	objTree     *ObjectTree
	scopeStack  []uint32
	pkgEndStack []uint32
	streamEnd   uint32

	resolvePasses    uint32
	mergedScopes     uint32
	relocatedObjects uint32

	mode parseMode
}

// NewParser creates a new AML parser instance that attaches parsed AML entities to
// the provided objTree and emits parse errors to errWriter.
func NewParser(errWriter io.Writer, objTree *ObjectTree) *Parser {
	return &Parser{
		errWriter: errWriter,
		objTree:   objTree,
	}
}

// ParseAML attempts to parse the AML byte-code contained in the supplied ACPI
// table tagging each scoped entity with the supplied table handle.
func (p *Parser) ParseAML(tableHandle uint8, tableName string, header *table.SDTHeader) *kernel.Error {
	p.init(tableHandle, tableName, header)

	// Parse raw object list starting at the root scope
	p.scopeEnter(0)
	if p.parseObjectList() == parseResultFailed {
		return errParsingAML
	}

	// Connect missing args to named objects
	if p.connectNamedObjArgs(0) != parseResultOk {
		return errParsingAML
	}

	// Resolve scope directives for non-executable blocks and relocate named
	// objects. These two tasks run in lockstep until both steps report success
	// or either reports an error. Running multiple passes is required to deal
	// with edge cases like this one:
	//  Scope (_SB)
	//  {
	//    Device(SBRG)
	//    {
	//      Name(_HID, 0xf00)
	//    }
	//  }
	//  Scope (_SB )
	//  {
	//    Scope(SBRG)
	//    {
	//      Device(^PCIE)
	//      {
	//        Name(_HID, 0xba1)
	//      }
	//    }
	//  }
	//  Scope (_SB)
	//  {
	//    Scope(PCIE)
	//    {
	//      Name(CRS, Zero)
	//    }
	//  }
	//
	// In this example, calling mergeScopeDirectives will place the PCIE device
	// into the SBRG scope. However, the PCIE device must actually be placed in
	// the _SB scope due to the ^ prefix in its name. The 3rd block expects to
	// find PCIE in the _SB scope and will fail with parseResultRequireExtraPass.
	//
	// Running relocateNamedObjects immediately after will place the PCIE device
	// at the correct location allowing us to run an extra mergeScopeDirectives
	// pass to fully resolve the scope directive.
	p.resolvePasses = 1
	for ; ; p.resolvePasses++ {
		mergeRes := p.mergeScopeDirectives(0)
		if mergeRes == parseResultFailed {
			return errParsingAML
		}

		relocateRes := p.relocateNamedObjects(0)
		if relocateRes == parseResultFailed {
			return errParsingAML
		}

		// Stop if both calls returned OK
		if mergeRes == parseResultOk && relocateRes == parseResultOk {
			break
		}
	}

	// Parse deferred blocks
	if p.parseDeferredBlocks(0) != parseResultOk {
		return errParsingAML
	}

	// Resolve method calls
	if p.resolveMethodCalls(0) != parseResultOk {
		return errParsingAML
	}

	// Connect missing args that include method invocations to remaining
	// non-named objects
	if p.connectNonNamedObjArgs(0) != parseResultOk {
		return errParsingAML
	}

	return nil
}

func (p *Parser) init(tableHandle uint8, tableName string, header *table.SDTHeader) {
	p.resetState(tableHandle, tableName)

	p.r.Init(
		uintptr(unsafe.Pointer(header)),
		header.Length,
		uint32(unsafe.Sizeof(table.SDTHeader{})),
	)

	// Keep track of the stream end for parsing deferred objects
	p.streamEnd = header.Length
	_ = p.pushPkgEnd(header.Length)
}

func (p *Parser) resetState(tableHandle uint8, tableName string) {
	p.tableHandle = tableHandle
	p.tableName = tableName
	p.resolvePasses = 0
	p.mergedScopes = 0
	p.relocatedObjects = 0
	p.mode = parseModeSkipAmbiguousBlocks

	p.scopeStack = nil
	p.pkgEndStack = nil

}

// parseObjectList tries to parse an AML object list. Object lists are usually
// specified together with a pkgLen block which is used to calculate the max
// read offset that the parser may reach.
func (p *Parser) parseObjectList() parseResult {
	for len(p.scopeStack) != 0 {
		// Consume up to the current package end
		for !p.r.EOF() {
			if p.parseNextObject() != parseResultOk {
				return parseResultFailed
			}
		}

		// If the pkgEnd stack matches the scope stack this means the parser
		// completed a scoped block and we need to pop it from the stack
		if len(p.pkgEndStack) == len(p.scopeStack) {
			p.scopeExit()
		}
		p.popPkgEnd()
	}

	return parseResultOk
}

// parseNextObject tries to parse a single object from the AML stream and
// attach it to the currently active scope.
func (p *Parser) parseNextObject() parseResult {
	curOffset := p.r.Offset()
	nextOp, res := p.nextOpcode()
	if nextOp == pOpNoop {
		return parseResultOk
	}

	if res == parseResultFailed {
		return p.parseNamePathOrMethodCall()
	}

	curObj := p.objTree.newObject(nextOp, p.tableHandle)
	curObj.amlOffset = curOffset
	p.objTree.append(p.scopeCurrent(), curObj)
	return p.parseObjectArgs(curObj)
}

func (p *Parser) parseObjectArgs(curObj *Object) parseResult {
	var res parseResult

	// Special case for constants that are used as TermArgs; just read the
	// value directly into the supplied curObject
	switch curObj.opcode {
	case pOpBytePrefix:
		curObj.value, res = p.parseNumConstant(1)
	case pOpWordPrefix:
		curObj.value, res = p.parseNumConstant(2)
	case pOpDwordPrefix:
		curObj.value, res = p.parseNumConstant(4)
	case pOpQwordPrefix:
		curObj.value, res = p.parseNumConstant(8)
	case pOpStringPrefix:
		curObj.value, res = p.parseString()
	default:
		res = p.parseArgs(&pOpcodeTable[curObj.infoIndex], curObj, 0)
	}

	if res == parseResultShortCircuit {
		res = parseResultOk
	}

	return res
}

func (p *Parser) parseArgs(info *pOpcodeInfo, curObj *Object, argOffset uint8) parseResult {
	var (
		argCount = info.argFlags.argCount()
		argObj   *Object
		res      = parseResultOk
	)

	if argCount == 0 {
		return parseResultOk
	}

	for argIndex := argOffset; argIndex < argCount && res == parseResultOk; argIndex++ {
		argObj, res = p.parseArg(info, curObj, info.argFlags.arg(argIndex))

		// Ignore nil args (e.g. result of parsing a pkgLen)
		if argObj != nil {
			p.objTree.append(curObj, argObj)
		}
	}

	return res
}

func (p *Parser) parseArg(info *pOpcodeInfo, curObj *Object, argType pArgType) (*Object, parseResult) {
	switch argType {
	case pArgTypeByteData, pArgTypeWordData, pArgTypeDwordData, pArgTypeQwordData, pArgTypeString, pArgTypeNameString:
		return p.parseSimpleArg(argType)
	case pArgTypeByteList:
		argObj := p.objTree.newObject(pOpIntByteList, p.tableHandle)
		p.parseByteList(argObj, p.r.pkgEnd-p.r.Offset())
		return argObj, parseResultOk
	case pArgTypePkgLen:
		origOffset := p.r.Offset()
		pkgLen, res := p.parsePkgLength()
		if res != parseResultOk {
			return nil, res
		}

		// If this opcode requires deferred parsing just keep track of
		// the package end and skip over it.
		if p.mode == parseModeSkipAmbiguousBlocks && (info.flags&pOpFlagDeferParsing != 0) {
			curObj.pkgEnd = origOffset + pkgLen
			p.r.SetOffset(curObj.pkgEnd)
			return nil, parseResultShortCircuit
		}

		if err := p.pushPkgEnd(origOffset + pkgLen); err != nil {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] %s\n", p.tableName, p.r.Offset(), err.Error())
			return nil, parseResultFailed
		}

		return nil, parseResultOk
	case pArgTypeFieldList:
		return nil, p.parseFieldElements(curObj)
	case pArgTypeTermArg, pArgTypeDataRefObj:
		// A TermObj may contain a method call to a not-yet defined
		// method.  Since the AML stream does not contain info about
		// the number of arguments in method calls the parser cannot
		// distinguish where the TermObj actually ends. To work around
		// this limitation, the parser will process any object up to
		// the current package end and append them as *siblings* to
		// curObj. We will then need to perform a second pass to
		// properly populate the arguments.
		if p.mode == parseModeAllBlocks {
			return p.parseStrictTermArg(curObj)
		}

		return nil, parseResultShortCircuit
	case pArgTypeTermList:
		// Create a new scope and shortcircuit the arg parser
		scope := p.objTree.newObject(pOpIntScopeBlock, p.tableHandle)
		scope.amlOffset = p.r.Offset()
		p.scopeEnter(scope.index)

		if p.mode == parseModeSkipAmbiguousBlocks {
			return scope, parseResultShortCircuit
		}

		// Attach scope to curObj so lookups work and consume all objects till the package end
		p.objTree.append(curObj, scope)
		for !p.r.EOF() {
			if p.parseNextObject() != parseResultOk {
				return nil, parseResultFailed
			}
		}
		p.scopeExit()
		p.objTree.detach(curObj, scope)
		return scope, parseResultOk
	default: // pArgTypeTarget, pArgTypeSimpleName, pArgTypeSuperName:
		return p.parseTarget()
	}
}

// parseNamePathOrMethodCall is invoked by the parser when it cannot parse an
// opcode from the AML stream. This indicates that the stream may contain
// either a method call or a named path reference. If the current parse mode is
// parseModeAllBlocks the method will return an error if the next term in the
// stream does not match an existing namepath or method name.
func (p *Parser) parseNamePathOrMethodCall() parseResult {
	curOffset := p.r.Offset()

	pathExpr, res := p.parseNameString()
	if res != parseResultOk {
		return parseResultFailed
	}

	// During the initial pass we cannot be sure about whether this is actually
	// a method name or a named path reference as AML allows for forward decls.
	if p.mode == parseModeSkipAmbiguousBlocks {
		curObj := p.objTree.newObject(pOpIntNamePathOrMethodCall, p.tableHandle)
		curObj.amlOffset = curOffset
		curObj.value = pathExpr
		p.objTree.append(p.scopeCurrent(), curObj)
		return parseResultOk
	}

	// Lookup the object pointed to by the path expression
	targetIndex := p.objTree.Find(
		p.objTree.ClosestNamedAncestor(p.scopeCurrent()),
		pathExpr,
	)

	if targetIndex == InvalidIndex {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] unable to resolve path expression %s\n", p.tableName, p.r.Offset(), pathExpr)
		return parseResultFailed
	}

	target := p.objTree.ObjectAt(targetIndex)
	curObj := p.objTree.newObject(pOpIntResolvedNamePath, p.tableHandle)
	curObj.amlOffset = curOffset
	curObj.value = targetIndex
	p.objTree.append(p.scopeCurrent(), curObj)
	if target.opcode != pOpMethod {
		return parseResultOk
	}

	// If target is a method definition we need to make curObj the active scope
	// and parse the number of args specified by the definition.
	curObj.opcode = pOpIntMethodCall
	curObj.infoIndex = pOpcodeTableIndex(curObj.opcode, true)
	p.scopeEnter(curObj.index)

	argCount := uint8(p.objTree.ArgAt(target, 1).value.(uint64) & 0x7)
	for argIndex := uint8(0); argIndex < argCount; argIndex++ {
		if p.parseNextObject() != parseResultOk {
			p.scopeExit()
			return parseResultFailed
		}
	}
	p.scopeExit()
	return parseResultOk
}

// parseStrictTermObj attempts to parse a TermArg from the stream using the
// strict grammar definitions from p. 1022 of the spec. As TermArgs may also
// contain method invocations, parseStrictTermArg must only be called after
// all static scope directives have been fully evaluated.
//
// Grammar:
// TermArg := Type2Opcode | DataObject | ArgObj | LocalObj
func (p *Parser) parseStrictTermArg(curObj *Object) (*Object, parseResult) {
	var (
		termObj   *Object
		curOffset = p.r.Offset()
	)

	nextOp, res := p.peekNextOpcode()
	if res != parseResultOk {
		// make curObj the active scope so that lookups can work and parse a
		// method call.
		p.scopeEnter(curObj.index)
		res = p.parseNamePathOrMethodCall()
		p.scopeExit()

		// parseNamePathOrMethodCall will append the parsed object to the scope
		// so we need to detach it.
		if res == parseResultOk {
			termObj = p.objTree.ObjectAt(curObj.lastArgIndex)
			p.objTree.detach(curObj, termObj)
		}

		if p.r.EOF() {
			p.popPkgEnd()
		}
		return termObj, res
	}

	if !pOpIsType2(nextOp) && !pOpIsDataObject(nextOp) && !pOpIsArg(nextOp) {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] encountered unexpected opcode %s while parsing termArg\n", p.tableName, p.r.Offset(), pOpcodeName(nextOp))
		return nil, parseResultFailed
	}

	_, _ = p.nextOpcode()
	termObj = p.objTree.newObject(nextOp, p.tableHandle)
	termObj.amlOffset = curOffset
	res = p.parseObjectArgs(termObj)
	if p.r.EOF() {
		p.popPkgEnd()
	}

	return termObj, res
}

func (p *Parser) parseSimpleArg(argType pArgType) (*Object, parseResult) {
	var (
		obj = p.objTree.newObject(0, p.tableHandle)
		res parseResult
	)

	obj.amlOffset = p.r.Offset()

	switch argType {
	case pArgTypeByteData:
		obj.opcode = pOpBytePrefix
		obj.value, res = p.parseNumConstant(1)
	case pArgTypeWordData:
		obj.opcode = pOpWordPrefix
		obj.value, res = p.parseNumConstant(2)
	case pArgTypeDwordData:
		obj.opcode = pOpDwordPrefix
		obj.value, res = p.parseNumConstant(4)
	case pArgTypeQwordData:
		obj.opcode = pOpQwordPrefix
		obj.value, res = p.parseNumConstant(8)
	case pArgTypeString:
		obj.opcode = pOpStringPrefix
		obj.value, res = p.parseString()
	case pArgTypeNameString:
		obj.opcode = pOpIntNamePath
		obj.value, res = p.parseNameString()
	default:
		return nil, parseResultFailed
	}

	obj.infoIndex = pOpcodeTableIndex(obj.opcode, true)
	return obj, res
}

// parseTarget attempts to pass a Target from the AML bytestream.
//
// Grammar:
// Target := SuperName | NullName
// NullName := 0x00
// SuperName := SimpleName | DebugObj | Type6Opcode
// Type6Opcode := DefRefOf | DefDerefOf | DefIndex | UserTermObj
// SimpleName := NameString | ArgObj | LocalObj
//
// UserTermObj is a control method invocation.
func (p *Parser) parseTarget() (*Object, parseResult) {
	// Peek next opcode
	origOffset := p.r.Offset()
	nextOp, res := p.nextOpcode()

	if res == parseResultOk {
		switch {
		case nextOp == pOpZero: // NullName == no target
			return nil, parseResultOk
		case pOpIsArg(nextOp) ||
			nextOp == pOpRefOf || nextOp == pOpDerefOf ||
			nextOp == pOpIndex || nextOp == pOpDebug:
			// This is a SuperName or a Type6Opcode
			obj := p.objTree.newObject(nextOp, p.tableHandle)
			obj.amlOffset = origOffset
			return obj, p.parseObjectArgs(obj)
		default:
			// Unexpected opcode
			return nil, parseResultFailed
		}
	}

	// In this case, this is either a method call or a named reference. Just
	// like in parseObjectList we assume it's a named reference.
	p.r.SetOffset(origOffset)
	curObj := p.objTree.newObject(pOpIntNamePath, p.tableHandle)
	curObj.amlOffset = origOffset
	curObj.value, res = p.parseNameString()
	return curObj, res
}

// parseFieldElements parses a list of named field elements from the AML stream.
//
// FieldList := <Nothing> | FieldElement FieldList
// FieldElement := NamedField | ReservedField | AccessField | ExtendedAccessField | ConnectField
// NamedField := NameSeg PkgLength
// ReservedField := 0x00 PkgLength
// AccessField := 0x1 AccessType AccessAttrib
// ConnectField := 0x02 NameString | 0x02 BufferData
// ExtendedAccessField := 0x3 AccessType ExtendedAccessAttrib AccessLength
func (p *Parser) parseFieldElements(curObj *Object) parseResult {
	var (
		nextFieldOffset uint32
		accessLength    uint8
		accessType      uint8
		accessAttrib    uint8
		lockType        uint8
		updateType      uint8

		field               *Object
		connection, connArg *Object
		appendAfter         = curObj
		connectionIndex     = InvalidIndex

		next     uint8
		nextOp   uint16
		err      error
		value    uint64
		pkgLen   uint32
		parseRes parseResult
	)

	// Initialize default flags based on the default flags of curObj
	initialFlags := uint8(p.objTree.ObjectAt(curObj.lastArgIndex).value.(uint64))
	accessType = initialFlags & 0xf        // bits [0:3]
	lockType = (initialFlags >> 4) & 0x1   // bits [4:4]
	updateType = (initialFlags >> 5) & 0x3 // bits [5:6]

	for !p.r.EOF() {
		next, _ = p.r.ReadByte()

		switch next {
		case 0x00: // ReservedField
			if pkgLen, parseRes = p.parsePkgLength(); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse pkgLen for ReservedField element\n", p.tableName, p.r.Offset())
				return parseResultFailed
			}

			nextFieldOffset += pkgLen
		case 0x01: // AccessField: change default access type for fields that follow
			if value, parseRes = p.parseNumConstant(1); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse AccessType for AccessField element\n", p.tableName, p.r.Offset())
				return parseRes
			}

			accessType = uint8(value & 0xff) // bits [0:7]

			if value, parseRes = p.parseNumConstant(1); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse AccessAttrib for AccessField element\n", p.tableName, p.r.Offset())
				return parseRes
			}

			accessAttrib = uint8(value & 0xff) // bits [0:7]
		case 0x03: // ExtAccessField
			if value, parseRes = p.parseNumConstant(1); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse AccessType for ExtAccessField element\n", p.tableName, p.r.Offset())
				return parseRes
			}

			accessType = uint8(value & 0xff) // bits [0:7]

			if value, parseRes = p.parseNumConstant(1); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse AccessAttrib for ExtAccessField element\n", p.tableName, p.r.Offset())
				return parseRes
			}

			accessAttrib = uint8(value & 0xff) // bits [0:7]

			if value, parseRes = p.parseNumConstant(1); parseRes == parseResultFailed {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse AccessLength for ExtAccessField element\n", p.tableName, p.r.Offset())
				return parseRes
			}

			accessLength = uint8(value & 0xff) // bits [0:7]
		case 0x02: // Connection
			// Connection can be either a namestring or a buffer
			next, err = p.r.ReadByte()
			if err != nil {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] unexpected end of stream while parsing Connection\n", p.tableName, p.r.Offset())
				return parseResultFailed
			}

			connection = p.objTree.newObject(pOpIntConnection, p.tableHandle)
			connectionIndex = connection.index
			p.objTree.append(curObj, connection)

			switch next {
			case uint8(pOpBuffer):
				origPkgEnd := p.r.pkgEnd
				origOffset := p.r.Offset()
				if pkgLen, parseRes = p.parsePkgLength(); parseRes != parseResultOk {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse pkgLen for Connection\n", p.tableName, p.r.Offset())
					return parseRes
				}

				dataLen := uint64(0)
				if pkgLen > 0 {
					if err = p.r.SetPkgEnd(origOffset + pkgLen); err != nil {
						kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] failed to set pkgEnd for Buffer\n", p.tableName, p.r.Offset())
						return parseResultFailed
					}

					if nextOp, parseRes = p.nextOpcode(); parseRes != parseResultOk {
						return parseRes
					}

					// Read data length
					parseRes = parseResultOk

					switch nextOp {
					case pOpBytePrefix:
						dataLen, parseRes = p.parseNumConstant(1)
					case pOpWordPrefix:
						dataLen, parseRes = p.parseNumConstant(2)
					case pOpDwordPrefix:
						dataLen, parseRes = p.parseNumConstant(4)
					}

					if parseRes == parseResultFailed {
						return parseRes
					}
				}

				connArg = p.objTree.newObject(pOpIntByteList, p.tableHandle)
				connArg.amlOffset = origOffset
				p.parseByteList(connArg, uint32(dataLen))

				// Restore previous pkg end and jump to end of buffer package
				_ = p.r.SetPkgEnd(origPkgEnd)
				p.r.SetOffset(origOffset + pkgLen)
			default:
				// unread first byte of namepath
				_ = p.r.UnreadByte()
				connArg = p.objTree.newObject(pOpIntNamePath, p.tableHandle)
				connArg.amlOffset = p.r.Offset()
				if connArg.value, parseRes = p.parseNameString(); parseRes != parseResultOk {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse namestring for Connection\n", p.tableName, p.r.Offset())
					return parseRes
				}
			}

			p.objTree.append(connection, connArg)
		default:
			// This is a named field and next is the first part of its name. We need
			// to rewind the stream
			_ = p.r.UnreadByte()

			field = p.objTree.newObject(pOpIntNamedField, p.tableHandle)
			field.amlOffset = p.r.Offset()
			for i := 0; i < amlNameLen; i++ {
				if field.name[i], err = p.r.ReadByte(); err != nil {
					return parseResultFailed
				}
			}

			if pkgLen, parseRes = p.parsePkgLength(); parseRes != parseResultOk {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] could not parse pkgLen for NamedField\n", p.tableName, p.r.Offset())
				return parseRes
			}

			// Populate field element and attach field object
			field.value = &fieldElement{
				offset:          nextFieldOffset,
				width:           pkgLen,
				accessLength:    accessLength,
				accessType:      accessType,
				accessAttrib:    accessAttrib,
				lockType:        lockType,
				updateType:      updateType,
				connectionIndex: connectionIndex,
				fieldIndex:      curObj.index,
			}

			// According to the spec, field elements appear at the same scope as
			// their Field/IndexField/BankField container
			p.objTree.appendAfter(p.objTree.ObjectAt(curObj.parentIndex), field, appendAfter)
			appendAfter = field

			// Calculate offset for next field
			nextFieldOffset += pkgLen
		}
	}

	// Skip to the end of the arg list for the current field object
	return parseResultShortCircuit
}

func (p *Parser) parseByteList(obj *Object, dataLen uint32) {
	obj.opcode = pOpIntByteList
	obj.infoIndex = pOpcodeTableIndex(obj.opcode, true)
	obj.value = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(dataLen),
		Cap:  int(dataLen),
		Data: p.r.DataPtr(),
	}))

	p.r.SetOffset(p.r.Offset() + dataLen)
}

// parsePkgLength parses a PkgLength value from the AML bytestream.
func (p *Parser) parsePkgLength() (uint32, parseResult) {
	origOffset := p.r.Offset()
	lead, err := p.r.ReadByte()
	if err != nil {
		p.r.SetOffset(origOffset)
		return 0, parseResultFailed
	}

	// The high 2 bits of the lead byte indicate how many bytes follow.
	var pkgLen uint32
	switch lead >> 6 {
	case 0:
		pkgLen = uint32(lead)
	case 1:
		b1, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b1)<<4 | uint32(lead&0xf)
	case 2:
		b1, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		b2, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b2)<<12 | uint32(b1)<<4 | uint32(lead&0xf)
	case 3:
		b1, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		b2, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		b3, err := p.r.ReadByte()
		if err != nil {
			p.r.SetOffset(origOffset)
			return 0, parseResultFailed
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b3)<<20 | uint32(b2)<<12 | uint32(b1)<<4 | uint32(lead&0xf)
	}

	return pkgLen, parseResultOk
}

// parseNumConstant parses a byte/word/dword or qword value from the AML bytestream.
func (p *Parser) parseNumConstant(numBytes uint8) (uint64, parseResult) {
	var (
		next byte
		err  error
		res  uint64
	)

	for c := uint8(0); c < numBytes; c++ {
		if next, err = p.r.ReadByte(); err != nil {
			return 0, parseResultFailed
		}

		res = res | (uint64(next) << (8 * c))
	}

	return res, parseResultOk
}

// parseString parses a string from the AML bytestream and returns back a
// []byte pointing at its contents.
func (p *Parser) parseString() ([]byte, parseResult) {
	// Read ASCII chars till we reach a null byte
	var (
		next byte
		err  error
		res  = parseResultOk
		str  = reflect.SliceHeader{Data: p.r.DataPtr()}
	)

	for {
		next, err = p.r.ReadByte()
		if err != nil {
			res = parseResultFailed
			break
		}

		if next == 0x00 {
			break
		} else if next >= 0x01 && next <= 0x7f { // AsciiChar
			str.Len++
		} else {
			res = parseResultFailed
			break
		}
	}

	str.Cap = str.Len
	return *(*[]byte)(unsafe.Pointer(&str)), res
}

// parseNameString parses a NameString from the AML bytestream and returns back
// a []byte pointing at its contents.
//
// Grammar:
// NameString := RootChar NamePath | PrefixPath NamePath
// PrefixPath := Nothing | '^' PrefixPath
// NamePath := NameSeg | DualNamePath | MultiNamePath | NullName
func (p *Parser) parseNameString() ([]byte, parseResult) {
	var (
		res         = parseResultOk
		str         = reflect.SliceHeader{Data: p.r.DataPtr()}
		next        byte
		err         error
		startOffset = p.r.Offset()
		endOffset   uint32
	)

	// Skip over RootChar ('\') and Caret ('^') prefixes
	for {
		if next, err = p.r.PeekByte(); err != nil {
			return nil, parseResultFailed
		}

		if next != '\\' && next != '^' {
			break
		}

		_, _ = p.r.ReadByte()
	}

	// Decode NamePath; note: the for loop above has already checked whether the
	// next byte is readable via the call to PeekByte. This call to read will
	// never error.
	next, _ = p.r.ReadByte()

	switch next {
	case 0x00: // NullName (null string or a name terminator)
		startOffset = p.r.Offset()
		// return empty string
	case 0x2e: // DualNamePath := DualNamePrefix NameSeg NameSeg
		endOffset = p.r.Offset() + uint32(amlNameLen*2)
		if endOffset > p.r.pkgEnd {
			return nil, parseResultFailed
		}
		p.r.SetOffset(endOffset)
	case 0x2f: // MultiNamePath := MultiNamePrefix SegCount NameSeg(SegCount)
		segCount, err := p.r.ReadByte()
		if segCount == 0 || err != nil {
			return nil, parseResultFailed
		}

		endOffset = p.r.Offset() + uint32(amlNameLen*segCount)
		if endOffset > p.r.pkgEnd {
			return nil, parseResultFailed
		}

		p.r.SetOffset(endOffset)
	default: // NameSeg := LeadNameChar NameChar NameChar NameChar
		// LeadNameChar := 'A' - 'Z' | '_'
		if (next < 'A' || next > 'Z') && next != '_' {
			return nil, parseResultFailed
		}

		endOffset = p.r.Offset() + uint32(amlNameLen-1)
		if endOffset > p.r.pkgEnd {
			return nil, parseResultFailed
		}

		// Skip past the remaning amlNameLen-1 chars
		p.r.SetOffset(endOffset)
	}

	str.Len = int(p.r.Offset() - startOffset)
	str.Cap = str.Len
	return *(*[]byte)(unsafe.Pointer(&str)), res
}

// peekNextOpcode returns the next opcode in the stream without advancing the
// stream pointer.
func (p *Parser) peekNextOpcode() (uint16, parseResult) {
	curOffset := p.r.Offset()
	op, res := p.nextOpcode()
	p.r.SetOffset(curOffset)

	return op, res
}

// nextOpcode decodes an AML opcode from the stream.
func (p *Parser) nextOpcode() (uint16, parseResult) {
	var (
		opLen uint32 = 1
		op    uint16
	)

	next, err := p.r.ReadByte()
	if err != nil {
		return 0xffff, parseResultFailed
	}

	op = uint16(next)

	if next == extOpPrefix {
		op = 0xff
		if next, err = p.r.ReadByte(); err != nil {
			_ = p.r.UnreadByte()
			return 0xffff, parseResultFailed
		}

		opLen++
		op += uint16(next)
	}

	// If this is not a valid opcode, rewind the stream.
	if pOpcodeTableIndex(op, false) == badOpcode {
		p.r.SetOffset(p.r.Offset() - opLen)
		return 0xffff, parseResultFailed
	}

	return op, parseResultOk
}

// scopeCurrent returns the currently active scope.
func (p *Parser) scopeCurrent() *Object {
	return p.objTree.ObjectAt(p.scopeStack[len(p.scopeStack)-1])
}

// scopeEnter enters the given scope.
func (p *Parser) scopeEnter(index uint32) {
	p.scopeStack = append(p.scopeStack, index)
}

// scopeExit exits the current scope.
func (p *Parser) scopeExit() {
	p.scopeStack = p.scopeStack[:len(p.scopeStack)-1]
}

func (p *Parser) pushPkgEnd(pkgEnd uint32) error {
	p.pkgEndStack = append(p.pkgEndStack, pkgEnd)
	return p.r.SetPkgEnd(pkgEnd)
}

func (p *Parser) popPkgEnd() {
	// Pop pkg end if one exists
	if len(p.pkgEndStack) != 0 {
		p.pkgEndStack = p.pkgEndStack[:len(p.pkgEndStack)-1]
	}

	// If the stack is not empty restore the last pushed pkgEnd
	if len(p.pkgEndStack) != 0 && len(p.pkgEndStack) != 0 {
		_ = p.r.SetPkgEnd(p.pkgEndStack[len(p.pkgEndStack)-1])
	}
}

// connectNamedObjArgs visits each named object in the AML tree and populates
// any missing args by consuming the object's siblings.
func (p *Parser) connectNamedObjArgs(objIndex uint32) parseResult {
	var (
		obj          = p.objTree.ObjectAt(objIndex)
		argObj       *Object
		argFlags     pOpArgTypeList
		argCount     uint8
		termArgIndex uint8
		namepath     []byte
		nameIndex    int
		ok           bool
	)

	// The arg list must be visited in reverse order to handle nesting
	for argIndex := obj.lastArgIndex; argIndex != InvalidIndex; argIndex = argObj.prevSiblingIndex {
		// Recursively process children first
		argObj = p.objTree.ObjectAt(argIndex)
		if p.connectNamedObjArgs(argObj.index) != parseResultOk {
			return parseResultFailed
		}

		// Ignore non-named objects and objects not defined by the table currently parsed
		if pOpcodeTable[argObj.infoIndex].flags&pOpFlagNamed == 0 || argObj.tableHandle != p.tableHandle || argObj.firstArgIndex == InvalidIndex || argObj.opcode == pOpIntScopeBlock {
			continue
		}

		// Named opcodes contain a namepath as the first arg
		if namepath, ok = p.objTree.ObjectAt(argObj.firstArgIndex).value.([]byte); !ok {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] named object of type %s without a valid name\n", p.tableName, argObj.amlOffset, pOpcodeName(argObj.opcode))
			return parseResultFailed
		}

		// The last amlNameLen part of the namepath contains the name for this object
		if nameIndex = len(namepath) - amlNameLen; nameIndex < 0 {
			return parseResultFailed
		}
		for i := 0; i < amlNameLen; i++ {
			argObj.name[i] = namepath[nameIndex+i]
		}

		// Check if this object's args specify a TermObj/DataRefObj which
		// would cause the parser to consume any object found till the
		// enclosing package end.
		argFlags = pOpcodeTable[argObj.infoIndex].argFlags
		argCount = argFlags.argCount()
		for termArgIndex = 0; termArgIndex < argCount; termArgIndex++ {
			if argType := argFlags.arg(termArgIndex); argType == pArgTypeTermArg || argType == pArgTypeDataRefObj {
				break
			}
		}

		// All args connected OR no term args; assume object has been completely parsed
		if p.objTree.NumArgs(argObj) == uint32(argCount) || termArgIndex >= argCount {
			continue
		}

		// The parser has already attached args [0, termArgIndex) to
		// the object and has parsed the remaining args as siblings to
		// the object. Detach the missing args from the sibling list and
		// attach them to object.
		if p.attachSiblingsAsArgs(obj, argObj, argCount-termArgIndex, false) != parseResultOk {
			return parseResultFailed
		}
	}

	return parseResultOk
}

// mergeScopeDirectives visits pOpScope objects inside non-executable blocks
// and attempts to merge their contents to the scope referenced by the Scope
// directive.
func (p *Parser) mergeScopeDirectives(objIndex uint32) parseResult {
	var (
		res           = parseResultOk
		obj           = p.objTree.ObjectAt(objIndex)
		firstArgIndex = obj.firstArgIndex
	)

	if objIndex == 0 {
		p.mergedScopes = 0
	}

	// Ignore executable section contents
	if pOpcodeTable[obj.infoIndex].flags&pOpFlagExecutable != 0 {
		return res
	}

	if obj.opcode == pOpScope && obj.tableHandle == p.tableHandle {
		if obj.firstArgIndex == InvalidIndex {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] malformed scope object\n", p.tableName, obj.amlOffset)
			return parseResultFailed
		}

		// Lookup target and make sure it resolves to a scoped object
		nameObj := p.objTree.ObjectAt(obj.firstArgIndex)
		targetName := nameObj.value.([]byte)
		targetIndex := p.objTree.Find(obj.parentIndex, targetName)

		// If the lookup failed we may need to run a couple more mergeScopeDirectives /
		// relocateNamedObjects passes to resolve things. If however no objects got
		// relocated in the previous pass then report this as an error.
		if targetIndex == InvalidIndex {
			if p.resolvePasses > 1 && p.relocatedObjects == 0 {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] unable to resolve reference to scope \"%s\"\n", p.tableName, obj.amlOffset, targetName)
				return parseResultFailed
			}
			return parseResultRequireExtraPass
		}

		// Unless the new parent is an pOpIntScopeBlock it will contain a nested
		// pOpIntScopeBlock which is where we should actually attach obj
		targetObj := p.objTree.ObjectAt(targetIndex)
		if targetObj.opcode != pOpIntScopeBlock {
			for targetIndex, targetObj = targetObj.firstArgIndex, nil; targetIndex != InvalidIndex; targetIndex = p.objTree.ObjectAt(targetIndex).nextSiblingIndex {
				if nextObj := p.objTree.ObjectAt(targetIndex); nextObj.opcode == pOpIntScopeBlock {
					targetObj = nextObj
					break
				}
			}

			if targetObj == nil {
				kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] reference to scope \"%s\" resolved to non-scope object\n", p.tableName, obj.amlOffset, targetName)
				return parseResultFailed
			}
		}

		// Relocate contents to the new scope
		contentsObj := p.objTree.ObjectAt(obj.lastArgIndex)
		firstArgIndex = contentsObj.firstArgIndex
		for siblingIndex := firstArgIndex; siblingIndex != InvalidIndex; {
			argObj := p.objTree.ObjectAt(siblingIndex)
			siblingIndex = argObj.nextSiblingIndex

			p.objTree.detach(contentsObj, argObj)
			p.objTree.append(targetObj, argObj)
		}

		// The scope object and its children can now be freed
		p.objTree.free(nameObj)
		p.objTree.free(contentsObj)
		p.objTree.free(obj)
		p.mergedScopes++
	}

	// Recursively process nested objects
	for siblingIndex := firstArgIndex; siblingIndex != InvalidIndex; {
		argObj := p.objTree.ObjectAt(siblingIndex)
		siblingIndex = argObj.nextSiblingIndex

		switch p.mergeScopeDirectives(argObj.index) {
		case parseResultFailed:
			return parseResultFailed
		case parseResultRequireExtraPass:
			res = parseResultRequireExtraPass
		}
	}

	return res
}

// relocateNamedObjects processes named objects whose name contains a relative
// or absolute path prefix and moves the objects to the appropriate parent.
func (p *Parser) relocateNamedObjects(objIndex uint32) parseResult {
	var (
		obj               = p.objTree.ObjectAt(objIndex)
		flags             = pOpcodeTable[obj.infoIndex].flags
		argObj, targetObj *Object
		targetIndex       uint32
		nameIndex         int
		namepath          []byte
		ok                bool
		res               = parseResultOk
	)

	if objIndex == 0 {
		p.relocatedObjects = 0
	}

	// Ignore executable section contents
	if flags&pOpFlagExecutable != 0 {
		return res
	}

	if flags&pOpFlagNamed != 0 && obj.firstArgIndex != InvalidIndex && obj.tableHandle == p.tableHandle && obj.opcode != pOpIntScopeBlock {
		// This is a named object. Check if its namepath requires relocation
		if namepath, ok = p.objTree.ObjectAt(obj.firstArgIndex).value.([]byte); !ok {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] named object of type %s without a valid name\n", p.tableName, obj.amlOffset, pOpcodeName(obj.opcode))
			return parseResultFailed
		}

		// If the namepath is longer than amlNameLen then it's a scoped path
		// expression that needs to be resolved using the closest named ancestor as
		// the scope. If the resolve succeeds, the object will be attached to the
		// resolved parent and the namepath will be cleaned up.
		if nameIndex = len(namepath) - amlNameLen; nameIndex > 0 {
			targetIndex = p.objTree.Find(p.objTree.ClosestNamedAncestor(obj), namepath[:nameIndex])
			if targetIndex == InvalidIndex {
				if p.resolvePasses > maxResolvePasses {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] unable to resolve relocation path %s for object of type %s after %d passes; aborting\n", p.tableName, obj.amlOffset, namepath[:], pOpcodeName(obj.opcode), p.resolvePasses)
					return parseResultFailed
				}
				return parseResultRequireExtraPass
			}

			targetObj = p.objTree.ObjectAt(targetIndex)
			if targetObj.opcode != pOpIntScopeBlock {
				for targetIndex, targetObj = targetObj.firstArgIndex, nil; targetIndex != InvalidIndex; targetIndex = p.objTree.ObjectAt(targetIndex).nextSiblingIndex {
					if nextObj := p.objTree.ObjectAt(targetIndex); nextObj.opcode == pOpIntScopeBlock {
						targetObj = nextObj
						break
					}
				}

				if targetObj == nil {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] relocation path \"%s\" resolved to non-scope object\n", p.tableName, obj.amlOffset, namepath[:])
					return parseResultFailed
				}
			}
			p.objTree.detach(p.objTree.ObjectAt(obj.parentIndex), obj)
			p.objTree.append(targetObj, obj)
			p.objTree.ObjectAt(obj.firstArgIndex).value = namepath[nameIndex:]
			p.relocatedObjects++
		}
	}

	// Recursively process nested objects
	for siblingIndex := obj.firstArgIndex; siblingIndex != InvalidIndex; {
		argObj = p.objTree.ObjectAt(siblingIndex)
		siblingIndex = argObj.nextSiblingIndex

		switch p.relocateNamedObjects(argObj.index) {
		case parseResultFailed:
			return parseResultFailed
		case parseResultRequireExtraPass:
			res = parseResultRequireExtraPass
		}
	}

	return res
}

// parseDeferredBlocks attempts to parse any objects that contain objects that
// are flagged as deferred (e.g. Buffers and BankFields).
func (p *Parser) parseDeferredBlocks(objIndex uint32) parseResult {
	obj := p.objTree.ObjectAt(objIndex)
	if pOpcodeTable[obj.infoIndex].flags&pOpFlagDeferParsing != 0 && obj.tableHandle == p.tableHandle {
		p.mode = parseModeAllBlocks

		// Set stream offset to the first arg
		p.r.SetPkgEnd(p.streamEnd)
		p.r.SetOffset(obj.amlOffset + 1)
		if obj.opcode > 0xff { // opcode length = 2
			_, _ = p.r.ReadByte()
		}

		if p.parseObjectArgs(obj) != parseResultOk {
			return parseResultFailed
		}

		// As we are using a different parse flow than the one used for
		// non-deferred we need to manually clean up the pkgEndStack after the
		// object has been parsed.
		for len(p.pkgEndStack) != 0 {
			p.popPkgEnd()
		}

		// The parseObjectArgs() call has parsed all children of the deferred node.
		// At this point we can simply return without processing the children.
		return parseResultOk
	}

	// Recursively process children
	for argIndex := obj.firstArgIndex; argIndex != InvalidIndex; argIndex = p.objTree.ObjectAt(argIndex).nextSiblingIndex {
		if p.parseDeferredBlocks(argIndex) != parseResultOk {
			return parseResultFailed
		}
	}

	return parseResultOk
}

// connectNonNamedObjArgs behaves in a similar way as connectNamedObjArgs but
// only operates on non-named objects.
func (p *Parser) connectNonNamedObjArgs(objIndex uint32) parseResult {
	var (
		obj          = p.objTree.ObjectAt(objIndex)
		argObj       *Object
		argFlags     pOpArgTypeList
		argCount     uint8
		termArgIndex uint8
	)

	// The arg list must be visited in reverse order to handle nesting
	for argIndex := obj.lastArgIndex; argIndex != InvalidIndex; argIndex = argObj.prevSiblingIndex {
		// Recursively process children first
		argObj = p.objTree.ObjectAt(argIndex)
		if p.connectNonNamedObjArgs(argObj.index) != parseResultOk {
			return parseResultFailed
		}

		// Ignore named objects and objects not defined by the table currently parsed
		if pOpcodeTable[argObj.infoIndex].flags&pOpFlagNamed != 0 || argObj.tableHandle != p.tableHandle {
			continue
		}

		// Check if this object's args specify a TermObj/DataRefObj which
		// would cause the parser to consume any object found till the
		// enclosing package end.
		argFlags = pOpcodeTable[argObj.infoIndex].argFlags
		argCount = argFlags.argCount()
		for termArgIndex = 0; termArgIndex < argCount; termArgIndex++ {
			if argType := argFlags.arg(termArgIndex); argType == pArgTypeTermArg || argType == pArgTypeDataRefObj {
				break
			}
		}

		// No term args OR we have parsed beyond the TermArg; assume object has been completely parsed
		if termArgIndex >= argCount || p.objTree.NumArgs(argObj) > uint32(termArgIndex) {
			continue
		}

		// The parser has already attached args [0, termArgIndex) to
		// the object and has parsed the remaining args as siblings to
		// the object. Detach the missing args from the sibling list and
		// attach them to object. The following call may also return back
		// parseResultRequireExtraPass which is OK at this stage.
		if p.attachSiblingsAsArgs(obj, argObj, argCount-termArgIndex, true) == parseResultFailed {
			return parseResultFailed
		}
	}

	return parseResultOk
}

// resolveMethodCalls visits each object with the pOpIntNamePathOrMethodCall
// opcode and resolves it to either a pOpIntMethodCall or a pOpIntNamePath based
// on the result of a lookup using the attached namepath.
//
// If the namepath lookup resolves into a method definition, the object will be
// mutated into an pOpIntMethodCall and the expected number of arguments (as
// specified by the referenced method) will be extracted from the object's
// sibling list.
//
// If the nemepath resolves into a non-method object then the object will be
// instead mutated into an pOpIntResolvedNamePath.
//
// If the lookup fails, then the object will be mutated into an pOpIntNamePath
// to be resolved at run-time.
func (p *Parser) resolveMethodCalls(objIndex uint32) parseResult {
	var (
		obj                 = p.objTree.ObjectAt(objIndex)
		ok                  bool
		argObj, resolvedObj *Object
		argCount            uint64
		targetIndex         uint32
	)

	// The arg list must be visited in reverse order to handle nesting:
	// FOOF( BARB(1,2), 3) will appear in the stream as: FOOF BARB 1 2 3
	// By processing the objects in reverse order we sequentially transform
	// the object list as follows:
	// FOOF BARB(1,2) 3
	// FOOF( BARB(1,2), 3)
	for argIndex := obj.lastArgIndex; argIndex != InvalidIndex; argIndex = argObj.prevSiblingIndex {
		// Recursively process children first
		argObj = p.objTree.ObjectAt(argIndex)
		if p.resolveMethodCalls(argObj.index) != parseResultOk {
			return parseResultFailed
		}

		if argObj.opcode != pOpIntNamePathOrMethodCall || argObj.tableHandle != p.tableHandle {
			continue
		}

		// Resolve namepath
		targetIndex = p.objTree.Find(argObj.parentIndex, argObj.value.([]byte))
		switch targetIndex {
		case InvalidIndex:
			// Treat this as a namepath to be resolved at run-time
			argObj.opcode = pOpIntNamePath
			argObj.infoIndex = pOpcodeTableIndex(argObj.opcode, true)
		default:
			resolvedObj = p.objTree.ObjectAt(targetIndex)

			switch resolvedObj.opcode {
			case pOpMethod:
				// Mutate into a method call with value pointing at the resolved method index
				argObj.opcode = pOpIntMethodCall
				argObj.infoIndex = pOpcodeTableIndex(argObj.opcode, true)
				argObj.value = resolvedObj.index

				// Consume the required number of args which are encoded as
				// bits [0:2] of the method obj 2nd arg
				methodFlagsObj := p.objTree.ArgAt(resolvedObj, 1)
				if methodFlagsObj == nil {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] target method \"%s\" is missing a flag object\n", p.tableName, argObj.amlOffset, resolvedObj.name[:])
					return parseResultFailed
				}

				argCount, ok = methodFlagsObj.value.(uint64)
				if !ok {
					kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] target method \"%s\" contains a malformed flag object\n", p.tableName, argObj.amlOffset, resolvedObj.name[:])
					return parseResultFailed
				}

				if p.attachSiblingsAsArgs(obj, argObj, uint8(argCount&0x7), true) != parseResultOk {
					return parseResultFailed
				}
			default:
				// Mutate into a resolved name path with value pointing at the resolved object intdex
				argObj.opcode = pOpIntResolvedNamePath
				argObj.infoIndex = pOpcodeTableIndex(argObj.opcode, true)
				argObj.value = resolvedObj.index
			}
		}
	}

	return parseResultOk
}

// attachSiblingsAsArgs detaches numArgs sibling nodes of targetObj and
// re-attaches them as args to targetObj. If we run out of sibling nodes and
// useParentSiblings is set to true, then attachSiblingsAsArgs will continue
// detaching siblings from the parent
func (p *Parser) attachSiblingsAsArgs(parentObj, targetObj *Object, numArgs uint8, useParentSiblings bool) parseResult {
	var siblingObj *Object

	for siblingIndex := targetObj.nextSiblingIndex; numArgs > 0; numArgs-- {
		if siblingIndex == InvalidIndex && useParentSiblings {
			siblingIndex = parentObj.nextSiblingIndex
		}

		if siblingIndex == InvalidIndex {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: 0x%x] unexpected arg count for opcode: %s (0x%x)\n", p.tableName, targetObj.amlOffset, pOpcodeName(targetObj.opcode), targetObj.opcode)
			return parseResultFailed
		}

		// Update siblingIndex before siblingObj gets detached
		siblingObj = p.objTree.ObjectAt(siblingIndex)
		siblingIndex = siblingObj.nextSiblingIndex

		p.objTree.detach(parentObj, siblingObj)
		p.objTree.append(targetObj, siblingObj)
	}
	return parseResultOk
}
