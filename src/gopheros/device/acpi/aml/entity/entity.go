package entity

import "gopheros/kernel"

// Entity is an interface implemented by all AML entities.
type Entity interface {
	// Opcode returns the AML op associated with this entity.
	Opcode() AMLOpcode

	// Name returns the entity's name or an empty string if no name is
	// associated with the entity.
	Name() string

	// Parent returns the Container of this entity.
	Parent() Container

	// SetParent updates the parent container reference.
	SetParent(Container)

	// TableHandle returns the handle of the ACPI table where this entity
	// was defined.
	TableHandle() uint8

	// Args returns the argument list for this entity.
	Args() []interface{}

	// SetArg adds an argument value at the specified argument index.
	SetArg(uint8, interface{}) bool
}

// Container is an interface that is implemented by entities contain a
// collection of other Entities and define an AML scope.
type Container interface {
	Entity

	// Children returns the list of entities that are children of this
	// container.
	Children() []Entity

	// Append adds an entity to a container.
	Append(Entity) bool

	// Remove searches the child list for an entity and removes it if found.
	Remove(Entity)

	// Last returns the last entity that was added to this container.
	Last() Entity
}

type FieldAccessTypeProvider interface {
	// DefaultAccessType returns the default FieldAccessType for any field unit
	// defined by this field.
	DefaultAccessType() FieldAccessType
}

// LazyRefResolver is an interface implemented by entities that contain symbol
// references that are lazily resolved after the full AML entity tree has been
// parsed.
type LazyRefResolver interface {
	// ResolveSymbolRefs receives as input the root of the AML entity tree and
	// attempts to resolve any symbol references using the scope searching rules
	// defined by the ACPI spec.
	ResolveSymbolRefs(Container) *kernel.Error
}

// Generic describes an entity without a name.
type Generic struct {
	_           uint8
	tableHandle uint8
	op          AMLOpcode
	args        []interface{}
	parent      Container
}

// NewGeneric returns a new generic AML entity.
func NewGeneric(op AMLOpcode, tableHandle uint8) *Generic {
	return &Generic{
		op:          op,
		tableHandle: tableHandle,
	}
}

// Opcode returns the AML op associated with this entity.
func (ent *Generic) Opcode() AMLOpcode { return ent.op }

// Name returns the entity's name. For this type of entity it always returns
// an empty string.
func (ent *Generic) Name() string { return "" }

// Parent returns the Container of this entity.
func (ent *Generic) Parent() Container { return ent.parent }

// SetParent updates the parent container reference.
func (ent *Generic) SetParent(parent Container) { ent.parent = parent }

// TableHandle returns the handle of the ACPI table where this entity was
// defined.
func (ent *Generic) TableHandle() uint8 { return ent.tableHandle }

// Args returns the argument list for this entity.
func (ent *Generic) Args() []interface{} { return ent.args }

// SetArg adds an argument value at the specified argument index.
func (ent *Generic) SetArg(_ uint8, arg interface{}) bool {
	ent.args = append(ent.args, arg)
	return true
}

// GenericNamed describes an entity whose name is specified as the argument at
// index zero.
type GenericNamed struct {
	Generic
	name string
}

// NewGenericNamed returns a new generic named AML entity.
func NewGenericNamed(op AMLOpcode, tableHandle uint8) *GenericNamed {
	return &GenericNamed{
		Generic: Generic{
			op:          op,
			tableHandle: tableHandle,
		},
	}
}

// Name returns the entity's name.
func (ent *GenericNamed) Name() string { return ent.name }

// SetArg adds an argument value at the specified argument index.
func (ent *GenericNamed) SetArg(argIndex uint8, arg interface{}) bool {
	// arg 0 is the entity name
	if argIndex == 0 {
		var ok bool
		ent.name, ok = arg.(string)
		return ok
	}

	ent.args = append(ent.args, arg)
	return true
}

// Const is an optionally named entity that contains a constant uint64 or
// string value.
type Const struct {
	GenericNamed
	Value interface{}
}

// NewConst creates a new AML constant entity.
func NewConst(op AMLOpcode, tableHandle uint8, initialValue interface{}) *Const {
	return &Const{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          op,
				tableHandle: tableHandle,
			},
		},
		Value: initialValue,
	}
}

// SetName allows the caller to override the name for a Const entity.
func (ent *Const) SetName(name string) { ent.name = name }

