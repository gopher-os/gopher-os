package aml

import (
	"bytes"
	"gopheros/device/acpi/table"
	"gopheros/kernel/kfmt"
	"io"
)

const (
	// According to the ACPI spec, methods can use up to 8 local args and
	// can receive up to 7 method args.
	maxLocalArgs  = 8
	maxMethodArgs = 7
)

var (
	errNilStoreOperands          = &Error{message: "vmStore: src and/or dst operands are nil"}
	errInvalidStoreDestination   = &Error{message: "vmStore: destination operand is not an AML entity"}
	errCopyFailed                = &Error{message: "vmCopyObject: copy failed"}
	errConversionFromEmptyString = &Error{message: "vmConvert: conversion from String requires a non-empty value"}
	errArgIndexOutOfBounds       = &Error{message: "vm: arg index out of bounds"}
	errDivideByZero              = &Error{message: "vm: division by zero"}
	errInvalidComparisonType     = &Error{message: "vm: logic opcodes can only be applied to Integer, String or Buffer arguments"}
	errWhileBodyNotScopedEntity  = &Error{message: "vmOpWHile: Wihile body must be a scoped entity"}
	errIfBodyNotScopedEntity     = &Error{message: "vmOpIf: If body must be a scoped entity"}
	errElseBodyNotScopedEntity   = &Error{message: "vmOpIf: Else body must be a scoped entity"}
)

// objRef is a pointer to an argument (local or global) or a named AML object.
type objRef struct {
	ref interface{}

	// isArgRef specifies whether this is a reference to a method argument.
	// Different rules (p.884) apply for this particular type of reference.
	isArgRef bool
}

// ctrlFlowType describes the different ways that the control flow can be altered
// while executing a set of AML opcodes.
type ctrlFlowType uint8

// The list of supported control flows.
const (
	ctrlFlowTypeNextOpcode ctrlFlowType = iota
	ctrlFlowTypeBreak
	ctrlFlowTypeContinue
	ctrlFlowTypeFnReturn
)

// execContext holds the AML interpreter state while an AML method executes.
type execContext struct {
	localArg  [maxLocalArgs]interface{}
	methodArg [maxMethodArgs]interface{}

	// ctrlFlow specifies how the VM should select the next instruction to
	// execute.
	ctrlFlow ctrlFlowType

	// retVal holds the return value from a method if ctrlFlow is set to
	// the value ctrlFlowTypeFnReturn or the intermediate value of an AML
	// opcode execution.
	retVal interface{}

	vm *VM
	IP uint32
}

// frame contains information about the location within a method (the VM
// instruction pointer) and the actual AML opcode that the VM was processing
// when an error occurred. Entry also contains information about the method
// name and the ACPI table that defined it.
type frame struct {
	table  string
	method string
	IP     uint32
	instr  string
}

// Error describes errors that occur while executing AML code.
type Error struct {
	message string

	// trace contains a list of trace entries that correspond to the AML method
	// invocations up to the point where an error occurred. To construct the
	// correct execution tree from a Trace, its entries must be processed in
	// LIFO order.
	trace []*frame
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.message
}

// StackTrace returns a formatted stack trace for this error.
func (e *Error) StackTrace() string {
	if len(e.trace) == 0 {
		return "No stack trace available"
	}

	var buf bytes.Buffer
	buf.WriteString("Stack trace:\n")

	// We need to process the trace list in LIFO order.
	for index, offset := 0, len(e.trace)-1; index < len(e.trace); index, offset = index+1, offset-1 {
		entry := e.trace[offset]
		kfmt.Fprintf(&buf, "[%3x] [%s] [%s():0x%x] opcode: %s\n", index, entry.table, entry.method, entry.IP, entry.instr)
	}

	return buf.String()
}

// VM is a structure that stores the output of the AML bytecode parser and
// provides methods for interpreting any executable opcode.
type VM struct {
	errWriter io.Writer

	tableResolver table.Resolver
	tableParser   *Parser

	// rootNS holds a pointer to the root of the ACPI tree.
	rootNS ScopeEntity

	// According to the ACPI spec, the Revision field in the DSDT specifies
	// whether integers are treated as 32 or 64-bits. The VM memoizes this
	// value so that it can be used by the data conversion helpers.
	sizeOfIntInBits int

	jumpTable         [numOpcodes + 1]opHandler
	tableHandleToName map[uint8]string
}

