; vim: set ft=nasm :
%include "constants.inc"

section .data
align 4

; GDT definition
gdt0:
gdt0_nil_seg: dw 0           ; Limit (low)
	      dw 0           ; Base (low)
	      db 0           ; Base (middle)
	      db 0           ; Access (exec/read)
	      db 0           ; Granularity
	      db 0           ; Base (high)
gdt0_cs_seg:  dw 0           ; Limit (low)
	      dw 0           ; Base (low)
	      db 0           ; Base (middle)
	      db 10011010b   ; Access (exec/read)
	      db 00100000b   ; Granularity
	      db 0           ; Base (high)
gdt0_ds_seg:  dw 0           ; Limit (low)
              dw 0           ; Base (low)
              db 0           ; Base (middle)
              db 10010010b   ; Access (read/write)
              db 00000000b   ; Granularity
              db 0           ; Base (high)

gdt0_desc:
	dw $ - gdt0 - 1  ; gdt size should be 1 byte less than actual length
	dq gdt0 - PAGE_OFFSET

NULL_SEG equ gdt0_nil_seg - gdt0
CS_SEG   equ gdt0_cs_seg - gdt0
DS_SEG   equ gdt0_ds_seg - gdt0

;------------------------------------------------------------------------------
; Error messages
;------------------------------------------------------------------------------
err_unsupported_bootloader db '[rt0_32] kernel not loaded by multiboot-compliant bootloader', 0
err_multiboot_data_too_big db '[rt0_32] multiboot information data length exceeds local buffer size', 0
err_cpuid_not_supported db '[rt0_32] the processor does not support the CPUID instruction', 0
err_longmode_not_supported db '[rt0_32] the processor does not support longmode which is required by this kernel', 0
err_sse_not_supported db '[rt0_32] the processor does not support SSE instructions which are required by this kernel', 0

section .bss
align 4096

; Reserve 3 pages for the initial page tables
page_table_l4:		resb 4096
page_table_l3:		resb 4096
page_table_l2:		resb 4096

; Reserve 16K for storing multiboot data and for the kernel stack
global multiboot_data ; Make this available to the 64-bit entrypoint
global stack_bottom
global stack_top
multiboot_data: resb 16384
stack_bottom:   resb 16384
stack_top:

section .rt0
bits 32
align 4

;------------------------------------------------------------------------------
; Kernel 32-bit entry point
;
; The boot loader will jump to this symbol after setting up the CPU according
; to the multiboot standard. At this point:
; - A20 is enabled
; - The CPU is using 32-bit protected mode
; - Interrupts are disabled
; - Paging is disabled
; - EAX contains the magic value ‘0x36d76289’; the presence of this value indicates
;   to the operating system that it was loaded by a Multiboot-compliant boot loader
; - EBX contains the 32-bit physical address of the Multiboot information structure
;------------------------------------------------------------------------------
global _rt0_32_entry
_rt0_32_entry:
	; Provide a stack 
	mov esp, stack_top - PAGE_OFFSET

	; Ensure we were booted by a bootloader supporting multiboot
	cmp eax, 0x36d76289
	jne _rt0_32_entry.unsupported_bootloader

	; Copy multiboot struct to our own buffer
	call _rt0_copy_multiboot_data

	; Check processor features
	call _rt0_check_cpuid_support
	call _rt0_check_longmode_support
	call _rt0_check_sse_support

	; Setup initial page tables, enable paging and enter longmode 
	call _rt0_populate_initial_page_tables
	call _rt0_enter_long_mode

	call _rt0_64_entry_trampoline

.unsupported_bootloader:
	mov edi, err_unsupported_bootloader - PAGE_OFFSET
	call write_string
	jmp _rt0_32_entry.halt

.halt:
	cli
	hlt

;------------------------------------------------------------------------------
; Copy multiboot information blocks from the address pointed to by ebx into a
; local buffer. This enables the kernel code to access them once paging is enabled.
;------------------------------------------------------------------------------
_rt0_copy_multiboot_data:
	mov esi, ebx
	mov edi, multiboot_data - PAGE_OFFSET

	mov ecx, dword [esi]
	cmp ecx, 16384
	jle _rt0_copy_multiboot_data.copy

	mov edi, err_multiboot_data_too_big - PAGE_OFFSET
	call write_string
	jmp _rt0_32_entry.halt

