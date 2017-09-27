package aml

import (
	"gopheros/kernel/kfmt"
	"io"
)

type resolver interface {
	Resolve(io.Writer, ScopeEntity) bool
}

// Entity is an interface implemented by all AML entities.
type Entity interface {
	Name() string
	Parent() ScopeEntity
	TableHandle() uint8

	setTableHandle(uint8)
	getOpcode() opcode
	setOpcode(opcode)
	setParent(ScopeEntity)
	getArgs() []interface{}
	setArg(uint8, interface{}) bool
}

// ScopeEntity is an interface that is implemented by entities that define an
// AML scope.
type ScopeEntity interface {
	Entity

	Children() []Entity
	Append(Entity) bool

	removeChild(Entity)
	lastChild() Entity
}

// unnamedEntity defines an unnamed entity that can be attached to a parent scope.
type unnamedEntity struct {
	tableHandle uint8
	op          opcode
	args        []interface{}
	parent      ScopeEntity
}

func (ent *unnamedEntity) getOpcode() opcode            { return ent.op }
func (ent *unnamedEntity) setOpcode(op opcode)          { ent.op = op }
func (ent *unnamedEntity) Name() string                 { return "" }
func (ent *unnamedEntity) Parent() ScopeEntity          { return ent.parent }
func (ent *unnamedEntity) setParent(parent ScopeEntity) { ent.parent = parent }
func (ent *unnamedEntity) getArgs() []interface{}       { return ent.args }
func (ent *unnamedEntity) setArg(_ uint8, arg interface{}) bool {
	ent.args = append(ent.args, arg)
	return true
}
func (ent *unnamedEntity) TableHandle() uint8     { return ent.tableHandle }
func (ent *unnamedEntity) setTableHandle(h uint8) { ent.tableHandle = h }

// namedEntity is a named entity that can be attached to the parent scope. The
// setArg() implementation for this type expects arg at index 0 to contain the
// entity name.
type namedEntity struct {
	tableHandle uint8
	op          opcode
	args        []interface{}
	parent      ScopeEntity

	name string
}

func (ent *namedEntity) getOpcode() opcode            { return ent.op }
func (ent *namedEntity) setOpcode(op opcode)          { ent.op = op }
func (ent *namedEntity) Name() string                 { return ent.name }
func (ent *namedEntity) Parent() ScopeEntity          { return ent.parent }
func (ent *namedEntity) setParent(parent ScopeEntity) { ent.parent = parent }
func (ent *namedEntity) getArgs() []interface{}       { return ent.args }
func (ent *namedEntity) setArg(argIndex uint8, arg interface{}) bool {
	// arg 0 is the entity name
	if argIndex == 0 {
		var ok bool
		ent.name, ok = arg.(string)
		return ok
	}

	ent.args = append(ent.args, arg)
	return true
}
func (ent *namedEntity) TableHandle() uint8     { return ent.tableHandle }
func (ent *namedEntity) setTableHandle(h uint8) { ent.tableHandle = h }

// constEntity is an unnamedEntity which always evaluates to a constant value.
// Calls to setArg for argument index 0 will memoize the argument value that is
// stored inside this entity.
type constEntity struct {
	tableHandle uint8
	op          opcode
	args        []interface{}
	parent      ScopeEntity

	val interface{}
}

func (ent *constEntity) getOpcode() opcode { return ent.op }
func (ent *constEntity) setOpcode(op opcode) {
	ent.op = op

	// special const opcode cases that have an implicit value
	switch ent.op {
	case opZero:
		ent.val = uint64(0)
	case opOne:
		ent.val = uint64(1)
	case opOnes:
		ent.val = uint64(1<<64 - 1)
	}
}
func (ent *constEntity) Name() string                 { return "" }
func (ent *constEntity) Parent() ScopeEntity          { return ent.parent }
func (ent *constEntity) setParent(parent ScopeEntity) { ent.parent = parent }
func (ent *constEntity) getArgs() []interface{}       { return ent.args }
func (ent *constEntity) setArg(argIndex uint8, arg interface{}) bool {
	ent.val = arg
	return argIndex == 0
}
func (ent *constEntity) TableHandle() uint8     { return ent.tableHandle }
func (ent *constEntity) setTableHandle(h uint8) { ent.tableHandle = h }

