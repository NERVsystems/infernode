package compiler

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NERVsystems/infernode/tools/godis/dis"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// ifaceImpl records one concrete implementation of an interface method.
type ifaceImpl struct {
	tag int32         // type tag ID for the concrete type
	fn  *ssa.Function // the concrete method
}

// Compiler compiles Go source to Dis bytecode.
type Compiler struct {
	strings      map[string]int32        // string literal → MP offset (deduplicating)
	reals        map[float64]int32       // float literal → MP offset (deduplicating)
	globals      map[string]int32        // global variable name → MP offset
	sysUsed      map[string]int          // Sys function name → LDT index
	mod          *ModuleData
	sysMPOff     int32
	errors       []string
	closureMap   map[ssa.Value]*ssa.Function // MakeClosure result → inner function
	closureRetFn map[*ssa.Function]*ssa.Function // func that returns a closure → inner fn
	// Interface dispatch: method name → concrete method function.
	methodMap    map[string]*ssa.Function // "TypeName.MethodName" → *ssa.Function
	// Type tag registry for tagged interface dispatch.
	typeTagMap    map[string]int32   // concrete type name → tag ID (starts at 1)
	typeTagNext   int32              // next tag to allocate
	ifaceDispatch map[string][]ifaceImpl // method name → [{tag, fn}, ...]
	excGlobalOff int32 // MP offset for exception bridge slot (lazy-allocated, 0 = not allocated)
	initFuncs    []*ssa.Function // user-defined init functions (init#1, init#2, ...) to call before main
	closureFuncTags    map[*ssa.Function]int32 // inner function → unique tag for dynamic dispatch
	closureFuncTagNext int32                   // next tag to allocate (starts at 1)
	BaseDir      string // directory containing main package (for resolving local imports)
}

// New creates a new Compiler.
func New() *Compiler {
	return &Compiler{
		strings:       make(map[string]int32),
		reals:         make(map[float64]int32),
		globals:       make(map[string]int32),
		sysUsed:       make(map[string]int),
		closureMap:    make(map[ssa.Value]*ssa.Function),
		closureRetFn:  make(map[*ssa.Function]*ssa.Function),
		methodMap:     make(map[string]*ssa.Function),
		typeTagMap:    make(map[string]int32),
		typeTagNext:   1, // tag 0 = nil interface
		ifaceDispatch:      make(map[string][]ifaceImpl),
		closureFuncTags:    make(map[*ssa.Function]int32),
		closureFuncTagNext: 1, // tag 0 = reserved
	}
}

// AllocTypeTag returns (or allocates) a unique integer tag for a concrete type name.
// Tag 0 is reserved for nil interfaces.
func (c *Compiler) AllocTypeTag(typeName string) int32 {
	if tag, ok := c.typeTagMap[typeName]; ok {
		return tag
	}
	tag := c.typeTagNext
	c.typeTagNext++
	c.typeTagMap[typeName] = tag
	return tag
}

// AllocClosureTag returns (or allocates) a unique integer tag for an inner function.
// Used for dynamic closure dispatch.
func (c *Compiler) AllocClosureTag(fn *ssa.Function) int32 {
	if tag, ok := c.closureFuncTags[fn]; ok {
		return tag
	}
	tag := c.closureFuncTagNext
	c.closureFuncTagNext++
	c.closureFuncTags[fn] = tag
	return tag
}

// registerClosure records that a MakeClosure instruction creates a closure for innerFn.
func (c *Compiler) registerClosure(mc *ssa.MakeClosure, innerFn *ssa.Function) {
	c.closureMap[mc] = innerFn
	// Also track the parent function's return: if this MakeClosure is returned,
	// callers of the parent can resolve the closure target.
	if mc.Parent() != nil {
		c.closureRetFn[mc.Parent()] = innerFn
	}
}

// resolveClosureTarget traces an SSA value back to determine which inner function
// a closure refers to. Returns nil if it cannot be statically resolved.
func (c *Compiler) resolveClosureTarget(v ssa.Value) *ssa.Function {
	// Direct MakeClosure result
	if fn, ok := c.closureMap[v]; ok {
		return fn
	}
	// Return value of a function that always returns a specific closure
	if call, ok := v.(*ssa.Call); ok {
		if callee, ok := call.Call.Value.(*ssa.Function); ok {
			if fn, ok := c.closureRetFn[callee]; ok {
				return fn
			}
		}
	}
	return nil
}

// ResolveInterfaceMethods finds all concrete implementations for a method name
// called on an interface. Returns a list of {tag, fn} pairs — one per concrete type.
func (c *Compiler) ResolveInterfaceMethods(methodName string) []ifaceImpl {
	if impls, ok := c.ifaceDispatch[methodName]; ok && len(impls) > 0 {
		return impls
	}
	return nil
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

// AllocReal allocates a float64 literal in the module data section,
// deduplicating identical values. Returns the MP offset.
func (c *Compiler) AllocReal(val float64) int32 {
	if off, ok := c.reals[val]; ok {
		return off
	}
	off := c.mod.AllocWord("real")
	c.reals[val] = off
	return off
}

// AllocExcGlobal lazily allocates the exception bridge slot in module data.
// This is a WORD (not pointer) used to pass exception values from handler to deferred closures.
func (c *Compiler) AllocExcGlobal() int32 {
	if c.excGlobalOff == 0 {
		c.excGlobalOff = c.mod.AllocWord("excval")
	}
	return c.excGlobalOff
}

// compiledFunc holds the compilation result for a single function.
type compiledFunc struct {
	fn     *ssa.Function
	result *lowerResult
}

// CompileFile compiles a single Go source file to a Dis module.
func (c *Compiler) CompileFile(filename string, src []byte) (*dis.Module, error) {
	return c.CompileFiles([]string{filename}, [][]byte{src})
}

// importResult holds the parsed/type-checked result of a local package import.
type importResult struct {
	pkg   *types.Package
	files []*ast.File
	info  *types.Info
}

// localImporter resolves imports: first checking known stubs, then looking for
// local package directories relative to baseDir.
type localImporter struct {
	stub    stubImporter
	baseDir string              // directory containing main package source
	fset    *token.FileSet      // shared fileset
	cache   map[string]*importResult // import path → result
	errors  *[]string           // shared error list
}

func (li *localImporter) Import(path string) (*types.Package, error) {
	// Try stub first (fmt, strings, math, etc.)
	pkg, err := li.stub.Import(path)
	if err == nil {
		return pkg, nil
	}

	// Check cache
	if result, ok := li.cache[path]; ok {
		return result.pkg, nil
	}

	// Resolve from disk: baseDir/path/
	dir := filepath.Join(li.baseDir, path)
	entries, dirErr := os.ReadDir(dir)
	if dirErr != nil {
		return nil, fmt.Errorf("unsupported import: %q (not a stub and directory %s not found)", path, dir)
	}

	// Parse all .go files in the directory
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		// Skip test files
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		src, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", filePath, readErr)
		}
		f, parseErr := parser.ParseFile(li.fset, entry.Name(), src, parser.AllErrors)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", filePath, parseErr)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .go files in %s", dir)
	}

	// Type-check with recursive import resolution
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	conf := &types.Config{
		Importer: li, // recursive: local packages can import other local packages
		Error: func(err error) {
			*li.errors = append(*li.errors, err.Error())
		},
	}
	// Determine package name from first file
	pkgName := files[0].Name.Name
	typePkg, checkErr := conf.Check(path, li.fset, files, info)
	if checkErr != nil {
		return nil, fmt.Errorf("typecheck %s: %w", pkgName, checkErr)
	}

	li.cache[path] = &importResult{pkg: typePkg, files: files, info: info}
	return typePkg, nil
}

