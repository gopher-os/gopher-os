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

  // Buffer whose size is given by a nested method invocation which is
  // evaluated at load-time
  Method(BLE1, 1, Serialized)
  {
    Return(Arg0 + 1)
  }

  Name(BUFZ, Buffer(BLEN(BLE1(0xff), 0x0f)){0x48, 0x49, 0x21} )

  Method(BLEN, 2, Serialized)
  {
    Return(Arg0 + Arg1 + 12)
  }

  // Load time assignment
  FET0 = 1

  // A buffer whose length is given by a named object
  Name(BUFL, One)
  Name(BUFF, Buffer(BUFL){})

  // Other entity types
  Event(HLO0)
  Mutex(MUT0,1)
  Signal(HLO0)

  // Other executable bits
  Method (EXE0, 1, Serialized)
  {
    // AML interpreter revision
    Local0 = Revision

    // NameString target
    Local1 = SizeOf(GLB1)

    Local0 = "my-handle"
    Load(DBG0, Local0)
    Unload(Local0)

    // CreateXXXField
    CreateBitField(Arg0, 0, WFL0)
    if(Arg0==0){
      Return(WFL0)
    }

    CreateByteField(Arg0, 0, WFL1)
    if(Arg0==1){
      Return(WFL1)
    }

    CreateWordField(Arg0, 0, WFL2)
    if(Arg0==2){
      Return(WFL2)
    }

    CreateDwordField(Arg0, 0, WFL3)
    if(Arg0==3){
      Return(WFL3)
    }

    CreateQwordField(Arg0, 0, WFL4)
    if(Arg0==4){
      Return(WFL4)
    }

    CreateField(Arg0, 0, 13, WFL5)
    if(Arg0==5){
      Return(WFL5)
    }

    // Example from p. 951 of the spec
    Store (
      LoadTable ("OEM1", "MYOEM", "TABLE1", "\\_SB.PCI0","MYD", Package () {0,"\\_SB.PCI0"}),
      Local0
    )

    // Various Assignments
    Local0 = 0xff
    Local0 = 0xffff
    Local0 = 0xffffffff
    Local0 = 0xffffffffffffffff
    Local0 = Zero
    Local0 = One
    Local0 = Ones
    Local0 = "FOO"

    // Conversions
    FromBCD(9, Arg0)
    ToBCD(Arg0, Local1)

    // Debugging and error handling
    Breakpoint
    Debug = "test"
    Fatal(0xf0, 0xdeadc0de, 1)

    // Mutex support
    Acquire(MUT0, 0xffff) // no timeout
    Release(MUT0)

    // Signal/Wait
    Reset(HLO0)
    Wait(HLO0, 0xffff)

    // Get monotonic timer value
    Local0 = Timer

    CopyObject(Local0, Local1)
    Return(ObjectType(Local1))
  }

  // BankField example from p. 910 of the spec
  // Define a 256-byte operational region in SystemIO space and name it GIO0
  OperationRegion (GIO0, SystemIO, 0x125, 0x100)
  Field (GIO0, ByteAcc, NoLock, Preserve) {
    GLB1, 1,
    GLB2, 1,
    Offset (1),            // Move to offset for byte 1
    BNK1, 4
  }

  BankField (GIO0, BNK1, 0, ByteAcc, Lock, WriteAsOnes) {
    Offset (0x30),
    FET0, 1,
    FET1, 1,
  }

  // SMBus fields access types
  OperationRegion(SMBD, SMBus, 0x4200, 0x100)
  Field(SMBD, BufferAcc, NoLock, WriteAsZeros){
    AccessAs(BufferAcc, SMBByte),
    SFL0, 8,
    AccessAs(BufferAcc, SMBWord),
    SFL1, 8,
    AccessAs(BufferAcc, SMBBlock),
    SFL2, 8,
    AccessAs(BufferAcc, SMBQuick),
    SFL3, 8,
    AccessAs(BufferAcc, SMBSendReceive),
    SFL4, 8,
    AccessAs(BufferAcc, SMBProcessCall),
    SFL5, 8,
    AccessAs(BufferAcc, SMBBlockProcessCall),
    SFL6, 8,
    AccessAs(BufferAcc, AttribBytes(0x12)),
    SFL7, 8,
    AccessAs(BufferAcc, AttribRawBytes(0x13)),
    SFL8, 8,
    AccessAs(BufferAcc, AttribRawProcessBytes(0x14)),
    SFL9, 8,
    //
    AccessAs(AnyAcc),
    SF10, 8,
    AccessAs(ByteAcc),
    SF11, 8,
    AccessAs(WordAcc),
    SF12, 8,
    AccessAs(DwordAcc),
    SF13, 8,
    AccessAs(QwordAcc),
    SF14, 8,
  }

  // Data Region
  DataTableRegion (REG0, "FOOF", "BAR", "BAZ")

  // Other resources
  Processor(CPU0, 1, 0x120, 6){}
  PowerResource(PWR0, 0, 0){}
  ThermalZone(TZ0){}

  // Method with various logic/flow changing opcodes
  Method (FLOW, 2, NotSerialized)
  {
    While(Arg0 < Arg1)
    {
      Arg0++

      If( Arg0 < 5 ) {
        Continue
      } Else {
        If ( Arg1 == 0xff ) {
          Break
        } Else {
          Arg0 = Arg0 + 1
        }
      }
    }

    Return(Arg0)
  }

  // Forward declaration to SCP0
  Scope(\_SB) {
    ThermalZone(^THRM){
      Name(DEF0, Ones)
    }
  }

  Scope(\THRM){
      Name(DEF1, Zero)
  }

  Method(\THRM.MTH0, 0, NotSerialized){
    Return(1)
  }
}
