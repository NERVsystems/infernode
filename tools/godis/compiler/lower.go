package compiler

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/NERVsystems/infernode/tools/godis/dis"
	"golang.org/x/tools/go/ssa"
)

// funcLowerer lowers a single SSA function to Dis instructions.
type funcLowerer struct {
	fn              *ssa.Function
	frame           *Frame
	comp            *Compiler  // parent compiler (for string allocation, etc.)
	insts           []dis.Inst
	valueMap        map[ssa.Value]int32 // SSA value → frame offset
	allocBase       map[ssa.Value]int32 // *ssa.Alloc → base frame offset of data
	blockPC         map[*ssa.BasicBlock]int32
	patches         []branchPatch // deferred branch target patches
	sysMPOff        int32         // offset of Sys module ref in MP
	sysUsed         map[string]int // function name → LDT index
	callTypeDescs   []dis.TypeDesc // type descriptors for call-site frames
	funcCallPatches []funcCallPatch // deferred patches for local function calls
	closurePtrSlot  int32           // frame offset of hidden closure pointer (for inner functions)
	deferStack      []ssa.CallCommon // LIFO stack of deferred calls
	hasRecover      bool             // true if a deferred closure calls recover()
	excSlotFP       int32            // frame pointer slot for exception data (pointer, for VM storage)
	handlers        []handlerInfo    // exception handler table entries
}

// handlerInfo records an exception handler for the current function.
type handlerInfo struct {
	eoff   int32 // frame offset for exception data
	pc1    int32 // start PC of protected range (function-local)
	pc2    int32 // end PC of protected range (exclusive, function-local)
	wildPC int32 // wildcard handler PC (function-local)
}

type branchPatch struct {
	instIdx int
	target  *ssa.BasicBlock
}

// funcCallPatch records an instruction that needs patching for a local function call.
const (
	patchIFRAME = iota // IFRAME src = callee's type descriptor ID
	patchICALL         // ICALL dst = callee's start PC
)

type funcCallPatch struct {
	instIdx   int
	callee    *ssa.Function
	patchKind int
}

func newFuncLowerer(fn *ssa.Function, comp *Compiler, sysMPOff int32, sysUsed map[string]int) *funcLowerer {
	return &funcLowerer{
		fn:        fn,
		frame:     NewFrame(),
		comp:      comp,
		valueMap:  make(map[ssa.Value]int32),
		allocBase: make(map[ssa.Value]int32),
		blockPC:   make(map[*ssa.BasicBlock]int32),
		sysMPOff:  sysMPOff,
		sysUsed:   sysUsed,
	}
}

// lowerResult contains the compilation output of a function.
type lowerResult struct {
	insts           []dis.Inst
	frame           *Frame
	callTypeDescs   []dis.TypeDesc  // extra type descriptors for call-site frames
	funcCallPatches []funcCallPatch // patches for local function calls
	handlers        []handlerInfo   // exception handler table entries
}

// lower compiles the function to Dis instructions.
func (fl *funcLowerer) lower() (*lowerResult, error) {
	if len(fl.fn.Blocks) == 0 {
		return nil, fmt.Errorf("function %s has no blocks", fl.fn.Name())
	}

	// Scan for recover() in deferred closures
	fl.scanForRecover()

	// Pre-allocate frame slots for all SSA values that need them
	fl.allocateSlots()

	// If this function has recover, allocate the exception frame slot
	if fl.hasRecover {
		fl.excSlotFP = fl.frame.AllocPointer("excdata")
	}

	// Emit preamble to load free vars from closure struct
	if len(fl.fn.FreeVars) > 0 {
		fl.emitFreeVarLoads()
	}

	// Record body start PC (after preamble, before user code)
	bodyStartPC := int32(len(fl.insts))

	// First pass: emit instructions for each basic block
	for _, block := range fl.fn.Blocks {
		fl.blockPC[block] = int32(len(fl.insts))
		if err := fl.lowerBlock(block); err != nil {
			return nil, fmt.Errorf("block %s: %w", block.Comment, err)
		}
	}

	// If this function has recover, append the exception handler epilogue
	if fl.hasRecover {
		fl.emitExceptionHandler(bodyStartPC)
	}

	// Second pass: patch branch targets
	for _, p := range fl.patches {
		targetPC := fl.blockPC[p.target]
		inst := &fl.insts[p.instIdx]
		inst.Dst = dis.Imm(targetPC)
	}

	return &lowerResult{
		insts:           fl.insts,
		frame:           fl.frame,
		callTypeDescs:   fl.callTypeDescs,
		funcCallPatches: fl.funcCallPatches,
		handlers:        fl.handlers,
	}, nil
}

// scanForRecover checks if any deferred closure in this function calls recover().
func (fl *funcLowerer) scanForRecover() {
	for _, anon := range fl.fn.AnonFuncs {
		if anonHasRecover(anon) {
			fl.hasRecover = true
			return
		}
	}
}

// anonHasRecover checks if a function (closure) calls the recover() builtin.
func anonHasRecover(fn *ssa.Function) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			builtin, ok := call.Call.Value.(*ssa.Builtin)
			if ok && builtin.Name() == "recover" {
				return true
			}
		}
	}
	return false
}

// emitExceptionHandler appends exception handler code after the normal function body.
// Layout:
//
//	[handlerPC]   MOVW excSlotFP(fp) → excGlobal(mp)  // bridge exception
//	              ... deferred calls (LIFO) ...
//	              RET
//
// The handler table entry covers [bodyStartPC, handlerPC).
func (fl *funcLowerer) emitExceptionHandler(bodyStartPC int32) {
	handlerPC := int32(len(fl.insts))
	excGlobalMP := fl.comp.AllocExcGlobal()

	// Copy exception string from frame slot to module-data bridge
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(fl.excSlotFP), dis.MP(excGlobalMP)))

	// Emit deferred calls in LIFO order (same as normal RunDefers)
	for i := len(fl.deferStack) - 1; i >= 0; i-- {
		call := fl.deferStack[i]
		fl.emitDeferredCall(call) //nolint: ignore error for handler path
	}

	// Zero return values (Go returns zero values when recovering from panic)
	regretOff := int32(dis.REGRET * dis.IBY2WD)
	results := fl.fn.Signature.Results()
	retOff := int32(0)
	for i := 0; i < results.Len(); i++ {
		dt := GoTypeToDis(results.At(i).Type())
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FPInd(regretOff, retOff)))
		retOff += dt.Size
	}

	fl.emit(dis.Inst0(dis.IRET))

	// Record the handler table entry
	fl.handlers = append(fl.handlers, handlerInfo{
		eoff:   fl.excSlotFP,
		pc1:    bodyStartPC,
		pc2:    handlerPC, // exclusive: [pc1, pc2)
		wildPC: handlerPC,
	})
}

// allocateSlots pre-allocates frame slots for parameters and all SSA values.
func (fl *funcLowerer) allocateSlots() {
	// For the init function, reserve space for Inferno's command parameters:
	//   offset 64 (MaxTemp+0): ctxt (ref Draw->Context) - pointer
	//   offset 72 (MaxTemp+8): args (list of string) - pointer
	// These are set by the command launcher before calling init.
	if fl.fn.Name() == "main" {
		fl.frame.AllocPointer("ctxt") // offset 64
		fl.frame.AllocPointer("args") // offset 72
	}

	// Free variables (closures): allocate hidden closure pointer param BEFORE regular params
	// The caller stores the closure pointer at MaxTemp+0, then regular args follow.
	if len(fl.fn.FreeVars) > 0 {
		fl.closurePtrSlot = fl.frame.AllocPointer("$closure")
	}

	// Parameters (after closure pointer if present)
	for _, p := range fl.fn.Params {
		if _, ok := p.Type().Underlying().(*types.Interface); ok {
			// Interface parameter: 2 consecutive WORDs (tag + value)
			base := fl.frame.AllocWord(p.Name() + ".tag")
			fl.frame.AllocWord(p.Name() + ".val")
			fl.valueMap[p] = base
		} else if st, ok := p.Type().Underlying().(*types.Struct); ok {
			// Struct parameter: allocate consecutive slots for each field
			base := fl.allocStructFields(st, p.Name())
			fl.valueMap[p] = base
		} else {
			dt := GoTypeToDis(p.Type())
			if dt.IsPtr {
				fl.valueMap[p] = fl.frame.AllocPointer(p.Name())
			} else {
				fl.valueMap[p] = fl.frame.AllocWord(p.Name())
			}
		}
	}

	// Allocate slots for free variable values (loaded from closure struct at entry)
	if len(fl.fn.FreeVars) > 0 {
		for _, fv := range fl.fn.FreeVars {
			if _, ok := fv.Type().Underlying().(*types.Interface); ok {
				base := fl.frame.AllocWord(fv.Name() + ".tag")
				fl.frame.AllocWord(fv.Name() + ".val")
				fl.valueMap[fv] = base
			} else {
				dt := GoTypeToDis(fv.Type())
				if dt.IsPtr {
					fl.valueMap[fv] = fl.frame.AllocPointer(fv.Name())
				} else {
					fl.valueMap[fv] = fl.frame.AllocWord(fv.Name())
				}
			}
		}
	}

	// All instructions that produce values
	for _, block := range fl.fn.Blocks {
		for _, instr := range block.Instrs {
			if v, ok := instr.(ssa.Value); ok {
				if _, exists := fl.valueMap[v]; exists {
					continue
				}
				if v.Name() == "" {
					continue // instructions that don't produce named values
				}
				// Skip instructions that allocate their own slots:
				// - Alloc, FieldAddr: LEA produces stack/MP address, not heap pointer
				// - IndexAddr: interior pointer into array, not GC-traced
				switch instr.(type) {
				case *ssa.Alloc, *ssa.FieldAddr, *ssa.IndexAddr:
					continue
				}
				// Tuple values (multi-return) need consecutive slots per element
				if tup, ok := v.Type().(*types.Tuple); ok {
					fl.valueMap[v] = fl.allocTupleSlots(tup, v.Name())
				} else if _, ok := v.Type().Underlying().(*types.Interface); ok {
					// Interface values: 2 consecutive WORDs (tag + value)
					base := fl.frame.AllocWord(v.Name() + ".tag")
					fl.frame.AllocWord(v.Name() + ".val")
					fl.valueMap[v] = base
				} else if st, ok := v.Type().Underlying().(*types.Struct); ok {
					// Struct values need consecutive slots for each field
					fl.valueMap[v] = fl.allocStructFields(st, v.Name())
				} else {
					dt := GoTypeToDis(v.Type())
					if dt.IsPtr {
						fl.valueMap[v] = fl.frame.AllocPointer(v.Name())
					} else {
						fl.valueMap[v] = fl.frame.AllocWord(v.Name())
					}
				}
			}
		}
	}
}

func (fl *funcLowerer) lowerBlock(block *ssa.BasicBlock) error {
	for _, instr := range block.Instrs {
		if err := fl.lowerInstr(instr); err != nil {
			return fmt.Errorf("instruction %v: %w", instr, err)
		}
	}
	return nil
}

func (fl *funcLowerer) lowerInstr(instr ssa.Instruction) error {
	switch instr := instr.(type) {
	case *ssa.Alloc:
		return fl.lowerAlloc(instr)
	case *ssa.BinOp:
		return fl.lowerBinOp(instr)
	case *ssa.UnOp:
		return fl.lowerUnOp(instr)
	case *ssa.Call:
		return fl.lowerCall(instr)
	case *ssa.Return:
		return fl.lowerReturn(instr)
	case *ssa.If:
		return fl.lowerIf(instr)
	case *ssa.Jump:
		return fl.lowerJump(instr)
	case *ssa.Phi:
		return fl.lowerPhi(instr)
	case *ssa.Store:
		return fl.lowerStore(instr)
	case *ssa.FieldAddr:
		return fl.lowerFieldAddr(instr)
	case *ssa.IndexAddr:
		return fl.lowerIndexAddr(instr)
	case *ssa.Extract:
		return fl.lowerExtract(instr)
	case *ssa.Slice:
		return fl.lowerSlice(instr)
	case *ssa.Go:
		return fl.lowerGo(instr)
	case *ssa.MakeChan:
		return fl.lowerMakeChan(instr)
	case *ssa.Send:
		return fl.lowerSend(instr)
	case *ssa.Select:
		return fl.lowerSelect(instr)
	case *ssa.MakeClosure:
		return fl.lowerMakeClosure(instr)
	case *ssa.MakeMap:
		return fl.lowerMakeMap(instr)
	case *ssa.MapUpdate:
		return fl.lowerMapUpdate(instr)
	case *ssa.Lookup:
		return fl.lowerLookup(instr)
	case *ssa.Index:
		return fl.lowerIndex(instr)
	case *ssa.Range:
		return fl.lowerRange(instr)
	case *ssa.Next:
		return fl.lowerNext(instr)
	case *ssa.Convert:
		return fl.lowerConvert(instr)
	case *ssa.ChangeType:
		return fl.lowerChangeType(instr)
	case *ssa.Defer:
		return fl.lowerDefer(instr)
	case *ssa.RunDefers:
		return fl.lowerRunDefers(instr)
	case *ssa.Panic:
		return fl.lowerPanic(instr)
	case *ssa.MakeInterface:
		return fl.lowerMakeInterface(instr)
	case *ssa.TypeAssert:
		return fl.lowerTypeAssert(instr)
	case *ssa.ChangeInterface:
		return fl.lowerChangeInterface(instr)
	case *ssa.DebugRef:
		return nil // ignore debug info
	default:
		return fmt.Errorf("unsupported instruction: %T (%v)", instr, instr)
	}
}

func (fl *funcLowerer) lowerAlloc(instr *ssa.Alloc) error {
	if instr.Heap {
		elemType := instr.Type().(*types.Pointer).Elem()
		if _, ok := elemType.Underlying().(*types.Array); ok {
			return fl.lowerHeapArrayAlloc(instr)
		}
		return fl.lowerHeapAlloc(instr)
	}
	// Stack allocation: the SSA value is a pointer (*T).
	// We allocate frame slots for the pointed-to value(s) and use LEA
	// to make the pointer slot point to the base.
	// The pointer slot is NOT a GC pointer because it points to a stack frame,
	// not the heap. The GC manages stack frames separately.
	elemType := instr.Type().(*types.Pointer).Elem()

	var baseSlot int32
	if _, ok := elemType.Underlying().(*types.Interface); ok {
		// Interface: 2 consecutive WORDs (tag + value)
		baseSlot = fl.frame.AllocWord("alloc:" + instr.Name() + ".tag")
		fl.frame.AllocWord("alloc:" + instr.Name() + ".val")
	} else if st, ok := elemType.Underlying().(*types.Struct); ok {
		// Struct: allocate one slot per field
		baseSlot = fl.allocStructFields(st, instr.Name())
	} else if at, ok := elemType.Underlying().(*types.Array); ok {
		// Fixed-size array: allocate N consecutive element slots
		baseSlot = fl.allocArrayElements(at, instr.Name())
	} else {
		dt := GoTypeToDis(elemType)
		if dt.IsPtr {
			baseSlot = fl.frame.AllocPointer("alloc:" + instr.Name())
		} else {
			baseSlot = fl.frame.AllocWord("alloc:" + instr.Name())
		}
	}

	// Track the base offset for FieldAddr
	fl.allocBase[instr] = baseSlot

	// Allocate the pointer slot as non-pointer (stack address, not heap pointer)
	ptrSlot := fl.frame.AllocWord("ptr:" + instr.Name())
	fl.valueMap[instr] = ptrSlot
	fl.emit(dis.Inst2(dis.ILEA, dis.FP(baseSlot), dis.FP(ptrSlot)))
	return nil
}

// lowerHeapAlloc emits INEW to heap-allocate a value.
// The result is a GC-traced pointer slot (unlike stack alloc which uses AllocWord).
func (fl *funcLowerer) lowerHeapAlloc(instr *ssa.Alloc) error {
	elemType := instr.Type().(*types.Pointer).Elem()

	// Create a type descriptor for the heap object
	tdLocalIdx := fl.makeHeapTypeDesc(elemType)

	// Allocate a GC-traced pointer slot for the result (this IS a heap pointer)
	ptrSlot := fl.frame.AllocPointer("heap:" + instr.Name())
	fl.valueMap[instr] = ptrSlot

	// Emit INEW $tdLocalIdx, dst(fp)
	// The local index is patched by Phase 4 to a global TD ID
	fl.emit(dis.Inst2(dis.INEW, dis.Imm(int32(tdLocalIdx)), dis.FP(ptrSlot)))

	return nil
}