// NewVM creates a new AML VM and initializes it with the default scope
// hierarchy and pre-defined objects contained in the ACPI specification.
func NewVM(errWriter io.Writer, resolver table.Resolver) *VM {
	root := defaultACPIScopes()

	return &VM{
		rootNS:        root,
		errWriter:     errWriter,
		tableResolver: resolver,
		tableParser:   NewParser(errWriter, root),
	}
}

// Init attempts to locate and parse the AML byte-code contained in the
// system's DSDT and SSDT tables.
func (vm *VM) Init() *Error {
	for _, tableName := range []string{"DSDT", "SSDT"} {
		header := vm.tableResolver.LookupTable(tableName)
		if header == nil {
			continue
		}

		tableHandle := vm.allocateTableHandle(tableName)
		if err := vm.tableParser.ParseAML(tableHandle, tableName, header); err != nil {
			return &Error{message: err.Module + ": " + err.Error()}
		}

		if tableName == "DSDT" {
			vm.sizeOfIntInBits = 32
			if header.Revision >= 2 {
				vm.sizeOfIntInBits = 64
			}
		}
	}

	vm.populateJumpTable()
	return vm.checkEntities()
}

// allocateTableHandle reserves a handle for tableName and updates the internal
// tableHandleToName map.
func (vm *VM) allocateTableHandle(tableName string) uint8 {
	if vm.tableHandleToName == nil {
		vm.tableHandleToName = make(map[uint8]string)
	}

	nextHandle := uint8(len(vm.tableHandleToName) + 1)
	vm.tableHandleToName[nextHandle] = tableName
	return nextHandle
}

// Lookup traverses a potentially nested absolute AML path and returns the
// Entity reachable via that path or nil if the path does not point to a
// defined Entity.
func (vm *VM) Lookup(absPath string) Entity {
	if absPath == "" || absPath[0] != '\\' {
		return nil
	}

	// If we just search for `\` return the root namespace
	if len(absPath) == 1 {
		return vm.rootNS
	}

	return scopeFindRelative(vm.rootNS, absPath[1:])
}

// checkEntities performs a DFS on the entity tree and initializes
// entities that defer their initialization until an AML interpreter
// is available.
func (vm *VM) checkEntities() *Error {
	var (
		err *Error
		ctx = &execContext{vm: vm}
	)

	vm.Visit(EntityTypeAny, func(_ int, ent Entity) bool {
		// Stop recursing after the first detected error
		if err != nil {
			return false
		}

		// Peek into named entities that wrap other entities
		if namedEnt, ok := ent.(*namedEntity); ok {
			if nestedEnt, ok := namedEnt.args[0].(Entity); ok {
				ent = nestedEnt
			}
		}

		switch typ := ent.(type) {
		case *Method:
			// Calculate the start and end IP value for each scoped entity inside the
			// method. This is required for emitting accurate stack traces when the
			// method is invoked.
			_ = calcIPOffsets(typ, 0)
			return false
		case *bufferEntity:
			// According to p.911-912 of the spec:
			// - if a size is specified but no initializer the VM should allocate
			//   a buffer of the requested size
			// - if both a size and initializer are specified but size > len(data)
			//   then the data needs to be padded with zeroes

			// Evaluate size arg as an integer
			var size interface{}
			if size, err = vmConvert(ctx, typ.size, valueTypeInteger); err != nil {
				return false
			}
			sizeAsInt := size.(uint64)

			if typ.data == nil {
				typ.data = make([]byte, size.(uint64))
			}

			if dataLen := uint64(len(typ.data)); dataLen < sizeAsInt {
				typ.data = append(typ.data, make([]byte, sizeAsInt-dataLen)...)
			}
		}

		return true
	})

	return err
}

// Visit performs a DFS on the AML namespace tree invoking the visitor for each
// encountered entity whose type matches entType. Namespace nodes are visited
// in parent to child order a property which allows the supplied visitor
// function to signal that it's children should not be visited.
func (vm *VM) Visit(entType EntityType, visitorFn Visitor) {
	scopeVisit(0, vm.rootNS, entType, visitorFn)
}

