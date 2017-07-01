package irq

// ExceptionNum defines an exception number that can be
// passed to the HandleException and HandleExceptionWithCode
// functions.
type ExceptionNum uint8

const (
	// DoubleFault occurs when an exception is unhandled
	// or when an exception occurs while the CPU is
	// trying to call an exception handler.
	DoubleFault = ExceptionNum(8)

	// GPFException is raised when a general protection fault occurs.
	GPFException = ExceptionNum(13)

	// PageFaultException is raised when a PDT or
	// PDT-entry is not present or when a privilege
	// and/or RW protection check fails.
	PageFaultException = ExceptionNum(14)
)

// ExceptionHandler is a function that handles an exception that does not push
// an error code to the stack. If the handler returns, any modifications to the
// supplied Frame and/or Regs pointers will be propagated back to the location
// where the exception occurred.
type ExceptionHandler func(*Frame, *Regs)

// ExceptionHandlerWithCode is a function that handles an exception that pushes
// an error code to the stack. If the handler returns, any modifications to the
// supplied Frame and/or Regs pointers will be propagated back to the location
// where the exception occurred.
type ExceptionHandlerWithCode func(uint64, *Frame, *Regs)

// HandleException registers an exception handler (without an error code) for
// the given interrupt number.
func HandleException(exceptionNum ExceptionNum, handler ExceptionHandler)

// HandleExceptionWithCode registers an exception handler (with an error code)
// for the given interrupt number.
func HandleExceptionWithCode(exceptionNum ExceptionNum, handler ExceptionHandlerWithCode)