// lowerHeapArrayAlloc emits NEWA to create a Dis Array for a heap-allocated [N]T.
// This creates a proper Dis Array (with length header) instead of a raw heap object,
// so Slice can just copy the pointer and INDW works for indexing.
func (fl *funcLowerer) lowerHeapArrayAlloc(instr *ssa.Alloc) error {
	arrType := instr.Type().(*types.Pointer).Elem().Underlying().(*types.Array)
	elemType := arrType.Elem()
	n := int(arrType.Len())

	// Create element type descriptor for NEWA
	elemTDIdx := fl.makeHeapTypeDesc(elemType)

	// Result: GC-traced Dis Array pointer
	ptrSlot := fl.frame.AllocPointer("harr:" + instr.Name())
	fl.valueMap[instr] = ptrSlot

	// NEWA length, $elemTD, dst
	fl.emit(dis.NewInst(dis.INEWA, dis.Imm(int32(n)), dis.Imm(int32(elemTDIdx)), dis.FP(ptrSlot)))

	return nil
}

// makeHeapTypeDesc creates a type descriptor for a heap-allocated object.
// Unlike call-site TDs which include the MaxTemp frame header, heap TDs
// describe the raw object layout starting at offset 0.
func (fl *funcLowerer) makeHeapTypeDesc(elemType types.Type) int {
	var size int
	var ptrOffsets []int

	if st, ok := elemType.Underlying().(*types.Struct); ok {
		off := 0
		for i := 0; i < st.NumFields(); i++ {
			fdt := GoTypeToDis(st.Field(i).Type())
			if fdt.IsPtr {
				ptrOffsets = append(ptrOffsets, off)
			}
			off += int(fdt.Size)
		}
		size = off
	} else {
		// For byte/uint8, use 1-byte element size (not word-sized frame slot)
		size = DisElementSize(elemType)
		dt := GoTypeToDis(elemType)
		if dt.IsPtr {
			ptrOffsets = append(ptrOffsets, 0)
		}
	}

	// Only align to word boundary for word-sized or larger types
	if size > 1 && size%dis.IBY2WD != 0 {
		size = (size + dis.IBY2WD - 1) &^ (dis.IBY2WD - 1)
	}

	td := dis.NewTypeDesc(0, size) // ID assigned by Phase 4
	for _, off := range ptrOffsets {
		td.SetPointer(off)
	}

	fl.callTypeDescs = append(fl.callTypeDescs, td)
	return len(fl.callTypeDescs) - 1
}

// allocStructFields allocates consecutive frame slots for each struct field.
// Returns the base offset (first field's offset).
func (fl *funcLowerer) allocStructFields(st *types.Struct, baseName string) int32 {
	var baseSlot int32
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		dt := GoTypeToDis(field.Type())
		var slot int32
		if dt.IsPtr {
			slot = fl.frame.AllocPointer(baseName + "." + field.Name())
		} else {
			slot = fl.frame.AllocWord(baseName + "." + field.Name())
		}
		if i == 0 {
			baseSlot = slot
		}
	}
	return baseSlot
}

// allocArrayElements allocates N consecutive frame slots for a fixed-size array.
// Returns the base offset (first element's offset).
func (fl *funcLowerer) allocArrayElements(at *types.Array, baseName string) int32 {
	elemDT := GoTypeToDis(at.Elem())
	n := int(at.Len())
	var baseSlot int32
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s[%d]", baseName, i)
		var slot int32
		if elemDT.IsPtr {
			slot = fl.frame.AllocPointer(name)
		} else {
			slot = fl.frame.AllocWord(name)
		}
		if i == 0 {
			baseSlot = slot
		}
	}
	return baseSlot
}

// allocTupleSlots allocates consecutive frame slots for each element of a tuple
// (multi-return value). Returns the base offset (first element's offset).
func (fl *funcLowerer) allocTupleSlots(tup *types.Tuple, baseName string) int32 {
	var baseSlot int32
	for i := 0; i < tup.Len(); i++ {
		name := fmt.Sprintf("%s#%d", baseName, i)
		var slot int32
		if _, ok := tup.At(i).Type().Underlying().(*types.Interface); ok {
			// Interface element: 2 WORDs (tag + value)
			slot = fl.frame.AllocWord(name + ".tag")
			fl.frame.AllocWord(name + ".val")
		} else {
			dt := GoTypeToDis(tup.At(i).Type())
			if dt.IsPtr {
				slot = fl.frame.AllocPointer(name)
			} else {
				slot = fl.frame.AllocWord(name)
			}
		}
		if i == 0 {
			baseSlot = slot
		}
	}
	return baseSlot
}

func (fl *funcLowerer) lowerBinOp(instr *ssa.BinOp) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)
	mid := fl.operandOf(instr.Y)

	t := instr.X.Type().Underlying()
	basic, _ := t.(*types.Basic)

	// Dis three-operand arithmetic: dst = mid OP src
	// For Go's X OP Y:
	//   Commutative ops (ADD, MUL, AND, OR, XOR): order doesn't matter
	//   Non-commutative ops (SUB, DIV, MOD, SHL, SHR): need mid=X, src=Y
	// We have src=operandOf(X), mid=operandOf(Y), so swap for non-commutative ops.
	switch instr.Op {
	case token.ADD:
		op := fl.arithOp(dis.IADDW, dis.IADDF, dis.IADDC, basic)
		if op == dis.IADDC {
			// String concatenation is non-commutative: dst = mid + src
			// We have src=X, mid=Y, want X+Y, so swap: mid=X, src=Y
			fl.emit(dis.NewInst(op, mid, src, dis.FP(dst)))
		} else {
			fl.emit(dis.NewInst(op, src, mid, dis.FP(dst)))
		}
	case token.SUB:
		fl.emit(dis.NewInst(fl.arithOp(dis.ISUBW, dis.ISUBF, 0, basic), mid, src, dis.FP(dst)))
	case token.MUL:
		fl.emit(dis.NewInst(fl.arithOp(dis.IMULW, dis.IMULF, 0, basic), src, mid, dis.FP(dst)))
	case token.QUO:
		op := fl.arithOp(dis.IDIVW, dis.IDIVF, 0, basic)
		if op == dis.IDIVW {
			fl.emitZeroDivCheck(mid) // ARM64 sdiv returns 0 on div-by-zero instead of trapping
		}
		fl.emit(dis.NewInst(op, mid, src, dis.FP(dst)))
	case token.REM:
		fl.emitZeroDivCheck(mid)
		fl.emit(dis.NewInst(dis.IMODW, mid, src, dis.FP(dst)))
	case token.AND:
		fl.emit(dis.NewInst(dis.IANDW, src, mid, dis.FP(dst)))
	case token.OR:
		fl.emit(dis.NewInst(dis.IORW, src, mid, dis.FP(dst)))
	case token.XOR:
		fl.emit(dis.NewInst(dis.IXORW, src, mid, dis.FP(dst)))
	case token.SHL:
		fl.emit(dis.NewInst(dis.ISHLW, mid, src, dis.FP(dst)))
	case token.SHR:
		fl.emit(dis.NewInst(dis.ISHRW, mid, src, dis.FP(dst)))

	// Comparisons: produce a boolean (0 or 1) in the destination
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return fl.lowerComparison(instr, basic, src, mid, dst)

	default:
		return fmt.Errorf("unsupported binary op: %v", instr.Op)
	}
	return nil
}

func (fl *funcLowerer) lowerComparison(instr *ssa.BinOp, basic *types.Basic, src, mid dis.Operand, dst int32) error {
	// Comparison result is a boolean. We emit:
	//   movw $0, dst    (assume false)
	//   bXX src, mid, +2  (if condition true, skip next)
	//   jmp +1          (skip the movw $1)
	//   movw $1, dst    (set true)
	//
	// Actually simpler: set to 1, branch if true, set to 0.
	// But Dis branches jump to a PC, not skip N. So we need:
	//   movw $1, dst
	//   bXX src, mid, PC+3  (if true, skip the movw $0)
	//   movw $0, dst

	truePC := int32(len(fl.insts)) + 3 // after movw $1, branch, movw $0

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	branchOp := fl.compBranchOp(instr.Op, basic)
	fl.emit(dis.NewInst(branchOp, src, mid, dis.Imm(truePC)))

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	return nil
}

func (fl *funcLowerer) lowerUnOp(instr *ssa.UnOp) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)

	switch instr.Op {
	case token.SUB: // negation
		t := instr.X.Type().Underlying()
		if basic, ok := t.(*types.Basic); ok && isFloat(basic) {
			fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(dst)))
		} else {
			// Integer negation: 0 - x → subw x, $0, dst (Dis: dst = mid - src = 0 - x)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.ISUBW, src, dis.FP(dst), dis.FP(dst)))
		}
	case token.NOT: // logical not
		// XOR with 1
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IXORW, dis.Imm(1), dis.FP(dst), dis.FP(dst)))
	case token.XOR: // bitwise complement
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IXORW, src, dis.FP(dst), dis.FP(dst)))
	case token.MUL: // pointer dereference *ptr
		addrOff := fl.slotOf(instr.X)
		// Check for interface dereference (2-word copy)
		if _, ok := instr.Type().Underlying().(*types.Interface); ok {
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrOff, 0), dis.FP(dst)))          // tag
			fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrOff, iby2wd), dis.FP(dst+iby2wd))) // value
		} else if st, ok := instr.Type().Underlying().(*types.Struct); ok {
		// Check for struct dereference (multi-word copy)
			fieldOff := int32(0)
			for i := 0; i < st.NumFields(); i++ {
				fdt := GoTypeToDis(st.Field(i).Type())
				if fdt.IsPtr {
					fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(addrOff, fieldOff), dis.FP(dst+fieldOff)))
				} else {
					fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrOff, fieldOff), dis.FP(dst+fieldOff)))
				}
				fieldOff += fdt.Size
			}
		} else if IsByteType(instr.Type()) {
			// Byte dereference: zero-extend byte to word via CVTBW
			fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addrOff, 0), dis.FP(dst)))
		} else {
			dt := GoTypeToDis(instr.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(addrOff, 0), dis.FP(dst)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrOff, 0), dis.FP(dst)))
			}
		}
	case token.ARROW: // channel receive <-ch
		// RECV: src = channel address, dst = destination address
		fl.emit(dis.Inst2(dis.IRECV, src, dis.FP(dst)))
	default:
		return fmt.Errorf("unsupported unary op: %v", instr.Op)
	}
	return nil
}

func (fl *funcLowerer) lowerCall(instr *ssa.Call) error {
	call := instr.Call

	// Interface method invocation (s.Method())
	if call.IsInvoke() {
		return fl.lowerInvokeCall(instr)
	}

	// Check if this is a call to a built-in like println
	if builtin, ok := call.Value.(*ssa.Builtin); ok {
		return fl.lowerBuiltinCall(instr, builtin)
	}

	// Check if this is a call to a function
	if callee, ok := call.Value.(*ssa.Function); ok {
		// Check if it's from inferno/sys package → Sys module call
		if callee.Package() != nil && callee.Package().Pkg.Path() == "inferno/sys" {
			return fl.lowerSysModuleCall(instr, callee)
		}
		// Intercept stdlib calls that map to Dis instructions
		if callee.Package() != nil {
			pkgPath := callee.Package().Pkg.Path()
			if handled, err := fl.lowerStdlibCall(instr, callee, pkgPath); handled {
				return err
			}
		}
		return fl.lowerDirectCall(instr, callee)
	}

	// Indirect call (closure or function value)
	if _, ok := call.Value.Type().Underlying().(*types.Signature); ok {
		return fl.lowerClosureCall(instr)
	}

	return fmt.Errorf("unsupported call target: %T", call.Value)
}

func (fl *funcLowerer) lowerBuiltinCall(instr *ssa.Call, builtin *ssa.Builtin) error {
	switch builtin.Name() {
	case "println", "print":
		return fl.lowerPrintln(instr)
	case "len":
		return fl.lowerLen(instr)
	case "cap":
		return fl.lowerCap(instr)
	case "copy":
		return fl.lowerCopy(instr)
	case "close":
		// TODO: channel close
		return nil
	case "append":
		return fl.lowerAppend(instr)
	case "delete":
		return fl.lowerMapDelete(instr)
	case "recover":
		return fl.lowerRecover(instr)
	default:
		return fmt.Errorf("unsupported builtin: %s", builtin.Name())
	}
}

// lowerRecover reads the exception bridge from module data and clears it.
// Returns a tagged interface: tag=0/value=0 for nil, or tag="string" tag/value=String* if caught.
func (fl *funcLowerer) lowerRecover(instr *ssa.Call) error {
	excMP := fl.comp.AllocExcGlobal()
	dst := fl.slotOf(instr) // interface slot: tag at dst, value at dst+8
	iby2wd := int32(dis.IBY2WD)

	// Read the exception string pointer from bridge
	tmpSlot := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVW, dis.MP(excMP), dis.FP(tmpSlot)))
	// Clear the bridge
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.MP(excMP)))

	// If exception is non-zero, set tag = string type tag, value = string ptr
	// If zero, set tag = 0, value = 0 (nil interface)
	stringTag := fl.comp.AllocTypeTag("string")

	// BEQW $0, tmpSlot, $nilPC → if tmpSlot == 0, skip to nil
	beqwIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(tmpSlot), dis.Imm(0)))

	// Non-nil: set tag and value
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(stringTag), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(tmpSlot), dis.FP(dst+iby2wd)))
	jmpIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// Nil: tag=0, value=0
	nilPC := int32(len(fl.insts))
	fl.insts[beqwIdx].Dst = dis.Imm(nilPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpIdx].Dst = dis.Imm(donePC)

	return nil
}

// lowerStdlibCall intercepts calls to Go stdlib packages and lowers them
// to Dis instructions. Returns (true, err) if handled, (false, nil) if not.
func (fl *funcLowerer) lowerStdlibCall(instr *ssa.Call, callee *ssa.Function, pkgPath string) (bool, error) {
	switch pkgPath {
	case "strconv":
		return fl.lowerStrconvCall(instr, callee)
	case "fmt":
		return fl.lowerFmtCall(instr, callee)
	case "errors":
		return fl.lowerErrorsCall(instr, callee)
	}
	return false, nil
}

func (fl *funcLowerer) lowerFmtCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Sprintf":
		return fl.lowerFmtSprintf(instr)
	case "Println":
		// fmt.Println(args...) → emit println-style output for each arg + newline
		// The SSA packs args into a []any slice. We trace back to find the original values.
		return fl.lowerFmtPrintln(instr)
	case "Printf":
		// fmt.Printf(format, args...) → emit fprint-style output
		// For now, fall through to direct call (will error)
		return false, nil
	case "Errorf":
		return fl.lowerFmtErrorf(instr)
	}
	return false, nil
}

// lowerFmtSprintf handles fmt.Sprintf by analyzing the format string and arguments.
// Parses the format string into literal segments and %verbs, emits inline Dis
// instructions to convert each arg and concatenate all pieces.
func (fl *funcLowerer) lowerFmtSprintf(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintfInline(instr)
	if !ok {
		return false, nil
	}
	dstSlot := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(strSlot), dis.FP(dstSlot)))
	return true, nil
}

// lowerFmtErrorf handles fmt.Errorf(format, args...) by formatting the string
// with Sprintf-style logic, then wrapping it as a tagged error interface.
func (fl *funcLowerer) lowerFmtErrorf(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintfInline(instr)
	if !ok {
		return false, nil
	}
	dst := fl.slotOf(instr)
	tag := fl.comp.AllocTypeTag("errorString")
	iby2wd := int32(dis.IBY2WD)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(tag), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(strSlot), dis.FP(dst+iby2wd)))
	return true, nil
}

