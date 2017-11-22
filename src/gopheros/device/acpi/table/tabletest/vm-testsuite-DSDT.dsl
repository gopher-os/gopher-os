DefinitionBlock ("vm-testsuite-DSDT.aml", "DSDT", 2, "GOPHER", "GOPHEROS", 0x00000002)
{
  // This example tests automatic allocation for buffers that specify a size
  // but not a data initializer.
  Scope(\_SB){
      // Buffer gets allocated with a length of 16 bytes
      Name(BUF0, Buffer(16){})

      // Buffer data gets padded with zeroes up to the declared length
      Name(BUF1, Buffer(16){0xde, 0xad, 0xc0, 0xde})

      OperationRegion(TOP1, GenericSerialBus, 0x00, 0x100) // GenericSerialBus device at command offset 0x00
      Field(TOP1, BufferAcc, NoLock, Preserve){
          Connection(I2cSerialBus(0x5b,,100000,, "\\_SB",,,,RawDataBuffer(16){})),
              AccessAs(BufferAcc, AttribBytes(4)),
              FLD2, 8,
      }
  }

  // Arithmetic ops
  Method (AR00, 1, NotSerialized)
  {
    Add(Arg0, 5, Local0)
    Return(Local0)
  }

  Method (ARI0, 1, NotSerialized)
  {
    Return(Arg0 + 5)
  }

  Method (AR01, 1, NotSerialized)
  {
    Subtract(Arg0, 5, Local0)
    Return(Local0)
  }

  Method (ARI1, 1, NotSerialized)
  {
    Return(Arg0 - 5)
  }

  Method (AR02, 1, NotSerialized)
  {
    Multiply(Arg0, 8, Local0)
    Return(Local0)
  }

  Method (ARI2, 1, NotSerialized)
  {
    Return(Arg0*8)
  }

  Method (AR03, 1, NotSerialized)
  {
    Local1 = Arg0
    Local1--
    Return(Local1)
  }

  Method (ARI3, 1, NotSerialized)
  {
    Return(Arg0--)
  }

  Method (AR04, 1, NotSerialized)
  {
    Local2 = Arg0
    Local2++
    Return(Local2)
  }

  Method (ARI4, 1, NotSerialized)
  {
    Return(Arg0++)
  }

  Method (AR05, 1, NotSerialized)
  {
    Mod(Arg0, 10, Local0)
    Return(Local0)
  }

  Method (ARI5, 1, NotSerialized)
  {
    Return(Arg0 % 10)
  }

  Method (AR06, 1, NotSerialized)
  {
    // Local0: remainder
    // Local1: quotient
    Divide(Arg0, 10, Local0, Local1)
    Return(Local0 + Local1)
  }

  Method (ARI6, 1, NotSerialized)
  {
    Return(Arg0 / 10)
  }

  // Bit ops

  Method (BI00, 1, NotSerialized)
  {
    ShiftLeft(Arg0, 3, Local0)
    Return(Local0)
  }

  Method (BI01, 1, NotSerialized)
  {
    ShiftRight(Arg0, 2, Local0)
    Return(Local0)
  }

  Method (BI02, 1, NotSerialized)
  {
    And(Arg0, 0xbadf00d, Local0)
    Return(Local0)
  }

  Method (BI03, 1, NotSerialized)
  {
    Or(Arg0, 8, Local0)
    Return(Local0)
  }

  Method (BI04, 1, NotSerialized)
  {
    Nand(Arg0, 8, Local0)
    Return(Local0)
  }

  Method (BI05, 1, NotSerialized)
  {
    Nor(Arg0, 0x7, Local0)
    Return(Local0)
  }

  Method (BI06, 1, NotSerialized)
  {
    Xor(Arg0, 16, Local0)
    Return(Local0)
  }

  Method (BI07, 1, NotSerialized)
  {
    Not(Arg0, Local0)
    Return(Local0)
  }

  Method (BI08, 1, NotSerialized)
  {
    Return(FindSetLeftBit(Arg0))
  }

  Method (BI09, 1, NotSerialized)
  {
    Return(FindSetRightBit(Arg0))
  }

  // Logic ops
  Method (LO00, 2, NotSerialized)
  {
    Return(Arg0 == Arg1)
  }

  Method (LO01, 2, NotSerialized)
  {
    Return(Arg0 > Arg1)
  }

  Method (LO02, 2, NotSerialized)
  {
    Return(Arg0 >= Arg1)
  }

  Method (LO03, 2, NotSerialized)
  {
    Return(Arg0 != Arg1)
  }

  Method (LO04, 2, NotSerialized)
  {
    Return(Arg0 < Arg1)
  }

  Method (LO05, 2, NotSerialized)
  {
    Return(Arg0 <= Arg1)
  }

  Method (LO06, 2, NotSerialized)
  {
    Return(Arg0 && Arg1)
  }

  Method (LO07, 2, NotSerialized)
  {
    Return(Arg0 || Arg1)
  }

  Method (LO08, 1, NotSerialized)
  {
    Return(!Arg0)
  }
}