// localPackages returns all locally-resolved packages (not stubs) in dependency order.
func (li *localImporter) localPackages() []*importResult {
	// Return in sorted order for determinism
	var paths []string
	for path := range li.cache {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var results []*importResult
	for _, p := range paths {
		results = append(results, li.cache[p])
	}
	return results
}

// CompileFiles compiles one or more Go source files to a Dis module.
// All files must declare the same package (typically "main").
func (c *Compiler) CompileFiles(filenames []string, sources [][]byte) (*dis.Module, error) {
	fset := token.NewFileSet()

	// Parse all files
	var files []*ast.File
	for i, filename := range filenames {
		file, err := parser.ParseFile(fset, filename, sources[i], parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filename, err)
		}
		files = append(files, file)
	}

	// Verify all files declare the same package
	if len(files) > 1 {
		pkgName := files[0].Name.Name
		for i := 1; i < len(files); i++ {
			if files[i].Name.Name != pkgName {
				return nil, fmt.Errorf("multiple packages: %s and %s", pkgName, files[i].Name.Name)
			}
		}
	}

	// Set up importer
	importer := &localImporter{
		baseDir: c.BaseDir,
		fset:    fset,
		cache:   make(map[string]*importResult),
		errors:  &c.errors,
	}

	// Type-check
	conf := &types.Config{
		Importer: importer,
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

	pkg, err := conf.Check("main", fset, files, info)
	if err != nil {
		return nil, fmt.Errorf("typecheck: %w", err)
	}

	// Build SSA
	ssaProg := ssa.NewProgram(fset, ssa.BuilderMode(0))

	// Create SSA packages for all imports
	localPkgs := make(map[string]*importResult) // path → result for local packages
	for _, imp := range pkg.Imports() {
		if result, ok := importer.cache[imp.Path()]; ok {
			// Local package — build with real AST
			ssaProg.CreatePackage(imp, result.files, result.info, true)
			localPkgs[imp.Path()] = result
		} else {
			// Stub package — no AST needed
			ssaProg.CreatePackage(imp, nil, nil, true)
		}
	}

	// Also create SSA packages for transitive local imports
	// (local packages may import other local packages)
	for path, result := range importer.cache {
		if _, ok := localPkgs[path]; ok {
			continue // already handled
		}
		// This is a transitively imported local package
		ssaProg.CreatePackage(result.pkg, result.files, result.info, true)
		localPkgs[path] = result
		// Also create SSA packages for ITS imports
		for _, transImp := range result.pkg.Imports() {
			if _, ok2 := importer.cache[transImp.Path()]; ok2 {
				continue // will be handled by outer loop or already done
			}
			// Stub import from a local package
			if ssaProg.Package(transImp) == nil {
				ssaProg.CreatePackage(transImp, nil, nil, true)
			}
		}
	}

	ssaPkg := ssaProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()

	// Build local packages too
	for _, result := range localPkgs {
		ssaImpPkg := ssaProg.Package(result.pkg)
		if ssaImpPkg != nil {
			ssaImpPkg.Build()
		}
	}

	// Find the main function
	mainFn := ssaPkg.Func("main")
	if mainFn == nil {
		return nil, fmt.Errorf("no main function found")
	}

	// Set up module data
	c.mod = NewModuleData()
	c.sysMPOff = c.mod.AllocPointer("sys") // Sys module ref at MP+0

	// Pre-register Sys functions (scan all user packages)
	userPkgs := map[*ssa.Package]bool{ssaPkg: true}
	for _, result := range localPkgs {
		ssaImpPkg := ssaProg.Package(result.pkg)
		if ssaImpPkg != nil {
			userPkgs[ssaImpPkg] = true
		}
	}
	c.scanSysCallsMulti(ssaProg, userPkgs)

	// Allocate "$Sys" path string in module data
	sysPathOff := c.AllocString("$Sys")

	// Allocate storage for package-level global variables in MP (main package)
	for _, mem := range ssaPkg.Members {
		if g, ok := mem.(*ssa.Global); ok {
			elemType := g.Type().(*types.Pointer).Elem()
			dt := GoTypeToDis(elemType)
			c.AllocGlobal(g.Name(), dt.IsPtr)
		}
	}

	// Allocate globals from local imported packages (prefixed to avoid collisions)
	for path, result := range localPkgs {
		ssaImpPkg := ssaProg.Package(result.pkg)
		if ssaImpPkg == nil {
			continue
		}
		for _, mem := range ssaImpPkg.Members {
			if g, ok := mem.(*ssa.Global); ok {
				elemType := g.Type().(*types.Pointer).Elem()
				dt := GoTypeToDis(elemType)
				globalName := path + "." + g.Name()
				c.AllocGlobal(globalName, dt.IsPtr)
			}
		}
	}

	// Collect all functions to compile: main first, then others alphabetically.
	// This includes both package-level functions and methods on named types.
	allFuncs := []*ssa.Function{mainFn}
	seen := map[*ssa.Function]bool{mainFn: true}

	// Collect from main package
	c.collectPackageFuncs(ssaProg, ssaPkg, &allFuncs, seen)

	// Collect from local imported packages (dependency order: imports first)
	for _, result := range importer.localPackages() {
		ssaImpPkg := ssaProg.Package(result.pkg)
		if ssaImpPkg != nil {
			c.collectPackageFuncs(ssaProg, ssaImpPkg, &allFuncs, seen)
		}
	}

	// Register synthetic errorString type for error interface dispatch.
	// Must happen after named type method scanning so it doesn't conflict.
	c.RegisterErrorString()

	// Recursively discover anonymous/inner functions (closures)
	for i := 0; i < len(allFuncs); i++ {
		for _, anon := range allFuncs[i].AnonFuncs {
			if !seen[anon] && len(anon.Blocks) > 0 {
				allFuncs = append(allFuncs, anon)
				seen[anon] = true
			}
		}
	}

	sort.Slice(allFuncs[1:], func(i, j int) bool {
		return allFuncs[1+i].Name() < allFuncs[1+j].Name()
	})

	// Pre-scan: discover closure relationships before compilation
	// This is needed because main is compiled first but may call closures
	// created by functions compiled later.
	c.scanClosures(allFuncs)

	// Discover bound method wrappers (e.g. (*T).Method$bound) from MakeClosure targets.
	// These are synthetic functions created by SSA that aren't package members or AnonFuncs.
	for _, innerFn := range c.closureMap {
		if !seen[innerFn] && len(innerFn.Blocks) > 0 {
			allFuncs = append(allFuncs, innerFn)
			seen[innerFn] = true
			// Also discover their anonymous functions recursively
			for i := len(allFuncs) - 1; i < len(allFuncs); i++ {
				for _, anon := range allFuncs[i].AnonFuncs {
					if !seen[anon] && len(anon.Blocks) > 0 {
						allFuncs = append(allFuncs, anon)
						seen[anon] = true
					}
				}
			}
		}
	}

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

			// Patch call-site type descriptor IDs
			// IFRAME/INEW: TD ID is in src operand
			if (inst.Op == dis.IFRAME || inst.Op == dis.INEW) && inst.Src.Mode == dis.AIMM {
				inst.Src.Val += int32(callTDOffset)
			}
			// NEWA: element TD ID is in mid operand
			if inst.Op == dis.INEWA && inst.Mid.Mode == dis.AIMM {
				inst.Mid.Val += int32(callTDOffset)
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

	// Phase 5.5: Collect exception handlers from all functions
	var allHandlers []dis.Handler
	for _, cf := range compiled {
		startPC := funcStartPC[cf.fn]
		for _, h := range cf.result.handlers {
			allHandlers = append(allHandlers, dis.Handler{
				EOffset: h.eoff,
				PC1:     h.pc1 + startPC,
				PC2:     h.pc2 + startPC,
				DescID:  -1, // string-only exceptions
				NE:      0,
				Etab:    nil,
				WildPC:  h.wildPC + startPC,
			})
		}
	}

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

	// Build module name from first filename
	moduleName := strings.TrimSuffix(filenames[0], ".go")
	if len(moduleName) > 0 {
		moduleName = strings.ToUpper(moduleName[:1]) + moduleName[1:]
	}

	mainTDID := int32(funcTDID[mainFn])

	m := dis.NewModule(moduleName)
	m.RuntimeFlags = dis.HASLDT
	if len(allHandlers) > 0 {
		m.RuntimeFlags |= dis.HASEXCEPT
		m.Handlers = allHandlers
	}
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

	m.SrcPath = filenames[0]

	_ = ssautil.AllFunctions(ssaProg) // for future use

	return m, nil
}

// collectPackageFuncs collects functions, methods, and init funcs from an SSA package.
func (c *Compiler) collectPackageFuncs(ssaProg *ssa.Program, ssaPkg *ssa.Package, allFuncs *[]*ssa.Function, seen map[*ssa.Function]bool) {
	for _, mem := range ssaPkg.Members {
		switch m := mem.(type) {
		case *ssa.Function:
			if !seen[m] && m.Name() != "init" && len(m.Blocks) > 0 {
				*allFuncs = append(*allFuncs, m)
				seen[m] = true
				// User-defined init functions appear as init#1, init#2, etc.
				if strings.HasPrefix(m.Name(), "init#") {
					c.initFuncs = append(c.initFuncs, m)
				}
			}
		case *ssa.Type:
			// Collect methods on named types
			nt, ok := m.Type().(*types.Named)
			if !ok {
				continue
			}
			for i := 0; i < nt.NumMethods(); i++ {
				method := ssaProg.FuncValue(nt.Method(i))
				if method != nil && !seen[method] && len(method.Blocks) > 0 {
					*allFuncs = append(*allFuncs, method)
					seen[method] = true
					// Register in methodMap for interface dispatch
					typeName := nt.Obj().Name()
					key := typeName + "." + method.Name()
					c.methodMap[key] = method
					// Register in ifaceDispatch with type tag
					tag := c.AllocTypeTag(typeName)
					c.ifaceDispatch[method.Name()] = append(
						c.ifaceDispatch[method.Name()],
						ifaceImpl{tag: tag, fn: method})
				}
			}
		}
	}
}

// scanClosures pre-scans all functions to discover closure relationships.
// For each function that contains a MakeClosure instruction, record:
// 1. The MakeClosure value → inner function mapping
// 2. If the function returns a MakeClosure, record parent → inner function
// 3. Allocate function tags for dynamic dispatch
func (c *Compiler) scanClosures(allFuncs []*ssa.Function) {
	for _, fn := range allFuncs {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if mc, ok := instr.(*ssa.MakeClosure); ok {
					innerFn := mc.Fn.(*ssa.Function)
					c.closureMap[mc] = innerFn
					c.closureRetFn[fn] = innerFn
					// Pre-allocate function tag for dynamic dispatch
					c.AllocClosureTag(innerFn)
				}
			}
		}
	}
}

func (c *Compiler) scanSysCalls(ssaProg *ssa.Program, pkg *ssa.Package) {
	c.scanSysCallsMulti(ssaProg, map[*ssa.Package]bool{pkg: true})
}

