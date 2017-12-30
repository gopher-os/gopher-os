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
// | push_0     | 0x01   |                                                       |                  | 0                                | push uint64(0)                                                                                               |
// | push_1     | 0x02   |                                                       |                  | 1                                | push uint64(1)                                                                                               |
// | push_ones  | 0x03   |                                                       |                  | MaxUint64                        | push uint64(MaxUint64)                                                                                       |
// | push_l0    | 0x04   |                                                       |                  | &Local0                          | push local0 address                                                                                          |
// | push_l1    | 0x05   |                                                       |                  | &Local1                          | push local1 address                                                                                          |
// | push_l2    | 0x06   |                                                       |                  | &Local2                          | push local2 address                                                                                          |
// | push_l3    | 0x07   |                                                       |                  | &Local3                          | push local3 address                                                                                          |
// | push_l4    | 0x08   |                                                       |                  | &Local4                          | push local4 address                                                                                          |
// | push_l5    | 0x09   |                                                       |                  | &Local5                          | push local5 address                                                                                          |
// | push_l6    | 0x0a   |                                                       |                  | &Local6                          | push local6 address                                                                                          |
// | push_l7    | 0x0b   |                                                       |                  | &Local7                          | push local7 address                                                                                          |
// | push_a0    | 0x08   |                                                       |                  | &Arg0                            | push arg0 address                                                                                            |
// | push_a1    | 0x0d   |                                                       |                  | &Arg1                            | push arg1 address                                                                                            |
// | push_a2    | 0x0e   |                                                       |                  | &Arg2                            | push arg2 address                                                                                            |
// | push_a3    | 0x0f   |                                                       |                  | &Arg3                            | push arg3 address                                                                                            |
// | push_a4    | 0x10   |                                                       |                  | &Arg4                            | push arg4 address                                                                                            |
// | push_a5    | 0x11   |                                                       |                  | &Arg5                            | push arg5 address                                                                                            |
// | push_a6    | 0x12   |                                                       |                  | &Arg6                            | push arg6 address                                                                                            |
// | push_buf   | 0x40   | 2: indexbyte1, indexbyte2                             |                  | entity ptr                       | push entity address for buffer pool index (indexbyte1<<8 + indexbyte2)                                       |
// | push_const | 0x41   | 2: indexbyte1, indexbyte2                             |                  | entity ptr                       | push entity address for const pool index (indexbyte1<<8 + indexbyte2)                                        |
// | push_pkg   | 0x42   | 2: indexbyte1, indexbyte2                             |                  | entity ptr                       | push entity address for package pool index (indexbyte1<<8 + indexbyte2)                                      |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | store_l0   | 0x13   |                                                       | value            |                                  | store value at local0 with no conversion                                                                     |
// | store_l1   | 0x14   |                                                       | value            |                                  | store value at local1 with no conversion                                                                     |
// | store_l2   | 0x15   |                                                       | value            |                                  | store value at local2 with no conversion                                                                     |
// | store_l3   | 0x16   |                                                       | value            |                                  | store value at local3 with no conversion                                                                     |
// | store_l4   | 0x17   |                                                       | value            |                                  | store value at local4 with no conversion                                                                     |
// | store_l5   | 0x18   |                                                       | value            |                                  | store value at local5 with no conversion                                                                     |
// | store_l6   | 0x19   |                                                       | value            |                                  | store value at local6 with no conversion                                                                     |
// | store_l7   | 0x1a   |                                                       | value            |                                  | store value at local7 with no conversion                                                                     |
// | store_a0   | 0x1b   |                                                       | value            |                                  | store value at arg0 with no conversion                                                                       |
// | store_a1   | 0x1c   |                                                       | value            |                                  | store value at arg1 with no conversion                                                                       |
// | store_a2   | 0x1d   |                                                       | value            |                                  | store value at arg2 with no conversion                                                                       |
// | store_a3   | 0x1e   |                                                       | value            |                                  | store value at arg3 with no conversion                                                                       |
// | store_a4   | 0x1f   |                                                       | value            |                                  | store value at arg4 with no conversion                                                                       |
// | store_a5   | 0x20   |                                                       | value            |                                  | store value at arg5 with no conversion                                                                       |
// | store_a6   | 0x21   |                                                       | value            |                                  | store value at arg6 with no conversion                                                                       |
// | store      | 0x22   |                                                       | dst, value       |                                  | store value to dst after applying implicit type conversion                                                   |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | pop        | 0x23   |                                                       | value            |                                  | pop and discard top value of the stack                                                                       |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | jmp        | 0x80   | 4: offsetbyte1, offsetbyte2, offsetbyte3, offsetbyte4 |                  |                                  | jump to abs address (offsetbyte1<<24 + offsetbyte2<<16 + offsetbyte3<<8 + offsetbyte4)                       |
// | je         | 0x81   | 4: offsetbyte1, offsetbyte2, offsetbyte3, offsetbyte4 | value1, value2   |                                  | jump to abs address (offsetbyte1<<24 + offsetbyte2<<16 + offsetbyte3<<8 + offsetbyte4)  if value1 == value2  |
// | jl         | 0x82   | 4: offsetbyte1, offsetbyte2, offsetbyte3, offsetbyte4 | value1, value2   |                                  | jump to abs address (offsetbyte1<<24 + offsetbyte2<<16 + offsetbyte3<<8 + offsetbyte4)  if value1 < value2   |
// | jg         | 0x83   | 4: offsetbyte1, offsetbyte2, offsetbyte3, offsetbyte4 | value1, value2   |                                  | jump to abs address (offsetbyte1<<24 + offsetbyte2<<16 + offsetbyte3<<8 + offsetbyte4)  if  value1 > value2  |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | call       | 0x43   | 2: indexbyte1, indexbyte2                             | value1,...valueN | result                           | call method at method pool index (indexbyte1<<8 + indexbyte2) and replace args on stack with the ret value   |
// | ret_void   | 0x24   |                                                       |                  |                                  | return from method, unwind stack and replace arg list                                                        |
// | ret        | 0x25   |                                                       | value            | value                            | return from method, unwind stack and replace arg list with return value                                      |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | add        | 0x26   |                                                       | value1, value2   | value1 + value2                  | perform integer addition                                                                                     |
// | sub        | 0x27   |                                                       | value1, value2   | value1 - value2                  | perform integer subtraction                                                                                  |
// | mul        | 0x28   |                                                       | value1, value2   | value1 * value2                  | perform integer multiplication                                                                               |
// | div        | 0x29   |                                                       | value1, value2   | value1 / value2, value1 % value2 | perform integer division                                                                                     |
// | mod        | 0x2a   |                                                       | value1, value2   | value1 % value2                  | perform integer module calculation                                                                           |
// |            |        |                                                       |                  |                                  |                                                                                                              |
// | shl        | 0x2b   |                                                       | value1, value2   | value1 << value2                 | shift value1 left                                                                                            |
// | shr        | 0x2c   |                                                       | value1, value2   | value1 >> value2                 | shift value1 right                                                                                           |
// | and        | 0x2d   |                                                       | value1, value2   | value1 & value2                  | bitwise and                                                                                                  |
// | or         | 0x2e   |                                                       | value1, value2   | value1 | value2                  | bitwise or                                                                                                   |
// | nand       | 0x2f   |                                                       | value1, value2   | value1 &^ value2                 | bitwise nand                                                                                                 |
// | nor        | 0x30   |                                                       | value1, value2   | ^(value1 | value2)               | bitwise nor                                                                                                  |
// | xor        | 0x31   |                                                       | value1, value2   | value1 ^ value2                  | bitwise xor                                                                                                  |
// | not        | 0x32   |                                                       | value            | !value % value2                  | bitwise not                                                                                                  |
// | fslb       | 0x33   |                                                       | value            | index of left-most set bit or 0  | find 1-based index of left-most set bit                                                                      |
// | fsrb       | 0x34   |                                                       | value            | index of right-most set bit or 0 | find 1-based index of right-most set bit                                                                     |
const (
	opNop         uint8 = 0x00
	opPushZero    uint8 = 0x01
	opPushOne     uint8 = 0x02
	opPushOnes    uint8 = 0x03
	opPushLocal0  uint8 = 0x04
	opPushLocal1  uint8 = 0x05
	opPushLocal2  uint8 = 0x06
	opPushLocal3  uint8 = 0x07
	opPushLocal4  uint8 = 0x08
	opPushLocal5  uint8 = 0x09
	opPushLocal6  uint8 = 0x0a
	opPushLocal7  uint8 = 0x0b
	opPushArg0    uint8 = 0x0c
	opPushArg1    uint8 = 0x0d
	opPushArg2    uint8 = 0x0e
	opPushArg3    uint8 = 0x0f
	opPushArg4    uint8 = 0x10
	opPushArg5    uint8 = 0x11
	opPushArg6    uint8 = 0x12
	opPushBuffer  uint8 = 0x40
	opPushConst   uint8 = 0x41
	opPushPkg     uint8 = 0x42
	opStoreLocal0 uint8 = 0x13
	opStoreLocal1 uint8 = 0x14
	opStoreLocal2 uint8 = 0x15
	opStoreLocal3 uint8 = 0x16
	opStoreLocal4 uint8 = 0x17
	opStoreLocal5 uint8 = 0x18
	opStoreLocal6 uint8 = 0x19
	opStoreLocal7 uint8 = 0x1a
	opStoreArg0   uint8 = 0x1b
	opStoreArg1   uint8 = 0x1c
	opStoreArg2   uint8 = 0x1d
	opStoreArg3   uint8 = 0x1e
	opStoreArg4   uint8 = 0x1f
	opStoreArg5   uint8 = 0x20
	opStoreArg6   uint8 = 0x21
	opStore       uint8 = 0x22
	opPop         uint8 = 0x23
	opJmp         uint8 = 0x80
	opJe          uint8 = 0x81
	opJl          uint8 = 0x82
	opJg          uint8 = 0x83
	opCall        uint8 = 0x43
	opRetVoid     uint8 = 0x24
	opRet         uint8 = 0x25
	opAdd         uint8 = 0x26
	opSub         uint8 = 0x27
	opMul         uint8 = 0x28
	opDiv         uint8 = 0x29
	opMod         uint8 = 0x2a
	opShl         uint8 = 0x2b
	opShr         uint8 = 0x2c
	opAnd         uint8 = 0x2d
	opOr          uint8 = 0x2e
	opNand        uint8 = 0x2f
	opNor         uint8 = 0x30
	opXor         uint8 = 0x31
	opNot         uint8 = 0x32
	opFindSlb     uint8 = 0x33 // find set left bit
	opFindSrb     uint8 = 0x34 // find set right bit
)

// instrEncodingLen returns the number of bytes that encode the instruction
// specified by opcode op.
func instrEncodingLen(op uint8) int {
	// Instead of extracting the 2 MSB and then multiplying by 2 we extract the 3
	// MSB and clear the right-most bit which yields the same result.
	return 1 + int((op>>5)&0x6)
}
