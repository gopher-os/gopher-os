package vm

import "gopheros/device/acpi/aml/entity"

// Context contains the result of transforming the methods defined by a
// hierarchy of AML entities into the bytecode format that can be executed by
// the VM implementation in this package.
type Context struct {
	// The root namespace for the parsed AML entities.
	rootNS entity.Container

	constants   []*entity.Const
	buffers     []*entity.Buffer
	packages    []*entity.Package
	methodCalls []*methodCall

	bytecode []uint8
}

// methodCall contains information about the bytecode address where a particular
// AML method begins as well as a link to the actual method entity.
type methodCall struct {
	entrypoint uint32
	method     *entity.Method
}
