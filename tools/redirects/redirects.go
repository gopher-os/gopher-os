package main

import (
	"debug/elf"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type redirect struct {
	src string
	dst string

	srcVMA uint64
	dstVMA uint64
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "[redirects] error: %s\n", err.Error())
	os.Exit(1)
}

func pkgPrefix() (string, error) {
	goPath := os.Getenv("GOPATH") + "/src/"
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(cwd, goPath), nil
}

func collectGoFiles(root string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}

		if filepath.Ext(path) == ".go" && !strings.Contains(path, "_test") {
			goFiles = append(goFiles, path)
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return goFiles, nil
}

func findRedirects(goFiles []string) ([]*redirect, error) {
	var redirects []*redirect

	prefix, err := pkgPrefix()
	if err != nil {
		return nil, err
	}

	for _, goFile := range goFiles {
		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", goFile, err)
		}

		cmap := ast.NewCommentMap(fset, f, f.Comments)
		cmap.Filter(f)
		for astNode, commentGroups := range cmap {
			fnDecl, ok := astNode.(*ast.FuncDecl)
			if !ok {
				continue
			}

			for _, commentGroup := range commentGroups {
				for _, comment := range commentGroup.List {
					if !strings.Contains(comment.Text, "go:redirect-from") {
						continue
					}

					// build qualified name to fn
					fqName := fmt.Sprintf("%s/%s.%s",
						prefix,
						goFile[:strings.LastIndexByte(goFile, '/')],
						fnDecl.Name,
					)

					fields := strings.Fields(comment.Text)
					if len(fields) != 2 || fields[0] != "//go:redirect-from" {
						return nil, fmt.Errorf("malformed go:redirect-from syntax for %q", fqName)
					}

					redirects = append(redirects, &redirect{
						src: fields[1],
						dst: fqName,
					})
				}
			}
		}
	}

	return redirects, nil
}

func elfRedirectTableOffset(imgFile string) (uint64, error) {
	f, err := elf.Open(imgFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	redirectsSection := f.Section(".goredirectstbl")
	if redirectsSection == nil {
		return 0, fmt.Errorf("%s: missing .goredirectstbl section", imgFile)
	}

	return redirectsSection.Offset, nil
}

func elfWriteRedirectTable(redirects []*redirect, imgFile string) error {
	redirectTableOffset, err := elfRedirectTableOffset(imgFile)
	if err != nil {
		return err
	}

	// Open kernel image file and seek to table offset
	f, err := os.OpenFile(imgFile, os.O_WRONLY, os.ModeType)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Seek(int64(redirectTableOffset), io.SeekStart); err != nil {
		return err
	}

	for _, redirect := range redirects {
		binary.Write(f, binary.LittleEndian, redirect.srcVMA)
		binary.Write(f, binary.LittleEndian, redirect.dstVMA)
	}

	return nil
}

func elfResolveRedirectSymbols(redirects []*redirect, imgFile string) error {
	f, err := elf.Open(imgFile)
	if err != nil {
		return err
	}
	defer f.Close()

	symbols, err := f.Symbols()
	if err != nil {
		return err
	}

	for _, redirect := range redirects {
		for _, symbol := range symbols {
			if symbol.Name == redirect.src {
				redirect.srcVMA = symbol.Value
			}
			if symbol.Name == redirect.dst {
				redirect.dstVMA = symbol.Value
			}
		}

		switch {
		case redirect.srcVMA == 0:
			return fmt.Errorf("%s: could not locate address of %q", imgFile, redirect.src)
		case redirect.dstVMA == 0:
			return fmt.Errorf("%s: could not locate address of %q", imgFile, redirect.dst)
		}
	}

	return nil
}

func main() {
	flag.Parse()
	if matches, _ := filepath.Glob("kernel/"); len(matches) != 1 {
		exit(errors.New("this tool must be run from the kernel root folder"))
	}

	if len(flag.Args()) == 0 {
		exit(errors.New("missing command"))
	}

	cmd := flag.Arg(0)
	var imgFile string
	switch cmd {
	case "count":
	case "populate-table":
		if len(flag.Args()) != 2 {
			exit(errors.New("populate-table requires the path to the kernel image as an argument"))
		}
		imgFile = flag.Arg(1)
	default:
		exit(fmt.Errorf("unknown command %q", cmd))
	}

	goFiles, err := collectGoFiles("kernel/")
	if err != nil {
		exit(err)
	}

	redirects, err := findRedirects(goFiles)
	if err != nil {
		exit(err)
	}

	if cmd == "count" {
		fmt.Printf("%d", len(redirects))
		return
	}

	if err = elfResolveRedirectSymbols(redirects, imgFile); err != nil {
		exit(err)
	}

	if err = elfWriteRedirectTable(redirects, imgFile); err != nil {
		exit(err)
	}
}
