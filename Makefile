OS = $(shell uname -s)
ARCH := x86_64
BUILD_DIR := build
BUILD_ABS_DIR := $(CURDIR)/$(BUILD_DIR)

kernel_target :=$(BUILD_DIR)/kernel-$(ARCH).bin
iso_target := $(BUILD_DIR)/kernel-$(ARCH).iso

# If your go is called something else set it on the commandline, like
# this: make run GO=go1.8
GO ?= go

ifeq ($(OS), Linux)
export SHELL := /bin/bash -o pipefail

LD := ld
AS := nasm

GOOS := linux
GOARCH := amd64
GOROOT := $(shell $(GO) env GOROOT)

LD_FLAGS := -n -T $(BUILD_DIR)/linker.ld -static --no-ld-generated-unwind-info
AS_FLAGS := -g -f elf64 -F dwarf -I arch/$(ARCH)/asm/ -dNUM_REDIRECTS=$(shell $(GO) run tools/redirects/redirects.go count)

MIN_OBJCOPY_VERSION := 2.26.0
HAVE_VALID_OBJCOPY := $(shell objcopy -V | head -1 | awk -F ' ' '{print "$(MIN_OBJCOPY_VERSION)\n" $$NF}' | sort -ct. -k1,1n -k2,2n && echo "y")

