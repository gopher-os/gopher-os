package errors

var (
	ErrInvalidParamValue = KernelError("invalid parameter value")
)

// KernelError is a trivial implementation of a kernel error message that doens't
// require a memory allocation. It is used as an alternative to errors.New.
type KernelError string

// Error implements the error interface.
func (err KernelError) Error() string {
	return string(err)
}
