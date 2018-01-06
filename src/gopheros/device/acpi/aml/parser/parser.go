package parser

import (
	"gopheros/device/acpi/aml/entity"
	"gopheros/device/acpi/table"
	"gopheros/kernel"
	"gopheros/kernel/kfmt"
	"io"
	"unsafe"
)

var (
	errParsingAML        = &kernel.Error{Module: "acpi_aml_parser", Message: "could not parse AML bytecode"}
	errResolvingEntities = &kernel.Error{Module: "acpi_aml_parser", Message: "AML bytecode contains unresolvable entities"}
)

type parseOpt uint8

const (
	parseOptSkipMethodBodies parseOpt = iota
	parseOptParseMethodBodies
)

// Parser implements an AML parser.
type Parser struct {
	r           amlStreamReader
	errWriter   io.Writer
	root        entity.Container
	scopeStack  []entity.Container
	tableName   string
	tableHandle uint8

	parseOptions parseOpt
}

// NewParser returns a new AML parser instance.
func NewParser(errWriter io.Writer, rootEntity entity.Container) *Parser {
	return &Parser{
		errWriter: errWriter,
		root:      rootEntity,
	}
}

// ParseAML attempts to parse the AML byte-code contained in the supplied ACPI
// table tagging each scoped entity with the supplied table handle. The parser
// emits any encountered errors to the specified errWriter.
func (p *Parser) ParseAML(tableHandle uint8, tableName string, header *table.SDTHeader) *kernel.Error {
	p.tableHandle = tableHandle
	p.tableName = tableName
	p.r.Init(
		uintptr(unsafe.Pointer(header)),
		header.Length,
		uint32(unsafe.Sizeof(table.SDTHeader{})),
	)

	// Pass 1: decode bytecode and build entitites without recursing into
	// function bodies.
	p.parseOptions = parseOptSkipMethodBodies
	p.scopeStack = nil
	p.scopeEnter(p.root)
	if !p.parseObjList(header.Length) {
		lastOp, _ := p.r.LastByte()
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] error parsing AML bytecode (last op 0x%x)\n", p.tableName, p.r.Offset()-1, lastOp)
		return errParsingAML
	}
	p.scopeExit()

	// Pass 2: parse method bodies, check entity parents and resolve all
	// symbol references
	var resolveFailed bool
	entity.Visit(0, p.root, entity.TypeAny, func(_ int, ent entity.Entity) bool {
		if method, isMethod := ent.(*entity.Method); isMethod {
			resolveFailed = resolveFailed || !p.parseMethodBody(method)

			// Don't recurse into method bodies; their contents
			// will be lazilly resolved by the VM
			return false
		}

		// Populate parents for any entity args that are also entities but are not
		// linked to a parent (e.g. a package inside a named entity).
		for _, arg := range ent.Args() {
			if argEnt, isArgEnt := arg.(entity.Entity); isArgEnt && argEnt.Parent() == nil {
				argEnt.SetParent(ent.Parent())
			}
		}

		// Resolve any symbol references
		if lazyRef, ok := ent.(entity.LazyRefResolver); ok {
			if err := lazyRef.ResolveSymbolRefs(p.root); err != nil {
				kfmt.Fprintf(p.errWriter, "%s\n", err.Message)
				resolveFailed = true
				return false
			}
		}

		return true
	})

	if resolveFailed {
		return errResolvingEntities
	}

	return nil
}

// parseObjList tries to parse an AML object list. Object lists are usually
// specified together with a pkgLen block which is used to calculate the max
// read offset that the parser may reach.
func (p *Parser) parseObjList(maxOffset uint32) bool {
	for !p.r.EOF() && p.r.Offset() < maxOffset {
		if !p.parseObj() {
			return false
		}
	}

	return true
}

