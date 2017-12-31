package vm

import (
	"bytes"
	"gopheros/device/acpi/aml/entity"
	"gopheros/kernel"
	"gopheros/kernel/kfmt"
)

type opDirection uint8

const (
	opDirectionIn opDirection = iota
	opDirectionOut
)

type opMapping struct {
	vmOp       uint8
	compilerFn func(*compilerContext, uint8, entity.Entity) *kernel.Error
}

type compilerContext struct {
	rootNS entity.Container

	// opcodeMap contains the mappings of AML opcodes into a VM opcode plus
	// a compiler function to handle the opcode conversion.
	opcodeMap map[entity.AMLOpcode]opMapping

	// methodCallMap maps an entity definition to an index in VM.methodCalls.
	methodCallMap map[*entity.Method]uint16

	vmCtx   *Context
	lastErr *kernel.Error
}

func newCompilerContext(rootNS entity.Container) *compilerContext {
	compCtx := &compilerContext{
		rootNS:        rootNS,
		methodCallMap: make(map[*entity.Method]uint16),
		vmCtx:         new(Context),
	}

	// the opcodeMap needs to be dynamically populated here instead of
	// having a shared, globally initialized map to prevent the Go compiler
	// from complaining about potential initialization loops (e.g.
	// compileFn -> compileStatement -> compileFn...)
	compCtx.opcodeMap = map[entity.AMLOpcode]opMapping{
		entity.OpReturn: {opRet, compileReturn},
		entity.OpStore:  {opNop, compileAssignment},
		// Arithmetic opcodes
		entity.OpAdd:             {opAdd, compileBinaryOperator},
		entity.OpSubtract:        {opSub, compileBinaryOperator},
		entity.OpMultiply:        {opMul, compileBinaryOperator},
		entity.OpIncrement:       {opAdd, compilePostfixOperator},
		entity.OpDecrement:       {opSub, compilePostfixOperator},
		entity.OpDivide:          {opDiv, compileDivisionOperator},
		entity.OpMod:             {opMod, compileBinaryOperator},
		entity.OpShiftLeft:       {opShl, compileBinaryOperator},
		entity.OpShiftRight:      {opShr, compileBinaryOperator},
		entity.OpAnd:             {opAnd, compileBinaryOperator},
		entity.OpOr:              {opOr, compileBinaryOperator},
		entity.OpNand:            {opNand, compileBinaryOperator},
		entity.OpNor:             {opNor, compileBinaryOperator},
		entity.OpXor:             {opXor, compileBinaryOperator},
		entity.OpNot:             {opNot, compileUnaryOperator},
		entity.OpFindSetLeftBit:  {opFindSlb, compileUnaryOperator},
		entity.OpFindSetRightBit: {opFindSrb, compileUnaryOperator},
		// Logic opcodes
		entity.OpLEqual:   {opJe, compileLogicOperator},
		entity.OpLGreater: {opJg, compileLogicOperator},
		entity.OpLLess:    {opJl, compileLogicOperator},
		entity.OpLnot:     {opXor, compileLogicNotOperator},
		// Logic and/or is modelled as a bitwise AND/OR on the output
		// of two logic expressions
		entity.OpLand: {opAnd, compileBinaryOperator},
		entity.OpLor:  {opOr, compileBinaryOperator},
	}

	return compCtx
}

// Compile returns a bytecode representation of the AML tree starting at
// rootNS that can be consumed by the AML VM.
func Compile(rootNS entity.Container) (*Context, *kernel.Error) {
	compCtx := newCompilerContext(rootNS)
	compileAMLTree(compCtx)

	if compCtx.lastErr != nil {
		return nil, compCtx.lastErr
	}

	return compCtx.vmCtx, nil
}

// compileAMLTree converts the AML entity tree inside the supplied
// compilerContext into a bytecode representation that can be consumed by the
// host VM. This function is intentionally separate from the above Compile
// function so tests can easily stub the opcodeMap field of compCtx.
func compileAMLTree(compCtx *compilerContext) {
	populateMethodMap(compCtx, compCtx.rootNS)
	entity.Visit(0, compCtx.rootNS, entity.TypeMethod, func(_ int, ent entity.Entity) bool {
		if compCtx.lastErr != nil {
			return false
		}

		compCtx.lastErr = compileMethod(compCtx, ent.(*entity.Method))
		return false
	})
}

// populateMethodMap visits all method entities under root and populates the
// VM.methodCalls list as well as compCtx.methodCallMap which is used by the
// asmContext to efficiently encode method calls to the correct index in the
// methodCalls list.
func populateMethodMap(compCtx *compilerContext, root entity.Container) {
	entity.Visit(0, root, entity.TypeMethod, func(_ int, ent entity.Entity) bool {
		m := &methodCall{method: ent.(*entity.Method), entrypoint: 0xffffffff}
		compCtx.vmCtx.methodCalls = append(compCtx.vmCtx.methodCalls, m)
		compCtx.methodCallMap[m.method] = uint16(len(compCtx.vmCtx.methodCalls) - 1)
		return false
	})
}

