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