asm_src_files := $(wildcard arch/$(ARCH)/asm/*.s)
asm_obj_files := $(patsubst arch/$(ARCH)/asm/%.s, $(BUILD_DIR)/arch/$(ARCH)/asm/%.o, $(asm_src_files))

.PHONY: kernel iso clean binutils_version_check

kernel: binutils_version_check kernel_image

kernel_image: $(kernel_target)
	@echo "[tools:redirects] populating kernel image redirect table"
	@$(GO) run tools/redirects/redirects.go populate-table $(kernel_target)
	
$(kernel_target): $(asm_obj_files) linker_script go.o
	@echo "[$(LD)] linking kernel-$(ARCH).bin"
	@$(LD) $(LD_FLAGS) -o $(kernel_target) $(asm_obj_files) $(BUILD_DIR)/go.o

go.o:
	@mkdir -p $(BUILD_DIR)

	@echo "[go] compiling go sources into a standalone .o file"
	@GOARCH=$(GOARCH) GOOS=$(GOOS) $(GO) build -n 2>&1 | sed \
	    -e "1s|^|set -e\n|" \
	    -e "1s|^|export GOOS=$(GOOS)\n|" \
	    -e "1s|^|export GOARCH=$(GOARCH)\n|" \
	    -e "1s|^|export GOROOT=$(GOROOT)\n|" \
	    -e "1s|^|WORK='$(BUILD_ABS_DIR)'\n|" \
	    -e "1s|^|alias pack='$(GO) tool pack'\n|" \
	    -e "/^mv/d" \
	    -e "s|-extld|-tmpdir='$(BUILD_ABS_DIR)' -linkmode=external -extldflags='-nostartfiles -nodefaultlibs -nostdlib -r' -extld|g" \
	    | sh 2>&1 | sed -e "s/^/  | /g"

	@# build/go.o is a elf32 object file but all go symbols are unexported. Our
	@# asm entrypoint code needs to know the address to 'main.main' so we use
	@# objcopy to make that symbol exportable. Since nasm does not support externs
	@# with slashes we create a global symbol alias for kernel.Kmain
	@echo "[objcopy] creating global symbol alias 'kernel.Kmain' for 'github.com/achilleasa/gopher-os/kernel.Kmain' in go.o"
	@objcopy \
		--add-symbol kernel.Kmain=.text:0x`nm $(BUILD_DIR)/go.o | grep "kmain.Kmain$$" | cut -d' ' -f1` \
		--globalize-symbol _rt0_interrupt_handlers \
		 $(BUILD_DIR)/go.o $(BUILD_DIR)/go.o

binutils_version_check:
	@echo "[binutils] checking that installed objcopy version is >= $(MIN_OBJCOPY_VERSION)"
	@if [ "$(HAVE_VALID_OBJCOPY)" != "y" ]; then echo "[binutils] error: a more up to date binutils installation is required" ; exit 1 ; fi

iso_prereq: xorriso_check grub-pc-bin_check

xorriso_check:
	@if xorriso --version >/dev/null 2>&1; then exit 0; else echo "Install xorriso via 'sudo apt install xorriso'." ; exit 1 ; fi

grub-pc-bin_check:
	@ if dpkg -l grub-pc-bin > /dev/null; then exit 0; else echo "Install package grub-pc-bin via 'sudo apt install grub-pc-bin'."; exit 1; fi

linker_script:
	@echo "[sed] extracting LMA and VMA from constants.inc"
	@echo "[gcc] pre-processing arch/$(ARCH)/script/linker.ld.in"
	@gcc `cat arch/$(ARCH)/asm/constants.inc | sed -e "/^$$/d; /^;/d; s/^/-D/g; s/\s*equ\s*/=/g;" | tr '\n' ' '` \
		-E -x \
		c arch/$(ARCH)/script/linker.ld.in | grep -v "^#" > $(BUILD_DIR)/linker.ld

$(BUILD_DIR)/arch/$(ARCH)/asm/%.o: arch/$(ARCH)/asm/%.s
	@mkdir -p $(shell dirname $@)
	@echo "[$(AS)] $<"
	@$(AS) $(AS_FLAGS) $< -o $@

iso: $(iso_target)

$(iso_target): iso_prereq kernel_image
	@echo "[grub] building ISO kernel-$(ARCH).iso"

	@mkdir -p $(BUILD_DIR)/isofiles/boot/grub
	@cp $(kernel_target) $(BUILD_DIR)/isofiles/boot/kernel.bin
	@cp arch/$(ARCH)/script/grub.cfg $(BUILD_DIR)/isofiles/boot/grub
	@grub-mkrescue -o $(iso_target) $(BUILD_DIR)/isofiles 2>&1 | sed -e "s/^/  | /g"
	@rm -r $(BUILD_DIR)/isofiles

else
VAGRANT_SRC_FOLDER = /home/vagrant/workspace/src/github.com/achilleasa/gopher-os

.PHONY: kernel iso vagrant-up vagrant-down vagrant-ssh run gdb clean

kernel:
	vagrant ssh -c 'cd $(VAGRANT_SRC_FOLDER); make kernel'

iso:
	vagrant ssh -c 'cd $(VAGRANT_SRC_FOLDER); make iso'

endif

run: iso
	qemu-system-$(ARCH) -cdrom $(iso_target) -d int,cpu_reset -no-reboot

gdb: iso
	qemu-system-$(ARCH) -M accel=tcg -s -S -cdrom $(iso_target) &
	sleep 1
	gdb \
	    -ex 'add-auto-load-safe-path $(pwd)' \
	    -ex 'set disassembly-flavor intel' \
	    -ex 'layout asm' \
	    -ex 'set arch i386:intel' \
	    -ex 'file $(kernel_target)' \
	    -ex 'target remote localhost:1234' \
	    -ex 'set arch i386:x86-64:intel'
	@killall qemu-system-$(ARCH) || true

clean:
	@test -d $(BUILD_DIR) && rm -rf $(BUILD_DIR) || true

lint: lint-check-deps
	@echo "[gometalinter] linting sources"
	@gometalinter.v1 \
		--disable-all \
		--enable=deadcode \
		--enable=errcheck \
		--enable=gosimple \
		--enable=ineffassign \
		--enable=misspell \
		--enable=staticcheck \
		--enable=vet \
		--enable=vetshadow \
		--enable=unconvert \
		--enable=varcheck \
		--enable=golint \
		--deadline 300s \
		--exclude 'return value not checked' \
		--exclude 'possible misuse of unsafe.Pointer' \
		./...

lint-check-deps:
	@$(GO) get -u gopkg.in/alecthomas/gometalinter.v1
	@gometalinter.v1 --install >/dev/null

test:
	$(GO) test -cover ./...
