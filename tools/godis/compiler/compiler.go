package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/NERVsystems/infernode/tools/godis/dis"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Compiler compiles Go source to Dis bytecode.
type Compiler struct {
	strings  map[string]int32 // string literal → MP offset (deduplicating)
	globals  map[string]int32 // global variable name → MP offset
	sysUsed  map[string]int   // Sys function name → LDT index
	mod      *ModuleData
	sysMPOff int32
	errors   []string
}

// New creates a new Compiler.
func New() *Compiler {
	return &Compiler{
		strings: make(map[string]int32),
		globals: make(map[string]int32),
		sysUsed: make(map[string]int),
	}
}

// AllocGlobal allocates storage for a global variable in the module data section.
// Returns the MP offset. Pointer-typed globals are tracked for GC.
func (c *Compiler) AllocGlobal(name string, isPtr bool) int32 {
	if off, ok := c.globals[name]; ok {
		return off
	}
	var off int32
	if isPtr {
		off = c.mod.AllocPointer("global:" + name)
	} else {
		off = c.mod.AllocWord("global:" + name)
	}
	c.globals[name] = off
	return off
}

// GlobalOffset returns the MP offset for a global variable, or -1 if not allocated.
func (c *Compiler) GlobalOffset(name string) (int32, bool) {
	off, ok := c.globals[name]
	return off, ok
}

// AllocString allocates a string literal in the module data section,
// deduplicating identical strings. Returns the MP offset.
func (c *Compiler) AllocString(s string) int32 {
	if off, ok := c.strings[s]; ok {
		return off
	}
	off := c.mod.AllocPointer("str")
	c.strings[s] = off
	return off
}

// compiledFunc holds the compilation result for a single function.
type compiledFunc struct {
	fn     *ssa.Function
	result *lowerResult
}

