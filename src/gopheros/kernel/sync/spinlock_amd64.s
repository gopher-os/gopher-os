#include "textflag.h"

TEXT ·archAcquireSpinlock(SB),NOSPLIT,$0-12
	MOVQ state+0(FP), AX
	MOVL attemptsBeforeYielding+8(FP), CX

try_acquire:
	MOVL $1, BX
	XCHGL 0(AX), BX
	TESTL BX, BX
	JNZ spin

	// Lock succesfully acquired 
	RET

spin:
	// Send hint to the CPU that we are in a spinlock loop
	PAUSE

	// Do a dirty read to check the state and try to acquire the lock 
	// once we detect it is free 
	MOVL 0(AX), BX
	TESTL BX, BX
	JZ try_acquire

	// Keep retrying till we exceed attemptsBeforeYielding; this allows us 
	// to grab the lock if a task on another CPU releases the lock while we 
	// spin.
	DECL CX
	JNZ spin

	// Yield (if yieldFn is set) and spin again
	MOVQ ·yieldFn+0(SB), AX
	TESTQ AX, AX
	JZ replenish_attempt_counter
	CALL 0(AX)

replenish_attempt_counter:
	MOVQ state+0(FP), AX
	MOVL attemptsBeforeYielding+8(FP), CX
	JMP spin
