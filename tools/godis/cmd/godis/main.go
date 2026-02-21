// godis compiles Go source files to Dis bytecode for the Inferno OS VM.
//
// Usage:
//
//	godis [-o output.dis] input.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NERVsystems/infernode/tools/godis/compiler"
)

func main() {
	output := flag.String("o", "", "output .dis file (default: input basename + .dis)")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: godis [-o output.dis] input.go\n")
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Default output name
	if *output == "" {
		base := filepath.Base(inputFile)
		*output = strings.TrimSuffix(base, ".go") + ".dis"
	}

	// Read input
	src, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "godis: %v\n", err)
		os.Exit(1)
	}

	// Compile
	c := compiler.New()
	mod, err := c.CompileFile(filepath.Base(inputFile), src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "godis: %v\n", err)
		os.Exit(1)
	}

	// Write output
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
		inputFile, *output, len(mod.Instructions), len(mod.TypeDescs))
}