// scanSysCallsMulti scans all functions in the given user packages for Sys module calls.
func (c *Compiler) scanSysCallsMulti(ssaProg *ssa.Program, userPkgs map[*ssa.Package]bool) {
	// Always register print at index 0 (used by println builtin)
	c.sysUsed["print"] = 0

	// Scan all functions (including methods) for sys module calls
	allFns := ssautil.AllFunctions(ssaProg)
	for fn := range allFns {
		if fn.Package() == nil || !userPkgs[fn.Package()] {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(*ssa.Call)
				if !ok {
					continue
				}
				callee, ok := call.Call.Value.(*ssa.Function)
				if !ok {
					continue
				}
				if callee.Package() != nil && callee.Package().Pkg.Path() == "inferno/sys" {
					disName, ok := sysGoToDisName[callee.Name()]
					if ok {
						if _, exists := c.sysUsed[disName]; !exists {
							c.sysUsed[disName] = len(c.sysUsed)
						}
					}
				}
			}
		}
	}
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

	// Float constants
	type realEntry struct {
		val float64
		off int32
	}
	var realEntries []realEntry
	for val, off := range c.reals {
		realEntries = append(realEntries, realEntry{val, off})
	}
	sort.Slice(realEntries, func(i, j int) bool {
		return realEntries[i].off < realEntries[j].off
	})
	for _, e := range realEntries {
		items = append(items, dis.DefReal(e.off, e.val))
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

type stubImporter struct {
	sysPackage *types.Package // cached sys package
}

func (si *stubImporter) Import(path string) (*types.Package, error) {
	switch path {
	case "fmt":
		return buildFmtPackage(), nil
	case "strconv":
		return buildStrconvPackage(), nil
	case "errors":
		return buildErrorsPackage(), nil
	case "strings":
		return buildStringsPackage(), nil
	case "math":
		return buildMathPackage(), nil
	case "os":
		return buildOsPackage(), nil
	case "time":
		return buildTimePackage(), nil
	case "sync":
		return buildSyncPackage(), nil
	case "sort":
		return buildSortPackage(), nil
	case "io":
		return buildIOPackage(), nil
	case "log":
		return buildLogPackage(), nil
	case "unicode":
		return buildUnicodePackage(), nil
	case "unicode/utf8":
		return buildUTF8Package(), nil
	case "path":
		return buildPathPackage(), nil
	case "math/bits":
		return buildMathBitsPackage(), nil
	case "math/rand":
		return buildMathRandPackage(), nil
	case "bytes":
		return buildBytesPackage(), nil
	case "encoding/hex":
		return buildEncodingHexPackage(), nil
	case "encoding/base64":
		return buildEncodingBase64Package(), nil
	case "path/filepath":
		return buildFilepathPackage(), nil
	case "slices":
		return buildSlicesPackage(), nil
	case "maps":
		return buildMapsPackage(), nil
	case "cmp":
		return buildCmpPackage(), nil
	case "context":
		return buildContextPackage(), nil
	case "sync/atomic":
		return buildSyncAtomicPackage(), nil
	case "bufio":
		return buildBufioPackage(), nil
	case "net/url":
		return buildNetURLPackage(), nil
	case "encoding/json":
		return buildEncodingJSONPackage(), nil
	case "runtime":
		return buildRuntimePackage(), nil
	case "reflect":
		return buildReflectPackage(), nil
	case "testing":
		return buildTestingPackage(), nil
	case "os/exec":
		return buildOsExecPackage(), nil
	case "os/signal":
		return buildOsSignalPackage(), nil
	case "io/ioutil":
		return buildIOUtilPackage(), nil
	case "io/fs":
		return buildIOFSPackage(), nil
	case "regexp":
		return buildRegexpPackage(), nil
	case "net/http":
		return buildNetHTTPPackage(), nil
	case "log/slog":
		return buildLogSlogPackage(), nil
	case "flag":
		return buildFlagPackage(), nil
	case "crypto/sha256":
		return buildCryptoSHA256Package(), nil
	case "crypto/md5":
		return buildCryptoMD5Package(), nil
	case "encoding/binary":
		return buildEncodingBinaryPackage(), nil
	case "encoding/csv":
		return buildEncodingCSVPackage(), nil
	case "math/big":
		return buildMathBigPackage(), nil
	case "text/template":
		return buildTextTemplatePackage(), nil
	case "embed":
		return buildEmbedPackage(), nil
	case "hash":
		return buildHashPackage(), nil
	case "hash/crc32":
		return buildHashCRC32Package(), nil
	case "net":
		return buildNetPackage(), nil
	case "inferno/sys":
		if si.sysPackage != nil {
			return si.sysPackage, nil
		}
		si.sysPackage = buildSysPackage()
		return si.sysPackage, nil
	default:
		return nil, fmt.Errorf("unsupported import: %q", path)
	}
}

// buildErrorsPackage creates the type-checked errors package stub
// with the signature for New(text string) error.
func buildErrorsPackage() *types.Package {
	pkg := types.NewPackage("errors", "errors")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func New(text string) error
	newSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "text", types.Typ[types.String])),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
		false)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New", newSig))

	// func Is(err, target error) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Is",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "err", errType),
				types.NewVar(token.NoPos, pkg, "target", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Unwrap(err error) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Unwrap",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func As(err error, target any) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "As",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "err", errType),
				types.NewVar(token.NoPos, pkg, "target", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Join(errs ...error) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Join",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "errs",
				types.NewSlice(errType))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			true)))

	pkg.MarkComplete()
	return pkg
}

// RegisterErrorString registers the synthetic errorString type in the
// interface dispatch table. errorString.Error() is handled inline (fn=nil)
// rather than calling a real function.
func (c *Compiler) RegisterErrorString() {
	tag := c.AllocTypeTag("errorString")
	c.ifaceDispatch["Error"] = append(
		c.ifaceDispatch["Error"],
		ifaceImpl{tag: tag, fn: nil})
}

// buildStrconvPackage creates the type-checked strconv package stub
// with signatures for Itoa, Atoi, FormatInt, FormatBool, ParseBool, etc.
func buildStrconvPackage() *types.Package {
	pkg := types.NewPackage("strconv", "strconv")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Itoa(i int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Itoa",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Atoi(s string) (int, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Atoi",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func FormatInt(i int64, base int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "FormatInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "base", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func FormatBool(b bool) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "FormatBool",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "b", types.Typ[types.Bool])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ParseBool(str string) (bool, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseBool",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "str", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func FormatFloat(f float64, fmt byte, prec, bitSize int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "FormatFloat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "fmt", types.Typ[types.Byte]),
				types.NewVar(token.NoPos, pkg, "prec", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "bitSize", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ParseInt(s string, base int, bitSize int) (int64, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "base", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "bitSize", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ParseUint(s string, base int, bitSize int) (uint64, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseUint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "base", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "bitSize", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func FormatUint(i uint64, base int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "FormatUint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "base", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Quote(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Quote",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Unquote(s string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Unquote",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func AppendInt(dst []byte, i int64, base int) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AppendInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "base", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func ParseFloat(s string, bitSize int) (float64, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseFloat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "bitSize", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildFmtPackage creates the type-checked fmt package stub
// with signatures for Sprintf, Printf, and Println.
func buildFmtPackage() *types.Package {
	pkg := types.NewPackage("fmt", "fmt")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anySlice := types.NewSlice(types.NewInterfaceType(nil, nil))

	// func Sprintf(format string, a ...any) string
	sprintfSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "format", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sprintf", sprintfSig))

	// func Printf(format string, a ...any) (int, error)
	printfSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "format", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Printf", printfSig))

	// func Println(a ...any) (int, error)
	printlnSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Println", printlnSig))

	// func Errorf(format string, a ...any) error
	errorfSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "format", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Errorf", errorfSig))

	// func Sprint(a ...any) string
	sprintSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sprint", sprintSig))

	// func Print(a ...any) (int, error)
	printSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Print", printSig))

	// io.Writer interface for Fprint/Fprintf/Fprintln
	writerIface := types.NewInterfaceType(nil, nil)

	// func Fprintf(w io.Writer, format string, a ...any) (int, error)
	fprintfSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "w", writerIface),
			types.NewVar(token.NoPos, pkg, "format", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprintf", fprintfSig))

	// func Fprintln(w io.Writer, a ...any) (int, error)
	fprintlnSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "w", writerIface),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprintln", fprintlnSig))

	// func Fprint(w io.Writer, a ...any) (int, error)
	fprintSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "w", writerIface),
			types.NewVar(token.NoPos, pkg, "a", anySlice),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType),
		),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprint", fprintSig))

	// func Sprintln(a ...any) string
	sprintlnSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sprintln", sprintlnSig))

	// func Sscan(str string, a ...any) (int, error)
	sscanSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "str", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType)),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sscan", sscanSig))

	// func Sscanf(str string, format string, a ...any) (int, error)
	sscanfSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "str", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "format", types.Typ[types.String]),
			types.NewVar(token.NoPos, pkg, "a", anySlice)),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "", errType)),
		true)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sscanf", sscanfSig))

	// type Stringer interface { String() string }
	stringerIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
				false)),
	}, nil)
	stringerIface.Complete()
	stringerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Stringer", nil),
		stringerIface, nil)
	scope.Insert(stringerType.Obj())

	pkg.MarkComplete()
	return pkg
}

