; vim: set ft=nasm :
%include "constants.inc"

bits 64

section .bss
align 8

; Allocate space for the interrupt descriptor table (IDT).
; This arch supports up to 256 interrupt handlers
%define IDT_ENTRIES 0xff
_rt0_idt_start:
	resq 2 * IDT_ENTRIES ; each 64-bit IDT entry is 16 bytes
_rt0_idt_end:

_rt0_idt_desc:
	resw 1
	resq 1

; Allocates space for the IRQ handlers pointers registered by the IRQ package 
_rt0_irq_handlers resq IDT_ENTRIES

r0_g_ptr:	  resq 1 ; fs:0x00 is a pointer to the current g struct
r0_g:
r0_g_stack_lo:    resq 1
r0_g_stack_hi:    resq 1
r0_g_stackguard0: resq 1  ; rsp compared to this value in go stack growth prologue
r0_g_stackguard1: resq 1  ; rsp compared to this value in C stack growth prologue

section .text 

;------------------------------------------------------------------------------
; Kernel 64-bit entry point
;
; The 32-bit entrypoint code jumps to this entrypoint after:
; - it has entered long mode and enabled paging
; - it has loaded a 64bit GDT
; - it has set up identity paging for the physical 0-8M region and the
;   PAGE_OFFSET to PAGE_OFFSET+8M region. 
;------------------------------------------------------------------------------
global _rt0_64_entry
_rt0_64_entry:
	call _rt0_64_load_idt

	; According to the x86_64 ABI, the fs:0 should point to the address of 
	; the user-space thread structure. The actual TLS structure is located 
	; just before that (aligned). Go code tries to fetch the address to the 
	; active go-routine's g struct by accessing fs:-8. What we need to do 
	; is to setup a mock g0 struct, populate its stack_lo/hi/guard fields 
	; and then use wrmsr to update the FS register
	extern stack_top 
	extern stack_bottom
	
	; Setup r0_g
	mov rax, stack_bottom
	mov rbx, stack_top
	mov rsi, r0_g
	mov qword [rsi+0], rax   ; stack_lo
	mov qword  [rsi+8], rbx  ; stack_hi
	mov qword [rsi+16], rax  ; stackguard0
	mov rax, r0_g_ptr
	mov qword [rax], rsi
	
	; Load 64-bit FS register address 
	; rax -> lower 32 bits 
	; rdx -> upper 32 bits
	mov ecx, 0xc0000100  ; fs_base
	mov rax, rsi         ; lower 32 bits 
	shr rsi, 32
	mov rdx, rsi         ; high 32 bits
	wrmsr

	; Call the kernel entry point passing a pointer to the multiboot data
	; copied by the 32-bit entry code
	extern multiboot_data
	extern _kernel_start
	extern _kernel_end
	extern kernel.Kmain
	
	mov rax, _kernel_end - PAGE_OFFSET
	push rax
	mov rax, _kernel_start - PAGE_OFFSET
	push rax
	mov rax, multiboot_data
	push rax
	call kernel.Kmain
	
	; Main should never return; halt the CPU
	mov rdi, err_kmain_returned
	call write_string

	cli
	hlt


;------------------------------------------------------------------------------
; Setup and load IDT. We preload each IDT entry with a pointer to a gate handler 
; but set it as inactive. The code in irq_amd64 is responsible for enabling 
; individual IDT entries when handlers are installed.
;------------------------------------------------------------------------------
_rt0_64_load_idt:
	mov rax, _rt0_idt_start

%assign gate_num 0 
%rep    IDT_ENTRIES
	mov rbx, _rt0_64_gate_entry_%+ gate_num
	mov word [rax], bx        ; gate entry bits 0-15
	mov word [rax+2], 0x8     ; GDT descriptor
	mov byte [rax+5], 0x0     ; Mark the entry as NOT present 
	shr rbx, 16
	mov word [rax+6], bx      ; gate entry bits 16-31
	shr rbx, 16
	mov dword [rax+8], ebx    ; gate entry bits 32-63

	add rax, 16		  ; size of IDT entry
%assign gate_num gate_num+1 
%endrep
	mov rax, _rt0_idt_desc
	mov word [rax], _rt0_idt_end - _rt0_idt_start - 1 ; similar to GDT this must be len(IDT) - 1
	mov rbx, _rt0_idt_start
	mov qword [rax+2], rbx
	lidt [rax]
ret


