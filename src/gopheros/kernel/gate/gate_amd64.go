package gate

import (
	"gopheros/kernel/kfmt"
	"io"
)

// Registers contains a snapshot of all register values when an exception,
// interrupt or syscall occurs.
type Registers struct {
	RAX uint64
	RBX uint64
	RCX uint64
	RDX uint64
	RSI uint64
	RDI uint64
	RBP uint64
	R8  uint64
	R9  uint64
	R10 uint64
	R11 uint64
	R12 uint64
	R13 uint64
	R14 uint64
	R15 uint64

	// Info contains the exception code for exceptions, the syscall number
	// for syscall entries or the IRQ number for HW interrupts.
	Info uint64

	// The return frame used by IRETQ
	RIP    uint64
	CS     uint64
	RFlags uint64
	RSP    uint64
	SS     uint64
}

// DumpTo outputs the register contents to w.
func (r *Registers) DumpTo(w io.Writer) {
	kfmt.Fprintf(w, "RAX = %16x RBX = %16x\n", r.RAX, r.RBX)
	kfmt.Fprintf(w, "RCX = %16x RDX = %16x\n", r.RCX, r.RDX)
	kfmt.Fprintf(w, "RSI = %16x RDI = %16x\n", r.RSI, r.RDI)
	kfmt.Fprintf(w, "RBP = %16x\n", r.RBP)
	kfmt.Fprintf(w, "R8  = %16x R9  = %16x\n", r.R8, r.R9)
	kfmt.Fprintf(w, "R10 = %16x R11 = %16x\n", r.R10, r.R11)
	kfmt.Fprintf(w, "R12 = %16x R13 = %16x\n", r.R12, r.R13)
	kfmt.Fprintf(w, "R14 = %16x R15 = %16x\n", r.R14, r.R15)
	kfmt.Fprintf(w, "\n")
	kfmt.Fprintf(w, "RIP = %16x CS  = %16x\n", r.RIP, r.CS)
	kfmt.Fprintf(w, "RSP = %16x SS  = %16x\n", r.RSP, r.SS)
	kfmt.Fprintf(w, "RFL = %16x\n", r.RFlags)
}

// InterruptNumber describes an x86 interrupt/exception/trap slot.
type InterruptNumber uint8

const (
	// DivideByZero occurs when dividing any number by 0 using the DIV or
	// IDIV instruction.
	DivideByZero = InterruptNumber(0)

	// NMI (non-maskable-interrupt) is a hardware interrupt that indicates
	// issues with RAM or unrecoverable hardware problems. It may also be
	// raised by the CPU when a watchdog timer is enabled.
	NMI = InterruptNumber(2)

	// Overflow occurs when an overflow occurs (e.g result of division
	// cannot fit into the registers used).
	Overflow = InterruptNumber(4)

	// BoundRangeExceeded occurs when the BOUND instruction is invoked with
	// an index out of range.
	BoundRangeExceeded = InterruptNumber(5)

	// InvalidOpcode occurs when the CPU attempts to execute an invalid or
	// undefined instruction opcode.
	InvalidOpcode = InterruptNumber(6)

	// DeviceNotAvailable occurs when the CPU attempts to execute an
	// FPU/MMX/SSE instruction while no FPU is available or while
	// FPU/MMX/SSE support has been disabled by manipulating the CR0
	// register.
	DeviceNotAvailable = InterruptNumber(7)

	// DoubleFault occurs when an unhandled exception occurs or when an
	// exception occurs within a running exception handler.
	DoubleFault = InterruptNumber(8)

	// InvalidTSS occurs when the TSS points to an invalid task segment
	// selector.
	InvalidTSS = InterruptNumber(10)

	// SegmentNotPresent occurs when the CPU attempts to invoke a present
	// gate with an invalid stack segment selector.
	SegmentNotPresent = InterruptNumber(11)

	// StackSegmentFault occurs when attempting to push/pop from a
	// non-canonical stack address or when the stack base/limit (set in
	// GDT) checks fail.
	StackSegmentFault = InterruptNumber(12)

	// GPFException occurs when a general protection fault occurs.
	GPFException = InterruptNumber(13)

	// PageFaultException occurs when a page directory table (PDT) or one
	// of its entries is not present or when a privilege and/or RW
	// protection check fails.
	PageFaultException = InterruptNumber(14)

	// FloatingPointException occurs while invoking an FP instruction while:
	//  - CR0.NE = 1 OR
	//  - an unmasked FP exception is pending
	FloatingPointException = InterruptNumber(16)

	// AlignmentCheck occurs when alignment checks are enabled and an
	// unaligmed memory access is performed.
	AlignmentCheck = InterruptNumber(17)

	// MachineCheck occurs when the CPU detects internal errors such as
	// memory-, bus- or cache-related errors.
	MachineCheck = InterruptNumber(18)

	// SIMDFloatingPointException occurs when an unmasked SSE exception
	// occurs while CR4.OSXMMEXCPT is set to 1. If the OSXMMEXCPT bit is
	// not set, SIMD FP exceptions cause InvalidOpcode exceptions instead.
	SIMDFloatingPointException = InterruptNumber(19)
)

// Init runs the appropriate CPU-specific initialization code for enabling
// support for interrupt handling.
func Init() {
	installIDT()
}

// HandleInterrupt ensures that the provided handler will be invoked when a
// particular interrupt number occurs. The value of the istOffset argument
// specifies the offset in the interrupt stack table (if 0 then IST is not
// used).
func HandleInterrupt(intNumber InterruptNumber, istOffset uint8, handler func(*Registers))

// installIDT populates idtDescriptor with the address of IDT and loads it to
// the CPU. All gate entries are initially marked as non-present and must be
// explicitly enabled via a call to install{Trap,IRQ,Task}Handler.
func installIDT()

// dispatchInterrupt is invoked by the interrupt gate entrypoints to route
// an incoming interrupt to the selected handler.
func dispatchInterrupt()

// interruptGateEntries contains a list of generated entries for each possible
// interrupt number. Depending on the
func interruptGateEntries()
