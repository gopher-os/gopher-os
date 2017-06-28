package cpu

var (
	cpuidFn = ID
)

// EnableInterrupts enables interrupt handling.
func EnableInterrupts()

// DisableInterrupts disables interrupt handling.
func DisableInterrupts()

// Halt stops instruction execution.
func Halt()

// FlushTLBEntry flushes a TLB entry for a particular virtual address.
func FlushTLBEntry(virtAddr uintptr)

// SwitchPDT sets the root page table directory to point to the specified
// physical address and flushes the TLB.
func SwitchPDT(pdtPhysAddr uintptr)

// ActivePDT returns the physical address of the currently active page table.
func ActivePDT() uintptr

// ReadCR2 returns the value stored in the CR2 register.
func ReadCR2() uint64

// ID returns information about the CPU and its features. It
// is implemented as a CPUID instruction with EAX=leaf and
// returns the values in EAX, EBX, ECX and EDX.
func ID(leaf uint32) (uint32, uint32, uint32, uint32)

// IsIntel returns true if the code is running on an Intel processor.
func IsIntel() bool {
	_, ebx, ecx, edx := cpuidFn(0)
	return ebx == 0x756e6547 && // "Genu"
		edx == 0x49656e69 && // "ineI"
		ecx == 0x6c65746e // "ntel"
}
