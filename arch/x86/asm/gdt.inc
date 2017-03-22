; vim: set ft=nasm :

%define SEG_NOEXEC        (0 << 3)
%define SEG_EXEC          (1 << 3)

%define SEG_NORW          (0 << 1)
%define SEG_R             (1 << 1)
%define SEG_W             (1 << 1)

%define SEG_GRAN_BYTE     (0 << 7)
%define SEG_GRAN_4K_PAGE  (1 << 7)

;------------------------------------------------------------------------------
; GDT_ENTRY_32 creates a GDT entry for a 32-bit descriptor. It automatically sets 
; the following bits:
; - Privl (ring) bits to 00 (ring 0)
; - Pr (present) bit to 1
; - Sz (size) bit to 1 (32-bit selector)
; - L (long-mode) bit to 0
;
; Args: base, limit, access, flags
;------------------------------------------------------------------------------
%macro GDT_ENTRY_32 4
	dw (%2 & 0xFFFF)                              ; limit 0:15
	dw (%1 & 0xFFFF)                              ; base 0:15
	db ((%1 >> 16) & 0xFF)                        ; base 16:23
	db (0x90 | %3)                                ; set Pr = 1, bit 5 = 1 (required)
	                                              ; and apply access byte flags
	db 0x40 | (%4 & 0xC0) | ((%2 >> 16) & 0xF)    ; set Sz and flags and limit bits 16:19
	db ((%1 >> 24) & 0xFF)                        ; base 24:31
%endmacro