// SetArg adds an argument value at the specified argument index.
func (ent *Const) SetArg(argIndex uint8, arg interface{}) bool {
	// Const entities accept at most one arg
	ent.Value = arg
	return argIndex == 0
}

// Scope is an optionally named entity that groups together multiple entities.
type Scope struct {
	GenericNamed
	children []Entity
}

// NewScope creates a new AML named scope entity.
func NewScope(op AMLOpcode, tableHandle uint8, name string) *Scope {
	return &Scope{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          op,
				tableHandle: tableHandle,
			},
			name: name,
		},
	}
}

// Children returns the list of entities that are children of this container.
func (ent *Scope) Children() []Entity { return ent.children }

// Append adds an entity to a container.
func (ent *Scope) Append(child Entity) bool {
	child.SetParent(ent)
	ent.children = append(ent.children, child)
	return true
}

// Remove searches the child list for an entity and removes it if found.
func (ent *Scope) Remove(child Entity) {
	for index := 0; index < len(ent.children); index++ {
		if ent.children[index] == child {
			ent.children = append(ent.children[:index], ent.children[index+1:]...)
			return
		}
	}
}

// Last returns the last entity that was added to this container.
func (ent *Scope) Last() Entity { return ent.children[len(ent.children)-1] }

// Buffer defines an AML buffer entity. The entity fields specify a size (arg
// 0) and an optional initializer.
type Buffer struct {
	Generic

	size interface{}
	data []byte
}

