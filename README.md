# gopheros
[![Build Status](https://travis-ci.org/achilleasa/gopher-os.svg?branch=master)](https://travis-ci.org/achilleasa/gopher-os)
[![codecov](https://codecov.io/gh/achilleasa/gopher-os/branch/master/graph/badge.svg)](https://codecov.io/gh/achilleasa/gopher-os)
[![Go Report Card](https://goreportcard.com/badge/github.com/achilleasa/gopher-os)](https://goreportcard.com/report/github.com/achilleasa/gopher-os)

Let's write an experimental OS in Go!

## Building

There are several dependancies you need to install for this to work.

You need to install qemu-system-x86_64, vagrant and of course golang.

On MacOS with homebrew run these commands to install the dependancies.

        $ brew install qemu
        $ brew cask install virtualbox
        $ brew cask install vagrant
        $ brew cask install vagrant-manager

To build the OS, clone the repo, cd to it, install the dependancies and enter these two commands.

        $ vagrant up
        $ make run