// scopeEntity is an optionally named entity that defines a scope.
type scopeEntity struct {
	tableHandle uint8
	op          opcode
	args        []interface{}
	parent      ScopeEntity

	name     string
	children []Entity
}

func (ent *scopeEntity) getOpcode() opcode            { return ent.op }
func (ent *scopeEntity) setOpcode(op opcode)          { ent.op = op }
func (ent *scopeEntity) Name() string                 { return ent.name }
func (ent *scopeEntity) Parent() ScopeEntity          { return ent.parent }
func (ent *scopeEntity) setParent(parent ScopeEntity) { ent.parent = parent }
func (ent *scopeEntity) getArgs() []interface{}       { return ent.args }
func (ent *scopeEntity) setArg(argIndex uint8, arg interface{}) bool {
	// arg 0 *may* be the entity name. If it's not a string just add it to
	// the arg list.
	if argIndex == 0 {
		var ok bool
		if ent.name, ok = arg.(string); ok {
			return true
		}
	}

	ent.args = append(ent.args, arg)
	return true
}
func (ent *scopeEntity) Children() []Entity { return ent.children }
func (ent *scopeEntity) Append(child Entity) bool {
	child.setParent(ent)
	ent.children = append(ent.children, child)
	return true
}
func (ent *scopeEntity) lastChild() Entity { return ent.children[len(ent.children)-1] }
func (ent *scopeEntity) removeChild(child Entity) {
	for index := 0; index < len(ent.children); index++ {
		if ent.children[index] == child {
			ent.children = append(ent.children[:index], ent.children[index+1:]...)
			return
		}
	}
}
func (ent *scopeEntity) TableHandle() uint8     { return ent.tableHandle }
func (ent *scopeEntity) setTableHandle(h uint8) { ent.tableHandle = h }

// bufferEntity defines a buffer object.
type bufferEntity struct {
	unnamedEntity

	size interface{}
	data []byte
}

func (ent *bufferEntity) setArg(argIndex uint8, arg interface{}) bool {
	switch argIndex {
	case 0: // size
		ent.size = arg
		return true
	case 1: // data
		if byteSlice, ok := arg.([]byte); ok {
			ent.data = byteSlice
			return true
		}
	}

	return false
}

// bufferFieldEntity describes a bit/byte/word/dword/qword or arbitrary length
// buffer field.
type bufferFieldEntity struct {
	namedEntity
}

func (ent *bufferFieldEntity) setArg(argIndex uint8, arg interface{}) bool {
	// opCreateField specifies the name using the arg at index 3 while
	// opCreateXXXField (byte, word e.t.c) specifies the name using the
	// arg at index 2
	if (ent.op == opCreateField && argIndex == 3) || argIndex == 2 {
		var ok bool
		ent.name, ok = arg.(string)
		return ok
	}
	ent.args = append(ent.args, arg)
	return true
}

// RegionSpace describes the memory space where a region is located.
type RegionSpace uint8

// The list of supported RegionSpace values.
const (
	RegionSpaceSystemMemory RegionSpace = iota
	RegionSpaceSystemIO
	RegionSpacePCIConfig
	RegionSpaceEmbeddedControl
	RegionSpaceSMBus
	RegionSpacePCIBarTarget
	RegionSpaceIPMI
)

// regionEntity defines a region located at a particular space (e.g in memory,
// an embedded controller, the SMBus e.t.c).
type regionEntity struct {
	namedEntity

	space RegionSpace
}

func (ent *regionEntity) setArg(argIndex uint8, arg interface{}) bool {
	var ok bool
	switch argIndex {
	case 0:
		ok = ent.namedEntity.setArg(argIndex, arg)
	case 1:
		// the parser will convert ByteData types to uint64
		var space uint64
		space, ok = arg.(uint64)
		ent.space = RegionSpace(space)
	case 2, 3:
		ent.args = append(ent.args, arg)
		ok = true
	}

	return ok
}