// NewBuffer creates a new AML buffer entity.
func NewBuffer(tableHandle uint8) *Buffer {
	return &Buffer{
		Generic: Generic{
			op:          OpBuffer,
			tableHandle: tableHandle,
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Buffer) SetArg(argIndex uint8, arg interface{}) bool {
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

// BufferField describes a bit/byte/word/dword/qword or arbitrary length
// region within a Buffer.
type BufferField struct {
	GenericNamed

	SourceBuf interface{}
	Index     interface{}
	NumBits   interface{}
}

// NewBufferField creates a new AML buffer field entity.
func NewBufferField(op AMLOpcode, tableHandle uint8, bits uint64) *BufferField {
	return &BufferField{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          op,
				tableHandle: tableHandle,
			},
		},
		NumBits: bits,
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *BufferField) SetArg(argIndex uint8, arg interface{}) bool {
	switch argIndex {
	case 0:
		ent.SourceBuf = arg
	case 1:
		ent.Index = arg
	case 2, 3:
		// opCreateField specifies the name using the arg at index 3
		// while opCreateXXXField (byte, word e.t.c) specifies the name
		// using the arg at index 2
		var ok bool
		if ent.name, ok = arg.(string); !ok {
			ent.NumBits = arg
		}
	}
	return argIndex <= 3
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

// Region defines a region located at a particular space (e.g in memory, an
// embedded controller, the SMBus e.t.c).
type Region struct {
	GenericNamed

	Space  RegionSpace
	Offset interface{}
	Len    interface{}
}

// NewRegion creates a new AML region entity.
func NewRegion(tableHandle uint8) *Region {
	return &Region{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          OpOpRegion,
				tableHandle: tableHandle,
			},
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Region) SetArg(argIndex uint8, arg interface{}) bool {
	var ok bool
	switch argIndex {
	case 0:
		ok = ent.GenericNamed.SetArg(argIndex, arg)
	case 1:
		// the parser will convert ByteData types to uint64
		var space uint64
		space, ok = arg.(uint64)
		ent.Space = RegionSpace(space)
	case 2:
		ent.Offset = arg
		ok = true
	case 3:
		ent.Len = arg
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

// FieldLockRule specifies what type of locking is required when accesing field.
type FieldLockRule uint8

// The list of supported FieldLockRule values.
const (
	FieldLockRuleNoLock FieldLockRule = iota
	FieldLockRuleLock
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

// Field is anobject that controls access to a host operating region. It is
// referenced by a list of FieldUnit objects that appear as siblings of a Field
// in the same scope.
type Field struct {
	Generic

	// The region which this field references.
	RegionName string
	Region     *Region

	AccessType FieldAccessType
	LockRule   FieldLockRule
	UpdateRule FieldUpdateRule
}

// NewField creates a new AML field entity.
func NewField(tableHandle uint8) *Field {
	return &Field{
		Generic: Generic{
			op:          OpField,
			tableHandle: tableHandle,
		},
	}
}

// DefaultAccessType returns the default FieldAccessType for any field unit
// defined by this field.
func (ent *Field) DefaultAccessType() FieldAccessType {
	return ent.AccessType
}

// SetArg adds an argument value at the specified argument index.
func (ent *Field) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		ent.RegionName, ok = arg.(string)
	case 1:
		uintVal, ok = arg.(uint64)

		ent.AccessType = FieldAccessType(uintVal & 0xf)        // access type; bits[0:3]
		ent.LockRule = FieldLockRule((uintVal >> 4) & 0x1)     // lock; bit 4
		ent.UpdateRule = FieldUpdateRule((uintVal >> 5) & 0x3) // update rule; bits[5:6]
	}

	return ok
}

// IndexField is a special field that groups together two field units so a
// index/data register pattern can be implemented. To write a value to an
// IndexField, the interpreter must first write the appropriate offset to
// the IndexRegister (using the alignment specifid by AccessType) and then
// write the actual value to the DataRegister.
type IndexField struct {
	Generic

	IndexRegName string
	IndexReg     *FieldUnit

	DataRegName string
	DataReg     *FieldUnit

	AccessType FieldAccessType
	LockRule   FieldLockRule
	UpdateRule FieldUpdateRule
}

// NewIndexField creates a new AML index field entity.
func NewIndexField(tableHandle uint8) *IndexField {
	return &IndexField{
		Generic: Generic{
			op:          OpIndexField,
			tableHandle: tableHandle,
		},
	}
}

// DefaultAccessType returns the default FieldAccessType for any field unit
// defined by this field.
func (ent *IndexField) DefaultAccessType() FieldAccessType {
	return ent.AccessType
}

// SetArg adds an argument value at the specified argument index.
func (ent *IndexField) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		ent.IndexRegName, ok = arg.(string)
	case 1:
		ent.DataRegName, ok = arg.(string)
	case 2:
		uintVal, ok = arg.(uint64)

		ent.AccessType = FieldAccessType(uintVal & 0xf)        // access type; bits[0:3]
		ent.LockRule = FieldLockRule((uintVal >> 4) & 0x1)     // lock; bit 4
		ent.UpdateRule = FieldUpdateRule((uintVal >> 5) & 0x3) // update rule; bits[5:6]
	}
	return ok
}

// BankField is a special field where a bank register must be used to select
// the appropriate bank region before accessing its contents.
type BankField struct {
	Generic

	// The region which this field references.
	RegionName string
	Region     *Region

	// The bank name which controls access to field units defined within this field.
	BankFieldUnitName string
	BankFieldUnit     *FieldUnit

	// The value that needs to be written to the bank field before accessing any field unit.
	BankFieldUnitValue interface{}

	AccessType FieldAccessType
	LockRule   FieldLockRule
	UpdateRule FieldUpdateRule
}

// NewBankField creates a new AML bank field entity.
func NewBankField(tableHandle uint8) *BankField {
	return &BankField{
		Generic: Generic{
			op:          OpBankField,
			tableHandle: tableHandle,
		},
	}
}

// DefaultAccessType returns the default FieldAccessType for any field unit
// defined by this field.
func (ent *BankField) DefaultAccessType() FieldAccessType {
	return ent.AccessType
}

// SetArg adds an argument value at the specified argument index.
func (ent *BankField) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		ent.RegionName, ok = arg.(string)
	case 1:
		ent.BankFieldUnitName, ok = arg.(string)
	case 2:
		ent.BankFieldUnitValue, ok = arg, true
	case 3:
		uintVal, ok = arg.(uint64)

		ent.AccessType = FieldAccessType(uintVal & 0xf)        // access type; bits[0:3]
		ent.LockRule = FieldLockRule((uintVal >> 4) & 0x1)     // lock; bit 4
		ent.UpdateRule = FieldUpdateRule((uintVal >> 5) & 0x3) // update rule; bits[5:6]
	}
	return ok
}