.copy:
	test ecx, ecx
	jz _rt0_copy_multiboot_data.done

	mov eax, dword[esi]
	mov dword [edi], eax
	add esi, 4
	add edi, 4
	sub ecx, 4
	jmp _rt0_copy_multiboot_data.copy

.done:
	ret

;------------------------------------------------------------------------------
; Check that the processor supports the CPUID instruction.
; 
; To check if CPUID is supported, we need to attempt to flip the ID bit (bit 21)
; in the FLAGS register. If that works, CPUID is available.
;
; Code taken from: http://wiki.osdev.org/Setting_Up_Long_Mode#x86_or_x86-64
;------------------------------------------------------------------------------
_rt0_check_cpuid_support:
	; Copy FLAGS in to EAX via stack
	pushfd
	pop eax

	; Copy to ECX as well for comparing later on
	mov ecx, eax

	; Flip the ID bit
	xor eax, 1 << 21

	; Copy EAX to FLAGS via the stack
	push eax
	popfd

	; Copy FLAGS back to EAX (with the flipped bit if CPUID is supported)
	pushfd
	pop eax

	; Restore FLAGS from the old version stored in ECX (i.e. flipping the
	; ID bit back if it was ever flipped).
	push ecx
	popfd

	; Compare EAX and ECX. If they are equal then that means the bit
	; wasn't flipped, and CPUID isn't supported.
	cmp eax, ecx
	je _rt0_check_cpuid_support.no_cpuid
	ret

.no_cpuid:
	mov edi, err_cpuid_not_supported - PAGE_OFFSET
	call write_string
	jmp _rt0_32_entry.halt

;------------------------------------------------------------------------------
; Check that the processor supports long mode
; Code taken from: http://wiki.osdev.org/Setting_Up_Long_Mode#x86_or_x86-64
;------------------------------------------------------------------------------
_rt0_check_longmode_support:
	; To check for longmode support we need to ensure that the CPUID instruction
	; can report it. To do this we need to query it first.
	mov eax, 0x80000000    ; Set the A-register to 0x80000000.
	cpuid
	cmp eax, 0x80000001    ; We need at least 0x80000001 to check for long mode.
	jb _rt0_check_longmode_support.no_long_mode

	mov eax, 0x80000001    ; Set the A-register to 0x80000001.
	cpuid
	test edx, 1 << 29      ; Test if the LM-bit, which is bit 29, is set in the D-register.
	jz _rt0_check_longmode_support.no_long_mode
	ret

.no_long_mode:
	mov edi, err_longmode_not_supported - PAGE_OFFSET
	call write_string
	jmp _rt0_32_entry.halt

;------------------------------------------------------------------------------
; Check for and enabl SSE support. Code taken from:
; http://wiki.osdev.org/SSE#Checking_for_SSE
;------------------------------------------------------------------------------
_rt0_check_sse_support:
	; check for SSE
	mov eax, 0x1
	cpuid
	test edx, 1<<25
	jz _rt0_check_sse_support.no_sse

	; Enable SSE
	mov eax, cr0
	and ax, 0xfffb      ; Clear coprocessor emulation CR0.EM
	or ax, 0x2          ; Set coprocessor monitoring  CR0.MP
	mov cr0, eax
	mov eax, cr4
	or ax, 3 << 9       ; Set CR4.OSFXSR and CR4.OSXMMEXCPT at the same time
	mov cr4, eax

	ret
.no_sse:
	mov edi, err_sse_not_supported - PAGE_OFFSET
	call write_string
	jmp _rt0_32_entry.halt

;------------------------------------------------------------------------------
; Setup minimal page tables to allow access to the following regions:
; - 0 to 8M
; - PAGE_OFFSET to PAGE_OFFSET + 8M
;
; The second region mapping allows us to access the kernel at its VMA when 
; paging is enabled.
;------------------------------------------------------------------------------
PAGE_PRESENT  equ (1 << 0)
PAGE_WRITABLE equ (1 << 1)
PAGE_2MB     equ (1 << 7)