func (p *Parser) parseObj() bool {
	var (
		curOffset uint32
		pkgLen    uint32
		info      *opcodeInfo
		ok        bool
	)

	// If we cannot decode the next opcode then this may be a method
	// invocation or a name reference.
	curOffset = p.r.Offset()
	if info, ok = p.nextOpcode(); !ok {
		p.r.SetOffset(curOffset)
		return p.parseNamedRef()
	}

	hasPkgLen := info.flags.is(opFlagHasPkgLen) || info.argFlags.contains(opArgTermList) || info.argFlags.contains(opArgFieldList)

	if hasPkgLen {
		curOffset = p.r.Offset()
		if pkgLen, ok = p.parsePkgLength(); !ok {
			return false
		}
	}

	// If we encounter a named scope we need to look it up and parse the arg list relative to it
	switch {
	case info.op == entity.OpScope:
		return p.parseScope(curOffset + pkgLen)
	case info.flags.is(opFlagNamed | opFlagScoped):
		return p.parseNamespacedObj(info, curOffset+pkgLen)
	}

	// Create appropriate object for opcode type and attach it to current scope unless it is
	// a device named scope in which case it may define a relative scope name
	obj := p.makeObjForOpcode(info)
	p.scopeCurrent().Append(obj)

	if argCount := info.argFlags.argCount(); argCount > 0 {
		for argIndex := uint8(0); argIndex < argCount; argIndex++ {
			if !p.parseArg(
				info,
				obj,
				argIndex,
				info.argFlags.arg(argIndex),
				curOffset+pkgLen,
			) {
				return false
			}
		}
	}

	return p.finalizeObj(info.op, obj)
}

// finalizeObj applies post-parse logic for special object types.
func (p *Parser) finalizeObj(op entity.AMLOpcode, obj entity.Entity) bool {
	switch op {
	case entity.OpElse:
		// If this is an else block we need to append it as an argument to the
		// If block
		// Pop Else block of the current scope
		curScope := p.scopeCurrent()
		curScope.Remove(curScope.Last())
		prevObj := curScope.Last()
		if prevObj.Opcode() != entity.OpIf {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] encountered else block without a matching if block\n", p.tableName, p.r.Offset())
			return false
		}

		// If predicate(0) then(1) else(2)
		prevObj.SetArg(2, obj)
	}

	return true
}

// parseScope reads a scope name from the AML bytestream, enters it and parses
// an objlist relative to it. The referenced scope must be one of:
// - one of the pre-defined scopes
// - device
// - processor
// - thermal zone
// - power resource
func (p *Parser) parseScope(maxReadOffset uint32) bool {
	name, ok := p.parseNameString()
	if !ok {
		return false
	}

	target := entity.FindInScope(p.scopeCurrent(), p.root, name)
	if target == nil {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] undefined scope: %s\n", p.tableName, p.r.Offset(), name)
		return false
	}

	switch target.Opcode() {
	case entity.OpDevice, entity.OpProcessor, entity.OpThermalZone, entity.OpPowerRes:
		// ok
	default:
		// Only allow if this is a named scope
		if target.Name() == "" {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] %s does not refer to a scoped object\n", p.tableName, p.r.Offset(), name)
			return false
		}
	}

	p.scopeEnter(target.(entity.Container))
	ok = p.parseObjList(maxReadOffset)
	p.scopeExit()

	return ok
}

// parseNamespacedObj reads a scope target name from the AML bytestream,
// attaches the appropriate object depending on the opcode to the correct
// parent scope and then parses any contained objects. The contained objects
// will be appended inside the newly constructed scope.
func (p *Parser) parseNamespacedObj(info *opcodeInfo, maxReadOffset uint32) bool {
	scopeExpr, ok := p.parseNameString()
	if !ok {
		return false
	}

	parent, name := entity.ResolveScopedPath(p.scopeCurrent(), p.root, scopeExpr)
	if parent == nil {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] undefined scope target: %s (current scope: %s)\n", p.tableName, p.r.Offset(), scopeExpr, p.scopeCurrent().Name())
		return false
	}

	var obj entity.Container
	switch info.op {
	case entity.OpDevice:
		obj = entity.NewDevice(p.tableHandle, name)
	case entity.OpProcessor:
		obj = entity.NewProcessor(p.tableHandle, name)
	case entity.OpPowerRes:
		obj = entity.NewPowerResource(p.tableHandle, name)
	case entity.OpThermalZone:
		obj = entity.NewThermalZone(p.tableHandle, name)
	case entity.OpMethod:
		obj = entity.NewMethod(p.tableHandle, name)
	default:
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] unsupported namespaced op: %s (current scope: %s)\n", p.tableName, p.r.Offset(), info.op.String(), p.scopeCurrent().Name())
		return false
	}

	// Parse any args that follow the name. The last arg is always an ArgTermList
	parent.Append(obj)
	for argIndex := uint8(1); argIndex < info.argFlags.argCount(); argIndex++ {
		if !p.parseArg(info, obj, argIndex, info.argFlags.arg(argIndex), maxReadOffset) {
			return false
		}
	}

	return ok && p.finalizeObj(info.op, obj)
}

