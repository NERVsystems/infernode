package compiler

// stdlib_lower.go — additional stdlib lowering implementations for packages
// added in stdlib_packages.go.

import (
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/NERVsystems/infernode/tools/godis/dis"
)

// ============================================================
// crypto/sha512 package
// ============================================================

func (fl *funcLowerer) lowerCryptoSHA512Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Sum512":
		// sha512.Sum512(data) → nil slice stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "New":
		// sha512.New() → 0 stub handle
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/subtle package
// ============================================================

func (fl *funcLowerer) lowerCryptoSubtleCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ConstantTimeCompare":
		// subtle.ConstantTimeCompare(x, y) → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "ConstantTimeSelect":
		// subtle.ConstantTimeSelect(v, x, y) → x when v=1, y when v=0
		vSlot := fl.materialize(instr.Call.Args[0])
		xSlot := fl.materialize(instr.Call.Args[1])
		ySlot := fl.materialize(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ySlot), dis.FP(dst)))
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(vSlot), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(xSlot), dis.FP(dst)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "ConstantTimeEq":
		// subtle.ConstantTimeEq(x, y) → 1 if x==y, else 0
		xSlot := fl.materialize(instr.Call.Args[0])
		ySlot := fl.materialize(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(xSlot), dis.FP(ySlot), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "XORBytes":
		// subtle.XORBytes(dst, x, y) → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/gob package
// ============================================================

func (fl *funcLowerer) lowerEncodingGobCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch {
	case name == "NewEncoder" || name == "NewDecoder":
		// gob.NewEncoder/NewDecoder → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case name == "Register" || name == "RegisterName":
		// gob.Register/RegisterName — no-op
		return true, nil
	case strings.Contains(name, "Encode") || strings.Contains(name, "Decode"):
		// Encoder.Encode / Decoder.Decode → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/ascii85 package
// ============================================================

func (fl *funcLowerer) lowerEncodingASCII85Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Encode":
		// ascii85.Encode(dst, src) → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "MaxEncodedLen":
		// ascii85.MaxEncodedLen(n) → n*5/4+4 (approximation)
		nSlot := fl.materialize(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.NewInst(dis.IMULW, dis.Imm(2), dis.FP(nSlot), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// container/list package
// ============================================================

func (fl *funcLowerer) lowerContainerListCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch {
	case name == "New":
		// list.New() → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "PushBack") || strings.Contains(name, "PushFront"):
		// List.PushBack/PushFront → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Len"):
		// List.Len() → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Front"):
		// List.Front() → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Remove"):
		// List.Remove(e) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Next"):
		// Element.Next() → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// container/ring package
// ============================================================

func (fl *funcLowerer) lowerContainerRingCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch {
	case name == "New":
		// ring.New(n) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Len"):
		// Ring.Len() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Next"):
		// Ring.Next() → nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// container/heap package
// ============================================================

func (fl *funcLowerer) lowerContainerHeapCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Init":
		// heap.Init(h) — no-op
		return true, nil
	case "Push":
		// heap.Push(h, x) — no-op
		return true, nil
	case "Pop":
		// heap.Pop(h) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// image package
// ============================================================

func (fl *funcLowerer) lowerImageCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Pt":
		// image.Pt(X, Y) → Point{X, Y}
		xSlot := fl.materialize(instr.Call.Args[0])
		ySlot := fl.materialize(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(xSlot), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ySlot), dis.FP(dst+iby2wd)))
		return true, nil
	case "Rect":
		// image.Rect(x0, y0, x1, y1) → Rectangle{Min{x0,y0}, Max{x1,y1}}
		x0Slot := fl.materialize(instr.Call.Args[0])
		y0Slot := fl.materialize(instr.Call.Args[1])
		x1Slot := fl.materialize(instr.Call.Args[2])
		y1Slot := fl.materialize(instr.Call.Args[3])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(x0Slot), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(y0Slot), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(x1Slot), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(y1Slot), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "NewRGBA":
		// image.NewRGBA(r) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// image/color package
// ============================================================

func (fl *funcLowerer) lowerImageColorCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// No function calls in image/color, just types and vars
	return false, nil
}

// ============================================================
// image/png and image/jpeg codecs
// ============================================================

func (fl *funcLowerer) lowerImageCodecCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Encode":
		// png.Encode / jpeg.Encode → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Decode":
		// png.Decode / jpeg.Decode → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// debug/buildinfo package
// ============================================================

func (fl *funcLowerer) lowerDebugBuildInfoCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ReadFile":
		// buildinfo.ReadFile(name) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// go/* packages (ast, token, parser, format)
// ============================================================

func (fl *funcLowerer) lowerGoToolCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch {
	case name == "NewFileSet":
		// token.NewFileSet() → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case name == "ParseFile":
		// parser.ParseFile(...) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case name == "Source":
		// format.Source(src) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net/http/cookiejar package
// ============================================================

func (fl *funcLowerer) lowerNetHTTPCookiejarCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "New":
		// cookiejar.New(o) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net/http/pprof package
// ============================================================

func (fl *funcLowerer) lowerNetHTTPPprofCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Index":
		// pprof.Index(w, r) — no-op
		return true, nil
	}
	return false, nil
}

// ============================================================
// os/user package
// ============================================================

func (fl *funcLowerer) lowerOsUserCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Current", "Lookup":
		// user.Current/Lookup → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// regexp/syntax package
// ============================================================

func (fl *funcLowerer) lowerRegexpSyntaxCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// Constants only, no function calls to lower
	return false, nil
}

// ============================================================
// runtime/debug package
// ============================================================

func (fl *funcLowerer) lowerRuntimeDebugCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Stack":
		// debug.Stack() → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "PrintStack", "FreeOSMemory":
		// no-op
		return true, nil
	case "SetGCPercent":
		// debug.SetGCPercent(percent) → 100 (previous value)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(100), dis.FP(dst)))
		return true, nil
	case "ReadBuildInfo":
		// debug.ReadBuildInfo() → (nil, false) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// runtime/pprof package
// ============================================================

func (fl *funcLowerer) lowerRuntimePprofCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "StartCPUProfile":
		// pprof.StartCPUProfile(w) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "StopCPUProfile":
		// no-op
		return true, nil
	}
	return false, nil
}

// ============================================================
// text/scanner package
// ============================================================

func (fl *funcLowerer) lowerTextScannerCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// No function calls to lower, just types
	return false, nil
}

// ============================================================
// text/tabwriter package
// ============================================================

func (fl *funcLowerer) lowerTextTabwriterCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewWriter":
		// tabwriter.NewWriter(...) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}