// FieldUnit describes a sub-region inside a parent field.
type FieldUnit struct {
	GenericNamed

	// Depending on what field this unit belongs to this will be a pointer
	// to one of: Field, BankField, IndexField
	Field interface{}

	// The access type to use. Inherited by parent field unless explicitly
	// changed via a directive in the field unit definition list.
	AccessType FieldAccessType

	// AccessAttrib is valid if AccessType is BufferAcc for the SMB or GPIO OpRegions.
	AccessAttrib FieldAccessAttrib

	// ByteCount is valid when AccessAttrib is one of: Bytes, RawBytes or RawProcessBytes
	ByteCount uint8

	// Field offset in parent region and its width.
	BitOffset uint32
	BitWidth  uint32

	// The connection resource for field access references (serial bus or GPIO).
	ConnectionName string
	Connection     Entity
}

// NewFieldUnit creates a new field unit entity.
func NewFieldUnit(tableHandle uint8, name string) *FieldUnit {
	return &FieldUnit{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          OpFieldUnit,
				tableHandle: tableHandle,
			},
			name: name,
		},
	}
}

// Reference holds a named reference to an AML symbol. The spec allows the
// symbol not to be defined at the time when the reference is parsed. In such a
// case (forward reference) it will be resolved after the entire AML stream has
// successfully been parsed.
type Reference struct {
	Generic

	TargetName string
	Target     Entity
}

// NewReference creates a new reference to a named entity.
func NewReference(tableHandle uint8, target string) *Reference {
	return &Reference{
		Generic: Generic{
			op:          OpName,
			tableHandle: tableHandle,
		},
		TargetName: target,
	}
}

// Method describes an invocable AML method.
type Method struct {
	Scope

	ArgCount   uint8
	Serialized bool
	SyncLevel  uint8
}

// NewMethod creats a new AML method entity.
func NewMethod(tableHandle uint8, name string) *Method {
	return &Method{
		Scope: Scope{
			GenericNamed: GenericNamed{
				Generic: Generic{
					op:          OpMethod,
					tableHandle: tableHandle,
				},
				name: name,
			},
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Method) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		// Arg0 is the name but it is actually defined when creating the entity
		ok = true
	case 1:
		// arg1 is the method flags
		uintVal, ok = arg.(uint64)

		ent.ArgCount = (uint8(uintVal) & 0x7)           // bits[0:2]
		ent.Serialized = (uint8(uintVal)>>3)&0x1 == 0x1 // bit 3
		ent.SyncLevel = (uint8(uintVal) >> 4) & 0xf     // bits[4:7]

	}
	return ok
}

// Invocation describes an AML method invocation.
type Invocation struct {
	Generic

	MethodName string
	MethodDef  *Method
}

// NewInvocation creates a new method invocation object.
func NewInvocation(tableHandle uint8, name string, args []interface{}) *Invocation {
	return &Invocation{
		Generic: Generic{
			op:          OpMethodInvocation,
			tableHandle: tableHandle,
			args:        args,
		},
		MethodName: name,
	}
}

// Device defines an AML device entity.
type Device struct {
	Scope
}

// NewDevice creates a new device object.
func NewDevice(tableHandle uint8, name string) *Device {
	return &Device{
		Scope: Scope{
			GenericNamed: GenericNamed{
				Generic: Generic{
					op:          OpDevice,
					tableHandle: tableHandle,
				},
				name: name,
			},
		},
	}
}

// Processor describes a AML processor entity. According to the spec, the use
// of processor operators is deprecated and processors should be declared as
// Device entities instead.
type Processor struct {
	Scope

	// A unique ID for this processor.
	ID uint8

	// The length of the processor register block. According to the spec,
	// this field may be zero.
	RegBlockLen uint8

	// The I/O address of the process register block.
	RegBlockAddr uint32
}

// NewProcessor creates a new processor object.
func NewProcessor(tableHandle uint8, name string) *Processor {
	return &Processor{
		Scope: Scope{
			GenericNamed: GenericNamed{
				Generic: Generic{
					op:          OpProcessor,
					tableHandle: tableHandle,
				},
				name: name,
			},
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Processor) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		// Arg0 is the name but it is actually defined when creating the entity
		ok = true
	case 1:
		// arg1 is the processor ID (ByteData)
		uintVal, ok = arg.(uint64)
		ent.ID = uint8(uintVal)
	case 2:
		// arg2 is the processor I/O reg block address (Dword)
		uintVal, ok = arg.(uint64)
		ent.RegBlockAddr = uint32(uintVal)
	case 3:
		// arg3 is the processor I/O reg block address len (ByteData)
		uintVal, ok = arg.(uint64)
		ent.RegBlockLen = uint8(uintVal)
	}
	return ok
}