func (p *Parser) parseArg(info *opcodeInfo, obj entity.Entity, argIndex uint8, argType opArgFlag, maxReadOffset uint32) bool {
	var (
		arg interface{}
		ok  bool
	)

	switch argType {
	case opArgNameString:
		arg, ok = p.parseNameString()
	case opArgByteData:
		arg, ok = p.parseNumConstant(1)
	case opArgWord:
		arg, ok = p.parseNumConstant(2)
	case opArgDword:
		arg, ok = p.parseNumConstant(4)
	case opArgQword:
		arg, ok = p.parseNumConstant(8)
	case opArgString:
		arg, ok = p.parseString()
	case opArgTermObj, opArgDataRefObj:
		arg, ok = p.parseArgObj()
	case opArgSimpleName:
		arg, ok = p.parseSimpleName()
	case opArgSuperName:
		arg, ok = p.parseSuperName()
	case opArgTarget:
		arg, ok = p.parseTarget()
	case opArgTermList:
		// If this is a method and the SkipMethodBodies option is set
		// then record the body start and end offset so we can parse
		// it at a later stage.
		if method, isMethod := obj.(*entity.Method); isMethod && p.parseOptions == parseOptSkipMethodBodies {
			method.BodyStartOffset = p.r.Offset()
			method.BodyEndOffset = maxReadOffset
			p.r.SetOffset(maxReadOffset)
			return true
		}

		// If object is a scoped entity enter it's scope before parsing
		// the term list. Otherwise, create an unnamed scope, attach it
		// as the next argument to obj and enter that.
		if s, isScopeEnt := obj.(entity.Container); isScopeEnt {
			p.scopeEnter(s)
		} else {
			// Create an unnamed scope (e.g if, else, while scope)
			ns := entity.NewScope(info.op, p.tableHandle, "")
			p.scopeEnter(ns)
			obj.SetArg(argIndex, ns)
		}

		ok = p.parseObjList(maxReadOffset)
		p.scopeExit()
		return ok
	case opArgFieldList:
		return p.parseFieldList(obj, maxReadOffset)
	case opArgByteList:
		var bl []byte
		for p.r.Offset() < maxReadOffset {
			b, err := p.r.ReadByte()
			if err != nil {
				return false
			}
			bl = append(bl, b)
		}
		arg, ok = bl, true
	}

	if !ok {
		return false
	}

	return obj.SetArg(argIndex, arg)
}

func (p *Parser) parseArgObj() (entity.Entity, bool) {
	if ok := p.parseObj(); !ok {
		return nil, false
	}

	curScope := p.scopeCurrent()
	obj := curScope.Last()
	curScope.Remove(obj)
	return obj, true
}

