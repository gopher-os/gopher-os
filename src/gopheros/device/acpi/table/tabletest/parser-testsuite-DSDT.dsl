// DSDT-parser-testsuite
//
// This file contains various ASL constructs to ensure that the AML parser
// properly handles all possible ASL opcodes it may encounter. This test file
// is used in addition to the DSDT.aml file obtained by running acpidump inside
// virtualbox.
DefinitionBlock ("parser-testsuite-DSDT.aml", "DSDT", 2, "GOPHER", "GOPHEROS", 0x00000002)
{
    OperationRegion (DBG0, SystemIO, 0x3000, 0x04)
        Field (DBG0, ByteAcc, NoLock, Preserve)
        {
            DHE1,   8
        }

    Device (DRV0)
    {
        Name (_ADR, Ones)

            // named entity containing qword const
            Name (H15F, 0xBADC0FEEDEADC0DE)
            Method (_GTF, 0, NotSerialized)  // _GTF: Get Task File
            {
                Return (H15F)
            }
    }

    // example from p. 268 of ACPI 6.2 spec
    Scope(\_SB){
        OperationRegion(TOP1, GenericSerialBus, 0x00, 0x100) // GenericSerialBus device at command offset 0x00

            Name (SDB0, ResourceTemplate() {})
            Field(TOP1, BufferAcc, NoLock, Preserve){
                Connection(SDB0), // Use the Resource Descriptor defined above
                    AccessAs(BufferAcc, AttribWord),
                    FLD0, 8,
                    FLD1, 8
            }

        Field(TOP1, BufferAcc, NoLock, Preserve){
            Connection(I2cSerialBus(0x5b,,100000,, "\\_SB",,,,RawDataBuffer(){3,9})),
                AccessAs(BufferAcc, AttribBytes(4)),
                FLD2, 8,
                AccessAs(BufferAcc, AttribRawBytes(3)),
                FLD3, 8,
                AccessAs(BufferAcc, AttribRawProcessBytes(2)),
                FLD4, 8
        }
    }

    // Other entity types
    Event(HLO0)
    Mutex(MUT0,1)
    Signal(HLO0)

    // Other executable bits
    Method (EXE0, 1, Serialized)
    {
        Local0 = Revision

        // NameString target
        Local1 = SizeOf(GLB1)

        Local0 = "my-handle"
        Load(DBG0, Local0)
        Unload(Local0)

        // Example from p. 951 of the spec
        Store (
                LoadTable ("OEM1", "MYOEM", "TABLE1", "\\_SB.PCI0","MYD",
                    Package () {0,"\\_SB.PCI0"}
                    ), Local0
              )

        FromBCD(9, Arg0)
        ToBCD(Arg0, Local1)

        Breakpoint
        Debug = "test"
        Fatal(0xf0, 0xdeadc0de, 1)

        Reset(HLO0)

        // Mutex support
        Acquire(MUT0, 0xffff) // no timeout
        Release(MUT0)

        // Signal/Wait
        Wait(HLO0, 0xffff)

        // Get monotonic timer value
        Local0 = Timer

        CopyObject(Local0, Local1)
        Return(ObjectType(Local1))
    }

    // Misc regions

    // BankField example from p. 899 of the spec
    // Define a 256-byte operational region in SystemIO space and name it GIO0
    OperationRegion (GIO0, SystemIO, 0x125, 0x100)
    Field (GIO0, ByteAcc, NoLock, Preserve) {
        GLB1, 1,
        GLB2, 1,
        Offset (1),            // Move to offset for byte 1
        BNK1, 4
    }

    BankField (GIO0, BNK1, 0, ByteAcc, NoLock, Preserve) {
        Offset (0x30),
        FET0, 1,
        FET1, 1
    }

    // Data Region
    DataTableRegion (REG0, "FOOF", "BAR", "BAZ")
}
