export SHELL := /bin/bash -o pipefail

OS = $(shell uname -s)
BUILD_DIR := build
BUILD_ABS_DIR := $(CURDIR)/$(BUILD_DIR)

VBOX_VM_NAME := gopher-os
QEMU ?= qemu-system-x86_64

# If your go is called something else set it on the commandline, like this: make run GO=go1.8
GO ?= go
GOARCH := amd64
GOROOT := $(shell $(GO) env GOROOT)

# Prepend build path to GOPATH so the compiled packages and linter dependencies
# end up inside the build folder
GOPATH := $(BUILD_ABS_DIR):$(shell pwd):$(GOPATH)

LD := ld
LD_FLAGS := -n -T $(BUILD_DIR)/linker.ld -static --no-ld-generated-unwind-info

AS := nasm
AS_FLAGS := -g -f elf64 -F dwarf -I $(BUILD_DIR)/ -I src/arch/$(GOARCH)/rt0/ \
	    -dNUM_REDIRECTS=$(shell GOPATH=$(GOPATH) $(GO) run tools/redirects/redirects.go count)

GC_FLAGS ?=

kernel_target :=$(BUILD_DIR)/kernel-$(GOARCH).bin
iso_target := $(BUILD_DIR)/kernel-$(ARCH).iso

FUZZ_PKG_LIST := src/gopheros/device/acpi/aml
# To append more entries to the above list use the following syntax
# FUZZ_PKG_LIST += path-to-pkg

ifeq ($(OS), Linux)
GOOS := linux

MIN_OBJCOPY_VERSION := 2.26.0
HAVE_VALID_OBJCOPY := $(shell objcopy -V | head -1 | awk -F ' ' '{print "$(MIN_OBJCOPY_VERSION)\n" $$NF}' | sort -ct. -k1,1n -k2,2n && echo "y")