;------------------------------------------------------------------------------
; Generate gate entries. Each gate handler pushes the address of the registered 
; handler to the stack before jumping to a dispatcher function.
;
; Some exceptions push an error code to the stack after the stack frame. This
; code must be popped off the stack before calling iretq. The generated handlers 
; are aware whether they need to deal with the code or not and jump to the 
; appropriate get dispatcher.
;------------------------------------------------------------------------------
%assign gate_num 0 
%rep    IDT_ENTRIES
extern _rt0_interrupt_handlers
_rt0_64_gate_entry_%+ gate_num:
	push rax
	mov rax, _rt0_interrupt_handlers
	add rax, 8*gate_num
	mov rax, [rax]
	xchg rax, [rsp]	; store handler address and restore original rax

	; For a list of gate numbers that push an error code see:
	; http://wiki.osdev.org/Exceptions
	%if (gate_num == 8) || (gate_num >= 10 && gate_num <= 14) || (gate_num == 17) || (gate_num == 30)
		jmp _rt0_64_gate_dispatcher_with_code
	%else
		jmp _rt0_64_gate_dispatcher_without_code
	%endif
%assign gate_num gate_num+1 
%endrep

%macro save_regs 0
	push r15 
	push r14
	push r13 
	push r12
	push r11
	push r10
	push r9
	push r8
	push rbp
	push rdi
	push rsi 
	push rdx 
	push rcx 
	push rbx 
	push rax
%endmacro

%macro restore_regs 0
	pop rax
	pop rbx
	pop rcx
	pop rdx
	pop rsi 
	pop rdi
	pop rbp 
	pop r8
	pop r9
	pop r10
	pop r11 
	pop r12
	pop r13
	pop r14
	pop r15
%endmacro

;------------------------------------------------------------------------------
; This dispatcher is invoked by gate entries that expect a code to be pushed 
; by the CPU to the stack. It performs the following functions:
; - save registers
; - push pointer to saved regs 
; - push pointer to stack frame 
; - read and push exception code 
; - invoke handler(code, &frame, &regs)
; - restore registers 
; - pop exception code from stack so rsp points to the stack frame
;------------------------------------------------------------------------------
_rt0_64_gate_dispatcher_with_code:
	; This is how the stack looks like when entering this function:
	; (each item is 8-bytes wide)
	;
	;------------------
	; handler address | <- pushed by gate_entry_xxx (RSP points here)
	;-----------------|
	; Exception code  | <- needs to be removed from stack before calling iretq
	;-----------------|
	; RIP             | <- exception frame             
	; CS              |
	; RFLAGS          |
	; RSP             |
	; SS              |
	;-----------------
	cld

	; save regs and push a pointer to them 
	save_regs
	mov rax, rsp   ; rax points to saved rax
	push rax       ; push pointer to saved regs

	; push pointer to exception stack frame (we have used 15 qwords for the 
	; saved registers plus one qword for the data pushed by the gate entry
	; plus one extra qword to jump over the exception code)
	add rax, 17*8
	push rax

	; push exception code (located between the stack frame and the saved regs)
	sub rax, 8
	push qword [rax]

	call [rsp + 18*8] ; call registered irq handler

	add rsp, 3 * 8    ; unshift the pushed arguments so rsp points to the saved regs 
	restore_regs

	add rsp, 16	  ; pop handler address and exception code off the stack before returning
	iretq


;------------------------------------------------------------------------------
; This dispatcher is invoked by gate entries that do not use exception codes.
; It performs the following functions:
; - save registers
; - push pointer to saved regs 
; - push pointer to stack frame 
; - invoke handler(&frame, &regs)
; - restore registers 
;------------------------------------------------------------------------------
_rt0_64_gate_dispatcher_without_code:
	; This is how the stack looks like when entering this function:
	; (each item is 8-bytes wide)
	;
	;------------------
	; handler address | <- pushed by gate_entry_xxx (RSP points here)
	;-----------------|
	; RIP             | <- exception frame             
	; CS              |
	; RFLAGS          |
	; RSP             |
	; SS              |
	;-----------------
	cld

	; save regs and push a pointer to them 
	save_regs
	mov rax, rsp   ; rax points to saved rax
	push rax       ; push pointer to saved regs

	; push pointer to exception stack frame (we have used 15 qwords for the 
	; saved registers plus one qword for the data pushed by the gate entry)
	add rax, 16*8
	push rax

	call [rsp + 17*8] ; call registered irq handler

	add rsp, 2 * 8    ; unshift the pushed arguments so rsp points to the saved regs 
	restore_regs
	
	add rsp, 8	  ; pop handler address off the stack before returning
	iretq

;------------------------------------------------------------------------------
; Error messages
;------------------------------------------------------------------------------
err_kmain_returned db '[rt0_64] kmain returned', 0

;------------------------------------------------------------------------------
; Write the NULL-terminated string contained in rdi to the screen using white
; text on red background.  Assumes that text-mode is enabled and that its
; physical address is 0xb8000.
;------------------------------------------------------------------------------
write_string:
	mov rbx,0xb8000
	mov ah, 0x4F
.next_char:
	mov al, byte[rdi]
	test al, al
	jz write_string.done

	mov word [rbx], ax
	add rbx, 2
	inc rdi
	jmp write_string.next_char

.done:
	ret
