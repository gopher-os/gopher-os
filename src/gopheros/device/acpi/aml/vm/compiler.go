package vm

import (
	"bytes"
	"gopheros/device/acpi/aml/entity"
	"gopheros/kernel"
	"gopheros/kernel/kfmt"
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
	compCtx.opcodeMap = map[entity.AMLOpcode]opMapping{}

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