// emitSprintfInline emits the core Sprintf format-and-concatenate logic.
// Returns the frame slot containing the resulting string and true if successful,
// or (0, false) if the format string can't be handled.
func (fl *funcLowerer) emitSprintfInline(instr *ssa.Call) (int32, bool) {
	args := instr.Call.Args
	if len(args) < 1 {
		return 0, false
	}

	// Check if format string is a constant
	fmtConst, ok := args[0].(*ssa.Const)
	if !ok {
		return 0, false
	}
	fmtStr := constant.StringVal(fmtConst.Value)

	// Parse the format string into segments: literal strings and verb indices.
	// E.g., "hello %s, you are %d" → ["hello ", %s(0), ", you are ", %d(1)]
	type segment struct {
		literal string // non-empty for literal text
		verb    byte   // 's', 'd', 'v', 'c', 'x' for a verb segment
		argIdx  int    // vararg index for verb segments
	}
	var segments []segment
	argIdx := 0
	i := 0
	for i < len(fmtStr) {
		pct := strings.IndexByte(fmtStr[i:], '%')
		if pct < 0 {
			// Rest is literal
			segments = append(segments, segment{literal: fmtStr[i:]})
			break
		}
		if pct > 0 {
			segments = append(segments, segment{literal: fmtStr[i : i+pct]})
		}
		i += pct + 1
		if i >= len(fmtStr) {
			return 0, false // trailing %
		}
		verb := fmtStr[i]
		switch verb {
		case 's', 'd', 'v', 'c', 'x':
			segments = append(segments, segment{verb: verb, argIdx: argIdx})
			argIdx++
			i++
		case '%':
			// %% → literal %
			segments = append(segments, segment{literal: "%"})
			i++
		default:
			return 0, false // unsupported verb
		}
	}

	if len(segments) == 0 {
		return 0, false
	}

	// Resolve each segment to a string slot.
	// Then concatenate them all with ADDC.
	var slotParts []dis.Operand
	for _, seg := range segments {
		if seg.literal != "" {
			// Allocate a constant string
			mp := fl.comp.AllocString(seg.literal)
			tmp := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mp), dis.FP(tmp)))
			slotParts = append(slotParts, dis.FP(tmp))
		} else {
			// Verb: trace the vararg to get the original value
			var val ssa.Value
			if len(args) == 2 {
				val = fl.traceVarargElement(args[1], seg.argIdx)
			}
			if val == nil {
				return 0, false
			}
			switch seg.verb {
			case 'd', 'v':
				// int → string via CVTWC
				src := fl.operandOf(val)
				tmp := fl.frame.AllocTemp(true)
				fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(tmp)))
				slotParts = append(slotParts, dis.FP(tmp))
			case 's':
				src := fl.operandOf(val)
				slotParts = append(slotParts, src)
			case 'c':
				// rune → 1-char string via INSC
				// INSC src_rune, index, dst_string
				// With dst=nil (0), creates new 1-char string
				valOp := fl.operandOf(val)
				tmp := fl.frame.AllocTemp(true)
				fl.emit(dis.NewInst(dis.IINSC, valOp, dis.Imm(0), dis.FP(tmp)))
				slotParts = append(slotParts, dis.FP(tmp))
			case 'x':
				// int → hex string. Dis has no hex instruction;
				// emit decimal (CVTWC) as fallback for now.
				src := fl.operandOf(val)
				tmp := fl.frame.AllocTemp(true)
				fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(tmp)))
				slotParts = append(slotParts, dis.FP(tmp))
			}
		}
	}

	if len(slotParts) == 1 {
		// Single segment — return the slot directly
		if slotParts[0].Mode == dis.AFP {
			return slotParts[0].Val, true
		}
		// Need to move to a temp slot
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, slotParts[0], dis.FP(tmp)))
		return tmp, true
	}

	// Concatenate: fold left with ADDC
	// ADDC: dst = mid + src (Dis three-operand convention)
	acc := slotParts[0]
	for i := 1; i < len(slotParts); i++ {
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.NewInst(dis.IADDC, slotParts[i], acc, dis.FP(tmp)))
		acc = dis.FP(tmp)
	}
	return acc.Val, true
}

// lowerFmtPrintln handles fmt.Println by tracing varargs to print each value.
func (fl *funcLowerer) lowerFmtPrintln(instr *ssa.Call) (bool, error) {
	args := instr.Call.Args
	if len(args) == 0 {
		// fmt.Println() → just print newline
		fl.emitSysPrint("\n")
		return true, nil
	}

	// The single arg is the varargs []any slice.
	// Trace back through the SSA to find individual elements.
	sliceVal := args[0]
	elements := fl.traceAllVarargElements(sliceVal)
	if elements == nil {
		return false, nil
	}

	for i, elem := range elements {
		if i > 0 {
			fl.emitSysPrint(" ")
		}
		if err := fl.emitPrintArg(elem); err != nil {
			return false, nil
		}
	}
	fl.emitSysPrint("\n")

	// If the result is used (fmt.Println returns (int, error)), set it
	if instr.Name() != "" {
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))          // int = 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))   // error.tag = 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd))) // error.val = 0
	}
	return true, nil
}

// lowerErrorsCall handles calls to the errors package.
func (fl *funcLowerer) lowerErrorsCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	if callee.Name() != "New" {
		return false, nil
	}
	return true, fl.lowerErrorsNew(instr)
}

// lowerErrorsNew lowers errors.New("msg") to a tagged error interface:
// tag = errorString tag, value = string.
func (fl *funcLowerer) lowerErrorsNew(instr *ssa.Call) error {
	textArg := instr.Call.Args[0]
	textOff := fl.materialize(textArg)
	dst := fl.slotOf(instr)
	tag := fl.comp.AllocTypeTag("errorString")
	iby2wd := int32(dis.IBY2WD)

	// Store tag
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(tag), dis.FP(dst)))
	// Store value (string) — use MOVW (interface value slot is WORD, not GC-traced)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(textOff), dis.FP(dst+iby2wd)))
	return nil
}

// traceVarargElement traces back from a []any slice to find the original value
// of element at the given index. Returns nil if it can't be resolved.
func (fl *funcLowerer) traceVarargElement(sliceVal ssa.Value, idx int) ssa.Value {
	// The SSA sequence for varargs is:
	//   t0 = new [N]any (varargs)       ← Alloc
	//   t1 = &t0[idx]                   ← IndexAddr
	//   t2 = make any <- T (val)        ← MakeInterface
	//   *t1 = t2                        ← Store
	//   t3 = slice t0[:]                ← Slice
	//   call Sprintf(fmt, t3...)
	// We need to trace t3 → t0 → find the Store at index idx → find the MakeInterface value.

	// Step 1: trace Slice → Alloc
	slice, ok := sliceVal.(*ssa.Slice)
	if !ok {
		return nil
	}
	alloc, ok := slice.X.(*ssa.Alloc)
	if !ok {
		return nil
	}

	// Step 2: find Store instructions that write to indexed positions of alloc
	for _, ref := range *alloc.Referrers() {
		idxAddr, ok := ref.(*ssa.IndexAddr)
		if !ok {
			continue
		}
		// Check if this IndexAddr is for our target index
		idxConst, ok := idxAddr.Index.(*ssa.Const)
		if !ok {
			continue
		}
		if int(idxConst.Int64()) != idx {
			continue
		}
		// Find the Store to this IndexAddr
		for _, storeRef := range *idxAddr.Referrers() {
			store, ok := storeRef.(*ssa.Store)
			if !ok {
				continue
			}
			// The stored value should be a MakeInterface
			if mi, ok := store.Val.(*ssa.MakeInterface); ok {
				return mi.X // Return the original value before interface boxing
			}
			return store.Val
		}
	}
	return nil
}

// traceAllVarargElements traces all elements from a []any varargs slice.
func (fl *funcLowerer) traceAllVarargElements(sliceVal ssa.Value) []ssa.Value {
	slice, ok := sliceVal.(*ssa.Slice)
	if !ok {
		return nil
	}
	alloc, ok := slice.X.(*ssa.Alloc)
	if !ok {
		return nil
	}
	// Determine array size
	arrType, ok := alloc.Type().(*types.Pointer)
	if !ok {
		return nil
	}
	arr, ok := arrType.Elem().(*types.Array)
	if !ok {
		return nil
	}
	n := int(arr.Len())
	elements := make([]ssa.Value, n)
	for i := 0; i < n; i++ {
		elem := fl.traceVarargElement(sliceVal, i)
		if elem == nil {
			return nil
		}
		elements[i] = elem
	}
	return elements
}

func (fl *funcLowerer) lowerStrconvCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Itoa":
		// strconv.Itoa(x int) string → CVTWC src, dst
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(dst)))
		return true, nil
	case "Atoi":
		// strconv.Atoi(s string) (int, error) → CVTCW src, dst
		// We only support the value; error is always nil.
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		// Tuple result: (int, error). int at dst, error interface at dst+8..dst+16.
		fl.emit(dis.Inst2(dis.ICVTCW, src, dis.FP(dst)))
		// Set error to nil interface (tag=0, val=0)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "FormatInt":
		// strconv.FormatInt(i int64, base int) string
		// Only support base 10 — use CVTWC (int→decimal string)
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerPrintln(instr *ssa.Call) error {
	// println maps to sys->print with a format string
	// For each argument, emit a sys->print call with the appropriate format
	args := instr.Call.Args

	for i, arg := range args {
		if i > 0 {
			// Print space separator
			fl.emitSysPrint(" ")
		}
		if err := fl.emitPrintArg(arg); err != nil {
			return err
		}
	}

	// Print newline
	fl.emitSysPrint("\n")

	return nil
}

func (fl *funcLowerer) emitPrintArg(arg ssa.Value) error {
	t := arg.Type().Underlying()
	basic, isBasic := t.(*types.Basic)

	if isBasic {
		switch {
		case basic.Kind() == types.String:
			return fl.emitSysPrintFmt("%s", arg)
		case basic.Info()&types.IsInteger != 0:
			return fl.emitSysPrintFmt("%d", arg)
		case basic.Info()&types.IsFloat != 0:
			return fl.emitSysPrintFmt("%g", arg)
		case basic.Kind() == types.Bool:
			return fl.emitSysPrintFmt("%d", arg) // print 0/1 for now
		}
	}

	// Default: try %d
	return fl.emitSysPrintFmt("%d", arg)
}

// emitSysPrint emits a sys->print(literal_string) call.
func (fl *funcLowerer) emitSysPrint(s string) {
	// Allocate a temp for the format string
	fmtOff := fl.frame.AllocPointer("")

	// Store string constant into frame
	// We use MOVP to load a string. For literal strings, we need to
	// create them in the module data section. For now, use a temporary approach:
	// we'll build the string in the data section and reference it via MP.
	// Actually, for sys->print, we need to set up the frame for the print call.

	// For sys->print, the frame layout is:
	//   MaxTemp+0: format string (pointer)
	//   MaxTemp+8: return value (int)
	// But print is varargs so frame size = 0 in sysmod.h.
	// The actual frame gets sized by the compiler based on usage.

	// Simplified approach: emit the string as an immediate load.
	// Dis doesn't have a "load string literal" instruction per se.
	// String literals are loaded from the data section via MOVP from MP.
	// We need to allocate space in MP for the string and emit MOVP mp+off, fp+off.

	mpOff := fl.comp.AllocString(s)

	// Load string from MP to frame
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mpOff), dis.FP(fmtOff)))

	// Set up print frame and call
	fl.emitSysCall("print", []callSiteArg{{fmtOff, true}})
}

// emitSysPrintFmt emits sys->print(fmt, arg).
func (fl *funcLowerer) emitSysPrintFmt(format string, arg ssa.Value) error {
	fmtOff := fl.frame.AllocPointer("")
	mpOff := fl.comp.AllocString(format)

	// Load format string
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mpOff), dis.FP(fmtOff)))

	// Materialize the argument into a frame slot
	argOff := fl.materialize(arg)

	// Set up print frame and call with format + arg
	fl.emitSysCall("print", []callSiteArg{
		{fmtOff, true},                          // format string (always pointer)
		{argOff, GoTypeToDis(arg.Type()).IsPtr},  // argument
	})

	return nil
}

// callSiteArg describes one argument being passed to a call.
type callSiteArg struct {
	srcOff int32 // frame offset of the source value
	isPtr  bool  // whether the argument is a pointer (for GC type map)
}

// emitSysCall emits a call to a Sys module function.
// For variadic functions (like print), it uses IFRAME with a local type
// descriptor. For fixed-frame functions, it uses IMFRAME.
func (fl *funcLowerer) emitSysCall(funcName string, args []callSiteArg) {
	ldtIdx, ok := fl.sysUsed[funcName]
	if !ok {
		ldtIdx = len(fl.sysUsed)
		fl.sysUsed[funcName] = ldtIdx
	}

	sf := LookupSysFunc(funcName)

	// Allocate a temp for the callee frame pointer.
	// NOT marked as a pointer: callee frames are on the stack, not heap.
	// After MCALL returns, this slot holds a stale pointer that GC must NOT trace.
	callFrame := fl.frame.AllocWord("")

	if sf != nil && sf.FrameSize == 0 {
		// Variadic function: use IFRAME with a local type descriptor
		tdID := fl.makeCallTypeDesc(args)
		fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(int32(tdID)), dis.FP(callFrame)))
	} else {
		// Fixed-frame function: use IMFRAME
		fl.emit(dis.NewInst(dis.IMFRAME, dis.MP(fl.sysMPOff), dis.Imm(int32(ldtIdx)), dis.FP(callFrame)))
	}

	// Set arguments in callee frame
	for i, arg := range args {
		calleeOff := int32(dis.MaxTemp) + int32(i)*int32(dis.IBY2WD)
		if arg.isPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.srcOff), dis.FPInd(callFrame, calleeOff)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.srcOff), dis.FPInd(callFrame, calleeOff)))
		}
	}

	// Set REGRET: point to a temp word where the return value goes.
	// REGRET is at offset REGRET*IBY2WD = 32 in the callee frame.
	retOff := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ILEA, dis.FP(retOff), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))

	// IMCALL: call the function
	fl.emit(dis.NewInst(dis.IMCALL, dis.FP(callFrame), dis.Imm(int32(ldtIdx)), dis.MP(fl.sysMPOff)))
}

// makeCallTypeDesc creates a type descriptor for a call-site frame.
// Returns the type descriptor ID.
func (fl *funcLowerer) makeCallTypeDesc(args []callSiteArg) int {
	// Frame layout: MaxTemp (64) + args
	frameSize := dis.MaxTemp + len(args)*dis.IBY2WD
	// Align to IBY2WD
	if frameSize%dis.IBY2WD != 0 {
		frameSize = (frameSize + dis.IBY2WD - 1) &^ (dis.IBY2WD - 1)
	}

	// ID will be assigned later by the compiler (2 + index)
	// Use a placeholder; the compiler will fix it
	td := dis.NewTypeDesc(0, frameSize)

	// Mark pointer arguments in the type map
	for i, arg := range args {
		if arg.isPtr {
			td.SetPointer(dis.MaxTemp + i*dis.IBY2WD)
		}
	}

	fl.callTypeDescs = append(fl.callTypeDescs, td)
	// Return the index; the actual ID will be computed by the compiler
	return len(fl.callTypeDescs) - 1
}

// sysGoToDisName maps Go function names in inferno/sys to Sys module function names.
var sysGoToDisName = map[string]string{
	"Fildes":   "fildes",
	"Open":     "open",
	"Write":    "write",
	"Read":     "read",
	"Fprint":   "fprint",
	"Sleep":    "sleep",
	"Millisec": "millisec",
}

