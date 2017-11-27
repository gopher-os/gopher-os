package aml

import (
	"gopheros/device/acpi/table"
	"io"
)

const (
	// According to the ACPI spec, methods can use up to 8 local args and
	// can receive up to 7 method args.
	maxLocalArgs  = 8
	maxMethodArgs = 7
)

var (
	errNilStoreOperands        = &Error{message: "vmStore: src and/or dst operands are nil"}
	errInvalidStoreDestination = &Error{message: "vmStore: destination operand is not an AML entity"}
	errCopyFailed              = &Error{message: "vmCopyObject: copy failed"}
	errConversionFailed        = &Error{message: "vmConvert: conversion failed"}
	errArgIndexOutOfBounds     = &Error{message: "vm: arg index out of bounds"}
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
}

// Error describes errors that occur while executing AML code.
type Error struct {
	message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.message
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

	jumpTable [numOpcodes]opHandler
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
	for tableHandle, tableName := range []string{"DSDT", "SSDT"} {
		header := vm.tableResolver.LookupTable(tableName)
		if header == nil {
			continue
		}
		if err := vm.tableParser.ParseAML(uint8(tableHandle+1), tableName, header); err != nil {
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
	return nil
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

// Visit performs a DFS on the AML namespace tree invoking the visitor for each
// encountered entity whose type matches entType. Namespace nodes are visited
// in parent to child order a property which allows the supplied visitor
// function to signal that it's children should not be visited.
func (vm *VM) Visit(entType EntityType, visitorFn Visitor) {
	scopeVisit(0, vm.rootNS, entType, visitorFn)
}

// execBlock attempts to execute all AML opcodes in the supplied scoped entity.
// If all opcodes are successfully executed, the provided execContext will be
// updated to reflect the current VM state. Otherwise, an error will be
// returned.
func (vm *VM) execBlock(ctx *execContext, block ScopeEntity) *Error {
	instrList := block.Children()
	numInstr := len(instrList)

	for instrIndex := 0; instrIndex < numInstr && ctx.ctrlFlow == ctrlFlowTypeNextOpcode; instrIndex++ {
		instr := instrList[instrIndex]
		if err := vm.jumpTable[instr.getOpcode()](ctx, instr); err != nil {
			return err
		}
	}

	return nil
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
