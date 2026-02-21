package compiler

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"

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
}

// lower compiles the function to Dis instructions.
func (fl *funcLowerer) lower() (*lowerResult, error) {
	if len(fl.fn.Blocks) == 0 {
		return nil, fmt.Errorf("function %s has no blocks", fl.fn.Name())
	}

	// Pre-allocate frame slots for all SSA values that need them
	fl.allocateSlots()

	// First pass: emit instructions for each basic block
	for _, block := range fl.fn.Blocks {
		fl.blockPC[block] = int32(len(fl.insts))
		if err := fl.lowerBlock(block); err != nil {
			return nil, fmt.Errorf("block %s: %w", block.Comment, err)
		}
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
	}, nil
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

	// Parameters
	for _, p := range fl.fn.Params {
		if st, ok := p.Type().Underlying().(*types.Struct); ok {
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

	// Free variables (closures - future phase)

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
				// Skip Alloc and FieldAddr: their pointer slots are allocated
				// as non-pointer (LEA produces stack/MP address, not heap pointer)
				switch instr.(type) {
				case *ssa.Alloc, *ssa.FieldAddr:
					continue
				}
				// Struct values need consecutive slots for each field
				if st, ok := v.Type().Underlying().(*types.Struct); ok {
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
	case *ssa.Convert:
		return fl.lowerConvert(instr)
	case *ssa.ChangeType:
		return fl.lowerChangeType(instr)
	case *ssa.MakeInterface:
		// TODO: interface support
		return nil
	case *ssa.DebugRef:
		return nil // ignore debug info
	default:
		return fmt.Errorf("unsupported instruction: %T (%v)", instr, instr)
	}
}

func (fl *funcLowerer) lowerAlloc(instr *ssa.Alloc) error {
	if instr.Heap {
		// TODO: heap allocation via INEW
		return fmt.Errorf("heap allocation not yet supported")
	}
	// Stack allocation: the SSA value is a pointer (*T).
	// We allocate frame slots for the pointed-to value(s) and use LEA
	// to make the pointer slot point to the base.
	// The pointer slot is NOT a GC pointer because it points to a stack frame,
	// not the heap. The GC manages stack frames separately.
	elemType := instr.Type().(*types.Pointer).Elem()

	var baseSlot int32
	if st, ok := elemType.Underlying().(*types.Struct); ok {
		// Struct: allocate one slot per field
		baseSlot = fl.allocStructFields(st, instr.Name())
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
		fl.emit(dis.NewInst(fl.arithOp(dis.IDIVW, dis.IDIVF, 0, basic), mid, src, dis.FP(dst)))
	case token.REM:
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
		// Check for struct dereference (multi-word copy)
		if st, ok := instr.Type().Underlying().(*types.Struct); ok {
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
		} else {
			dt := GoTypeToDis(instr.Type())
			if dt.IsPtr {
				fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(addrOff, 0), dis.FP(dst)))
			} else {
				fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrOff, 0), dis.FP(dst)))
			}
		}
	case token.ARROW: // channel receive <-ch
		// TODO: IRECV
		return fmt.Errorf("channel receive not yet supported")
	default:
		return fmt.Errorf("unsupported unary op: %v", instr.Op)
	}
	return nil
}

func (fl *funcLowerer) lowerCall(instr *ssa.Call) error {
	call := instr.Call

	// Check if this is a call to a built-in like println
	if builtin, ok := call.Value.(*ssa.Builtin); ok {
		return fl.lowerBuiltinCall(instr, builtin)
	}

	// Check if this is a call to a local function
	if callee, ok := call.Value.(*ssa.Function); ok {
		return fl.lowerDirectCall(instr, callee)
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
		// TODO
		return fmt.Errorf("cap not yet supported")
	case "append":
		// TODO
		return fmt.Errorf("append not yet supported")
	default:
		return fmt.Errorf("unsupported builtin: %s", builtin.Name())
	}
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

func (fl *funcLowerer) lowerDirectCall(instr *ssa.Call, callee *ssa.Function) error {
	call := instr.Call

	// Materialize all arguments first (may emit instructions for constants)
	type argInfo struct {
		off   int32
		isPtr bool
		st    *types.Struct // non-nil if this is a struct value argument
	}
	var args []argInfo
	for _, arg := range call.Args {
		off := fl.materialize(arg)
		dt := GoTypeToDis(arg.Type())
		var st *types.Struct
		if s, ok := arg.Type().Underlying().(*types.Struct); ok {
			st = s
		}
		args = append(args, argInfo{off, dt.IsPtr, st})
	}

	// Allocate callee frame slot (NOT a GC pointer - stack allocated, stale after return)
	callFrame := fl.frame.AllocWord("")

	// IFRAME $0, callFrame(fp) — TD ID is placeholder, patched by compiler
	iframeIdx := len(fl.insts)
	fl.emit(dis.Inst2(dis.IFRAME, dis.Imm(0), dis.FP(callFrame)))

	// Set arguments in callee frame (args start at MaxTemp = 64)
	calleeOff := int32(dis.MaxTemp)
	for _, arg := range args {
		if arg.st != nil {
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
			calleeOff += int32(dis.IBY2WD)
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arg.off), dis.FPInd(callFrame, calleeOff)))
			calleeOff += int32(dis.IBY2WD)
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

func (fl *funcLowerer) lowerReturn(instr *ssa.Return) error {
	if len(instr.Results) > 0 {
		// Store return value through REGRET pointer: 0(32(fp))
		// REGRET is at offset REGRET*IBY2WD = 32 in the frame header
		regretOff := int32(dis.REGRET * dis.IBY2WD)
		off := fl.materialize(instr.Results[0])
		dt := GoTypeToDis(instr.Results[0].Type())
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(off), dis.FPInd(regretOff, 0)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(off), dis.FPInd(regretOff, 0)))
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
		src := fl.operandOf(phi.Edges[edgeIdx])
		dt := GoTypeToDis(phi.Type())
		if dt.IsPtr {
			fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(dst)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
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
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(valOff), dis.FPInd(addrOff, 0)))
	}
	return nil
}

func (fl *funcLowerer) lowerFieldAddr(instr *ssa.FieldAddr) error {
	// FieldAddr produces a pointer to a field within a struct.
	// instr.X is the struct pointer, instr.Field is the field index.
	base, ok := fl.allocBase[instr.X]
	if !ok {
		return fmt.Errorf("FieldAddr on non-stack-allocated struct (not yet supported)")
	}

	// Compute the field's frame offset
	structType := instr.X.Type().(*types.Pointer).Elem().Underlying().(*types.Struct)
	fieldOff := int32(0)
	for i := 0; i < instr.Field; i++ {
		dt := GoTypeToDis(structType.Field(i).Type())
		fieldOff += dt.Size
	}

	fieldSlot := base + fieldOff

	// Allocate a non-pointer slot for the field address (points to stack, not heap)
	ptrSlot := fl.frame.AllocWord("faddr:" + instr.Name())
	fl.valueMap[instr] = ptrSlot
	fl.emit(dis.Inst2(dis.ILEA, dis.FP(fieldSlot), dis.FP(ptrSlot)))
	return nil
}

func (fl *funcLowerer) lowerIndexAddr(instr *ssa.IndexAddr) error {
	return fmt.Errorf("index address not yet supported")
}

func (fl *funcLowerer) lowerConvert(instr *ssa.Convert) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)
	// For now, just copy the value (works for same-size integer conversions)
	fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
	return nil
}

func (fl *funcLowerer) lowerChangeType(instr *ssa.ChangeType) error {
	dst := fl.slotOf(instr)
	src := fl.operandOf(instr.X)
	fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
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