// lowerSysModuleCall emits an IMFRAME/IMCALL sequence for a call to an
// inferno/sys package function, targeting the Dis Sys module.
func (fl *funcLowerer) lowerSysModuleCall(instr *ssa.Call, callee *ssa.Function) error {
	goName := callee.Name()
	disName, ok := sysGoToDisName[goName]
	if !ok {
		return fmt.Errorf("unsupported sys function: %s", goName)
	}

	sf := LookupSysFunc(disName)
	if sf == nil {
		return fmt.Errorf("unknown Sys function: %s", disName)
	}

	// Register this Sys function in the LDT
	ldtIdx, ok := fl.sysUsed[disName]
	if !ok {
		ldtIdx = len(fl.sysUsed)
		fl.sysUsed[disName] = ldtIdx
	}

	// Materialize all arguments
	type argSlot struct {
		off   int32
		isPtr bool
	}
	var args []argSlot
	for _, arg := range instr.Call.Args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		args = append(args, argSlot{off, dt.IsPtr})
	}

	// Allocate callee frame slot (not GC-traced — stack frame, stale after return)
	callFrame := fl.frame.AllocWord("")

	if sf.FrameSize == 0 {
		// Variadic function (fprint, print): use IFRAME with custom TD
		var callArgs []callSiteArg
		for _, a := range args {
			callArgs = append(callArgs, callSiteArg{a.off, a.isPtr})
		}
		tdID := fl.makeCallTypeDesc(callArgs)
		fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(int32(tdID)), dis.FP(callFrame)))
	} else {
		// Fixed-frame function: use IMFRAME
		fl.emit(dis.NewInst(dis.IMFRAME, dis.MP(fl.sysMPOff), dis.Imm(int32(ldtIdx)), dis.FP(callFrame)))
	}

	// Set arguments in callee frame (args start at MaxTemp = 64)
	for i, arg := range args {
		calleeOff := int32(dis.MaxTemp) + int32(i)*int32(dis.IBY2WD)
		if arg.isPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
		}
	}

	// Set up REGRET if function returns a value
	sig := callee.Signature
	if sig.Results().Len() > 0 {
		retSlot := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(retSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
	}

	// IMCALL: call the Sys module function
	fl.emit(dis.NewInst(dis.IMCALL, dis.FP(callFrame), dis.Imm(int32(ldtIdx)), dis.MP(fl.sysMPOff)))

	return nil
}

func (fl *funcLowerer) lowerGo(instr *ssa.Go) error {
	call := instr.Call

	callee, ok := call.Value.(*ssa.Function)
	if !ok {
		return fmt.Errorf("go statement with non-function target: %T", call.Value)
	}

	// Materialize all arguments
	type goArgInfo struct {
		off     int32
		isPtr   bool
		isIface bool
		st      *types.Struct
	}
	var args []goArgInfo
	for _, arg := range call.Args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		var st *types.Struct
		if s, ok := arg.Type().Underlying().(*types.Struct); ok {
			st = s
		}
		_, isIface := arg.Type().Underlying().(*types.Interface)
		args = append(args, goArgInfo{off, dt.IsPtr, isIface, st})
	}

	// Allocate callee frame slot
	callFrame := fl.frame.AllocWord("")

	// IFRAME $tdID, callFrame(fp)
	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	// Set arguments in callee frame
	iby2wd := int32(dis.IBY2WD)
	calleeOff := int32(dis.MaxTemp)
	for _, arg := range args {
		if arg.isIface {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off+iby2wd), dis.FPInd(callFrame, calleeOff+iby2wd)))
			calleeOff += 2 * iby2wd
		} else if arg.st != nil {
			fieldOff := int32(0)
			for i := 0; i < arg.st.NumFields(); i++ {
				fdt := GoTypeToDis(arg.st.Field(i).Type())
				if fdt.IsPtr {
					fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				} else {
					fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				}
				fieldOff += fdt.Size
			}
			calleeOff += GoTypeToDis(arg.st).Size
		} else if arg.isPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += iby2wd
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += iby2wd
		}
	}

	// SPAWN callFrame(fp), $targetPC (instead of CALL)
	ispawnIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.ISPAWN, dis.FP(callFrame), dis.Imm(0)))

	// Record patches
	fl.funcCallPatches = append(fl.funcCallPatches,
		funcCallPatch{instIdx: iframeIdx, callee: callee, patchKind: patchIFRAME},
		funcCallPatch{instIdx: ispawnIdx, callee: callee, patchKind: patchICALL}, // same patch kind — dst = PC
	)

	return nil
}

func (fl *funcLowerer) lowerMakeChan(instr *ssa.MakeChan) error {
	chanType := instr.Type().(*types.Chan)
	elemType := chanType.Elem()

	// Select NEWC variant based on element type
	newcOp := channelNewcOp(elemType)

	// Destination: the channel pointer slot (already allocated as pointer)
	dst := fl.slotOf(instr)

	// Unbuffered: just dst operand, no middle → R.m == R.d → buffer size 0
	fl.emit(dis.Inst1(newcOp, dis.FP(dst)))

	return nil
}

// channelNewcOp selects the appropriate NEWC variant for a channel element type.
func channelNewcOp(elemType types.Type) dis.Op {
	dt := GoTypeToDis(elemType)
	if dt.IsPtr {
		return dis.INEWCP
	}
	if IsByteType(elemType) {
		return dis.INEWCB
	}
	if basic, ok := elemType.Underlying().(*types.Basic); ok {
		if basic.Info()&types.IsFloat != 0 {
			return dis.INEWCF
		}
	}
	return dis.INEWCW
}

func (fl *funcLowerer) lowerSend(instr *ssa.Send) error {
	// Materialize the value to send into a frame slot
	valOff := fl.materialize(instr.X)

	// Get the channel slot
	chanOff := fl.slotOf(instr.Chan)

	// SEND src=valueAddr, dst=channelAddr
	fl.emit(dis.Inst2(dis.ISEND, dis.FP(valOff), dis.FP(chanOff)))

	return nil
}

func (fl *funcLowerer) lowerSelect(instr *ssa.Select) error {
	states := instr.States

	// Count sends and recvs
	var nsend, nrecv int
	for _, s := range states {
		if s.Dir == types.SendOnly {
			nsend++
		} else {
			nrecv++
		}
	}

	if nsend > 0 && nrecv > 0 {
		return fmt.Errorf("mixed send/recv select not yet supported")
	}

	nTotal := nsend + nrecv

	// Tuple base: [index (0), recvOk (8), v0 (16), v1 (24), ...]
	tupleBase := fl.slotOf(instr)

	// Allocate contiguous Alt structure in frame:
	//   nsend (WORD), nrecv (WORD), then N × {Channel* (ptr), void* (word)}
	altBase := fl.frame.AllocWord("alt.nsend")
	fl.frame.AllocWord("alt.nrecv")
	for i := 0; i < nTotal; i++ {
		fl.frame.AllocPointer(fmt.Sprintf("alt.ac%d.c", i))
		fl.frame.AllocWord(fmt.Sprintf("alt.ac%d.ptr", i))
	}

	// Fill in header
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(int32(nsend)), dis.FP(altBase)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(int32(nrecv)), dis.FP(altBase+int32(dis.IBY2WD))))

	// Fill in entries (all same direction, so Go order = Dis order)
	acOff := altBase + 2*int32(dis.IBY2WD)
	for i, s := range states {
		chanOff := fl.slotOf(s.Chan)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(chanOff), dis.FP(acOff)))

		if s.Dir == types.SendOnly {
			// Send: ptr = address of value to send
			valOff := fl.materialize(s.Send)
			fl.emit(dis.Inst2(dis.ILEA, dis.FP(valOff), dis.FP(acOff+int32(dis.IBY2WD))))
		} else {
			// Recv: ptr = address in tuple where received value goes
			recvOff := tupleBase + int32((2+i))*int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.ILEA, dis.FP(recvOff), dis.FP(acOff+int32(dis.IBY2WD))))
		}

		acOff += 2 * int32(dis.IBY2WD)
	}

	// Emit ALT (blocking) or NBALT (non-blocking / has default)
	if instr.Blocking {
		fl.emit(dis.Inst2(dis.IALT, dis.FP(altBase), dis.FP(tupleBase)))
	} else {
		fl.emit(dis.Inst2(dis.INBALT, dis.FP(altBase), dis.FP(tupleBase)))
	}

	// Set recvOk = 1 (Dis channels can't be closed)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(tupleBase+int32(dis.IBY2WD))))

	return nil
}

func (fl *funcLowerer) lowerDirectCall(instr *ssa.Call, callee *ssa.Function) error {
	call := instr.Call

	// Materialize all arguments first (may emit instructions for constants)
	type argInfo struct {
		off     int32
		isPtr   bool
		isIface bool         // true if interface type (2 words)
		st      *types.Struct // non-nil if this is a struct value argument
	}
	var args []argInfo
	for _, arg := range call.Args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		var st *types.Struct
		if s, ok := arg.Type().Underlying().(*types.Struct); ok {
			st = s
		}
		_, isIface := arg.Type().Underlying().(*types.Interface)
		args = append(args, argInfo{off, dt.IsPtr, isIface, st})
	}

	// Allocate callee frame slot (NOT a GC pointer - stack allocated, stale after return)
	callFrame := fl.frame.AllocWord("")

	// IFRAME $0, callFrame(fp) — TD ID is placeholder, patched by compiler
	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	// Set arguments in callee frame (args start at MaxTemp = 64)
	iby2wd := int32(dis.IBY2WD)
	calleeOff := int32(dis.MaxTemp)
	for _, arg := range args {
		if arg.isIface {
			// Interface argument: copy 2 words (tag + value)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off+iby2wd), dis.FPInd(callFrame, calleeOff+iby2wd)))
			calleeOff += 2 * iby2wd
		} else if arg.st != nil {
			// Struct argument: multi-word copy
			fieldOff := int32(0)
			for i := 0; i < arg.st.NumFields(); i++ {
				fdt := GoTypeToDis(arg.st.Field(i).Type())
				if fdt.IsPtr {
					fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				} else {
					fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				}
				fieldOff += fdt.Size
			}
			calleeOff += GoTypeToDis(arg.st).Size
		} else if arg.isPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += iby2wd
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += iby2wd
		}
	}

	// Set up REGRET if function returns a value
	sig := callee.Signature
	if sig.Results().Len() > 0 {
		retSlot := fl.slotOf(instr) // caller's slot where result lands
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(retSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
	}

	// CALL callFrame(fp), $0 — target PC is placeholder, patched by compiler
	icallIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.ICALL, dis.FP(callFrame), dis.Imm(0)))

	// Record patches for the compiler to resolve
	fl.funcCallPatches = append(fl.funcCallPatches,
		funcCallPatch{instIdx: iframeIdx, callee: callee, patchKind: patchIFRAME},
		funcCallPatch{instIdx: icallIdx, callee: callee, patchKind: patchICALL},
	)

	return nil
}

// lowerInvokeCall handles interface method calls (s.Method()).
// For single-implementation interfaces: direct call (fast path).
// For multi-implementation: BEQW dispatch chain on type tag.
func (fl *funcLowerer) lowerInvokeCall(instr *ssa.Call) error {
	call := instr.Call
	methodName := call.Method.Name()

	// Resolve all concrete implementations of this method
	impls := fl.comp.ResolveInterfaceMethods(methodName)
	if len(impls) == 0 {
		return fmt.Errorf("cannot resolve interface method %s (no implementation found)", methodName)
	}

	// The receiver is call.Value (tagged interface: tag at +0, value at +8).
	ifaceSlot := fl.materialize(call.Value) // interface base slot
	iby2wd := int32(dis.IBY2WD)

	// Materialize additional arguments
	type argInfo struct {
		off   int32
		isPtr bool
	}
	var extraArgs []argInfo
	for _, arg := range call.Args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		extraArgs = append(extraArgs, argInfo{off, dt.IsPtr})
	}

	// emitCallForImpl emits IFRAME + arg copy + REGRET + ICALL for one callee,
	// using ifaceSlot+8 as the receiver value.
	emitCallForImpl := func(callee *ssa.Function) {
		callFrame := fl.frame.AllocWord("")

		iframeIdx := len(fl.insts)
		fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

		// Set receiver (first param at MaxTemp)
		calleeOff := int32(dis.MaxTemp)
		recvValueSlot := ifaceSlot + iby2wd // value part of interface
		if len(callee.Params) > 0 {
			recvType := callee.Params[0].Type()
			paramDT := GoTypeToDis(recvType)
			if st, ok := recvType.Underlying().(*types.Struct); ok && paramDT.Size > iby2wd {
				// Struct receiver: interface value holds a pointer to struct data.
				// Copy each field through the pointer.
				fieldOff := int32(0)
				for i := 0; i < st.NumFields(); i++ {
					fdt := GoTypeToDis(st.Field(i).Type())
					if fdt.IsPtr {
						fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(recvValueSlot, fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
					} else {
						fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(recvValueSlot, fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
					}
					fieldOff += fdt.Size
				}
				calleeOff += paramDT.Size
			} else if paramDT.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FP(recvValueSlot), dis.FPInd(callFrame, calleeOff)))
				calleeOff += iby2wd
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(recvValueSlot), dis.FPInd(callFrame, calleeOff)))
				calleeOff += iby2wd
			}
		}

		// Set additional arguments
		for _, arg := range extraArgs {
			if arg.isPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			}
			calleeOff += iby2wd
		}

		// Set up REGRET if function returns a value
		sig := callee.Signature
		if sig.Results().Len() > 0 && instr.Name() != "" {
			retSlot := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.ILEA, dis.FP(retSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
		}

		// CALL
		icallIdx := len(fl.insts)
		fl.emit(dis.Inst2(dis.ICALL, dis.FP(callFrame), dis.Imm(0)))

		fl.funcCallPatches = append(fl.funcCallPatches,
			funcCallPatch{instIdx: iframeIdx, callee: callee, patchKind: patchIFRAME},
			funcCallPatch{instIdx: icallIdx, callee: callee, patchKind: patchICALL},
		)
	}

	// emitSyntheticInline emits inline code for a synthetic method (fn==nil).
	// Currently handles errorString.Error() — the value IS the string.
	emitSyntheticInline := func() {
		if instr.Name() != "" {
			resultDst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(ifaceSlot+iby2wd), dis.FP(resultDst)))
		}
	}

	if len(impls) == 1 {
		// Single-implementation fast path: direct call or inline synthetic
		if impls[0].fn == nil {
			emitSyntheticInline()
		} else {
			emitCallForImpl(impls[0].fn)
		}
		return nil
	}

	// Multi-implementation dispatch: BEQW chain on type tag
	// Layout:
	//   BEQW $tag1, FP(ifaceSlot), $call1_pc
	//   BEQW $tag2, FP(ifaceSlot), $call2_pc
	//   RAISE "unknown type"
	//   call1: IFRAME/args/ICALL; JMP exit
	//   call2: IFRAME/args/ICALL; JMP exit
	//   exit:

	var beqwIdxs []int
	for _, impl := range impls {
		idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(impl.tag), dis.FP(ifaceSlot), dis.Imm(0))) // dst patched below
		beqwIdxs = append(beqwIdxs, idx)
	}

	// Default: panic with unknown type
	panicStr := fl.comp.AllocString("unknown type in interface dispatch")
	panicSlot := fl.frame.AllocPointer("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(panicStr), dis.FP(panicSlot)))
	fl.emit(dis.Inst1(dis.IRAISE, dis.FP(panicSlot)))

	// Emit call sequence for each impl, patch BEQW targets
	var exitJmps []int
	for i, impl := range impls {
		// Patch BEQW to jump here
		fl.insts[beqwIdxs[i]].Dst = dis.Imm(int32(len(fl.insts)))

		if impl.fn == nil {
			emitSyntheticInline()
		} else {
			emitCallForImpl(impl.fn)
		}

		// JMP to exit (placeholder)
		exitIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		exitJmps = append(exitJmps, exitIdx)
	}

	// Patch exit JMPs
	exitPC := int32(len(fl.insts))
	for _, idx := range exitJmps {
		fl.insts[idx].Dst = dis.Imm(exitPC)
	}

	return nil
}

func (fl *funcLowerer) lowerReturn(instr *ssa.Return) error {
	if len(instr.Results) > 0 {
		// Store return values through REGRET pointer.
		// REGRET is at offset REGRET*IBY2WD = 32 in the frame header.
		// Multiple results go at successive offsets from REGRET.
		regretOff := int32(dis.REGRET * dis.IBY2WD)
		iby2wd := int32(dis.IBY2WD)
		retOff := int32(0)
		for _, result := range instr.Results {
			if _, ok := result.Type().Underlying().(*types.Interface); ok {
				// Interface return: copy 2 words (tag + value)
				off := fl.materialize(result)
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(off), dis.FPInd(regretOff, retOff)))
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(off+iby2wd), dis.FPInd(regretOff, retOff+iby2wd)))
				retOff += 2 * iby2wd
			} else {
				off := fl.materialize(result)
				dt := GoTypeToDis(result.Type())
				if dt.IsPtr {
					fl.emit(dis.Inst2(dis.IMOVP, dis.FP(off), dis.FPInd(regretOff, retOff)))
				} else {
					fl.emit(dis.Inst2(dis.IMOVW, dis.FP(off), dis.FPInd(regretOff, retOff)))
				}
				retOff += dt.Size
			}
		}
	}
	fl.emit(dis.Inst0(dis.IRET))
	return nil
}