// buildSysPackage creates the type-checked inferno/sys package with
// FD type and function signatures matching the Inferno Sys module.
func buildSysPackage() *types.Package {
	pkg := types.NewPackage("inferno/sys", "sys")

	// FD type: opaque struct wrapping a file descriptor
	fdStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "fd", types.Typ[types.Int], false),
	}, nil)
	fdNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FD", nil), fdStruct, nil)
	fdPtr := types.NewPointer(fdNamed)

	scope := pkg.Scope()
	scope.Insert(fdNamed.Obj())

	// Fildes(fd int) *FD
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fildes",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "fd", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", fdPtr)),
			false)))

	// Open(name string, mode int) *FD
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "mode", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", fdPtr)),
			false)))

	// Write(fd *FD, buf []byte, n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "fd", fdPtr),
				types.NewVar(token.NoPos, nil, "buf", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Read(fd *FD, buf []byte, n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "fd", fdPtr),
				types.NewVar(token.NoPos, nil, "buf", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Fprint(fd *FD, s string, args ...any) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "fd", fdPtr),
				types.NewVar(token.NoPos, nil, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Sleep(ms int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sleep",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "ms", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Millisec() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Millisec",
		types.NewSignatureType(nil, nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Create(name string, mode int, perm int) *FD
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Create",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "mode", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "perm", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", fdPtr)),
			false)))

	// Seek(fd *FD, off int, start int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Seek",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "fd", fdPtr),
				types.NewVar(token.NoPos, nil, "off", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "start", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Bind(name string, old string, flags int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Bind",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "old", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "flags", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Chdir(path string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Chdir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Remove(name string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Remove",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Pipe(fds []FD) int — simplified: takes slice of *FD, returns int
	// In Limbo: pipe(fds: array of ref FD): int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pipe",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "fds", types.NewSlice(fdPtr))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Dup(old int, new int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dup",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "old", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "new_", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Pctl(flags int, movefd []int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pctl",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "flags", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "movefd", types.NewSlice(types.Typ[types.Int]))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// Constants: OREAD=0, OWRITE=1, ORDWR=2, OTRUNC=16, ORCLOSE=64, OEXCL=4096
	scope.Insert(types.NewConst(token.NoPos, pkg, "OREAD", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "OWRITE", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ORDWR", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "OTRUNC", types.Typ[types.Int], constant.MakeInt64(16)))

	// Bind flags
	scope.Insert(types.NewConst(token.NoPos, pkg, "MREPL", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MBEFORE", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MAFTER", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MCREATE", types.Typ[types.Int], constant.MakeInt64(4)))

	// Pctl flags
	scope.Insert(types.NewConst(token.NoPos, pkg, "NEWPGRP", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "FORKNS", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "FORKFD", types.Typ[types.Int], constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NEWFD", types.Typ[types.Int], constant.MakeInt64(8)))

	// Seek constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "SEEKSTART", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SEEKRELA", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SEEKEND", types.Typ[types.Int], constant.MakeInt64(2)))

	pkg.MarkComplete()
	return pkg
}

// buildStringsPackage creates the type-checked strings package stub.
func buildStringsPackage() *types.Package {
	pkg := types.NewPackage("strings", "strings")
	scope := pkg.Scope()

	// func Contains(s, substr string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Contains",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "substr", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func HasPrefix(s, prefix string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "HasPrefix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func HasSuffix(s, suffix string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "HasSuffix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "suffix", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Index(s, substr string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Index",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "substr", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func TrimSpace(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TrimSpace",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Split(s, sep string) []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Split",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "sep", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	// func Join(elems []string, sep string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Join",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "elems", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, pkg, "sep", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Replace(s, old, new string, n int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Replace",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "old", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "new", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ToUpper(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ToUpper",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ToLower(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ToLower",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Repeat(s string, count int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Repeat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "count", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Count(s, substr string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Count",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "substr", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func EqualFold(s, t string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "EqualFold",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "t", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Fields(s string) []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fields",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	// func Trim(s string, cutset string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Trim",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "cutset", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func TrimLeft(s string, cutset string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TrimLeft",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "cutset", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func TrimRight(s string, cutset string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TrimRight",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "cutset", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func TrimPrefix(s, prefix string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TrimPrefix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func TrimSuffix(s, suffix string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TrimSuffix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "suffix", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ReplaceAll(s, old, new string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReplaceAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "old", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "new", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ContainsRune(s string, r rune) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ContainsRune",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func ContainsAny(s, chars string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ContainsAny",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "chars", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IndexByte(s string, c byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IndexByte",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "c", types.Typ[types.Byte])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func IndexRune(s string, r rune) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IndexRune",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func LastIndex(s, substr string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LastIndex",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "substr", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Title(s string) string (deprecated but still used)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Title",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Map(mapping func(rune) rune, s string) string
	mapFuncSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
		types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Rune])),
		false)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Map",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "mapping", mapFuncSig),
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func NewReader(s string) *Reader
	readerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "s", types.Typ[types.String], false),
	}, nil)
	readerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerStruct, nil)
	scope.Insert(readerType.Obj())
	readerPtr := types.NewPointer(readerType)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerPtr)),
			false)))

	// type Builder struct { ... }
	builderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "buf", types.Typ[types.String], false),
	}, nil)
	builderType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Builder", nil),
		builderStruct, nil)
	scope.Insert(builderType.Obj())
	builderPtr := types.NewPointer(builderType)
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteString",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)))
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Len",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil, nil, nil, false)))
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteByte",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "c", types.Typ[types.Byte])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)))
	builderType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteRune",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "b", builderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "r", types.Typ[types.Rune])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)))

	// func NewReplacer(oldnew ...string) *Replacer
	replacerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "r", types.Typ[types.Int], false),
	}, nil)
	replacerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Replacer", nil),
		replacerStruct, nil)
	scope.Insert(replacerType.Obj())
	replacerPtr := types.NewPointer(replacerType)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReplacer",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "oldnew",
				types.NewSlice(types.Typ[types.String]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", replacerPtr)),
			true)))

	// func Cut(s, sep string) (before, after string, found bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Cut",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "sep", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "before", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "after", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "found", types.Typ[types.Bool])),
			false)))

	// func CutPrefix(s, prefix string) (after string, found bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CutPrefix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "after", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "found", types.Typ[types.Bool])),
			false)))

	// func CutSuffix(s, suffix string) (before string, found bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CutSuffix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "suffix", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "before", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "found", types.Typ[types.Bool])),
			false)))

	// func Clone(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Clone",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func SplitN(s, sep string, n int) []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SplitN",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "sep", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	// func SplitAfter(s, sep string) []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SplitAfter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "sep", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildMathPackage creates the type-checked math package stub.
func buildMathPackage() *types.Package {
	pkg := types.NewPackage("math", "math")
	scope := pkg.Scope()

	// func Abs(x float64) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Abs",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func Sqrt(x float64) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sqrt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func Min(x, y float64) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Min",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "y", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func Max(x, y float64) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Max",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "y", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	f64f64 := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
				false)))
	}
	f64f64("Floor")
	f64f64("Ceil")
	f64f64("Round")
	f64f64("Trunc")
	f64f64("Log")
	f64f64("Log2")
	f64f64("Log10")
	f64f64("Exp")
	f64f64("Sin")
	f64f64("Cos")
	f64f64("Tan")

	f64f64f64 := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(
					types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64]),
					types.NewVar(token.NoPos, pkg, "y", types.Typ[types.Float64])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
				false)))
	}
	f64f64f64("Pow")
	f64f64f64("Mod")
	f64f64f64("Remainder")
	f64f64f64("Dim")
	f64f64f64("Copysign")

	// func Inf(sign int) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Inf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "sign", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func NaN() float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NaN",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func IsNaN(f float64) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsNaN",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsInf(f float64, sign int) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsInf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "sign", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Signbit(x float64) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Signbit",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Float64bits(f float64) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64bits",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func Float64frombits(b uint64) float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64frombits",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "b", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// Constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "Pi", types.Typ[types.UntypedFloat], constant.MakeFloat64(3.141592653589793)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "E", types.Typ[types.UntypedFloat], constant.MakeFloat64(2.718281828459045)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Phi", types.Typ[types.UntypedFloat], constant.MakeFloat64(1.618033988749895)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Ln2", types.Typ[types.UntypedFloat], constant.MakeFloat64(0.6931471805599453)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Ln10", types.Typ[types.UntypedFloat], constant.MakeFloat64(2.302585092994046)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Log2E", types.Typ[types.UntypedFloat], constant.MakeFloat64(1.4426950408889634)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Log10E", types.Typ[types.UntypedFloat], constant.MakeFloat64(0.4342944819032518)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxFloat64", types.Typ[types.UntypedFloat], constant.MakeFloat64(1.7976931348623157e+308)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SmallestNonzeroFloat64", types.Typ[types.UntypedFloat], constant.MakeFloat64(5e-324)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxInt", types.Typ[types.UntypedInt], constant.MakeInt64(9223372036854775807)))

	pkg.MarkComplete()
	return pkg
}

// buildOsPackage creates the type-checked os package stub.
func buildOsPackage() *types.Package {
	pkg := types.NewPackage("os", "os")
	scope := pkg.Scope()

	// func Exit(code int)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Exit",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "code", types.Typ[types.Int])),
			nil,
			false)))

	// func Getenv(key string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getenv",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "key", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Getwd() (string, error)
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getwd",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// var Args []string
	scope.Insert(types.NewVar(token.NoPos, pkg, "Args", types.NewSlice(types.Typ[types.String])))

	// var Stdin, Stdout, Stderr *File (simplified as int for now)
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "fd", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	scope.Insert(types.NewVar(token.NoPos, pkg, "Stdin", filePtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Stdout", filePtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Stderr", filePtr))

	// func Mkdir(name string, perm uint32) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Mkdir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "perm", types.Typ[types.Uint32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Remove(name string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Remove",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ReadFile(name string) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteFile(name string, data []byte, perm uint32) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "perm", types.Typ[types.Uint32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Chdir(dir string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Chdir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Rename(oldpath, newpath string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Rename",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "oldpath", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "newpath", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func MkdirAll(path string, perm uint32) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MkdirAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "perm", types.Typ[types.Uint32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func RemoveAll(path string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RemoveAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func TempDir() string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TempDir",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func UserHomeDir() (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "UserHomeDir",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Environ() []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Environ",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	// func Setenv(key, value string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Setenv",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "key", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func IsNotExist(err error) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsNotExist",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsExist(err error) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsExist",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsPermission(err error) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsPermission",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildTimePackage creates the type-checked time package stub.
