; vim: set ft=nasm :

section .bss
align 4

; Reserve 16K for our stack. Stacks should be aligned to 16 byte boundaries.
stack_bottom:
	resb 16384 	; 16 KiB
stack_top:

; Reserve some extra space for our tls_0 block; GO functions expect the
; GS segment register to point to the current TLS so we need to initialize this
; first before invoking any go functions
tls0:
g0_ptr:	        resd 1 ; gs:0x00 is a pointer to the current g struct
                       ; in our case it should point to g0
g0:
g0_stack_lo:    resd 1
g0_stack_hi:    resd 1
g0_stackguard0: resd 1  ; sp compared to this value in go stack growth prologue
g0_stackguard1: resd 1  ; sp compared to this value in C stack growth prologue

section .text
bits 32
align 4

;------------------------------------------------------------------------------
; Kernel arch-specific entry point
;
; The boot loader will jump to this symbol after setting up the CPU according
; to the multiboot standard. At this point:
; - The CPU is using 32-bit protected mode
; - Interrupts are disabled
; - Paging is disabled
;------------------------------------------------------------------------------
global _rt0_entry
_rt0_entry:
	; Initalize our stack by pointing ESP to the BSS-allocated stack. In x86,
	; stack grows downwards so we need to point ESP to stack_top
	mov esp, stack_top

 	; Load initial GDT
 	call _rt0_load_gdt

	; init g0 so we can invoke Go functions
	mov dword [gs:0x00], g0
	mov dword [g0_stack_hi], stack_top
	mov dword [g0_stack_lo], stack_bottom
	mov dword [g0_stackguard0], stack_bottom

	extern main.main
	call main.main

	; Main should never return; halt the CPU
	cli
	hlt
.end:

;------------------------------------------------------------------------------
; Load GDT and flush CPU caches
;------------------------------------------------------------------------------

_rt0_load_gdt:
	; Go code uses the GS register to access the TLS. Set the base address
	; for the GS descriptor to point to our tls0 table
	mov eax, tls0
	mov ebx, gdt0_gs_seg
	mov [ebx+2], al
	mov [ebx+3], ah
	shr eax, 16
	mov [ebx+4], al

	lgdt [gdt0_desc]

	; GDT has been loaded but the CPU still has the previous GDT data in cache.
	; We need to manually update the descriptors and use a JMP command to set
	; the CS segment descriptor
	jmp CS_SEG:update_descriptors
update_descriptors:
	mov ax, DS_SEG
	mov ds, ax
	mov es, ax
	mov fs, ax
	mov ax, GS_SEG
	mov gs, ax

	ret

;------------------------------------------------------------------------------
; GDT definition
;------------------------------------------------------------------------------
%include "gdt.inc"

align 2
gdt0:

gdt0_nil_seg: GDT_ENTRY_32 0x00, 0x0, 0x0, 0x0				        ; nil descriptor (not used by CPU but required by some emulators)
gdt0_cs_seg:  GDT_ENTRY_32 0x00, 0xFFFFF, SEG_EXEC | SEG_R, SEG_GRAN_4K_PAGE    ; code descriptor
gdt0_ds_seg:  GDT_ENTRY_32 0x00, 0xFFFFF, SEG_NOEXEC | SEG_W, SEG_GRAN_4K_PAGE  ; data descriptor
gdt0_gs_seg:  GDT_ENTRY_32 0x00, 0x40, SEG_NOEXEC | SEG_W, SEG_GRAN_BYTE        ; TLS descriptor (required in order to use go segmented stacks)

gdt0_desc:
	dw gdt0_desc - gdt0 - 1  ; gdt size should be 1 byte less than actual length
	dd gdt0

NULL_SEG equ gdt0_nil_seg - gdt0
CS_SEG   equ gdt0_cs_seg - gdt0
DS_SEG   equ gdt0_ds_seg - gdt0
GS_SEG   equ gdt0_gs_seg - gdt0
