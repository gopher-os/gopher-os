#include "textflag.h"

#define NUM_IDT_ENTRIES 256
#define IDT_ENTRY_SIZE 16
#define IDT_ENTRY_SIZE_SHIFT 4

#define ENTRY_TYPE_INTERRUPT_GATE 0x8e

// The 64-bit SIDT consists of 10 bytes and has the following layout:
//   BYTE
// [00 - 01] size of IDT minus 1
// [02 - 09] address of the IDT
GLOBL ·idtDescriptor<>(SB), NOPTR, $10

// The 64-bit IDT consists of NUM_IDT_ENTRIES slots containing 16-byte entries
// with the following layout:
//   BYTE
// [00 - 01] 64-bit gate entry address (bits 0:15)
// [02 - 03] CS selector
// [04 - 04] interrupt stack table offset (bits 0:2)
// [05 - 05] gate type and attributes
// [06 - 07] 64-bit gate entry address (bits 16:31)
// [08 - 11] 64-bit gate entry address (bits 32:63)
// [12 - 15] reserved
GLOBL ·idt<>(SB), NOPTR, $NUM_IDT_ENTRIES*IDT_ENTRY_SIZE

// A list of 256 function pointers for installed gate handlers. These pointers 
// serve as the jump targets for the trap/int/task dispatchers.
GLOBL ·gateHandlers<>(SB), NOPTR, $NUM_IDT_ENTRIES*8

// installIDT populates idtDescriptor with the address of IDT and loads it to 
// the CPU. All gate entries are initially marked as non-present and must be 
// explicitly enabled by invoking HandleInterrupt.
TEXT ·installIDT(SB),NOSPLIT,$0
	LEAQ ·idtDescriptor<>(SB), AX
	MOVW $(NUM_IDT_ENTRIES*IDT_ENTRY_SIZE)-1, 0(AX)
	LEAQ ·idt<>(SB), BX
	MOVQ BX, 2(AX)
	MOVQ 0(AX), IDTR 	// LIDT[RAX]
	RET

// HandleInterrupt ensures that the provided handler will be invoked when a
// particular interrupt number occurs. The value of the istOffset argument
// specifies the offset in the interrupt stack table (if 0 then IST is not
// used).
TEXT ·HandleInterrupt(SB),NOSPLIT,$0-10
	MOVBQZX intNumber+0(FP), CX 
	
	// Dereference pointer to trap handler and copy it into gateHandlers
	MOVQ handler+8(FP), BX
	MOVQ 0(BX), BX 
	LEAQ ·gateHandlers<>+0(SB), DI
	MOVQ BX, (DI)(CX*8)
	
	// Calculate IDT entry address
	LEAQ ·idt<>+0(SB), DI
	MOVQ CX, BX
	SHLQ $IDT_ENTRY_SIZE_SHIFT, BX
	ADDQ BX, DI

	// The trap gate entries have variable lengths depending on whether 
	// the CPU pushes an exception code or not. Each generated entry ends
	// with a sequence of 4 NOPs (0x90). The code below uses this information
	// to locate the correct entry point address.
	LEAQ ·interruptGateEntries(SB), SI // SI points to entry for trap 0
check_next_entry:
	TESTB CX, CX
	JZ update_idt_entry

find_nop_delimiter:
	INCQ SI
	CMPL 0(SI), $0x90909090
	JNE find_nop_delimiter

	// SI points to the 4xNOP delimiter start 
	ADDQ $4, SI
	DECB CX
	JMP check_next_entry

update_idt_entry:
	// IDT entry layout (bytes)
	// ------------------------
	// [00-01] bits 0-15 of 64-bit handler address
	// [02-03] CS selector 
	// [04-04] interrupt stack table offset (IST) 
	// [05-05] gate type/attributes
	// [06-07] bits 16-31 of 64-bit handler address 
	// [08-11] bits 32-63 of 64-bit handler address
	// [12-15] reserved
	//-------------------------

	// Mark entry as non-present while updating the handler address 
	MOVB $0, 5(DI)

	// Use the kernel CS selector from the rt0-loaded GDT and use the 
	// specified IST offset
	MOVW $0x8, 2(DI)
	MOVB istOffset+1(FP), AX
	MOVB AX, 4(DI)

	// Copy the entrypoint address from SI
	MOVW SI, 0(DI)
	SHRQ $16, SI
	MOVW SI, 6(DI)
	SHRQ $16, SI
	MOVL SI, 8(DI)

	// Mark entry as a present, 32-bit interrupt gate
	MOVB $ENTRY_TYPE_INTERRUPT_GATE, 5(DI)

	RET