// Duration is int64 (nanoseconds). Time is a struct wrapping milliseconds.
func buildTimePackage() *types.Package {
	pkg := types.NewPackage("time", "time")
	scope := pkg.Scope()

	// type Duration int64 (nanoseconds)
	durationType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Duration", nil),
		types.Typ[types.Int64], nil)
	scope.Insert(durationType.Obj())

	// type Time struct { msec int }
	timeStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "msec", types.Typ[types.Int], false),
	}, nil)
	timeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Time", nil),
		timeStruct, nil)
	scope.Insert(timeType.Obj())

	// func Now() Time
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Now",
		types.NewSignatureType(nil, nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", timeType)),
			false)))

	// func Since(t Time) Duration
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Since",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "t", timeType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", durationType)),
			false)))

	// func Sleep(d Duration)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sleep",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "d", durationType)),
			nil,
			false)))

	// func After(d Duration) <-chan Time
	chanType := types.NewChan(types.RecvOnly, timeType)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "After",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "d", durationType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", chanType)),
			false)))

	// Duration constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "Nanosecond", durationType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Microsecond", durationType, constant.MakeInt64(1000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Millisecond", durationType, constant.MakeInt64(1000000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Second", durationType, constant.MakeInt64(1000000000)))

	// func (d Duration) Milliseconds() int64
	durationType.AddMethod(types.NewFunc(token.NoPos, pkg, "Milliseconds",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "d", durationType),
			nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64])),
			false)))

	// func (t Time) Sub(u Time) Duration
	timeType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sub",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "t", timeType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "u", timeType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", durationType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildSyncPackage() *types.Package {
	pkg := types.NewPackage("sync", "sync")
	scope := pkg.Scope()

	// type Mutex struct{ locked int }
	mutexStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "locked", types.Typ[types.Int], false),
	}, nil)
	mutexType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Mutex", nil),
		mutexStruct, nil)
	scope.Insert(mutexType.Obj())

	mutexPtr := types.NewPointer(mutexType)
	mutexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Lock",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "m", mutexPtr),
			nil, nil, nil, nil, false)))
	mutexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Unlock",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "m", mutexPtr),
			nil, nil, nil, nil, false)))

	// type WaitGroup struct{ count int; ch chan int }
	wgStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "count", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "ch", types.NewChan(types.SendRecv, types.Typ[types.Int]), false),
	}, nil)
	wgType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "WaitGroup", nil),
		wgStruct, nil)
	scope.Insert(wgType.Obj())

	wgPtr := types.NewPointer(wgType)
	wgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Add",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "wg", wgPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "delta", types.Typ[types.Int])),
			nil, false)))
	wgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Done",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "wg", wgPtr),
			nil, nil, nil, nil, false)))
	wgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Wait",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "wg", wgPtr),
			nil, nil, nil, nil, false)))

	// type Once struct{ done int }
	onceStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "done", types.Typ[types.Int], false),
	}, nil)
	onceType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Once", nil),
		onceStruct, nil)
	scope.Insert(onceType.Obj())

	oncePtr := types.NewPointer(onceType)
	onceType.AddMethod(types.NewFunc(token.NoPos, pkg, "Do",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "o", oncePtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "f",
				types.NewSignatureType(nil, nil, nil, nil, nil, false))),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildSortPackage() *types.Package {
	pkg := types.NewPackage("sort", "sort")
	scope := pkg.Scope()

	// func Ints(x []int)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Ints",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "x",
				types.NewSlice(types.Typ[types.Int]))),
			nil, false)))

	// func Strings(x []string)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Strings",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "x",
				types.NewSlice(types.Typ[types.String]))),
			nil, false)))

	// func Slice(x any, less func(i, j int) bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Slice",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "x", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "less",
					types.NewSignatureType(nil, nil, nil,
						types.NewTuple(
							types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int]),
							types.NewVar(token.NoPos, nil, "j", types.Typ[types.Int])),
						types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
						false))),
			nil, false)))

	// func IntsAreSorted(x []int) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IntsAreSorted",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "x",
				types.NewSlice(types.Typ[types.Int]))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
			false)))

	// func Float64s(x []float64)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64s",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "x",
				types.NewSlice(types.Typ[types.Float64]))),
			nil, false)))

	// func Search(n int, f func(int) bool) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Search",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "f",
					types.NewSignatureType(nil, nil, nil,
						types.NewTuple(types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int])),
						types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
						false))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// func SearchInts(a []int, x int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SearchInts",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "a", types.NewSlice(types.Typ[types.Int])),
				types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// func SearchStrings(a []string, x string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SearchStrings",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "a", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "x", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// func SliceIsSorted(x any, less func(i, j int) bool) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SliceIsSorted",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "x", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "less",
					types.NewSignatureType(nil, nil, nil,
						types.NewTuple(
							types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int]),
							types.NewVar(token.NoPos, nil, "j", types.Typ[types.Int])),
						types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
						false))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
			false)))

	// func Reverse(data Interface) Interface — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Reverse",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "data", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildIOPackage() *types.Package {
	pkg := types.NewPackage("io", "io")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Reader interface { Read(p []byte) (n int, err error) }
	readerIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Read",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewSlice(types.Typ[types.Byte]))),
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
					types.NewVar(token.NoPos, nil, "err", errType)),
				false)),
	}, nil)
	readerIface.Complete()
	readerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerIface, nil)
	scope.Insert(readerType.Obj())

	// type Writer interface { Write(p []byte) (n int, err error) }
	writerIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Write",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewSlice(types.Typ[types.Byte]))),
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
					types.NewVar(token.NoPos, nil, "err", errType)),
				false)),
	}, nil)
	writerIface.Complete()
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerIface, nil)
	scope.Insert(writerType.Obj())

	// var EOF error
	scope.Insert(types.NewVar(token.NoPos, pkg, "EOF", errType))

	// var Discard Writer
	scope.Insert(types.NewVar(token.NoPos, pkg, "Discard", writerType))

	// func ReadAll(r Reader) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteString(w Writer, s string) (n int, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", writerType),
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func Copy(dst Writer, src Reader) (written int64, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Copy",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", writerType),
				types.NewVar(token.NoPos, pkg, "src", readerType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "written", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func NopCloser(r Reader) ReadCloser — simplified as Reader return
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NopCloser",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildLogPackage() *types.Package {
	pkg := types.NewPackage("log", "log")
	scope := pkg.Scope()

	// func Println(v ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Println",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "v",
				types.NewSlice(types.NewInterfaceType(nil, nil)))),
			nil, true)))

	// func Printf(format string, v ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Printf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "format", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "v",
					types.NewSlice(types.NewInterfaceType(nil, nil)))),
			nil, true)))

	// func Fatal(v ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fatal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "v",
				types.NewSlice(types.NewInterfaceType(nil, nil)))),
			nil, true)))

	// func Fatalf(format string, v ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fatalf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "format", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "v",
					types.NewSlice(types.NewInterfaceType(nil, nil)))),
			nil, true)))

	pkg.MarkComplete()
	return pkg
}

func buildUnicodePackage() *types.Package {
	pkg := types.NewPackage("unicode", "unicode")
	scope := pkg.Scope()

	runeBool := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
				false)))
	}
	runeBool("IsLetter")
	runeBool("IsDigit")
	runeBool("IsSpace")
	runeBool("IsUpper")
	runeBool("IsLower")
	runeBool("IsPunct")
	runeBool("IsControl")
	runeBool("IsGraphic")
	runeBool("IsPrint")
	runeBool("IsNumber")

	runeRune := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Rune])),
				false)))
	}
	runeRune("ToUpper")
	runeRune("ToLower")
	runeRune("ToTitle")

	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxRune", types.Typ[types.Rune], constant.MakeInt64(0x10FFFF)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxASCII", types.Typ[types.Rune], constant.MakeInt64(0x7F)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxLatin1", types.Typ[types.Rune], constant.MakeInt64(0xFF)))

	pkg.MarkComplete()
	return pkg
}

func buildUTF8Package() *types.Package {
	pkg := types.NewPackage("unicode/utf8", "utf8")
	scope := pkg.Scope()

	// func RuneLen(r rune) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RuneLen",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func RuneCountInString(s string) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RuneCountInString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func RuneCount(p []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RuneCount",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func ValidString(s string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ValidString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Valid(p []byte) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Valid",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func DecodeRuneInString(s string) (rune, int)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DecodeRuneInString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Rune]),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func EncodeRune(p []byte, r rune) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "EncodeRune",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Rune])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// Constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "RuneSelf", types.Typ[types.Int], constant.MakeInt64(0x80)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MaxRune", types.Typ[types.Rune], constant.MakeInt64(0x10FFFF)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "UTFMax", types.Typ[types.Int], constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "RuneError", types.Typ[types.Rune], constant.MakeInt64(0xFFFD)))

	pkg.MarkComplete()
	return pkg
}

func buildPathPackage() *types.Package {
	pkg := types.NewPackage("path", "path")
	scope := pkg.Scope()

	ss := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
				false)))
	}
	ss("Base")
	ss("Dir")
	ss("Ext")
	ss("Clean")

	// func IsAbs(path string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsAbs",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Join(elem ...string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Join",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "elem", types.NewSlice(types.Typ[types.String]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			true)))

	// func Split(path string) (dir, file string)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Split",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "file", types.Typ[types.String])),
			false)))

	// func Match(pattern, name string) (bool, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Match",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "pattern", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, pkg, "", types.Universe.Lookup("error").Type())),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildMathBitsPackage() *types.Package {
	pkg := types.NewPackage("math/bits", "bits")
	scope := pkg.Scope()

	uintInt := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
				false)))
	}
	uintInt("OnesCount")
	uintInt("LeadingZeros")
	uintInt("TrailingZeros")
	uintInt("Len")

	uint64Int := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint64])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
				false)))
	}
	uint64Int("OnesCount64")
	uint64Int("LeadingZeros64")
	uint64Int("TrailingZeros64")
	uint64Int("Len64")

	// func RotateLeft(x uint, k int) uint
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RotateLeft",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint]),
				types.NewVar(token.NoPos, pkg, "k", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint])),
			false)))

	// func RotateLeft64(x uint64, k int) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RotateLeft64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "k", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func ReverseBytes64(x uint64) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReverseBytes64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func Reverse64(x uint64) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Reverse64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// Constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "UintSize", types.Typ[types.Int], constant.MakeInt64(64)))

	pkg.MarkComplete()
	return pkg
}