_rt0_populate_initial_page_tables:
	; The CPU uses bits 39-47 of the virtual address as an index to the P4 table.
	mov eax, page_table_l3 - PAGE_OFFSET
	or eax, PAGE_PRESENT | PAGE_WRITABLE
	mov ebx, page_table_l4 - PAGE_OFFSET
	mov [ebx], eax 
	
	; Recursively map the last P4 entry to itself. This allows us to use 
	; specially crafted memory addresses to access the page tables themselves
	mov ecx, ebx 
	or ecx, PAGE_PRESENT | PAGE_WRITABLE 
	mov [ebx + 511*8], ecx

	; Also map the addresses starting at PAGE_OFFSET to the same P3 table. 
	; To find the P4 index for PAGE_OFFSET we need to extract bits 39-47
	; of its address.
	mov ecx, (PAGE_OFFSET >> 39) & 511
	mov [ebx + ecx*8], eax 

	; The CPU uses bits 30-38 as an index to the P3 table. We just need to map 
	; entry 0 from the P3 table to point to the P2 table .
	mov eax, page_table_l2 - PAGE_OFFSET
	or eax, PAGE_PRESENT | PAGE_WRITABLE 
	mov ebx, page_table_l3 - PAGE_OFFSET
	mov [ebx], eax 

	; For the L2 table we enable the huge page bit which allows us to specify 
	; 2M pages without needing to use the L1 table. To cover the required 
	; 0-8M region we need to provide 4 2M page entries at indices 0 to 4.
	mov ecx, 0
	mov ebx, page_table_l2 - PAGE_OFFSET
.next_page:
	mov eax, 1 << 21  ; 2M
	mul ecx           ; eax *= ecx
	or eax, PAGE_PRESENT | PAGE_WRITABLE | PAGE_2MB
	mov [ebx + ecx*8], eax

	inc ecx 
	cmp ecx, 4
	jne _rt0_populate_initial_page_tables.next_page

	ret

;------------------------------------------------------------------------------
; Load P4 table, enable PAE, enter long mode and finally enable paging
;------------------------------------------------------------------------------
_rt0_enter_long_mode:
	; Load page table map pointer to cr3
	mov eax, page_table_l4 - PAGE_OFFSET
	mov cr3, eax  

	; Enable PAE support 
	mov eax, cr4 
	or eax, 1 << 5
	mov cr4, eax

	; Now enable long mode (bit 8) and the no-execute support (bit 11) by 
	; modifying the EFER MSR
	mov ecx, 0xc0000080
	rdmsr	; read msr value to eax
	or eax, (1 << 8) | (1<<11)
	wrmsr

	; Finally enable paging (bit 31) and user/kernel page write protection (bit 16)
	mov eax, cr0
	or eax, (1 << 31) | (1<<16)
	mov cr0, eax

	; We are in 32-bit compatibility submode. We need to load a 64bit GDT 
	; and perform a far jmp to switch to long mode
	mov eax, gdt0_desc - PAGE_OFFSET
	lgdt [eax]

	; set ds and es segments
	; to set the cs segment we need to perform a far jmp
	mov ax, DS_SEG
	mov ds, ax
	mov es, ax
	mov fs, ax
	mov gs, ax
	mov ss, ax

	jmp CS_SEG:.flush_gdt - PAGE_OFFSET
.flush_gdt:
	ret

;------------------------------------------------------------------------------
; Write the NULL-terminated string contained in edi to the screen using white
; text on red background.  Assumes that text-mode is enabled and that its
; physical address is 0xb8000.
;------------------------------------------------------------------------------
write_string:
	mov ebx,0xb8000
	mov ah, 0x4F
.next_char:
	mov al, byte[edi]
	test al, al
	jz write_string.done

	mov word [ebx], ax
	add ebx, 2
	inc edi
	jmp write_string.next_char

.done:
	ret


;------------------------------------------------------------------------------
; Set up the stack pointer to the virtual address of the stack and jump to the 
; 64-bit entrypoint.
;------------------------------------------------------------------------------
bits 64 
_rt0_64_entry_trampoline:
	; The currently loaded GDT points to the physical address of gdt0. This
	; works for now since we identity map the first 8M of the kernel. When
	; we set up a proper PDT for the VMA address of the kernel, the 0-8M
	; mapping will be invalid causing a page fault when the CPU tries to
	; restore the segment registers while returning from the page fault
	; handler.
	;
	; To fix this, we need to update the GDT so it uses the 48-bit virtual 
	; address of gdt0.
	mov rax, gdt0_desc
	mov rbx, gdt0
	mov qword [rax+2], rbx
	lgdt [rax]

	mov rsp, stack_top     ; now that paging is enabled we can load the stack 
			       ; with the virtual address of the allocated stack.
	
	; Jump to 64-bit entry
	extern _rt0_64_entry
	mov rax, _rt0_64_entry
	jmp rax
