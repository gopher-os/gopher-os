package kernel

import "testing"

func TestKernelError(t *testing.T) {
	err := &Error{
		Module:  "foo",
		Message: "error message",
	}

	if err.Error() != err.Message {
		t.Fatalf("expected to err.Error() to return %q; got %q", err.Message, err.Error())
	}
}
