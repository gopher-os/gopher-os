package vm

// This is the list of opcodes used by the AML virtual machine. The 2 MSB of
// each opcode indicates the number of operands that need to be decoded by the
// VM:
// - 00 : no operands
// - 01 : 2 byte operands (word)
// - 10 : 4 byte operands (dword)
//
// The length of a particular instruction in bytes (including its opcode)
// can be found via the instrEncodingLen function.
//
// | mnemonic   | opcode | operands [count]: [operand labels]                    | stack before     | stack after                      | description                                                                                                  |
// |------------|--------|-------------------------------------------------------|------------------|----------------------------------|--------------------------------------------------------------------------------------------------------------|
// | nop        | 0x00   |                                                       |                  |                                  | nop                                                                                                          |
const (
	opNop uint8 = 0x00
)

// instrEncodingLen returns the number of bytes that encode the instruction
// specified by opcode op.
func instrEncodingLen(op uint8) int {
	// Instead of extracting the 2 MSB and then multiplying by 2 we extract the 3
	// MSB and clear the right-most bit which yields the same result.
	return 1 + int((op>>5)&0x6)
}
