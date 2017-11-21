package aml

import "testing"

func TestVMOpStoreErrors(t *testing.T) {
	// Wrong arg count
	if err := vmOpStore(nil, new(unnamedEntity)); err != errArgIndexOutOfBounds {
		t.Errorf("expected to get errArgIndexOutOfBounds; got %v", err)
	}

	// Error loading the value to be stored
	expErr := &Error{message: "something went wrong"}

	vm := NewVM(nil, nil)
	vm.populateJumpTable()
	vm.jumpTable[0] = func(_ *execContext, ent Entity) *Error {
		return expErr
	}

	ent := &unnamedEntity{
		args: []interface{}{
			// vmLoad will try to eval jumptable[0] which we monkey-patched to return an error
			&unnamedEntity{op: 0},
			uint64(128),
		},
	}

	if err := vmOpStore(&execContext{vm: vm}, ent); err != expErr {
		t.Errorf("expected to get error %q; got %v", expErr.Error(), err)
	}

	// Error storing the value
	ent.args = []interface{}{
		uint64(128),
		uint64(0xf00), // storing to a non-AML entity is an error
	}

	if err := vmOpStore(nil, ent); err != errInvalidStoreDestination {
		t.Errorf("expected to get errInvalidStoreDestination; got %v", err)
	}
}
