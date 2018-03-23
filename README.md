# gopher-os [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

| Tests / Coverage                                                       | Go 1.7.x            | Go 1.8.x            | Go 1.9.x            | Go 1.10.x            | Go 1.x            |
|------------------------------------------------------------------------|---------------------|---------------------|---------------------|----------------------|-------------------|
| [![Build Status][0]][6] [![Coverage][7]][8] [![Go Report Card][9]][10] | [![go 1.7.x][1]][6] | [![go 1.8.x][2]][6] | [![Go 1.9.x][3]][6] | [![go 1.10.x][4]][6] | [![go 1.x][5]][6] |

[0]: https://travis-ci.org/achilleasa/gopher-os.svg?branch=master
[1]: https://travis-matrix-badges.herokuapp.com/repos/achilleasa/gopher-os/branches/master/1
[2]: https://travis-matrix-badges.herokuapp.com/repos/achilleasa/gopher-os/branches/master/2
[3]: https://travis-matrix-badges.herokuapp.com/repos/achilleasa/gopher-os/branches/master/3
[4]: https://travis-matrix-badges.herokuapp.com/repos/achilleasa/gopher-os/branches/master/4
[5]: https://travis-matrix-badges.herokuapp.com/repos/achilleasa/gopher-os/branches/master/5
[6]: https://travis-ci.org/achilleasa/gopher-os
[7]: https://codecov.io/gh/achilleasa/gopher-os/branch/master/graph/badge.svg
[8]: https://codecov.io/gh/achilleasa/gopher-os
[9]: https://goreportcard.com/badge/github.com/achilleasa/gopher-os
[10]: https://goreportcard.com/report/github.com/achilleasa/gopher-os

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