asm_src_files := $(wildcard src/arch/$(GOARCH)/rt0/*.s)
asm_obj_files := $(patsubst src/arch/$(GOARCH)/rt0/%.s, $(BUILD_DIR)/arch/$(GOARCH)/rt0/%.o, $(asm_src_files))

.PHONY: kernel iso clean binutils_version_check

kernel: binutils_version_check kernel_image

kernel_image: $(kernel_target)
	@echo "[tools:redirects] populating kernel image redirect table"
	@GOPATH=$(GOPATH) $(GO) run tools/redirects/redirects.go populate-table $(kernel_target)

$(kernel_target): asm_files linker_script go.o
	@echo "[$(LD)] linking kernel-$(GOARCH).bin"
	@$(LD) $(LD_FLAGS) -o $(kernel_target) $(asm_obj_files) $(BUILD_DIR)/go.o

go.o:
	@mkdir -p $(BUILD_DIR)

	@echo "[go] compiling go sources into a standalone .o file"
	@GOARCH=$(GOARCH) GOOS=$(GOOS) GOPATH=$(GOPATH) $(GO) build -gcflags '$(GC_FLAGS)' -n gopheros 2>&1 | sed \
	    -e "1s|^|set -e\n|" \
	    -e "1s|^|export GOOS=$(GOOS)\n|" \
	    -e "1s|^|export GOARCH=$(GOARCH)\n|" \
	    -e "1s|^|export GOROOT=$(GOROOT)\n|" \
	    -e "1s|^|export CGO_ENABLED=0\n|" \
	    -e "1s|^|alias pack='$(GO) tool pack'\n|" \
	    -e "/^mv/d" \
	    -e "/\/buildid/d" \
	    -e "s|-extld|-tmpdir='$(BUILD_ABS_DIR)' -linkmode=external -extldflags='-nostartfiles -nodefaultlibs -nostdlib -r' -extld|g" \
	    -e 's|$$WORK|$(BUILD_ABS_DIR)|g' \
            | sh 2>&1 |  sed -e "s/^/  | /g"

	@# build/go.o is a elf32 object file but all go symbols are unexported. Our
	@# asm entrypoint code needs to know the address to 'main.main' so we use
	@# objcopy to make that symbol exportable. Since nasm does not support externs
	@# with slashes we create a global symbol alias for kernel.Kmain
	@echo "[objcopy] create kernel.Kmain alias to gopheros/kernel/kmain.Kmain"
	@echo "[objcopy] globalizing symbols {runtime.g0/m0/physPageSize}"
	@objcopy \
		--add-symbol kernel.Kmain=.text:0x`nm $(BUILD_DIR)/go.o | grep "kmain.Kmain$$" | cut -d' ' -f1` \
		--globalize-symbol runtime.g0 \
		--globalize-symbol runtime.m0 \
		--globalize-symbol runtime.physPageSize \
		 $(BUILD_DIR)/go.o $(BUILD_DIR)/go.o

binutils_version_check:
	@echo "[binutils] checking that installed objcopy version is >= $(MIN_OBJCOPY_VERSION)"
	@if [ "$(HAVE_VALID_OBJCOPY)" != "y" ]; then echo "[binutils] error: a more up to date binutils installation is required" ; exit 1 ; fi

iso_prereq: xorriso_check grub-mkrescue_check

xorriso_check:
	@if xorriso --version >/dev/null 2>&1; then exit 0; else echo "Install xorriso via 'sudo apt install xorriso'." ; exit 1 ; fi

grub-mkrescue_check:
	@if grub-mkrescue --version >/dev/null 2>&1; then exit 0; else echo "Install package grub-pc-bin via 'sudo apt install grub-pc-bin'."; exit 1; fi

linker_script:
	@echo "[sed] extracting LMA and VMA from constants.inc"
	@echo "[gcc] pre-processing arch/$(GOARCH)/script/linker.ld.in"
	@gcc `cat src/arch/$(GOARCH)/rt0/constants.inc | sed -e "/^$$/d; /^;/d; s/^/-D/g; s/\s*equ\s*/=/g;" | tr '\n' ' '` \
		-E -x \
		c src/arch/$(GOARCH)/script/linker.ld.in | grep -v "^#" > $(BUILD_DIR)/linker.ld

$(BUILD_DIR)/go_asm_offsets.inc:
	@mkdir -p $(BUILD_DIR)

	@echo "[tools:offsets] calculating OS/arch-specific offsets for g, m and stack structs"
	@GOROOT=$(GOROOT) GOPATH=$(GOPATH) $(GO) run tools/offsets/offsets.go -target-os $(GOOS) -target-arch $(GOARCH) -go-binary $(GO) -out $@

$(BUILD_DIR)/arch/$(GOARCH)/rt0/%.o: src/arch/$(GOARCH)/rt0/%.s
	@mkdir -p $(shell dirname $@)
	@echo "[$(AS)] $<"
	@$(AS) $(AS_FLAGS) $< -o $@

asm_files: $(BUILD_DIR)/go_asm_offsets.inc $(asm_obj_files)

iso: $(iso_target)

$(iso_target): iso_prereq kernel_image
	@echo "[grub] building ISO kernel-$(GOARCH).iso"

	@mkdir -p $(BUILD_DIR)/isofiles/boot/grub
	@cp $(kernel_target) $(BUILD_DIR)/isofiles/boot/kernel.bin
	@cp src/arch/$(GOARCH)/script/grub.cfg $(BUILD_DIR)/isofiles/boot/grub
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

run-qemu: GC_FLAGS += -B
run-qemu: iso
	$(QEMU) -cdrom $(iso_target) -vga std -d int,cpu_reset -no-reboot

run-vbox: iso
	VBoxManage createvm --name $(VBOX_VM_NAME) --ostype "Linux_64" --register || true
	VBoxManage storagectl $(VBOX_VM_NAME) --name "IDE Controller" --add ide || true
	VBoxManage storageattach $(VBOX_VM_NAME) --storagectl "IDE Controller" --port 0 --device 0 --type dvddrive \
		--medium $(iso_target) || true
	VBoxManage startvm $(VBOX_VM_NAME)

# When building gdb target disable optimizations (-N) and inlining (l) of Go code
gdb: GC_FLAGS += -N -l
gdb: iso
	$(QEMU) -M accel=tcg -vga std -s -S -cdrom $(iso_target) &
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
	@killall $(QEMU) || true

clean:
	@test -d $(BUILD_DIR) && rm -rf $(BUILD_DIR) || true

lint: lint-check-deps
	@echo "[gometalinter] linting sources"
	@GOCACHE=off GOPATH=$(GOPATH) PATH=$(BUILD_ABS_DIR)/bin:$(PATH) gometalinter.v1 \
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
		--enable=gofmt \
		--deadline 300s \
		--exclude 'possible misuse of unsafe.Pointer' \
		--exclude 'x \^ 0 always equals x' \
		--exclude 'dispatchInterrupt is unused' \
		--exclude 'interruptGateEntries is unused' \
		--exclude 'yieldFn is unused' \
		src/...

lint-check-deps:
	@echo [go get] installing linter dependencies
	@GOPATH=$(GOPATH) $(GO) get -u -t gopkg.in/alecthomas/gometalinter.v1
	@GOPATH=$(GOPATH) PATH=$(BUILD_ABS_DIR)/bin:$(PATH) gometalinter.v1 --install >/dev/null

test:
	GOCACHE=off GOPATH=$(GOPATH) $(GO) test -cover gopheros/...

fuzz-deps:
	@mkdir -p $(BUILD_DIR)/fuzz
	@echo [go get] installing go-fuzz dependencies
	@GOPATH=$(GOPATH) $(GO) get -u github.com/dvyukov/go-fuzz/...

%.fuzzpkg: %
	@echo [go-fuzz] fuzzing: $<
	@GOPATH=$(GOPATH) PATH=$(BUILD_ABS_DIR)/bin:$(PATH) go-fuzz-build -o $(BUILD_ABS_DIR)/fuzz/$(subst /,_,$<).zip $(subst src/,,$<)
	@mkdir -p $(BUILD_ABS_DIR)/fuzz/corpus/$(subst /,_,$<)/corpus
	@echo [go-fuzz] + grepping for corpus file hints in $<
	@grep "go-fuzz-corpus+=" $</*fuzz.go | cut -d'=' -f2 | tr '\n' '\0' | xargs -0 -I@ sh -c 'export F="@"; cp $$F $(BUILD_ABS_DIR)/fuzz/corpus/$(subst /,_,$<)/corpus/ && echo "[go fuzz]   + copy extra corpus file: $$F"'
	@go-fuzz -bin=$(BUILD_ABS_DIR)/fuzz/$(subst /,_,$<).zip -workdir=$(BUILD_ABS_DIR)/fuzz/corpus/$(subst /,_,$<) 2>&1 | sed -e "s/^/  | /g"

test-fuzz: fuzz-deps $(addsuffix .fuzzpkg,$(FUZZ_PKG_LIST))

collect-coverage:
	GOPATH=$(GOPATH) sh coverage.sh