func (fl *funcLowerer) lowerIf(instr *ssa.If) error {
	// The condition is already a boolean in a frame slot
	condOff := fl.slotOf(instr.Cond)

	// If condition != 0, jump to the true block (Succs[0])
	// Otherwise fall through to false block (Succs[1])
	trueBlock := instr.Block().Succs[0]
	falseBlock := instr.Block().Succs[1]
	thisBlock := instr.Block()

	trueHasPhi := blockHasPhis(trueBlock)
	falseHasPhi := blockHasPhis(falseBlock)

	if !trueHasPhi && !falseHasPhi {
		// Simple case: no phis in either successor
		patchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(condOff), dis.Imm(0), dis.Imm(0)))
		fl.patches = append(fl.patches, branchPatch{instIdx: patchIdx, target: trueBlock})

		patchIdx = len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.patches = append(fl.patches, branchPatch{instIdx: patchIdx, target: falseBlock})
	} else {
		// Phis present: emit separate phi-move blocks for each path.
		// Layout:
		//   BNEW cond, $0, truePhiPC    (if true, skip false path)
		//   [false phi moves]
		//   JMP falseBlock
		//   [true phi moves]            (truePhiPC lands here)
		//   JMP trueBlock

		// BNEW with placeholder — patched below once we know truePhiPC
		bnewIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(condOff), dis.Imm(0), dis.Imm(0)))

		// False path: emit phi moves then jump
		fl.emitPhiMoves(thisBlock, falseBlock)
		falseJmpIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.patches = append(fl.patches, branchPatch{instIdx: falseJmpIdx, target: falseBlock})

		// True path starts here — patch the BNEW target
		truePhiPC := int32(len(fl.insts))
		fl.insts[bnewIdx].Dst = dis.Imm(truePhiPC)

		fl.emitPhiMoves(thisBlock, trueBlock)
		trueJmpIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.patches = append(fl.patches, branchPatch{instIdx: trueJmpIdx, target: trueBlock})
	}

	return nil
}

func (fl *funcLowerer) lowerJump(instr *ssa.Jump) error {
	target := instr.Block().Succs[0]

	// Emit phi moves for the target block before jumping
	fl.emitPhiMoves(instr.Block(), target)

	patchIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.patches = append(fl.patches, branchPatch{instIdx: patchIdx, target: target})
	return nil
}

func (fl *funcLowerer) lowerPhi(instr *ssa.Phi) error {
	// Phi nodes are handled by emitPhiMoves in lowerIf/lowerJump.
	// The moves are inserted at the end of each predecessor block,
	// before the terminating branch/jump instruction.
	return nil
}

// emitPhiMoves emits MOV instructions for phi nodes when transitioning
// from 'from' to 'to'. For each phi in 'to', this emits a move from the
// value corresponding to the 'from' edge.
func (fl *funcLowerer) emitPhiMoves(from, to *ssa.BasicBlock) {
	// Find which edge index 'from' is in 'to's predecessor list
	edgeIdx := -1
	for i, pred := range to.Preds {
		if pred == from {
			edgeIdx = i
			break
		}
	}
	if edgeIdx < 0 {
		return
	}

	for _, instr := range to.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break // phis are always at the start of a block
		}
		dst := fl.slotOf(phi)
		if _, ok := phi.Type().Underlying().(*types.Interface); ok {
			// Interface phi: copy 2 words (tag + value)
			edge := phi.Edges[edgeIdx]
			if c, ok := edge.(*ssa.Const); ok && c.Value == nil {
				// nil interface: tag=0, value=0
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+int32(dis.IBY2WD))))
			} else {
				srcSlot := fl.slotOf(edge)
				fl.copyIface(srcSlot, dst)
			}
		} else {
			src := fl.operandOf(phi.Edges[edgeIdx])
			dt := GoTypeToDis(phi.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(dst)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
			}
		}
	}
}

// blockHasPhis returns true if the block starts with any Phi instructions.
func blockHasPhis(b *ssa.BasicBlock) bool {
	if len(b.Instrs) == 0 {
		return false
	}
	_, ok := b.Instrs[0].(*ssa.Phi)
	return ok
}

func (fl *funcLowerer) lowerStore(instr *ssa.Store) error {
	addrOff := fl.slotOf(instr.Addr)

	// Check if storing an interface value (2-word: tag + value)
	if _, ok := instr.Val.Type().Underlying().(*types.Interface); ok {
		valBase := fl.slotOf(instr.Val)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(valBase), dis.FPInd(addrOff, 0)))          // tag
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(valBase+iby2wd), dis.FPInd(addrOff, iby2wd))) // value
		return nil
	}

	// Check if storing a struct value (multi-word)
	if st, ok := instr.Val.Type().Underlying().(*types.Struct); ok {
		valBase := fl.slotOf(instr.Val)
		fieldOff := int32(0)
		for i := 0; i < st.NumFields(); i++ {
			fdt := GoTypeToDis(st.Field(i).Type())
			if fdt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FP(valBase+fieldOff), dis.FPInd(addrOff, fieldOff)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(valBase+fieldOff), dis.FPInd(addrOff, fieldOff)))
			}
			fieldOff += fdt.Size
		}
		return nil
	}

	valOff := fl.materialize(instr.Val)
	dt := GoTypeToDis(instr.Val.Type())
	if dt.IsPtr {
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(valOff), dis.FPInd(addrOff, 0)))
	} else if IsByteType(instr.Val.Type()) {
		// Byte store: truncate word to byte via CVTWB
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(valOff), dis.FPInd(addrOff, 0)))
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(valOff), dis.FPInd(addrOff, 0)))
	}
	return nil
}

func (fl *funcLowerer) lowerFieldAddr(instr *ssa.FieldAddr) error {
	// FieldAddr produces a pointer to a field within a struct.
	// instr.X is the struct pointer, instr.Field is the field index.
	structType := instr.X.Type().(*types.Pointer).Elem().Underlying().(*types.Struct)
	fieldOff := int32(0)
	for i := 0; i < instr.Field; i++ {
		dt := GoTypeToDis(structType.Field(i).Type())
		fieldOff += dt.Size
	}

	// Interior pointer slots are AllocWord (not GC-traced).
	// For stack allocs: points into the frame, not the heap.
	// For heap allocs: interior pointer; the base pointer in its
	// GC-traced slot keeps the object alive.
	ptrSlot := fl.frame.AllocWord("faddr:" + instr.Name())
	fl.valueMap[instr] = ptrSlot

	base, ok := fl.allocBase[instr.X]
	if ok {
		// Stack-allocated struct: field is at base + fieldOff in the frame
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(base+fieldOff), dis.FP(ptrSlot)))
	} else {
		// Heap pointer or call result: use indirect addressing.
		// basePtrSlot holds a heap pointer; LEA FPInd computes
		// *(fp+basePtrSlot) + fieldOff = &heapObj[fieldOff]
		basePtrSlot := fl.slotOf(instr.X)
		fl.emit(dis.Inst2(dis.ILEA, dis.FPInd(basePtrSlot, fieldOff), dis.FP(ptrSlot)))
	}
	return nil
}

func (fl *funcLowerer) lowerSlice(instr *ssa.Slice) error {
	xType := instr.X.Type()

	switch xt := xType.Underlying().(type) {
	case *types.Pointer:
		// *[N]T → []T: create Dis array from fixed-size array
		arrType, ok := xt.Elem().Underlying().(*types.Array)
		if !ok {
			return fmt.Errorf("Slice on non-array pointer: %v", xt.Elem())
		}
		return fl.lowerArrayToSlice(instr, arrType)
	case *types.Slice:
		// []T → []T: sub-slicing (SLICEA)
		return fl.lowerSliceSubSlice(instr)
	case *types.Basic:
		if xt.Kind() == types.String {
			return fl.lowerStringSlice(instr)
		}
		return fmt.Errorf("Slice on unsupported basic type: %v", xt)
	default:
		return fmt.Errorf("Slice on unsupported type: %T", xType.Underlying())
	}
}

func (fl *funcLowerer) lowerArrayToSlice(instr *ssa.Slice, arrType *types.Array) error {
	_, isStack := fl.allocBase[instr.X]

	if !isStack {
		// Heap array: already a Dis Array, just copy the pointer
		srcSlot := fl.slotOf(instr.X)
		dstSlot := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot), dis.FP(dstSlot)))
		return nil
	}

	// Stack array: create Dis Array via NEWA and copy elements
	elemType := arrType.Elem()
	elemDT := GoTypeToDis(elemType)
	n := int(arrType.Len())
	base := fl.allocBase[instr.X]

	elemTDIdx := fl.makeHeapTypeDesc(elemType)
	dstSlot := fl.slotOf(instr)

	// NEWA length, $elemTD, dst
	fl.emit(dis.NewInst(dis.INEWA, dis.Imm(int32(n)), dis.Imm(int32(elemTDIdx)), dis.FP(dstSlot)))

	for i := 0; i < n; i++ {
		// Get element address in Dis array using INDW
		tempAddr := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(dstSlot), dis.FP(tempAddr), dis.Imm(int32(i))))

		// Copy from stack to Dis array element
		srcOff := base + int32(i)*elemDT.Size
		if elemDT.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcOff), dis.FPInd(tempAddr, 0)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcOff), dis.FPInd(tempAddr, 0)))
		}
	}

	return nil
}

// lowerSliceSubSlice handles s[low:high] on a slice type using SLICEA.
// SLICEA: src=start, mid=end, dst=array (modifies dst in-place)
func (fl *funcLowerer) lowerSliceSubSlice(instr *ssa.Slice) error {
	srcSlot := fl.materialize(instr.X)
	dstSlot := fl.slotOf(instr)

	// Copy source to destination first (SLICEA modifies dst in-place)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot), dis.FP(dstSlot)))

	// Low: default 0
	var lowOp dis.Operand
	if instr.Low != nil {
		lowOp = fl.operandOf(instr.Low)
	} else {
		lowOp = dis.Imm(0)
	}

	// High: default len(src)
	var highOp dis.Operand
	if instr.High != nil {
		highOp = fl.operandOf(instr.High)
	} else {
		lenSlot := fl.frame.AllocWord("slice.len")
		fl.emit(dis.Inst2(dis.ILENA, dis.FP(srcSlot), dis.FP(lenSlot)))
		highOp = dis.FP(lenSlot)
	}

	// SLICEA low, high, dst
	fl.emit(dis.NewInst(dis.ISLICEA, lowOp, highOp, dis.FP(dstSlot)))
	return nil
}

// lowerStringSlice handles s[low:high] on a string type using SLICEC.
// SLICEC: src=start, mid=end, dst=string (modifies dst in-place)
func (fl *funcLowerer) lowerStringSlice(instr *ssa.Slice) error {
	srcSlot := fl.materialize(instr.X)
	dstSlot := fl.slotOf(instr)

	// Copy source to destination first (SLICEC modifies dst in-place)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot), dis.FP(dstSlot)))

	// Low: default 0
	var lowOp dis.Operand
	if instr.Low != nil {
		lowOp = fl.operandOf(instr.Low)
	} else {
		lowOp = dis.Imm(0)
	}

	// High: default len(src)
	var highOp dis.Operand
	if instr.High != nil {
		highOp = fl.operandOf(instr.High)
	} else {
		lenSlot := fl.frame.AllocWord("strslice.len")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(srcSlot), dis.FP(lenSlot)))
		highOp = dis.FP(lenSlot)
	}

	// SLICEC low, high, dst
	fl.emit(dis.NewInst(dis.ISLICEC, lowOp, highOp, dis.FP(dstSlot)))
	return nil
}

func (fl *funcLowerer) lowerIndexAddr(instr *ssa.IndexAddr) error {
	// IndexAddr produces a pointer to an element within an array or slice.
	// Result is an interior pointer (AllocWord, not GC-traced).
	ptrSlot := fl.frame.AllocWord("iaddr:" + instr.Name())
	fl.valueMap[instr] = ptrSlot

	xType := instr.X.Type()

	switch xt := xType.Underlying().(type) {
	case *types.Pointer:
		// *[N]T — pointer to fixed-size array
		if arrType, ok := xt.Elem().Underlying().(*types.Array); ok {
			return fl.lowerArrayIndexAddr(instr, arrType, ptrSlot)
		}
		return fmt.Errorf("IndexAddr on pointer to non-array: %v", xt.Elem())
	case *types.Slice:
		// []T — Dis array, use INDW for element address
		return fl.lowerSliceIndexAddr(instr, ptrSlot)
	default:
		return fmt.Errorf("IndexAddr on unsupported type: %T (%v)", xType.Underlying(), xType)
	}
}

func (fl *funcLowerer) lowerArrayIndexAddr(instr *ssa.IndexAddr, arrType *types.Array, ptrSlot int32) error {
	base, ok := fl.allocBase[instr.X]
	if ok {
		// Stack-allocated array: elements are consecutive frame slots
		elemSize := GoTypeToDis(arrType.Elem()).Size
		if c, isConst := instr.Index.(*ssa.Const); isConst {
			idx, _ := constant.Int64Val(c.Value)
			off := int32(idx) * elemSize
			fl.emit(dis.Inst2(dis.ILEA, dis.FP(base+off), dis.FP(ptrSlot)))
		} else {
			// Dynamic index: compute address = &FP[base] + index * elemSize
			baseAddr := fl.frame.AllocWord("dynidx.base")
			fl.emit(dis.Inst2(dis.ILEA, dis.FP(base), dis.FP(baseAddr)))
			idxSlot := fl.materialize(instr.Index)
			offSlot := fl.frame.AllocWord("dynidx.off")
			fl.emit(dis.NewInst(dis.IMULW, dis.Imm(elemSize), dis.FP(idxSlot), dis.FP(offSlot)))
			fl.emit(dis.NewInst(dis.IADDW, dis.FP(offSlot), dis.FP(baseAddr), dis.FP(ptrSlot)))
		}
	} else {
		// Heap Dis Array: use INDB for byte elements, INDW for word elements
		arrSlot := fl.slotOf(instr.X)
		idxOp := fl.operandOf(instr.Index)
		indOp := dis.IINDW
		if IsByteType(arrType.Elem()) {
			indOp = dis.IINDB
		}
		fl.emit(dis.NewInst(indOp, dis.FP(arrSlot), dis.FP(ptrSlot), idxOp))
	}
	return nil
}

func (fl *funcLowerer) lowerSliceIndexAddr(instr *ssa.IndexAddr, ptrSlot int32) error {
	// IND{W,B}: src=array, mid=resultAddr, dst=index
	// Bounds-checked: panics if index >= len or array is nil
	arrSlot := fl.slotOf(instr.X)
	idxOp := fl.operandOf(instr.Index)

	// Use INDB for byte slices, INDW for word-sized elements
	sliceType := instr.X.Type().Underlying().(*types.Slice)
	indOp := dis.IINDW
	if IsByteType(sliceType.Elem()) {
		indOp = dis.IINDB
	}
	fl.emit(dis.NewInst(indOp, dis.FP(arrSlot), dis.FP(ptrSlot), idxOp))
	return nil
}

// lowerIndex handles *ssa.Index: element access on arrays and strings.
// For strings, uses INDC; for arrays, uses IND{W,B} + load through pointer.
func (fl *funcLowerer) lowerIndex(instr *ssa.Index) error {
	xType := instr.X.Type().Underlying()
	dstSlot := fl.slotOf(instr)

	if basic, ok := xType.(*types.Basic); ok && basic.Kind() == types.String {
		// String indexing: INDC src=string, mid=index, dst=result(WORD)
		strOp := fl.operandOf(instr.X)
		idxOp := fl.operandOf(instr.Index)
		fl.emit(dis.NewInst(dis.IINDC, strOp, idxOp, dis.FP(dstSlot)))
		return nil
	}

	return fmt.Errorf("Index on unsupported type: %T (%v)", xType, xType)
}