// Emit interrupt dispatching code for traps where the CPU pushes an exception
// code to the stack. The code below just pushes the handler's address to the
// stack and jumps to dispatchInterrupt. 
//
// This code uses some tricks to bypass Go assembler limitations:
// - replace PUSH with: SUBQ $8, RSP; MOVQ X, 0(RSP). This prevents the Go 
//   assembler from complaining about unbalanced PUSH/POP statements.
// - use a PUSH/RET (0xc3 byte instead of RET mnemonic) trick instead of a 
//   "JMP dispatchInterrupt" to prevent the optimizer from optimizing away all 
//   but the first entry in interruptGateEntries.
//
// Finally, each entry block ends with a series of 4 NOP instructions. This 
// delimiter is used by the HandleInterrupt implementation to locate the correct 
// entrypoint address for a particular interrupt.
#define INT_ENTRY_WITH_CODE(num) \
	SUBQ $16, SP;                          \
	MOVQ R15, 0(SP);                       \
	MOVQ ·gateHandlers<>+8*num(SB), R15;   \
	MOVQ R15, 8(SP);                       \
	LEAQ ·dispatchInterrupt(SB), R15;      \
	XCHGQ R15, 0(SP);                      \
	BYTE $0xc3;                            \
	BYTE $0x90; BYTE $0x90; BYTE $0x90; BYTE $0x90;

// Emit interrupt dispatching code for traps where the CPU does not push an
// exception code to the stack. The implementation is identical with the
// INT_ENTRY_WITH_CODE above with the exception that the interrupt number is
// manually pushed to the stack before the handler address so both entry
// variants can use the same dispatching code.
#define INT_ENTRY_WITHOUT_CODE(num) \
	SUBQ $24, SP;                          \
	MOVQ R15, 0(SP);                       \
	MOVQ ·gateHandlers<>+8*num(SB), R15;   \
	MOVQ R15, 8(SP);                       \
	MOVQ $num, 16(SP);                     \
	LEAQ ·dispatchInterrupt(SB), R15;      \
	XCHGQ R15, 0(SP);                      \
	BYTE $0xc3;                            \
	BYTE $0x90; BYTE $0x90; BYTE $0x90; BYTE $0x90;