// execMethod creates a new execution context and invokes the given method
// passing along the supplied args. It populates the retVal of the input
// context with the result of the method invocation.
func (vm *VM) execMethod(ctx *execContext, method *Method, args ...interface{}) *Error {
	var (
		invCtx = execContext{vm: vm}
		err    *Error
	)

	// Resolve invocation args and populate methodArgs for the new context
	for argIndex := 0; argIndex < len(args); argIndex++ {
		invCtx.methodArg[argIndex], err = vmLoad(ctx, args[argIndex])
		if err != nil {
			err.trace = append(err.trace, &frame{
				table:  vm.tableHandleToName[method.TableHandle()],
				method: method.Name(),
				IP:     0,
				instr:  "read method args",
			})
			return err
		}
	}

	// Execute method and resolve the return value before storing it to the
	// parent context's retVal.
	if err = execBlock(&invCtx, method); err == nil {
		ctx.retVal, err = vmLoad(&invCtx, invCtx.retVal)
	}

	// Populate missing data in captured trace till we reach a frame that has its
	// table name field populated.
	if err != nil {
		for index := len(err.trace) - 1; index >= 0; index-- {
			if err.trace[index].table != "" {
				break
			}

			err.trace[index].table = vm.tableHandleToName[method.TableHandle()]
			err.trace[index].method = method.Name()
		}
	}

	return err
}

// execBlock attempts to execute all AML opcodes in the supplied scoped entity.
// If all opcodes are successfully executed, the provided execContext will be
// updated to reflect the current VM state. Otherwise, an error will be
// returned.
func execBlock(ctx *execContext, block ScopeEntity) *Error {
	var (
		instrList  = block.Children()
		numInstr   = len(instrList)
		instrIndex int
		lastIP     uint32
	)

	for ctx.IP, instrIndex = block.blockStartIPOffset(), 0; instrIndex < numInstr && ctx.ctrlFlow == ctrlFlowTypeNextOpcode; instrIndex++ {
		// If the opcode executes a scoped block then ctx.IP will be modified and
		// unless we keep track of its original value we will not be able to
		// provide an accurate trace if the opcode handler returns back an error.
		ctx.IP++
		lastIP = ctx.IP

		instr := instrList[instrIndex]
		if err := ctx.vm.jumpTable[instr.getOpcode()](ctx, instr); err != nil {
			// Append an entry to the stack trace; the parent execMethod call will
			// automatically populate the missing method and table information.
			err.trace = append(err.trace, &frame{
				IP:    lastIP,
				instr: instr.getOpcode().String(),
			})
			return err
		}
	}

	return nil
}

// calcIPOffsets visits all scoped entities inside the method m and updates
// their start and end IP offset values relative to the provided relIP value.
func calcIPOffsets(scope ScopeEntity, relIP uint32) uint32 {
	var startIP = relIP

	for _, ent := range scope.Children() {
		relIP++

		switch ent.getOpcode() {
		case opIf, opWhile:
			// arg 0 is the preficate which we must exclude from the calculation
			for argIndex, arg := range ent.getArgs() {
				if argIndex == 0 {
					continue
				}

				if argEnt, isScopedEnt := arg.(ScopeEntity); isScopedEnt {
					// Recursively visit scoped entities and adjust the current IP
					relIP = calcIPOffsets(argEnt, relIP)
				}
			}
		}
	}

	scope.setBlockIPOffsets(startIP, relIP)
	return relIP
}

// defaultACPIScopes constructs a tree of scoped entities that correspond to
// the predefined scopes contained in the ACPI specification and returns back
// its root node.
func defaultACPIScopes() ScopeEntity {
	rootNS := &scopeEntity{op: opScope, name: `\`}
	rootNS.Append(&scopeEntity{op: opScope, name: `_GPE`}) // General events in GPE register block
	rootNS.Append(&scopeEntity{op: opScope, name: `_PR_`}) // ACPI 1.0 processor namespace
	rootNS.Append(&scopeEntity{op: opScope, name: `_SB_`}) // System bus with all device objects
	rootNS.Append(&scopeEntity{op: opScope, name: `_SI_`}) // System indicators
	rootNS.Append(&scopeEntity{op: opScope, name: `_TZ_`}) // ACPI 1.0 thermal zone namespace

	return rootNS
}
