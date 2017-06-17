; vim: set ft=nasm :

section .text
bits 64

global x_cgo_callers
global x_cgo_init
global x_cgo_mmap
global x_cgo_notify_runtime_init_done
global x_cgo_sigaction
global x_cgo_thread_start
global x_cgo_setenv
global x_cgo_unsetenv
global _cgo_yield

; Stubs for missing cgo functions to keep the linker happy
x_cgo_callers:
x_cgo_init:
x_cgo_mmap:
x_cgo_notify_runtime_init_done:
x_cgo_sigaction:
x_cgo_thread_start:
x_cgo_setenv:
x_cgo_unsetenv:
_cgo_yield:
	ret
