#include "textflag.h"

TEXT ·EnableInterrupts(SB),NOSPLIT,$0
	STI
	RET

TEXT ·DisableInterrupts(SB),NOSPLIT,$0
	CLI
	RET

TEXT ·Halt(SB),NOSPLIT,$0
	CLI
	HLT
	RET

TEXT ·FlushTLBEntry(SB),NOSPLIT,$0
	MOVQ virtAddr+0(FP), AX
	INVLPG (AX)
	RET

TEXT ·SwitchPDT(SB),NOSPLIT,$0
	// loading CR3 also triggers a TLB flush
	MOVQ pdtPhysAddr+0(FP), AX
	MOVQ AX, CR3
	RET

TEXT ·ActivePDT(SB),NOSPLIT,$0
	MOVQ CR3, AX
	MOVQ AX, ret+0(FP)
	RET

TEXT ·ReadCR2(SB),NOSPLIT,$0
	MOVQ CR2, AX
	MOVQ AX, ret+0(FP)
	RET

TEXT ·ID(SB),NOSPLIT,$0
	MOVQ leaf+0(FP), AX
	CPUID
	MOVL AX, ret+0(FP)
	MOVL BX, ret+4(FP)
	MOVL CX, ret+8(FP)
	MOVL DX, ret+12(FP)
	RET

TEXT ·PortWriteByte(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	MOVB val+2(FP), AX
	BYTE $0xee // out al, dx
	RET

TEXT ·PortWriteWord(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	MOVW val+2(FP), AX
	BYTE $0x66 
	BYTE $0xef  // out ax, dx
	RET

TEXT ·PortWriteDword(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	MOVL val+2(FP), AX
	BYTE $0xef  // out eax, dx
	RET

TEXT ·PortReadByte(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	BYTE $0xec  // in al, dx
	MOVB AX, ret+0(FP)
	RET

TEXT ·PortReadWord(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	BYTE $0x66  
	BYTE $0xed  // in ax, dx
	MOVW AX, ret+0(FP)
	RET

TEXT ·PortReadDword(SB),NOSPLIT,$0
	MOVW port+0(FP), DX
	BYTE $0xed  // in eax, dx
	MOVL AX, ret+0(FP)
	RET
