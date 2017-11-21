package aml

import "testing"

func TestFlowExpressionErrors(t *testing.T) {
	t.Run("opReturn errors", func(t *testing.T) {
		// opReturn expects an argument to evaluate as the return value
		if err := vmOpReturn(nil, new(unnamedEntity)); err != errArgIndexOutOfBounds {
			t.Errorf("expected to get errArgIndexOutOfBounds; got %v", err)
		}
	})
}
