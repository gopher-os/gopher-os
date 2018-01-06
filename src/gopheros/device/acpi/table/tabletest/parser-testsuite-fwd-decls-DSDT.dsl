DefinitionBlock ("parser-testsuite-fwd-decls-DSDT.aml", "DSDT", 2, "GOPHER", "GOPHEROS", 0x00000002)
{
  Scope(\_SB){
    Method (NST1, 2, NotSerialized)
    {
      Return ("something")
    }
  }

  Method(NST0, 1, NotSerialized)
  {
    // NST1 is declared after NST0 (forward declaration)
    NST1(Arg0)

    // This version of NST1 is defined above and has a different signature.
    // The parser should be able to resolve it to the correct method and
    // parse the correct number of arguments
    Return(\_SB.NST1(NST1(123), "arg"))
  }

  // The declaration of NST1 in the AML stream occurs after the declaration
  // of NST0 method above.
  Method(NST1, 1, NotSerialized)
  {
    Return(Arg0+42)
  }
}
