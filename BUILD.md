## Building running and debugging gopher-os

The project Makefile contains targets for building either the kernel image or 
a bootable ISO while running on Linux or OSX.

## Building on Linux

To compile gopher-os wheh running on Linux you need a fairly recent version of:
- binutils (>= 2.26.0)
- xorriso
- grub
- nasm 
- gcc (for GNU ld)
- go (1.6+; recommended: 1.8)

The above dependencies can be installed using the appropriate package manager 
for each particular Linux distribution.

## Building on OSX

To properly link the kernel object files so that the bootloader can pick up the 
multi-boot signature we need to be able to control the linker configuration. For
the time being this is only possible when using GNU ld ([lld](https://lld.llvm.org/) 
is a potential alternative but doesn't yet fully support linker scripts).

You can still build the kernel using [vagrant](https://www.vagrantup.com/). For
this purpose, a Vagrantfile is provided so all you need to do is just install 
vagrant on your machine and run `vagrant up` before running any of the following 
make commands.

## Supported Makefile build targets 

The project Makefile will work on both Linux and OSX (using vagrant) targets.
When running under OSX, the Makefile will ensure that all build-related commands
actually run inside the vagrant box. The following build targets are
supported:
- `kernel`: compile the code into an elf32 binary.
- `iso`: compile the code and build a bootable ISO using grub as the
  bootloader.

## Booting the gopher-os ISO file 

Once the kernel ISO is successfully built, either [qemu](http://www.qemu-project.org/) or
[virtualbox](https://www.virtualbox.org/) can be used to boot it. The Makefile 
provides handy targets for doing this:
- `make run-qemu` 
- `make run-vbox`

## Supported kernel command line options 

To apply any of the following command line arguments there are two options:
1) patch [grub.cfg](src/arch/x86_64/script/grub.cfg) before building the kernel image and 
   append the required command line arguments at the end of the lines starting with `multiboot2`
2) alternatively, you can boot the ISO, wait for the grub menu to appear and press `e`. This 
   will bring up an editor where you can modify the command line before booting the kernel.

The following command line options are currently supported:

| Command | Description 
|-----------------------|-------------
|consoleFont=$fontName  | use a particular font name (e.g terminus10x18). This option is only used by console drivers supporting bitmap fonts. The set of built-in fonts is located [here](src/gopheros/device/video/console/font). If this option is not specified, the console driver will pick the best font size for the console resolution
|consoleLogo=off        | disable the console logo. This option is only valid for console drivers that support logos.

## Debugging the kernel 

If you wish to debug the kernel, you need to install gdb. Unfortunately the 
gdb version that ships with most Linux distributions (and also the one that 
can be installed with `brew` on OSX) has a bug which prevents gdb from properly 
handling CPU switches from 32-bit protected to 64-bit long mode. This causes 
problems when trying to debug the kernel while it is running on qemu. The 
solution to this problem is to manually compile and install a patched gdb version which is 
available [here](https://github.com/phil-opp/binutils-gdb).

The Makefile provides a `gdb` target which compiles the kernel, builds the ISO 
file, launches qemu and attaches an interactive gdb session to it.
