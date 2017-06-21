#include "textflag.h"

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