// FieldAccessType specifies the type of access (byte, word, e.t.c) used to
// read/write to a field.
type FieldAccessType uint8

// The list of supported FieldAccessType values.
const (
	FieldAccessTypeAny FieldAccessType = iota
	FieldAccessTypeByte
	FieldAccessTypeWord
	FieldAccessTypeDword
	FieldAccessTypeQword
	FieldAccessTypeBuffer
)

// FieldUpdateRule specifies how a field value is updated when a write uses
// a value with a smaller width than the field.
type FieldUpdateRule uint8

// The list of supported FieldUpdateRule values.
const (
	FieldUpdateRulePreserve FieldUpdateRule = iota
	FieldUpdateRuleWriteAsOnes
	FieldUpdateRuleWriteAsZeros
)

// FieldAccessAttrib specifies additional information about a particular field
// access.
type FieldAccessAttrib uint8

// The list of supported FieldAccessAttrib values.
const (
	FieldAccessAttribQuick            FieldAccessAttrib = 0x02
	FieldAccessAttribSendReceive                        = 0x04
	FieldAccessAttribByte                               = 0x06
	FieldAccessAttribWord                               = 0x08
	FieldAccessAttribBlock                              = 0x0a
	FieldAccessAttribBytes                              = 0x0b // byteCount contains the number of bytes
	FieldAccessAttribProcessCall                        = 0x0c
	FieldAccessAttribBlockProcessCall                   = 0x0d
	FieldAccessAttribRawBytes                           = 0x0e // byteCount contains the number of bytes
	FieldAccessAttribRawProcessBytes                    = 0x0f // byteCount contains the number of bytes
)

// fieldEntity is a named object that encapsulates the data shared between regular
// fields and index fields.
type fieldEntity struct {
	namedEntity

	bitOffset uint32
	bitWidth  uint32

	lock       bool
	updateRule FieldUpdateRule

	// accessAttrib is valid if accessType is BufferAcc
	// for the SMB or GPIO OpRegions.
	accessAttrib FieldAccessAttrib
	accessType   FieldAccessType

	// byteCount is valid when accessAttrib is one of:
	// Bytes, RawBytes or RawProcessBytes
	byteCount uint8
}

// fieldUnitEntity is a field defined inside an operating region.
type fieldUnitEntity struct {
	fieldEntity

	// The connection which this field references.
	connectionName     string
	resolvedConnection Entity

	// The region which this field references.
	regionName     string
	resolvedRegion *regionEntity
}

func (ent *fieldUnitEntity) Resolve(errWriter io.Writer, rootNs ScopeEntity) bool {
	var ok bool
	if ent.connectionName != "" && ent.resolvedConnection == nil {
		if ent.resolvedConnection = scopeFind(ent.parent, rootNs, ent.connectionName); ent.resolvedConnection == nil {
			kfmt.Fprintf(errWriter, "[field %s] could not resolve connection reference: %s\n", ent.name, ent.connectionName)
			return false
		}
	}

	if ent.resolvedRegion == nil {
		if ent.resolvedRegion, ok = scopeFind(ent.parent, rootNs, ent.regionName).(*regionEntity); !ok {
			kfmt.Fprintf(errWriter, "[field %s] could not resolve referenced region: %s\n", ent.name, ent.regionName)
		}
	}

	return ent.resolvedRegion != nil
}

// indexFieldEntity is a special field that groups together two field units so a
// index/data register pattern can be implemented. To write a value to an
// indexField, the interpreter must first write the appropriate offset to
// the indexRegister (using the alignment specifid by accessType) and then
// write the actual value to the dataRegister.
type indexFieldEntity struct {
	fieldEntity

	// The connection which this field references.
	connectionName     string
	resolvedConnection Entity

	indexRegName string
	indexReg     *fieldUnitEntity

	dataRegName string
	dataReg     *fieldUnitEntity
}