// compileMethod receives an AML method entity and emits the appropriate
// bytecode for the statements it contains. Calls to compileMethod will also
// populate the method's entrypoint address which is used by the VM to
// implement the "call" opcode.
func compileMethod(compCtx *compilerContext, method *entity.Method) *kernel.Error {
	var errBuf bytes.Buffer

	// Setup the entrypoint address for this method.
	compCtx.vmCtx.methodCalls[compCtx.methodCallMap[method]].entrypoint = uint32(len(compCtx.vmCtx.bytecode))
	for stmtIndex, stmt := range method.Children() {
		if err := compileStatement(compCtx, stmt); err != nil {
			kfmt.Fprintf(&errBuf, "[%s:%d] %s", method.Name(), stmtIndex, err.Message)
			err.Message = errBuf.String()
			return err
		}
	}

	return nil
}

// compileReturn generates the appropriate opcode stream to returning from a
// method invocation. If the supplied entity contains no operands then this
// function will emit a "ret_void" opcode.
func compileReturn(compCtx *compilerContext, vmOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) == 0 {
		vmOp = opRetVoid
	} else if err := compileOperand(compCtx, operands[0], opDirectionIn); err != nil {
		return err
	}

	emit8(compCtx, vmOp)
	return nil
}

// compileAssignment generates the appropriate opcode stream for handling
// an assignment of one AML entity to another.
func compileAssignment(compCtx *compilerContext, _ uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) != 2 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for assignment",
		}
	}

	// Compile operands
	if err := compileOperand(compCtx, operands[0], opDirectionIn); err != nil {
		return err
	}
	return compileOperand(compCtx, operands[1], opDirectionOut)
}

// compilePostfixOperator generates the appropriate opcode stream for a postfix
// operator (e.g. x++, or x--) that also stores a copy of the result back to
// itself.
func compilePostfixOperator(compCtx *compilerContext, vmOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 1 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for postfix operator " + ent.Opcode().String(),
		}
	}

	// Compile operands (source, one)
	if err := compileOperand(compCtx, operands[0], opDirectionIn); err != nil {
		return err
	}
	emit8(compCtx, opPushOne)

	// Emit opcode for the operator
	emit8(compCtx, vmOp)

	// Always store result back to source
	return compileOperand(compCtx, operands[0], opDirectionOut)
}

// compileUnaryOperator generates the appropriate opcode stream for a unary
// operator that optionally stores the results into a second operand.
func compileUnaryOperator(compCtx *compilerContext, vmOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 1 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for unary operator " + ent.Opcode().String(),
		}
	}

	// Compile operand
	if err := compileOperand(compCtx, operands[0], opDirectionIn); err != nil {
		return err
	}

	// Emit opcode for the operator
	emit8(compCtx, vmOp)

	// Store the result if a destination operand is specified
	if len(operands) == 2 && !isNilTarget(operands[1]) {
		return compileOperand(compCtx, operands[1], opDirectionOut)
	}

	return nil
}

// compileBinaryOperator generates the appropriate opcode stream for a binary
// operator that optionally stores the results into a third operand.
func compileBinaryOperator(compCtx *compilerContext, vmOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 2 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for binary operator " + ent.Opcode().String(),
		}
	}

	// Compile operands
	for opIndex := 0; opIndex < 2; opIndex++ {
		if err := compileOperand(compCtx, operands[opIndex], opDirectionIn); err != nil {
			return err
		}
	}

	// Emit opcode for the operator
	emit8(compCtx, vmOp)

	// Store the result if a destination operand is specified
	if len(operands) == 3 && !isNilTarget(operands[2]) {
		return compileOperand(compCtx, operands[2], opDirectionOut)
	}

	return nil
}

// compileLogicOperator generates a stream of conditional jump opcodes that
// implement a logic operator.
func compileLogicOperator(compCtx *compilerContext, vmJmpOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 2 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for logic operator " + ent.Opcode().String(),
		}
	}

	// To emulate the logic operation the following sequence of
	// instructions gets generated:
	//
	//   push op1
	//   push op2
	//   vmJmpOp true_label ; IP += 11
	//   push_0             ; cmp evaluated to false
	//   jmp done_label     ; IP += 6
	// true_label: push_1   ; cmp evaluated to true
	// done_label:
	for opIndex := 0; opIndex < 2; opIndex++ {
		if err := compileOperand(compCtx, operands[opIndex], opDirectionIn); err != nil {
			return err
		}
	}

	emit32(compCtx, vmJmpOp, uint32(len(compCtx.vmCtx.bytecode))+11)
	emit8(compCtx, opPushZero)
	emit32(compCtx, opJmp, uint32(len(compCtx.vmCtx.bytecode))+6)
	emit8(compCtx, opPushOne)

	return nil
}

// compileLogicNotOperator generates the appropriate opcode stream for toggling
// the top-most stack entry between 0 and 1.
func compileLogicNotOperator(compCtx *compilerContext, _ uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 1 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for operator " + ent.Opcode().String(),
		}
	}

	// Compile operand
	if err := compileOperand(compCtx, operands[0], opDirectionIn); err != nil {
		return err
	}

	// The top of the stack will now be a 0/1 value. By XOR-ing with 1
	// we can toggle the value
	emit8(compCtx, opPushOne)
	emit8(compCtx, opXor)

	return nil
}