// dispatchInterrupt is invoked by the interrupt gate entrypoints to route 
// an incoming interrupt to the selected handler.
//
// Callers MUST ensure that the stack has the following layout before calling 
// dispatchInterrupt:
//
// |-----------------| <=== SP after jumping to dispatchInterrupt
// | handler address | <- pushed by the interrupt entry code
// |-----------------|
// | exception code  | <- pushed by CPU or a dummy code pushed by the gate entry
// |-----------------|
// | RIP             | <- pushed by CPU (exception frame)
// | CS              |
// | RFLAGS          |
// | RSP             |
// | SS              |
// |-----------------|
//
// Once the handler returns, the GP regs are restored and the stack is unwinded
// so that the CPU can resume excecution of the code that triggered the
// interrupt.
//
// Interrupts are automatically disabled by the CPU upon entry and re-enabled
// when this function returns.
//--------------------------- -----------------------------------------
TEXT ·dispatchInterrupt(SB),NOSPLIT,$0
	// Save GP regs. The push order MUST match the field layout in the 
	// Registers struct.
	XCHGQ R15, 0(SP) // Swap handler address on stack with R15 contents
	PUSHQ R14
	PUSHQ R13
	PUSHQ R12
	PUSHQ R11
	PUSHQ R10
	PUSHQ R9
	PUSHQ R8
	PUSHQ BP 
	PUSHQ DI 
	PUSHQ SI 
	PUSHQ DX
	PUSHQ CX 
	PUSHQ BX 
	PUSHQ AX

	// Save XMM regs; the amd64 Go runtime uses SSE instructions to implement 
	// functionality such as memmove which may trigger page faults (e.g
	// when resizing a slice and copying the data to the new location). As
	// the registered handler may clobber the xmm regs we need to save them 
	// here and restore them once the handler returns.
	SUBQ $16*16, SP
	MOVOU X0, 0*16(SP)
	MOVOU X1, 1*16(SP)
	MOVOU X2, 2*16(SP)
	MOVOU X3, 3*16(SP)
	MOVOU X4, 4*16(SP)
	MOVOU X5, 5*16(SP)
	MOVOU X6, 6*16(SP)
	MOVOU X7, 7*16(SP)
	MOVOU X8, 8*16(SP)
	MOVOU X9, 9*16(SP)
	MOVOU X10, 10*16(SP)
	MOVOU X11, 11*16(SP)
	MOVOU X12, 12*16(SP)
	MOVOU X13, 13*16(SP)
	MOVOU X14, 14*16(SP)
	MOVOU X15, 15*16(SP)

	// Setup call stack and invoke handler 
	MOVQ SP, R14
	ADDQ $16*16, R14
	PUSHQ R14
	CALL R15
	ADDQ $8, SP

	// Restore XMM regs
	MOVOU 0*16(SP), X0
	MOVOU 1*16(SP), X1
	MOVOU 2*16(SP), X2
	MOVOU 3*16(SP), X3
	MOVOU 4*16(SP), X4
	MOVOU 5*16(SP), X5
	MOVOU 6*16(SP), X6
	MOVOU 7*16(SP), X7
	MOVOU 8*16(SP), X8
	MOVOU 9*16(SP), X9
	MOVOU 10*16(SP), X10
	MOVOU 11*16(SP), X11
	MOVOU 12*16(SP), X12
	MOVOU 13*16(SP), X13
	MOVOU 14*16(SP), X14
	MOVOU 15*16(SP), X15
	ADDQ $16*16, SP

	// Restore GP regs
	POPQ AX 
	POPQ BX 
	POPQ CX 
	POPQ DX
	POPQ SI
	POPQ DI
	POPQ BP
	POPQ R8
	POPQ R9
	POPQ R10
	POPQ R11
	POPQ R12
	POPQ R13
	POPQ R14
	POPQ R15
	
	// Handler must manually pop the exception (real or dummy) from the stack 
	// before returning; interrupts will be automatically enabled by the 
	// CPU upon returning.
	ADDQ $8, SP
	IRETQ

