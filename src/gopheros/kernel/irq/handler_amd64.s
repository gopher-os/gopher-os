#include "textflag.h"

// The maximum number of interrupt handlers is 256 so we need to allocate space
// for 256 x 8-byte pointers. This symbol is made global by the Makefile so it
// can be accessed by the gate entries defined in the rt0 assembly code.
GLOBL _rt0_interrupt_handlers(SB), NOPTR, $2048

// In 64-bit mode SIDT stores 8+2 bytes for the IDT address and limit
GLOBL _rt0_idtr<>(SB), NOPTR, $10

TEXT ·HandleException(SB),NOSPLIT,$0
	JMP ·HandleExceptionWithCode(SB)
	RET

TEXT ·HandleExceptionWithCode(SB),NOSPLIT,$0
	// Install the handler address in _rt0_interrupt_handlers
	LEAQ _rt0_interrupt_handlers+0(SB), CX
	MOVBQZX exceptionNum+0(FP), AX // exceptionNum is a uint8 so we zero-extend it to 64bits
	MOVQ handler+8(FP), BX
	MOVQ 0(BX), BX      // dereference pointer to handler fn
	MOVQ BX, (CX)(AX*8)
	
	// To enable the handler we need to lookup the appropriate IDT entry
	// and modify its type/attribute byte. To acquire the IDT base address
	// we use the SIDT instruction.
	MOVQ IDTR, _rt0_idtr<>+0(SB)
	LEAQ _rt0_idtr<>(SB), CX
	MOVQ 2(CX), CX     // CX points to IDT base address
	SHLQ $4, AX        // Each IDT entry uses 16 bytes so we multiply num by 16
	ADDQ AX, CX        // and add it to CX to get the address of the IDT entry 
			   // we want to tweak
	
	MOVB $0x8e, 5(CX)  // 32/64-bit ring-0 interrupt gate that is present
	                   // see: http://wiki.osdev.org/Interrupt_Descriptor_Table

	RET
