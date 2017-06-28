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
	INVLPG virtAddr+0(FP)
	RET

TEXT ·SwitchPDT(SB),NOSPLIT,$0
	// loading CR3 also triggers a TLB flush
	MOVQ pdtPhysAddr+0(FP), CR3
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