// interruptGateEntries contains a list of generated entries for each possible
// interrupt number. Depending on the 
TEXT ·interruptGateEntries(SB),NOSPLIT,$0
	// For a list of gate numbers that push an error code see: http://wiki.osdev.org/Exceptions
	INT_ENTRY_WITHOUT_CODE(0) INT_ENTRY_WITHOUT_CODE(1) INT_ENTRY_WITHOUT_CODE(2) INT_ENTRY_WITHOUT_CODE(3) INT_ENTRY_WITHOUT_CODE(4) INT_ENTRY_WITHOUT_CODE(5) INT_ENTRY_WITHOUT_CODE(6) INT_ENTRY_WITHOUT_CODE(7) 
	INT_ENTRY_WITH_CODE(8)
	INT_ENTRY_WITHOUT_CODE(9)
	INT_ENTRY_WITH_CODE(10) INT_ENTRY_WITH_CODE(11) INT_ENTRY_WITH_CODE(12) INT_ENTRY_WITH_CODE(13) INT_ENTRY_WITH_CODE(14)
	INT_ENTRY_WITHOUT_CODE(15) INT_ENTRY_WITHOUT_CODE(16)
	INT_ENTRY_WITH_CODE(17)
	INT_ENTRY_WITHOUT_CODE(18) INT_ENTRY_WITHOUT_CODE(19) INT_ENTRY_WITHOUT_CODE(20) INT_ENTRY_WITHOUT_CODE(21) INT_ENTRY_WITHOUT_CODE(22) INT_ENTRY_WITHOUT_CODE(23) INT_ENTRY_WITHOUT_CODE(24) INT_ENTRY_WITHOUT_CODE(25) INT_ENTRY_WITHOUT_CODE(26) INT_ENTRY_WITHOUT_CODE(27) INT_ENTRY_WITHOUT_CODE(28) INT_ENTRY_WITHOUT_CODE(29)
	INT_ENTRY_WITH_CODE(30)
	INT_ENTRY_WITHOUT_CODE(31) INT_ENTRY_WITHOUT_CODE(32) INT_ENTRY_WITHOUT_CODE(33) INT_ENTRY_WITHOUT_CODE(34) INT_ENTRY_WITHOUT_CODE(35) INT_ENTRY_WITHOUT_CODE(36) INT_ENTRY_WITHOUT_CODE(37) INT_ENTRY_WITHOUT_CODE(38) INT_ENTRY_WITHOUT_CODE(39) INT_ENTRY_WITHOUT_CODE(40) INT_ENTRY_WITHOUT_CODE(41) INT_ENTRY_WITHOUT_CODE(42)
	INT_ENTRY_WITHOUT_CODE(43) INT_ENTRY_WITHOUT_CODE(44) INT_ENTRY_WITHOUT_CODE(45) INT_ENTRY_WITHOUT_CODE(46) INT_ENTRY_WITHOUT_CODE(47) INT_ENTRY_WITHOUT_CODE(48) INT_ENTRY_WITHOUT_CODE(49) INT_ENTRY_WITHOUT_CODE(50) INT_ENTRY_WITHOUT_CODE(51) INT_ENTRY_WITHOUT_CODE(52) INT_ENTRY_WITHOUT_CODE(53) INT_ENTRY_WITHOUT_CODE(54)
	INT_ENTRY_WITHOUT_CODE(55) INT_ENTRY_WITHOUT_CODE(56) INT_ENTRY_WITHOUT_CODE(57) INT_ENTRY_WITHOUT_CODE(58) INT_ENTRY_WITHOUT_CODE(59) INT_ENTRY_WITHOUT_CODE(60) INT_ENTRY_WITHOUT_CODE(61) INT_ENTRY_WITHOUT_CODE(62) INT_ENTRY_WITHOUT_CODE(63) INT_ENTRY_WITHOUT_CODE(64) INT_ENTRY_WITHOUT_CODE(65) INT_ENTRY_WITHOUT_CODE(66)
	INT_ENTRY_WITHOUT_CODE(67) INT_ENTRY_WITHOUT_CODE(68) INT_ENTRY_WITHOUT_CODE(69) INT_ENTRY_WITHOUT_CODE(70) INT_ENTRY_WITHOUT_CODE(71) INT_ENTRY_WITHOUT_CODE(72) INT_ENTRY_WITHOUT_CODE(73) INT_ENTRY_WITHOUT_CODE(74) INT_ENTRY_WITHOUT_CODE(75) INT_ENTRY_WITHOUT_CODE(76) INT_ENTRY_WITHOUT_CODE(77) INT_ENTRY_WITHOUT_CODE(78)
	INT_ENTRY_WITHOUT_CODE(79) INT_ENTRY_WITHOUT_CODE(80) INT_ENTRY_WITHOUT_CODE(81) INT_ENTRY_WITHOUT_CODE(82) INT_ENTRY_WITHOUT_CODE(83) INT_ENTRY_WITHOUT_CODE(84) INT_ENTRY_WITHOUT_CODE(85) INT_ENTRY_WITHOUT_CODE(86) INT_ENTRY_WITHOUT_CODE(87) INT_ENTRY_WITHOUT_CODE(88) INT_ENTRY_WITHOUT_CODE(89) INT_ENTRY_WITHOUT_CODE(90)
	INT_ENTRY_WITHOUT_CODE(91) INT_ENTRY_WITHOUT_CODE(92) INT_ENTRY_WITHOUT_CODE(93) INT_ENTRY_WITHOUT_CODE(94) INT_ENTRY_WITHOUT_CODE(95) INT_ENTRY_WITHOUT_CODE(96) INT_ENTRY_WITHOUT_CODE(97) INT_ENTRY_WITHOUT_CODE(98) INT_ENTRY_WITHOUT_CODE(99) INT_ENTRY_WITHOUT_CODE(100) INT_ENTRY_WITHOUT_CODE(101) INT_ENTRY_WITHOUT_CODE(102)
	INT_ENTRY_WITHOUT_CODE(103) INT_ENTRY_WITHOUT_CODE(104) INT_ENTRY_WITHOUT_CODE(105) INT_ENTRY_WITHOUT_CODE(106) INT_ENTRY_WITHOUT_CODE(107) INT_ENTRY_WITHOUT_CODE(108) INT_ENTRY_WITHOUT_CODE(109) INT_ENTRY_WITHOUT_CODE(110) INT_ENTRY_WITHOUT_CODE(111) INT_ENTRY_WITHOUT_CODE(112) INT_ENTRY_WITHOUT_CODE(113) INT_ENTRY_WITHOUT_CODE(114)
	INT_ENTRY_WITHOUT_CODE(115) INT_ENTRY_WITHOUT_CODE(116) INT_ENTRY_WITHOUT_CODE(117) INT_ENTRY_WITHOUT_CODE(118) INT_ENTRY_WITHOUT_CODE(119) INT_ENTRY_WITHOUT_CODE(120) INT_ENTRY_WITHOUT_CODE(121) INT_ENTRY_WITHOUT_CODE(122) INT_ENTRY_WITHOUT_CODE(123) INT_ENTRY_WITHOUT_CODE(124) INT_ENTRY_WITHOUT_CODE(125) INT_ENTRY_WITHOUT_CODE(126)
	INT_ENTRY_WITHOUT_CODE(127) INT_ENTRY_WITHOUT_CODE(128) INT_ENTRY_WITHOUT_CODE(129) INT_ENTRY_WITHOUT_CODE(130) INT_ENTRY_WITHOUT_CODE(131) INT_ENTRY_WITHOUT_CODE(132) INT_ENTRY_WITHOUT_CODE(133) INT_ENTRY_WITHOUT_CODE(134) INT_ENTRY_WITHOUT_CODE(135) INT_ENTRY_WITHOUT_CODE(136) INT_ENTRY_WITHOUT_CODE(137) INT_ENTRY_WITHOUT_CODE(138)
	INT_ENTRY_WITHOUT_CODE(139) INT_ENTRY_WITHOUT_CODE(140) INT_ENTRY_WITHOUT_CODE(141) INT_ENTRY_WITHOUT_CODE(142) INT_ENTRY_WITHOUT_CODE(143) INT_ENTRY_WITHOUT_CODE(144) INT_ENTRY_WITHOUT_CODE(145) INT_ENTRY_WITHOUT_CODE(146) INT_ENTRY_WITHOUT_CODE(147) INT_ENTRY_WITHOUT_CODE(148) INT_ENTRY_WITHOUT_CODE(149) INT_ENTRY_WITHOUT_CODE(150)
	INT_ENTRY_WITHOUT_CODE(151) INT_ENTRY_WITHOUT_CODE(152) INT_ENTRY_WITHOUT_CODE(153) INT_ENTRY_WITHOUT_CODE(154) INT_ENTRY_WITHOUT_CODE(155) INT_ENTRY_WITHOUT_CODE(156) INT_ENTRY_WITHOUT_CODE(157) INT_ENTRY_WITHOUT_CODE(158) INT_ENTRY_WITHOUT_CODE(159) INT_ENTRY_WITHOUT_CODE(160) INT_ENTRY_WITHOUT_CODE(161) INT_ENTRY_WITHOUT_CODE(162)
	INT_ENTRY_WITHOUT_CODE(163) INT_ENTRY_WITHOUT_CODE(164) INT_ENTRY_WITHOUT_CODE(165) INT_ENTRY_WITHOUT_CODE(166) INT_ENTRY_WITHOUT_CODE(167) INT_ENTRY_WITHOUT_CODE(168) INT_ENTRY_WITHOUT_CODE(169) INT_ENTRY_WITHOUT_CODE(170) INT_ENTRY_WITHOUT_CODE(171) INT_ENTRY_WITHOUT_CODE(172) INT_ENTRY_WITHOUT_CODE(173) INT_ENTRY_WITHOUT_CODE(174)
	INT_ENTRY_WITHOUT_CODE(175) INT_ENTRY_WITHOUT_CODE(176) INT_ENTRY_WITHOUT_CODE(177) INT_ENTRY_WITHOUT_CODE(178) INT_ENTRY_WITHOUT_CODE(179) INT_ENTRY_WITHOUT_CODE(180) INT_ENTRY_WITHOUT_CODE(181) INT_ENTRY_WITHOUT_CODE(182) INT_ENTRY_WITHOUT_CODE(183) INT_ENTRY_WITHOUT_CODE(184) INT_ENTRY_WITHOUT_CODE(185) INT_ENTRY_WITHOUT_CODE(186)
	INT_ENTRY_WITHOUT_CODE(187) INT_ENTRY_WITHOUT_CODE(188) INT_ENTRY_WITHOUT_CODE(189) INT_ENTRY_WITHOUT_CODE(190) INT_ENTRY_WITHOUT_CODE(191) INT_ENTRY_WITHOUT_CODE(192) INT_ENTRY_WITHOUT_CODE(193) INT_ENTRY_WITHOUT_CODE(194) INT_ENTRY_WITHOUT_CODE(195) INT_ENTRY_WITHOUT_CODE(196) INT_ENTRY_WITHOUT_CODE(197) INT_ENTRY_WITHOUT_CODE(198)
	INT_ENTRY_WITHOUT_CODE(199) INT_ENTRY_WITHOUT_CODE(200) INT_ENTRY_WITHOUT_CODE(201) INT_ENTRY_WITHOUT_CODE(202) INT_ENTRY_WITHOUT_CODE(203) INT_ENTRY_WITHOUT_CODE(204) INT_ENTRY_WITHOUT_CODE(205) INT_ENTRY_WITHOUT_CODE(206) INT_ENTRY_WITHOUT_CODE(207) INT_ENTRY_WITHOUT_CODE(208) INT_ENTRY_WITHOUT_CODE(209) INT_ENTRY_WITHOUT_CODE(210)
	INT_ENTRY_WITHOUT_CODE(211) INT_ENTRY_WITHOUT_CODE(212) INT_ENTRY_WITHOUT_CODE(213) INT_ENTRY_WITHOUT_CODE(214) INT_ENTRY_WITHOUT_CODE(215) INT_ENTRY_WITHOUT_CODE(216) INT_ENTRY_WITHOUT_CODE(217) INT_ENTRY_WITHOUT_CODE(218) INT_ENTRY_WITHOUT_CODE(219) INT_ENTRY_WITHOUT_CODE(220) INT_ENTRY_WITHOUT_CODE(221) INT_ENTRY_WITHOUT_CODE(222)
	INT_ENTRY_WITHOUT_CODE(223) INT_ENTRY_WITHOUT_CODE(224) INT_ENTRY_WITHOUT_CODE(225) INT_ENTRY_WITHOUT_CODE(226) INT_ENTRY_WITHOUT_CODE(227) INT_ENTRY_WITHOUT_CODE(228) INT_ENTRY_WITHOUT_CODE(229) INT_ENTRY_WITHOUT_CODE(230) INT_ENTRY_WITHOUT_CODE(231) INT_ENTRY_WITHOUT_CODE(232) INT_ENTRY_WITHOUT_CODE(233) INT_ENTRY_WITHOUT_CODE(234)
	INT_ENTRY_WITHOUT_CODE(235) INT_ENTRY_WITHOUT_CODE(236) INT_ENTRY_WITHOUT_CODE(237) INT_ENTRY_WITHOUT_CODE(238) INT_ENTRY_WITHOUT_CODE(239) INT_ENTRY_WITHOUT_CODE(240) INT_ENTRY_WITHOUT_CODE(241) INT_ENTRY_WITHOUT_CODE(242) INT_ENTRY_WITHOUT_CODE(243) INT_ENTRY_WITHOUT_CODE(244) INT_ENTRY_WITHOUT_CODE(245) INT_ENTRY_WITHOUT_CODE(246)
	INT_ENTRY_WITHOUT_CODE(247) INT_ENTRY_WITHOUT_CODE(248) INT_ENTRY_WITHOUT_CODE(249) INT_ENTRY_WITHOUT_CODE(250) INT_ENTRY_WITHOUT_CODE(251) INT_ENTRY_WITHOUT_CODE(252) INT_ENTRY_WITHOUT_CODE(253) INT_ENTRY_WITHOUT_CODE(254) INT_ENTRY_WITHOUT_CODE(255)
	RET

