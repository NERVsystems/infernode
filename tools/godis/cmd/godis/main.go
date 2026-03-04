// godis compiles Go source files to Dis bytecode for the Inferno OS VM.
//
// Usage:
//
//	godis [-o output.dis] file1.go [file2.go ...]
//	godis -pkg [-o output.dis] ./pkgdir/
//	godis -link pkg=path.dis [-link pkg2=path2.dis] [-o output.dis] file.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NERVsystems/infernode/tools/godis/compiler"
	"github.com/NERVsystems/infernode/tools/godis/dis"
)

// linkFlag collects multiple -link pkg=path.dis flags.
type linkFlag []string

func (f *linkFlag) String() string { return strings.Join(*f, ", ") }
func (f *linkFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func main() {
	output := flag.String("o", "", "output .dis file (default: first input basename + .dis)")
	pkgMode := flag.Bool("pkg", false, "compile a library package (not main) with exported functions")
	var links linkFlag
	flag.Var(&links, "link", "link pre-compiled package: pkg=path.dis (repeatable)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: godis [-o output.dis] file1.go [file2.go ...]\n")
		fmt.Fprintf(os.Stderr, "       godis -pkg [-o output.dis] ./pkgdir/\n")
		fmt.Fprintf(os.Stderr, "       godis -link pkg=path.dis [-o output.dis] file.go\n")
		os.Exit(1)
	}

	c := compiler.New()

	if *pkgMode {
		dir := flag.Arg(0)
		c.BaseDir = dir

		if *output == "" {
			base := filepath.Base(dir)
			*output = base + ".dis"
		}

		mod, err := c.CompilePackage(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "godis: %v\n", err)
			os.Exit(1)
		}

		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "godis: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		if err := mod.Encode(f); err != nil {
			fmt.Fprintf(os.Stderr, "godis: encode: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("godis: %s → %s (%d instructions, %d types, %d exports)\n",
			dir, *output, len(mod.Instructions), len(mod.TypeDescs), len(mod.Links)-1)
		return
	}

	// Read source files
	var filenames []string
	var sources [][]byte
	for i := 0; i < flag.NArg(); i++ {
		inputFile := flag.Arg(i)
		src, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "godis: %v\n", err)
			os.Exit(1)
		}
		filenames = append(filenames, filepath.Base(inputFile))
		sources = append(sources, src)
	}

	if *output == "" {
		base := filepath.Base(flag.Arg(0))
		*output = strings.TrimSuffix(base, ".go") + ".dis"
	}

	c.BaseDir = filepath.Dir(flag.Arg(0))

	// Parse -link flags and compile with cross-module linking if any
	var mod *dis.Module
	var err error

	if len(links) > 0 {
		var linkedPkgs []compiler.LinkedPkg
		for _, l := range links {
			parts := strings.SplitN(l, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "godis: invalid -link flag %q (expected pkg=path.dis)\n", l)
				os.Exit(1)
			}
			linkedPkgs = append(linkedPkgs, compiler.LinkedPkg{
				PkgPath: parts[0],
				DisPath: parts[1],
			})
		}
		mod, err = c.CompileLinked(filenames, sources, linkedPkgs)
	} else {
		mod, err = c.CompileFiles(filenames, sources)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "godis: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "godis: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := mod.Encode(f); err != nil {
		fmt.Fprintf(os.Stderr, "godis: encode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("godis: %s → %s (%d instructions, %d types)\n",
		flag.Arg(0), *output, len(mod.Instructions), len(mod.TypeDescs))
}
