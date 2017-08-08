# gopher-os
[![Build Status](https://travis-ci.org/achilleasa/gopher-os.svg?branch=master)](https://travis-ci.org/achilleasa/gopher-os)
[![codecov](https://codecov.io/gh/achilleasa/gopher-os/branch/master/graph/badge.svg)](https://codecov.io/gh/achilleasa/gopher-os)
[![Go Report Card](https://goreportcard.com/badge/github.com/achilleasa/gopher-os)](https://goreportcard.com/report/github.com/achilleasa/gopher-os)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

The goal of this project is to build a 64-bit POSIX-compliant tick-less kernel
with a Linux-compatible syscall implementation using [Go](https://golang.org). 

This project is not about building yet another OS but rather exists to serve as
proof that Go is indeed a suitable tool for writing low level code that runs
at ring-0.

**Note**: This project is still in the early stages of development and is not yet
in a usable state. In fact, if you build the ISO and boot it, the kernel will 
eventually panic with a `Kmain returned` error.

To find out more about the current project status and feature roadmap take a
look at the [status](STATUS.md) page.

## Building and running gopher-os 

TLDR version: `make run-qemu` or `make run-vbox`. 

A detailed guide about building, running and debugging gopher-os on
Linux/OSX as well as the list of supported boot command line options are
available [here](BUILD.md).

## How does it look?

80x25 (stadard 8x16 font): ![80x25 with standard 8x16 font][cons-80x25]

1024x768 (10x18 font): ![1024x768x32 with 10x18 font][cons-1024x768]

2560x1600 (14x28 font): ![retina mode (2560x1600) with 14x28 font][cons-2560x1600]

[cons-80x25]: https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBb3NHY1JtTFFZckU
[cons-1024x768]: https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBZ1M3MTNjc3NaOXM
[cons-2560x1600]: https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBbjBNSEJlTmJTelE

## Contributing

gopher-os is Open Source. Feel free to contribute! To get started take a look 
at the contributing [guide](CONTRIBUTING.md).

## Licence

gopher-os is distributed under the [MIT](LICENSE) license.