// CompileFile compiles a single Go source file to a Dis module.
func (c *Compiler) CompileFile(filename string, src []byte) (*dis.Module, error) {
	// Parse
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	// Type-check
	conf := &types.Config{
		Importer: &stubImporter{},
		Error: func(err error) {
			c.errors = append(c.errors, err.Error())
		},
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, err := conf.Check("main", fset, []*ast.File{file}, info)
	if err != nil {
		return nil, fmt.Errorf("typecheck: %w", err)
	}

	// Build SSA
	ssaProg := ssa.NewProgram(fset, ssa.BuilderMode(0))
	ssaPkg := ssaProg.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()

	// Find the main function
	mainFn := ssaPkg.Func("main")
	if mainFn == nil {
		return nil, fmt.Errorf("no main function found")
	}

	// Set up module data
	c.mod = NewModuleData()
	c.sysMPOff = c.mod.AllocPointer("sys") // Sys module ref at MP+0

	// Pre-register Sys functions
	c.scanSysCalls(ssaPkg)

	// Allocate "$Sys" path string in module data
	sysPathOff := c.AllocString("$Sys")

	// Allocate storage for package-level global variables in MP
	for _, mem := range ssaPkg.Members {
		if g, ok := mem.(*ssa.Global); ok {
			elemType := g.Type().(*types.Pointer).Elem()
			dt := GoTypeToDis(elemType)
			c.AllocGlobal(g.Name(), dt.IsPtr)
		}
	}

	// Collect all functions to compile: main first, then others alphabetically
	allFuncs := []*ssa.Function{mainFn}
	for _, mem := range ssaPkg.Members {
		if fn, ok := mem.(*ssa.Function); ok {
			if fn != mainFn && fn.Name() != "init" && len(fn.Blocks) > 0 {
				allFuncs = append(allFuncs, fn)
			}
		}
	}
	sort.Slice(allFuncs[1:], func(i, j int) bool {
		return allFuncs[1+i].Name() < allFuncs[1+j].Name()
	})

	// Phase 1: Compile all functions
	var compiled []compiledFunc
	for _, fn := range allFuncs {
		fl := newFuncLowerer(fn, c, c.sysMPOff, c.sysUsed)
		result, err := fl.lower()
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", fn.Name(), err)
		}
		compiled = append(compiled, compiledFunc{fn, result})
	}

	// Phase 2: Assign type descriptor IDs
	// TD 0 = module data (MP)
	// TD 1..N = function frame type descriptors (main=1, then others)
	// TD N+1.. = call-site type descriptors
	funcTDID := make(map[*ssa.Function]int)
	nextTD := 1
	for _, cf := range compiled {
		funcTDID[cf.fn] = nextTD
		nextTD++
	}
	callTDBase := nextTD

	// Phase 3: Compute function start PCs
	// Layout: [LOAD preamble] [main insts] [func1 insts] [func2 insts] ...
	entryLen := int32(1) // just the LOAD instruction
	funcStartPC := make(map[*ssa.Function]int32)
	offset := entryLen
	for _, cf := range compiled {
		funcStartPC[cf.fn] = offset
		offset += int32(len(cf.result.insts))
	}

	// Phase 4: Patch all instructions
	callTDOffset := callTDBase
	for _, cf := range compiled {
		startPC := funcStartPC[cf.fn]

		// Build set of instruction indices that have funcCallPatches
		patchedInsts := make(map[int]bool)
		for _, p := range cf.result.funcCallPatches {
			patchedInsts[p.instIdx] = true
			inst := &cf.result.insts[p.instIdx]
			switch p.patchKind {
			case patchIFRAME:
				inst.Src = dis.Imm(int32(funcTDID[p.callee]))
			case patchICALL:
				inst.Dst = dis.Imm(funcStartPC[p.callee])
			}
		}

		for i := range cf.result.insts {
			if patchedInsts[i] {
				continue // already patched above
			}
			inst := &cf.result.insts[i]

			// Patch call-site IFRAME type descriptor IDs
			if inst.Op == dis.IFRAME && inst.Src.Mode == dis.AIMM {
				inst.Src.Val += int32(callTDOffset)
			}

			// Patch intra-function branch targets to global PCs
			if inst.Op.IsBranch() && inst.Dst.Mode == dis.AIMM {
				inst.Dst.Val += startPC
			}
		}

		callTDOffset += len(cf.result.callTypeDescs)
	}

	// Phase 5: Build type descriptor array
	var allTypeDescs []dis.TypeDesc
	allTypeDescs = append(allTypeDescs, dis.TypeDesc{}) // TD 0 = MP (filled in later)

	for _, cf := range compiled {
		allTypeDescs = append(allTypeDescs, cf.result.frame.TypeDesc(funcTDID[cf.fn]))
	}

	// Add call-site type descriptors
	tdID := callTDBase
	for _, cf := range compiled {
		for i := range cf.result.callTypeDescs {
			cf.result.callTypeDescs[i].ID = tdID + i
		}
		allTypeDescs = append(allTypeDescs, cf.result.callTypeDescs...)
		tdID += len(cf.result.callTypeDescs)
	}

	allTypeDescs[0] = c.mod.TypeDesc(0)

	// Phase 6: Concatenate instructions
	var allInsts []dis.Inst
	allInsts = append(allInsts,
		dis.NewInst(dis.ILOAD, dis.MP(sysPathOff), dis.Imm(0), dis.MP(c.sysMPOff)),
	)
	for _, cf := range compiled {
		allInsts = append(allInsts, cf.result.insts...)
	}

	// Ensure last instruction is RET
	if len(allInsts) == 0 || allInsts[len(allInsts)-1].Op != dis.IRET {
		allInsts = append(allInsts, dis.Inst0(dis.IRET))
	}

	// Build module name from filename
	moduleName := strings.TrimSuffix(filename, ".go")
	if len(moduleName) > 0 {
		moduleName = strings.ToUpper(moduleName[:1]) + moduleName[1:]
	}

	mainTDID := int32(funcTDID[mainFn])

	m := dis.NewModule(moduleName)
	m.RuntimeFlags = dis.HASLDT
	m.Instructions = allInsts
	m.TypeDescs = allTypeDescs
	m.DataSize = c.mod.Size()
	m.EntryPC = 0
	m.EntryType = mainTDID

	// Build data section with all string literals
	m.Data = c.buildDataSection()

	// Build links (exported functions)
	// Signature 0x4244b354 is for init(ctxt: ref Draw->Context, args: list of string)
	m.Links = []dis.Link{
		{PC: 0, DescID: mainTDID, Sig: 0x4244b354, Name: "init"},
	}

	// Build LDT
	m.LDT = c.buildLDT()

	m.SrcPath = filename

	_ = ssautil.AllFunctions(ssaProg) // for future use

	return m, nil
}

func (c *Compiler) scanSysCalls(pkg *ssa.Package) {
	c.sysUsed["print"] = 0
}

func (c *Compiler) buildDataSection() []dis.DataItem {
	var items []dis.DataItem

	type strEntry struct {
		s   string
		off int32
	}
	var entries []strEntry
	for s, off := range c.strings {
		entries = append(entries, strEntry{s, off})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].off < entries[j].off
	})

	for _, e := range entries {
		items = append(items, dis.DefString(e.off, e.s))
	}
	return items
}

func (c *Compiler) buildLDT() [][]dis.Import {
	if len(c.sysUsed) == 0 {
		return nil
	}

	type entry struct {
		name string
		idx  int
	}
	var entries []entry
	for name, idx := range c.sysUsed {
		entries = append(entries, entry{name, idx})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].idx < entries[j].idx
	})

	var imports []dis.Import
	for _, e := range entries {
		sf := LookupSysFunc(e.name)
		if sf != nil {
			imports = append(imports, dis.Import{
				Sig:  sf.Sig,
				Name: sf.Name,
			})
		}
	}
	return [][]dis.Import{imports}
}

type stubImporter struct{}

func (si *stubImporter) Import(path string) (*types.Package, error) {
	switch path {
	case "fmt":
		pkg := types.NewPackage("fmt", "fmt")
		return pkg, nil
	default:
		return nil, fmt.Errorf("unsupported import: %q", path)
	}
}
