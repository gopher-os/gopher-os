## Current project status 

Here is the list of features currently working as well as some of the next 
steps in the project roadmap.

#### Core kernel features 
- Bootloader-related
	- [x] Multboot structure parsing (boot cmdline, memory maps, framebuffer and kernel image details)
- CPU 
	- [x] CPUID wrapper
	- [x] Port R/W abstraction
- Memory management
	- [x] Physical frame allocators (bootmem-based, bitmap allocator)
	- [x] VMM system (page table management, virtual address space reservations, page RW/NX bits, page walk/translation helpers and copy-on-write pages)
- Exception handling
	- [x] Page fault handling (also used to implement CoW)
	- [x] GPF handling 
- Hardware detection/abstraction layer
	- [x] Multiboot-based HW detection 
	- [ ] ACPI-based HW detection

#### Supported Go language features:
- [x] Go allocator 
- [x] Maps 
- [x] Interfaces 
- [x] Package init() functions
- [x] Defer
- [x] Panic
- [ ] GC
- [ ] Go-routines

#### Device drivers
- Console
	- [x] Text-mode console 
	- [x] Vesa-fb (15, 16, 24 and 32 bpp) console with support for bitmap fonts and (optional) logo
- TTY
	- [x] Simple VT
- ACPI 6.2 support (**in progress**)
	- [ ] ACPI table detection and parsing 
	- [ ] AML parser/interpreter
- Interrupt handling chip drivers
	- [ ] APIC
- Timer and time-keeping drivers
	- [ ] APM timer 
	- [ ] APIC timer 
	- [ ] HPET
	- [ ] RTC
- Timekeeping system 
	- [ ] Monotonic clock (configurable timer implementation)
### Feature roadmap 

Here is a list of features planned for the future:
- RAMDISK support (tar/bz2)
- Loadable modules (using a mechanism analogous to Go plugins)
- Tasks and scheduling 
- Network device drivers
- Hypervisor support
- POSIX-compliant VFS