// emitFreeVarLoads emits preamble instructions for an inner function to load
// free variables from the closure struct into frame slots.
// Closure struct layout: {freevar0, freevar1, ...}
func (fl *funcLowerer) emitFreeVarLoads() {
	off := int32(0)
	iby2wd := int32(dis.IBY2WD)
	for _, fv := range fl.fn.FreeVars {
		fvSlot := fl.valueMap[fv]
		if _, ok := fv.Type().Underlying().(*types.Interface); ok {
			// Interface free var: load 2 words (tag + value)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(fl.closurePtrSlot, off), dis.FP(fvSlot)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(fl.closurePtrSlot, off+iby2wd), dis.FP(fvSlot+iby2wd)))
			off += 2 * iby2wd
		} else {
			dt := GoTypeToDis(fv.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(fl.closurePtrSlot, off), dis.FP(fvSlot)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(fl.closurePtrSlot, off), dis.FP(fvSlot)))
			}
			off += dt.Size
		}
	}
}

// lowerMakeClosure creates a heap-allocated closure struct containing captured
// free variables. The inner function is resolved statically at call sites.
// Layout: {freevar0, freevar1, ...}
func (fl *funcLowerer) lowerMakeClosure(instr *ssa.MakeClosure) error {
	innerFn := instr.Fn.(*ssa.Function)
	bindings := instr.Bindings

	// Register this MakeClosure so call sites can resolve the inner function
	fl.comp.registerClosure(instr, innerFn)

	// Build closure struct type descriptor for free vars only
	closureSize := int32(0)
	var ptrOffsets []int
	for _, binding := range bindings {
		dt := GoTypeToDis(binding.Type())
		if dt.IsPtr {
			ptrOffsets = append(ptrOffsets, int(closureSize))
		}
		closureSize += dt.Size
	}

	if closureSize == 0 {
		// No free vars — closure is just a function reference
		// Store nil (H) as the closure pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, dis.Imm(0), dis.FP(dst)))
		return nil
	}

	td := dis.NewTypeDesc(0, int(closureSize))
	for _, off := range ptrOffsets {
		td.SetPointer(off)
	}
	fl.callTypeDescs = append(fl.callTypeDescs, td)
	closureTDIdx := len(fl.callTypeDescs) - 1

	// NEW closure struct
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.INEW, dis.Imm(int32(closureTDIdx)), dis.FP(dst)))

	// Store free var values
	iby2wd := int32(dis.IBY2WD)
	off := int32(0)
	for _, binding := range bindings {
		if _, ok := binding.Type().Underlying().(*types.Interface); ok {
			// Interface binding: store 2 words (tag + value)
			srcSlot := fl.slotOf(binding)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot), dis.FPInd(dst, off)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot+iby2wd), dis.FPInd(dst, off+iby2wd)))
			off += 2 * iby2wd
		} else {
			src := fl.operandOf(binding)
			dt := GoTypeToDis(binding.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, src, dis.FPInd(dst, off)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, src, dis.FPInd(dst, off)))
			}
			off += dt.Size
		}
	}

	return nil
}

// lowerClosureCall emits a statically-resolved call through a closure.
// The target inner function is determined by tracing the callee value back to
// its MakeClosure. The closure struct pointer is passed as a hidden first param.
func (fl *funcLowerer) lowerClosureCall(instr *ssa.Call) error {
	call := instr.Call

	// Resolve the target inner function statically
	innerFn := fl.comp.resolveClosureTarget(call.Value)
	if innerFn == nil {
		return fmt.Errorf("cannot statically resolve closure target for %v", call.Value)
	}

	closureSlot := fl.slotOf(call.Value)

	// Set up callee frame (NOT a GC pointer)
	callFrame := fl.frame.AllocWord("")

	// IFRAME with inner function's TD (placeholder, patched later)
	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	// Pass closure pointer as hidden first param at MaxTemp+0
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(closureSlot), dis.FPInd(callFrame, int32(dis.MaxTemp))))

	// Pass actual args starting at MaxTemp+8 (after closure pointer)
	iby2wd := int32(dis.IBY2WD)
	calleeOff := int32(dis.MaxTemp) + iby2wd
	for _, arg := range call.Args {
		argOff := fl.materialize(arg)
		if _, ok := arg.Type().Underlying().(*types.Interface); ok {
			// Interface arg: copy 2 words (tag + value)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(argOff), dis.FPInd(callFrame, calleeOff)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(argOff+iby2wd), dis.FPInd(callFrame, calleeOff+iby2wd)))
			calleeOff += 2 * iby2wd
		} else {
			dt := GoTypeToDis(arg.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FP(argOff), dis.FPInd(callFrame, calleeOff)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FP(argOff), dis.FPInd(callFrame, calleeOff)))
			}
			calleeOff += dt.Size
		}
	}

	// Set up REGRET if function returns a value
	sig := call.Value.Type().Underlying().(*types.Signature)
	if sig.Results().Len() > 0 {
		retSlot := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(retSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
	}

	// CALL with direct PC (placeholder, patched later)
	icallIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.ICALL, dis.FP(callFrame), dis.Imm(0)))

	// Record patches — same as direct calls
	fl.funcCallPatches = append(fl.funcCallPatches,
		funcCallPatch{instIdx: iframeIdx, callee: innerFn, patchKind: patchIFRAME},
		funcCallPatch{instIdx: icallIdx, callee: innerFn, patchKind: patchICALL},
	)

	return nil
}

// ============================================================
// Map operations
//
// Go maps are lowered to a heap-allocated ADT with parallel arrays:
//   offset 0:  PTR  keys array
//   offset 8:  PTR  values array
//   offset 16: WORD count
//
// Operations use linear scan (O(n) per lookup/update/delete).
// ============================================================

// lowerMakeMap creates a new empty map.
func (fl *funcLowerer) lowerMakeMap(instr *ssa.MakeMap) error {
	mapTDIdx := fl.makeMapTD()
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.INEW, dis.Imm(int32(mapTDIdx)), dis.FP(dst)))
	// Initialize pointer fields to H (-1) since NEW memsets to 0.
	// SLICELA treats 0 as a valid pointer and crashes; it only skips H.
	// Use MOVW (not MOVP) to avoid destroy(0) on the freshly zeroed slots.
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FPInd(dst, 0)))  // keys = H
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FPInd(dst, 8)))  // values = H
	// count stays at 0
	return nil
}

// lowerMapUpdate inserts or updates a key-value pair in a map.
// Flow: scan for existing key → found: update value, done
//                              → not found: grow arrays, append, done
func (fl *funcLowerer) lowerMapUpdate(instr *ssa.MapUpdate) error {
	mapSlot := fl.slotOf(instr.Map)
	keySlot := fl.materialize(instr.Key)
	valSlot := fl.materialize(instr.Value)

	mapType := instr.Map.Type().Underlying().(*types.Map)
	keyType := mapType.Key()
	valType := mapType.Elem()

	// Allocate temps. Pointer temps are initialized to H via MOVW to avoid
	// destroy(0) crash on frame free if a code path doesn't write them.
	cnt := fl.frame.AllocWord("mu.cnt")
	idx := fl.frame.AllocWord("mu.idx")
	keysArr := fl.allocPtrTemp("mu.keys")
	valsArr := fl.allocPtrTemp("mu.vals")
	tmpPtr := fl.frame.AllocWord("mu.ptr") // interior pointer, non-GC
	tmpKey := fl.allocMapKeyTemp(keyType, "mu.tmpk")
	newCnt := fl.frame.AllocWord("mu.ncnt")
	newKeys := fl.allocPtrTemp("mu.nkeys")
	newVals := fl.allocPtrTemp("mu.nvals")

	// Load count
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 16), dis.FP(cnt)))

	// if cnt == 0, skip scan → goto grow
	skipScanIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(cnt), dis.Imm(0), dis.Imm(0)))

	// Load keys array for scanning
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 0), dis.FP(keysArr)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(idx)))

	// Scan loop
	loopPC := int32(len(fl.insts))
	loopEndIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(idx), dis.FP(cnt), dis.Imm(0)))

	// Load keys[idx]
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitLoadThrough(tmpKey, tmpPtr, keyType)

	// Compare: if keys[idx] == target key, goto found
	foundIdx := len(fl.insts)
	fl.emit(dis.NewInst(fl.mapKeyBranchEq(keyType), dis.FP(tmpKey), dis.FP(keySlot), dis.Imm(0)))

	// idx++, loop back
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(idx), dis.FP(idx)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// === found: update value at idx ===
	foundPC := int32(len(fl.insts))
	fl.insts[foundIdx].Dst = dis.Imm(foundPC)

	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 8), dis.FP(valsArr)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(valsArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitStoreThrough(valSlot, tmpPtr, valType)

	doneJmp := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// === grow: append new entry ===
	growPC := int32(len(fl.insts))
	fl.insts[skipScanIdx].Dst = dis.Imm(growPC)
	fl.insts[loopEndIdx].Dst = dis.Imm(growPC)

	// newCnt = cnt + 1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(cnt), dis.FP(newCnt)))

	// New keys array, copy old (SLICELA skips if source is H), store new key
	keyTDIdx := fl.makeHeapTypeDesc(keyType)
	fl.emit(dis.NewInst(dis.INEWA, dis.FP(newCnt), dis.Imm(int32(keyTDIdx)), dis.FP(newKeys)))
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FPInd(mapSlot, 0), dis.Imm(0), dis.FP(newKeys)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(newKeys), dis.FP(tmpPtr), dis.FP(cnt)))
	fl.emitStoreThrough(keySlot, tmpPtr, keyType)

	// New values array, copy old, store new value
	valTDIdx := fl.makeHeapTypeDesc(valType)
	fl.emit(dis.NewInst(dis.INEWA, dis.FP(newCnt), dis.Imm(int32(valTDIdx)), dis.FP(newVals)))
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FPInd(mapSlot, 8), dis.Imm(0), dis.FP(newVals)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(newVals), dis.FP(tmpPtr), dis.FP(cnt)))
	fl.emitStoreThrough(valSlot, tmpPtr, valType)

	// Update map struct
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(newKeys), dis.FPInd(mapSlot, 0)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(newVals), dis.FPInd(mapSlot, 8)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(newCnt), dis.FPInd(mapSlot, 16)))

	// === done ===
	donePC := int32(len(fl.insts))
	fl.insts[doneJmp].Dst = dis.Imm(donePC)

	return nil
}

// lowerLookup handles map key lookup and string indexing.
func (fl *funcLowerer) lowerLookup(instr *ssa.Lookup) error {
	switch instr.X.Type().Underlying().(type) {
	case *types.Map:
		return fl.lowerMapLookup(instr)
	default:
		// String indexing: s[i] → byte value
		return fl.lowerStringIndex(instr)
	}
}

// lowerStringIndex handles s[i] on a string using INDC.
// INDC: src=string, mid=index(WORD), dst=result(WORD)
func (fl *funcLowerer) lowerStringIndex(instr *ssa.Lookup) error {
	strOp := fl.operandOf(instr.X)
	idxOp := fl.operandOf(instr.Index)
	dstSlot := fl.slotOf(instr)

	// INDC string, index, result
	fl.emit(dis.NewInst(dis.IINDC, strOp, idxOp, dis.FP(dstSlot)))
	return nil
}

// lowerMapLookup scans the parallel key array for a match and returns the value.
// If CommaOk, returns (value, bool) tuple; otherwise just the value (zero if missing).
func (fl *funcLowerer) lowerMapLookup(instr *ssa.Lookup) error {
	mapSlot := fl.slotOf(instr.X)
	keySlot := fl.materialize(instr.Index)

	mapType := instr.X.Type().Underlying().(*types.Map)
	keyType := mapType.Key()
	valType := mapType.Elem()

	// Result slots
	var valDst, okDst int32
	if instr.CommaOk {
		tupleBase := fl.slotOf(instr)
		valDst = tupleBase
		okDst = tupleBase + int32(dis.IBY2WD)
	} else {
		valDst = fl.slotOf(instr)
	}

	// Initialize result: value = 0, ok = false
	valDT := GoTypeToDis(valType)
	if valDT.IsPtr {
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(valDst))) // H for pointer zero-val
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(valDst)))
	}
	if instr.CommaOk {
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(okDst)))
	}

	// Temps
	cnt := fl.frame.AllocWord("lu.cnt")
	idx := fl.frame.AllocWord("lu.idx")
	keysArr := fl.allocPtrTemp("lu.keys")
	valsArr := fl.allocPtrTemp("lu.vals")
	tmpPtr := fl.frame.AllocWord("lu.ptr")
	tmpKey := fl.allocMapKeyTemp(keyType, "lu.tmpk")

	// Load count
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 16), dis.FP(cnt)))

	// if cnt == 0, goto done (empty map)
	skipIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(cnt), dis.Imm(0), dis.Imm(0)))

	// Load keys array
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 0), dis.FP(keysArr)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(idx)))

	// Scan loop
	loopPC := int32(len(fl.insts))
	loopEndIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(idx), dis.FP(cnt), dis.Imm(0)))

	// Load keys[idx]
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitLoadThrough(tmpKey, tmpPtr, keyType)

	// Compare
	foundIdx := len(fl.insts)
	fl.emit(dis.NewInst(fl.mapKeyBranchEq(keyType), dis.FP(tmpKey), dis.FP(keySlot), dis.Imm(0)))

	// idx++, loop back
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(idx), dis.FP(idx)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// === found: load value ===
	foundPC := int32(len(fl.insts))
	fl.insts[foundIdx].Dst = dis.Imm(foundPC)

	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 8), dis.FP(valsArr)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(valsArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitLoadThrough(valDst, tmpPtr, valType)

	if instr.CommaOk {
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(okDst)))
	}

	// === done ===
	donePC := int32(len(fl.insts))
	fl.insts[skipIdx].Dst = dis.Imm(donePC)
	fl.insts[loopEndIdx].Dst = dis.Imm(donePC)

	return nil
}

// lowerMapDelete removes a key from a map using swap-with-last strategy.
func (fl *funcLowerer) lowerMapDelete(instr *ssa.Call) error {
	mapArg := instr.Call.Args[0]
	keyArg := instr.Call.Args[1]
	mapSlot := fl.slotOf(mapArg)
	keySlot := fl.materialize(keyArg)

	mapType := mapArg.Type().Underlying().(*types.Map)
	keyType := mapType.Key()
	valType := mapType.Elem()

	// Temps
	cnt := fl.frame.AllocWord("dl.cnt")
	idx := fl.frame.AllocWord("dl.idx")
	keysArr := fl.allocPtrTemp("dl.keys")
	valsArr := fl.allocPtrTemp("dl.vals")
	tmpPtr := fl.frame.AllocWord("dl.ptr")
	tmpPtr2 := fl.frame.AllocWord("dl.ptr2")
	tmpKey := fl.allocMapKeyTemp(keyType, "dl.tmpk")
	lastIdx := fl.frame.AllocWord("dl.last")

	// Load count
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 16), dis.FP(cnt)))

	// if cnt == 0, nothing to delete
	skipIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(cnt), dis.Imm(0), dis.Imm(0)))

	// Load arrays
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 0), dis.FP(keysArr)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 8), dis.FP(valsArr)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(idx)))

	// Scan
	loopPC := int32(len(fl.insts))
	loopEndIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(idx), dis.FP(cnt), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitLoadThrough(tmpKey, tmpPtr, keyType)

	foundIdx := len(fl.insts)
	fl.emit(dis.NewInst(fl.mapKeyBranchEq(keyType), dis.FP(tmpKey), dis.FP(keySlot), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(idx), dis.FP(idx)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// === found: swap with last, decrement count ===
	foundPC := int32(len(fl.insts))
	fl.insts[foundIdx].Dst = dis.Imm(foundPC)

	// lastIdx = cnt - 1
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(cnt), dis.FP(lastIdx)))

	// if idx == lastIdx, skip swap (already at end)
	skipSwapIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(idx), dis.FP(lastIdx), dis.Imm(0)))

	// keys[idx] = keys[lastIdx]
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr2), dis.FP(lastIdx)))
	fl.emitLoadThrough(tmpKey, tmpPtr2, keyType)
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitStoreThrough(tmpKey, tmpPtr, keyType)

	// values[idx] = values[lastIdx]
	tmpVal := fl.frame.AllocWord("dl.tmpv")
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(valsArr), dis.FP(tmpPtr2), dis.FP(lastIdx)))
	fl.emitLoadThrough(tmpVal, tmpPtr2, valType)
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(valsArr), dis.FP(tmpPtr), dis.FP(idx)))
	fl.emitStoreThrough(tmpVal, tmpPtr, valType)

	// skip swap target
	skipSwapPC := int32(len(fl.insts))
	fl.insts[skipSwapIdx].Dst = dis.Imm(skipSwapPC)

	// count--
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lastIdx), dis.FPInd(mapSlot, 16)))

	// === done ===
	donePC := int32(len(fl.insts))
	fl.insts[skipIdx].Dst = dis.Imm(donePC)
	fl.insts[loopEndIdx].Dst = dis.Imm(donePC)

	return nil
}