func (p *Parser) makeObjForOpcode(info *opcodeInfo) entity.Entity {
	var obj entity.Entity

	switch {
	case info.op == entity.OpOpRegion:
		obj = entity.NewRegion(p.tableHandle)
	case info.op == entity.OpBuffer:
		obj = entity.NewBuffer(p.tableHandle)
	case info.op == entity.OpMutex:
		obj = entity.NewMutex(p.tableHandle)
	case info.op == entity.OpEvent:
		obj = entity.NewEvent(p.tableHandle)
	case info.op == entity.OpField:
		obj = entity.NewField(p.tableHandle)
	case info.op == entity.OpIndexField:
		obj = entity.NewIndexField(p.tableHandle)
	case info.op == entity.OpBankField:
		obj = entity.NewBankField(p.tableHandle)
	case info.op == entity.OpCreateField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 0)
	case info.op == entity.OpCreateBitField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 1)
	case info.op == entity.OpCreateByteField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 8)
	case info.op == entity.OpCreateWordField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 16)
	case info.op == entity.OpCreateDWordField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 32)
	case info.op == entity.OpCreateQWordField:
		obj = entity.NewBufferField(info.op, p.tableHandle, 64)
	case info.op == entity.OpZero:
		obj = entity.NewConst(info.op, p.tableHandle, uint64(0))
	case info.op == entity.OpOne:
		obj = entity.NewConst(info.op, p.tableHandle, uint64(1))
	case info.op == entity.OpOnes:
		obj = entity.NewConst(info.op, p.tableHandle, uint64((1<<64)-1))
	case info.flags.is(opFlagConstant):
		obj = entity.NewConst(info.op, p.tableHandle, nil) // will be parsed as an arg
	case info.op == entity.OpPackage || info.op == entity.OpVarPackage:
		obj = entity.NewPackage(info.op, p.tableHandle)
	case info.flags.is(opFlagScoped):
		obj = entity.NewScope(info.op, p.tableHandle, "")
	case info.flags.is(opFlagNamed):
		obj = entity.NewGenericNamed(info.op, p.tableHandle)
	default:
		obj = entity.NewGeneric(info.op, p.tableHandle)
	}

	return obj
}

// parseMethodBody parses the entities that make up a method's body. After the
// entire AML tree has been parsed, the parser makes a second pass and calls
// parseMethodBody for each Method entity.
//
// By deferring the parsing of the method body, we ensure that the parser can
// lookup the method declarations (even if forward declarations are used) for
// each method invocation. As method declarations contain information about the
// expected argument count, the parser can use this information to properly
// parse the invocation arguments. For more details see: parseNamedRef
func (p *Parser) parseMethodBody(method *entity.Method) bool {
	p.parseOptions = parseOptParseMethodBodies
	p.scopeEnter(method)
	p.r.SetOffset(method.BodyStartOffset)
	ok := p.parseArg(&opcodeTable[methodOpInfoIndex], method, 2, opArgTermList, method.BodyEndOffset)
	p.scopeExit()

	return ok
}

// parseNamedRef attempts to parse either a method invocation or a named
// reference. As AML allows for forward references, the actual contents for
// this entity will not be known until the entire AML stream has been parsed.
//
// Grammar:
// MethodInvocation := NameString TermArgList
// TermArgList = Nothing | TermArg TermArgList
// TermArg = Type2Opcode | DataObject | ArgObj | LocalObj | MethodInvocation
func (p *Parser) parseNamedRef() bool {
	name, ok := p.parseNameString()
	if !ok {
		return false
	}

	// Check if this is a method invocation
	ent := entity.FindInScope(p.scopeCurrent(), p.root, name)
	if methodDef, isMethod := ent.(*entity.Method); isMethod {
		var (
			curOffset uint32
			argIndex  uint8
			arg       entity.Entity
			argList   []interface{}
		)

		for argIndex < methodDef.ArgCount && !p.r.EOF() {
			// Peek next opcode
			curOffset = p.r.Offset()
			nextOpcode, ok := p.nextOpcode()
			p.r.SetOffset(curOffset)

			switch {
			case ok && (entity.OpIsType2(nextOpcode.op) || entity.OpIsArg(nextOpcode.op) || entity.OpIsDataObject(nextOpcode.op)):
				arg, ok = p.parseArgObj()
			default:
				// It may be a nested invocation or named ref
				ok = p.parseNamedRef()
				if ok {
					arg = p.scopeCurrent().Last()
					p.scopeCurrent().Remove(arg)
				}
			}

			// No more TermArgs to parse
			if !ok {
				p.r.SetOffset(curOffset)
				break
			}

			argList = append(argList, arg)
			argIndex++
		}

		// Check whether all expected arguments have been parsed
		if argIndex != methodDef.ArgCount {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] unexpected arglist end for method %s invocation: expected %d; got %d\n", p.tableName, p.r.Offset(), name, methodDef.ArgCount, argIndex)
			return false
		}

		return p.scopeCurrent().Append(entity.NewInvocation(p.tableHandle, methodDef, argList))
	}

	// Otherwise this is a reference to a named entity
	return p.scopeCurrent().Append(entity.NewReference(p.tableHandle, name))
}

