DefinitionBlock ("vm-testsuite-DSDT.aml", "DSDT", 2, "GOPHER", "GOPHEROS", 0x00000002)
{
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
}
