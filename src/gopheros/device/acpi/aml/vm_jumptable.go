package aml

// opHandler is a function that implements an AML opcode.
type opHandler func(*execContext, Entity) *Error

// populateJumpTable assigns the functions that implement the various AML
// opcodes to the VM's jump table.
func (vm *VM) populateJumpTable() {
	for i := 0; i < len(vm.jumpTable); i++ {
		vm.jumpTable[i] = opExecNotImplemented
	}

	// Control-flow opcodes
	vm.jumpTable[opReturn] = vmOpReturn
	vm.jumpTable[opBreak] = vmOpBreak
	vm.jumpTable[opContinue] = vmOpContinue
	vm.jumpTable[opWhile] = vmOpWhile
	vm.jumpTable[opIf] = vmOpIf

	// ALU opcodes
	vm.jumpTable[opAdd] = vmOpAdd
	vm.jumpTable[opSubtract] = vmOpSubtract
	vm.jumpTable[opIncrement] = vmOpIncrement
	vm.jumpTable[opDecrement] = vmOpDecrement
	vm.jumpTable[opMultiply] = vmOpMultiply
	vm.jumpTable[opDivide] = vmOpDivide
	vm.jumpTable[opMod] = vmOpMod

	vm.jumpTable[opShiftLeft] = vmOpShiftLeft
	vm.jumpTable[opShiftRight] = vmOpShiftRight
	vm.jumpTable[opAnd] = vmOpBitwiseAnd
	vm.jumpTable[opOr] = vmOpBitwiseOr
	vm.jumpTable[opNand] = vmOpBitwiseNand
	vm.jumpTable[opNor] = vmOpBitwiseNor
	vm.jumpTable[opXor] = vmOpBitwiseXor
	vm.jumpTable[opNot] = vmOpBitwiseNot
	vm.jumpTable[opFindSetLeftBit] = vmOpFindSetLeftBit
	vm.jumpTable[opFindSetRightBit] = vmOpFindSetRightBit

	vm.jumpTable[opLnot] = vmOpLogicalNot
	vm.jumpTable[opLand] = vmOpLogicalAnd
	vm.jumpTable[opLor] = vmOpLogicalOr
	vm.jumpTable[opLEqual] = vmOpLogicalEqual
	vm.jumpTable[opLLess] = vmOpLogicalLess
	vm.jumpTable[opLGreater] = vmOpLogicalGreater

	// Store-related opcodes
	vm.jumpTable[opStore] = vmOpStore

	// Method invocation dispatcher; to avoid clashes with real AML opcodes,
	// method invocations are assigned an opcode with value (lastOpcode + 1)
	vm.jumpTable[numOpcodes] = vmOpMethodInvocation
}

// opExecNotImplemented is a placeholder handler that returns a non-implemented
// opcode error.
func opExecNotImplemented(_ *execContext, ent Entity) *Error {
	return &Error{
		message: "opcode " + ent.getOpcode().String() + " not implemented",
	}
}