// compileDivisionOperator generates the appropriate opcode stream for a
// division operator that optionally stores the remainder and the quotient to
// the optional third and fourth operands.
func compileDivisionOperator(compCtx *compilerContext, vmOp uint8, ent entity.Entity) *kernel.Error {
	operands := ent.Args()
	if len(operands) < 2 {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "unexpected operand count for operator " + ent.Opcode().String(),
		}
	}

	// Compile operands
	for opIndex := 0; opIndex < 2; opIndex++ {
		if err := compileOperand(compCtx, operands[opIndex], opDirectionIn); err != nil {
			return err
		}
	}

	// Emit opcode for the operator
	emit8(compCtx, vmOp)

	// After the division, the top of the stack contains the remainder. If
	// we need to store it emit a store opcode before popping it of the
	// stack
	if len(operands) >= 3 && !isNilTarget(operands[2]) {
		if err := compileOperand(compCtx, operands[2], opDirectionOut); err != nil {
			return err
		}
	}
	emit8(compCtx, opPop)

	// The top of the stack now contains the quotient which can optionally be
	// stored in the fourth operand
	if len(operands) == 4 && !isNilTarget(operands[3]) {
		return compileOperand(compCtx, operands[3], opDirectionOut)
	}
	return nil
}

// compileOperand generates the appropriate opcode stream for an operator's
// opcode. The dir argument controls whether the operand is read from or
// written to. As AML statements may contain nested statements as operands,
// this function will fallback to a call to compileStatement if the operand is
// not a local arg, method arg or a constant.
func compileOperand(compCtx *compilerContext, arg interface{}, dir opDirection) *kernel.Error {
	ent, isEnt := arg.(entity.Entity)
	if !isEnt {
		return &kernel.Error{
			Module:  "acpi_aml_compiler",
			Message: "compileArg: arg must be an AML entity",
		}
	}
	entOp := ent.Opcode()

	if entity.OpIsLocalArg(entOp) {
		outOp := opPushLocal0
		if dir == opDirectionOut {
			outOp = opStoreLocal0
		}

		emit8(compCtx, outOp+uint8(entOp-entity.OpLocal0))
		return nil
	} else if entity.OpIsMethodArg(entOp) {
		outOp := opPushArg0
		if dir == opDirectionOut {
			outOp = opStoreArg0
		}

		emit8(compCtx, outOp+uint8(entOp-entity.OpArg0))
		return nil
	} else if constant, isConst := ent.(*entity.Const); isConst {
		if dir == opDirectionOut {
			return &kernel.Error{
				Module:  "acpi_aml_compiler",
				Message: "compileArg: attempt to store value to a constant",
			}
		}

		// Special constants
		switch constant.Value {
		case uint64(0):
			emit8(compCtx, opPushZero)
		case uint64(1):
			emit8(compCtx, opPushOne)
		case uint64((1 << 64) - 1):
			emit8(compCtx, opPushOnes)
		default:
			compCtx.vmCtx.constants = append(compCtx.vmCtx.constants, constant)
			emit16(compCtx, opPushConst, uint16(len(compCtx.vmCtx.constants)-1))
		}
		return nil
	}

	return compileStatement(compCtx, ent)
}

// isNilTarget returns true if t is nil or a nil const entity.
func isNilTarget(target interface{}) bool {
	if target == nil {
		return true
	}

	if ent, ok := target.(*entity.Const); ok {
		return ent.Value != nil && ent.Value != uint64(0)
	}

	return false
}

// compileStatement receives as input an AML entity that represents a method
// statement and emits the appropriate bytecode for executing it.
func compileStatement(compCtx *compilerContext, ent entity.Entity) *kernel.Error {
	mapping, hasMapping := compCtx.opcodeMap[ent.Opcode()]
	if !hasMapping {
		emit8(compCtx, opNop)
		return nil
	}

	return mapping.compilerFn(compCtx, mapping.vmOp, ent)
}

// emit8 appends an opcode that does not require any operands to the generated
// bytecode stream.
func emit8(compCtx *compilerContext, op uint8) {
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, op)
}

// emit16 appends an opcode followed by an word operand to the generated
// bytecode stream. The word operand will be encoded in big-endian format.
func emit16(compCtx *compilerContext, op uint8, operand uint16) {
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, op)
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8((operand>>8)&0xff))
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8(operand&0xff))
}

// emit32 appends an opcode followed by an dword operand to the generated
// bytecode stream. The word operand will be encoded in big-endian format.
func emit32(compCtx *compilerContext, op uint8, operand uint32) {
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, op)
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8((operand>>24)&0xff))
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8((operand>>16)&0xff))
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8((operand>>8)&0xff))
	compCtx.vmCtx.bytecode = append(compCtx.vmCtx.bytecode, uint8(operand&0xff))
}