func (p *Parser) nextOpcode() (*opcodeInfo, bool) {
	next, err := p.r.ReadByte()
	if err != nil {
		return nil, false
	}

	if next != extOpPrefix {
		index := opcodeMap[next]
		if index == badOpcode {
			return nil, false
		}
		return &opcodeTable[index], true
	}

	// Scan next byte to figure out the opcode
	if next, err = p.r.ReadByte(); err != nil {
		return nil, false
	}

	index := extendedOpcodeMap[next]
	if index == badOpcode {
		return nil, false
	}
	return &opcodeTable[index], true
}

// parseFieldList parses a list of FieldElements until the reader reaches
// maxReadOffset and appends them to the current scope. Depending on the opcode
// this method will emit either fieldUnit objects or indexField objects
//
// Grammar:
// FieldElement := NamedField | ReservedField | AccessField | ExtendedAccessField | ConnectField
// NamedField := NameSeg PkgLength
// ReservedField := 0x00 PkgLength
// AccessField := 0x1 AccessType AccessAttrib
// ConnectField := 0x02 NameString | 0x02 BufferData
// ExtendedAccessField := 0x3 AccessType ExtendedAccessType AccessLength
func (p *Parser) parseFieldList(fieldEnt entity.Entity, maxReadOffset uint32) bool {
	var (
		ok bool

		accessType entity.FieldAccessType

		bitWidth           uint32
		curBitOffset       uint32
		connectionName     string
		unitName           string
		resolvedConnection entity.Entity
		accessAttrib       entity.FieldAccessAttrib
		accessByteCount    uint8
	)

	// Load default field access rule; it applies to all field units unless
	// overridden via a directive in the field unit list
	if accessProvider, isProvider := fieldEnt.(entity.FieldAccessTypeProvider); isProvider {
		accessType = accessProvider.DefaultAccessType()
	} else {
		// not a field entity
		return false
	}

	for p.r.Offset() < maxReadOffset {
		next, err := p.r.ReadByte()
		if err != nil {
			return false
		}

		switch next {
		case 0x00: // ReservedField; generated by the Offset() command
			bitWidth, ok = p.parsePkgLength()
			if !ok {
				return false
			}

			curBitOffset += bitWidth
			continue
		case 0x1: // AccessField; set access attributes for following fields
			next, err := p.r.ReadByte()
			if err != nil {
				return false
			}
			accessType = entity.FieldAccessType(next & 0xf) // access type; bits[0:3]

			attrib, err := p.r.ReadByte()
			if err != nil {
				return false
			}

			// To specify AccessAttribBytes, RawBytes and RawProcessBytes
			// the ASL compiler will emit an ExtendedAccessField opcode.
			accessByteCount = 0
			accessAttrib = entity.FieldAccessAttrib(attrib)

			continue
		case 0x2: // ConnectField => <0x2> NameString> | <0x02> TermObj => Buffer
			curOffset := p.r.Offset()
			if connectionName, ok = p.parseNameString(); !ok {
				// Rewind and try parsing it as an object
				p.r.SetOffset(curOffset)
				if resolvedConnection, ok = p.parseArgObj(); !ok {
					return false
				}
			}
		case 0x3: // ExtendedAccessField => <0x03> AccessType ExtendedAccessAttrib AccessLength
			next, err := p.r.ReadByte()
			if err != nil {
				return false
			}
			accessType = entity.FieldAccessType(next & 0xf) // access type; bits[0:3]

			extAccessAttrib, err := p.r.ReadByte()
			if err != nil {
				return false
			}

			accessByteCount, err = p.r.ReadByte()
			if err != nil {
				return false
			}

			switch extAccessAttrib {
			case 0x0b:
				accessAttrib = entity.FieldAccessAttribBytes
			case 0xe:
				accessAttrib = entity.FieldAccessAttribRawBytes
			case 0x0f:
				accessAttrib = entity.FieldAccessAttribRawProcessBytes
			}
		default: // NamedField
			_ = p.r.UnreadByte()
			if unitName, ok = p.parseNameString(); !ok {
				return false
			}

			bitWidth, ok = p.parsePkgLength()
			if !ok {
				return false
			}

			// According to the spec, the field elements are should
			// be visible at the same scope as the Field that declares them
			unit := entity.NewFieldUnit(p.tableHandle, unitName)
			unit.Field = fieldEnt
			unit.AccessType = accessType
			unit.AccessAttrib = accessAttrib
			unit.ByteCount = accessByteCount
			unit.BitOffset = curBitOffset
			unit.BitWidth = bitWidth
			unit.ConnectionName = connectionName
			unit.Connection = resolvedConnection

			p.scopeCurrent().Append(unit)
			curBitOffset += bitWidth
		}
	}

	return ok && p.r.Offset() == maxReadOffset
}

