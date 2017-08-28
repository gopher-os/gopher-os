package table

// Resolver is an interface implemented by objects that can lookup an ACPI table
// by its name.
//
// LookupTable attempts to locate a table by name returning back a pointer to
// its standard header or nil if the table could not be found. The resolver
// must make sure that the entire table contents are mapped so they can be
// accessed by the caller.
type Resolver interface {
	LookupTable(string) *SDTHeader
}

// RSDPDescriptor defines the root system descriptor pointer for ACPI 1.0. This
// is used as the entry-point for parsing ACPI data.
type RSDPDescriptor struct {
	// The signature must contain "RSD PTR " (last byte is a space).
	Signature [8]byte

	// A value that when added to the sum of all other bytes contained in
	// this descriptor should result in the value 0.
	Checksum uint8

	OEMID [6]byte

	// ACPI revision number. It is 0 for ACPI1.0 and 2 for versions 2.0 to 6.2.
	Revision uint8

	// Physical address of 32-bit root system descriptor table.
	RSDTAddr uint32
}

// ExtRSDPDescriptor extends RSDPDescriptor with additional fields. It is used
// when RSDPDescriptor.revision > 1.
type ExtRSDPDescriptor struct {
	RSDPDescriptor

	// The size of the 64-bit root system descriptor table.
	Length uint32

	// Physical address of 64-bit root system descriptor table.
	XSDTAddr uint64

	// A value that when added to the sum of all other bytes contained in
	// this descriptor should result in the value 0.
	ExtendedChecksum uint8

	reserved [3]byte
}

// SDTHeader defines the common header for all ACPI-related tables.
type SDTHeader struct {
	// The signature defines the table type.
	Signature [4]byte

	// The length of the table
	Length uint32

	// If this header belongs to a DSDT/SSDT table, the revision is also
	// used to indicate whether the AML VM should treat integers as 32-bits
	// (revision < 2) or 64-bits (revision >= 2).
	Revision uint8

	// A value that when added to the sum of all other bytes in the table
	// should result in the value 0.
	Checksum uint8

	// OEM specific information
	OEMID       [6]byte
	OEMTableID  [8]byte
	OEMRevision uint32

	// Information about the ASL compiler that generated this table
	CreatorID       uint32
	CreatorRevision uint32
}

// AddressSpace defines the location where a set of registers resides.
type AddressSpace uint8

// The list of supported address space types.
const (
	AddressSpaceSysMemory AddressSpace = iota
	AddressSpaceSysIO
	AddressSpacePCI
	AddressSpaceEmbController
	AddressSpaceSMBus
	AddressSpaceFuncFixedHW = 0x7f
)

// GenericAddress specifies a register range located in a particular address
// space.
type GenericAddress struct {
	Space      AddressSpace
	BitWidth   uint8
	BitOffset  uint8
	AccessSize uint8
	Address    uint64
}

// PowerProfileType describes a power profile referenced by the FADT table.
type PowerProfileType uint8

// The list of supported power profile types
const (
	PowerProfileUnspecified PowerProfileType = iota
	PowerProfileDesktop
	PowerProfileMobile
	PowerProfileWorkstation
	PowerProfileEnterpriseServer
	PowerProfileSOHOServer
	PowerProfileAppliancePC
	PowerProfilePerformanceServer
)

// FADT64 contains the 64-bit FADT extensions which are used by ACPI2+
type FADT64 struct {
	FirmwareControl uint64

	Dsdt uint64

	PM1aEventBlock   GenericAddress
	PM1bEventBlock   GenericAddress
	PM1aControlBlock GenericAddress
	PM1bControlBlock GenericAddress
	PM2ControlBlock  GenericAddress
	PMTimerBlock     GenericAddress
	GPE0Block        GenericAddress
	GPE1Block        GenericAddress
}