func buildMathRandPackage() *types.Package {
	pkg := types.NewPackage("math/rand", "rand")
	scope := pkg.Scope()

	// func Intn(n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Intn",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Int() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Float64() float64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])),
			false)))

	// func Seed(seed int64)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Seed",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "seed", types.Typ[types.Int64])),
			nil, false)))

	// func Intn31() int32
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int31",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// func Int63() int64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int63",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildBytesPackage() *types.Package {
	pkg := types.NewPackage("bytes", "bytes")
	scope := pkg.Scope()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// func Contains(b, subslice []byte) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Contains",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "b", byteSlice),
				types.NewVar(token.NoPos, pkg, "subslice", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Equal(a, b []byte) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Equal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "a", byteSlice),
				types.NewVar(token.NoPos, pkg, "b", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Compare(a, b []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compare",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "a", byteSlice),
				types.NewVar(token.NoPos, pkg, "b", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	bbBool := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(
					types.NewVar(token.NoPos, pkg, "s", byteSlice),
					types.NewVar(token.NoPos, pkg, "prefix", byteSlice)),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
				false)))
	}
	bbBool("HasPrefix")
	bbBool("HasSuffix")

	// func Index(s, sep []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Index",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", byteSlice),
				types.NewVar(token.NoPos, pkg, "sep", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func IndexByte(b []byte, c byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IndexByte",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "b", byteSlice),
				types.NewVar(token.NoPos, pkg, "c", types.Typ[types.Byte])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Count(s, sep []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Count",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", byteSlice),
				types.NewVar(token.NoPos, pkg, "sep", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	bbs := func(name string) {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "s", byteSlice)),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", byteSlice)),
				false)))
	}
	bbs("TrimSpace")
	bbs("ToLower")
	bbs("ToUpper")

	// func Repeat(b []byte, count int) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Repeat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "b", byteSlice),
				types.NewVar(token.NoPos, pkg, "count", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", byteSlice)),
			false)))

	// func Join(s [][]byte, sep []byte) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Join",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", types.NewSlice(byteSlice)),
				types.NewVar(token.NoPos, pkg, "sep", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", byteSlice)),
			false)))

	// func Split(s, sep []byte) [][]byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Split",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", byteSlice),
				types.NewVar(token.NoPos, pkg, "sep", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(byteSlice))),
			false)))

	// func Replace(s, old, new []byte, n int) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Replace",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", byteSlice),
				types.NewVar(token.NoPos, pkg, "old", byteSlice),
				types.NewVar(token.NoPos, pkg, "new", byteSlice),
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", byteSlice)),
			false)))

	// func ReplaceAll(s, old, new []byte) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReplaceAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", byteSlice),
				types.NewVar(token.NoPos, pkg, "old", byteSlice),
				types.NewVar(token.NoPos, pkg, "new", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", byteSlice)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildEncodingHexPackage() *types.Package {
	pkg := types.NewPackage("encoding/hex", "hex")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func EncodeToString(src []byte) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "EncodeToString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "src", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func DecodeString(s string) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DecodeString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func EncodedLen(n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "EncodedLen",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func DecodedLen(x int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DecodedLen",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildEncodingBase64Package() *types.Package {
	pkg := types.NewPackage("encoding/base64", "base64")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Encoding struct{ ... } (opaque)
	encStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "alphabet", types.Typ[types.String], false),
	}, nil)
	encType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Encoding", nil),
		encStruct, nil)
	scope.Insert(encType.Obj())
	encPtr := types.NewPointer(encType)

	// Methods
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "EncodeToString",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "src", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "DecodeString",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "EncodedLen",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "DecodedLen",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// var StdEncoding *Encoding
	scope.Insert(types.NewVar(token.NoPos, pkg, "StdEncoding", encPtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "URLEncoding", encPtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "RawStdEncoding", encPtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "RawURLEncoding", encPtr))

	pkg.MarkComplete()
	return pkg
}

// buildFilepathPackage creates the type-checked path/filepath package stub.
func buildFilepathPackage() *types.Package {
	pkg := types.NewPackage("path/filepath", "filepath")
	scope := pkg.Scope()

	// func Join(elem ...string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Join",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "elem",
				types.NewSlice(types.Typ[types.String]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			true)))

	// func Base(path string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Base",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Dir(path string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Ext(path string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Ext",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Clean(path string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Clean",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Abs(path string) (string, error)
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Abs",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Rel(basepath, targpath string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Rel",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "basepath", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "targpath", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func IsAbs(path string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsAbs",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// const Separator = '/'
	scope.Insert(types.NewConst(token.NoPos, pkg, "Separator", types.Typ[types.UntypedRune],
		constant.MakeInt64('/')))

	pkg.MarkComplete()
	return pkg
}

// buildSlicesPackage creates the type-checked slices package stub (Go 1.21+).
func buildSlicesPackage() *types.Package {
	pkg := types.NewPackage("slices", "slices")
	scope := pkg.Scope()

	// Note: slices functions are generic in real Go, but we stub them with
	// concrete types. The compiler handles type specialization at call sites.

	// func Contains[S ~[]E, E comparable](s S, v E) bool
	// Stubbed as Contains([]any, any) bool
	anySlice := types.NewSlice(types.NewInterfaceType(nil, nil))
	anyType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Contains",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", anySlice),
				types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Index[S ~[]E, E comparable](s S, v E) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Index",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s", anySlice),
				types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Reverse[S ~[]E, E any](s S)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Reverse",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", anySlice)),
			nil, false)))

	// func Sort[S ~[]E, E cmp.Ordered](s S)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sort",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", anySlice)),
			nil, false)))

	// func Compact[S ~[]E, E comparable](s S) S
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compact",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", anySlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anySlice)),
			false)))

	// func Equal[S ~[]E, E comparable](s1, s2 S) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Equal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "s1", anySlice),
				types.NewVar(token.NoPos, pkg, "s2", anySlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildMapsPackage creates the type-checked maps package stub (Go 1.21+).
func buildMapsPackage() *types.Package {
	pkg := types.NewPackage("maps", "maps")
	scope := pkg.Scope()

	// Stubbed with interface types for generic functions
	anyType := types.NewInterfaceType(nil, nil)
	anySlice := types.NewSlice(anyType)
	anyMap := types.NewMap(anyType, anyType)

	// func Keys[M ~map[K]V, K comparable, V any](m M) []K
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Keys",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "m", anyMap)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anySlice)),
			false)))

	// func Values[M ~map[K]V, K comparable, V any](m M) []V
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Values",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "m", anyMap)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anySlice)),
			false)))

	// func Clone[M ~map[K]V, K comparable, V any](m M) M
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Clone",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "m", anyMap)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anyMap)),
			false)))

	// func Equal[M1, M2 ~map[K]V, K, V comparable](m1 M1, m2 M2) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Equal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "m1", anyMap),
				types.NewVar(token.NoPos, pkg, "m2", anyMap)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Copy[M1 ~map[K]V, M2 ~map[K]V, K comparable, V any](dst M1, src M2)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Copy",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", anyMap),
				types.NewVar(token.NoPos, pkg, "src", anyMap)),
			nil, false)))

	// func DeleteFunc[M ~map[K]V, K comparable, V any](m M, del func(K, V) bool)
	// Skipping DeleteFunc for now due to function type complexity

	pkg.MarkComplete()
	return pkg
}

// buildCmpPackage creates the type-checked cmp package stub (Go 1.21+).
func buildCmpPackage() *types.Package {
	pkg := types.NewPackage("cmp", "cmp")
	scope := pkg.Scope()

	// type Ordered = comparable (simplified as interface{})
	orderedType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Ordered", nil),
		types.NewInterfaceType(nil, nil), nil)
	scope.Insert(orderedType.Obj())

	// func Compare[T Ordered](x, y T) int
	anyType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compare",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", anyType),
				types.NewVar(token.NoPos, pkg, "y", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Less[T Ordered](x, y T) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Less",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", anyType),
				types.NewVar(token.NoPos, pkg, "y", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Or[T comparable](vals ...T) T
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Or",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "vals",
				types.NewSlice(anyType))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anyType)),
			true)))

	pkg.MarkComplete()
	return pkg
}

// buildContextPackage creates the type-checked context package stub.
func buildContextPackage() *types.Package {
	pkg := types.NewPackage("context", "context")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Context interface { ... }
	ctxIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Err",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
				false)),
	}, nil)
	ctxIface.Complete()
	ctxType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Context", nil),
		ctxIface, nil)
	scope.Insert(ctxType.Obj())

	// type CancelFunc func()
	cancelSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	cancelType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "CancelFunc", nil),
		cancelSig, nil)
	scope.Insert(cancelType.Obj())

	// func Background() Context
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Background",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ctxType)),
			false)))

	// func TODO() Context
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TODO",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ctxType)),
			false)))

	// func WithCancel(parent Context) (Context, CancelFunc)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WithCancel",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "parent", ctxType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", ctxType),
				types.NewVar(token.NoPos, pkg, "", cancelType)),
			false)))

	// func WithValue(parent Context, key, val any) Context
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WithValue",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "parent", ctxType),
				types.NewVar(token.NoPos, pkg, "key", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "val", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ctxType)),
			false)))

	// var Canceled error
	scope.Insert(types.NewVar(token.NoPos, pkg, "Canceled", errType))

	pkg.MarkComplete()
	return pkg
}