func (ent *indexFieldEntity) Resolve(errWriter io.Writer, rootNs ScopeEntity) bool {
	var ok bool
	if ent.connectionName != "" && ent.resolvedConnection == nil {
		if ent.resolvedConnection = scopeFind(ent.parent, rootNs, ent.connectionName); ent.resolvedConnection == nil {
			kfmt.Fprintf(errWriter, "[field %s] could not resolve connection reference: %s\n", ent.name, ent.connectionName)
			return false
		}
	}

	if ent.indexReg == nil {
		if ent.indexReg, ok = scopeFind(ent.parent, rootNs, ent.indexRegName).(*fieldUnitEntity); !ok {
			kfmt.Fprintf(errWriter, "[indexField %s] could not resolve referenced index register: %s\n", ent.name, ent.indexRegName)
		}
	}

	if ent.dataReg == nil {
		if ent.dataReg, ok = scopeFind(ent.parent, rootNs, ent.dataRegName).(*fieldUnitEntity); !ok {
			kfmt.Fprintf(errWriter, "[dataField %s] could not resolve referenced data register: %s\n", ent.name, ent.dataRegName)
		}
	}

	return ent.indexReg != nil && ent.dataReg != nil
}

// namedReference holds a named reference to an AML symbol. The spec allows
// the symbol not to be defined at the time when the reference is parsed. In
// such a case (forward reference) it will be resolved after the entire AML
// stream has successfully been parsed.
type namedReference struct {
	unnamedEntity

	targetName string
	target     Entity
}

func (ref *namedReference) Resolve(errWriter io.Writer, rootNs ScopeEntity) bool {
	if ref.target = scopeFind(ref.parent, rootNs, ref.targetName); ref.target == nil {
		kfmt.Fprintf(errWriter, "could not resolve referenced symbol: %s (parent: %s)\n", ref.targetName, ref.parent.Name())
		return false
	}

	return true
}

// methodInvocationEntity describes an AML method invocation.
type methodInvocationEntity struct {
	unnamedEntity

	methodDef *Method
}

// Method defines an invocable AML method.
type Method struct {
	scopeEntity

	tableHandle uint8
	argCount    uint8
	serialized  bool
	syncLevel   uint8
}

func (m *Method) getOpcode() opcode { return opMethod }

// Device defines a device.
type Device struct {
	scopeEntity

	tableHandle uint8

	// The methodMap keeps track of all methods exposed by this device.
	methodMap map[string]*Method
}

func (d *Device) getOpcode() opcode      { return opDevice }
func (d *Device) setTableHandle(h uint8) { d.tableHandle = h }

// TableHandle returns the handle of the ACPI table that defines this device.
func (d *Device) TableHandle() uint8 { return d.tableHandle }

// mutexEntity represents a named mutex object
type mutexEntity struct {
	parent ScopeEntity

	// isGlobal is set to true for the pre-defined global mutex (\_GL object)
	isGlobal bool

	name        string
	syncLevel   uint8
	tableHandle uint8
}

func (ent *mutexEntity) getOpcode() opcode            { return opMutex }
func (ent *mutexEntity) setOpcode(op opcode)          {}
func (ent *mutexEntity) Name() string                 { return ent.name }
func (ent *mutexEntity) Parent() ScopeEntity          { return ent.parent }
func (ent *mutexEntity) setParent(parent ScopeEntity) { ent.parent = parent }
func (ent *mutexEntity) getArgs() []interface{}       { return nil }
func (ent *mutexEntity) setArg(argIndex uint8, arg interface{}) bool {
	// arg 0 is the mutex name
	if argIndex == 0 {
		var ok bool
		ent.name, ok = arg.(string)
		return ok
	}

	// arg1 is the sync level (bits 0:3)
	syncLevel, ok := arg.(uint64)
	ent.syncLevel = uint8(syncLevel) & 0xf
	return ok
}
func (ent *mutexEntity) TableHandle() uint8     { return ent.tableHandle }
func (ent *mutexEntity) setTableHandle(h uint8) { ent.tableHandle = h }

// eventEntity represents a named ACPI sync event.
type eventEntity struct {
	namedEntity
}
