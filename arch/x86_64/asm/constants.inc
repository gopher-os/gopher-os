; vim: set ft=nasm :

; The bootloader load the kernel at LOAD_ADDRESS and jumps to the rt0_32 entrypoint 
; at this address. 
LOAD_ADDRESS equ 0x100000

; Page offset is the start of the 48-bit upper half canonical memory region 
; The kernel is compiled with a VMA equal to PAGE_OFFSET + LOAD_ADDRESS but 
; loaded at physical address LOAD_ADDRESS. 
PAGE_OFFSET equ 0xffff800000000000


