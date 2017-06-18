; vim: set ft=nasm :
%include "constants.inc"

section .bss
align 8

r0_g_ptr:	  resq 1 ; fs:0x00 is a pointer to the current g struct
r0_g:
r0_g_stack_lo:    resq 1
r0_g_stack_hi:    resq 1
r0_g_stackguard0: resq 1  ; rsp compared to this value in go stack growth prologue
r0_g_stackguard1: resq 1  ; rsp compared to this value in C stack growth prologue

section .text 
bits 64

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
