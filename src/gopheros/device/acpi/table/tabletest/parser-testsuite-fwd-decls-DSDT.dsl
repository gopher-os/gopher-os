DefinitionBlock ("parser-testsuite-fwd-decls-DSDT.aml", "DSDT", 2, "GOPHER", "GOPHEROS", 0x00000002)
{
  Method(NST0, 1, NotSerialized)
  {
    // Invoke a method which has not been defined at the time the parser
    // reaches this block (forward declaration)
    Return(NST1(Arg0))
  }

  // The declaration of NST1 in the AML stream occurs after the declaration
  // of NST0 method above.
  Method(NST1, 1, NotSerialized)
  {
    Return(Arg0+42)
  }
}