// parsePkgLength parses a PkgLength value from the AML bytestream.
func (p *Parser) parsePkgLength() (uint32, bool) {
	lead, err := p.r.ReadByte()
	if err != nil {
		return 0, false
	}

	// The high 2 bits of the lead byte indicate how many bytes follow.
	var pkgLen uint32
	switch lead >> 6 {
	case 0:
		pkgLen = uint32(lead)
	case 1:
		b1, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b1)<<4 | uint32(lead&0xf)
	case 2:
		b1, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		b2, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b2)<<12 | uint32(b1)<<4 | uint32(lead&0xf)
	case 3:
		b1, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		b2, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		b3, err := p.r.ReadByte()
		if err != nil {
			return 0, false
		}

		// lead bits 0-3 are the lsb of the length nybble
		pkgLen = uint32(b3)<<20 | uint32(b2)<<12 | uint32(b1)<<4 | uint32(lead&0xf)
	}

	return pkgLen, true
}

// parseNumConstant parses a byte/word/dword or qword value from the AML bytestream.
func (p *Parser) parseNumConstant(numBytes uint8) (uint64, bool) {
	var (
		next byte
		err  error
		res  uint64
	)

	for c := uint8(0); c < numBytes; c++ {
		if next, err = p.r.ReadByte(); err != nil {
			return 0, false
		}

		res = res | (uint64(next) << (8 * c))
	}

	return res, true
}

// parseString parses a string from the AML bytestream.
func (p *Parser) parseString() (string, bool) {
	// Read ASCII chars till we reach a null byte
	var (
		next byte
		err  error
		str  []byte
	)

	for {
		next, err = p.r.ReadByte()
		if err != nil {
			return "", false
		}

		if next == 0x00 {
			break
		} else if next >= 0x01 && next <= 0x7f { // AsciiChar
			str = append(str, next)
		} else {
			return "", false
		}
	}
	return string(str), true
}

// parseSuperName attempts to pass a SuperName from the AML bytestream.
//
// Grammar:
// SuperName := SimpleName | DebugObj | Type6Opcode
// SimpleName := NameString | ArgObj | LocalObj
func (p *Parser) parseSuperName() (interface{}, bool) {
	// Try parsing as SimpleName
	curOffset := p.r.Offset()
	if obj, ok := p.parseSimpleName(); ok {
		return obj, ok
	}

	// Rewind and try parsing as object
	p.r.SetOffset(curOffset)
	return p.parseArgObj()
}