// PowerResource describes a AML power resource entity.
type PowerResource struct {
	Scope

	// The deepest system sleep level OSPM must maintain to keep this power
	// resource on (0 equates to S0, 1 equates to S1, and so on).
	SystemLevel uint8

	// ResourceOrder provides the system with the order in which Power
	// Resources must be enabled or disabled. Each unique resourceorder
	// value represents a level, and any number of power resources may have
	// the same level. Power Resource levels are enabled from low values to
	// high values and are disabled from high values to low values.
	ResourceOrder uint16
}

// NewPowerResource creates a new power resource object.
func NewPowerResource(tableHandle uint8, name string) *PowerResource {
	return &PowerResource{
		Scope: Scope{
			GenericNamed: GenericNamed{
				Generic: Generic{
					op:          OpPowerRes,
					tableHandle: tableHandle,
				},
				name: name,
			},
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *PowerResource) SetArg(argIndex uint8, arg interface{}) bool {
	var (
		ok      bool
		uintVal uint64
	)

	switch argIndex {
	case 0:
		// Arg0 is the name but it is actually defined when creating the entity
		ok = true
	case 1:
		// arg1 is the system level (ByteData)
		uintVal, ok = arg.(uint64)
		ent.SystemLevel = uint8(uintVal)
	case 2:
		// arg2 is the resource order (WordData)
		uintVal, ok = arg.(uint64)
		ent.ResourceOrder = uint16(uintVal)
	}
	return ok
}

// ThermalZone describes a AML thermal zone entity.
type ThermalZone struct {
	Scope
}

// NewThermalZone creates a new thermal zone object.
func NewThermalZone(tableHandle uint8, name string) *ThermalZone {
	return &ThermalZone{
		Scope: Scope{
			GenericNamed: GenericNamed{
				Generic: Generic{
					op:          OpThermalZone,
					tableHandle: tableHandle,
				},
				name: name,
			},
		},
	}
}

// Mutex describes a AML mutex entity.
type Mutex struct {
	GenericNamed

	// IsGlobal is set to true for the pre-defined global mutex (\_GL object)
	IsGlobal bool

	SyncLevel uint8
}

// NewMutex creates a new mutex object.
func NewMutex(tableHandle uint8) *Mutex {
	return &Mutex{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          OpMutex,
				tableHandle: tableHandle,
			},
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Mutex) SetArg(argIndex uint8, arg interface{}) bool {
	var ok bool
	switch argIndex {
	case 0:
		// arg 0 is the mutex name
		ent.name, ok = arg.(string)
	case 1:
		// arg1 is the sync level (bits 0:3)
		var syncLevel uint64
		syncLevel, ok = arg.(uint64)
		ent.SyncLevel = uint8(syncLevel) & 0xf
	}
	return ok
}

// Event represents a named ACPI sync event.
type Event struct {
	GenericNamed
}

// NewEvent creates a new event object.
func NewEvent(tableHandle uint8) *Event {
	return &Event{
		GenericNamed: GenericNamed{
			Generic: Generic{
				op:          OpEvent,
				tableHandle: tableHandle,
			},
		},
	}
}

// Package is an entity that contains one of the following entity types:
// - constant data objects (int, string, buffer or package)
// - named references to data objects (int, string, buffer, buffer field,
//   field unit or package)
// - named references to non-data objects (device, event, method, mutex, region
//   power resource, processor or thermal zone)
type Package struct {
	Generic

	// The number of elements in the package. In most cases, the package
	// length is known at compile-time and will be emitted as a const
	// value.  However, the standard also allows dynamic definition of
	// package elements (e.g. inside a method). In the latter case (or if
	// the package contains more that 255 elements) this will be a
	// expression that the VM needs to evaluate as an integer value.
	NumElements interface{}
}

// NewPackage creates a new package entity with the OpPackage or the
// OpVarPackage opcodes.
func NewPackage(op AMLOpcode, tableHandle uint8) *Package {
	return &Package{
		Generic: Generic{
			op:          op,
			tableHandle: tableHandle,
		},
	}
}

// SetArg adds an argument value at the specified argument index.
func (ent *Package) SetArg(argIndex uint8, arg interface{}) bool {
	// Package entities define the number of elements as the first arg.
	if argIndex == 0 {
		ent.NumElements = arg
		return true
	}

	return ent.Generic.SetArg(argIndex, arg)
}
