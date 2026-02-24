package compiler

// stdlib_lower3.go — lowering implementations for additional stdlib packages.

import (
	"golang.org/x/tools/go/ssa"

	"github.com/NERVsystems/infernode/tools/godis/dis"
)

func (fl *funcLowerer) lowerEncodingBase32Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewEncoding":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "EncodeToString":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "DecodeString":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Encode":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	case "Decode":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "EncodedLen", "DecodedLen":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "WithPadding":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoDESCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewCipher", "NewTripleDESCipher":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoRC4Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewCipher":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "XORKeyStream":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	case "Reset":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerSyscallCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Getenv":
		// Getenv(key) → ("", false)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Getpid":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "Getuid", "Getgid":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Exit":
		// Exit(code) — no-op stub (could emit IRAISE)
		return true, nil
	case "Open":
		// Open(path, mode, perm) → (-1, ENOSYS)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Close":
		// Close(fd) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Read", "Write":
		// Read/Write(fd, p) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerMathCmplxCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Abs", "Phase":
		// Returns float64
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "IsNaN", "IsInf":
		// Returns bool
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Sqrt", "Exp", "Log", "Sin", "Cos", "Tan",
		"Asin", "Acos", "Atan", "Sinh", "Cosh", "Tanh",
		"Conj", "Log10", "Log2", "Pow":
		// Returns complex128 (2 float64s = 2 words on Dis)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Polar":
		// Returns (r, theta float64)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Rect", "Inf", "NaN":
		// Returns complex128
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerNetSMTPCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "SendMail":
		// SendMail(...) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "PlainAuth", "CRAMMD5Auth":
		// Returns Auth (interface) stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Dial":
		// Dial(addr) → (*Client, error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Close", "Mail", "Rcpt", "Quit":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerNetRPCCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Dial", "DialHTTP":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "NewServer":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Register":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Call":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Close":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerTextTemplateParseCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// text/template/parse only has types and constants, no lowerable functions
	return false, nil
}

func (fl *funcLowerer) lowerEncodingASN1Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Marshal", "MarshalWithParams":
		// Marshal(val) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Unmarshal", "UnmarshalWithParams":
		// Unmarshal(b, val) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoX509PkixCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// Only types, no functions to lower
	switch callee.Name() {
	case "String":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoDSACall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// Only types and constants, no lowerable functions
	return false, nil
}

func (fl *funcLowerer) lowerNetRPCJSONRPCCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Dial":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch name {
	case "Available", "Size", "HashFunc":
		// Hash method calls
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerHashAdler32Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "New":
		// New() → hash.Hash32 (interface)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Checksum":
		// Checksum(data) → uint32
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerHashCRC64Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "New":
		// New(tab) → hash.Hash64 (interface)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "MakeTable":
		// MakeTable(poly) → *Table
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Checksum":
		// Checksum(data, tab) → uint64
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerEncodingCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	// encoding package only has interfaces (BinaryMarshaler, TextMarshaler, etc.)
	// No package-level functions to lower
	return false, nil
}

func (fl *funcLowerer) lowerGoConstantCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "MakeBool", "MakeString", "MakeInt64", "MakeFloat64":
		// Make*(x) → Value (interface)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "BoolVal":
		// BoolVal(x) → bool
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "StringVal":
		// StringVal(x) → string
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Int64Val", "Float64Val":
		// Int64Val/Float64Val(x) → (val, exact bool)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Compare":
		// Compare(x, op, y) → bool
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerGoScannerCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Error":
		// Error.Error() → string
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Len":
		// ErrorList.Len() → int
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerRuntimeTraceCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Start":
		// Start(w) → error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Stop":
		// Stop() — no-op
		return true, nil
	case "IsEnabled":
		// IsEnabled() → bool (always false in Dis VM)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

func (fl *funcLowerer) lowerCryptoECDHCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "P256", "P384", "P521", "X25519":
		// Curve factory → Curve struct
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "PublicKey":
		// PrivateKey.PublicKey() → *PublicKey
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Bytes":
		// PrivateKey.Bytes() or PublicKey.Bytes() → []byte
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "ECDH":
		// PrivateKey.ECDH(remote) → ([]byte, error)
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

func (fl *funcLowerer) lowerMathRandV2Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Int", "IntN", "N":
		// Returns int
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Int64", "Int64N":
		// Returns int64
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Uint32":
		// Returns uint32
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Uint64":
		// Returns uint64
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Float32":
		// Returns float32
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Float64":
		// Returns float64
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Shuffle":
		// Shuffle(n, swap) — no-op
		return true, nil
	}
	return false, nil
}

// ============================================================
// testing package
// ============================================================

func (fl *funcLowerer) lowerTestingCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()

	// Package-level functions
	if callee.Signature.Recv() == nil {
		switch name {
		case "Short", "Verbose":
			// testing.Short/Verbose() → false
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	}

	// Method calls on T, B, M
	switch name {
	// Logging/error methods — no-op (test harness not applicable in Dis)
	case "Error", "Errorf", "Fatal", "Fatalf", "Log", "Logf", "Skip", "Skipf":
		return true, nil
	// Control flow methods — no-op
	case "Fail", "FailNow", "SkipNow", "Helper", "Parallel", "Cleanup",
		"Setenv", "ResetTimer", "StartTimer", "StopTimer", "ReportAllocs",
		"SetBytes", "ReportMetric":
		return true, nil
	// Bool-returning methods
	case "Failed", "Skipped":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	// String-returning methods
	case "Name", "TempDir":
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	// Run returns bool
	case "Run":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	// Deadline returns (time, bool)
	case "Deadline":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}
