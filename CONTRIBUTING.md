# Contributing Guide

## Getting Started

- Make sure you have a [GitHub Account](https://github.com/signup/free).
- Make sure you have [Git](http://git-scm.com/) installed on your system.
- [Fork](https://help.github.com/articles/fork-a-repo) the [repository](https://github.com/achilleasa/gopher-os) on GitHub.

## Making Changes

 - [Create a branch](https://help.github.com/articles/creating-and-deleting-branches-within-your-repository) for your changes.
 - [Commit your code](http://git-scm.com/book/en/Git-Basics-Recording-Changes-to-the-Repository) for each logical change (see [tips for creating better commit messages](http://robots.thoughtbot.com/5-useful-tips-for-a-better-commit-message)).
 - [Push your change](https://help.github.com/articles/pushing-to-a-remote) to your fork.
 - [Create a Pull Request](https://help.github.com/articles/creating-a-pull-request) on GitHub for your change.

The PR description should be as detailed as possible. This makes reviewing
much easier while at the same time serves as additional documentation that can 
be referenced by future commits/PRs.

This project treats the root folder of the repository as a Go [workspace](https://golang.org/doc/code.html#Workspaces). This
approach has several benefits:
- it keeps import paths short (no github.com/... prefix)
- it makes forking and merging easier
- it simplifies debugging (more compact symbol names)

To develop for gopher-os you need to tweak your GOPATH so that the repository
folder is listed before any other GOPATH entry. This allows tools like
`goimports` to figure out the correct (short) import path for any gopher-os
package that your code imports. A simple way to do this would be by running the 
following command: ```export GOPATH=`pwd`:$GOPATH```.

## Unit tests and code linting

Before submitting a PR make sure:
- that your code passes all lint checks: `make lint`
- you provide the appropriate unit-tests to ensure that the coverage does not 
  drop below the existing value (currently 100%). Otherwise, when you submit the 
  PR, the CI builder ([travis-ci](https://travis-ci.org)) will flag the build as 
  broken.

Reaching 100% coverage is quite hard and requires the code to be designed with
testability in mind. This can get quite tricky if the code you are testing
relies on code that cannot be executed while running the tests. For example, if
the code you are currently working on needs to map some pages to virtual memory
then any call to the vmm package from your test code will cause the `go test`
to segfault.

In cases like this, you need to design the code so calls to such packages can
be easily mocked while testing. If you are looking for inspiration here are
some examples that follow this approach: 
- [bitmap allocator tests](https://github.com/achilleasa/gopher-os/blob/d804b17ed8651705f098d01bda65d8f0ded2c88e/src/gopheros/kernel/mem/pmm/allocator/bitmap_allocator_test.go#L15)
- [text console driver tests](https://github.com/achilleasa/gopher-os/blob/4b25971cef4bfd01877e3b5e948ee07a8f219608/src/gopheros/device/video/console/vga_text_test.go#L276)
