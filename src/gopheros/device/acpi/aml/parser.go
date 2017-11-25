package aml

import (
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

// Parser implements an AML parser.
type Parser struct {
	r           amlStreamReader
	errWriter   io.Writer
	root        ScopeEntity
	scopeStack  []ScopeEntity
	tableName   string
	tableHandle uint8
}

// NewParser returns a new AML parser instance.
func NewParser(errWriter io.Writer, rootEntity ScopeEntity) *Parser {
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

	// Pass 1: decode bytecode and build entitites
	p.scopeStack = nil
	p.scopeEnter(p.root)
	if !p.parseObjList(header.Length) {
		lastOp, _ := p.r.LastByte()
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] error parsing AML bytecode (last op 0x%x)\n", p.tableName, p.r.Offset()-1, lastOp)
		return errParsingAML
	}
	p.scopeExit()

	// Pass 2: resolve forward references
	var resolveFailed bool
	scopeVisit(0, p.root, EntityTypeAny, func(_ int, ent Entity) bool {
		if res, ok := ent.(resolver); ok && !res.Resolve(p.errWriter, p.root) {
			resolveFailed = true
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
	// invocation or a name reference. If neither is the case, we need to
	// rewind the stream and parse a method invocation before giving up.
	curOffset = p.r.Offset()
	if info, ok = p.nextOpcode(); !ok {
		p.r.SetOffset(curOffset)
		return p.parseMethodInvocationOrNameRef()
	}

	hasPkgLen := info.flags.is(opFlagHasPkgLen) || info.argFlags.contains(opArgTermList) || info.argFlags.contains(opArgFieldList)

	if hasPkgLen {
		curOffset = p.r.Offset()
		if pkgLen, ok = p.parsePkgLength(); !ok {
			return false
		}
	}

	// If we encounter a named scope we need to look it up and parse the arg list relative to it
	switch info.op {
	case opScope:
		return p.parseScope(curOffset + pkgLen)
	case opDevice, opMethod:
		return p.parseNamespacedObj(info.op, curOffset+pkgLen)
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
func (p *Parser) finalizeObj(op opcode, obj Entity) bool {
	obj.setTableHandle(p.tableHandle)

	switch op {
	case opElse:
		// If this is an else block we need to append it as an argument to the
		// If block
		// Pop Else block of the current scope
		curScope := p.scopeCurrent()
		curScope.removeChild(curScope.lastChild())
		prevObj := curScope.lastChild()
		if prevObj.getOpcode() != opIf {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] encountered else block without a matching if block\n", p.tableName, p.r.Offset())
			return false
		}

		// If predicate(0) then(1) else(2)
		prevObj.setArg(2, obj)
	case opDevice:
		// Build method map
		dev := obj.(*Device)
		dev.methodMap = make(map[string]*Method)
		scopeVisit(0, dev, EntityTypeMethod, func(_ int, ent Entity) bool {
			method := ent.(*Method)
			dev.methodMap[method.name] = method
			return false
		})
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

	target := scopeFind(p.scopeCurrent(), p.root, name)
	if target == nil {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] undefined scope: %s\n", p.tableName, p.r.Offset(), name)
		return false
	}

	switch target.getOpcode() {
	case opDevice, opProcessor, opThermalZone, opPowerRes:
		// ok
	default:
		// Only allow if this is a named scope
		if target.Name() == "" {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] %s does not refer to a scoped object\n", p.tableName, p.r.Offset(), name)
			return false
		}
	}

	p.scopeEnter(target.(ScopeEntity))
	ok = p.parseObjList(maxReadOffset)
	p.scopeExit()

	return ok
}

// parseNamespacedObj reads a scope target name from the AML bytestream,
// attaches the device or method (depending on the opcode) object to the
// correct parent scope, enters the device scope and parses the object list
// contained in the device definition.
func (p *Parser) parseNamespacedObj(op opcode, maxReadOffset uint32) bool {
	scopeExpr, ok := p.parseNameString()
	if !ok {
		return false
	}

	parent, name := scopeResolvePath(p.scopeCurrent(), p.root, scopeExpr)
	if parent == nil {
		kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] undefined scope target: %s (current scope: %s)\n", p.tableName, p.r.Offset(), scopeExpr, p.scopeCurrent().Name())
		return false
	}

	var obj ScopeEntity
	switch op {
	case opDevice:
		obj = &Device{scopeEntity: scopeEntity{name: name}}
	case opMethod:
		m := &Method{scopeEntity: scopeEntity{name: name}}

		flags, flagOk := p.parseNumConstant(1)
		if !flagOk {
			return false
		}
		m.argCount = (uint8(flags) & 0x7)           // bits[0:2]
		m.serialized = (uint8(flags)>>3)&0x1 == 0x1 // bit 3
		m.syncLevel = (uint8(flags) >> 4) & 0xf     // bits[4:7]

		obj = m
	}

	parent.Append(obj)
	p.scopeEnter(obj)
	ok = p.parseObjList(maxReadOffset)
	p.scopeExit()

	return ok && p.finalizeObj(op, obj)
}

func (p *Parser) parseArg(info *opcodeInfo, obj Entity, argIndex uint8, argType opArgFlag, maxReadOffset uint32) bool {
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
		// If object is a scoped entity enter it's scope before parsing
		// the term list. Otherwise, create an unnamed scope, attach it
		// as the next argument to obj and enter that.
		if s, isScopeEnt := obj.(ScopeEntity); isScopeEnt {
			p.scopeEnter(s)
		} else {
			ns := &scopeEntity{op: opScope}
			p.scopeEnter(ns)
			obj.setArg(argIndex, ns)
		}

		ok = p.parseObjList(maxReadOffset)
		p.scopeExit()
		return ok
	case opArgFieldList:
		return p.parseFieldList(info.op, obj.getArgs(), maxReadOffset)
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

	return obj.setArg(argIndex, arg)
}

func (p *Parser) parseArgObj() (Entity, bool) {
	if ok := p.parseObj(); !ok {
		return nil, false
	}

	curScope := p.scopeCurrent()
	obj := curScope.lastChild()
	curScope.removeChild(obj)
	return obj, true
}

func (p *Parser) makeObjForOpcode(info *opcodeInfo) Entity {
	var obj Entity

	switch {
	case info.op == opOpRegion:
		obj = new(regionEntity)
	case info.op == opBuffer:
		obj = new(bufferEntity)
	case info.op == opMutex:
		obj = new(mutexEntity)
	case info.op == opEvent:
		obj = new(eventEntity)
	case opIsBufferField(info.op):
		obj = new(bufferFieldEntity)
	case info.flags.is(opFlagConstant):
		obj = new(constEntity)
	case info.flags.is(opFlagScoped):
		obj = new(scopeEntity)
	case info.flags.is(opFlagNamed):
		obj = new(namedEntity)
	default:
		obj = new(unnamedEntity)
	}

	obj.setOpcode(info.op)
	return obj
}

// parseMethodInvocationOrNameRef attempts to parse a method invocation and its term
// args. This method first scans the NameString and performs a lookup. If the
// lookup returns a method definition then we consult it to figure out how many
// arguments we need to parse.
//
// Grammar:
// MethodInvocation := NameString TermArgList
// TermArgList = Nothing | TermArg TermArgList
// TermArg = Type2Opcode | DataObject | ArgObj | LocalObj | MethodInvocation
func (p *Parser) parseMethodInvocationOrNameRef() bool {
	invocationStartOffset := p.r.Offset()
	name, ok := p.parseNameString()
	if !ok {
		return false
	}

	// Lookup Name and try matching it to a function definition
	if methodDef, ok := scopeFind(p.scopeCurrent(), p.root, name).(*Method); ok {
		var (
			invocation = &methodInvocationEntity{
				methodDef: methodDef,
			}
			curOffset uint32
			argIndex  uint8
			arg       Entity
		)

		for argIndex < methodDef.argCount && !p.r.EOF() {
			// Peek next opcode
			curOffset = p.r.Offset()
			nextOpcode, ok := p.nextOpcode()
			p.r.SetOffset(curOffset)

			switch {
			case ok && (opIsType2(nextOpcode.op) || opIsArg(nextOpcode.op) || opIsDataObject(nextOpcode.op)):
				arg, ok = p.parseArgObj()
			default:
				// It may be a nested invocation or named ref
				ok = p.parseMethodInvocationOrNameRef()
				if ok {
					arg = p.scopeCurrent().lastChild()
					p.scopeCurrent().removeChild(arg)
				}
			}

			// No more TermArgs to parse
			if !ok {
				p.r.SetOffset(curOffset)
				break
			}

			invocation.setArg(argIndex, arg)
			argIndex++
		}

		if argIndex != methodDef.argCount {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d] argument mismatch (exp: %d, got %d) for invocation of method: %s\n", p.tableName, invocationStartOffset, methodDef.argCount, argIndex, name)
			return false
		}

		p.scopeCurrent().Append(invocation)
		return true
	}

	// This is a name reference; assume it's a forward reference for now
	// and delegate its resolution to a post-parse step.
	p.scopeCurrent().Append(&namedReference{targetName: name})
	return true
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
func (p *Parser) parseFieldList(op opcode, args []interface{}, maxReadOffset uint32) bool {
	var (
		// for fieldUnit, name0 is the region name and name1 is not used;
		// for indexField,
		name0, name1 string
		flags        uint64

		ok              bool
		bitWidth        uint32
		curBitOffset    uint32
		accessAttrib    FieldAccessAttrib
		accessByteCount uint8
		unitName        string
	)

	switch op {
	case opField: // Field := PkgLength Region AccessFlags FieldList
		if len(args) != 2 {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d, opcode 0x%2x] invalid arg count: %d\n", p.tableName, p.r.Offset(), uint32(op), len(args))
			return false
		}

		name0, ok = args[0].(string)
		if !ok {
			return false
		}

		flags, ok = args[1].(uint64)
		if !ok {
			return false
		}
	case opIndexField: // Field := PkgLength IndexFieldName DataFieldName AccessFlags FieldList
		if len(args) != 3 {
			kfmt.Fprintf(p.errWriter, "[table: %s, offset: %d, opcode 0x%2x] invalid arg count: %d\n", p.tableName, p.r.Offset(), uint32(op), len(args))
			return false
		}

		name0, ok = args[0].(string)
		if !ok {
			return false
		}

		name1, ok = args[1].(string)
		if !ok {
			return false
		}

		flags, ok = args[2].(uint64)
		if !ok {
			return false
		}
	}

	// Decode flags
	accessType := FieldAccessType(flags & 0xf)        // access type; bits[0:3]
	lock := (flags>>4)&0x1 == 0x1                     // lock; bit 4
	updateRule := FieldUpdateRule((flags >> 5) & 0x3) // update rule; bits[5:6]

	var (
		connectionName     string
		resolvedConnection Entity
	)

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
			accessType = FieldAccessType(next & 0xf) // access type; bits[0:3]

			attrib, err := p.r.ReadByte()
			if err != nil {
				return false
			}

			// To specify AccessAttribBytes, RawBytes and RawProcessBytes
			// the ASL compiler will emit an ExtendedAccessField opcode.
			accessByteCount = 0
			accessAttrib = FieldAccessAttrib(attrib)

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
			accessType = FieldAccessType(next & 0xf) // access type; bits[0:3]

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
				accessAttrib = FieldAccessAttribBytes
			case 0xe:
				accessAttrib = FieldAccessAttribRawBytes
			case 0x0f:
				accessAttrib = FieldAccessAttribRawProcessBytes
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
			// be visible at the same scope as the Field/IndexField
			switch op {
			case opField:
				p.scopeCurrent().Append(&fieldUnitEntity{
					fieldEntity: fieldEntity{
						namedEntity: namedEntity{
							tableHandle: p.tableHandle,
							op:          op,
							name:        unitName,
						},
						bitOffset:    curBitOffset,
						bitWidth:     bitWidth,
						lock:         lock,
						updateRule:   updateRule,
						accessType:   accessType,
						accessAttrib: accessAttrib,
						byteCount:    accessByteCount,
					},
					connectionName:     connectionName,
					resolvedConnection: resolvedConnection,
					regionName:         name0,
				})
			case opIndexField:
				p.scopeCurrent().Append(&indexFieldEntity{
					fieldEntity: fieldEntity{
						namedEntity: namedEntity{
							tableHandle: p.tableHandle,
							op:          op,
							name:        unitName,
						},
						bitOffset:    curBitOffset,
						bitWidth:     bitWidth,
						lock:         lock,
						updateRule:   updateRule,
						accessType:   accessType,
						accessAttrib: accessAttrib,
						byteCount:    accessByteCount,
					},
					connectionName:     connectionName,
					resolvedConnection: resolvedConnection,
					indexRegName:       name0,
					dataRegName:        name1,
				})
			}

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
	case ok && nextOpcode.op >= opLocal0 && nextOpcode.op <= opLocal7:
		obj, ok = &unnamedEntity{op: nextOpcode.op}, true
	case ok && nextOpcode.op >= opArg0 && nextOpcode.op <= opArg6:
		obj, ok = &unnamedEntity{op: nextOpcode.op}, true
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
		case nextOpcode.op == opZero: // this is actually a NullName
			p.r.SetOffset(curOffset + 1)
			return &constEntity{op: opStringPrefix, val: ""}, true
		case opIsArg(nextOpcode.op) || nextOpcode.op == opRefOf || nextOpcode.op == opDerefOf || nextOpcode.op == opIndex || nextOpcode.op == opDebug: // LocalObj | ArgObj | Type6 | DebugObj
		default:
			// Unexpected opcode
			return nil, false
		}

		// We can use parseObj for parsing
		return p.parseArgObj()
	}

	// In this case, this is either a NameString or a control method invocation.
	if ok := p.parseMethodInvocationOrNameRef(); ok {
		obj := p.scopeCurrent().lastChild()
		p.scopeCurrent().removeChild(obj)
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
func (p *Parser) scopeCurrent() ScopeEntity {
	return p.scopeStack[len(p.scopeStack)-1]
}

// scopeEnter enters the given scope.
func (p *Parser) scopeEnter(s ScopeEntity) {
	p.scopeStack = append(p.scopeStack, s)
}

// scopeExit exits the current scope.
func (p *Parser) scopeExit() {
	p.scopeStack = p.scopeStack[:len(p.scopeStack)-1]
}
