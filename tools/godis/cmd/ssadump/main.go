package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/tools/go/ssa"
)

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, os.Args[1], nil, parser.AllErrors)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	conf := &types.Config{Importer: nil, Error: func(e error) {}}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)

	ssaProg := ssa.NewProgram(fset, ssa.BuilderMode(0))
	ssaPkg := ssaProg.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()

	for _, mem := range ssaPkg.Members {
		if g, ok := mem.(*ssa.Global); ok {
			fmt.Printf("=== GLOBAL: %s (type: %s) ===\n", g.Name(), g.Type())
		}
	}
	for _, mem := range ssaPkg.Members {
		if fn, ok := mem.(*ssa.Function); ok {
			if len(fn.Blocks) > 0 {
				fmt.Printf("\n=== %s ===\n", fn.Name())
				for _, b := range fn.Blocks {
					fmt.Printf("  block %d: %s\n", b.Index, b.Comment)
					for _, instr := range b.Instrs {
						fmt.Printf("    %T: %s\n", instr, instr)
						for _, op := range instr.Operands(nil) {
							if *op != nil {
								fmt.Printf("      operand: %T %s (type: %s)\n", *op, (*op).Name(), (*op).Type())
							}
						}
					}
				}
			}
		}
	}
}
