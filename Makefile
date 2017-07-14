OS = $(shell uname -s)
ARCH := x86_64
BUILD_DIR := build
BUILD_ABS_DIR := $(CURDIR)/$(BUILD_DIR)

kernel_target :=$(BUILD_DIR)/kernel-$(ARCH).bin
iso_target := $(BUILD_DIR)/kernel-$(ARCH).iso

# If your go is called something else set it on the commandline, like
# this: make run GO=go1.8
GO ?= go
GOPATH := $(GOPATH):$(shell pwd)

ifeq ($(OS), Linux)
export SHELL := /bin/bash -o pipefail

LD := ld
AS := nasm

GOOS := linux
GOARCH := amd64
GOROOT := $(shell $(GO) env GOROOT)

GC_FLAGS ?=
LD_FLAGS := -n -T $(BUILD_DIR)/linker.ld -static --no-ld-generated-unwind-info
AS_FLAGS := -g -f elf64 -F dwarf -I $(BUILD_DIR)/ -I src/arch/$(ARCH)/asm/ \
	    -dNUM_REDIRECTS=$(shell GOPATH=$(GOPATH) $(GO) run tools/redirects/redirects.go count)

MIN_OBJCOPY_VERSION := 2.26.0
HAVE_VALID_OBJCOPY := $(shell objcopy -V | head -1 | awk -F ' ' '{print "$(MIN_OBJCOPY_VERSION)\n" $$NF}' | sort -ct. -k1,1n -k2,2n && echo "y")

asm_src_files := $(wildcard src/arch/$(ARCH)/asm/*.s)
asm_obj_files := $(patsubst src/arch/$(ARCH)/asm/%.s, $(BUILD_DIR)/arch/$(ARCH)/asm/%.o, $(asm_src_files))

.PHONY: kernel iso clean binutils_version_check

kernel: binutils_version_check kernel_image

kernel_image: $(kernel_target)
	@echo "[tools:redirects] populating kernel image redirect table"
	@GOPATH=$(GOPATH) $(GO) run tools/redirects/redirects.go populate-table $(kernel_target)

$(kernel_target): asm_files linker_script go.o
	@echo "[$(LD)] linking kernel-$(ARCH).bin"
	@$(LD) $(LD_FLAGS) -o $(kernel_target) $(asm_obj_files) $(BUILD_DIR)/go.o

go.o:
	@mkdir -p $(BUILD_DIR)

	@echo "[go] compiling go sources into a standalone .o file"
	@GOARCH=$(GOARCH) GOOS=$(GOOS) GOPATH=$(GOPATH) $(GO) build -gcflags '$(GC_FLAGS)' -n gopheros 2>&1 | sed \
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
	@echo "[objcopy] create kernel.Kmain alias to gopheros/kernel/kmain.Kmain"
	@echo "[objcopy] globalizing symbols {_rt0_interrupt_handlers, runtime.g0/m0/physPageSize}"
	@objcopy \
		--add-symbol kernel.Kmain=.text:0x`nm $(BUILD_DIR)/go.o | grep "kmain.Kmain$$" | cut -d' ' -f1` \
		--globalize-symbol _rt0_interrupt_handlers \
		--globalize-symbol runtime.g0 \
		--globalize-symbol runtime.m0 \
		--globalize-symbol runtime.physPageSize \
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
	@gcc `cat src/arch/$(ARCH)/asm/constants.inc | sed -e "/^$$/d; /^;/d; s/^/-D/g; s/\s*equ\s*/=/g;" | tr '\n' ' '` \
		-E -x \
		c src/arch/$(ARCH)/script/linker.ld.in | grep -v "^#" > $(BUILD_DIR)/linker.ld

$(BUILD_DIR)/go_asm_offsets.inc:
	@mkdir -p $(BUILD_DIR)

	@echo "[tools:offsets] calculating OS/arch-specific offsets for g, m and stack structs"
	@GOPATH=$(GOPATH) $(GO) run tools/offsets/offsets.go -target-os $(GOOS) -target-arch $(GOARCH) -go-binary $(GO) -out $@

$(BUILD_DIR)/arch/$(ARCH)/asm/%.o: src/arch/$(ARCH)/asm/%.s
	@mkdir -p $(shell dirname $@)
	@echo "[$(AS)] $<"
	@$(AS) $(AS_FLAGS) $< -o $@

asm_files: $(BUILD_DIR)/go_asm_offsets.inc $(asm_obj_files)

iso: $(iso_target)

$(iso_target): iso_prereq kernel_image
	@echo "[grub] building ISO kernel-$(ARCH).iso"

	@mkdir -p $(BUILD_DIR)/isofiles/boot/grub
	@cp $(kernel_target) $(BUILD_DIR)/isofiles/boot/kernel.bin
	@cp src/arch/$(ARCH)/script/grub.cfg $(BUILD_DIR)/isofiles/boot/grub
	@grub-mkrescue -o $(iso_target) $(BUILD_DIR)/isofiles 2>&1 | sed -e "s/^/  | /g"
	@rm -r $(BUILD_DIR)/isofiles

else
VAGRANT_SRC_FOLDER = /home/vagrant/workspace

.PHONY: kernel iso vagrant-up vagrant-down vagrant-ssh run gdb clean lint lint-check-deps test collect-coverage

kernel:
	vagrant ssh -c 'cd $(VAGRANT_SRC_FOLDER); make GC_FLAGS="$(GC_FLAGS)" kernel'

iso:
	vagrant ssh -c 'cd $(VAGRANT_SRC_FOLDER); make GC_FLAGS="$(GC_FLAGS)" iso'

endif

run: GC_FLAGS += -B
run: iso
	qemu-system-$(ARCH) -cdrom $(iso_target) -vga std -d int,cpu_reset -no-reboot

# When building gdb target disable optimizations (-N) and inlining (l) of Go code
gdb: GC_FLAGS += -N -l
gdb: iso
	qemu-system-$(ARCH) -M accel=tcg -vga std -s -S -cdrom $(iso_target) &
	sleep 1
	gdb \
	    -ex 'add-auto-load-safe-path $(pwd)' \
	    -ex 'set disassembly-flavor intel' \
	    -ex 'layout split' \
	    -ex 'set arch i386:intel' \
	    -ex 'file $(kernel_target)' \
	    -ex 'target remote localhost:1234' \
	    -ex 'set arch i386:x86-64:intel' \
	    -ex 'source $(GOROOT)/src/runtime/runtime-gdb.py' \
	    -ex 'set substitute-path $(VAGRANT_SRC_FOLDER) $(shell pwd)'
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
		--exclude 'x \^ 0 always equals x' \
		./...

lint-check-deps:
	@GOPATH=$(GOPATH) $(GO) get -u -t gopkg.in/alecthomas/gometalinter.v1
	@gometalinter.v1 --install >/dev/null

test:
	GOPATH=$(GOPATH) $(GO) test -cover gopheros/...

collect-coverage:
	GOPATH=$(GOPATH) sh coverage.sh