// parseSimpleName attempts to pass a SimpleName from the AML bytestream.
//
// Grammar:
// SimpleName := NameString | ArgObj | LocalObj
func (p *Parser) parseSimpleName() (interface{}, bool) {
	// Peek next opcode
	curOffset := p.r.Offset()
	nextOpcode, ok := p.nextOpcode()

	var obj interface{}

	switch {
	case ok && entity.OpIsArg(nextOpcode.op):
		obj, ok = entity.NewGeneric(nextOpcode.op, p.tableHandle), true
	default:
		// Rewind and try parsing as NameString
		p.r.SetOffset(curOffset)
		obj, ok = p.parseNameString()
	}

	return obj, ok
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
func (p *Parser) parseTarget() (interface{}, bool) {
	// Peek next opcode
	curOffset := p.r.Offset()
	nextOpcode, ok := p.nextOpcode()
	p.r.SetOffset(curOffset)

	if ok {
		switch {
		case nextOpcode.op == entity.OpZero: // this is actually a NullName
			p.r.SetOffset(curOffset + 1)
			return entity.NewConst(entity.OpStringPrefix, p.tableHandle, ""), true
		case entity.OpIsArg(nextOpcode.op) || nextOpcode.op == entity.OpRefOf || nextOpcode.op == entity.OpDerefOf || nextOpcode.op == entity.OpIndex || nextOpcode.op == entity.OpDebug: // LocalObj | ArgObj | Type6 | DebugObj
		default:
			// Unexpected opcode
			return nil, false
		}

		// We can use parseObj for parsing
		return p.parseArgObj()
	}

	// In this case, this is either a NameString or a control method invocation.
	if ok := p.parseNamedRef(); ok {
		obj := p.scopeCurrent().Last()
		p.scopeCurrent().Remove(obj)
		return obj, ok
	}

	return nil, false
}

// parseNameString parses a NameString from the AML bytestream.
//
// Grammar:
// NameString := RootChar NamePath | PrefixPath NamePath
// PrefixPath := Nothing | '^' PrefixPath
// NamePath := NameSeg | DualNamePath | MultiNamePath | NullName
func (p *Parser) parseNameString() (string, bool) {
	var str []byte

	// NameString := RootChar NamePath | PrefixPath NamePath
	next, err := p.r.PeekByte()
	if err != nil {
		return "", false
	}

	switch next {
	case '\\': // RootChar
		str = append(str, next)
		_, _ = p.r.ReadByte()
	case '^': // PrefixPath := Nothing | '^' PrefixPath
		str = append(str, next)
		_, _ = p.r.ReadByte()
		for {
			next, err = p.r.PeekByte()
			if err != nil {
				return "", false
			}

			if next != '^' {
				break
			}

			str = append(str, next)
			_, _ = p.r.ReadByte()
		}
	}

	// NamePath := NameSeg | DualNamePath | MultiNamePath | NullName
	next, err = p.r.ReadByte()
	if err != nil {
		return "", false
	}
	var readCount int
	switch next {
	case 0x00: // NullName
	case 0x2e: // DualNamePath := DualNamePrefix NameSeg NameSeg
		readCount = 8 // NameSeg x 2
	case 0x2f: // MultiNamePath := MultiNamePrefix SegCount NameSeg(SegCount)
		segCount, err := p.r.ReadByte()
		if segCount == 0 || err != nil {
			return "", false
		}

		readCount = int(segCount) * 4
	default: // NameSeg := LeadNameChar NameChar NameChar NameChar
		// LeadNameChar := 'A' - 'Z' | '_'
		if (next < 'A' || next > 'Z') && next != '_' {
			return "", false
		}

		str = append(str, next) // LeadNameChar
		readCount = 3           // NameChar x 3
	}

	for index := 0; readCount > 0; readCount, index = readCount-1, index+1 {
		next, err := p.r.ReadByte()
		if err != nil {
			return "", false
		}

		// Inject a '.' every 4 chars except for the last segment so
		// scoped lookups can work properly.
		if index > 0 && index%4 == 0 && readCount > 1 {
			str = append(str, '.')
		}

		str = append(str, next)
	}

	return string(str), true
}

// scopeCurrent returns the currently active scope.
func (p *Parser) scopeCurrent() entity.Container {
	return p.scopeStack[len(p.scopeStack)-1]
}

// scopeEnter enters the given scope.
func (p *Parser) scopeEnter(s entity.Container) {
	p.scopeStack = append(p.scopeStack, s)
}

// scopeExit exits the current scope.
func (p *Parser) scopeExit() {
	p.scopeStack = p.scopeStack[:len(p.scopeStack)-1]
}