// FADT (Fixed ACPI Description Table) is an ACPI table containing information
// about fixed register blocks used for power management.
type FADT struct {
	SDTHeader

	FirmwareCtrl uint32
	Dsdt         uint32

	reserved uint8

	PreferredPowerManagementProfile PowerProfileType
	SCIInterrupt                    uint16
	SMICommandPort                  uint32
	AcpiEnable                      uint8
	AcpiDisable                     uint8
	S4BIOSReq                       uint8
	PSTATEControl                   uint8
	PM1aEventBlock                  uint32
	PM1bEventBlock                  uint32
	PM1aControlBlock                uint32
	PM1bControlBlock                uint32
	PM2ControlBlock                 uint32
	PMTimerBlock                    uint32
	GPE0Block                       uint32
	GPE1Block                       uint32
	PM1EventLength                  uint8
	PM1ControlLength                uint8
	PM2ControlLength                uint8
	PMTimerLength                   uint8
	GPE0Length                      uint8
	GPE1Length                      uint8
	GPE1Base                        uint8
	CStateControl                   uint8
	WorstC2Latency                  uint16
	WorstC3Latency                  uint16
	FlushSize                       uint16
	FlushStride                     uint16
	DutyOffset                      uint8
	DutyWidth                       uint8
	DayAlarm                        uint8
	MonthAlarm                      uint8
	Century                         uint8

	// Reserved in ACPI 1.0; used since ACPI 2.0+
	BootArchitectureFlags uint16

	reserved2 uint8
	Flags     uint32

	ResetReg GenericAddress

	ResetValue uint8
	reserved3  [3]uint8

	// 64-bit pointers to the above structures used by ACPI 2.0+
	Ext FADT64
}

// MADT (Multiple APIC Description Table) is an ACPI table containing
// information about the interrupt controllers and the number of installed
// CPUs. Following the table header are a series of variable sized records
// (MADTEntry) which contain additional information.
type MADT struct {
	SDTHeader

	LocalControllerAddress uint32
	Flags                  uint32
}

// MADTEntryLocalAPIC describes a single physical processor and its local
// interrupt controller.
type MADTEntryLocalAPIC struct {
	ProcessorID uint8
	APICID      uint8
	Flags       uint32
}

// MADTEntryIOAPIC describes an I/O Advanced Programmable Interrupt Controller.
type MADTEntryIOAPIC struct {
	APICID   uint8
	reserved uint8

	// Address contains the address of the controller.
	Address uint32

	// SysInterruptBase defines the first interrupt number that this
	// controller handles.
	SysInterruptBase uint32
}

// MADTEntryInterruptSrcOverride contains the data for an Interrupt Source
// Override.  This mechanism is used to map IRQ sources to global system
// interrupts.
type MADTEntryInterruptSrcOverride struct {
	BusSrc          uint8
	IRQSrc          uint8
	GlobalInterrupt uint32
	Flags           uint16
}

// MADTEntryNMI describes a non-maskable interrupt that we need to set up for
// a single processor or all processors.
type MADTEntryNMI struct {
	// Processor specifies the local APIC that we need to configure for
	// this NMI. If set to 0xff we need to configure all processor APICs.
	Processor uint8

	Flags uint16

	// This value will be either 0 or 1 and specifies which entry in the
	// local vector table of the processor's local APIC we need to setup.
	LINT uint8
}

// MADTEntryType describes the type of a MADT record.
type MADTEntryType uint8

// The list of supported MADT entry types.
const (
	MADTEntryTypeLocalAPIC MADTEntryType = iota
	MADTEntryTypeIOAPIC
	MADTEntryTypeIntSrcOverride
	MADTEntryTypeNMI
)

// MADTEntry describes a MADT table entry that follows the MADT definition. As
// MADT entries are variable sized records, this struct works as a union. The
// consumer of this struct must check the type value before accessing the union
// values.
type MADTEntry struct {
	Type   MADTEntryType
	Length uint8
}