// lowerRange initializes a map or string iterator.
// For maps: the iterator is an index (WORD) starting at 0.
func (fl *funcLowerer) lowerRange(instr *ssa.Range) error {
	iterSlot := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iterSlot)))
	return nil
}

// lowerNext advances a map iterator and returns (ok, key, value).
func (fl *funcLowerer) lowerNext(instr *ssa.Next) error {
	if instr.IsString {
		return fl.lowerStringNext(instr)
	}

	rangeInstr := instr.Iter.(*ssa.Range)
	mapSlot := fl.slotOf(rangeInstr.X)
	iterSlot := fl.slotOf(rangeInstr)

	mapType := rangeInstr.X.Type().Underlying().(*types.Map)
	keyType := mapType.Key()
	valType := mapType.Elem()

	// Result tuple: (ok WORD @0, key @8, value @16+)
	tupleBase := fl.slotOf(instr)
	okSlot := tupleBase
	keyDT := GoTypeToDis(keyType)
	keySlot := tupleBase + int32(dis.IBY2WD)
	valSlot := keySlot + keyDT.Size

	// Load count from map
	cnt := fl.frame.AllocWord("next.cnt")
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 16), dis.FP(cnt)))

	// if index < count goto hasMore
	hasMoreIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(iterSlot), dis.FP(cnt), dis.Imm(0)))

	// exhausted: ok = false, jump to end
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(okSlot)))
	endIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// hasMore:
	fl.insts[hasMoreIdx].Dst = dis.Imm(int32(len(fl.insts)))

	tmpPtr := fl.frame.AllocWord("next.ptr")

	// Only load key if its tuple slot type is valid (not _ blank identifier).
	// When the key is unused, SSA gives it types.Invalid which allocates as WORD,
	// but the actual map key may be a pointer type. MOVP into a WORD slot
	// whose initial value is 0 (not H) crashes on destroy(0).
	tup := instr.Type().(*types.Tuple)
	if tup.At(1).Type() != types.Typ[types.Invalid] {
		keysArr := fl.frame.AllocWord("next.keys")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 0), dis.FP(keysArr)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(keysArr), dis.FP(tmpPtr), dis.FP(iterSlot)))
		fl.emitLoadThrough(keySlot, tmpPtr, keyType)
	}

	// Only load value if its tuple slot type is valid.
	if tup.At(2).Type() != types.Typ[types.Invalid] {
		valsArr := fl.frame.AllocWord("next.vals")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 8), dis.FP(valsArr)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(valsArr), dis.FP(tmpPtr), dis.FP(iterSlot)))
		fl.emitLoadThrough(valSlot, tmpPtr, valType)
	}

	// ok = true
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(okSlot)))

	// index++
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iterSlot), dis.FP(iterSlot)))

	// end:
	fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
	return nil
}

// lowerStringNext advances a string iterator and returns (ok, index, rune).
func (fl *funcLowerer) lowerStringNext(instr *ssa.Next) error {
	rangeInstr := instr.Iter.(*ssa.Range)
	strSlot := fl.materialize(rangeInstr.X)
	iterSlot := fl.slotOf(rangeInstr)

	// Result tuple: (ok WORD @0, index WORD @8, rune WORD @16)
	tupleBase := fl.slotOf(instr)
	okSlot := tupleBase
	idxSlot := tupleBase + int32(dis.IBY2WD)
	runeSlot := idxSlot + int32(dis.IBY2WD)

	// Get string length
	lenSlot := fl.frame.AllocWord("strnext.len")
	fl.emit(dis.Inst2(dis.ILENC, dis.FP(strSlot), dis.FP(lenSlot)))

	// if index < length goto hasMore
	hasMoreIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(iterSlot), dis.FP(lenSlot), dis.Imm(0)))

	// exhausted: ok = false, jump to end
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(okSlot)))
	endIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// hasMore:
	fl.insts[hasMoreIdx].Dst = dis.Imm(int32(len(fl.insts)))

	tup := instr.Type().(*types.Tuple)

	// Load index (byte position) if used
	if tup.At(1).Type() != types.Typ[types.Invalid] {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(iterSlot), dis.FP(idxSlot)))
	}

	// Load rune at current index if used
	if tup.At(2).Type() != types.Typ[types.Invalid] {
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(strSlot), dis.FP(iterSlot), dis.FP(runeSlot)))
	}

	// ok = true
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(okSlot)))

	// index++ (advance by 1 character)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iterSlot), dis.FP(iterSlot)))

	// end:
	fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
	return nil
}

// makeMapTD creates a type descriptor for the map ADT: {keys PTR, values PTR, count WORD}.
func (fl *funcLowerer) makeMapTD() int {
	td := dis.NewTypeDesc(0, 24) // 3 * 8 bytes
	td.SetPointer(0)             // keys
	td.SetPointer(8)             // values
	fl.callTypeDescs = append(fl.callTypeDescs, td)
	return len(fl.callTypeDescs) - 1
}

// allocPtrTemp allocates a GC-traced pointer slot and initializes it to H (-1)
// using MOVW (not MOVP) to avoid destroy(0) on the zeroed frame slot.
func (fl *funcLowerer) allocPtrTemp(name string) int32 {
	slot := fl.frame.AllocPointer(name)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(slot)))
	return slot
}

// allocMapKeyTemp allocates a temp slot for a map key, with H init for pointer types.
func (fl *funcLowerer) allocMapKeyTemp(keyType types.Type, name string) int32 {
	dt := GoTypeToDis(keyType)
	if dt.IsPtr {
		return fl.allocPtrTemp(name)
	}
	return fl.frame.AllocWord(name)
}

// mapKeyBranchEq returns the branch-if-equal opcode for the given key type.
func (fl *funcLowerer) mapKeyBranchEq(keyType types.Type) dis.Op {
	if isStringType(keyType.Underlying()) {
		return dis.IBEQC
	}
	return dis.IBEQW
}

// emitLoadThrough loads a value from memory at *ptrSlot into dstSlot.
func (fl *funcLowerer) emitLoadThrough(dstSlot, ptrSlot int32, t types.Type) {
	dt := GoTypeToDis(t)
	if dt.IsPtr {
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(ptrSlot, 0), dis.FP(dstSlot)))
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(ptrSlot, 0), dis.FP(dstSlot)))
	}
}

// emitStoreThrough stores a value from srcSlot to memory at *ptrSlot.
func (fl *funcLowerer) emitStoreThrough(srcSlot, ptrSlot int32, t types.Type) {
	dt := GoTypeToDis(t)
	if dt.IsPtr {
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot), dis.FPInd(ptrSlot, 0)))
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot), dis.FPInd(ptrSlot, 0)))
	}
}

// ============================================================
// Defer operations
//
// Static defers are inlined at RunDefers sites in LIFO order.
// SSA values are immutable, so arguments captured at Defer time
// remain valid at RunDefers time.
// ============================================================

// lowerDefer captures a deferred call onto the LIFO stack.
// No instructions are emitted — the call is expanded at RunDefers time.
func (fl *funcLowerer) lowerDefer(instr *ssa.Defer) error {
	fl.deferStack = append(fl.deferStack, instr.Call)
	return nil
}

// lowerRunDefers emits all deferred calls in LIFO order (last defer = first call).
func (fl *funcLowerer) lowerRunDefers(instr *ssa.RunDefers) error {
	for i := len(fl.deferStack) - 1; i >= 0; i-- {
		call := fl.deferStack[i]
		if err := fl.emitDeferredCall(call); err != nil {
			return fmt.Errorf("deferred call %d: %w", i, err)
		}
	}
	return nil
}

// emitDeferredCall dispatches a single deferred call by callee type.
func (fl *funcLowerer) emitDeferredCall(call ssa.CallCommon) error {
	switch callee := call.Value.(type) {
	case *ssa.Builtin:
		return fl.emitDeferredBuiltin(callee, call.Args)
	case *ssa.Function:
		fl.emitDeferredDirectCall(callee, call.Args)
		return nil
	default:
		// Closure call (function value with Signature type)
		if _, ok := call.Value.Type().Underlying().(*types.Signature); ok {
			return fl.emitDeferredClosureCall(call)
		}
		return fmt.Errorf("unsupported deferred call target: %T", call.Value)
	}
}

// emitDeferredBuiltin handles deferred builtin calls (e.g., defer println("bye")).
func (fl *funcLowerer) emitDeferredBuiltin(builtin *ssa.Builtin, args []ssa.Value) error {
	switch builtin.Name() {
	case "println", "print":
		for i, arg := range args {
			if i > 0 {
				fl.emitSysPrint(" ")
			}
			if err := fl.emitPrintArg(arg); err != nil {
				return err
			}
		}
		fl.emitSysPrint("\n")
		return nil
	default:
		return fmt.Errorf("unsupported deferred builtin: %s", builtin.Name())
	}
}

// emitDeferredDirectCall emits IFRAME + args + CALL for a deferred function.
func (fl *funcLowerer) emitDeferredDirectCall(callee *ssa.Function, args []ssa.Value) {
	type argInfo struct {
		off   int32
		isPtr bool
		st    *types.Struct
	}
	var argInfos []argInfo
	for _, arg := range args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		var st *types.Struct
		if s, ok := arg.Type().Underlying().(*types.Struct); ok {
			st = s
		}
		argInfos = append(argInfos, argInfo{off, dt.IsPtr, st})
	}

	callFrame := fl.frame.AllocWord("")
	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	calleeOff := int32(dis.MaxTemp)
	for _, a := range argInfos {
		if a.st != nil {
			fieldOff := int32(0)
			for i := 0; i < a.st.NumFields(); i++ {
				fdt := GoTypeToDis(a.st.Field(i).Type())
				if fdt.IsPtr {
					fl.emit(dis.Inst2(dis.IMOVP, dis.FP(a.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				} else {
					fl.emit(dis.Inst2(dis.IMOVW, dis.FP(a.off+fieldOff), dis.FPInd(callFrame, calleeOff+fieldOff)))
				}
				fieldOff += fdt.Size
			}
			calleeOff += GoTypeToDis(a.st).Size
		} else if a.isPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(a.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += int32(dis.IBY2WD)
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(a.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += int32(dis.IBY2WD)
		}
	}

	icallIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.ICALL, dis.FP(callFrame), dis.Imm(0)))

	fl.funcCallPatches = append(fl.funcCallPatches,
		funcCallPatch{instIdx: iframeIdx, callee: callee, patchKind: patchIFRAME},
		funcCallPatch{instIdx: icallIdx, callee: callee, patchKind: patchICALL},
	)
}

// emitDeferredClosureCall emits a call to a deferred closure.
func (fl *funcLowerer) emitDeferredClosureCall(call ssa.CallCommon) error {
	innerFn := fl.comp.resolveClosureTarget(call.Value)
	if innerFn == nil {
		return fmt.Errorf("cannot statically resolve deferred closure target for %v", call.Value)
	}

	closureSlot := fl.slotOf(call.Value)
	callFrame := fl.frame.AllocWord("")

	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	// Pass closure pointer as hidden first param
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(closureSlot), dis.FPInd(callFrame, int32(dis.MaxTemp))))

	// Pass actual args
	calleeOff := int32(dis.MaxTemp + dis.IBY2WD)
	for _, arg := range call.Args {
		argOff := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(argOff), dis.FPInd(callFrame, calleeOff)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(argOff), dis.FPInd(callFrame, calleeOff)))
		}
		calleeOff += dt.Size
	}

	icallIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.ICALL, dis.FP(callFrame), dis.Imm(0)))

	fl.funcCallPatches = append(fl.funcCallPatches,
		funcCallPatch{instIdx: iframeIdx, callee: innerFn, patchKind: patchIFRAME},
		funcCallPatch{instIdx: icallIdx, callee: innerFn, patchKind: patchICALL},
	)
	return nil
}

// emitZeroDivCheck emits an explicit zero-divisor check before integer division.
// ARM64's sdiv instruction returns 0 for division by zero instead of trapping,
// so we must check explicitly and raise "zero divide" to match Go semantics.
// Layout: BNEW divisor, $0, $+2; RAISE "zero divide"(mp)
func (fl *funcLowerer) emitZeroDivCheck(divisor dis.Operand) {
	zdivStr := fl.comp.AllocString("zero divide")
	skipPC := int32(len(fl.insts)) + 2 // skip over BNEW and RAISE
	fl.emit(dis.NewInst(dis.IBNEW, divisor, dis.Imm(0), dis.Imm(skipPC)))
	fl.emit(dis.Inst{Op: dis.IRAISE, Src: dis.MP(zdivStr), Mid: dis.NoOperand, Dst: dis.NoOperand})
}

// lowerPanic emits IRAISE with the panic value (a string).
// IRAISE: src = pointer to string exception value.
func (fl *funcLowerer) lowerPanic(instr *ssa.Panic) error {
	src := fl.operandOf(instr.X)
	fl.emit(dis.Inst{Op: dis.IRAISE, Src: src, Mid: dis.NoOperand, Dst: dis.NoOperand})
	return nil
}

// copyIface copies a 2-word interface value (tag + value) between frame slots.
func (fl *funcLowerer) copyIface(src, dst int32) {
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(src), dis.FP(dst)))                          // tag
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(src+int32(dis.IBY2WD)), dis.FP(dst+int32(dis.IBY2WD)))) // value
}

// concreteTypeName extracts the concrete type name for type tag allocation.
func concreteTypeName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return t.String()
}

// lowerMakeInterface stores the underlying value into a tagged interface slot.
// Interface layout: [tag (WORD)] [value (WORD)].
// Tag is the type tag ID for the concrete type.
// Value is the raw value (for ≤1 word) or pointer to struct data (for >1 word).
func (fl *funcLowerer) lowerMakeInterface(instr *ssa.MakeInterface) error {
	srcSlot := fl.materialize(instr.X)
	dstSlot := fl.slotOf(instr)
	tag := fl.comp.AllocTypeTag(concreteTypeName(instr.X.Type()))
	dt := GoTypeToDis(instr.X.Type())

	// Store type tag at dst+0
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(tag), dis.FP(dstSlot)))
	// Store value at dst+8
	if dt.Size > int32(dis.IBY2WD) {
		// Struct or multi-word: store address of the data
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(srcSlot), dis.FP(dstSlot+int32(dis.IBY2WD))))
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot), dis.FP(dstSlot+int32(dis.IBY2WD))))
	}
	return nil
}