// buildSyncAtomicPackage creates the type-checked sync/atomic package stub.
func buildSyncAtomicPackage() *types.Package {
	pkg := types.NewPackage("sync/atomic", "atomic")
	scope := pkg.Scope()

	// func AddInt32(addr *int32, delta int32) int32
	int32Ptr := types.NewPointer(types.Typ[types.Int32])
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddInt32",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int32Ptr),
				types.NewVar(token.NoPos, pkg, "delta", types.Typ[types.Int32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// func AddInt64(addr *int64, delta int64) int64
	int64Ptr := types.NewPointer(types.Typ[types.Int64])
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddInt64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int64Ptr),
				types.NewVar(token.NoPos, pkg, "delta", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64])),
			false)))

	// func LoadInt32(addr *int32) int32
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LoadInt32",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "addr", int32Ptr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// func LoadInt64(addr *int64) int64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LoadInt64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "addr", int64Ptr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64])),
			false)))

	// func StoreInt32(addr *int32, val int32)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StoreInt32",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int32Ptr),
				types.NewVar(token.NoPos, pkg, "val", types.Typ[types.Int32])),
			nil, false)))

	// func StoreInt64(addr *int64, val int64)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StoreInt64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int64Ptr),
				types.NewVar(token.NoPos, pkg, "val", types.Typ[types.Int64])),
			nil, false)))

	// func CompareAndSwapInt32(addr *int32, old, new int32) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CompareAndSwapInt32",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int32Ptr),
				types.NewVar(token.NoPos, pkg, "old", types.Typ[types.Int32]),
				types.NewVar(token.NoPos, pkg, "new_", types.Typ[types.Int32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func CompareAndSwapInt64(addr *int64, old, new int64) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CompareAndSwapInt64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", int64Ptr),
				types.NewVar(token.NoPos, pkg, "old", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "new_", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildBufioPackage creates the type-checked bufio package stub.
func buildBufioPackage() *types.Package {
	pkg := types.NewPackage("bufio", "bufio")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// Import io types for Reader/Writer references
	readerType := types.NewInterfaceType(nil, nil)
	writerType := types.NewInterfaceType(nil, nil)

	// type Scanner struct { ... }
	scannerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "src", readerType, false),
	}, nil)
	scannerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Scanner", nil),
		scannerStruct, nil)
	scope.Insert(scannerType.Obj())
	scannerPtr := types.NewPointer(scannerType)

	// func NewScanner(r io.Reader) *Scanner
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewScanner",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", scannerPtr)),
			false)))

	// type Reader struct { ... }
	readerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "rd", readerType, false),
	}, nil)
	bufReaderType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerStruct, nil)
	scope.Insert(bufReaderType.Obj())
	bufReaderPtr := types.NewPointer(bufReaderType)

	// func NewReader(rd io.Reader) *Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "rd", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", bufReaderPtr)),
			false)))

	// type Writer struct { ... }
	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "wr", writerType, false),
	}, nil)
	bufWriterType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(bufWriterType.Obj())
	bufWriterPtr := types.NewPointer(bufWriterType)

	// func NewWriter(w io.Writer) *Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", writerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", bufWriterPtr)),
			false)))

	// func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ScanLines",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "atEOF", types.Typ[types.Bool])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "advance", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "token_", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func ScanWords similar signature
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ScanWords",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "atEOF", types.Typ[types.Bool])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "advance", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "token_", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildNetURLPackage creates the type-checked net/url package stub.
func buildNetURLPackage() *types.Package {
	pkg := types.NewPackage("net/url", "url")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type URL struct { Scheme, Host, Path, RawQuery, Fragment string }
	urlStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Scheme", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Host", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "RawQuery", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Fragment", types.Typ[types.String], false),
	}, nil)
	urlType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "URL", nil),
		urlStruct, nil)
	scope.Insert(urlType.Obj())
	urlPtr := types.NewPointer(urlType)

	// func Parse(rawURL string) (*URL, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Parse",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "rawURL", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", urlPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func QueryEscape(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "QueryEscape",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func QueryUnescape(s string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "QueryUnescape",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func PathEscape(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PathEscape",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func PathUnescape(s string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PathUnescape",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type Values map[string][]string
	valuesType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Values", nil),
		types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.String])), nil)
	scope.Insert(valuesType.Obj())

	// func ParseQuery(query string) (Values, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseQuery",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "query", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", valuesType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildEncodingJSONPackage creates the type-checked encoding/json package stub.
func buildEncodingJSONPackage() *types.Package {
	pkg := types.NewPackage("encoding/json", "json")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anyType := types.NewInterfaceType(nil, nil)

	// func Marshal(v any) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Marshal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func MarshalIndent(v any, prefix, indent string) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MarshalIndent",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "v", anyType),
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "indent", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Unmarshal(data []byte, v any) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Unmarshal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Valid(data []byte) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Valid",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func Compact(dst *bytes.Buffer, src []byte) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compact",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", anyType),
				types.NewVar(token.NoPos, pkg, "src", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildRuntimePackage creates the type-checked runtime package stub.
func buildRuntimePackage() *types.Package {
	pkg := types.NewPackage("runtime", "runtime")
	scope := pkg.Scope()

	// func GOMAXPROCS(n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "GOMAXPROCS",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NumCPU() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NumCPU",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NumGoroutine() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NumGoroutine",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Gosched()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Gosched",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func GC()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "GC",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func Goexit()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Goexit",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// var GOOS string
	scope.Insert(types.NewVar(token.NoPos, pkg, "GOOS", types.Typ[types.String]))

	// var GOARCH string
	scope.Insert(types.NewVar(token.NoPos, pkg, "GOARCH", types.Typ[types.String]))

	// func Caller(skip int) (pc uintptr, file string, line int, ok bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Caller",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "skip", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "pc", types.Typ[types.Uintptr]),
				types.NewVar(token.NoPos, pkg, "file", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "line", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "ok", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildReflectPackage creates a minimal type-checked reflect package stub.
func buildReflectPackage() *types.Package {
	pkg := types.NewPackage("reflect", "reflect")
	scope := pkg.Scope()

	// type Kind uint
	kindType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Kind", nil),
		types.Typ[types.Uint], nil)
	scope.Insert(kindType.Obj())

	// type Type interface { ... }
	typeIface := types.NewInterfaceType(nil, nil)
	typeIface.Complete()
	typeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Type", nil),
		typeIface, nil)
	scope.Insert(typeType.Obj())

	// type Value struct { ... }
	valueStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "val", types.Typ[types.Int], false),
	}, nil)
	valueType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Value", nil),
		valueStruct, nil)
	scope.Insert(valueType.Obj())

	// func TypeOf(i any) Type
	anyType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TypeOf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "i", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", typeType)),
			false)))

	// func ValueOf(i any) Value
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ValueOf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "i", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueType)),
			false)))

	// func DeepEqual(x, y any) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DeepEqual",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", anyType),
				types.NewVar(token.NoPos, pkg, "y", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildTestingPackage creates a minimal type-checked testing package stub.
func buildTestingPackage() *types.Package {
	pkg := types.NewPackage("testing", "testing")
	scope := pkg.Scope()

	// type T struct { ... }
	tStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "name", types.Typ[types.String], false),
	}, nil)
	tType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "T", nil),
		tStruct, nil)
	scope.Insert(tType.Obj())

	// type B struct { ... }
	bStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "name", types.Typ[types.String], false),
	}, nil)
	bType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "B", nil),
		bStruct, nil)
	scope.Insert(bType.Obj())

	pkg.MarkComplete()
	return pkg
}

// buildOsExecPackage creates the type-checked os/exec package stub.
func buildOsExecPackage() *types.Package {
	pkg := types.NewPackage("os/exec", "exec")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Cmd struct { Path string; Args []string }
	cmdStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Args", types.NewSlice(types.Typ[types.String]), false),
	}, nil)
	cmdType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Cmd", nil),
		cmdStruct, nil)
	scope.Insert(cmdType.Obj())
	cmdPtr := types.NewPointer(cmdType)

	// func Command(name string, arg ...string) *Cmd
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Command",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "arg", types.NewSlice(types.Typ[types.String]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", cmdPtr)),
			true)))

	// func LookPath(file string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LookPath",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "file", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildOsSignalPackage creates the type-checked os/signal package stub.
func buildOsSignalPackage() *types.Package {
	pkg := types.NewPackage("os/signal", "signal")
	scope := pkg.Scope()

	// func Notify(c chan<- os.Signal, sig ...os.Signal)
	// Simplified: use interface{} for Signal type
	sigType := types.NewInterfaceType(nil, nil)
	sigChan := types.NewChan(types.SendOnly, sigType)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Notify",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "c", sigChan),
				types.NewVar(token.NoPos, pkg, "sig", types.NewSlice(sigType))),
			nil, true)))

	// func Stop(c chan<- os.Signal)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Stop",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "c", sigChan)),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

// buildIOUtilPackage creates the type-checked io/ioutil package stub (deprecated).
func buildIOUtilPackage() *types.Package {
	pkg := types.NewPackage("io/ioutil", "ioutil")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func ReadFile(filename string) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "filename", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteFile(filename string, data []byte, perm uint32) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "filename", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "perm", types.Typ[types.Uint32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ReadAll(r io.Reader) ([]byte, error)
	readerType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func TempDir(dir, pattern string) (string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TempDir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "pattern", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// var Discard io.Writer
	scope.Insert(types.NewVar(token.NoPos, pkg, "Discard", types.NewInterfaceType(nil, nil)))

	// func NopCloser(r io.Reader) io.ReadCloser — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NopCloser",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildIOFSPackage creates the type-checked io/fs package stub.
func buildIOFSPackage() *types.Package {
	pkg := types.NewPackage("io/fs", "fs")
	scope := pkg.Scope()

	// type FileMode uint32
	fileModeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FileMode", nil),
		types.Typ[types.Uint32], nil)
	scope.Insert(fileModeType.Obj())

	// type FileInfo interface { ... }
	fileInfoIface := types.NewInterfaceType(nil, nil)
	fileInfoIface.Complete()
	fileInfoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FileInfo", nil),
		fileInfoIface, nil)
	scope.Insert(fileInfoType.Obj())

	// type FS interface { Open(name string) (File, error) }
	fsIface := types.NewInterfaceType(nil, nil)
	fsIface.Complete()
	fsType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FS", nil),
		fsIface, nil)
	scope.Insert(fsType.Obj())

	// type DirEntry interface { ... }
	dirEntryIface := types.NewInterfaceType(nil, nil)
	dirEntryIface.Complete()
	dirEntryType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "DirEntry", nil),
		dirEntryIface, nil)
	scope.Insert(dirEntryType.Obj())

	// var ErrNotExist, ErrExist, ErrPermission error
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrNotExist", errType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrExist", errType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrPermission", errType))

	pkg.MarkComplete()
	return pkg
}

// buildRegexpPackage creates the type-checked regexp package stub.
func buildRegexpPackage() *types.Package {
	pkg := types.NewPackage("regexp", "regexp")
	scope := pkg.Scope()

	// type Regexp struct { ... }
	regexpStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "pattern", types.Typ[types.String], false),
	}, nil)
	regexpType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Regexp", nil),
		regexpStruct, nil)
	scope.Insert(regexpType.Obj())
	regexpPtr := types.NewPointer(regexpType)

	// func Compile(expr string) (*Regexp, error)
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "expr", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", regexpPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func MustCompile(str string) *Regexp
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MustCompile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "str", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", regexpPtr)),
			false)))

	// func MatchString(pattern string, s string) (bool, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MatchString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "pattern", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func QuoteMeta(s string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "QuoteMeta",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildNetHTTPPackage creates a minimal type-checked net/http package stub.
func buildNetHTTPPackage() *types.Package {
	pkg := types.NewPackage("net/http", "http")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Header map[string][]string
	headerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Header", nil),
		types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.String])), nil)
	scope.Insert(headerType.Obj())

	// type Request struct { Method, URL string; Header Header }
	reqStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Method", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Host", types.Typ[types.String], false),
	}, nil)
	reqType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Request", nil),
		reqStruct, nil)
	scope.Insert(reqType.Obj())

	// type Response struct { StatusCode int; Body io.ReadCloser }
	respStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "StatusCode", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Status", types.Typ[types.String], false),
	}, nil)
	respType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Response", nil),
		respStruct, nil)
	scope.Insert(respType.Obj())
	respPtr := types.NewPointer(respType)

	// func Get(url string) (*Response, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Get",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "url", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", respPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Post(url, contentType string, body io.Reader) (*Response, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Post",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "url", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "contentType", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "body", types.NewInterfaceType(nil, nil))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", respPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Status codes
	scope.Insert(types.NewConst(token.NoPos, pkg, "StatusOK", types.Typ[types.Int],
		constant.MakeInt64(200)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "StatusNotFound", types.Typ[types.Int],
		constant.MakeInt64(404)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "StatusInternalServerError", types.Typ[types.Int],
		constant.MakeInt64(500)))

	// type HandlerFunc func(ResponseWriter, *Request)
	// Simplified: just register the type name
	handlerFuncType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "HandlerFunc", nil),
		types.NewSignatureType(nil, nil, nil, nil, nil, false), nil)
	scope.Insert(handlerFuncType.Obj())

	pkg.MarkComplete()
	return pkg
}

// buildLogSlogPackage creates the type-checked log/slog package stub.
func buildLogSlogPackage() *types.Package {
	pkg := types.NewPackage("log/slog", "slog")
	scope := pkg.Scope()

	anySlice := types.NewSlice(types.NewInterfaceType(nil, nil))

	// func Info(msg string, args ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Info",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "msg", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "args", anySlice)),
			nil, true)))

	// func Warn(msg string, args ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Warn",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "msg", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "args", anySlice)),
			nil, true)))

	// func Error(msg string, args ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "msg", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "args", anySlice)),
			nil, true)))

	// func Debug(msg string, args ...any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Debug",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "msg", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "args", anySlice)),
			nil, true)))

	// func String(key, value string) Attr — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "key", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// func Int(key string, value int) Attr
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "key", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildFlagPackage creates the type-checked flag package stub.
func buildFlagPackage() *types.Package {
	pkg := types.NewPackage("flag", "flag")
	scope := pkg.Scope()

	// func Parse()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Parse",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func String(name string, value string, usage string) *string
	strPtr := types.NewPointer(types.Typ[types.String])
	scope.Insert(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "usage", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", strPtr)),
			false)))

	// func Int(name string, value int, usage string) *int
	intPtr := types.NewPointer(types.Typ[types.Int])
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "usage", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", intPtr)),
			false)))

	// func Bool(name string, value bool, usage string) *bool
	boolPtr := types.NewPointer(types.Typ[types.Bool])
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Bool",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, pkg, "usage", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", boolPtr)),
			false)))

	// func Arg(i int) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Arg",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func Args() []string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Args",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String]))),
			false)))

	// func NArg() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NArg",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NFlag() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NFlag",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildCryptoSHA256Package creates the type-checked crypto/sha256 package stub.
func buildCryptoSHA256Package() *types.Package {
	pkg := types.NewPackage("crypto/sha256", "sha256")
	scope := pkg.Scope()

	// const Size = 32
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int],
		constant.MakeInt64(32)))

	// func Sum256(data []byte) [32]byte — simplified as func([]byte) []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum256",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewArray(types.Typ[types.Byte], 32))),
			false)))

	// func New() hash.Hash — simplified as func() interface{}
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildCryptoMD5Package creates the type-checked crypto/md5 package stub.
func buildCryptoMD5Package() *types.Package {
	pkg := types.NewPackage("crypto/md5", "md5")
	scope := pkg.Scope()

	// const Size = 16
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int],
		constant.MakeInt64(16)))

	// func Sum(data []byte) [16]byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewArray(types.Typ[types.Byte], 16))),
			false)))

	// func New() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildEncodingBinaryPackage creates the type-checked encoding/binary package stub.
func buildEncodingBinaryPackage() *types.Package {
	pkg := types.NewPackage("encoding/binary", "binary")
	scope := pkg.Scope()

	// type ByteOrder interface { ... }
	byteOrderIface := types.NewInterfaceType(nil, nil)
	byteOrderIface.Complete()
	byteOrderType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ByteOrder", nil),
		byteOrderIface, nil)
	scope.Insert(byteOrderType.Obj())

	// var BigEndian, LittleEndian ByteOrder
	scope.Insert(types.NewVar(token.NoPos, pkg, "BigEndian", byteOrderType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "LittleEndian", byteOrderType))

	// func Write(w io.Writer, order ByteOrder, data any) error
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "order", byteOrderType),
				types.NewVar(token.NoPos, pkg, "data", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Read(r io.Reader, order ByteOrder, data any) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "order", byteOrderType),
				types.NewVar(token.NoPos, pkg, "data", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func PutUvarint(buf []byte, x uint64) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PutUvarint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "buf", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Uvarint(buf []byte) (uint64, int)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Uvarint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "buf", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildEncodingCSVPackage creates the type-checked encoding/csv package stub.
func buildEncodingCSVPackage() *types.Package {
	pkg := types.NewPackage("encoding/csv", "csv")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Reader struct { ... }
	readerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil), false),
	}, nil)
	readerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerStruct, nil)
	scope.Insert(readerType.Obj())
	readerPtr := types.NewPointer(readerType)

	// func NewReader(r io.Reader) *Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerPtr)),
			false)))

	// type Writer struct { ... }
	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil), false),
	}, nil)
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(writerType.Obj())
	writerPtr := types.NewPointer(writerType)

	// func NewWriter(w io.Writer) *Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", writerPtr)),
			false)))
	_ = errType

	pkg.MarkComplete()
	return pkg
}

// buildMathBigPackage creates the type-checked math/big package stub.
func buildMathBigPackage() *types.Package {
	pkg := types.NewPackage("math/big", "big")
	scope := pkg.Scope()

	// type Int struct { ... }
	intStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "val", types.Typ[types.Int64], false),
	}, nil)
	intType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Int", nil),
		intStruct, nil)
	scope.Insert(intType.Obj())
	intPtr := types.NewPointer(intType)

	// func NewInt(x int64) *Int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", intPtr)),
			false)))

	// type Float struct { ... }
	floatStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "val", types.Typ[types.Float64], false),
	}, nil)
	floatType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Float", nil),
		floatStruct, nil)
	scope.Insert(floatType.Obj())
	floatPtr := types.NewPointer(floatType)

	// func NewFloat(x float64) *Float
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFloat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", floatPtr)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// buildTextTemplatePackage creates the type-checked text/template package stub.
func buildTextTemplatePackage() *types.Package {
	pkg := types.NewPackage("text/template", "template")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Template struct { ... }
	tmplStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "name", types.Typ[types.String], false),
	}, nil)
	tmplType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Template", nil),
		tmplStruct, nil)
	scope.Insert(tmplType.Obj())
	tmplPtr := types.NewPointer(tmplType)

	// func New(name string) *Template
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", tmplPtr)),
			false)))
	_ = errType

	pkg.MarkComplete()
	return pkg
}

// buildEmbedPackage creates the type-checked embed package stub.
func buildEmbedPackage() *types.Package {
	pkg := types.NewPackage("embed", "embed")
	scope := pkg.Scope()

	// type FS struct { ... }
	fsStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "files", types.Typ[types.Int], false),
	}, nil)
	fsType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FS", nil),
		fsStruct, nil)
	scope.Insert(fsType.Obj())

	pkg.MarkComplete()
	return pkg
}

// buildHashPackage creates the type-checked hash package stub.
func buildHashPackage() *types.Package {
	pkg := types.NewPackage("hash", "hash")
	scope := pkg.Scope()

	// type Hash interface { ... }
	hashIface := types.NewInterfaceType(nil, nil)
	hashIface.Complete()
	hashType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Hash", nil),
		hashIface, nil)
	scope.Insert(hashType.Obj())

	// type Hash32 interface { ... }
	hash32Iface := types.NewInterfaceType(nil, nil)
	hash32Iface.Complete()
	hash32Type := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Hash32", nil),
		hash32Iface, nil)
	scope.Insert(hash32Type.Obj())

	pkg.MarkComplete()
	return pkg
}

// buildHashCRC32Package creates the type-checked hash/crc32 package stub.
func buildHashCRC32Package() *types.Package {
	pkg := types.NewPackage("hash/crc32", "crc32")
	scope := pkg.Scope()

	// const Size = 4
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int],
		constant.MakeInt64(4)))

	// func ChecksumIEEE(data []byte) uint32
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ChecksumIEEE",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint32])),
			false)))

	// func New(tab *Table) hash.Hash32 — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "tab", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// var IEEETable *Table — simplified as interface
	scope.Insert(types.NewVar(token.NoPos, pkg, "IEEETable", types.NewInterfaceType(nil, nil)))

	pkg.MarkComplete()
	return pkg
}

// buildNetPackage creates the type-checked net package stub.
func buildNetPackage() *types.Package {
	pkg := types.NewPackage("net", "net")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Conn interface { ... }
	connIface := types.NewInterfaceType(nil, nil)
	connIface.Complete()
	connType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Conn", nil),
		connIface, nil)
	scope.Insert(connType.Obj())

	// type Listener interface { ... }
	listenerIface := types.NewInterfaceType(nil, nil)
	listenerIface.Complete()
	listenerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Listener", nil),
		listenerIface, nil)
	scope.Insert(listenerType.Obj())

	// type Addr interface { ... }
	addrIface := types.NewInterfaceType(nil, nil)
	addrIface.Complete()
	addrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Addr", nil),
		addrIface, nil)
	scope.Insert(addrType.Obj())

	// func Dial(network, address string) (Conn, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dial",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", connType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Listen(network, address string) (Listener, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Listen",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", listenerType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func JoinHostPort(host, port string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "JoinHostPort",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "host", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "port", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func SplitHostPort(hostport string) (host, port string, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SplitHostPort",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "hostport", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "host", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "port", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}