// lowerTypeAssert extracts a concrete value from a tagged interface.
// Interface layout: [tag (WORD)] [value (WORD)].
// For non-commaok: checks tag, panics on mismatch.
// For commaok: checks tag, returns (value, ok).
func (fl *funcLowerer) lowerTypeAssert(instr *ssa.TypeAssert) error {
	srcSlot := fl.slotOf(instr.X) // interface base: tag at srcSlot, value at srcSlot+8
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	// Check if asserting to another interface type
	if _, isIface := instr.AssertedType.Underlying().(*types.Interface); isIface {
		// Interface-to-interface: just copy the 2 words (tag stays the same)
		if instr.CommaOk {
			// Result tuple: (interface, bool)
			fl.copyIface(srcSlot, dst)
			// ok = (tag != 0) — any non-nil interface satisfies
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+2*iby2wd)))
		} else {
			fl.copyIface(srcSlot, dst)
		}
		return nil
	}

	dt := GoTypeToDis(instr.AssertedType)
	tag := fl.comp.AllocTypeTag(concreteTypeName(instr.AssertedType))

	if instr.CommaOk {
		// Result is a tuple: (value, bool).
		// Layout depends on value size: value at dst, ok after value.
		//   BEQW $tag, FP(srcSlot), $match_pc
		//   MOVW $0, FP(dst)          ; value = 0
		//   MOVW $0, FP(dst+dtSize)   ; ok = false
		//   JMP $done_pc
		// match_pc:
		//   MOVW/MOVP FP(srcSlot+8) → FP(dst)  ; copy value
		//   MOVW $1, FP(dst+dtSize)             ; ok = true
		// done_pc:

		beqwIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(tag), dis.FP(srcSlot), dis.Imm(0))) // placeholder

		// No match path
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+dt.Size)))
		jmpIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0))) // placeholder

		// Match path
		matchPC := int32(len(fl.insts))
		fl.insts[beqwIdx].Dst = dis.Imm(matchPC)
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot+iby2wd), dis.FP(dst)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot+iby2wd), dis.FP(dst)))
		}
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+dt.Size)))

		// Patch JMP
		donePC := int32(len(fl.insts))
		fl.insts[jmpIdx].Dst = dis.Imm(donePC)
	} else {
		// Non-commaok: check tag, panic on mismatch
		//   BEQW $tag, FP(srcSlot), $ok_pc
		//   RAISE "interface conversion"
		// ok_pc:
		//   MOVW/MOVP FP(srcSlot+8) → FP(dst)

		beqwIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(tag), dis.FP(srcSlot), dis.Imm(0))) // placeholder

		// Panic path
		panicStr := fl.comp.AllocString("interface conversion")
		panicSlot := fl.frame.AllocPointer("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(panicStr), dis.FP(panicSlot)))
		fl.emit(dis.Inst1(dis.IRAISE, dis.FP(panicSlot)))

		// OK path
		okPC := int32(len(fl.insts))
		fl.insts[beqwIdx].Dst = dis.Imm(okPC)
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcSlot+iby2wd), dis.FP(dst)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcSlot+iby2wd), dis.FP(dst)))
		}
	}
	return nil
}

// lowerChangeInterface converts between interface types (copy 2-word tag+value).
func (fl *funcLowerer) lowerChangeInterface(instr *ssa.ChangeInterface) error {
	srcSlot := fl.slotOf(instr.X)
	dstSlot := fl.slotOf(instr)
	fl.copyIface(srcSlot, dstSlot)
	return nil
}

func (fl *funcLowerer) lowerConvert(instr *ssa.Convert) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)

	srcType := instr.X.Type().Underlying()
	dstType := instr.Type().Underlying()

	// string → []byte (CVTCA)
	if isStringType(srcType) && isByteSlice(dstType) {
		fl.emit(dis.Inst2(dis.ICVTCA, src, dis.FP(dst)))
		return nil
	}

	// []byte → string (CVTAC)
	if isByteSlice(srcType) && isStringType(dstType) {
		fl.emit(dis.Inst2(dis.ICVTAC, src, dis.FP(dst)))
		return nil
	}

	// int/rune → string (create 1-char string from character code point)
	// SSA generates this for string(rune(x)) or string(65)
	if isIntegerType(srcType) && isStringType(dstType) {
		// INSC: src=rune, mid=index(0), dst=string
		// When dst is H (nil), INSC creates a new 1-char string.
		fl.emit(dis.NewInst(dis.IINSC, src, dis.Imm(0), dis.FP(dst)))
		return nil
	}

	// Default: same-size integer/pointer conversions
	fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
	return nil
}

func isIntegerType(t types.Type) bool {
	b, ok := t.(*types.Basic)
	if !ok {
		return false
	}
	switch b.Kind() {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return true
	}
	return false
}

func isStringType(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.String
}

func isByteSlice(t types.Type) bool {
	s, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	b, ok := s.Elem().(*types.Basic)
	return ok && (b.Kind() == types.Byte || b.Kind() == types.Uint8)
}

func (fl *funcLowerer) lowerChangeType(instr *ssa.ChangeType) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)
	fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
	return nil
}

func (fl *funcLowerer) lowerExtract(instr *ssa.Extract) error {
	// Extract pulls value #Index from a tuple (multi-return).
	// The tuple is stored as consecutive frame slots.
	tupleSlot := fl.slotOf(instr.Tuple)
	tup := instr.Tuple.Type().(*types.Tuple)

	// Compute offset of element #Index within the tuple
	elemOff := int32(0)
	for i := 0; i < instr.Index; i++ {
		dt := GoTypeToDis(tup.At(i).Type())
		elemOff += dt.Size
	}

	dst := fl.slotOf(instr)
	if _, ok := instr.Type().Underlying().(*types.Interface); ok {
		// Interface extract: copy 2 words (tag + value)
		fl.copyIface(tupleSlot+elemOff, dst)
	} else {
		dt := GoTypeToDis(instr.Type())
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tupleSlot+elemOff), dis.FP(dst)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(tupleSlot+elemOff), dis.FP(dst)))
		}
	}
	return nil
}

func (fl *funcLowerer) lowerLen(instr *ssa.Call) error {
	arg := instr.Call.Args[0]
	dst := fl.slotOf(instr)
	src := fl.operandOf(arg)

	t := arg.Type().Underlying()
	switch t.(type) {
	case *types.Slice, *types.Array:
		fl.emit(dis.Inst2(dis.ILENA, src, dis.FP(dst)))
	default:
		// string
		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(dst)))
	}
	return nil
}

func (fl *funcLowerer) lowerAppend(instr *ssa.Call) error {
	args := instr.Call.Args
	if len(args) != 2 {
		return fmt.Errorf("append: expected 2 args (slice, slice...), got %d", len(args))
	}

	// SSA transforms append(s, elems...) so both args are slices
	oldSlice := args[0]
	newSlice := args[1]

	// Get element type from slice type
	sliceType := oldSlice.Type().Underlying().(*types.Slice)
	elemType := sliceType.Elem()

	oldOff := fl.slotOf(oldSlice)
	newOff := fl.slotOf(newSlice)

	// Get lengths of both slices
	oldLenSlot := fl.frame.AllocWord("append.oldlen")
	newLenSlot := fl.frame.AllocWord("append.newlen")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(oldOff), dis.FP(oldLenSlot)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(newOff), dis.FP(newLenSlot)))

	// Total length = oldLen + newLen
	totalLenSlot := fl.frame.AllocWord("append.total")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(newLenSlot), dis.FP(oldLenSlot), dis.FP(totalLenSlot)))

	// Allocate new array with total length
	elemTDIdx := fl.makeHeapTypeDesc(elemType)
	dstSlot := fl.slotOf(instr) // result slot (pointer)
	fl.emit(dis.NewInst(dis.INEWA, dis.FP(totalLenSlot), dis.Imm(int32(elemTDIdx)), dis.FP(dstSlot)))

	// Copy old elements at offset 0
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FP(oldOff), dis.Imm(0), dis.FP(dstSlot)))

	// Copy new elements at offset oldLen
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FP(newOff), dis.FP(oldLenSlot), dis.FP(dstSlot)))

	return nil
}

// lowerCap handles cap(s) — Dis arrays have len == cap.
func (fl *funcLowerer) lowerCap(instr *ssa.Call) error {
	arg := instr.Call.Args[0]
	dst := fl.slotOf(instr)
	src := fl.operandOf(arg)
	fl.emit(dis.Inst2(dis.ILENA, src, dis.FP(dst)))
	return nil
}

// lowerCopy handles copy(dst, src) on slices.
// Copies min(len(dst), len(src)) elements from src to dst[0:].
// Returns the number of elements copied.
func (fl *funcLowerer) lowerCopy(instr *ssa.Call) error {
	dstArr := instr.Call.Args[0]
	srcArr := instr.Call.Args[1]
	dstArrSlot := fl.slotOf(dstArr)
	srcArrSlot := fl.slotOf(srcArr)

	// Get lengths
	dstLen := fl.frame.AllocWord("copy.dstlen")
	srcLen := fl.frame.AllocWord("copy.srclen")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(dstArrSlot), dis.FP(dstLen)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(srcArrSlot), dis.FP(srcLen)))

	// min = srcLen (start with srcLen, reduce to dstLen if smaller)
	minLen := fl.frame.AllocWord("copy.min")
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcLen), dis.FP(minLen)))

	// if dstLen < srcLen, min = dstLen
	// BLTW: if src < mid goto dst  →  if dstLen < srcLen goto skip
	skipPC := fl.emit(dis.NewInst(dis.IBLTW, dis.FP(dstLen), dis.FP(srcLen), dis.Imm(0)))
	// dstLen >= srcLen, min stays srcLen → jump over
	noSwapPC := fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	// dstLen < srcLen, min = dstLen
	fl.insts[skipPC].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(dstLen), dis.FP(minLen)))
	fl.insts[noSwapPC].Dst = dis.Imm(int32(len(fl.insts)))

	// Sub-slice src to [0:min] in a temp
	srcCopy := fl.allocPtrTemp("copy.srctmp")
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(srcArrSlot), dis.FP(srcCopy)))
	fl.emit(dis.NewInst(dis.ISLICEA, dis.Imm(0), dis.FP(minLen), dis.FP(srcCopy)))

	// SLICELA copies srcCopy into dstArr at offset 0
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FP(srcCopy), dis.Imm(0), dis.FP(dstArrSlot)))

	// Return value = minLen (only if result is used)
	if instr.Name() != "" {
		resultSlot := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(minLen), dis.FP(resultSlot)))
	}
	return nil
}

// Helper methods

func (fl *funcLowerer) emit(inst dis.Inst) int32 {
	pc := int32(len(fl.insts))
	fl.insts = append(fl.insts, inst)
	return pc
}

func (fl *funcLowerer) slotOf(v ssa.Value) int32 {
	if off, ok := fl.valueMap[v]; ok {
		return off
	}
	// Handle globals: allocate a frame slot and emit LEA to load the MP address
	if g, ok := v.(*ssa.Global); ok {
		return fl.loadGlobalAddr(g)
	}
	// Allocate on demand
	if _, ok := v.Type().Underlying().(*types.Interface); ok {
		// Interface: 2 consecutive WORDs (tag + value)
		off := fl.frame.AllocWord(v.Name() + ".tag")
		fl.frame.AllocWord(v.Name() + ".val")
		fl.valueMap[v] = off
		return off
	}
	dt := GoTypeToDis(v.Type())
	var off int32
	if dt.IsPtr {
		off = fl.frame.AllocPointer(v.Name())
	} else {
		off = fl.frame.AllocWord(v.Name())
	}
	fl.valueMap[v] = off
	return off
}

// loadGlobalAddr allocates a frame slot and emits LEA to load a global's MP address.
// The result is cached in valueMap so LEA is only emitted once per function.
// The slot is NOT marked as a GC pointer because it points to module data (MP),
// not the heap. The GC manages MP separately via the MP type descriptor.
func (fl *funcLowerer) loadGlobalAddr(g *ssa.Global) int32 {
	mpOff, ok := fl.comp.GlobalOffset(g.Name())
	if !ok {
		elemType := g.Type().(*types.Pointer).Elem()
		dt := GoTypeToDis(elemType)
		mpOff = fl.comp.AllocGlobal(g.Name(), dt.IsPtr)
	}
	slot := fl.frame.AllocWord("gaddr:" + g.Name()) // NOT pointer: MP address, not heap
	fl.emit(dis.Inst2(dis.ILEA, dis.MP(mpOff), dis.FP(slot)))
	fl.valueMap[g] = slot
	return slot
}

// materialize ensures a value is in a frame slot and returns its offset.
// For constants, this emits the load instruction. Globals are handled by slotOf.
func (fl *funcLowerer) materialize(v ssa.Value) int32 {
	// If it's a constant, we need to emit code to load it
	if c, ok := v.(*ssa.Const); ok {
		// Interface constant (nil interface): allocate 2 WORDs
		if _, ok := v.Type().Underlying().(*types.Interface); ok {
			off := fl.frame.AllocWord("")
			fl.frame.AllocWord("")
			// Explicitly zero both tag and value — non-pointer frame
			// slots are NOT guaranteed to be zero-initialized by the VM.
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(off)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(off+int32(dis.IBY2WD))))
			return off
		}
		dt := GoTypeToDis(v.Type())
		var off int32
		if dt.IsPtr {
			off = fl.frame.AllocPointer("")
		} else {
			off = fl.frame.AllocWord("")
		}

		if c.Value == nil {
			// nil/zero - slot is already zeroed
			return off
		}

		switch c.Value.Kind() {
		case constant.Int:
			val, _ := constant.Int64Val(c.Value)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(int32(val)), dis.FP(off)))
		case constant.Bool:
			if constant.BoolVal(c.Value) {
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(off)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(off)))
			}
		case constant.String:
			s := constant.StringVal(c.Value)
			mpOff := fl.comp.AllocString(s)
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mpOff), dis.FP(off)))
		}
		return off
	}
	return fl.slotOf(v)
}

func (fl *funcLowerer) operandOf(v ssa.Value) dis.Operand {
	// Check if it's a constant
	if c, ok := v.(*ssa.Const); ok {
		return fl.constOperand(c)
	}
	// Otherwise it's in a frame slot (slotOf handles globals via loadGlobalAddr)
	return dis.FP(fl.slotOf(v))
}

func (fl *funcLowerer) constOperand(c *ssa.Const) dis.Operand {
	if c.Value == nil {
		// nil / zero value
		return dis.Imm(0)
	}
	switch c.Value.Kind() {
	case constant.Int:
		val, _ := constant.Int64Val(c.Value)
		if val >= -0x20000000 && val <= 0x1FFFFFFF {
			return dis.Imm(int32(val))
		}
		// Large constant: must be stored in a frame slot
		off := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(int32(val)), dis.FP(off)))
		return dis.FP(off)
	case constant.Bool:
		if constant.BoolVal(c.Value) {
			return dis.Imm(1)
		}
		return dis.Imm(0)
	case constant.String:
		// String constants need to be in the data section
		s := constant.StringVal(c.Value)
		mpOff := fl.comp.AllocString(s)
		off := fl.frame.AllocPointer("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mpOff), dis.FP(off)))
		return dis.FP(off)
	default:
		return dis.Imm(0)
	}
}

func (fl *funcLowerer) arithOp(intOp, floatOp, stringOp dis.Op, basic *types.Basic) dis.Op {
	if basic == nil {
		return intOp
	}
	if isFloat(basic) {
		return floatOp
	}
	if basic.Kind() == types.String && stringOp != 0 {
		return stringOp
	}
	return intOp
}

func (fl *funcLowerer) compBranchOp(op token.Token, basic *types.Basic) dis.Op {
	isF := basic != nil && isFloat(basic)
	isC := basic != nil && basic.Kind() == types.String

	switch op {
	case token.EQL:
		if isF {
			return dis.IBEQF
		}
		if isC {
			return dis.IBEQC
		}
		return dis.IBEQW
	case token.NEQ:
		if isF {
			return dis.IBNEF
		}
		if isC {
			return dis.IBNEC
		}
		return dis.IBNEW
	case token.LSS:
		if isF {
			return dis.IBLTF
		}
		if isC {
			return dis.IBLTC
		}
		return dis.IBLTW
	case token.LEQ:
		if isF {
			return dis.IBLEF
		}
		if isC {
			return dis.IBLEC
		}
		return dis.IBLEW
	case token.GTR:
		if isF {
			return dis.IBGTF
		}
		if isC {
			return dis.IBGTC
		}
		return dis.IBGTW
	case token.GEQ:
		if isF {
			return dis.IBGEF
		}
		if isC {
			return dis.IBGEC
		}
		return dis.IBGEW
	}
	return dis.IBEQW
}

func isFloat(basic *types.Basic) bool {
	return basic.Info()&types.IsFloat != 0
}
