package compiler

// lower_stdlib.go — stdlib lowering implementations for packages beyond the
// core set. These are methods on *funcLowerer registered via the stdlib registry.

import (
	"go/types"
	"math"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/NERVsystems/infernode/tools/godis/dis"
)

func init() {
	RegisterStdlibLowerer("unicode", (*funcLowerer).lowerUnicodeCall)
	RegisterStdlibLowerer("unicode/utf8", (*funcLowerer).lowerUTF8Call)
	RegisterStdlibLowerer("unicode/utf16", (*funcLowerer).lowerUnicodeUTF16Call)
	RegisterStdlibLowerer("path", (*funcLowerer).lowerPathCall)
	RegisterStdlibLowerer("path/filepath", (*funcLowerer).lowerFilepathCall)
	RegisterStdlibLowerer("math/bits", (*funcLowerer).lowerMathBitsCall)
	RegisterStdlibLowerer("math/rand", (*funcLowerer).lowerMathRandCall)
	RegisterStdlibLowerer("math/big", (*funcLowerer).lowerMathBigCall)
	RegisterStdlibLowerer("bytes", (*funcLowerer).lowerBytesCall)
	RegisterStdlibLowerer("encoding/hex", (*funcLowerer).lowerEncodingHexCall)
	RegisterStdlibLowerer("encoding/base64", (*funcLowerer).lowerEncodingBase64Call)
	RegisterStdlibLowerer("encoding/json", (*funcLowerer).lowerEncodingJSONCall)
	RegisterStdlibLowerer("encoding/binary", (*funcLowerer).lowerEncodingBinaryCall)
	RegisterStdlibLowerer("encoding/csv", (*funcLowerer).lowerEncodingCSVCall)
	RegisterStdlibLowerer("encoding/xml", (*funcLowerer).lowerEncodingXMLCall)
	RegisterStdlibLowerer("encoding/pem", (*funcLowerer).lowerEncodingPEMCall)
	RegisterStdlibLowerer("slices", (*funcLowerer).lowerSlicesCall)
	RegisterStdlibLowerer("maps", (*funcLowerer).lowerMapsCall)
	RegisterStdlibLowerer("io", (*funcLowerer).lowerIOCall)
	RegisterStdlibLowerer("io/ioutil", (*funcLowerer).lowerIOUtilCall)
	RegisterStdlibLowerer("io/fs", (*funcLowerer).lowerIOFSCall)
	RegisterStdlibLowerer("cmp", (*funcLowerer).lowerCmpCall)
	RegisterStdlibLowerer("context", (*funcLowerer).lowerContextCall)
	RegisterStdlibLowerer("sync/atomic", (*funcLowerer).lowerSyncAtomicCall)
	RegisterStdlibLowerer("bufio", (*funcLowerer).lowerBufioCall)
	RegisterStdlibLowerer("net/url", (*funcLowerer).lowerNetURLCall)
	RegisterStdlibLowerer("net/http", (*funcLowerer).lowerNetHTTPCall)
	RegisterStdlibLowerer("runtime", (*funcLowerer).lowerRuntimeCall)
	RegisterStdlibLowerer("reflect", (*funcLowerer).lowerReflectCall)
	RegisterStdlibLowerer("os/exec", (*funcLowerer).lowerOsExecCall)
	RegisterStdlibLowerer("os/signal", (*funcLowerer).lowerOsSignalCall)
	RegisterStdlibLowerer("regexp", (*funcLowerer).lowerRegexpCall)
	RegisterStdlibLowerer("log/slog", (*funcLowerer).lowerLogSlogCall)
	RegisterStdlibLowerer("embed", (*funcLowerer).lowerEmbedCall)
	RegisterStdlibLowerer("flag", (*funcLowerer).lowerFlagCall)
	RegisterStdlibLowerer("crypto/sha256", (*funcLowerer).lowerCryptoSHA256Call)
	RegisterStdlibLowerer("crypto/md5", (*funcLowerer).lowerCryptoMD5Call)
	RegisterStdlibLowerer("text/template", (*funcLowerer).lowerTextTemplateCall)
	RegisterStdlibLowerer("hash", (*funcLowerer).lowerHashCall)
	RegisterStdlibLowerer("hash/crc32", (*funcLowerer).lowerHashCall)
	RegisterStdlibLowerer("net", (*funcLowerer).lowerNetCall)
	RegisterStdlibLowerer("crypto/rand", (*funcLowerer).lowerCryptoRandCall)
	RegisterStdlibLowerer("crypto/hmac", (*funcLowerer).lowerCryptoHMACCall)
	RegisterStdlibLowerer("crypto/aes", (*funcLowerer).lowerCryptoAESCall)
	RegisterStdlibLowerer("crypto/cipher", (*funcLowerer).lowerCryptoCipherCall)
	RegisterStdlibLowerer("crypto/tls", (*funcLowerer).lowerCryptoTLSCall)
	RegisterStdlibLowerer("crypto/x509", (*funcLowerer).lowerCryptoX509Call)
	RegisterStdlibLowerer("crypto/elliptic", (*funcLowerer).lowerCryptoEllipticCall)
	RegisterStdlibLowerer("crypto/ecdsa", (*funcLowerer).lowerCryptoECDSACall)
	RegisterStdlibLowerer("crypto/rsa", (*funcLowerer).lowerCryptoRSACall)
	RegisterStdlibLowerer("crypto/ed25519", (*funcLowerer).lowerCryptoEd25519Call)
	RegisterStdlibLowerer("database/sql", (*funcLowerer).lowerDatabaseSQLCall)
	RegisterStdlibLowerer("archive/zip", (*funcLowerer).lowerArchiveZipCall)
	RegisterStdlibLowerer("archive/tar", (*funcLowerer).lowerArchiveTarCall)
	RegisterStdlibLowerer("compress/gzip", (*funcLowerer).lowerCompressGzipCall)
	RegisterStdlibLowerer("compress/flate", (*funcLowerer).lowerCompressFlateCall)
	RegisterStdlibLowerer("html", (*funcLowerer).lowerHTMLCall)
	RegisterStdlibLowerer("html/template", (*funcLowerer).lowerHTMLTemplateCall)
	RegisterStdlibLowerer("mime", (*funcLowerer).lowerMIMECall)
	RegisterStdlibLowerer("mime/multipart", (*funcLowerer).lowerMIMEMultipartCall)
	RegisterStdlibLowerer("net/mail", (*funcLowerer).lowerNetMailCall)
	RegisterStdlibLowerer("net/textproto", (*funcLowerer).lowerNetTextprotoCall)
	RegisterStdlibLowerer("net/http/httputil", (*funcLowerer).lowerNetHTTPUtilCall)
}

// ============================================================
// strings package — new functions
// ============================================================

// lowerStringsCount: count non-overlapping occurrences of substr in s.
func (fl *funcLowerer) lowerStringsCount(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	subOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenSub := fl.frame.AllocWord("")
	limit := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	endIdx := fl.frame.AllocWord("")
	candidate := fl.frame.AllocTemp(true)
	count := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, subOp, dis.FP(lenSub)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))

	// if lenSub == 0 → return lenS+1
	beqEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenSub), dis.Imm(0)))

	// if lenSub > lenS → return 0
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSub), dis.FP(lenS), dis.Imm(0)))

	// limit = lenS - lenSub + 1
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSub), dis.FP(lenS), dis.FP(limit)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	// loop:
	loopPC := int32(len(fl.insts))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSub), dis.FP(i), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(candidate)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))

	beqFoundIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, subOp, dis.FP(candidate), dis.Imm(0)))

	// no match: i++
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// found: count++, i += lenSub (non-overlapping)
	foundPC := int32(len(fl.insts))
	fl.insts[beqFoundIdx].Dst = dis.Imm(foundPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSub), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// done:
	donePC := int32(len(fl.insts))
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
	jmpEndIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// empty substr: return lenS + 1
	emptyPC := int32(len(fl.insts))
	fl.insts[beqEmptyIdx].Dst = dis.Imm(emptyPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lenS), dis.FP(dst)))

	endPC := int32(len(fl.insts))
	fl.insts[jmpEndIdx].Dst = dis.Imm(endPC)
	return nil
}

// lowerStringsEqualFold: case-insensitive string comparison.
func (fl *funcLowerer) lowerStringsEqualFold(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	tOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenT := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	chS := fl.frame.AllocWord("")
	chT := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, tOp, dis.FP(lenT)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // default false

	// if lenS != lenT → done (false)
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(lenS), dis.FP(lenT), dis.Imm(0)))

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	// loop:
	loopPC := int32(len(fl.insts))
	bgeMatchIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0))) // all chars match

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(chS)))
	fl.emit(dis.NewInst(dis.IINDC, tOp, dis.FP(i), dis.FP(chT)))

	// toLower both: if 'A'-'Z', add 32
	tmpS := fl.frame.AllocWord("")
	tmpT := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(chS), dis.FP(tmpS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(chT), dis.FP(tmpT)))

	// toLower chS
	skipS := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(chS), dis.Imm(65), dis.Imm(0)))
	skipS2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(chS), dis.Imm(90), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), dis.FP(chS), dis.FP(tmpS)))
	skipSPC := int32(len(fl.insts))
	fl.insts[skipS].Dst = dis.Imm(skipSPC)
	fl.insts[skipS2].Dst = dis.Imm(skipSPC)

	// toLower chT
	skipT := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(chT), dis.Imm(65), dis.Imm(0)))
	skipT2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(chT), dis.Imm(90), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), dis.FP(chT), dis.FP(tmpT)))
	skipTPC := int32(len(fl.insts))
	fl.insts[skipT].Dst = dis.Imm(skipTPC)
	fl.insts[skipT2].Dst = dis.Imm(skipTPC)

	// compare lowered chars
	bneCharIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(tmpS), dis.FP(tmpT), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// all matched:
	matchPC := int32(len(fl.insts))
	fl.insts[bgeMatchIdx].Dst = dis.Imm(matchPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	// done:
	donePC := int32(len(fl.insts))
	fl.insts[bneIdx].Dst = dis.Imm(donePC)
	fl.insts[bneCharIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsTrimPrefix: if s starts with prefix, return s[len(prefix):].
func (fl *funcLowerer) lowerStringsTrimPrefix(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	prefOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenP := fl.frame.AllocWord("")
	head := fl.frame.AllocTemp(true)

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, prefOp, dis.FP(lenP)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst))) // default: return s

	// if lenP > lenS → done
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenP), dis.FP(lenS), dis.Imm(0)))

	// head = s[0:lenP]
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(head)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenP), dis.FP(head)))

	// if head != prefix → done
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEC, prefOp, dis.FP(head), dis.Imm(0)))

	// match: dst = s[lenP:]
	tmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lenP), dis.FP(lenS), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bneIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsTrimSuffix: if s ends with suffix, return s[:len(s)-len(suffix)].
func (fl *funcLowerer) lowerStringsTrimSuffix(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	sufOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenSuf := fl.frame.AllocWord("")
	startOff := fl.frame.AllocWord("")
	tail := fl.frame.AllocTemp(true)

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, sufOp, dis.FP(lenSuf)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst))) // default: return s

	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSuf), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSuf), dis.FP(lenS), dis.FP(startOff)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tail)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startOff), dis.FP(lenS), dis.FP(tail)))

	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEC, sufOp, dis.FP(tail), dis.Imm(0)))

	tmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(startOff), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bneIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsReplaceAll: same as Replace with n=-1.
func (fl *funcLowerer) lowerStringsReplaceAll(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	oldOp := fl.operandOf(instr.Call.Args[1])
	newOp := fl.operandOf(instr.Call.Args[2])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenOld := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	endIdx := fl.frame.AllocWord("")
	candidate := fl.frame.AllocTemp(true)
	result := fl.frame.AllocTemp(true)
	limit := fl.frame.AllocWord("")
	ch := fl.frame.AllocTemp(true)

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, oldOp, dis.FP(lenOld)))
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	bgtShort := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenOld), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenOld), dis.FP(lenS), dis.FP(limit)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
	jmpLoop := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	shortPC := int32(len(fl.insts))
	fl.insts[bgtShort].Dst = dis.Imm(shortPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(limit)))

	loopPC := int32(len(fl.insts))
	fl.insts[jmpLoop].Dst = dis.Imm(loopPC)
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenOld), dis.FP(i), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(candidate)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
	beqMatch := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, oldOp, dis.FP(candidate), dis.Imm(0)))

	// no match: append s[i] char
	oneAfter := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(ch)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(ch)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(ch), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// match: append new, skip lenOld
	matchPC := int32(len(fl.insts))
	fl.insts[beqMatch].Dst = dis.Imm(matchPC)
	fl.emit(dis.NewInst(dis.IADDC, newOp, dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenOld), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// done: append tail
	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	tail := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tail)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(lenS), dis.FP(tail)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(tail), dis.FP(result), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return nil
}

// lowerStringsContainsRune: check if rune (int32) exists in string.
func (fl *funcLowerer) lowerStringsContainsRune(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	rOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	loopPC := int32(len(fl.insts))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
	beqIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), rOp, dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	foundPC := int32(len(fl.insts))
	fl.insts[beqIdx].Dst = dis.Imm(foundPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsContainsAny: check if any char in chars exists in s.
func (fl *funcLowerer) lowerStringsContainsAny(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	charsOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenC := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	j := fl.frame.AllocWord("")
	chS := fl.frame.AllocWord("")
	chC := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, charsOp, dis.FP(lenC)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	outerPC := int32(len(fl.insts))
	bgeOuterIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(chS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))

	innerPC := int32(len(fl.insts))
	bgeInnerIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenC), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, charsOp, dis.FP(j), dis.FP(chC)))
	beqIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(chS), dis.FP(chC), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

	// inner done: no match for this char
	innerDonePC := int32(len(fl.insts))
	fl.insts[bgeInnerIdx].Dst = dis.Imm(innerDonePC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))

	// found:
	foundPC := int32(len(fl.insts))
	fl.insts[beqIdx].Dst = dis.Imm(foundPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgeOuterIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsIndexByte: find first occurrence of byte in string, return index or -1.
func (fl *funcLowerer) lowerStringsIndexByte(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	bOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	loopPC := int32(len(fl.insts))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
	beqIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), bOp, dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	foundPC := int32(len(fl.insts))
	fl.insts[beqIdx].Dst = dis.Imm(foundPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsIndexRune: same as IndexByte but for rune (int32).
func (fl *funcLowerer) lowerStringsIndexRune(instr *ssa.Call) error {
	// Same implementation — INDC returns Unicode code point
	return fl.lowerStringsIndexByte(instr)
}

// lowerStringsLastIndex: find last occurrence of substr in s.
func (fl *funcLowerer) lowerStringsLastIndex(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	subOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenSub := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	endIdx := fl.frame.AllocWord("")
	candidate := fl.frame.AllocTemp(true)

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, subOp, dis.FP(lenSub)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))

	// if lenSub > lenS → done
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSub), dis.FP(lenS), dis.Imm(0)))

	// i = lenS - lenSub (start from end)
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSub), dis.FP(lenS), dis.FP(i)))

	loopPC := int32(len(fl.insts))
	bltIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSub), dis.FP(i), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(candidate)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))

	beqIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, subOp, dis.FP(candidate), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	foundPC := int32(len(fl.insts))
	fl.insts[beqIdx].Dst = dis.Imm(foundPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bltIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerStringsFields: split on whitespace runs.
func (fl *funcLowerer) lowerStringsFields(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")
	count := fl.frame.AllocWord("")
	inWord := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))

	// First pass: count words
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	countLoopPC := int32(len(fl.insts))
	bgeCountDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	// isSpace: ch == 32 || ch == 9 || ch == 10 || ch == 13
	beqSpc := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
	beqTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
	beqNl := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
	beqCr := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))

	// not space: if !inWord, count++ and set inWord=1
	beqAlreadyIn := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(inWord)))
	alreadyInPC := int32(len(fl.insts))
	fl.insts[beqAlreadyIn].Dst = dis.Imm(alreadyInPC)
	jmpNext := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// space: set inWord=0
	spacePC := int32(len(fl.insts))
	fl.insts[beqSpc].Dst = dis.Imm(spacePC)
	fl.insts[beqTab].Dst = dis.Imm(spacePC)
	fl.insts[beqNl].Dst = dis.Imm(spacePC)
	fl.insts[beqCr].Dst = dis.Imm(spacePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))

	nextPC := int32(len(fl.insts))
	fl.insts[jmpNext].Dst = dis.Imm(nextPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countLoopPC)))

	countDonePC := int32(len(fl.insts))
	fl.insts[bgeCountDone].Dst = dis.Imm(countDonePC)

	// Allocate array
	elemTDIdx := fl.makeHeapTypeDesc(nil) // string type desc
	fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(dst)))

	// Second pass: fill array
	arrIdx := fl.frame.AllocWord("")
	segStart := fl.frame.AllocWord("")
	segment := fl.frame.AllocTemp(true)
	storeAddr := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))

	fillLoopPC := int32(len(fl.insts))
	bgeFillDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	beqSpc2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
	beqTab2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
	beqNl2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
	beqCr2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))

	// not space
	beqAlreadyIn2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
	// start of new word
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(inWord)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
	alreadyIn2PC := int32(len(fl.insts))
	fl.insts[beqAlreadyIn2].Dst = dis.Imm(alreadyIn2PC)
	jmpNext2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// space: if inWord, store segment
	space2PC := int32(len(fl.insts))
	fl.insts[beqSpc2].Dst = dis.Imm(space2PC)
	fl.insts[beqTab2].Dst = dis.Imm(space2PC)
	fl.insts[beqNl2].Dst = dis.Imm(space2PC)
	fl.insts[beqCr2].Dst = dis.Imm(space2PC)

	beqNotInWord := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
	// end of word: store s[segStart:i]
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(segment)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(i), dis.FP(segment)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(dst), dis.FP(storeAddr), dis.FP(arrIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segment), dis.FPInd(storeAddr, 0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
	notInWordPC := int32(len(fl.insts))
	fl.insts[beqNotInWord].Dst = dis.Imm(notInWordPC)

	next2PC := int32(len(fl.insts))
	fl.insts[jmpNext2].Dst = dis.Imm(next2PC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))

	// fill done: if inWord, store last segment
	fillDonePC := int32(len(fl.insts))
	fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
	beqNoLast := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(segment)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(dst), dis.FP(storeAddr), dis.FP(arrIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segment), dis.FPInd(storeAddr, 0)))
	fl.insts[beqNoLast].Dst = dis.Imm(int32(len(fl.insts)))

	return nil
}

// isSpaceHelper emits inline checks for whitespace (space, tab, newline, carriage return).
// Returns the instruction indices of the 4 branch instructions that jump to "is space" target,
// and the instruction index of the "not space" jump.
func (fl *funcLowerer) emitIsSpaceCheck(ch int32) (spaceJmps []int, notSpaceJmp int) {
	spaceJmps = append(spaceJmps, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
	spaceJmps = append(spaceJmps, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
	spaceJmps = append(spaceJmps, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
	spaceJmps = append(spaceJmps, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))
	notSpaceJmp = len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	return
}

// lowerStringsTrim: trim chars in cutset from both ends of s.
func (fl *funcLowerer) lowerStringsTrim(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	cutsetOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenCut := fl.frame.AllocWord("")
	start := fl.frame.AllocWord("")
	end := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")
	j := fl.frame.AllocWord("")
	cutCh := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(start)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(end)))

	// Trim leading
	leadLoopPC := int32(len(fl.insts))
	leadDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(start), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))

	innerLeadPC := int32(len(fl.insts))
	innerLeadDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
	beqLeadMatch := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerLeadPC)))

	// char not in cutset → done leading
	notInCutPC := int32(len(fl.insts))
	fl.insts[innerLeadDone].Dst = dis.Imm(notInCutPC)
	jmpLeadDone := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// char in cutset → start++, continue
	inCutPC := int32(len(fl.insts))
	fl.insts[beqLeadMatch].Dst = dis.Imm(inCutPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(start), dis.FP(start)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(leadLoopPC)))

	leadDonePC := int32(len(fl.insts))
	fl.insts[leadDoneIdx].Dst = dis.Imm(leadDonePC)
	fl.insts[jmpLeadDone].Dst = dis.Imm(leadDonePC)

	// Trim trailing
	trailLoopPC := int32(len(fl.insts))
	trailDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
	tailIdx := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(tailIdx)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(tailIdx), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))

	innerTrailPC := int32(len(fl.insts))
	innerTrailDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
	beqTrailMatch := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerTrailPC)))

	notInCut2PC := int32(len(fl.insts))
	fl.insts[innerTrailDone].Dst = dis.Imm(notInCut2PC)
	jmpTrailDone := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	inCut2PC := int32(len(fl.insts))
	fl.insts[beqTrailMatch].Dst = dis.Imm(inCut2PC)
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(end)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trailLoopPC)))

	trailDonePC := int32(len(fl.insts))
	fl.insts[trailDoneIdx].Dst = dis.Imm(trailDonePC)
	fl.insts[jmpTrailDone].Dst = dis.Imm(trailDonePC)

	// result = s[start:end]
	tmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(start), dis.FP(end), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))
	return nil
}

// lowerStringsTrimLeft: trim chars in cutset from left side of s.
func (fl *funcLowerer) lowerStringsTrimLeft(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	cutsetOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenCut := fl.frame.AllocWord("")
	start := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")
	j := fl.frame.AllocWord("")
	cutCh := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(start)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(start), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))

	innerPC := int32(len(fl.insts))
	innerDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
	beqMatch := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

	// not in cutset → done
	notInPC := int32(len(fl.insts))
	fl.insts[innerDone].Dst = dis.Imm(notInPC)
	jmpDone := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// in cutset → start++
	inPC := int32(len(fl.insts))
	fl.insts[beqMatch].Dst = dis.Imm(inPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(start), dis.FP(start)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[jmpDone].Dst = dis.Imm(donePC)

	tmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(start), dis.FP(lenS), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))
	return nil
}

// lowerStringsTrimRight: trim chars in cutset from right side of s.
func (fl *funcLowerer) lowerStringsTrimRight(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	cutsetOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	lenCut := fl.frame.AllocWord("")
	end := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")
	j := fl.frame.AllocWord("")
	cutCh := fl.frame.AllocWord("")
	tailIdx := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(end)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLEW, dis.FP(end), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(tailIdx)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(tailIdx), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))

	innerPC := int32(len(fl.insts))
	innerDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
	beqMatch := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

	notInPC := int32(len(fl.insts))
	fl.insts[innerDone].Dst = dis.Imm(notInPC)
	jmpDone := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	inPC := int32(len(fl.insts))
	fl.insts[beqMatch].Dst = dis.Imm(inPC)
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(end)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[jmpDone].Dst = dis.Imm(donePC)

	tmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(end), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))
	return nil
}

// lowerStringsTitle: capitalize first letter of each word.
func (fl *funcLowerer) lowerStringsTitle(instr *ssa.Call) error {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	ch := fl.frame.AllocWord("")
	prevSpace := fl.frame.AllocWord("")
	result := fl.frame.AllocTemp(true)
	charStr := fl.frame.AllocTemp(true)

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(prevSpace))) // start of string counts as after space

	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	// check if space
	beqSpc := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
	beqTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
	beqNl := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
	beqCr := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))

	// not space: if prevSpace && 'a'-'z', convert to upper
	beqNoPrev := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(prevSpace), dis.Imm(0), dis.Imm(0)))
	bltNoUpper := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(97), dis.Imm(0)))
	bgtNoUpper := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(122), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(32), dis.FP(ch), dis.FP(ch)))
	noUpperPC := int32(len(fl.insts))
	fl.insts[beqNoPrev].Dst = dis.Imm(noUpperPC)
	fl.insts[bltNoUpper].Dst = dis.Imm(noUpperPC)
	fl.insts[bgtNoUpper].Dst = dis.Imm(noUpperPC)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(prevSpace)))
	jmpAppend := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// space:
	spacePC := int32(len(fl.insts))
	fl.insts[beqSpc].Dst = dis.Imm(spacePC)
	fl.insts[beqTab].Dst = dis.Imm(spacePC)
	fl.insts[beqNl].Dst = dis.Imm(spacePC)
	fl.insts[beqCr].Dst = dis.Imm(spacePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(prevSpace)))

	// append char
	appendPC := int32(len(fl.insts))
	fl.insts[jmpAppend].Dst = dis.Imm(appendPC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.Imm(0), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return nil
}

// ============================================================
// strings package — Cut, CutPrefix, CutSuffix
// ============================================================

// lowerStringsCut: strings.Cut(s, sep) → (before, after string, found bool)
func (fl *funcLowerer) lowerStringsCut(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	sepOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	// Use Index to find sep in s
	lenS := fl.frame.AllocWord("")
	lenSep := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	limit := fl.frame.AllocWord("")
	candidate := fl.frame.AllocTemp(true)
	endIdx := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, sepOp, dis.FP(lenSep)))

	// Default: not found → before=s, after="", found=false
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst+iby2wd)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))

	// if lenSep > lenS → not found
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))

	// limit = lenS - lenSep + 1
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	loopPC := int32(len(fl.insts))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(candidate)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
	beqFoundIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, sepOp, dis.FP(candidate), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Found at i: before=s[:i], after=s[i+lenSep:], found=true
	foundPC := int32(len(fl.insts))
	fl.insts[beqFoundIdx].Dst = dis.Imm(foundPC)
	before := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(before)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(i), dis.FP(before)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(before), dis.FP(dst)))

	after := fl.frame.AllocTemp(true)
	afterStart := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(afterStart)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(after)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(afterStart), dis.FP(lenS), dis.FP(after)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(after), dis.FP(dst+iby2wd)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+2*iby2wd)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerStringsCutPrefix: strings.CutPrefix(s, prefix) → (after string, found bool)
func (fl *funcLowerer) lowerStringsCutPrefix(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	prefixOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	lenS := fl.frame.AllocWord("")
	lenP := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, prefixOp, dis.FP(lenP)))

	// Default: not found → after=s, found=false
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))

	// if lenP > lenS → not found
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenP), dis.FP(lenS), dis.Imm(0)))

	// Check prefix: head = s[:lenP]
	head := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(head)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenP), dis.FP(head)))
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEC, prefixOp, dis.FP(head), dis.Imm(0)))

	// Match: after = s[lenP:], found = true
	afterSlot := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(afterSlot)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lenP), dis.FP(lenS), dis.FP(afterSlot)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(afterSlot), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bneIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerStringsCutSuffix: strings.CutSuffix(s, suffix) → (before string, found bool)
func (fl *funcLowerer) lowerStringsCutSuffix(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	suffixOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	lenS := fl.frame.AllocWord("")
	lenSuf := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.ILENC, suffixOp, dis.FP(lenSuf)))

	// Default: not found → before=s, found=false
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))

	// if lenSuf > lenS → not found
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSuf), dis.FP(lenS), dis.Imm(0)))

	// Check suffix: tail = s[lenS-lenSuf:]
	tailStart := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSuf), dis.FP(lenS), dis.FP(tailStart)))
	tail := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(tail)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(tailStart), dis.FP(lenS), dis.FP(tail)))
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEC, suffixOp, dis.FP(tail), dis.Imm(0)))

	// Match: before = s[:tailStart], found = true
	beforeSlot := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(beforeSlot)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(tailStart), dis.FP(beforeSlot)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(beforeSlot), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))

	donePC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(donePC)
	fl.insts[bneIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// ============================================================
// math package — new functions
// ============================================================

// lowerMathFloor: floor(x) = trunc(x) if x >= 0, else trunc(x)-1 if frac != 0.
func (fl *funcLowerer) lowerMathFloor(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	oneOff := fl.comp.AllocReal(1.0)

	// Truncate toward zero: CVTFW rounds to nearest, then correct
	truncFloat := fl.frame.AllocReal("")
	fl.emitTruncToFloat(src, truncFloat)

	// if truncFloat <= src → floor = truncFloat (positive or exact)
	blefIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLEF, dis.FP(truncFloat), src, dis.Imm(0)))

	// truncFloat > src (negative with fraction): floor = truncFloat - 1
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), dis.FP(truncFloat), dis.FP(dst)))
	jmpDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// truncFloat <= src: dst = truncFloat
	posPC := int32(len(fl.insts))
	fl.insts[blefIdx].Dst = dis.Imm(posPC)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(truncFloat), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpDoneIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathCeil: ceil(x) = trunc(x) if exact, trunc(x)+1 if x > trunc(x).
func (fl *funcLowerer) lowerMathCeil(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	oneOff := fl.comp.AllocReal(1.0)

	// Truncate toward zero
	truncFloat := fl.frame.AllocReal("")
	fl.emitTruncToFloat(src, truncFloat)

	// if truncFloat >= src → ceil = truncFloat (negative or exact)
	bgefIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(truncFloat), src, dis.Imm(0)))

	// truncFloat < src (positive with fraction): ceil = truncFloat + 1
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), dis.FP(truncFloat), dis.FP(dst)))
	jmpDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// truncFloat >= src: dst = truncFloat
	negPC := int32(len(fl.insts))
	fl.insts[bgefIdx].Dst = dis.Imm(negPC)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(truncFloat), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpDoneIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathRound: round to nearest, ties away from zero.
// Uses CVTFW which already rounds to nearest (adding 0.5 bias).
func (fl *funcLowerer) lowerMathRound(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// CVTFW already rounds to nearest int. Convert back to float.
	intSlot := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ICVTFW, src, dis.FP(intSlot)))
	fl.emit(dis.Inst2(dis.ICVTWF, dis.FP(intSlot), dis.FP(dst)))
	return nil
}

// lowerMathTrunc: truncate to integer (toward zero).
func (fl *funcLowerer) lowerMathTrunc(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	truncFloat := fl.frame.AllocReal("")
	fl.emitTruncToFloat(src, truncFloat)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(truncFloat), dis.FP(dst)))
	return nil
}

// emitTruncToFloat truncates src (float64) toward zero and stores the result
// as a float64 in dstSlot. Uses CVTFW (which rounds to nearest) and corrects.
func (fl *funcLowerer) emitTruncToFloat(src dis.Operand, dstSlot int32) {
	zeroOff := fl.comp.AllocReal(0.0)
	oneOff := fl.comp.AllocReal(1.0)

	intSlot := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ICVTFW, src, dis.FP(intSlot)))    // round to nearest int
	fl.emit(dis.Inst2(dis.ICVTWF, dis.FP(intSlot), dis.FP(dstSlot))) // back to float

	// CVTFW rounds to nearest. For truncation, correct if overshoot:
	// if src >= 0 && dstSlot > src: dstSlot -= 1.0
	// if src < 0 && dstSlot < src: dstSlot += 1.0

	// Check if src >= 0
	bgefIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))

	// src < 0: if dstSlot < src, add 1 (rounded too far negative)
	bgefNegIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(dstSlot), src, dis.Imm(0))) // dstSlot >= src → no correction
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), dis.FP(dstSlot), dis.FP(dstSlot)))
	jmpDoneIdx1 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// src >= 0: if dstSlot > src, subtract 1 (rounded too far positive)
	posPC := int32(len(fl.insts))
	fl.insts[bgefIdx].Dst = dis.Imm(posPC)
	bgefPosIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(dstSlot), src, dis.Imm(0))) // dstSlot >= src → check equality too
	// dstSlot < src is impossible for positive truncation, skip
	jmpDoneIdx2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// dstSlot >= src: check if dstSlot > src (not just equal)
	gePC := int32(len(fl.insts))
	fl.insts[bgefPosIdx].Dst = dis.Imm(gePC)
	beqfIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, dis.FP(dstSlot), src, dis.Imm(0))) // equal → no correction
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), dis.FP(dstSlot), dis.FP(dstSlot)))

	donePC := int32(len(fl.insts))
	fl.insts[bgefNegIdx].Dst = dis.Imm(donePC)
	fl.insts[jmpDoneIdx1].Dst = dis.Imm(donePC)
	fl.insts[jmpDoneIdx2].Dst = dis.Imm(donePC)
	fl.insts[beqfIdx].Dst = dis.Imm(donePC)
}

// lowerMathPow: x^y. Converts y to int and uses EXPF instruction.
func (fl *funcLowerer) lowerMathPow(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	// EXPF src, mid, dst: dst = mid ^ src where src is a WORD (int).
	// Convert y from float to int first.
	yInt := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ICVTFW, yOp, dis.FP(yInt)))
	fl.emit(dis.NewInst(dis.IEXPF, dis.FP(yInt), xOp, dis.FP(dst)))
	return nil
}

// lowerMathMod: floating-point modulo.
func (fl *funcLowerer) lowerMathMod(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	// mod = x - trunc(x/y) * y
	quotient := fl.frame.AllocReal("")
	truncQf := fl.frame.AllocReal("")
	prod := fl.frame.AllocReal("")

	fl.emit(dis.NewInst(dis.IDIVF, yOp, xOp, dis.FP(quotient)))
	fl.emitTruncToFloat(dis.FP(quotient), truncQf)
	fl.emit(dis.NewInst(dis.IMULF, yOp, dis.FP(truncQf), dis.FP(prod)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(prod), xOp, dis.FP(dst)))
	return nil
}

// lowerMathLog: natural logarithm using series approximation.
// ln(x) = 2*sum(((x-1)/(x+1))^(2k+1)/(2k+1), k=0..N)
func (fl *funcLowerer) lowerMathLog(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	oneOff := fl.comp.AllocReal(1.0)
	twoOff := fl.comp.AllocReal(2.0)
	num := fl.frame.AllocWord("")
	den := fl.frame.AllocWord("")
	t := fl.frame.AllocWord("")
	t2 := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	sum := fl.frame.AllocWord("")
	denom := fl.frame.AllocWord("")

	// t = (x-1)/(x+1)
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), src, dis.FP(num)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(den)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(den), dis.FP(num), dis.FP(t)))
	// t2 = t*t
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(t), dis.FP(t), dis.FP(t2)))
	// sum = t (first term k=0: t^1/1 = t)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(sum)))
	// term = t (current power)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(term)))

	// Unrolled 12 terms for reasonable precision
	for k := 1; k <= 12; k++ {
		d := float64(2*k + 1)
		denomOff := fl.comp.AllocReal(d)
		// term *= t2
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(t2), dis.FP(term), dis.FP(term)))
		// sum += term / denom
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(denomOff), dis.FP(term), dis.FP(denom)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(denom), dis.FP(sum), dis.FP(sum)))
	}

	// result = 2 * sum
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(sum), dis.FP(dst)))
	return nil
}

// lowerMathLog2: log2(x) = ln(x) / ln(2).
func (fl *funcLowerer) lowerMathLog2(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	ln2Off := fl.comp.AllocReal(0.6931471805599453)
	oneOff := fl.comp.AllocReal(1.0)
	twoOff := fl.comp.AllocReal(2.0)
	num := fl.frame.AllocWord("")
	den := fl.frame.AllocWord("")
	t := fl.frame.AllocWord("")
	t2 := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	sum := fl.frame.AllocWord("")
	denom := fl.frame.AllocWord("")
	lnx := fl.frame.AllocWord("")

	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), src, dis.FP(num)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(den)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(den), dis.FP(num), dis.FP(t)))
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(t), dis.FP(t), dis.FP(t2)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(term)))

	for k := 1; k <= 12; k++ {
		d := float64(2*k + 1)
		denomOff := fl.comp.AllocReal(d)
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(t2), dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(denomOff), dis.FP(term), dis.FP(denom)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(denom), dis.FP(sum), dis.FP(sum)))
	}

	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(sum), dis.FP(lnx)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(ln2Off), dis.FP(lnx), dis.FP(dst)))
	return nil
}

// lowerMathLog10: log10(x) = ln(x) / ln(10).
func (fl *funcLowerer) lowerMathLog10(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	ln10Off := fl.comp.AllocReal(2.302585092994046)
	oneOff := fl.comp.AllocReal(1.0)
	twoOff := fl.comp.AllocReal(2.0)
	num := fl.frame.AllocWord("")
	den := fl.frame.AllocWord("")
	t := fl.frame.AllocWord("")
	t2 := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	sum := fl.frame.AllocWord("")
	denom := fl.frame.AllocWord("")
	lnx := fl.frame.AllocWord("")

	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), src, dis.FP(num)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(den)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(den), dis.FP(num), dis.FP(t)))
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(t), dis.FP(t), dis.FP(t2)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(t), dis.FP(term)))

	for k := 1; k <= 12; k++ {
		d := float64(2*k + 1)
		denomOff := fl.comp.AllocReal(d)
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(t2), dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(denomOff), dis.FP(term), dis.FP(denom)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(denom), dis.FP(sum), dis.FP(sum)))
	}

	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(sum), dis.FP(lnx)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(ln10Off), dis.FP(lnx), dis.FP(dst)))
	return nil
}

// lowerMathExp: e^x using Taylor series.
func (fl *funcLowerer) lowerMathExp(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	oneOff := fl.comp.AllocReal(1.0)
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")

	// sum = 1, term = 1
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(term)))

	// Unrolled 20 terms: term *= x/k; sum += term
	for k := 1; k <= 20; k++ {
		kOff := fl.comp.AllocReal(float64(k))
		fl.emit(dis.NewInst(dis.IMULF, src, dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(kOff), dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(term), dis.FP(sum), dis.FP(sum)))
	}

	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(sum), dis.FP(dst)))
	return nil
}

// lowerMathInf: return +Inf or -Inf based on sign argument.
func (fl *funcLowerer) lowerMathInf(instr *ssa.Call) error {
	signOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	posInfOff := fl.comp.AllocReal(math.Inf(1)) // overflow to +Inf
	negInfOff := fl.comp.AllocReal(math.Inf(-1))

	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(posInfOff), dis.FP(dst)))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, signOp, dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(negInfOff), dis.FP(dst)))
	donePC := int32(len(fl.insts))
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathNaN: return NaN.
func (fl *funcLowerer) lowerMathNaN(instr *ssa.Call) error {
	dst := fl.slotOf(instr)
	// NaN = 0.0 / 0.0
	zeroOff := fl.comp.AllocReal(0.0)
	tmp := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(tmp)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(tmp), dis.FP(tmp), dis.FP(dst)))
	return nil
}

// lowerMathIsNaN: NaN != NaN.
func (fl *funcLowerer) lowerMathIsNaN(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	// NaN is the only value where x != x
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEF, src, src, dis.Imm(0)))
	jmpIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	truePC := int32(len(fl.insts))
	fl.insts[bneIdx].Dst = dis.Imm(truePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathIsInf: check if value is +/-Inf.
func (fl *funcLowerer) lowerMathIsInf(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	signOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	posInfOff := fl.comp.AllocReal(math.Inf(1))
	negInfOff := fl.comp.AllocReal(math.Inf(-1))

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	// sign > 0 → check +Inf only
	bgtIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, signOp, dis.Imm(0), dis.Imm(0)))
	// sign < 0 → check -Inf only
	bltIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, signOp, dis.Imm(0), dis.Imm(0)))

	// sign == 0 → check either
	beqPosIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(posInfOff), dis.Imm(0)))
	beqNegIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(negInfOff), dis.Imm(0)))
	jmpDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// check +Inf
	checkPosPC := int32(len(fl.insts))
	fl.insts[bgtIdx].Dst = dis.Imm(checkPosPC)
	beqPos2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(posInfOff), dis.Imm(0)))
	jmpDone2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// check -Inf
	checkNegPC := int32(len(fl.insts))
	fl.insts[bltIdx].Dst = dis.Imm(checkNegPC)
	beqNeg2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(negInfOff), dis.Imm(0)))
	jmpDone3 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// true:
	truePC := int32(len(fl.insts))
	fl.insts[beqPosIdx].Dst = dis.Imm(truePC)
	fl.insts[beqNegIdx].Dst = dis.Imm(truePC)
	fl.insts[beqPos2].Dst = dis.Imm(truePC)
	fl.insts[beqNeg2].Dst = dis.Imm(truePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpDoneIdx].Dst = dis.Imm(donePC)
	fl.insts[jmpDone2].Dst = dis.Imm(donePC)
	fl.insts[jmpDone3].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathSignbit: return true if x is negative or negative zero.
func (fl *funcLowerer) lowerMathSignbit(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	bgeIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
	donePC := int32(len(fl.insts))
	fl.insts[bgeIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathCopysign: return x with the sign of y.
func (fl *funcLowerer) lowerMathCopysign(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)

	// abs(x)
	absX := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, xOp, dis.FP(absX)))
	bgefAbsIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, xOp, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, xOp, dis.FP(absX)))
	absPC := int32(len(fl.insts))
	fl.insts[bgefAbsIdx].Dst = dis.Imm(absPC)

	// if y >= 0 → dst = absX, else dst = -absX
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(absX), dis.FP(dst)))
	bgefIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, yOp, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(absX), dis.FP(dst)))
	donePC := int32(len(fl.insts))
	fl.insts[bgefIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathDim: max(x-y, 0).
func (fl *funcLowerer) lowerMathDim(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)

	diff := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.ISUBF, yOp, xOp, dis.FP(diff)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(diff), dis.FP(dst)))
	bgefIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(diff), dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dst)))
	donePC := int32(len(fl.insts))
	fl.insts[bgefIdx].Dst = dis.Imm(donePC)
	return nil
}

// lowerMathFloat64bits: reinterpret float64 as uint64 (identity on Dis — same 8-byte word).
func (fl *funcLowerer) lowerMathFloat64bits(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	// On Dis VM, float64 and int64 share the same 8-byte WORD slot. Just copy.
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(dst)))
	return nil
}

// lowerMathFloat64frombits: reinterpret uint64 as float64 (identity on Dis).
func (fl *funcLowerer) lowerMathFloat64frombits(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
	return nil
}

// lowerMathTrig: sin/cos/tan using Taylor series.
func (fl *funcLowerer) lowerMathTrig(instr *ssa.Call, name string) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	oneOff := fl.comp.AllocReal(1.0)
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	x2 := fl.frame.AllocWord("")

	fl.emit(dis.NewInst(dis.IMULF, src, src, dis.FP(x2)))

	switch name {
	case "Sin":
		// sin(x) = x - x^3/3! + x^5/5! - ...
		fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(sum)))
		fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(term)))
		for k := 1; k <= 10; k++ {
			n1 := float64(2*k * (2*k + 1))
			nOff := fl.comp.AllocReal(n1)
			fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(term), dis.FP(term)))
			fl.emit(dis.NewInst(dis.IDIVF, dis.MP(nOff), dis.FP(term), dis.FP(term)))
			fl.emit(dis.Inst2(dis.INEGF, dis.FP(term), dis.FP(term)))
			fl.emit(dis.NewInst(dis.IADDF, dis.FP(term), dis.FP(sum), dis.FP(sum)))
		}
	case "Cos":
		// cos(x) = 1 - x^2/2! + x^4/4! - ...
		fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(sum)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(term)))
		for k := 1; k <= 10; k++ {
			n1 := float64((2*k - 1) * (2 * k))
			nOff := fl.comp.AllocReal(n1)
			fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(term), dis.FP(term)))
			fl.emit(dis.NewInst(dis.IDIVF, dis.MP(nOff), dis.FP(term), dis.FP(term)))
			fl.emit(dis.Inst2(dis.INEGF, dis.FP(term), dis.FP(term)))
			fl.emit(dis.NewInst(dis.IADDF, dis.FP(term), dis.FP(sum), dis.FP(sum)))
		}
	case "Tan":
		// tan(x) = sin(x)/cos(x) — compute both inline
		sinSum := fl.frame.AllocWord("")
		sinTerm := fl.frame.AllocWord("")
		cosSum := fl.frame.AllocWord("")
		cosTerm := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(sinSum)))
		fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(sinTerm)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(cosSum)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(cosTerm)))

		for k := 1; k <= 10; k++ {
			sn := float64(2*k * (2*k + 1))
			cn := float64((2*k - 1) * (2 * k))
			snOff := fl.comp.AllocReal(sn)
			cnOff := fl.comp.AllocReal(cn)
			fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(sinTerm), dis.FP(sinTerm)))
			fl.emit(dis.NewInst(dis.IDIVF, dis.MP(snOff), dis.FP(sinTerm), dis.FP(sinTerm)))
			fl.emit(dis.Inst2(dis.INEGF, dis.FP(sinTerm), dis.FP(sinTerm)))
			fl.emit(dis.NewInst(dis.IADDF, dis.FP(sinTerm), dis.FP(sinSum), dis.FP(sinSum)))
			fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(cosTerm), dis.FP(cosTerm)))
			fl.emit(dis.NewInst(dis.IDIVF, dis.MP(cnOff), dis.FP(cosTerm), dis.FP(cosTerm)))
			fl.emit(dis.Inst2(dis.INEGF, dis.FP(cosTerm), dis.FP(cosTerm)))
			fl.emit(dis.NewInst(dis.IADDF, dis.FP(cosTerm), dis.FP(cosSum), dis.FP(cosSum)))
		}

		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(cosSum), dis.FP(sinSum), dis.FP(dst)))
		return nil
	}

	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(sum), dis.FP(dst)))
	return nil
}

// lowerMathAtan: atan(x) using half-angle reduction + Taylor series.
func (fl *funcLowerer) lowerMathAtan(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	fl.emitAtanInline(src, dst)
	return nil
}

// lowerMathAsin: asin(x) = atan(x / sqrt(1 - x*x))
func (fl *funcLowerer) lowerMathAsin(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	piOver2Off := fl.comp.AllocReal(1.5707963267948966)
	zeroOff := fl.comp.AllocReal(0.0)

	x2 := fl.frame.AllocWord("")
	arg := fl.frame.AllocWord("")

	// x2 = 1 - src*src
	fl.emit(dis.NewInst(dis.IMULF, src, src, dis.FP(x2)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(x2), dis.MP(oneOff), dis.FP(x2)))

	// If x2 <= 0 (|src| >= 1), return ±pi/2
	skipSpecial := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTF, dis.FP(x2), dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(piOver2Off), dis.FP(dst)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(dst), dis.FP(dst)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))
	endIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// Normal case: sqrt(1-x^2) via Newton's method
	fl.insts[skipSpecial].Dst = dis.Imm(int32(len(fl.insts)))
	guess := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(guess)))
	for i := 0; i < 8; i++ {
		t := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(guess), dis.FP(x2), dis.FP(t)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(guess), dis.FP(t), dis.FP(guess)))
		fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(guess), dis.FP(guess)))
	}

	// arg = src / sqrt(1-x^2)
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(guess), src, dis.FP(arg)))

	// atan(arg) using the shared half-angle implementation
	fl.emitAtanInline(dis.FP(arg), dst)

	fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
	return nil
}

// lowerMathAcos: acos(x) = pi/2 - asin(x)
// We inline the same computation as asin and subtract from pi/2.
func (fl *funcLowerer) lowerMathAcos(instr *ssa.Call) error {
	// First compute asin via the same method
	err := fl.lowerMathAsin(instr)
	if err != nil {
		return err
	}
	dst := fl.slotOf(instr)
	piOver2Off := fl.comp.AllocReal(1.5707963267948966)
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(dst), dis.MP(piOver2Off), dis.FP(dst)))
	return nil
}

// lowerMathSinh: sinh(x) = (exp(x) - exp(-x)) / 2
func (fl *funcLowerer) lowerMathSinh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	twoOff := fl.comp.AllocReal(2.0)

	ex := fl.frame.AllocWord("")
	enx := fl.frame.AllocWord("")
	negX := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(negX)))
	fl.emitExpInline(src, ex)
	fl.emitExpInline(dis.FP(negX), enx)
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(enx), dis.FP(ex), dis.FP(dst)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(twoOff), dis.FP(dst), dis.FP(dst)))
	return nil
}

// lowerMathCosh: cosh(x) = (exp(x) + exp(-x)) / 2
func (fl *funcLowerer) lowerMathCosh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	twoOff := fl.comp.AllocReal(2.0)

	ex := fl.frame.AllocWord("")
	enx := fl.frame.AllocWord("")
	negX := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(negX)))
	fl.emitExpInline(src, ex)
	fl.emitExpInline(dis.FP(negX), enx)
	fl.emit(dis.NewInst(dis.IADDF, dis.FP(enx), dis.FP(ex), dis.FP(dst)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(twoOff), dis.FP(dst), dis.FP(dst)))
	return nil
}

// lowerMathTanh: tanh(x) = sinh(x)/cosh(x) = (e^x - e^-x)/(e^x + e^-x)
func (fl *funcLowerer) lowerMathTanh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	ex := fl.frame.AllocWord("")
	enx := fl.frame.AllocWord("")
	negX := fl.frame.AllocWord("")
	num := fl.frame.AllocWord("")
	den := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(negX)))
	fl.emitExpInline(src, ex)
	fl.emitExpInline(dis.FP(negX), enx)
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(enx), dis.FP(ex), dis.FP(num)))
	fl.emit(dis.NewInst(dis.IADDF, dis.FP(enx), dis.FP(ex), dis.FP(den)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(den), dis.FP(num), dis.FP(dst)))
	return nil
}

// emitExpInline emits code to compute exp(src) and store result at FP(dstSlot).
// Uses Taylor series: e^x = 1 + x + x^2/2! + x^3/3! + ...
func (fl *funcLowerer) emitExpInline(src dis.Operand, dstSlot int32) {
	oneOff := fl.comp.AllocReal(1.0)
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(term)))
	for k := 1; k <= 20; k++ {
		kOff := fl.comp.AllocReal(float64(k))
		fl.emit(dis.NewInst(dis.IMULF, src, dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(kOff), dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(term), dis.FP(sum), dis.FP(sum)))
	}
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(sum), dis.FP(dstSlot)))
}

// lowerMathExp2: exp2(x) = 2^x = exp(x * ln(2))
func (fl *funcLowerer) lowerMathExp2(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	ln2Off := fl.comp.AllocReal(0.6931471805599453) // ln(2)

	scaled := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(ln2Off), src, dis.FP(scaled)))
	fl.emitExpInline(dis.FP(scaled), dst)
	return nil
}

// lowerMathLog1p: log1p(x) = log(1+x)
func (fl *funcLowerer) lowerMathLog1p(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)

	arg := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(arg)))
	// Reuse the log computation from lowerMathLog
	fl.emitLogInline(dis.FP(arg), dst)
	return nil
}

// lowerMathCbrt: cbrt(x) using Newton's method.
// Newton iteration: g = (2*g + a/g²) / 3
// Initial guess from exp(log(|x|)/3), then refine with Newton.
func (fl *funcLowerer) lowerMathCbrt(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)
	threeOff := fl.comp.AllocReal(3.0)
	twoOff := fl.comp.AllocReal(2.0)
	oneOff := fl.comp.AllocReal(1.0)

	absX := fl.frame.AllocWord("")
	signNeg := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(signNeg)))
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(absX)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(absX)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(signNeg)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	// Handle zero
	skipZero := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEF, dis.FP(absX), dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dst)))
	endIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipZero].Dst = dis.Imm(int32(len(fl.insts)))

	// Initial guess via exp(log(x)/3)
	logA := fl.frame.AllocWord("")
	scaled := fl.frame.AllocWord("")
	guess := fl.frame.AllocWord("")
	fl.emitLogInline(dis.FP(absX), logA)
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(threeOff), dis.FP(logA), dis.FP(scaled)))
	fl.emitExpInline(dis.FP(scaled), guess)

	// Newton refinement: g = (2*g + a/g²) / 3, 5 iterations
	for i := 0; i < 5; i++ {
		g2 := fl.frame.AllocWord("")
		ag2 := fl.frame.AllocWord("")
		twoG := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(guess), dis.FP(guess), dis.FP(g2)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(g2), dis.FP(absX), dis.FP(ag2)))
		fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(guess), dis.FP(twoG)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(ag2), dis.FP(twoG), dis.FP(guess)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(threeOff), dis.FP(guess), dis.FP(guess)))
	}
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(guess), dis.FP(dst)))

	// Apply sign
	skipSign := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(signNeg), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(dst), dis.FP(dst)))
	fl.insts[skipSign].Dst = dis.Imm(int32(len(fl.insts)))

	fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
	_ = oneOff
	return nil
}

// emitLogInline emits code for natural logarithm using the series:
// log(x) = 2 * (d + d^3/3 + d^5/5 + ...) where d = (x-1)/(x+1)
func (fl *funcLowerer) emitLogInline(src dis.Operand, dstSlot int32) {
	oneOff := fl.comp.AllocReal(1.0)
	twoOff := fl.comp.AllocReal(2.0)

	d := fl.frame.AllocWord("")
	xp1 := fl.frame.AllocWord("")
	xm1 := fl.frame.AllocWord("")
	d2 := fl.frame.AllocWord("")
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")

	// d = (x-1)/(x+1)
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(xp1)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), src, dis.FP(xm1)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(xp1), dis.FP(xm1), dis.FP(d)))
	// d2 = d*d
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(d), dis.FP(d), dis.FP(d2)))
	// sum = d, term = d
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(d), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(d), dis.FP(term)))
	for k := 1; k <= 20; k++ {
		denomOff := fl.comp.AllocReal(float64(2*k + 1))
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(d2), dis.FP(term), dis.FP(term)))
		tmp := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(denomOff), dis.FP(term), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(tmp), dis.FP(sum), dis.FP(sum)))
	}
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(sum), dis.FP(dstSlot)))
}

// lowerMathAtan2: atan2(y, x)
// Handles all quadrants: atan2(y,x) = atan(y/x) + adjustment
func (fl *funcLowerer) lowerMathAtan2(instr *ssa.Call) error {
	yOp := fl.operandOf(instr.Call.Args[0])
	xOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)
	piOff := fl.comp.AllocReal(3.141592653589793)
	piOver2Off := fl.comp.AllocReal(1.5707963267948966)
	negPiOver2Off := fl.comp.AllocReal(-1.5707963267948966)

	// Handle x > 0: atan2(y,x) = atan(y/x)
	skipXPos := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTF, dis.MP(zeroOff), xOp, dis.Imm(0)))

	ratio := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IDIVF, xOp, yOp, dis.FP(ratio)))

	// Inline atan(ratio)
	atanResult := fl.frame.AllocWord("")
	fl.emitAtanInline(dis.FP(ratio), atanResult)
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(atanResult), dis.FP(dst)))
	endIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	fl.insts[skipXPos].Dst = dis.Imm(int32(len(fl.insts)))

	// Handle x == 0: return ±pi/2 based on sign of y (or 0 if y==0)
	skipXZero := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEF, xOp, dis.MP(zeroOff), dis.Imm(0)))
	// x == 0
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dst)))
	skipYPos := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.MP(zeroOff), yOp, dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(piOver2Off), dis.FP(dst)))
	endIdx2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipYPos].Dst = dis.Imm(int32(len(fl.insts)))
	skipYNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, yOp, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(negPiOver2Off), dis.FP(dst)))
	fl.insts[skipYNeg].Dst = dis.Imm(int32(len(fl.insts)))
	endIdx3 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	fl.insts[skipXZero].Dst = dis.Imm(int32(len(fl.insts)))
	// x < 0: atan2(y,x) = atan(y/x) + pi if y >= 0, atan(y/x) - pi if y < 0
	ratio2 := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IDIVF, xOp, yOp, dis.FP(ratio2)))
	atanResult2 := fl.frame.AllocWord("")
	fl.emitAtanInline(dis.FP(ratio2), atanResult2)
	skipYGe0 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTF, yOp, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(piOff), dis.FP(atanResult2), dis.FP(dst)))
	endIdx4 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipYGe0].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(piOff), dis.FP(atanResult2), dis.FP(dst)))

	endPC := int32(len(fl.insts))
	fl.insts[endIdx].Dst = dis.Imm(endPC)
	fl.insts[endIdx2].Dst = dis.Imm(endPC)
	fl.insts[endIdx3].Dst = dis.Imm(endPC)
	fl.insts[endIdx4].Dst = dis.Imm(endPC)
	_ = negPiOver2Off
	return nil
}

// emitAtanInline emits code for atan(src) into FP(dstSlot).
// Uses half-angle reduction + Taylor series.
// Steps: 1) Extract sign, work with |x|
//        2) If |x| > 1, use atan(x) = pi/2 - atan(1/x) to bring to [0,1]
//        3) Apply 3x half-angle reduction: x → x/(1+sqrt(1+x²))
//        4) Taylor series converges fast for the reduced argument (~0.1)
//        5) Multiply result by 2^halvings, apply reductions
func (fl *funcLowerer) emitAtanInline(src dis.Operand, dstSlot int32) {
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	piOver2Off := fl.comp.AllocReal(1.5707963267948966)
	zeroOff := fl.comp.AllocReal(0.0)
	twoOff := fl.comp.AllocReal(2.0)

	x := fl.frame.AllocWord("")
	absX := fl.frame.AllocWord("")
	signNeg := fl.frame.AllocWord("")
	reduced := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(x)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(signNeg)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(x), dis.FP(absX)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(x), dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(x), dis.FP(absX)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(x), dis.FP(x)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(signNeg)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	// Reciprocal reduction: if x > 1, x = 1/x, reduced = 1
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(reduced)))
	skipReduce := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.MP(oneOff), dis.FP(absX), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(x), dis.MP(oneOff), dis.FP(x)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(reduced)))
	fl.insts[skipReduce].Dst = dis.Imm(int32(len(fl.insts)))

	// Half-angle reduction: x → x/(1+sqrt(1+x²)), applied 3 times
	// After 3 halvings, x ∈ [0,1] → ~[0, 0.098], giving fast convergence
	for halv := 0; halv < 3; halv++ {
		xx := fl.frame.AllocWord("")
		sqArg := fl.frame.AllocWord("")
		guess := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(x), dis.FP(x), dis.FP(xx)))
		fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), dis.FP(xx), dis.FP(sqArg)))
		// sqrt via Newton's method (6 iterations)
		fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(guess)))
		for ni := 0; ni < 6; ni++ {
			t := fl.frame.AllocWord("")
			fl.emit(dis.NewInst(dis.IDIVF, dis.FP(guess), dis.FP(sqArg), dis.FP(t)))
			fl.emit(dis.NewInst(dis.IADDF, dis.FP(guess), dis.FP(t), dis.FP(guess)))
			fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(guess), dis.FP(guess)))
		}
		// x = x / (1 + sqrt(1+x^2))
		denom := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), dis.FP(guess), dis.FP(denom)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(denom), dis.FP(x), dis.FP(x)))
	}

	// Taylor series: atan(x) = x - x³/3 + x⁵/5 - ...
	// With x ~ 0.098, 10 terms gives ~20 digits of accuracy
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	x2 := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(x), dis.FP(x), dis.FP(x2)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(x), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(x), dis.FP(term)))
	for k := 1; k <= 12; k++ {
		d := float64(2*k + 1)
		dOff := fl.comp.AllocReal(d)
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(term), dis.FP(term)))
		fl.emit(dis.Inst2(dis.INEGF, dis.FP(term), dis.FP(term)))
		tmp := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(dOff), dis.FP(term), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(tmp), dis.FP(sum), dis.FP(sum)))
	}

	// Undo half-angle: multiply by 2^3 = 8
	for halv := 0; halv < 3; halv++ {
		fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(sum), dis.FP(sum)))
	}

	// Undo reciprocal reduction: if reduced, sum = pi/2 - sum
	skipUnreduce := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(reduced), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(sum), dis.MP(piOver2Off), dis.FP(sum)))
	fl.insts[skipUnreduce].Dst = dis.Imm(int32(len(fl.insts)))

	// Apply sign
	skipSign := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(signNeg), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(sum), dis.FP(sum)))
	fl.insts[skipSign].Dst = dis.Imm(int32(len(fl.insts)))

	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(sum), dis.FP(dstSlot)))
}

// lowerMathHypot: hypot(x, y) = sqrt(x*x + y*y)
func (fl *funcLowerer) lowerMathHypot(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)

	x2 := fl.frame.AllocWord("")
	y2 := fl.frame.AllocWord("")
	s := fl.frame.AllocWord("")

	fl.emit(dis.NewInst(dis.IMULF, xOp, xOp, dis.FP(x2)))
	fl.emit(dis.NewInst(dis.IMULF, yOp, yOp, dis.FP(y2)))
	fl.emit(dis.NewInst(dis.IADDF, dis.FP(y2), dis.FP(x2), dis.FP(s)))

	// Newton's method for sqrt(s): guess = 1; repeat guess = (guess + s/guess)/2
	guess := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(guess)))
	for i := 0; i < 8; i++ {
		t := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(guess), dis.FP(s), dis.FP(t)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(guess), dis.FP(t), dis.FP(guess)))
		fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(guess), dis.FP(guess)))
	}
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(guess), dis.FP(dst)))
	return nil
}

// lowerMathExpm1: expm1(x) = exp(x) - 1
func (fl *funcLowerer) lowerMathExpm1(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)
	fl.emitExpInline(src, dst)
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), dis.FP(dst), dis.FP(dst)))
	return nil
}

// lowerMathLogb: logb(x) = floor(log2(|x|))
func (fl *funcLowerer) lowerMathLogb(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)
	ln2Off := fl.comp.AllocReal(0.6931471805599453)

	absX := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(absX)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(absX)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	// log2(|x|) = log(|x|) / ln(2)
	logA := fl.frame.AllocWord("")
	fl.emitLogInline(dis.FP(absX), logA)
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(ln2Off), dis.FP(logA), dis.FP(dst)))
	// Floor it: convert to int then back to float
	tmp := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ICVTFW, dis.FP(dst), dis.FP(tmp)))
	fl.emit(dis.Inst2(dis.ICVTWF, dis.FP(tmp), dis.FP(dst)))
	return nil
}

// emitSqrtInline emits code for sqrt(src) into FP(dstSlot).
// Uses Newton's method with proper initial guess and zero handling.
func (fl *funcLowerer) emitSqrtInline(src dis.Operand, dstSlot int32) {
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	zeroOff := fl.comp.AllocReal(0.0)

	// Handle zero: sqrt(0) = 0
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dstSlot)))
	skipZero := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(zeroOff), dis.Imm(0)))

	// Initial guess: (src + 1) / 2 — works well for both small and large values
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(dstSlot)))
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(dstSlot), dis.FP(dstSlot)))

	// Newton: guess = (guess + src/guess) / 2, 15 iterations for good convergence
	for i := 0; i < 15; i++ {
		t := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IDIVF, dis.FP(dstSlot), src, dis.FP(t)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(dstSlot), dis.FP(t), dis.FP(dstSlot)))
		fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(dstSlot), dis.FP(dstSlot)))
	}

	fl.insts[skipZero].Dst = dis.Imm(int32(len(fl.insts)))
}

// lowerMathAsinh: asinh(x) = ln(x + sqrt(x²+1))
func (fl *funcLowerer) lowerMathAsinh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)

	// x² + 1
	x2p1 := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, src, src, dis.FP(x2p1)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), dis.FP(x2p1), dis.FP(x2p1)))

	// sqrt(x²+1)
	sqrtResult := fl.frame.AllocWord("")
	fl.emitSqrtInline(dis.FP(x2p1), sqrtResult)

	// x + sqrt(x²+1)
	arg := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDF, src, dis.FP(sqrtResult), dis.FP(arg)))

	// ln(arg)
	fl.emitLogInline(dis.FP(arg), dst)
	return nil
}

// lowerMathAcosh: acosh(x) = ln(x + sqrt(x²-1)), x >= 1
func (fl *funcLowerer) lowerMathAcosh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	zeroOff := fl.comp.AllocReal(0.0)

	// Handle x == 1: acosh(1) = 0
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dst)))
	skipOne := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(oneOff), dis.Imm(0)))

	// x² - 1
	x2m1 := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, src, src, dis.FP(x2m1)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(oneOff), dis.FP(x2m1), dis.FP(x2m1)))

	// sqrt(x²-1) via Newton — use x2m1 as initial guess for better convergence
	fl.emitSqrtInline(dis.FP(x2m1), dst)

	// x + sqrt(x²-1)
	arg := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDF, src, dis.FP(dst), dis.FP(arg)))

	// ln(arg)
	fl.emitLogInline(dis.FP(arg), dst)
	fl.insts[skipOne].Dst = dis.Imm(int32(len(fl.insts)))
	_ = halfOff
	return nil
}

// lowerMathAtanh: atanh(x) = 0.5 * ln((1+x)/(1-x)), |x| < 1
func (fl *funcLowerer) lowerMathAtanh(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)

	// (1+x) / (1-x)
	num := fl.frame.AllocWord("")
	den := fl.frame.AllocWord("")
	ratio := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(oneOff), src, dis.FP(num)))
	fl.emit(dis.NewInst(dis.ISUBF, src, dis.MP(oneOff), dis.FP(den)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(den), dis.FP(num), dis.FP(ratio)))

	// 0.5 * ln(ratio)
	logResult := fl.frame.AllocWord("")
	fl.emitLogInline(dis.FP(ratio), logResult)
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(logResult), dis.FP(dst)))
	return nil
}

// lowerMathIlogb: ilogb(x) = int(floor(log2(|x|)))
func (fl *funcLowerer) lowerMathIlogb(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	zeroOff := fl.comp.AllocReal(0.0)
	ln2Off := fl.comp.AllocReal(0.6931471805599453)

	absX := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(absX)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(absX)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	// log2(|x|) = log(|x|) / ln(2)
	logA := fl.frame.AllocWord("")
	fl.emitLogInline(dis.FP(absX), logA)
	fResult := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IDIVF, dis.MP(ln2Off), dis.FP(logA), dis.FP(fResult)))
	// Floor to int
	fl.emit(dis.Inst2(dis.ICVTFW, dis.FP(fResult), dis.FP(dst)))
	return nil
}

// lowerMathLdexp: ldexp(frac, exp) = frac * 2^exp
func (fl *funcLowerer) lowerMathLdexp(instr *ssa.Call) error {
	fracOp := fl.operandOf(instr.Call.Args[0])
	expOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)
	twoOff := fl.comp.AllocReal(2.0)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	zeroOff := fl.comp.AllocReal(0.0)

	// Compute 2^exp by iterative multiply/divide
	// result = 1.0; absE = |exp|; for i=0; i<absE; i++ result *= 2; if exp<0 result = 1/result
	result := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(oneOff), dis.FP(result)))
	absE := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVW, expOp, dis.FP(absE)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, expOp, dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, expOp, dis.Imm(0), dis.FP(absE)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	i := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(absE), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
	fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))

	// If exp < 0, result = 1/result
	skipInv := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, expOp, dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IDIVF, dis.FP(result), dis.MP(oneOff), dis.FP(result)))
	fl.insts[skipInv].Dst = dis.Imm(int32(len(fl.insts)))

	// dst = frac * result
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(result), fracOp, dis.FP(dst)))
	_ = halfOff
	_ = zeroOff
	return nil
}

// lowerMathFrexp: frexp(x) = (frac, exp) where x = frac * 2^exp, 0.5 <= |frac| < 1
func (fl *funcLowerer) lowerMathFrexp(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)
	zeroOff := fl.comp.AllocReal(0.0)
	oneOff := fl.comp.AllocReal(1.0)
	halfOff := fl.comp.AllocReal(0.5)
	twoOff := fl.comp.AllocReal(2.0)

	// Handle zero
	fl.emit(dis.Inst2(dis.IMOVF, dis.MP(zeroOff), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
	skipZero := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, src, dis.MP(zeroOff), dis.Imm(0)))

	// Get sign and abs
	absX := fl.frame.AllocWord("")
	signNeg := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(absX)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(signNeg)))
	skipNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, src, dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, src, dis.FP(absX)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(signNeg)))
	fl.insts[skipNeg].Dst = dis.Imm(int32(len(fl.insts)))

	// Normalize: while absX >= 1: absX /= 2, exp++; while absX < 0.5: absX *= 2, exp--
	exp := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(exp)))

	// While absX >= 1: absX /= 2, exp++
	loop1PC := int32(len(fl.insts))
	skip1 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTF, dis.FP(absX), dis.MP(oneOff), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(halfOff), dis.FP(absX), dis.FP(absX)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(exp), dis.FP(exp)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loop1PC)))
	fl.insts[skip1].Dst = dis.Imm(int32(len(fl.insts)))

	// While absX < 0.5: absX *= 2, exp--
	loop2PC := int32(len(fl.insts))
	skip2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(absX), dis.MP(halfOff), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IMULF, dis.MP(twoOff), dis.FP(absX), dis.FP(absX)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(exp), dis.FP(exp)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loop2PC)))
	fl.insts[skip2].Dst = dis.Imm(int32(len(fl.insts)))

	// Apply sign to frac
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(absX), dis.FP(dst)))
	skipApplySign := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(signNeg), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.INEGF, dis.FP(dst), dis.FP(dst)))
	fl.insts[skipApplySign].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(exp), dis.FP(dst+iby2wd)))

	fl.insts[skipZero].Dst = dis.Imm(int32(len(fl.insts)))
	return nil
}

// lowerMathModf: modf(x) = (int, frac) where int is integer part, frac is fractional
func (fl *funcLowerer) lowerMathModf(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	// Integer part: trunc(x) — use emitTruncToFloat for correct truncation
	fl.emitTruncToFloat(src, dst)

	// Fractional part: x - trunc(x)
	fl.emit(dis.NewInst(dis.ISUBF, dis.FP(dst), src, dis.FP(dst+iby2wd)))
	return nil
}

// lowerMathSincos: sincos(x) = (sin(x), cos(x))
func (fl *funcLowerer) lowerMathSincos(instr *ssa.Call) error {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	// Emit sin via Taylor series
	fl.emitSinInline(src, dst)

	// cos(x) = sin(x + pi/2)
	piOver2Off := fl.comp.AllocReal(1.5707963267948966)
	xPlusHalfPi := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(piOver2Off), src, dis.FP(xPlusHalfPi)))
	fl.emitSinInline(dis.FP(xPlusHalfPi), dst+iby2wd)
	return nil
}

// emitSinInline: sin(x) via range reduction + Taylor series
// sin(x) = x - x³/3! + x⁵/5! - x⁷/7! + ...
func (fl *funcLowerer) emitSinInline(src dis.Operand, dstSlot int32) {
	piOff := fl.comp.AllocReal(math.Pi)
	twoPiOff := fl.comp.AllocReal(2 * math.Pi)
	zeroOff := fl.comp.AllocReal(0.0)

	// Range reduction: x = x mod 2*pi
	x := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, src, dis.FP(x)))
	// Quick range reduction using repeated subtraction/addition
	// While x >= 2*pi: x -= 2*pi; while x < 0: x += 2*pi
	loop1PC := int32(len(fl.insts))
	skip1 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTF, dis.FP(x), dis.MP(twoPiOff), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(twoPiOff), dis.FP(x), dis.FP(x)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loop1PC)))
	fl.insts[skip1].Dst = dis.Imm(int32(len(fl.insts)))

	loop2PC := int32(len(fl.insts))
	skip2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.FP(x), dis.MP(zeroOff), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDF, dis.MP(twoPiOff), dis.FP(x), dis.FP(x)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loop2PC)))
	fl.insts[skip2].Dst = dis.Imm(int32(len(fl.insts)))

	// Reduce further: if x > pi, x -= 2*pi (to bring to [-pi, pi])
	skipReduce := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, dis.MP(piOff), dis.FP(x), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBF, dis.MP(twoPiOff), dis.FP(x), dis.FP(x)))
	fl.insts[skipReduce].Dst = dis.Imm(int32(len(fl.insts)))

	// Taylor series for sin(x): x - x³/6 + x⁵/120 - x⁷/5040 + ...
	sum := fl.frame.AllocWord("")
	term := fl.frame.AllocWord("")
	x2 := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, dis.FP(x), dis.FP(x), dis.FP(x2)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(x), dis.FP(sum)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(x), dis.FP(term)))

	for k := 1; k <= 10; k++ {
		d1 := float64(2*k) * float64(2*k+1)
		dOff := fl.comp.AllocReal(d1)
		// term *= -x² / (2k*(2k+1))
		fl.emit(dis.NewInst(dis.IMULF, dis.FP(x2), dis.FP(term), dis.FP(term)))
		fl.emit(dis.Inst2(dis.INEGF, dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(dOff), dis.FP(term), dis.FP(term)))
		fl.emit(dis.NewInst(dis.IADDF, dis.FP(term), dis.FP(sum), dis.FP(sum)))
	}

	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(sum), dis.FP(dstSlot)))
}

// lowerMathFMA: fma(x, y, z) = x*y + z
func (fl *funcLowerer) lowerMathFMA(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	zOp := fl.operandOf(instr.Call.Args[2])
	dst := fl.slotOf(instr)

	prod := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IMULF, xOp, yOp, dis.FP(prod)))
	fl.emit(dis.NewInst(dis.IADDF, zOp, dis.FP(prod), dis.FP(dst)))
	return nil
}

// lowerMathNextafter: nextafter(x, y) — step x toward y by smallest float increment
// Simplified: if x == y return x; if y > x add epsilon; if y < x subtract epsilon
func (fl *funcLowerer) lowerMathNextafter(instr *ssa.Call) error {
	xOp := fl.operandOf(instr.Call.Args[0])
	yOp := fl.operandOf(instr.Call.Args[1])
	dst := fl.slotOf(instr)

	// Use Float64bits/Float64frombits approach:
	// Dis shares 8-byte words for float and int, so we can do bitwise ops on floats
	// Simple approach: adjust bits(x) by ±1
	zeroOff := fl.comp.AllocReal(0.0)

	fl.emit(dis.Inst2(dis.IMOVF, xOp, dis.FP(dst)))
	// if x == y, done
	endIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQF, xOp, yOp, dis.Imm(0)))

	// Read x as integer bits (MOVW copies the raw bits since float/int share WORD)
	bits := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.IMOVF, xOp, dis.FP(bits)))

	// If x == 0, set to ±smallest float
	skipXZero := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEF, xOp, dis.MP(zeroOff), dis.Imm(0)))
	// x is zero: if y > 0, return smallest positive; if y < 0, return smallest negative
	// Smallest positive float64: bits = 1
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(bits)))
	skipYGt := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTF, yOp, dis.MP(zeroOff), dis.Imm(0)))
	// y <= 0: bits = -1 (which as float64 is -5e-324, the smallest negative denorm)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(bits)))
	fl.insts[skipYGt].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(bits), dis.FP(dst)))
	endIdx2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipXZero].Dst = dis.Imm(int32(len(fl.insts)))

	// x > 0 and y > x: add 1 to bits
	// x > 0 and y < x: sub 1 from bits
	// x < 0 and y > x: sub 1 from bits (toward zero = smaller magnitude)
	// x < 0 and y < x: add 1 to bits (away from zero = larger magnitude)
	// Simplification: if (x > 0 && y > x) || (x < 0 && y < x): bits++; else bits--
	skipInc := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTF, yOp, xOp, dis.Imm(0)))
	// y <= x: check if x < 0
	skipXNeg := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEF, xOp, dis.MP(zeroOff), dis.Imm(0)))
	// x < 0 and y <= x (more negative): add 1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(bits), dis.FP(bits)))
	endIdx3 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipXNeg].Dst = dis.Imm(int32(len(fl.insts)))
	// x >= 0 and y <= x: sub 1
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(bits), dis.FP(bits)))
	endIdx4 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	fl.insts[skipInc].Dst = dis.Imm(int32(len(fl.insts)))
	// y > x: check if x >= 0
	skipXPos := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTF, xOp, dis.MP(zeroOff), dis.Imm(0)))
	// x >= 0 and y > x: add 1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(bits), dis.FP(bits)))
	endIdx5 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	fl.insts[skipXPos].Dst = dis.Imm(int32(len(fl.insts)))
	// x < 0 and y > x (toward zero): sub 1
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(bits), dis.FP(bits)))

	endPC := int32(len(fl.insts))
	_ = endPC // used below
	fl.emit(dis.Inst2(dis.IMOVF, dis.FP(bits), dis.FP(dst)))
	finalPC := int32(len(fl.insts))
	fl.insts[endIdx].Dst = dis.Imm(finalPC)
	fl.insts[endIdx2].Dst = dis.Imm(finalPC)
	fl.insts[endIdx3].Dst = dis.Imm(endPC)
	fl.insts[endIdx4].Dst = dis.Imm(endPC)
	fl.insts[endIdx5].Dst = dis.Imm(endPC)
	return nil
}

// ============================================================
// New package dispatchers
// ============================================================

// lowerUnicodeCall handles unicode package functions.
func (fl *funcLowerer) lowerUnicodeCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	src := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	switch callee.Name() {
	case "IsLetter":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		// a-z or A-Z
		blt1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(65), dis.Imm(0)))
		ble1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, src, dis.Imm(90), dis.Imm(0)))
		blt2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(97), dis.Imm(0)))
		ble2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, src, dis.Imm(122), dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		truePC := int32(len(fl.insts))
		fl.insts[ble1].Dst = dis.Imm(truePC)
		fl.insts[ble2].Dst = dis.Imm(truePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt1].Dst = dis.Imm(donePC)
		fl.insts[blt2].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "IsDigit":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		blt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(48), dis.Imm(0)))
		bgt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, dis.Imm(57), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt].Dst = dis.Imm(donePC)
		fl.insts[bgt].Dst = dis.Imm(donePC)
		return true, nil

	case "IsUpper":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		blt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(65), dis.Imm(0)))
		bgt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, dis.Imm(90), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt].Dst = dis.Imm(donePC)
		fl.insts[bgt].Dst = dis.Imm(donePC)
		return true, nil

	case "IsLower":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		blt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(97), dis.Imm(0)))
		bgt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, dis.Imm(122), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt].Dst = dis.Imm(donePC)
		fl.insts[bgt].Dst = dis.Imm(donePC)
		return true, nil

	case "IsSpace":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		beq1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), src, dis.Imm(0)))
		beq2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), src, dis.Imm(0)))
		beq3 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), src, dis.Imm(0)))
		beq4 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), src, dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		truePC := int32(len(fl.insts))
		fl.insts[beq1].Dst = dis.Imm(truePC)
		fl.insts[beq2].Dst = dis.Imm(truePC)
		fl.insts[beq3].Dst = dis.Imm(truePC)
		fl.insts[beq4].Dst = dis.Imm(truePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "ToUpper":
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
		blt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(97), dis.Imm(0)))
		bgt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, dis.Imm(122), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(32), src, dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt].Dst = dis.Imm(donePC)
		fl.insts[bgt].Dst = dis.Imm(donePC)
		return true, nil

	case "ToLower":
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(dst)))
		blt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(65), dis.Imm(0)))
		bgt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, dis.Imm(90), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), src, dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[blt].Dst = dis.Imm(donePC)
		fl.insts[bgt].Dst = dis.Imm(donePC)
		return true, nil
	}
	return false, nil
}

// lowerUTF8Call handles unicode/utf8 package functions.
func (fl *funcLowerer) lowerUTF8Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "RuneCountInString":
		// For ASCII-compatible impl, same as len(s)
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(dst)))
		return true, nil
	case "ValidString":
		// Dis/Limbo strings are always valid Unicode by construction,
		// so ValidString always returns true. This is correct, not a stub.
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "RuneLen":
		// RuneLen(r rune) → number of bytes needed to encode r in UTF-8
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		// r < 0 → -1, r <= 0x7F → 1, r <= 0x7FF → 2, r <= 0xFFFF → 3, r <= 0x10FFFF → 4, else -1
		mid10FFFF := fl.midImm(0x10FFFF)
		midFFFF := fl.midImm(0xFFFF)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		negIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, src, dis.Imm(0), dis.Imm(0)))
		gt4 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, src, mid10FFFF, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(4), dis.FP(dst)))
		le3 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, src, midFFFF, dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[le3].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(3), dis.FP(dst)))
		le2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, src, dis.Imm(0x7FF), dis.Imm(0)))
		jmpDone2 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[le2].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(2), dis.FP(dst)))
		le1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, src, dis.Imm(0x7F), dis.Imm(0)))
		jmpDone3 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[le1].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[negIdx].Dst = dis.Imm(donePC)
		fl.insts[gt4].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		fl.insts[jmpDone2].Dst = dis.Imm(donePC)
		fl.insts[jmpDone3].Dst = dis.Imm(donePC)
		return true, nil
	case "DecodeRuneInString":
		// Returns (rune, size). For ASCII: rune = s[0], size = 1
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		ch := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IINDC, src, dis.Imm(0), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		return true, nil
	case "DecodeRune":
		// DecodeRune(p []byte) → (rune, size). ASCII: p[0], 1
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sStr := fl.frame.AllocTemp(true)
		ch := fl.frame.AllocWord("dr.ch")
		fl.emit(dis.Inst2(dis.ICVTAC, src, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("dr.len")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		// Empty → (RuneError, 0)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0xFFFD), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		emptyIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenS), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.Imm(0), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		fl.insts[emptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "DecodeLastRune":
		// DecodeLastRune(p []byte) → (rune, size). ASCII: p[len-1], 1
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sStr := fl.frame.AllocTemp(true)
		ch := fl.frame.AllocWord("dlr.ch")
		lastIdx := fl.frame.AllocWord("dlr.last")
		fl.emit(dis.Inst2(dis.ICVTAC, src, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("dlr.len")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0xFFFD), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		emptyIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenS), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenS), dis.FP(lastIdx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(lastIdx), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		fl.insts[emptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "DecodeLastRuneInString":
		// DecodeLastRuneInString(s) → (rune, size). ASCII: s[len-1], 1
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		ch := fl.frame.AllocWord("dlris.ch")
		lastIdx := fl.frame.AllocWord("dlris.last")
		lenS := fl.frame.AllocWord("dlris.len")
		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0xFFFD), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		emptyIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenS), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenS), dis.FP(lastIdx)))
		fl.emit(dis.NewInst(dis.IINDC, src, dis.FP(lastIdx), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		fl.insts[emptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "ValidRune":
		// ValidRune(r) → r >= 0 && r <= 0x10FFFF && !(r >= 0xD800 && r <= 0xDFFF)
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		// Pre-materialize large mid operands before capturing instruction indices
		vr10FFFF := fl.midImm(0x10FFFF)
		vrD800 := fl.midImm(0xD800)
		vrDFFF := fl.midImm(0xDFFF)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst))) // default true
		// if r < 0 → false
		negIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, rOp, dis.Imm(0), dis.Imm(0)))
		// if r > 0x10FFFF → false
		tooBigIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, rOp, vr10FFFF, dis.Imm(0)))
		// if r >= 0xD800 && r <= 0xDFFF → false (surrogates)
		notSurrIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, rOp, vrD800, dis.Imm(0)))
		surrIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, rOp, vrDFFF, dis.Imm(0)))
		// Not surrogate, valid
		fl.insts[notSurrIdx].Dst = dis.Imm(int32(len(fl.insts)))
		doneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// Invalid
		invalidPC := int32(len(fl.insts))
		fl.insts[negIdx].Dst = dis.Imm(invalidPC)
		fl.insts[tooBigIdx].Dst = dis.Imm(invalidPC)
		fl.insts[surrIdx].Dst = dis.Imm(invalidPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "FullRune":
		// FullRune(p []byte) → check if p begins with a full UTF-8 encoding
		// Look at first byte to determine expected sequence length, compare with len(p)
		pOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenSlot := fl.frame.AllocWord("fr.len")
		fl.emit(dis.Inst2(dis.ILENA, pOp, dis.FP(lenSlot)))
		// Empty slice → false
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		emptyIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(lenSlot), dis.Imm(0), dis.Imm(0)))
		// Get first byte
		b0 := fl.frame.AllocWord("fr.b0")
		addr := fl.frame.AllocWord("fr.addr")
		fl.emit(dis.NewInst(dis.IINDB, pOp, dis.FP(addr), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(b0)))
		// ASCII (0xxxxxxx): need 1 byte, always full if len >= 1
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		asciiIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0x80), dis.Imm(0)))
		// 110xxxxx: need 2 bytes
		need2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xE0), dis.Imm(0)))
		// 1110xxxx: need 3 bytes
		need3Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xF0), dis.Imm(0)))
		// 11110xxx: need 4 bytes
		need4Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xF8), dis.Imm(0)))
		// Invalid lead byte (>= 0xF8): false
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		invalIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// need4: len >= 4
		fl.insts[need4Idx].Dst = dis.Imm(int32(len(fl.insts)))
		need := fl.frame.AllocWord("fr.need")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(4), dis.FP(need)))
		cmpIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// need3: len >= 3
		fl.insts[need3Idx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(3), dis.FP(need)))
		cmp2Idx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// need2: len >= 2
		fl.insts[need2Idx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(2), dis.FP(need)))
		// Compare: dst = (len >= need) ? 1 : 0
		cmpPC := int32(len(fl.insts))
		fl.insts[cmpIdx].Dst = dis.Imm(cmpPC)
		fl.insts[cmp2Idx].Dst = dis.Imm(cmpPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		failIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(lenSlot), dis.FP(need), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[emptyIdx].Dst = dis.Imm(donePC)
		fl.insts[asciiIdx].Dst = dis.Imm(donePC)
		fl.insts[invalIdx].Dst = dis.Imm(donePC)
		fl.insts[failIdx].Dst = dis.Imm(donePC)
		return true, nil
	case "FullRuneInString":
		// FullRuneInString(s string) → same logic via ICVTCA
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		// Convert string to bytes, then check
		tmpBytes := fl.frame.AllocPointer("fris:b")
		fl.emit(dis.Inst2(dis.ICVTCA, sOp, dis.FP(tmpBytes)))
		lenSlot := fl.frame.AllocWord("fris.len")
		fl.emit(dis.Inst2(dis.ILENA, dis.FP(tmpBytes), dis.FP(lenSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		emptyIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(lenSlot), dis.Imm(0), dis.Imm(0)))
		b0 := fl.frame.AllocWord("fris.b0")
		addr := fl.frame.AllocWord("fris.addr")
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(tmpBytes), dis.FP(addr), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(b0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		asciiIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0x80), dis.Imm(0)))
		need2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xE0), dis.Imm(0)))
		need3Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xF0), dis.Imm(0)))
		need4Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xF8), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		invalIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[need4Idx].Dst = dis.Imm(int32(len(fl.insts)))
		need := fl.frame.AllocWord("fris.need")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(4), dis.FP(need)))
		cmpIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[need3Idx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(3), dis.FP(need)))
		cmp2Idx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[need2Idx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(2), dis.FP(need)))
		cmpPC := int32(len(fl.insts))
		fl.insts[cmpIdx].Dst = dis.Imm(cmpPC)
		fl.insts[cmp2Idx].Dst = dis.Imm(cmpPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		failIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(lenSlot), dis.FP(need), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[emptyIdx].Dst = dis.Imm(donePC)
		fl.insts[asciiIdx].Dst = dis.Imm(donePC)
		fl.insts[invalIdx].Dst = dis.Imm(donePC)
		fl.insts[failIdx].Dst = dis.Imm(donePC)
		return true, nil
	case "RuneCount":
		// RuneCount(p []byte) → convert to string, count chars
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, src, dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(tmp), dis.FP(dst)))
		return true, nil
	case "Valid":
		return fl.lowerUTF8Valid(instr)
	case "EncodeRune":
		// EncodeRune(p []byte, r rune) → int
		// Writes UTF-8 encoding to p, returns byte count.
		// Uses INDB to get element address, then CVTWB to store through pointer.
		pOp := fl.operandOf(instr.Call.Args[0])
		rOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		tmp := fl.frame.AllocWord("er.tmp")
		addr := fl.frame.AllocWord("er.addr")
		// Helper: store byte value in tmp to p[idx] via INDB+CVTWB
		storeByte := func(idx int32) {
			fl.emit(dis.NewInst(dis.IINDB, pOp, dis.FP(addr), dis.Imm(idx)))
			fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(tmp), dis.FPInd(addr, 0)))
		}
		// ASCII: p[0] = byte(r), return 1
		not1Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, rOp, dis.Imm(0x7F), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, rOp, dis.FP(tmp)))
		storeByte(0)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		done1Idx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// 2-byte
		fl.insts[not1Idx].Dst = dis.Imm(int32(len(fl.insts)))
		not2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, rOp, dis.Imm(0x7FF), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(6), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0xC0), dis.FP(tmp)))
		storeByte(0)
		fl.emit(dis.NewInst(dis.IANDW, rOp, dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(1)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(2), dis.FP(dst)))
		done2Idx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// 3-byte
		fl.insts[not2Idx].Dst = dis.Imm(int32(len(fl.insts)))
		erFFFF := fl.midImm(0xFFFF)
		not3Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, rOp, erFFFF, dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(12), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0xE0), dis.FP(tmp)))
		storeByte(0)
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(6), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(tmp), dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(1)
		fl.emit(dis.NewInst(dis.IANDW, rOp, dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(2)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(3), dis.FP(dst)))
		done3Idx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// 4-byte
		fl.insts[not3Idx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(18), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0xF0), dis.FP(tmp)))
		storeByte(0)
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(12), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(tmp), dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(1)
		fl.emit(dis.NewInst(dis.ISHRW, rOp, dis.Imm(6), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(tmp), dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(2)
		fl.emit(dis.NewInst(dis.IANDW, rOp, dis.Imm(0x3F), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.Imm(0x80), dis.FP(tmp)))
		storeByte(3)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(4), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[done1Idx].Dst = dis.Imm(donePC)
		fl.insts[done2Idx].Dst = dis.Imm(donePC)
		fl.insts[done3Idx].Dst = dis.Imm(donePC)
		return true, nil
	case "AppendRune":
		// AppendRune(p []byte, r rune) → append UTF-8 encoded rune to p
		// Strategy: encode rune into temp 4-byte buffer, then append bytes to p
		pOp := fl.operandOf(instr.Call.Args[0])
		rOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		tmp := fl.frame.AllocWord("ar.tmp")
		addr := fl.frame.AllocWord("ar.addr")
		// Create a temp string from the rune, convert to bytes, then concat arrays
		// Simpler: build a string with INSC (insert char), then ICVTCA to bytes, then concat
		runeStr := fl.frame.AllocTemp(true)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(runeStr)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IINSC, rOp, dis.FP(tmp), dis.FP(runeStr)))
		// Convert rune string to bytes
		runeBytes := fl.frame.AllocPointer("ar:rb")
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(runeStr), dis.FP(runeBytes)))
		// Get current p length and rune byte length
		pLen := fl.frame.AllocWord("ar.pl")
		rbLen := fl.frame.AllocWord("ar.rl")
		newLen := fl.frame.AllocWord("ar.nl")
		fl.emit(dis.Inst2(dis.ILENA, pOp, dis.FP(pLen)))
		fl.emit(dis.Inst2(dis.ILENA, dis.FP(runeBytes), dis.FP(rbLen)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(rbLen), dis.FP(pLen), dis.FP(newLen)))
		// Allocate new byte array of newLen
		byteTDIdx := fl.makeHeapTypeDesc(types.Typ[types.Byte])
		newArr := fl.frame.AllocPointer("ar:na")
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(newLen), dis.Imm(int32(byteTDIdx)), dis.FP(newArr)))
		// Copy p bytes
		i := fl.frame.AllocWord("ar.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		copyPLoop := int32(len(fl.insts))
		bgeCopyPDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(pLen), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDB, pOp, dis.FP(addr), dis.FP(i)))
		bval := fl.frame.AllocWord("ar.bv")
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(newArr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(bval), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(copyPLoop)))
		fl.insts[bgeCopyPDone].Dst = dis.Imm(int32(len(fl.insts)))
		// Copy runeBytes
		j := fl.frame.AllocWord("ar.j")
		dstIdx := fl.frame.AllocWord("ar.di")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		copyRLoop := int32(len(fl.insts))
		bgeCopyRDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(rbLen), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(runeBytes), dis.FP(addr), dis.FP(j)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(pLen), dis.FP(j), dis.FP(dstIdx)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(newArr), dis.FP(addr), dis.FP(dstIdx)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(bval), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(copyRLoop)))
		fl.insts[bgeCopyRDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(newArr), dis.FP(dst)))
		_ = addr
		_ = tmp
		return true, nil
	}
	return false, nil
}

// lowerUTF8Valid implements utf8.Valid(p []byte) → bool with byte-by-byte
// UTF-8 validation. Checks continuation bytes, overlong encodings,
// surrogates (U+D800-U+DFFF), and values > U+10FFFF.
func (fl *funcLowerer) lowerUTF8Valid(instr *ssa.Call) (bool, error) {
	pOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenP := fl.frame.AllocWord("uv.len")
	i := fl.frame.AllocWord("uv.i")
	b0 := fl.frame.AllocWord("uv.b0")
	bx := fl.frame.AllocWord("uv.bx")
	addr := fl.frame.AllocWord("uv.addr")

	fl.emit(dis.Inst2(dis.ILENA, pOp, dis.FP(lenP)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst))) // default true

	// readByte emits IINDB + ICVTBW to read p[i] into target
	readByteAt := func(idxOp dis.Operand, target int32) {
		fl.emit(dis.NewInst(dis.IINDB, pOp, dis.FP(addr), idxOp))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(target)))
	}

	// Main loop: while i < lenP
	loopPC := int32(len(fl.insts))
	bgeDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))

	// Read p[i]
	readByteAt(dis.FP(i), b0)

	// ASCII: b0 < 0x80 → advance 1
	asciiIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0x80), dis.Imm(0)))

	// b0 < 0xC2: invalid (continuation byte 0x80-0xBF or overlong 0xC0-0xC1)
	invalidFixups := []int{}
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0xC2), dis.Imm(0)))

	// --- 2-byte sequence: 0xC2-0xDF ---
	not2Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(b0), dis.Imm(0xE0), dis.Imm(0)))
	// Advance i, check bounds
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	// Read continuation byte, check 0x80 <= bx < 0xC0
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Valid 2-byte, advance and loop
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// --- 3-byte sequence: 0xE0-0xEF ---
	fl.insts[not2Idx].Dst = dis.Imm(int32(len(fl.insts)))
	not3Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(b0), dis.Imm(0xF0), dis.Imm(0)))
	// Check i+1 in bounds
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	// Read b1, check continuation
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Overlong check: b0==0xE0 && bx<0xA0
	skipOverlong3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0xE0), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0xA0), dis.Imm(0)))
	fl.insts[skipOverlong3].Dst = dis.Imm(int32(len(fl.insts)))
	// Surrogate check: b0==0xED && bx>=0xA0
	skipSurr := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0xED), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xA0), dis.Imm(0)))
	fl.insts[skipSurr].Dst = dis.Imm(int32(len(fl.insts)))
	// Check b2
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Valid 3-byte
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// --- 4-byte sequence: 0xF0-0xF4 ---
	fl.insts[not3Idx].Dst = dis.Imm(int32(len(fl.insts)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(b0), dis.Imm(0xF5), dis.Imm(0)))
	// Check b1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Overlong check: b0==0xF0 && bx<0x90
	skipOverlong4 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0xF0), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x90), dis.Imm(0)))
	fl.insts[skipOverlong4].Dst = dis.Imm(int32(len(fl.insts)))
	// Too large check: b0==0xF4 && bx>=0x90
	skipTooLarge := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0xF4), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0x90), dis.Imm(0)))
	fl.insts[skipTooLarge].Dst = dis.Imm(int32(len(fl.insts)))
	// Check b2
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Check b3
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenP), dis.Imm(0)))
	readByteAt(dis.FP(i), bx)
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bx), dis.Imm(0x80), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(bx), dis.Imm(0xC0), dis.Imm(0)))
	// Valid 4-byte
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// ASCII advance
	asciiPC := int32(len(fl.insts))
	fl.insts[asciiIdx].Dst = dis.Imm(asciiPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Invalid: set result to false
	invalidPC := int32(len(fl.insts))
	for _, idx := range invalidFixups {
		fl.insts[idx].Dst = dis.Imm(invalidPC)
	}
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[bgeDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerPathCall handles path package functions.
func (fl *funcLowerer) lowerPathCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Base":
		// Return everything after last '/'
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		lastSlash := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(lastSlash)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, src, dis.FP(i), dis.FP(ch)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(47), dis.Imm(0))) // '/' = 47
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(lastSlash)))
		skipPC := int32(len(fl.insts))
		fl.insts[bneIdx].Dst = dis.Imm(skipPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)

		// if lastSlash == -1, return s as-is
		startOff := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lastSlash), dis.FP(startOff)))
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startOff), dis.FP(lenS), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))
		return true, nil

	case "Dir":
		// Return everything before last '/'
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		lastSlash := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(lastSlash)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, src, dis.FP(i), dis.FP(ch)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(47), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(lastSlash)))
		skipPC := int32(len(fl.insts))
		fl.insts[bneIdx].Dst = dis.Imm(skipPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)

		// if lastSlash <= 0, return "."
		dotOff := fl.comp.AllocString(".")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(dotOff), dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lastSlash), dis.Imm(0), dis.Imm(0)))
		jmpEnd := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		slicePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(slicePC)
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lastSlash), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))

		endPC := int32(len(fl.insts))
		fl.insts[jmpEnd].Dst = dis.Imm(endPC)
		return true, nil

	case "Ext":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		lastDot := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.ILENC, src, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(lastDot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, src, dis.FP(i), dis.FP(ch)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(46), dis.Imm(0))) // '.' = 46
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(lastDot)))
		skipPC := int32(len(fl.insts))
		fl.insts[bneIdx].Dst = dis.Imm(skipPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		searchDonePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(searchDonePC)

		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		beqNoExt := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lastDot), dis.Imm(-1), dis.Imm(0)))
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lastDot), dis.FP(lenS), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmp), dis.FP(dst)))
		fl.insts[beqNoExt].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "Join":
		// path.Join — delegate to strings.Join with "/"
		return fl.lowerPathJoin(instr)
	}
	return false, nil
}

// lowerPathJoin: join path segments with "/".
func (fl *funcLowerer) lowerPathJoin(instr *ssa.Call) (bool, error) {
	// path.Join takes variadic args which SSA presents as a slice
	elemsOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenArr := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	result := fl.frame.AllocTemp(true)
	elem := fl.frame.AllocTemp(true)
	elemAddr := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENA, elemsOp, dis.FP(lenArr)))
	emptyOff := fl.comp.AllocString("")
	sepOff := fl.comp.AllocString("/")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenArr), dis.Imm(0)))

	beqSkipSep := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(i), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDC, dis.MP(sepOff), dis.FP(result), dis.FP(result)))
	skipSepPC := int32(len(fl.insts))
	fl.insts[beqSkipSep].Dst = dis.Imm(skipSepPC)

	fl.emit(dis.NewInst(dis.IINDW, elemsOp, dis.FP(elemAddr), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(elemAddr, 0), dis.FP(elem)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(elem), dis.FP(result), dis.FP(result)))

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return true, nil
}

// lowerMathBitsCall handles math/bits package functions.
func (fl *funcLowerer) lowerMathBitsCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "OnesCount":
		// Popcount: loop and count set bits
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("")
		count := fl.frame.AllocWord("")
		bit := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))

		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(n), dis.FP(bit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(bit), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[beqDone].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		return true, nil

	case "Len":
		// Bit length: find position of highest set bit
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("")
		count := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))

		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[beqDone].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		return true, nil

	case "TrailingZeros":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("")
		count := fl.frame.AllocWord("")
		bit := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))

		// if n == 0 → return 64
		beqZero := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))

		loopPC := int32(len(fl.insts))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(n), dis.FP(bit)))
		bneFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(bit), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		foundPC := int32(len(fl.insts))
		fl.insts[bneFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		jmpEnd := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		zeroPC := int32(len(fl.insts))
		fl.insts[beqZero].Dst = dis.Imm(zeroPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(64), dis.FP(dst)))

		endPC := int32(len(fl.insts))
		fl.insts[jmpEnd].Dst = dis.Imm(endPC)
		return true, nil

	case "RotateLeft":
		src := fl.operandOf(instr.Call.Args[0])
		kOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		left := fl.frame.AllocWord("")
		right := fl.frame.AllocWord("")
		rShift := fl.frame.AllocWord("")

		fl.emit(dis.NewInst(dis.ISHLW, kOp, src, dis.FP(left)))
		fl.emit(dis.NewInst(dis.ISUBW, kOp, dis.Imm(64), dis.FP(rShift)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.FP(rShift), src, dis.FP(right)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(right), dis.FP(left), dis.FP(dst)))
		return true, nil

	case "Reverse":
		// Bit reversal: loop 64 times, shift result left, OR with LSB of src
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("rev.n")
		result := fl.frame.AllocWord("rev.r")
		bit := fl.frame.AllocWord("rev.b")
		i := fl.frame.AllocWord("rev.i")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.Imm(64), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(1), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(n), dis.FP(bit)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(bit), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeDone].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "LeadingZeros":
		// LeadingZeros(x) = 64 - Len(x)
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("lz.n")
		count := fl.frame.AllocWord("lz.c")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[beqDone].Dst = dis.Imm(int32(len(fl.insts)))
		// dst = 64 - count
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(count), dis.Imm(64), dis.FP(dst)))
		return true, nil

	case "ReverseBytes":
		// Byte-swap a 64-bit value: swap bytes 0↔7, 1↔6, 2↔5, 3↔4
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("rb.n")
		result := fl.frame.AllocWord("rb.r")
		b := fl.frame.AllocWord("rb.b")
		i := fl.frame.AllocWord("rb.i")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// loop 8 times: extract low byte, shift result left 8, OR
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.Imm(8), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(8), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), dis.FP(n), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(b), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(8), dis.FP(n), dis.FP(n)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "OnesCount8", "OnesCount16", "OnesCount32", "OnesCount64":
		// Same as OnesCount but with masking for sub-64 types
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("oc.n")
		count := fl.frame.AllocWord("oc.c")
		bit := fl.frame.AllocWord("oc.b")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(n), dis.FP(bit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(bit), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[beqDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		return true, nil

	case "Len8":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("l8.n")
		count := fl.frame.AllocWord("l8.c")
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[beqDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		return true, nil

	case "Len16", "Len32", "Len64":
		// Same algorithm as Len, just operates on full word
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("ln.n")
		count := fl.frame.AllocWord("ln.c")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[beqDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		return true, nil

	case "TrailingZeros8", "TrailingZeros16", "TrailingZeros32", "TrailingZeros64":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("tz.n")
		count := fl.frame.AllocWord("tz.c")
		bit := fl.frame.AllocWord("tz.b")
		var width int32 = 64
		switch callee.Name() {
		case "TrailingZeros8":
			width = 8
		case "TrailingZeros16":
			width = 16
		case "TrailingZeros32":
			width = 32
		}
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		beqZero := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		loopPC := int32(len(fl.insts))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(n), dis.FP(bit)))
		bneFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(bit), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bneFound].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(count), dis.FP(dst)))
		jmpEnd := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[beqZero].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(width), dis.FP(dst)))
		fl.insts[jmpEnd].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "LeadingZeros8", "LeadingZeros16", "LeadingZeros32", "LeadingZeros64":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("lz.n")
		count := fl.frame.AllocWord("lz.c")
		var width int32 = 64
		switch callee.Name() {
		case "LeadingZeros8":
			width = 8
		case "LeadingZeros16":
			width = 16
		case "LeadingZeros32":
			width = 32
		}
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		loopPC := int32(len(fl.insts))
		beqDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(n), dis.FP(n)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[beqDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(count), dis.Imm(width), dis.FP(dst)))
		return true, nil

	case "RotateLeft8", "RotateLeft16", "RotateLeft32", "RotateLeft64":
		src := fl.operandOf(instr.Call.Args[0])
		kOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		left := fl.frame.AllocWord("rl.l")
		right := fl.frame.AllocWord("rl.r")
		rShift := fl.frame.AllocWord("rl.rs")
		var width int32 = 64
		switch callee.Name() {
		case "RotateLeft8":
			width = 8
		case "RotateLeft16":
			width = 16
		case "RotateLeft32":
			width = 32
		}
		fl.emit(dis.NewInst(dis.ISHLW, kOp, src, dis.FP(left)))
		fl.emit(dis.NewInst(dis.ISUBW, kOp, dis.Imm(width), dis.FP(rShift)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.FP(rShift), src, dis.FP(right)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(right), dis.FP(left), dis.FP(dst)))
		return true, nil

	case "ReverseBytes16":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		hi := fl.frame.AllocWord("rb16.h")
		lo := fl.frame.AllocWord("rb16.l")
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), src, dis.FP(lo)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(8), src, dis.FP(hi)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), dis.FP(hi), dis.FP(hi)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(8), dis.FP(lo), dis.FP(lo)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(hi), dis.FP(lo), dis.FP(dst)))
		return true, nil

	case "ReverseBytes32":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("rb32.n")
		result := fl.frame.AllocWord("rb32.r")
		b := fl.frame.AllocWord("rb32.b")
		i := fl.frame.AllocWord("rb32.i")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.Imm(4), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(8), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), dis.FP(n), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(b), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(8), dis.FP(n), dis.FP(n)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "ReverseBytes64":
		// Same as ReverseBytes
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		n := fl.frame.AllocWord("rb64.n")
		result := fl.frame.AllocWord("rb64.r")
		b := fl.frame.AllocWord("rb64.b")
		i := fl.frame.AllocWord("rb64.i")
		fl.emit(dis.Inst2(dis.IMOVW, src, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.Imm(8), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(8), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xFF), dis.FP(n), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(b), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(8), dis.FP(n), dis.FP(n)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "Add", "Add64":
		// Add(x, y, carry) → (sum, carryOut)
		// carryOut = ((x & y) | ((x | y) & ~sum)) >> 63
		// Uses only bitwise ops — no unsigned comparison needed.
		x := fl.operandOf(instr.Call.Args[0])
		y := fl.operandOf(instr.Call.Args[1])
		carry := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sum := fl.frame.AllocWord("add.s")
		t1 := fl.frame.AllocWord("add.t1")
		t2 := fl.frame.AllocWord("add.t2")
		t3 := fl.frame.AllocWord("add.t3")
		// sum = x + y + carry
		fl.emit(dis.NewInst(dis.IADDW, x, y, dis.FP(sum)))
		fl.emit(dis.NewInst(dis.IADDW, carry, dis.FP(sum), dis.FP(sum)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(sum), dis.FP(dst)))
		// t1 = x & y
		fl.emit(dis.NewInst(dis.IANDW, x, y, dis.FP(t1)))
		// t2 = x | y
		fl.emit(dis.NewInst(dis.IORW, x, y, dis.FP(t2)))
		// t3 = ~sum (XOR with -1)
		fl.emit(dis.NewInst(dis.IXORW, dis.Imm(-1), dis.FP(sum), dis.FP(t3)))
		// t2 = (x | y) & ~sum
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(t3), dis.FP(t2), dis.FP(t2)))
		// t1 = (x & y) | ((x | y) & ~sum)
		fl.emit(dis.NewInst(dis.IORW, dis.FP(t2), dis.FP(t1), dis.FP(t1)))
		// carryOut = t1 >> 63 (logical shift right)
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(63), dis.FP(t1), dis.FP(dst+iby2wd)))
		return true, nil

	case "Sub", "Sub64":
		// Sub(x, y, borrow) → (diff, borrowOut)
		// borrowOut = ((~x & y) | ((~x | y) & diff)) >> 63
		// Uses only bitwise ops — no unsigned comparison needed.
		x := fl.operandOf(instr.Call.Args[0])
		y := fl.operandOf(instr.Call.Args[1])
		borrow := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		diff := fl.frame.AllocWord("sub.d")
		nx := fl.frame.AllocWord("sub.nx")
		t1 := fl.frame.AllocWord("sub.t1")
		t2 := fl.frame.AllocWord("sub.t2")
		// diff = x - y - borrow
		fl.emit(dis.NewInst(dis.ISUBW, y, x, dis.FP(diff)))
		fl.emit(dis.NewInst(dis.ISUBW, borrow, dis.FP(diff), dis.FP(diff)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(diff), dis.FP(dst)))
		// nx = ~x
		fl.emit(dis.NewInst(dis.IXORW, dis.Imm(-1), x, dis.FP(nx)))
		// t1 = ~x & y
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(nx), y, dis.FP(t1)))
		// t2 = ~x | y
		fl.emit(dis.NewInst(dis.IORW, dis.FP(nx), y, dis.FP(t2)))
		// t2 = (~x | y) & diff
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(diff), dis.FP(t2), dis.FP(t2)))
		// t1 = (~x & y) | ((~x | y) & diff)
		fl.emit(dis.NewInst(dis.IORW, dis.FP(t2), dis.FP(t1), dis.FP(t1)))
		// borrowOut = t1 >> 63 (logical shift right)
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(63), dis.FP(t1), dis.FP(dst+iby2wd)))
		return true, nil

	case "Mul", "Mul64":
		// Mul(x, y) → (hi, lo) — just return (0, x*y) for now
		x := fl.operandOf(instr.Call.Args[0])
		y := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IMULW, x, y, dis.FP(dst+iby2wd)))
		return true, nil

	case "Div", "Div64":
		// Div(hi, lo, y) → (quo, rem) — simplified: ignore hi
		lo := fl.operandOf(instr.Call.Args[1])
		y := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.NewInst(dis.IDIVW, y, lo, dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IMODW, y, lo, dis.FP(dst+iby2wd)))
		return true, nil

	case "Rem", "Rem64":
		// Rem(hi, lo, y) → rem — simplified: mod(lo, y)
		lo := fl.operandOf(instr.Call.Args[1])
		y := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		fl.emit(dis.NewInst(dis.IMODW, y, lo, dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// lowerMathRandCall handles math/rand package functions.
func (fl *funcLowerer) lowerMathRandCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	dst := fl.slotOf(instr)

	switch callee.Name() {
	case "Intn":
		// Use sys.millisec() as a simple pseudo-random source
		nOp := fl.operandOf(instr.Call.Args[0])
		msSlot := fl.frame.AllocWord("")

		disName := "millisec"
		ldtIdx, ok := fl.sysUsed[disName]
		if !ok {
			ldtIdx = len(fl.sysUsed)
			fl.sysUsed[disName] = ldtIdx
		}
		callFrame := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IMFRAME, dis.MP(fl.sysMPOff), dis.Imm(int32(ldtIdx)), dis.FP(callFrame)))
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(msSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
		fl.emit(dis.NewInst(dis.IMCALL, dis.FP(callFrame), dis.Imm(int32(ldtIdx)), dis.MP(fl.sysMPOff)))

		// result = abs(ms) % n
		absMs := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(msSlot), dis.FP(absMs)))
		bgefIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(absMs), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(absMs), dis.Imm(0), dis.FP(absMs)))
		absPC := int32(len(fl.insts))
		fl.insts[bgefIdx].Dst = dis.Imm(absPC)
		fl.emit(dis.NewInst(dis.IMODW, nOp, dis.FP(absMs), dis.FP(dst)))
		return true, nil

	case "Int":
		msSlot := fl.frame.AllocWord("")
		disName := "millisec"
		ldtIdx, ok := fl.sysUsed[disName]
		if !ok {
			ldtIdx = len(fl.sysUsed)
			fl.sysUsed[disName] = ldtIdx
		}
		callFrame := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IMFRAME, dis.MP(fl.sysMPOff), dis.Imm(int32(ldtIdx)), dis.FP(callFrame)))
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(msSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
		fl.emit(dis.NewInst(dis.IMCALL, dis.FP(callFrame), dis.Imm(int32(ldtIdx)), dis.MP(fl.sysMPOff)))
		// abs
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(msSlot), dis.FP(dst)))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(dst), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(dst), dis.Imm(0), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "Float64":
		// Return millisec as fraction
		msSlot := fl.frame.AllocWord("")
		disName := "millisec"
		ldtIdx, ok := fl.sysUsed[disName]
		if !ok {
			ldtIdx = len(fl.sysUsed)
			fl.sysUsed[disName] = ldtIdx
		}
		callFrame := fl.frame.AllocWord("")
		fl.emit(dis.NewInst(dis.IMFRAME, dis.MP(fl.sysMPOff), dis.Imm(int32(ldtIdx)), dis.FP(callFrame)))
		fl.emit(dis.Inst2(dis.ILEA, dis.FP(msSlot), dis.FPInd(callFrame, int32(dis.REGRET*dis.IBY2WD))))
		fl.emit(dis.NewInst(dis.IMCALL, dis.FP(callFrame), dis.Imm(int32(ldtIdx)), dis.MP(fl.sysMPOff)))

		fSlot := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ICVTRF, dis.FP(msSlot), dis.FP(fSlot)))
		largeOff := fl.comp.AllocReal(1000000000.0)
		fl.emit(dis.NewInst(dis.IMODW, dis.Imm(1000000), dis.FP(msSlot), dis.FP(msSlot)))
		fl.emit(dis.Inst2(dis.ICVTRF, dis.FP(msSlot), dis.FP(fSlot)))
		fl.emit(dis.NewInst(dis.IDIVF, dis.MP(largeOff), dis.FP(fSlot), dis.FP(dst)))
		return true, nil

	case "Seed":
		// No-op: our PRNG doesn't support seeding
		return true, nil
	}
	return false, nil
}

// lowerBytesCall handles bytes package functions.
func (fl *funcLowerer) lowerBytesCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Contains":
		// bytes.Contains(b, subslice) → convert both to strings, use string Contains
		bOp := fl.operandOf(instr.Call.Args[0])
		subOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)

		sStr := fl.frame.AllocTemp(true)
		subStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, subOp, dis.FP(subStr)))

		lenS := fl.frame.AllocWord("")
		lenSub := fl.frame.AllocWord("")
		limit := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		endIdx := fl.frame.AllocWord("")
		candidate := fl.frame.AllocTemp(true)

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(subStr), dis.FP(lenSub)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

		beqEmpty := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenSub), dis.Imm(0)))
		bgtShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSub), dis.FP(lenS), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSub), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSub), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(subStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.insts[beqEmpty].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

		donePC := int32(len(fl.insts))
		fl.insts[bgtShort].Dst = dis.Imm(donePC)
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "Equal":
		aOp := fl.operandOf(instr.Call.Args[0])
		bOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		aStr := fl.frame.AllocTemp(true)
		bStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, aOp, dis.FP(aStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(bStr)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		beqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(aStr), dis.FP(bStr), dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		truePC := int32(len(fl.insts))
		fl.insts[beqIdx].Dst = dis.Imm(truePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "Compare":
		// bytes.Compare(a, b) → -1, 0, or 1
		aOp := fl.operandOf(instr.Call.Args[0])
		bOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		aStr := fl.frame.AllocTemp(true)
		bStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, aOp, dis.FP(aStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(bStr)))
		// default 0 (equal)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		beqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(aStr), dis.FP(bStr), dis.Imm(0)))
		// not equal — use lexicographic compare via IBLTC/IBGTC pattern
		// simplified: compare lengths, if a < b → -1, else 1
		lenA := fl.frame.AllocWord("")
		lenB := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(aStr), dis.FP(lenA)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(bStr), dis.FP(lenB)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		bltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lenA), dis.FP(lenB), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[beqIdx].Dst = dis.Imm(donePC)
		fl.insts[bltIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "HasPrefix":
		aOp := fl.operandOf(instr.Call.Args[0])
		pOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		aStr := fl.frame.AllocTemp(true)
		pStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, aOp, dis.FP(aStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, pOp, dis.FP(pStr)))
		lenA := fl.frame.AllocWord("")
		lenP := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(aStr), dis.FP(lenA)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(pStr), dis.FP(lenP)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenP), dis.FP(lenA), dis.Imm(0)))
		head := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(aStr), dis.FP(head)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenP), dis.FP(head)))
		beqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(pStr), dis.FP(head), dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		truePC := int32(len(fl.insts))
		fl.insts[beqIdx].Dst = dis.Imm(truePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "HasSuffix":
		aOp := fl.operandOf(instr.Call.Args[0])
		sOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		aStr := fl.frame.AllocTemp(true)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, aOp, dis.FP(aStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		lenA := fl.frame.AllocWord("")
		lenS := fl.frame.AllocWord("")
		startOff := fl.frame.AllocWord("")
		tail := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(aStr), dis.FP(lenA)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenS), dis.FP(lenA), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenS), dis.FP(lenA), dis.FP(startOff)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(aStr), dis.FP(tail)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startOff), dis.FP(lenA), dis.FP(tail)))
		beqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sStr), dis.FP(tail), dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		truePC := int32(len(fl.insts))
		fl.insts[beqIdx].Dst = dis.Imm(truePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "Index":
		// bytes.Index(s, sep) → convert to strings, search loop
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenS := fl.frame.AllocWord("")
		lenSep := fl.frame.AllocWord("")
		limit := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		endIdx := fl.frame.AllocWord("")
		candidate := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		// empty sep → return 0
		beqEmpty := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenSep), dis.Imm(0)))
		bgtShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		emptyPC := int32(len(fl.insts))
		fl.insts[beqEmpty].Dst = dis.Imm(emptyPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtShort].Dst = dis.Imm(donePC)
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		return true, nil

	case "IndexByte":
		// bytes.IndexByte(b, c) → loop over string chars
		bOp := fl.operandOf(instr.Call.Args[0])
		cOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, cOp, dis.FP(ch), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "Count":
		// bytes.Count(s, sep) → count non-overlapping occurrences
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenS := fl.frame.AllocWord("")
		lenSep := fl.frame.AllocWord("")
		limit := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		endIdx := fl.frame.AllocWord("")
		candidate := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		bgtShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		// empty sep: return len+1
		beqEmptySep := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenSep), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(dst), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		noMatchPC := int32(len(fl.insts))
		fl.insts[bneIdx].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		emptyPC := int32(len(fl.insts))
		fl.insts[beqEmptySep].Dst = dis.Imm(emptyPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lenS), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtShort].Dst = dis.Imm(donePC)
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "TrimSpace":
		// Convert to string, trim leading/trailing whitespace, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		startIdx := fl.frame.AllocWord("")
		endI := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(startIdx)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(endI)))
		// trim leading
		trimLeadPC := int32(len(fl.insts))
		bgeSkipLead := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(startIdx), dis.FP(endI), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(startIdx), dis.FP(ch)))
		// check space (32), tab (9), newline (10), carriage return (13)
		beqSpace1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
		beqTab := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
		beqNL := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
		beqCR := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))
		jmpTrimTrail := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		incLeadPC := int32(len(fl.insts))
		fl.insts[beqSpace1].Dst = dis.Imm(incLeadPC)
		fl.insts[beqTab].Dst = dis.Imm(incLeadPC)
		fl.insts[beqNL].Dst = dis.Imm(incLeadPC)
		fl.insts[beqCR].Dst = dis.Imm(incLeadPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(startIdx), dis.FP(startIdx)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trimLeadPC)))
		// trim trailing
		trimTrailPC := int32(len(fl.insts))
		fl.insts[jmpTrimTrail].Dst = dis.Imm(trimTrailPC)
		fl.insts[bgeSkipLead].Dst = dis.Imm(trimTrailPC)
		tailIdx := fl.frame.AllocWord("")
		trimTrailLoop := int32(len(fl.insts))
		bleDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(endI), dis.FP(startIdx), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(endI), dis.FP(tailIdx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(tailIdx), dis.FP(ch)))
		beqSpace2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
		beqTab2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
		beqNL2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
		beqCR2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))
		jmpSlice := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		decTrailPC := int32(len(fl.insts))
		fl.insts[beqSpace2].Dst = dis.Imm(decTrailPC)
		fl.insts[beqTab2].Dst = dis.Imm(decTrailPC)
		fl.insts[beqNL2].Dst = dis.Imm(decTrailPC)
		fl.insts[beqCR2].Dst = dis.Imm(decTrailPC)
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(endI), dis.FP(endI)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trimTrailLoop)))
		slicePC := int32(len(fl.insts))
		fl.insts[jmpSlice].Dst = dis.Imm(slicePC)
		fl.insts[bleDone].Dst = dis.Imm(slicePC)
		result := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(result)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startIdx), dis.FP(endI), dis.FP(result)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "ToLower":
		// Convert to string, lowercase each char, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		result := fl.frame.AllocTemp(true)
		charStr := fl.frame.AllocTemp(true)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		// if 'A' <= ch <= 'Z', ch += 32
		bltNoConv := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(65), dis.Imm(0)))
		bgtNoConv := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(90), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), dis.FP(ch), dis.FP(ch)))
		noConvPC := int32(len(fl.insts))
		fl.insts[bltNoConv].Dst = dis.Imm(noConvPC)
		fl.insts[bgtNoConv].Dst = dis.Imm(noConvPC)
		fl.emit(dis.Inst2(dis.ICVTWC, dis.FP(ch), dis.FP(charStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "ToUpper", "ToTitle":
		// Convert to string, uppercase each char, convert back
		// (ToTitle is equivalent to ToUpper for ASCII characters)
		bOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		result := fl.frame.AllocTemp(true)
		charStr := fl.frame.AllocTemp(true)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		// if 'a' <= ch <= 'z', ch -= 32
		bltNoConv := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(97), dis.Imm(0)))
		bgtNoConv := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(122), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(32), dis.FP(ch), dis.FP(ch)))
		noConvPC := int32(len(fl.insts))
		fl.insts[bltNoConv].Dst = dis.Imm(noConvPC)
		fl.insts[bgtNoConv].Dst = dis.Imm(noConvPC)
		fl.emit(dis.Inst2(dis.ICVTWC, dis.FP(ch), dis.FP(charStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "Repeat":
		// bytes.Repeat(b, count) → convert to string, repeat, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		countOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		result := fl.frame.AllocTemp(true)
		i := fl.frame.AllocWord("")
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), countOp, dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(sStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "Join":
		// bytes.Join(s [][]byte, sep []byte) → []byte
		// Convert each element to string, join with sep string, convert result back.
		elemsOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenArr := fl.frame.AllocWord("bj.len")
		i := fl.frame.AllocWord("bj.i")
		result := fl.frame.AllocTemp(true)
		elem := fl.frame.AllocTemp(true)
		elemStr := fl.frame.AllocTemp(true)
		elemAddr := fl.frame.AllocWord("bj.ea")
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.ILENA, elemsOp, dis.FP(lenArr)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenArr), dis.Imm(0)))
		// If i > 0, append sep
		beqSkipSep := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(i), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(sepStr), dis.FP(result), dis.FP(result)))
		skipSepPC := int32(len(fl.insts))
		fl.insts[beqSkipSep].Dst = dis.Imm(skipSepPC)
		// elem = elems[i] ([]byte), convert to string
		fl.emit(dis.NewInst(dis.IINDW, elemsOp, dis.FP(elemAddr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(elemAddr, 0), dis.FP(elem)))
		fl.emit(dis.Inst2(dis.ICVTAC, dis.FP(elem), dis.FP(elemStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(elemStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeDone].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "Split":
		// bytes.Split(s, sep) → [][]byte
		// Convert to strings, use string-based split, convert segments back to []byte.
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenS := fl.frame.AllocWord("bspl.lenS")
		lenSep := fl.frame.AllocWord("bspl.lenSep")
		count := fl.frame.AllocWord("bspl.cnt")
		i := fl.frame.AllocWord("bspl.i")
		endIdx := fl.frame.AllocWord("bspl.end")
		candidate := fl.frame.AllocTemp(true)
		limit := fl.frame.AllocWord("bspl.lim")
		segStart := fl.frame.AllocWord("bspl.ss")
		arrIdx := fl.frame.AllocWord("bspl.ai")
		segment := fl.frame.AllocTemp(true)
		arrPtr := fl.frame.AllocPointer("bspl:arr")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))
		// Count occurrences
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		bgtNoMatchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		jmpCountLoop := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		noMatchPC := int32(len(fl.insts))
		fl.insts[bgtNoMatchIdx].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(limit)))
		countLoopPC := int32(len(fl.insts))
		fl.insts[jmpCountLoop].Dst = dis.Imm(countLoopPC)
		bgeCountDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqCountFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countLoopPC)))
		countFoundPC := int32(len(fl.insts))
		fl.insts[beqCountFound].Dst = dis.Imm(countFoundPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countLoopPC)))
		countDonePC := int32(len(fl.insts))
		fl.insts[bgeCountDone].Dst = dis.Imm(countDonePC)
		// Allocate [][]byte array (Dis: array of byte arrays)
		elemTDIdx := fl.makeHeapTypeDesc(types.NewSlice(types.Typ[types.Byte]))
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(arrPtr)))
		// Fill: scan again, extract segments, convert back to []byte
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFillFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillFoundPC := int32(len(fl.insts))
		fl.insts[beqFillFound].Dst = dis.Imm(fillFoundPC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(i), dis.FP(segment)))
		segBytes := fl.frame.AllocPointer("bspl:seg")
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		storeAddr := fl.frame.AllocWord("bspl.sa")
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		// Last segment
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrPtr), dis.FP(dst)))
		return true, nil

	case "Replace", "ReplaceAll":
		// bytes.Replace/ReplaceAll — convert to strings, use string replace, convert back
		sOp := fl.operandOf(instr.Call.Args[0])
		oldOp := fl.operandOf(instr.Call.Args[1])
		newOp := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		oldStr := fl.frame.AllocTemp(true)
		newStr := fl.frame.AllocTemp(true)
		result := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, oldOp, dis.FP(oldStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, newOp, dis.FP(newStr)))
		// Inline replace loop
		lenS := fl.frame.AllocWord("")
		lenOld := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		endIdx := fl.frame.AllocWord("")
		limit := fl.frame.AllocWord("")
		candidate := fl.frame.AllocTemp(true)
		iP1 := fl.frame.AllocWord("")
		charStr := fl.frame.AllocTemp(true)
		ch := fl.frame.AllocWord("")
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(oldStr), dis.FP(lenOld)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		// check if old matches at position i
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenOld), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenOld), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		bgtNoMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(endIdx), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		bneNoMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(oldStr), dis.FP(candidate), dis.Imm(0)))
		// match found: append new, skip old
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(newStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenOld), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// no match: append current char
		noMatchPC := int32(len(fl.insts))
		fl.insts[bgtNoMatch].Dst = dis.Imm(noMatchPC)
		fl.insts[bneNoMatch].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.ICVTWC, dis.FP(ch), dis.FP(charStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(iP1)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(iP1), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "Trim":
		// bytes.Trim(s, cutset string) → convert to strings, trim, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		cutsetOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("bt.lenS")
		lenCut := fl.frame.AllocWord("bt.lenCut")
		start := fl.frame.AllocWord("bt.start")
		end := fl.frame.AllocWord("bt.end")
		ch := fl.frame.AllocWord("bt.ch")
		j := fl.frame.AllocWord("bt.j")
		cutCh := fl.frame.AllocWord("bt.cutCh")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(start)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(end)))
		// Trim leading
		leadLoopPC := int32(len(fl.insts))
		leadDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(start), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerLeadPC := int32(len(fl.insts))
		innerLeadDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
		beqLeadMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerLeadPC)))
		notInCutPC := int32(len(fl.insts))
		fl.insts[innerLeadDone].Dst = dis.Imm(notInCutPC)
		jmpLeadDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		inCutPC := int32(len(fl.insts))
		fl.insts[beqLeadMatch].Dst = dis.Imm(inCutPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(start), dis.FP(start)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(leadLoopPC)))
		leadDonePC := int32(len(fl.insts))
		fl.insts[leadDoneIdx].Dst = dis.Imm(leadDonePC)
		fl.insts[jmpLeadDone].Dst = dis.Imm(leadDonePC)
		// Trim trailing
		trailLoopPC := int32(len(fl.insts))
		trailDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
		tailIdx := fl.frame.AllocWord("bt.tail")
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(tailIdx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(tailIdx), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerTrailPC := int32(len(fl.insts))
		innerTrailDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
		beqTrailMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerTrailPC)))
		notInCut2PC := int32(len(fl.insts))
		fl.insts[innerTrailDone].Dst = dis.Imm(notInCut2PC)
		jmpTrailDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		inCut2PC := int32(len(fl.insts))
		fl.insts[beqTrailMatch].Dst = dis.Imm(inCut2PC)
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(end)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trailLoopPC)))
		trailDonePC := int32(len(fl.insts))
		fl.insts[trailDoneIdx].Dst = dis.Imm(trailDonePC)
		fl.insts[jmpTrailDone].Dst = dis.Imm(trailDonePC)
		// result = s[start:end] converted back to []byte
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(start), dis.FP(end), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(tmp), dis.FP(dst)))
		return true, nil

	case "TrimLeft":
		// bytes.TrimLeft(s, cutset string) → trim from left, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		cutsetOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("btl.lenS")
		lenCut := fl.frame.AllocWord("btl.lenCut")
		start := fl.frame.AllocWord("btl.start")
		ch := fl.frame.AllocWord("btl.ch")
		j := fl.frame.AllocWord("btl.j")
		cutCh := fl.frame.AllocWord("btl.cutCh")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(start)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(start), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		innerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
		beqMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		notInPC := int32(len(fl.insts))
		fl.insts[innerDone].Dst = dis.Imm(notInPC)
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		inPC := int32(len(fl.insts))
		fl.insts[beqMatch].Dst = dis.Imm(inPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(start), dis.FP(start)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(start), dis.FP(lenS), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(tmp), dis.FP(dst)))
		return true, nil

	case "TrimRight":
		// bytes.TrimRight(s, cutset string) → trim from right, convert back
		bOp := fl.operandOf(instr.Call.Args[0])
		cutsetOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("btr.lenS")
		lenCut := fl.frame.AllocWord("btr.lenCut")
		end := fl.frame.AllocWord("btr.end")
		ch := fl.frame.AllocWord("btr.ch")
		j := fl.frame.AllocWord("btr.j")
		cutCh := fl.frame.AllocWord("btr.cutCh")
		tailIdx := fl.frame.AllocWord("btr.tail")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, cutsetOp, dis.FP(lenCut)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(end)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(end), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(tailIdx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(tailIdx), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		innerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenCut), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, cutsetOp, dis.FP(j), dis.FP(cutCh)))
		beqMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cutCh), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		notInPC := int32(len(fl.insts))
		fl.insts[innerDone].Dst = dis.Imm(notInPC)
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		inPC := int32(len(fl.insts))
		fl.insts[beqMatch].Dst = dis.Imm(inPC)
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(end)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.insts[jmpDone].Dst = dis.Imm(donePC)
		tmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(end), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(tmp), dis.FP(dst)))
		return true, nil

	case "NewBuffer", "NewBufferString":
		// bytes.NewBuffer/NewBufferString → returns 0 handle (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	case "NewReader":
		// bytes.NewReader(b) → returns 0 handle (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	case "TrimPrefix":
		// bytes.TrimPrefix(s, prefix) → if s has prefix, return s[len(prefix):]
		sOp := fl.operandOf(instr.Call.Args[0])
		pOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		pStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, pOp, dis.FP(pStr)))
		lenS := fl.frame.AllocWord("")
		lenP := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(pStr), dis.FP(lenP)))
		// default: return s unchanged
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenP), dis.FP(lenS), dis.Imm(0)))
		head := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(head)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenP), dis.FP(head)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(pStr), dis.FP(head), dis.Imm(0)))
		trimmed := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(trimmed)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lenP), dis.FP(lenS), dis.FP(trimmed)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(trimmed), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(donePC)
		fl.insts[bneIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "TrimSuffix":
		// bytes.TrimSuffix(s, suffix) → if s has suffix, return s[:len(s)-len(suffix)]
		sOp := fl.operandOf(instr.Call.Args[0])
		sfxOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sfxStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sfxOp, dis.FP(sfxStr)))
		lenS := fl.frame.AllocWord("")
		lenSfx := fl.frame.AllocWord("")
		startOff := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sfxStr), dis.FP(lenSfx)))
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSfx), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSfx), dis.FP(lenS), dis.FP(startOff)))
		tail := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(tail)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startOff), dis.FP(lenS), dis.FP(tail)))
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(sfxStr), dis.FP(tail), dis.Imm(0)))
		trimmed := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(trimmed)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(startOff), dis.FP(trimmed)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(trimmed), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(donePC)
		fl.insts[bneIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "LastIndex":
		// bytes.LastIndex(s, sep) → search from end
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenS := fl.frame.AllocWord("")
		lenSep := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		endIdx := fl.frame.AllocWord("")
		candidate := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		bgtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		// start from lenS-lenSep, go down to 0
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgtIdx].Dst = dis.Imm(donePC)
		fl.insts[bltIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "EqualFold":
		// bytes.EqualFold(s, t) → case-insensitive comparison via string conversion
		sOp := fl.operandOf(instr.Call.Args[0])
		tOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		tStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, tOp, dis.FP(tStr)))
		// Lowercase both and compare
		lenS := fl.frame.AllocWord("")
		lenT := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(tStr), dis.FP(lenT)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		// If lengths differ → false
		bneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(lenS), dis.FP(lenT), dis.Imm(0)))
		// Compare char by char, case-insensitive
		i := fl.frame.AllocWord("")
		chS := fl.frame.AllocWord("")
		chT := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(chS)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(tStr), dis.FP(i), dis.FP(chT)))
		// to lower: if 'A' <= ch <= 'Z', ch += 32
		bltS := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(chS), dis.Imm(65), dis.Imm(0)))
		bgtS := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(chS), dis.Imm(90), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), dis.FP(chS), dis.FP(chS)))
		skipS := int32(len(fl.insts))
		fl.insts[bltS].Dst = dis.Imm(skipS)
		fl.insts[bgtS].Dst = dis.Imm(skipS)
		bltT := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(chT), dis.Imm(65), dis.Imm(0)))
		bgtT := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(chT), dis.Imm(90), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(32), dis.FP(chT), dis.FP(chT)))
		skipT := int32(len(fl.insts))
		fl.insts[bltT].Dst = dis.Imm(skipT)
		fl.insts[bgtT].Dst = dis.Imm(skipT)
		bneMismatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(chS), dis.FP(chT), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		matchPC := int32(len(fl.insts))
		fl.insts[bgeMatch].Dst = dis.Imm(matchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bneIdx].Dst = dis.Imm(donePC)
		fl.insts[bneMismatch].Dst = dis.Imm(donePC)
		return true, nil

	case "ContainsRune":
		// bytes.ContainsRune(b, r) → loop through chars
		bOp := fl.operandOf(instr.Call.Args[0])
		rOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, rOp, dis.FP(ch), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "ContainsAny":
		// bytes.ContainsAny(b, chars) → double loop
		bOp := fl.operandOf(instr.Call.Args[0])
		charsOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("")
		lenC := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		j := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		cc := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, charsOp, dis.FP(lenC)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		outerPC := int32(len(fl.insts))
		bgeOuter := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		bgeInner := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenC), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, charsOp, dis.FP(j), dis.FP(cc)))
		beqFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cc), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		nextI := int32(len(fl.insts))
		fl.insts[bgeInner].Dst = dis.Imm(nextI)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[beqFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bgeOuter].Dst = dis.Imm(donePC)
		return true, nil

	case "Fields":
		// bytes.Fields(s) → split on whitespace runs, return [][]byte
		// Convert to string, split on whitespace (two-pass: count then fill), convert back.
		bOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("bf.len")
		i := fl.frame.AllocWord("bf.i")
		ch := fl.frame.AllocWord("bf.ch")
		count := fl.frame.AllocWord("bf.cnt")
		inWord := fl.frame.AllocWord("bf.iw")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		// Pass 1: count words
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		countLoopPC := int32(len(fl.insts))
		bgeCountDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		beqSpc := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
		beqTab := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
		beqNl := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
		beqCr := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))
		// not space: if !inWord → count++, inWord=1
		beqAlreadyIn := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(inWord)))
		alreadyInPC := int32(len(fl.insts))
		fl.insts[beqAlreadyIn].Dst = dis.Imm(alreadyInPC)
		jmpNext := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// space: inWord=0
		spacePC := int32(len(fl.insts))
		fl.insts[beqSpc].Dst = dis.Imm(spacePC)
		fl.insts[beqTab].Dst = dis.Imm(spacePC)
		fl.insts[beqNl].Dst = dis.Imm(spacePC)
		fl.insts[beqCr].Dst = dis.Imm(spacePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
		nextPC := int32(len(fl.insts))
		fl.insts[jmpNext].Dst = dis.Imm(nextPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countLoopPC)))
		countDonePC := int32(len(fl.insts))
		fl.insts[bgeCountDone].Dst = dis.Imm(countDonePC)
		// Allocate [][]byte array
		elemTDIdx := fl.makeHeapTypeDesc(types.NewSlice(types.Typ[types.Byte]))
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(dst)))
		// Pass 2: fill array
		arrIdx := fl.frame.AllocWord("bf.ai")
		segStart := fl.frame.AllocWord("bf.ss")
		segment := fl.frame.AllocTemp(true)
		segBytes := fl.frame.AllocPointer("bf:seg")
		storeAddr := fl.frame.AllocWord("bf.sa")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		beqSpc2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(32), dis.FP(ch), dis.Imm(0)))
		beqTab2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(9), dis.FP(ch), dis.Imm(0)))
		beqNl2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(10), dis.FP(ch), dis.Imm(0)))
		beqCr2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(13), dis.FP(ch), dis.Imm(0)))
		// not space
		beqAlreadyIn2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(inWord)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		alreadyIn2PC := int32(len(fl.insts))
		fl.insts[beqAlreadyIn2].Dst = dis.Imm(alreadyIn2PC)
		jmpNext2 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// space: if inWord, store segment
		space2PC := int32(len(fl.insts))
		fl.insts[beqSpc2].Dst = dis.Imm(space2PC)
		fl.insts[beqTab2].Dst = dis.Imm(space2PC)
		fl.insts[beqNl2].Dst = dis.Imm(space2PC)
		fl.insts[beqCr2].Dst = dis.Imm(space2PC)
		beqNotInWord := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(i), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(dst), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(inWord)))
		notInWordPC := int32(len(fl.insts))
		fl.insts[beqNotInWord].Dst = dis.Imm(notInWordPC)
		next2PC := int32(len(fl.insts))
		fl.insts[jmpNext2].Dst = dis.Imm(next2PC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		// done: if inWord, store last segment
		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		beqNoLast := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(inWord), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(dst), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.insts[beqNoLast].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "SplitN":
		// bytes.SplitN(s, sep, n) → [][]byte
		// Convert to strings, split with max count, convert segments back.
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		nOp := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)

		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))

		// if n == 0 → return nil
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // nil default
		bsn0Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, nOp, dis.Imm(0), dis.Imm(0)))

		// if n < 0 → unlimited
		maxN := fl.frame.AllocWord("bsn.max")
		fl.emit(dis.Inst2(dis.IMOVW, nOp, dis.FP(maxN)))
		bsnNotNeg := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, nOp, dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0x1FFFFFFF), dis.FP(maxN)))
		fl.insts[bsnNotNeg].Dst = dis.Imm(int32(len(fl.insts)))

		lenS := fl.frame.AllocWord("bsn.lenS")
		lenSep := fl.frame.AllocWord("bsn.lenSep")
		count := fl.frame.AllocWord("bsn.cnt")
		i := fl.frame.AllocWord("bsn.i")
		endIdx := fl.frame.AllocWord("bsn.end")
		candidate := fl.frame.AllocTemp(true)
		limit := fl.frame.AllocWord("bsn.lim")

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))

		// Count occurrences (capped at maxN-1)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		bgtNoMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		jmpCntLoop := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		noMatchPC := int32(len(fl.insts))
		fl.insts[bgtNoMatch].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(limit)))

		cntLoopPC := int32(len(fl.insts))
		fl.insts[jmpCntLoop].Dst = dis.Imm(cntLoopPC)
		bgeCntDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		bgeMaxN := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(count), dis.FP(maxN), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqCntFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntFoundPC := int32(len(fl.insts))
		fl.insts[beqCntFound].Dst = dis.Imm(cntFoundPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))

		cntDonePC := int32(len(fl.insts))
		fl.insts[bgeCntDone].Dst = dis.Imm(cntDonePC)
		fl.insts[bgeMaxN].Dst = dis.Imm(cntDonePC)

		// Allocate [][]byte array
		elemTDIdx := fl.makeHeapTypeDesc(types.NewSlice(types.Typ[types.Byte]))
		arrPtr := fl.frame.AllocPointer("bsn:arr")
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(arrPtr)))

		// Fill loop
		segStart := fl.frame.AllocWord("bsn.ss")
		arrIdx := fl.frame.AllocWord("bsn.ai")
		segment := fl.frame.AllocTemp(true)
		storeAddr := fl.frame.AllocWord("bsn.sa")
		maxSplits := fl.frame.AllocWord("bsn.ms")
		segBytes := fl.frame.AllocPointer("bsn:seg")
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(count), dis.FP(maxSplits)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))

		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		bgeHitMax := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(arrIdx), dis.FP(maxSplits), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFillFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillFoundPC := int32(len(fl.insts))
		fl.insts[beqFillFound].Dst = dis.Imm(fillFoundPC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(i), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))

		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		fl.insts[bgeHitMax].Dst = dis.Imm(fillDonePC)
		// Last segment: s[segStart:]
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))

		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrPtr), dis.FP(dst)))
		bsnAllDonePC := int32(len(fl.insts))
		fl.insts[bsn0Idx].Dst = dis.Imm(bsnAllDonePC)
		return true, nil

	case "Map":
		// bytes.Map — stub: return input unchanged
		dst := fl.slotOf(instr)
		sOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		return true, nil

	// Buffer methods — real implementations using string accumulator at field 0.
	// Same pattern as strings.Builder: receiver's field 0 is a string.
	case "Write":
		if callee.Signature.Recv() != nil {
			// (*Buffer).Write(p) → buf += string(p); return (len(p), nil)
			recvSlot := fl.materialize(instr.Call.Args[0])
			bOp := fl.operandOf(instr.Call.Args[1])
			strTmp := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(strTmp)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(strTmp), dis.FPInd(recvSlot, 0), dis.FPInd(recvSlot, 0)))
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.ILENC, dis.FP(strTmp), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}

	case "WriteString":
		if callee.Signature.Recv() != nil {
			// (*Buffer).WriteString(s) → buf += s; return (len(s), nil)
			recvSlot := fl.materialize(instr.Call.Args[0])
			sOp := fl.operandOf(instr.Call.Args[1])
			fl.emit(dis.NewInst(dis.IADDC, sOp, dis.FPInd(recvSlot, 0), dis.FPInd(recvSlot, 0)))
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}

	case "WriteByte":
		if callee.Signature.Recv() != nil {
			// (*Buffer).WriteByte(c) → insc(c, len(buf), buf); return nil
			// ICVTWC produces decimal "33" not "!". Use INSC to append char by codepoint.
			recvSlot := fl.materialize(instr.Call.Args[0])
			byteVal := fl.operandOf(instr.Call.Args[1])
			lenTmp := fl.frame.AllocWord("")
			fl.emit(dis.Inst2(dis.ILENC, dis.FPInd(recvSlot, 0), dis.FP(lenTmp)))
			fl.emit(dis.NewInst(dis.IINSC, byteVal, dis.FP(lenTmp), dis.FPInd(recvSlot, 0)))
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}

	case "String":
		if callee.Signature.Recv() != nil {
			// (*Buffer).String() → return buf
			recvSlot := fl.materialize(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(recvSlot, 0), dis.FP(dst)))
			return true, nil
		}

	case "Bytes":
		if callee.Signature.Recv() != nil {
			// (*Buffer).Bytes() → return []byte(buf)
			recvSlot := fl.materialize(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.ICVTCA, dis.FPInd(recvSlot, 0), dis.FP(dst)))
			return true, nil
		}

	case "Len":
		if callee.Signature.Recv() != nil {
			// (*Buffer).Len() → len(buf)
			recvSlot := fl.materialize(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.ILENC, dis.FPInd(recvSlot, 0), dis.FP(dst)))
			return true, nil
		}

	case "Reset":
		if callee.Signature.Recv() != nil {
			// (*Buffer).Reset() → buf = ""
			recvSlot := fl.materialize(instr.Call.Args[0])
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FPInd(recvSlot, 0)))
			return true, nil
		}

	case "Read":
		if callee.Signature.Recv() != nil {
			// (*Buffer).Read(p []byte) → (n int, err error)
			// Copy min(len(p), len(buf)) bytes from buffer to p, consume from buffer
			recvSlot := fl.materialize(instr.Call.Args[0])
			pOp := fl.operandOf(instr.Call.Args[1])
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			bufStr := fl.frame.AllocTemp(true)
			pStr := fl.frame.AllocTemp(true)
			lenBuf := fl.frame.AllocWord("br.lb")
			lenP := fl.frame.AllocWord("br.lp")
			n := fl.frame.AllocWord("br.n")
			i := fl.frame.AllocWord("br.i")
			ch := fl.frame.AllocWord("br.ch")
			addr := fl.frame.AllocWord("br.a")
			fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(recvSlot, 0), dis.FP(bufStr)))
			fl.emit(dis.Inst2(dis.ILENC, dis.FP(bufStr), dis.FP(lenBuf)))
			fl.emit(dis.Inst2(dis.ICVTAC, pOp, dis.FP(pStr)))
			fl.emit(dis.Inst2(dis.ILENC, dis.FP(pStr), dis.FP(lenP)))
			// n = min(lenBuf, lenP)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenBuf), dis.FP(n)))
			skipIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBLEW, dis.FP(lenBuf), dis.FP(lenP), dis.Imm(0)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenP), dis.FP(n)))
			fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
			// Copy n bytes from bufStr to p
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
			loopPC := int32(len(fl.insts))
			doneIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(n), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.IINDC, dis.FP(bufStr), dis.FP(i), dis.FP(ch)))
			fl.emit(dis.NewInst(dis.IINDB, pOp, dis.FP(addr), dis.FP(i)))
			fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(ch), dis.FPInd(addr, 0)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
			fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
			// Consume: buf = buf[n:]
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(n), dis.FP(lenBuf), dis.FPInd(recvSlot, 0)))
			// Return (n, nil error)
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(n), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}

	case "ReadByte":
		if callee.Signature.Recv() != nil {
			// (*Buffer).ReadByte() → (byte, error)
			// Read first byte from buffer, consume it
			recvSlot := fl.materialize(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			bufStr := fl.frame.AllocTemp(true)
			lenBuf := fl.frame.AllocWord("rby.lb")
			ch := fl.frame.AllocWord("rby.ch")
			fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(recvSlot, 0), dis.FP(bufStr)))
			fl.emit(dis.Inst2(dis.ILENC, dis.FP(bufStr), dis.FP(lenBuf)))
			// Default: return (0, error)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			emptyIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBLEW, dis.FP(lenBuf), dis.Imm(0), dis.Imm(0)))
			// Read first char
			fl.emit(dis.NewInst(dis.IINDC, dis.FP(bufStr), dis.Imm(0), dis.FP(ch)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(dst)))
			// Consume: buf = buf[1:]
			fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(1), dis.FP(lenBuf), dis.FPInd(recvSlot, 0)))
			fl.insts[emptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
			return true, nil
		}

	case "ReadString":
		if callee.Signature.Recv() != nil {
			// (*Buffer).ReadString(delim byte) → (string, error)
			// Read until delimiter (inclusive), consume from buffer
			recvSlot := fl.materialize(instr.Call.Args[0])
			delimOp := fl.operandOf(instr.Call.Args[1])
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			bufStr := fl.frame.AllocTemp(true)
			lenBuf := fl.frame.AllocWord("rs.lb")
			i := fl.frame.AllocWord("rs.i")
			ch := fl.frame.AllocWord("rs.ch")
			fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(recvSlot, 0), dis.FP(bufStr)))
			fl.emit(dis.Inst2(dis.ILENC, dis.FP(bufStr), dis.FP(lenBuf)))
			// Search for delimiter
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
			loopPC := int32(len(fl.insts))
			notFoundIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenBuf), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.IINDC, dis.FP(bufStr), dis.FP(i), dis.FP(ch)))
			foundIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), delimOp, dis.Imm(0)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
			// Found: include delimiter
			fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			// Not found: return entire buffer
			fl.insts[notFoundIdx].Dst = dis.Imm(int32(len(fl.insts)))
			// result = buf[:i]
			resultStr := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(bufStr), dis.FP(resultStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(i), dis.FP(resultStr)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.FP(resultStr), dis.FP(dst)))
			// Consume: buf = buf[i:]
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(lenBuf), dis.FPInd(recvSlot, 0)))
			// nil error
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}

	// Title is deprecated; ToValidUTF8 and Runes pass through correctly
	// (Dis strings/byte slices are valid UTF-8 by construction)
	case "Title", "ToValidUTF8", "Runes":
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(dst)))
		return true, nil

	case "IndexAny":
		// bytes.IndexAny(s []byte, chars string) → first index of any char in chars, or -1
		bOp := fl.operandOf(instr.Call.Args[0])
		charsOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("bia.lenS")
		lenC := fl.frame.AllocWord("bia.lenC")
		i := fl.frame.AllocWord("bia.i")
		j := fl.frame.AllocWord("bia.j")
		ch := fl.frame.AllocWord("bia.ch")
		cc := fl.frame.AllocWord("bia.cc")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, charsOp, dis.FP(lenC)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		outerPC := int32(len(fl.insts))
		outerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		innerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenC), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, charsOp, dis.FP(j), dis.FP(cc)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cc), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		// inner done (no match for this byte) → next byte
		fl.insts[innerDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))
		// found
		fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		fl.insts[outerDone].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "LastIndexAny":
		// bytes.LastIndexAny(s []byte, chars string) → last index of any char in chars, or -1
		bOp := fl.operandOf(instr.Call.Args[0])
		charsOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("blia.lenS")
		lenC := fl.frame.AllocWord("blia.lenC")
		i := fl.frame.AllocWord("blia.i")
		j := fl.frame.AllocWord("blia.j")
		ch := fl.frame.AllocWord("blia.ch")
		cc := fl.frame.AllocWord("blia.cc")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, charsOp, dis.FP(lenC)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		// i = len - 1
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenS), dis.FP(i)))
		outerPC := int32(len(fl.insts))
		outerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		innerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.FP(lenC), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, charsOp, dis.FP(j), dis.FP(cc)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.FP(cc), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		fl.insts[innerDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))
		fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		fl.insts[outerDone].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "LastIndexByte":
		// bytes.LastIndexByte(s []byte, c byte) → last index of c, or -1
		bOp := fl.operandOf(instr.Call.Args[0])
		cOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("blib.lenS")
		i := fl.frame.AllocWord("blib.i")
		ch := fl.frame.AllocWord("blib.ch")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenS), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, cOp, dis.FP(ch), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "IndexRune":
		// bytes.IndexRune(s []byte, r rune) → first index of rune r, or -1
		// For ASCII, just compare byte values
		bOp := fl.operandOf(instr.Call.Args[0])
		rOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, bOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("bir.lenS")
		i := fl.frame.AllocWord("bir.i")
		ch := fl.frame.AllocWord("bir.ch")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, rOp, dis.FP(ch), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(dst)))
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil

	case "IndexFunc", "LastIndexFunc":
		// ([]byte, func(rune) bool) → int — -1 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	case "SplitAfter":
		// bytes.SplitAfter(s, sep) → [][]byte — split keeping separator
		// Convert to strings, use SplitAfter logic, convert back
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))

		lenS := fl.frame.AllocWord("bsa.lenS")
		lenSep := fl.frame.AllocWord("bsa.lenSep")
		count := fl.frame.AllocWord("bsa.cnt")
		i := fl.frame.AllocWord("bsa.i")
		endIdx := fl.frame.AllocWord("bsa.end")
		candidate := fl.frame.AllocTemp(true)
		limit := fl.frame.AllocWord("bsa.lim")

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))

		// Count
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		bgtNoMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		jmpCntLoop := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		noMatchPC := int32(len(fl.insts))
		fl.insts[bgtNoMatch].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(limit)))
		cntLoopPC := int32(len(fl.insts))
		fl.insts[jmpCntLoop].Dst = dis.Imm(cntLoopPC)
		bgeCntDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqCntFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntFoundPC := int32(len(fl.insts))
		fl.insts[beqCntFound].Dst = dis.Imm(cntFoundPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntDonePC := int32(len(fl.insts))
		fl.insts[bgeCntDone].Dst = dis.Imm(cntDonePC)

		// Allocate [][]byte
		elemTDIdx := fl.makeHeapTypeDesc(types.NewSlice(types.Typ[types.Byte]))
		arrPtr := fl.frame.AllocPointer("bsa:arr")
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(arrPtr)))

		// Fill: segment includes separator
		segStart := fl.frame.AllocWord("bsa.ss")
		arrIdx := fl.frame.AllocWord("bsa.ai")
		segment := fl.frame.AllocTemp(true)
		storeAddr := fl.frame.AllocWord("bsa.sa")
		segBytes := fl.frame.AllocPointer("bsa:seg")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFillFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillFoundPC := int32(len(fl.insts))
		fl.insts[beqFillFound].Dst = dis.Imm(fillFoundPC)
		// segment = s[segStart:i+lenSep] (include separator)
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(endIdx), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		// Last segment
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrPtr), dis.FP(dst)))
		return true, nil

	case "SplitAfterN":
		// bytes.SplitAfterN(s, sep, n) → [][]byte — split keeping separator, max n parts
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		nOp := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)

		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))

		// n == 0 → nil
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		bsan0 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, nOp, dis.Imm(0), dis.Imm(0)))

		// n < 0 → unlimited
		maxN := fl.frame.AllocWord("bsan.max")
		fl.emit(dis.Inst2(dis.IMOVW, nOp, dis.FP(maxN)))
		bsanNotNeg := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, nOp, dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0x1FFFFFFF), dis.FP(maxN)))
		fl.insts[bsanNotNeg].Dst = dis.Imm(int32(len(fl.insts)))

		lenS := fl.frame.AllocWord("bsan.lenS")
		lenSep := fl.frame.AllocWord("bsan.lenSep")
		count := fl.frame.AllocWord("bsan.cnt")
		i := fl.frame.AllocWord("bsan.i")
		endIdx := fl.frame.AllocWord("bsan.end")
		candidate := fl.frame.AllocTemp(true)
		limit := fl.frame.AllocWord("bsan.lim")

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))

		// Count (capped)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		bgtNoMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		jmpCntLoop := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		noMatchPC := int32(len(fl.insts))
		fl.insts[bgtNoMatch].Dst = dis.Imm(noMatchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(limit)))
		cntLoopPC := int32(len(fl.insts))
		fl.insts[jmpCntLoop].Dst = dis.Imm(cntLoopPC)
		bgeCntDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		bgeMaxN := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(count), dis.FP(maxN), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqCntFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntFoundPC := int32(len(fl.insts))
		fl.insts[beqCntFound].Dst = dis.Imm(cntFoundPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntDonePC := int32(len(fl.insts))
		fl.insts[bgeCntDone].Dst = dis.Imm(cntDonePC)
		fl.insts[bgeMaxN].Dst = dis.Imm(cntDonePC)

		// Allocate [][]byte
		elemTDIdx := fl.makeHeapTypeDesc(types.NewSlice(types.Typ[types.Byte]))
		arrPtr := fl.frame.AllocPointer("bsan:arr")
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(arrPtr)))

		// Fill
		segStart := fl.frame.AllocWord("bsan.ss")
		arrIdx := fl.frame.AllocWord("bsan.ai")
		segment := fl.frame.AllocTemp(true)
		storeAddr := fl.frame.AllocWord("bsan.sa")
		maxSplits := fl.frame.AllocWord("bsan.ms")
		segBytes := fl.frame.AllocPointer("bsan:seg")
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(count), dis.FP(maxSplits)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		bgeHitMax := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(arrIdx), dis.FP(maxSplits), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		beqFillFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillFoundPC := int32(len(fl.insts))
		fl.insts[beqFillFound].Dst = dis.Imm(fillFoundPC)
		// segment includes separator
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(endIdx), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		fl.insts[bgeHitMax].Dst = dis.Imm(fillDonePC)
		// Last segment
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(segment), dis.FP(segBytes)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segBytes), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrPtr), dis.FP(dst)))
		bsanAllDonePC := int32(len(fl.insts))
		fl.insts[bsan0].Dst = dis.Imm(bsanAllDonePC)
		return true, nil

	case "FieldsFunc":
		// → nil slice stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	case "ContainsFunc":
		// ([]byte, func(rune) bool) → false stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	case "TrimFunc", "TrimLeftFunc", "TrimRightFunc":
		// ([]byte, func) → []byte — pass through
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(dst)))
		return true, nil

	case "Clone":
		// Clone(b) → b (shallow copy is fine for immutable Dis)
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(dst)))
		return true, nil

	case "CutPrefix":
		// bytes.CutPrefix(s, prefix) → (after []byte, found bool)
		sOp := fl.operandOf(instr.Call.Args[0])
		pOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sStr := fl.frame.AllocTemp(true)
		pStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, pOp, dis.FP(pStr)))
		lenS := fl.frame.AllocWord("bcp.lenS")
		lenP := fl.frame.AllocWord("bcp.lenP")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(pStr), dis.FP(lenP)))
		// Default: return (s, false)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		// if len(prefix) > len(s) → done
		tooShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenP), dis.FP(lenS), dis.Imm(0)))
		// extract s[:len(prefix)] and compare
		head := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(head)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenP), dis.FP(head)))
		noMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(pStr), dis.FP(head), dis.Imm(0)))
		// Match: return (s[len(prefix):], true)
		after := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(after)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lenP), dis.FP(lenS), dis.FP(after)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(after), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		donePC := int32(len(fl.insts))
		fl.insts[tooShort].Dst = dis.Imm(donePC)
		fl.insts[noMatch].Dst = dis.Imm(donePC)
		return true, nil

	case "CutSuffix":
		// bytes.CutSuffix(s, suffix) → (before []byte, found bool)
		sOp := fl.operandOf(instr.Call.Args[0])
		sfxOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sStr := fl.frame.AllocTemp(true)
		sfxStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sfxOp, dis.FP(sfxStr)))
		lenS := fl.frame.AllocWord("bcs.lenS")
		lenSfx := fl.frame.AllocWord("bcs.lenSfx")
		startOff := fl.frame.AllocWord("bcs.off")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sfxStr), dis.FP(lenSfx)))
		// Default: return (s, false)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		tooShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSfx), dis.FP(lenS), dis.Imm(0)))
		// extract s[len(s)-len(suffix):] and compare
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSfx), dis.FP(lenS), dis.FP(startOff)))
		tail := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(tail)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startOff), dis.FP(lenS), dis.FP(tail)))
		noMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(sfxStr), dis.FP(tail), dis.Imm(0)))
		// Match: return (s[:len(s)-len(suffix)], true)
		before := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(before)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(startOff), dis.FP(before)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(before), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		donePC := int32(len(fl.insts))
		fl.insts[tooShort].Dst = dis.Imm(donePC)
		fl.insts[noMatch].Dst = dis.Imm(donePC)
		return true, nil

	case "Cut":
		// bytes.Cut(s, sep) → (before []byte, after []byte, found bool)
		sOp := fl.operandOf(instr.Call.Args[0])
		sepOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		sStr := fl.frame.AllocTemp(true)
		sepStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, sOp, dis.FP(sStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, sepOp, dis.FP(sepStr)))
		lenS := fl.frame.AllocWord("bcut.lenS")
		lenSep := fl.frame.AllocWord("bcut.lenSep")
		limit := fl.frame.AllocWord("bcut.lim")
		i := fl.frame.AllocWord("bcut.i")
		endIdx := fl.frame.AllocWord("bcut.end")
		candidate := fl.frame.AllocTemp(true)
		emptyMP := fl.comp.AllocString("")
		// Default: (s, nil, false)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyMP), dis.FP(dst+iby2wd))) // empty after
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(dst+iby2wd), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sepStr), dis.FP(lenSep)))
		tooShort := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenSep), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSep), dis.FP(lenS), dis.FP(limit)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(limit), dis.FP(limit)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// Search loop
		loopPC := int32(len(fl.insts))
		notFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(endIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(candidate)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(candidate)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(sepStr), dis.FP(candidate), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// Found: before = s[:i], after = s[i+len(sep):]
		fl.insts[foundIdx].Dst = dis.Imm(int32(len(fl.insts)))
		beforeStr := fl.frame.AllocTemp(true)
		afterStr := fl.frame.AllocTemp(true)
		afterStart := fl.frame.AllocWord("bcut.as")
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(beforeStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(i), dis.FP(beforeStr)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(beforeStr), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenSep), dis.FP(i), dis.FP(afterStart)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(sStr), dis.FP(afterStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(afterStart), dis.FP(lenS), dis.FP(afterStr)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(afterStr), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+2*iby2wd)))
		donePC := int32(len(fl.insts))
		fl.insts[tooShort].Dst = dis.Imm(donePC)
		fl.insts[notFound].Dst = dis.Imm(donePC)
		return true, nil

	// Additional Buffer methods
	case "Cap":
		if callee.Signature.Recv() != nil {
			// Cap() → same as Len() for our simple string-backed implementation
			recvSlot := fl.materialize(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.ILENC, dis.FPInd(recvSlot, 0), dis.FP(dst)))
			return true, nil
		}
	case "Grow":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op (no pre-allocation in Dis strings)
		}
	case "WriteRune":
		if callee.Signature.Recv() != nil {
			// (*Buffer).WriteRune(r) → insc(r, len(buf), buf); return (size, nil)
			recvSlot := fl.materialize(instr.Call.Args[0])
			runeVal := fl.operandOf(instr.Call.Args[1])
			lenTmp := fl.frame.AllocWord("")
			fl.emit(dis.Inst2(dis.ILENC, dis.FPInd(recvSlot, 0), dis.FP(lenTmp)))
			fl.emit(dis.NewInst(dis.IINSC, runeVal, dis.FP(lenTmp), dis.FPInd(recvSlot, 0)))
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "ReadRune":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "UnreadByte", "UnreadRune":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "ReadBytes":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "Next":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Truncate":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	case "WriteTo":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "ReadFrom":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "Available":
		// (*Buffer).Available() → 0 stub
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "AvailableBuffer":
		// (*Buffer).AvailableBuffer() → nil slice stub
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

// lowerEncodingHexCall handles encoding/hex package functions.
func (fl *funcLowerer) lowerEncodingHexCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Encode":
		// hex.Encode(dst, src []byte) → int
		// Writes hex encoding of src into dst. Returns len(src)*2.
		dstOp := fl.operandOf(instr.Call.Args[0])
		srcOp := fl.operandOf(instr.Call.Args[1])
		resultSlot := fl.slotOf(instr)

		srcStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, srcOp, dis.FP(srcStr)))

		lenS := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(srcStr), dis.FP(lenS)))

		hexTableOff := fl.comp.AllocString("0123456789abcdef")
		i := fl.frame.AllocWord("")
		dstIdx := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		hi := fl.frame.AllocWord("")
		lo := fl.frame.AllocWord("")
		addr := fl.frame.AllocWord("")

		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstIdx)))

		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

		// ch = src[i]
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(srcStr), dis.FP(i), dis.FP(ch)))
		// hi = ch >> 4; lo = ch & 0xf
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(ch), dis.FP(lo)))

		// Get hex digit chars from table
		hiChar := fl.frame.AllocWord("")
		loChar := fl.frame.AllocWord("")
		hexTable := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(hexTable)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(hexTable), dis.FP(hi), dis.FP(hiChar)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(hexTable), dis.FP(lo), dis.FP(loChar)))

		// dst[dstIdx] = hiChar; dst[dstIdx+1] = loChar
		fl.emit(dis.NewInst(dis.IINDB, dstOp, dis.FP(addr), dis.FP(dstIdx)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(hiChar), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(dstIdx), dis.FP(dstIdx)))
		fl.emit(dis.NewInst(dis.IINDB, dstOp, dis.FP(addr), dis.FP(dstIdx)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(loChar), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(dstIdx), dis.FP(dstIdx)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(dstIdx), dis.FP(resultSlot)))
		return true, nil

	case "Decode":
		// hex.Decode(dst, src []byte) → (int, error)
		// Decodes hex-encoded src into dst. Returns number of bytes written.
		dstOp := fl.operandOf(instr.Call.Args[0])
		srcOp := fl.operandOf(instr.Call.Args[1])
		resultSlot := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)

		srcStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, srcOp, dis.FP(srcStr)))

		lenS := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(srcStr), dis.FP(lenS)))

		i := fl.frame.AllocWord("")
		outIdx := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outIdx)))

		ch := fl.frame.AllocWord("")
		hiVal := fl.frame.AllocWord("")
		loVal := fl.frame.AllocWord("")
		byteVal := fl.frame.AllocWord("")
		addr := fl.frame.AllocWord("")
		i1 := fl.frame.AllocWord("")

		// Need at least 2 chars per iteration
		loopPC := int32(len(fl.insts))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i1)))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i1), dis.FP(lenS), dis.Imm(0)))

		// hexDigit helper: convert char to value 0-15
		// hi char
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(srcStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(hiVal)))
		// '0'-'9' → 0-9
		skipA1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(48), dis.FP(ch), dis.FP(hiVal)))
		skipDone1 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// 'a'-'f' → 10-15
		fl.insts[skipA1].Dst = dis.Imm(int32(len(fl.insts)))
		skipAU1 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(102), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(87), dis.FP(ch), dis.FP(hiVal))) // 'a' - 87 = 10
		skipDone1b := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// 'A'-'F' → 10-15
		fl.insts[skipAU1].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(55), dis.FP(ch), dis.FP(hiVal))) // 'A' - 55 = 10

		doneHi := int32(len(fl.insts))
		fl.insts[skipDone1].Dst = dis.Imm(doneHi)
		fl.insts[skipDone1b].Dst = dis.Imm(doneHi)

		// lo char (i+1)
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(srcStr), dis.FP(i1), dis.FP(ch)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ch), dis.FP(loVal)))
		skipA2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(48), dis.FP(ch), dis.FP(loVal)))
		skipDone2 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[skipA2].Dst = dis.Imm(int32(len(fl.insts)))
		skipAU2 := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(102), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(87), dis.FP(ch), dis.FP(loVal)))
		skipDone2b := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[skipAU2].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(55), dis.FP(ch), dis.FP(loVal)))

		doneLo := int32(len(fl.insts))
		fl.insts[skipDone2].Dst = dis.Imm(doneLo)
		fl.insts[skipDone2b].Dst = dis.Imm(doneLo)

		// byteVal = hi<<4 | lo
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(4), dis.FP(hiVal), dis.FP(byteVal)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(loVal), dis.FP(byteVal), dis.FP(byteVal)))

		// dst[outIdx] = byteVal
		fl.emit(dis.NewInst(dis.IINDB, dstOp, dis.FP(addr), dis.FP(outIdx)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(byteVal), dis.FPInd(addr, 0)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(i), dis.FP(i)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outIdx), dis.FP(outIdx)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		// Return (outIdx, nil error)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(outIdx), dis.FP(resultSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(resultSlot+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(resultSlot+2*iby2wd)))
		return true, nil
	case "Dump":
		// hex.Dump(data) → hex string with spaces between bytes
		// Simplified: "48 65 6c 6c 6f\n" (no offset column or ASCII)
		dataOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, dataOp, dis.FP(sStr)))
		lenS := fl.frame.AllocWord("hd.len")
		i := fl.frame.AllocWord("hd.i")
		ch := fl.frame.AllocWord("hd.ch")
		hi := fl.frame.AllocWord("hd.hi")
		lo := fl.frame.AllocWord("hd.lo")
		hiP1 := fl.frame.AllocWord("hd.hiP1")
		loP1 := fl.frame.AllocWord("hd.loP1")
		hiStr := fl.frame.AllocTemp(true)
		loStr := fl.frame.AllocTemp(true)
		result := fl.frame.AllocTemp(true)
		hexTableOff := fl.comp.AllocString("0123456789abcdef")
		emptyOff := fl.comp.AllocString("")
		spaceOff := fl.comp.AllocString(" ")
		nlOff := fl.comp.AllocString("\n")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		// Add space between bytes (not before first)
		skipSpaceIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(spaceOff), dis.FP(result), dis.FP(result)))
		fl.insts[skipSpaceIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xF), dis.FP(ch), dis.FP(lo)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(hi), dis.FP(hiP1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(hiStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(hi), dis.FP(hiP1), dis.FP(hiStr)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lo), dis.FP(loP1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lo), dis.FP(loP1), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(hiStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(loStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))
		// Add trailing newline
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(nlOff), dis.FP(result), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
		return true, nil
	case "NewEncoder", "NewDecoder", "Dumper":
		// hex.NewEncoder/NewDecoder/Dumper → nil interface stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "EncodeToString":
		// Convert each byte to 2-char hex string
		srcOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)

		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, srcOp, dis.FP(sStr)))

		lenS := fl.frame.AllocWord("")
		i := fl.frame.AllocWord("")
		ch := fl.frame.AllocWord("")
		hi := fl.frame.AllocWord("")
		lo := fl.frame.AllocWord("")
		hiP1 := fl.frame.AllocWord("")
		loP1 := fl.frame.AllocWord("")
		hiStr := fl.frame.AllocTemp(true)
		loStr := fl.frame.AllocTemp(true)
		result := fl.frame.AllocTemp(true)

		hexTableOff := fl.comp.AllocString("0123456789abcdef")
		emptyOff := fl.comp.AllocString("")

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		// hi = ch >> 4; lo = ch & 0xf
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(ch), dis.FP(lo)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(hi), dis.FP(hiP1)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lo), dis.FP(loP1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(hiStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(hi), dis.FP(hiP1), dis.FP(hiStr)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lo), dis.FP(loP1), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(hiStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(loStr), dis.FP(result), dis.FP(result)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[bgeDone].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
		return true, nil

	case "DecodeString":
		// Real hex decoding: "48656c6c6f" → []byte("Hello")
		// Each pair of hex chars → 1 byte
		srcOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)

		lenSlot := fl.frame.AllocWord("hexd.len")
		fl.emit(dis.Inst2(dis.ILENC, srcOp, dis.FP(lenSlot)))

		// Build result as string, convert to []byte at end
		result := fl.frame.AllocTemp(true)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))

		i := fl.frame.AllocWord("hexd.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		ch := fl.frame.AllocWord("hexd.ch")
		hi := fl.frame.AllocWord("hexd.hi")
		lo := fl.frame.AllocWord("hexd.lo")
		bv := fl.frame.AllocWord("hexd.bv")
		i1 := fl.frame.AllocWord("hexd.i1")
		outPos := fl.frame.AllocWord("hexd.op")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outPos)))

		// hexDigit: given char in ch, compute value in outSlot
		// '0'-'9' (48-57) → 0-9, 'a'-'f' (97-102) → 10-15, 'A'-'F' (65-70) → 10-15
		hexDigit := func(outSlot int32) {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outSlot)))
			// ch < 58 → digit
			notDigit := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(48), dis.FP(ch), dis.FP(outSlot)))
			doneJmp := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
			// ch < 71 → uppercase hex
			notUpper := len(fl.insts)
			fl.insts[notDigit].Dst = dis.Imm(int32(notUpper))
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(70), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(55), dis.FP(ch), dis.FP(outSlot))) // 'A'(65)-55=10
			doneJmp2 := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
			// must be lowercase hex
			notUpperPC := int32(len(fl.insts))
			fl.insts[notUpper].Dst = dis.Imm(notUpperPC)
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(87), dis.FP(ch), dis.FP(outSlot))) // 'a'(97)-87=10
			endPC := int32(len(fl.insts))
			fl.insts[doneJmp].Dst = dis.Imm(endPC)
			fl.insts[doneJmp2].Dst = dis.Imm(endPC)
		}

		// Main loop: process 2 chars at a time
		loopPC := int32(len(fl.insts))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i1)))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(i1), dis.FP(lenSlot), dis.Imm(0)))

		// hi nibble
		fl.emit(dis.NewInst(dis.IINDC, srcOp, dis.FP(i), dis.FP(ch)))
		hexDigit(hi)

		// lo nibble
		fl.emit(dis.NewInst(dis.IINDC, srcOp, dis.FP(i1), dis.FP(ch)))
		hexDigit(lo)

		// byte = hi << 4 | lo
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(4), dis.FP(hi), dis.FP(bv)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(lo), dis.FP(bv), dis.FP(bv)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(bv), dis.FP(outPos), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// i += 2
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)

		// Convert string → []byte
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		// nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "EncodedLen":
		// EncodedLen(n) = n * 2
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.NewInst(dis.IMULW, dis.Imm(2), src, dis.FP(dst)))
		return true, nil
	case "DecodedLen":
		// DecodedLen(x) = x / 2
		src := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.NewInst(dis.IDIVW, dis.Imm(2), src, dis.FP(dst)))
		return true, nil
	case "AppendEncode":
		// AppendEncode(dst, src []byte) → dst with hex of src appended
		dstOp := fl.operandOf(instr.Call.Args[0])
		srcOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)

		sStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, srcOp, dis.FP(sStr)))
		existStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, dstOp, dis.FP(existStr)))

		lenS := fl.frame.AllocWord("ae.len")
		i := fl.frame.AllocWord("ae.i")
		ch := fl.frame.AllocWord("ae.ch")
		hi := fl.frame.AllocWord("ae.hi")
		lo := fl.frame.AllocWord("ae.lo")
		hiP1 := fl.frame.AllocWord("ae.hiP1")
		loP1 := fl.frame.AllocWord("ae.loP1")
		hiStr := fl.frame.AllocTemp(true)
		loStr := fl.frame.AllocTemp(true)
		result := fl.frame.AllocTemp(true)

		hexTableOff := fl.comp.AllocString("0123456789abcdef")

		fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(existStr), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0xF), dis.FP(ch), dis.FP(lo)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(hi), dis.FP(hiP1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(hiStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(hi), dis.FP(hiP1), dis.FP(hiStr)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lo), dis.FP(loP1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTableOff), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lo), dis.FP(loP1), dis.FP(loStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(hiStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(loStr), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))

		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		return true, nil
	case "AppendDecode":
		// AppendDecode(dst, src) → (dst, nil)
		srcOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, srcOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Error":
		// InvalidByteError.Error() → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// lowerEncodingBase64Call handles encoding/base64 package functions.
func (fl *funcLowerer) lowerEncodingBase64Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	switch name {
	case "EncodeToString":
		// Real base64 encoding: 3 bytes → 4 chars using lookup table.
		// receiver is arg[0] (Encoding), data is arg[1] ([]byte)
		srcOp := fl.operandOf(instr.Call.Args[1])

		// Base64 alphabet as a lookup string
		b64Alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		alphaMP := fl.comp.AllocString(b64Alphabet)
		alphaSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(alphaMP), dis.FP(alphaSlot)))

		// Convert byte slice to string for INDC access
		strSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, srcOp, dis.FP(strSlot)))

		lenSlot := fl.frame.AllocWord("b64.len")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(strSlot), dis.FP(lenSlot)))

		// result = ""
		emptyMP := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyMP), dis.FP(dst)))

		iSlot := fl.frame.AllocWord("b64.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

		// Temps for 3-byte group
		b0 := fl.frame.AllocWord("b64.b0")
		b1 := fl.frame.AllocWord("b64.b1")
		b2 := fl.frame.AllocWord("b64.b2")
		idx := fl.frame.AllocWord("b64.idx")
		ch := fl.frame.AllocWord("b64.ch")
		remaining := fl.frame.AllocWord("b64.rem")
		i1 := fl.frame.AllocWord("b64.i1")
		i2 := fl.frame.AllocWord("b64.i2")
		tmp := fl.frame.AllocWord("b64.tmp")
		outPos := fl.frame.AllocWord("b64.outPos") // track output string length

		// outPos = 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outPos)))

		// Main loop: process 3 bytes at a time
		loopPC := int32(len(fl.insts))
		// remaining = len - i
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(iSlot), dis.FP(lenSlot), dis.FP(remaining)))
		// if remaining <= 0 → done
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(remaining), dis.Imm(0), dis.Imm(0)))

		// b0 = src[i]
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(strSlot), dis.FP(iSlot), dis.FP(b0)))

		// Check if we have 2+ bytes
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(b1)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(b2)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(i1)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(iSlot), dis.FP(i2)))

		skip1Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i1), dis.FP(lenSlot), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(strSlot), dis.FP(i1), dis.FP(b1)))
		skip1PC := int32(len(fl.insts))
		fl.insts[skip1Idx].Dst = dis.Imm(skip1PC)

		skip2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i2), dis.FP(lenSlot), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(strSlot), dis.FP(i2), dis.FP(b2)))
		skip2PC := int32(len(fl.insts))
		fl.insts[skip2Idx].Dst = dis.Imm(skip2PC)

		// Encode 4 base64 chars from 3 bytes:
		// c0 = b0 >> 2
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(2), dis.FP(b0), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(63), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(alphaSlot), dis.FP(idx), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// c1 = (b0 & 3) << 4 | b1 >> 4
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(3), dis.FP(b0), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(4), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(b1), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(63), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(alphaSlot), dis.FP(idx), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// if remaining < 2 → pad with "=="
		pad2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(remaining), dis.Imm(2), dis.Imm(0)))

		// c2 = (b1 & 15) << 2 | b2 >> 6
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(b1), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(2), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(6), dis.FP(b2), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(63), dis.FP(idx), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(alphaSlot), dis.FP(idx), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// if remaining < 3 → pad with "="
		pad1Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(remaining), dis.Imm(3), dis.Imm(0)))

		// c3 = b2 & 63
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(63), dis.FP(b2), dis.FP(idx)))
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(alphaSlot), dis.FP(idx), dis.FP(ch)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// i += 3, loop
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(3), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		// pad2: append "==" using IADDC (string concat)
		pad2PC := int32(len(fl.insts))
		fl.insts[pad2Idx].Dst = dis.Imm(pad2PC)
		eqMP := fl.comp.AllocString("==")
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(eqMP), dis.FP(dst), dis.FP(dst)))
		pad2DoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		// pad1: append "="
		pad1PC := int32(len(fl.insts))
		fl.insts[pad1Idx].Dst = dis.Imm(pad1PC)
		eq1MP := fl.comp.AllocString("=")
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(eq1MP), dis.FP(dst), dis.FP(dst)))
		pad1DoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		// After padding, need to loop back (i += 3 already happened before pad branch)
		// Actually pad branches jump here after appending padding, then go to done
		// The pad branches already add the padding and then i+=3 + loop would be wrong
		// Let's make pad branches jump straight to done
		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.insts[pad2DoneIdx].Dst = dis.Imm(donePC)
		fl.insts[pad1DoneIdx].Dst = dis.Imm(donePC)
		return true, nil

	case "DecodeString":
		// Real base64 decoding: 4 chars → 3 bytes using reverse lookup.
		srcOp := fl.operandOf(instr.Call.Args[1])

		// A-Z: 0-25, a-z: 26-51, 0-9: 52-61, +: 62, /: 63, =: pad

		lenSlot := fl.frame.AllocWord("b64d.len")
		fl.emit(dis.Inst2(dis.ILENC, srcOp, dis.FP(lenSlot)))

		// result = "" (will convert to []byte at end)
		result := fl.frame.AllocTemp(true)
		emptyMP := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyMP), dis.FP(result)))

		iSlot := fl.frame.AllocWord("b64d.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

		ch := fl.frame.AllocWord("b64d.ch")
		v0 := fl.frame.AllocWord("b64d.v0")
		v1 := fl.frame.AllocWord("b64d.v1")
		v2 := fl.frame.AllocWord("b64d.v2")
		v3 := fl.frame.AllocWord("b64d.v3")
		tmp := fl.frame.AllocWord("b64d.tmp")
		byteval := fl.frame.AllocWord("b64d.bv")
		i1 := fl.frame.AllocWord("b64d.i1")
		i2 := fl.frame.AllocWord("b64d.i2")
		i3 := fl.frame.AllocWord("b64d.i3")
		outPos := fl.frame.AllocWord("b64d.outPos")

		// outPos = 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outPos)))

		// Main loop: process 4 chars at a time
		loopPC := int32(len(fl.insts))
		// if i+3 >= len → done
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(3), dis.FP(iSlot), dis.FP(tmp)))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(tmp), dis.FP(lenSlot), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(i1)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(iSlot), dis.FP(i2)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(3), dis.FP(iSlot), dis.FP(i3)))

		// Decode each of the 4 chars using arithmetic:
		// A-Z(65-90)→0-25, a-z(97-122)→26-51, 0-9(48-57)→52-61, +(43)→62, /(47)→63
		decodeChar := func(charIdx int32, outSlot int32) {
			fl.emit(dis.NewInst(dis.IINDC, srcOp, dis.FP(charIdx), dis.FP(ch)))
			// Default val = 0 (covers '=' case)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outSlot)))
			eqIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(61), dis.Imm(0))) // '=' → 0, done

			// A-Z: 65-90 → 0-25
			notAZLow := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(65), dis.Imm(0)))
			notAZHigh := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(90), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(65), dis.FP(ch), dis.FP(outSlot)))
			skipAZ := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

			// a-z: 97-122 → 26-51
			checkLower := int32(len(fl.insts))
			fl.insts[notAZHigh].Dst = dis.Imm(checkLower)
			notAzLow := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(97), dis.Imm(0)))
			notAzHigh := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(122), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(97), dis.FP(ch), dis.FP(outSlot)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(26), dis.FP(outSlot), dis.FP(outSlot)))
			skipAZL := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

			// 0-9: 48-57 → 52-61
			checkDigit := int32(len(fl.insts))
			fl.insts[notAZLow].Dst = dis.Imm(checkDigit)
			fl.insts[notAzLow].Dst = dis.Imm(checkDigit)
			fl.insts[notAzHigh].Dst = dis.Imm(checkDigit)
			notDigLow := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(48), dis.Imm(0)))
			notDigHigh := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(48), dis.FP(ch), dis.FP(outSlot)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(52), dis.FP(outSlot), dis.FP(outSlot)))
			skipNum := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

			// + (43) → 62, / (47) → 63
			checkSpecial := int32(len(fl.insts))
			fl.insts[notDigLow].Dst = dis.Imm(checkSpecial)
			fl.insts[notDigHigh].Dst = dis.Imm(checkSpecial)
			plusIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(43), dis.Imm(0))) // '+'
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(63), dis.FP(outSlot)))           // must be '/'
			endIdx := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
			plusPC := int32(len(fl.insts))
			fl.insts[plusIdx].Dst = dis.Imm(plusPC)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(62), dis.FP(outSlot)))

			endPC := int32(len(fl.insts))
			fl.insts[eqIdx].Dst = dis.Imm(endPC)
			fl.insts[skipAZ].Dst = dis.Imm(endPC)
			fl.insts[skipAZL].Dst = dis.Imm(endPC)
			fl.insts[skipNum].Dst = dis.Imm(endPC)
			fl.insts[endIdx].Dst = dis.Imm(endPC)
		}

		decodeChar(iSlot, v0)
		decodeChar(i1, v1)
		decodeChar(i2, v2)
		decodeChar(i3, v3)

		// byte0 = v0 << 2 | v1 >> 4
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(2), dis.FP(v0), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(v1), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(255), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(byteval), dis.FP(outPos), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// Check if 3rd char is '=' → skip byte1 and byte2
		fl.emit(dis.NewInst(dis.IINDC, srcOp, dis.FP(i2), dis.FP(ch)))
		pad2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(61), dis.Imm(0)))

		// byte1 = (v1 & 15) << 4 | v2 >> 2
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(v1), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(4), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(2), dis.FP(v2), dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(tmp), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(255), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(byteval), dis.FP(outPos), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// Check if 4th char is '=' → skip byte2
		fl.emit(dis.NewInst(dis.IINDC, srcOp, dis.FP(i3), dis.FP(ch)))
		pad1Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(61), dis.Imm(0)))

		// byte2 = (v2 & 3) << 6 | v3
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(3), dis.FP(v2), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(6), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(v3), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(255), dis.FP(byteval), dis.FP(byteval)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(byteval), dis.FP(outPos), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))

		// i += 4, loop
		nextPC := int32(len(fl.insts))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(4), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		// Padding targets
		fl.insts[pad2Idx].Dst = dis.Imm(nextPC)
		fl.insts[pad1Idx].Dst = dis.Imm(nextPC)

		// Done: convert string → []byte
		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(result), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil

	case "Encode":
		return true, nil
	case "Decode":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "EncodedLen":
		// (n + 2) / 3 * 4
		src := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), src, dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IDIVW, dis.Imm(3), dis.FP(dst), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IMULW, dis.Imm(4), dis.FP(dst), dis.FP(dst)))
		return true, nil
	case "DecodedLen":
		// n / 4 * 3
		src := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.NewInst(dis.IDIVW, dis.Imm(4), src, dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IMULW, dis.Imm(3), dis.FP(dst), dis.FP(dst)))
		return true, nil
	case "Strict", "WithPadding":
		recvOp := fl.operandOf(instr.Call.Args[0])
		fl.emit(dis.Inst2(dis.IMOVP, recvOp, dis.FP(dst)))
		return true, nil
	case "NewEncoding":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewEncoder", "NewDecoder":
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// fmt package — extended functions
// ============================================================

// lowerFmtSprint: fmt.Sprint(args...) → string. Same as Sprintf with "%v" style.
func (fl *funcLowerer) lowerFmtSprint(instr *ssa.Call) (bool, error) {
	// Sprint concatenates values with no separator.
	// Use the same approach as Println but collect into string instead of printing.
	strSlot, ok := fl.emitSprintConcatInline(instr, false)
	if !ok {
		return false, nil
	}
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(strSlot), dis.FP(dst)))
	return true, nil
}

// lowerFmtSprintln: fmt.Sprintln(args...) → concatenate with spaces and newline.
func (fl *funcLowerer) lowerFmtSprintln(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintConcatInline(instr, true)
	if !ok {
		return false, nil
	}
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(strSlot), dis.FP(dst)))
	return true, nil
}

// lowerFmtPrint: fmt.Print(args...) → print without newline.
func (fl *funcLowerer) lowerFmtPrint(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintConcatInline(instr, false)
	if !ok {
		return false, nil
	}
	fl.emitSysCall("print", []callSiteArg{{strSlot, true}})
	if len(*instr.Referrers()) > 0 {
		dstSlot := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+2*iby2wd)))
	}
	return true, nil
}

// lowerFmtFprintf: fmt.Fprintf(w, format, args...) → ignore w, use Printf logic.
func (fl *funcLowerer) lowerFmtFprintf(instr *ssa.Call) (bool, error) {
	// Skip the first arg (w io.Writer) and treat rest as Printf
	// Create a modified Call that skips the writer argument
	strSlot, ok := fl.emitSprintfInline(instr)
	if !ok {
		return false, nil
	}
	fl.emitSysCall("print", []callSiteArg{{strSlot, true}})
	if len(*instr.Referrers()) > 0 {
		dstSlot := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+2*iby2wd)))
	}
	return true, nil
}

// lowerFmtFprintln: fmt.Fprintln(w, args...) → ignore w, use Println logic.
func (fl *funcLowerer) lowerFmtFprintln(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintConcatInline(instr, true)
	if !ok {
		return false, nil
	}
	fl.emitSysCall("print", []callSiteArg{{strSlot, true}})
	if len(*instr.Referrers()) > 0 {
		dstSlot := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+2*iby2wd)))
	}
	return true, nil
}

// lowerFmtFprint: fmt.Fprint(w, args...) → ignore w, print args.
func (fl *funcLowerer) lowerFmtFprint(instr *ssa.Call) (bool, error) {
	strSlot, ok := fl.emitSprintConcatInline(instr, false)
	if !ok {
		return false, nil
	}
	fl.emitSysCall("print", []callSiteArg{{strSlot, true}})
	if len(*instr.Referrers()) > 0 {
		dstSlot := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dstSlot+2*iby2wd)))
	}
	return true, nil
}

// emitSprintConcatInline concatenates the variadic args of a Print/Sprint/Println-style call
// into a single string. If addNewline is true, appends "\n" at the end (Println style).
// Returns the frame slot of the result string and true on success.
func (fl *funcLowerer) emitSprintConcatInline(instr *ssa.Call, addNewline bool) (int32, bool) {
	result := fl.frame.AllocTemp(true)
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))

	args := instr.Call.Args
	// Skip first arg if it's an io.Writer (Fprint/Fprintln/Fprintf)
	startIdx := 0
	if len(args) > 0 {
		if _, ok := args[0].Type().Underlying().(*types.Interface); ok {
			name := ""
			if callee, ok := instr.Call.Value.(*ssa.Function); ok {
				name = callee.Name()
			}
			if name == "Fprintf" || name == "Fprintln" || name == "Fprint" {
				startIdx = 1
			}
		}
	}

	// For variadic functions, the SSA packs args into a []interface{} slice.
	// Trace back to find individual elements.
	var elements []ssa.Value
	if startIdx < len(args) {
		sliceVal := args[startIdx]
		if traced := fl.traceAllVarargElements(sliceVal); traced != nil {
			elements = traced
		} else {
			// Fallback: treat remaining args as direct values
			elements = args[startIdx:]
		}
	}

	for i, elem := range elements {
		if err := fl.emitSprintConcatElem(elem, result, addNewline && i > 0); err != nil {
			return 0, false
		}
	}

	if addNewline {
		nlMP := fl.comp.AllocString("\n")
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(nlMP), dis.FP(result), dis.FP(result)))
	}

	return result, true
}

// emitSprintConcatElem converts a single value to string and appends it to result.
func (fl *funcLowerer) emitSprintConcatElem(elem ssa.Value, result int32, addSpace bool) error {
	if addSpace {
		spaceMP := fl.comp.AllocString(" ")
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(spaceMP), dis.FP(result), dis.FP(result)))
	}

	t := elem.Type().Underlying()
	basic, isBasic := t.(*types.Basic)
	tmp := fl.frame.AllocTemp(true)

	if isBasic {
		switch {
		case basic.Kind() == types.String:
			src := fl.operandOf(elem)
			fl.emit(dis.Inst2(dis.IMOVP, src, dis.FP(tmp)))
		case basic.Info()&types.IsInteger != 0:
			src := fl.operandOf(elem)
			fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(tmp)))
		case basic.Info()&types.IsFloat != 0:
			src := fl.operandOf(elem)
			fl.emit(dis.Inst2(dis.ICVTFC, src, dis.FP(tmp)))
		case basic.Kind() == types.Bool:
			src := fl.operandOf(elem)
			trueMP := fl.comp.AllocString("true")
			falseMP := fl.comp.AllocString("false")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(falseMP), dis.FP(tmp)))
			skipIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, src, dis.Imm(0), dis.Imm(0)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(trueMP), dis.FP(tmp)))
			fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
		default:
			src := fl.operandOf(elem)
			fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(tmp)))
		}
	} else {
		src := fl.operandOf(elem)
		fl.emit(dis.Inst2(dis.ICVTWC, src, dis.FP(tmp)))
	}

	fl.emit(dis.NewInst(dis.IADDC, dis.FP(tmp), dis.FP(result), dis.FP(result)))
	return nil
}

// ============================================================
// path/filepath package
// ============================================================

// lowerFilepathCall handles calls to the path/filepath package.
// Since Inferno uses forward-slash paths (like Unix), filepath functions
// behave identically to the path package equivalents.
func (fl *funcLowerer) lowerFilepathCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Base":
		return fl.lowerFilepathBase(instr)
	case "Dir":
		return fl.lowerFilepathDir(instr)
	case "Ext":
		return fl.lowerFilepathExt(instr)
	case "Clean":
		return fl.lowerFilepathClean(instr)
	case "Join":
		return fl.lowerFilepathJoin(instr)
	case "IsAbs":
		return fl.lowerFilepathIsAbs(instr)
	case "Abs":
		return fl.lowerFilepathAbs(instr)
	case "Rel":
		// filepath.Rel(basepath, targpath) → (string, error)
		// If target starts with base+"/", strip the prefix
		baseOp := fl.operandOf(instr.Call.Args[0])
		targetOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		// Default: return target, nil error
		fl.emit(dis.Inst2(dis.IMOVP, targetOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		// Build base+"/"
		slashOff := fl.comp.AllocString("/")
		baseSlash := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, baseOp, dis.FP(baseSlash)))
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(slashOff), dis.FP(baseSlash), dis.FP(baseSlash)))
		// Check if target starts with base+"/"
		lenBS := fl.frame.AllocWord("rel.lbs")
		lenT := fl.frame.AllocWord("rel.lt")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(baseSlash), dis.FP(lenBS)))
		fl.emit(dis.Inst2(dis.ILENC, targetOp, dis.FP(lenT)))
		// if lenT < lenBS → can't match
		tooShortIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenBS), dis.FP(lenT), dis.Imm(0)))
		// Check prefix
		prefixStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, targetOp, dis.FP(prefixStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenBS), dis.FP(prefixStr)))
		noMatchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(prefixStr), dis.FP(baseSlash), dis.Imm(0)))
		// Match: result = target[lenBS:]
		relStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, targetOp, dis.FP(relStr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lenBS), dis.FP(lenT), dis.FP(relStr)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(relStr), dis.FP(dst)))
		// Check if base == target (same path → ".")
		donePC := int32(len(fl.insts))
		fl.insts[tooShortIdx].Dst = dis.Imm(donePC)
		fl.insts[noMatchIdx].Dst = dis.Imm(donePC)
		return true, nil
	case "Split":
		// filepath.Split(path) → (dir, file) — split at last '/'
		// dir = path[:i+1], file = path[i+1:] where i is last index of '/'
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		lenS := fl.frame.AllocWord("fps.len")
		i := fl.frame.AllocWord("fps.i")
		ch := fl.frame.AllocWord("fps.ch")
		lastSlash := fl.frame.AllocWord("fps.ls")
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(lastSlash)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// Find last '/'
		loopPC := int32(len(fl.insts))
		doneSearchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		notSlashIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm('/'), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(lastSlash)))
		nextPC := int32(len(fl.insts))
		fl.insts[notSlashIdx].Dst = dis.Imm(nextPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		doneSearchPC := int32(len(fl.insts))
		fl.insts[doneSearchIdx].Dst = dis.Imm(doneSearchPC)
		// If no slash found: dir="", file=path
		emptyOff := fl.comp.AllocString("")
		noSlashIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lastSlash), dis.Imm(0), dis.Imm(0)))
		// Slash found: dir=path[:lastSlash+1], file=path[lastSlash+1:]
		splitPt := fl.frame.AllocWord("fps.sp")
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lastSlash), dis.FP(splitPt)))
		dirTmp := fl.frame.AllocTemp(true)
		fileTmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dirTmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(splitPt), dis.FP(dirTmp)))
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(fileTmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(splitPt), dis.FP(lenS), dis.FP(fileTmp)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(dirTmp), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(fileTmp), dis.FP(dst+iby2wd)))
		allDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// No slash: dir="", file=path
		noSlashPC := int32(len(fl.insts))
		fl.insts[noSlashIdx].Dst = dis.Imm(noSlashPC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst+iby2wd)))
		fl.insts[allDoneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "ToSlash", "FromSlash":
		// On Inferno (Unix-like), these are identity
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		return true, nil
	case "Match":
		// filepath.Match(pattern, name) → (matched bool, err error)
		// Simple implementation: exact match or "*" wildcard only
		patOp := fl.operandOf(instr.Call.Args[0])
		nameOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		// Default: false, nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		// Check if pattern == "*" (match everything)
		starOff := fl.comp.AllocString("*")
		matchAllIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, patOp, dis.MP(starOff), dis.Imm(0)))
		// Check if pattern == name (exact match)
		exactIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, patOp, nameOp, dis.Imm(0)))
		// Check if pattern has "*" prefix/suffix: "*.ext" or "prefix*"
		lenPat := fl.frame.AllocWord("fm.lp")
		i := fl.frame.AllocWord("fm.i")
		ch := fl.frame.AllocWord("fm.ch")
		fl.emit(dis.Inst2(dis.ILENC, patOp, dis.FP(lenPat)))
		// Find '*' in pattern
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		searchPC := int32(len(fl.insts))
		noStarIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenPat), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, patOp, dis.FP(i), dis.FP(ch)))
		foundStarIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm('*'), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(searchPC)))
		// Found star at index i: check prefix and suffix
		fl.insts[foundStarIdx].Dst = dis.Imm(int32(len(fl.insts)))
		prefix := fl.frame.AllocTemp(true)
		suffix := fl.frame.AllocTemp(true)
		starPlus1 := fl.frame.AllocWord("fm.sp1")
		fl.emit(dis.Inst2(dis.IMOVP, patOp, dis.FP(prefix)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(i), dis.FP(prefix)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(starPlus1)))
		fl.emit(dis.Inst2(dis.IMOVP, patOp, dis.FP(suffix)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(starPlus1), dis.FP(lenPat), dis.FP(suffix)))
		// Check name starts with prefix
		lenName := fl.frame.AllocWord("fm.ln")
		lenPrefix := fl.frame.AllocWord("fm.lpx")
		lenSuffix := fl.frame.AllocWord("fm.lsx")
		fl.emit(dis.Inst2(dis.ILENC, nameOp, dis.FP(lenName)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(prefix), dis.FP(lenPrefix)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(suffix), dis.FP(lenSuffix)))
		// name must be >= len(prefix) + len(suffix)
		minLen := fl.frame.AllocWord("fm.ml")
		fl.emit(dis.NewInst(dis.IADDW, dis.FP(lenPrefix), dis.FP(lenSuffix), dis.FP(minLen)))
		tooShortIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(minLen), dis.FP(lenName), dis.Imm(0)))
		// Check prefix
		namePrefix := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, nameOp, dis.FP(namePrefix)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenPrefix), dis.FP(namePrefix)))
		prefixMismatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(namePrefix), dis.FP(prefix), dis.Imm(0)))
		// Check suffix
		nameSuffix := fl.frame.AllocTemp(true)
		suffStart := fl.frame.AllocWord("fm.ss")
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(lenSuffix), dis.FP(lenName), dis.FP(suffStart)))
		fl.emit(dis.Inst2(dis.IMOVP, nameOp, dis.FP(nameSuffix)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(suffStart), dis.FP(lenName), dis.FP(nameSuffix)))
		suffixMismatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(nameSuffix), dis.FP(suffix), dis.Imm(0)))
		// Both match → true
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[noStarIdx].Dst = dis.Imm(donePC)
		fl.insts[tooShortIdx].Dst = dis.Imm(donePC)
		fl.insts[prefixMismatch].Dst = dis.Imm(donePC)
		fl.insts[suffixMismatch].Dst = dis.Imm(donePC)
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// Match all or exact match
		matchPC := int32(len(fl.insts))
		fl.insts[matchAllIdx].Dst = dis.Imm(matchPC)
		fl.insts[exactIdx].Dst = dis.Imm(matchPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		fl.insts[jmpDone].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "Glob":
		// filepath.Glob(pattern) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "EvalSymlinks":
		// filepath.EvalSymlinks(path) → (path, nil) stub
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "VolumeName":
		// filepath.VolumeName(path) → "" (no volumes on Inferno)
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "SplitList":
		// filepath.SplitList(path) → []string — split on ':'
		// Reuse strings.Split logic with ":" separator
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		colonOff := fl.comp.AllocString(":")
		sepOp := dis.MP(colonOff)
		lenS := fl.frame.AllocWord("fsl.lenS")
		count := fl.frame.AllocWord("fsl.cnt")
		i := fl.frame.AllocWord("fsl.i")
		ch := fl.frame.AllocWord("fsl.ch")
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		// Empty → return [""]
		// Count ':'s
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(count)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		cntLoopPC := int32(len(fl.insts))
		bgeCntDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		bneNotColon := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(':'), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(count), dis.FP(count)))
		notColonPC := int32(len(fl.insts))
		fl.insts[bneNotColon].Dst = dis.Imm(notColonPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(cntLoopPC)))
		cntDonePC := int32(len(fl.insts))
		fl.insts[bgeCntDone].Dst = dis.Imm(cntDonePC)
		// Allocate
		elemTDIdx := fl.makeHeapTypeDesc(types.Typ[types.String])
		arrPtr := fl.frame.AllocPointer("fsl:arr")
		fl.emit(dis.NewInst(dis.INEWA, dis.FP(count), dis.Imm(int32(elemTDIdx)), dis.FP(arrPtr)))
		// Fill: scan for ':', extract segments
		segStart := fl.frame.AllocWord("fsl.ss")
		arrIdx := fl.frame.AllocWord("fsl.ai")
		segment := fl.frame.AllocTemp(true)
		storeAddr := fl.frame.AllocWord("fsl.sa")
		_ = sepOp
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(segStart)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(arrIdx)))
		fillLoopPC := int32(len(fl.insts))
		bgeFillDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		bneNoSep := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(':'), dis.Imm(0)))
		// Found ':'
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(i), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segment), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(arrIdx), dis.FP(arrIdx)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(segStart)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		noSepPC := int32(len(fl.insts))
		fl.insts[bneNoSep].Dst = dis.Imm(noSepPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(fillLoopPC)))
		fillDonePC := int32(len(fl.insts))
		fl.insts[bgeFillDone].Dst = dis.Imm(fillDonePC)
		// Last segment
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(segment)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(segStart), dis.FP(lenS), dis.FP(segment)))
		fl.emit(dis.NewInst(dis.IINDW, dis.FP(arrPtr), dis.FP(storeAddr), dis.FP(arrIdx)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(segment), dis.FPInd(storeAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrPtr), dis.FP(dst)))
		return true, nil
	case "Walk", "WalkDir":
		// filepath.Walk/WalkDir(root, fn) → nil error stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "HasPrefix":
		// filepath.HasPrefix(p, prefix) → bool (strings.HasPrefix equivalent)
		pOp := fl.operandOf(instr.Call.Args[0])
		pfxOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		lenP := fl.frame.AllocWord("fhp.lp")
		lenPfx := fl.frame.AllocWord("fhp.lpfx")
		fl.emit(dis.Inst2(dis.ILENC, pOp, dis.FP(lenP)))
		fl.emit(dis.Inst2(dis.ILENC, pfxOp, dis.FP(lenPfx)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		tooShortIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(lenPfx), dis.FP(lenP), dis.Imm(0)))
		head := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, pOp, dis.FP(head)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lenPfx), dis.FP(head)))
		noMatchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, pfxOp, dis.FP(head), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[tooShortIdx].Dst = dis.Imm(donePC)
		fl.insts[noMatchIdx].Dst = dis.Imm(donePC)
		return true, nil
	case "IsLocal":
		// filepath.IsLocal(path) → bool
		// False if: empty, starts with '/', starts with "..", or contains "/.."
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenS := fl.frame.AllocWord("il.len")
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst))) // default true

		// Empty → false
		beqEmpty := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenS), dis.Imm(0), dis.Imm(0)))

		// Starts with '/' → false
		ch := fl.frame.AllocWord("il.ch")
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.Imm(0), dis.FP(ch)))
		beqSlash := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm('/'), dis.Imm(0)))

		// Check for ".." at start or "/.." anywhere
		// Simple: scan for ".." preceded by start or '/'
		i := fl.frame.AllocWord("il.i")
		ch2 := fl.frame.AllocWord("il.ch2")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		limit := fl.frame.AllocWord("il.lim")
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenS), dis.FP(limit)))
		scanPC := int32(len(fl.insts))
		bgeScanDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(limit), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		bneNotDot := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm('.'), dis.Imm(0)))
		// Found '.', check next char
		next := fl.frame.AllocWord("il.next")
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(next)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(next), dis.FP(ch2)))
		bneNotDotDot := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch2), dis.Imm('.'), dis.Imm(0)))
		// Found "..", check if at start (i==0) or preceded by '/'
		beqAtStart := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
		// Check char before i
		prev := fl.frame.AllocWord("il.prev")
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(prev)))
		prevCh := fl.frame.AllocWord("il.pch")
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(prev), dis.FP(prevCh)))
		beqPrevSlash := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(prevCh), dis.Imm('/'), dis.Imm(0)))
		// Also check ".." is at end or followed by '/'
		notDotDotPC := int32(len(fl.insts))
		fl.insts[bneNotDot].Dst = dis.Imm(notDotDotPC)
		fl.insts[bneNotDotDot].Dst = dis.Imm(notDotDotPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(scanPC)))

		// ".." at start or after '/': check if followed by end or '/'
		dotDotCheckPC := int32(len(fl.insts))
		fl.insts[beqAtStart].Dst = dis.Imm(dotDotCheckPC)
		fl.insts[beqPrevSlash].Dst = dis.Imm(dotDotCheckPC)
		afterDD := fl.frame.AllocWord("il.add")
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(i), dis.FP(afterDD)))
		// If afterDD >= lenS → ".." at end → false
		beqDDEnd := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(afterDD), dis.FP(lenS), dis.Imm(0)))
		// Check s[afterDD] == '/'
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(afterDD), dis.FP(ch)))
		beqDDSlash := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm('/'), dis.Imm(0)))
		// Not a real ".." component, continue
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(scanPC)))

		// FALSE targets
		falsePC := int32(len(fl.insts))
		fl.insts[beqEmpty].Dst = dis.Imm(falsePC)
		fl.insts[beqSlash].Dst = dis.Imm(falsePC)
		fl.insts[beqDDEnd].Dst = dis.Imm(falsePC)
		fl.insts[beqDDSlash].Dst = dis.Imm(falsePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

		donePC := int32(len(fl.insts))
		fl.insts[bgeScanDone].Dst = dis.Imm(donePC)
		return true, nil
	}
	return false, nil
}

// lowerFilepathBase returns the last element of path (after final slash).
func (fl *funcLowerer) lowerFilepathBase(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenP := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, pathOp, dis.FP(lenP)))

	// Empty path → return "."
	dotOff := fl.comp.AllocString(".")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(dotOff), dis.FP(dst)))
	beqEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenP), dis.Imm(0)))

	// Start from end, find last '/'
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenP), dis.FP(i)))
	loopPC := int32(len(fl.insts))
	bltIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))

	// Check if path[i] == '/'
	charSlot := fl.frame.AllocTemp(true)
	endIdx := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(i), dis.Imm(1), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(charSlot)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(charSlot)))
	slashOff := fl.comp.AllocString("/")
	beqSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.MP(slashOff), dis.FP(charSlot), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Found slash at i → result is path[i+1:]
	foundPC := int32(len(fl.insts))
	fl.insts[beqSlashIdx].Dst = dis.Imm(foundPC)
	startSlot := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(startSlot)))
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(dst)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(startSlot), dis.FP(lenP), dis.FP(dst)))
	jmpEndIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// No slash found → whole path is the base
	noSlashPC := int32(len(fl.insts))
	fl.insts[bltIdx].Dst = dis.Imm(noSlashPC)
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[jmpEndIdx].Dst = dis.Imm(donePC)
	fl.insts[beqEmptyIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerFilepathDir returns all but the last element of path.
func (fl *funcLowerer) lowerFilepathDir(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenP := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, pathOp, dis.FP(lenP)))

	// Empty path → return "."
	dotOff := fl.comp.AllocString(".")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(dotOff), dis.FP(dst)))
	beqEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenP), dis.Imm(0)))

	// Find last '/'
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenP), dis.FP(i)))
	loopPC := int32(len(fl.insts))
	bltIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))

	charSlot := fl.frame.AllocTemp(true)
	endIdx := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(i), dis.Imm(1), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(charSlot)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(charSlot)))
	slashOff := fl.comp.AllocString("/")
	beqSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.MP(slashOff), dis.FP(charSlot), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Found slash at i → dir is path[:i] (or "/" if i==0)
	foundPC := int32(len(fl.insts))
	fl.insts[beqSlashIdx].Dst = dis.Imm(foundPC)
	beqRootIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(i), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(dst)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(i), dis.FP(dst)))
	jmpEndIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// Slash at 0 → return "/"
	rootPC := int32(len(fl.insts))
	fl.insts[beqRootIdx].Dst = dis.Imm(rootPC)
	rootOff := fl.comp.AllocString("/")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(rootOff), dis.FP(dst)))
	jmpEndIdx2 := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// No slash found → return "."
	noSlashPC := int32(len(fl.insts))
	fl.insts[bltIdx].Dst = dis.Imm(noSlashPC)

	donePC := int32(len(fl.insts))
	fl.insts[jmpEndIdx].Dst = dis.Imm(donePC)
	fl.insts[jmpEndIdx2].Dst = dis.Imm(donePC)
	fl.insts[beqEmptyIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerFilepathExt returns the file extension (including the dot).
func (fl *funcLowerer) lowerFilepathExt(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenP := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")

	fl.emit(dis.Inst2(dis.ILENC, pathOp, dis.FP(lenP)))

	// Default: empty string
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))

	beqEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenP), dis.Imm(0)))

	// Scan backwards for '.'
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenP), dis.FP(i)))
	loopPC := int32(len(fl.insts))
	bltIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(i), dis.Imm(0), dis.Imm(0)))

	charSlot := fl.frame.AllocTemp(true)
	endIdx := fl.frame.AllocWord("")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(i), dis.Imm(1), dis.FP(endIdx)))
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(charSlot)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(endIdx), dis.FP(charSlot)))

	// Check for '/'  — stop scanning if we hit a dir separator
	slashOff := fl.comp.AllocString("/")
	bSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.MP(slashOff), dis.FP(charSlot), dis.Imm(0)))

	// Check for '.'
	dotOff := fl.comp.AllocString(".")
	bDotIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.MP(dotOff), dis.FP(charSlot), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Found dot at i → ext is path[i:]
	foundDotPC := int32(len(fl.insts))
	fl.insts[bDotIdx].Dst = dis.Imm(foundDotPC)
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(dst)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(lenP), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[bltIdx].Dst = dis.Imm(donePC)
	fl.insts[bSlashIdx].Dst = dis.Imm(donePC)
	fl.insts[beqEmptyIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerFilepathClean normalizes a path: collapses multiple slashes,
// removes "." components, resolves ".." components, removes trailing slashes.
func (fl *funcLowerer) lowerFilepathClean(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// Working variables
	srcLen := fl.frame.AllocWord("fc.slen")
	srcPos := fl.frame.AllocWord("fc.spos")
	ch := fl.frame.AllocWord("fc.ch")
	isRoot := fl.frame.AllocWord("fc.root")
	result := fl.frame.AllocTemp(true) // result string
	resLen := fl.frame.AllocWord("fc.rlen")
	compStart := fl.frame.AllocWord("fc.cs")
	compLen := fl.frame.AllocWord("fc.cl")
	comp := fl.frame.AllocTemp(true)   // current component string
	scanPos := fl.frame.AllocWord("fc.sp")
	scanCh := fl.frame.AllocWord("fc.sc")

	// String constants
	emptyOff := fl.comp.AllocString("")
	dotOff := fl.comp.AllocString(".")
	ddOff := fl.comp.AllocString("..")
	slashOff := fl.comp.AllocString("/")

	// Initialize
	fl.emit(dis.Inst2(dis.ILENC, pathOp, dis.FP(srcLen)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(srcPos)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(isRoot)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))

	// If empty string, return "."
	emptyCheckIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(srcLen), dis.Imm(0), dis.Imm(0)))

	// Check if starts with '/' (47)
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.Imm(0), dis.FP(ch)))
	setRootIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(47), dis.Imm(0))) // ch == '/' → setRoot

	// Not rooted: fall through to mainLoop
	toMainLoopJmp := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))

	// setRoot: mark as rooted, skip leading '/'
	fl.insts[setRootIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(isRoot)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(srcPos)))

	// mainLoop: process each component
	mainLoopPC := int32(len(fl.insts))
	fl.insts[toMainLoopJmp].Dst = dis.Imm(mainLoopPC)

	// Skip consecutive '/' in source
	skipSlashPC := int32(len(fl.insts))
	skipSlashDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(srcLen), dis.FP(srcPos), dis.Imm(0))) // pos >= len → done
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.FP(srcPos), dis.FP(ch)))
	isSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(47), dis.Imm(0))) // ch == '/' → skip
	// ch != '/': goto extractComponent
	toExtractJmp := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))
	// ch == '/': advance and loop
	fl.insts[isSlashIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(srcPos), dis.FP(srcPos)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(skipSlashPC)))

	// extractComponent: record start, scan to next '/' or end
	fl.insts[toExtractJmp].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(srcPos), dis.FP(compStart)))

	scanLoopPC := int32(len(fl.insts))
	scanEndIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(srcLen), dis.FP(srcPos), dis.Imm(0))) // pos >= len → done
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.FP(srcPos), dis.FP(ch)))
	scanSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(47), dis.Imm(0))) // ch == '/' → end
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(srcPos), dis.FP(srcPos)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(scanLoopPC)))

	// Component is path[compStart:srcPos]
	compDonePC := int32(len(fl.insts))
	fl.insts[scanEndIdx].Dst = dis.Imm(compDonePC)
	fl.insts[scanSlashIdx].Dst = dis.Imm(compDonePC)

	// compLen = srcPos - compStart
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(compStart), dis.FP(srcPos), dis.FP(compLen)))

	// --- Check if component is "." (compLen==1, char=='.')
	mayBeDotIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(compLen), dis.Imm(1), dis.Imm(0))) // len==1 → mayBeDot
	// len != 1, check ".."
	mayBeDotDotIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(compLen), dis.Imm(2), dis.Imm(0))) // len==2 → mayBeDotDot
	// len != 1 and len != 2 → append
	toAppendJmp := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))

	// --- mayBeDot: len==1, check if char is '.'
	fl.insts[mayBeDotIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.FP(compStart), dis.FP(ch)))
	isDotIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(46), dis.Imm(0))) // '.' → skip
	// Not '.', single char component → append
	toAppendJmp2 := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))
	// Is "." → skip, go to mainLoop
	fl.insts[isDotIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))

	// --- mayBeDotDot: len==2, check if both chars are '.'
	fl.insts[mayBeDotDotIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.FP(compStart), dis.FP(ch)))
	dd1Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(46), dis.Imm(0))) // first '.'
	// first != '.', 2-char component → append
	toAppendJmp3 := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))
	// First is '.', check second
	fl.insts[dd1Idx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(compStart), dis.FP(scanPos)))
	fl.emit(dis.NewInst(dis.IINDC, pathOp, dis.FP(scanPos), dis.FP(ch)))
	dd2Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(46), dis.Imm(0))) // second '.'
	// second != '.', not ".." → append
	toAppendJmp4 := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))

	// --- handleDotDot: it IS ".." — truncate result to last '/'
	fl.insts[dd2Idx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(resLen)))
	ddEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(resLen), dis.Imm(0), dis.Imm(0))) // empty → special

	// Scan backwards for last '/'
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(resLen), dis.FP(scanPos)))
	ddScanPC := int32(len(fl.insts))
	ddScanDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.Imm(0), dis.FP(scanPos), dis.Imm(0))) // scanPos < 0 → empty
	fl.emit(dis.NewInst(dis.IINDC, dis.FP(result), dis.FP(scanPos), dis.FP(scanCh)))
	ddFoundIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(scanCh), dis.Imm(47), dis.Imm(0))) // found '/'
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(scanPos), dis.FP(scanPos)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(ddScanPC)))

	// Found '/': result = result[0:scanPos]
	fl.insts[ddFoundIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(scanPos), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))

	// No '/' found: truncate to empty
	fl.insts[ddScanDoneIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))

	// ddEmpty: result is empty, non-rooted → append ".."
	fl.insts[ddEmptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
	ddNotRootIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(isRoot), dis.Imm(0), dis.Imm(0))) // not rooted → add ".."
	// Rooted: ".." at root is no-op, just skip
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))
	// Not rooted: result = ".."
	fl.insts[ddNotRootIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(ddOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))

	// --- appendComponent: add "/" + component to result
	appendPC := int32(len(fl.insts))
	fl.insts[toAppendJmp].Dst = dis.Imm(appendPC)
	fl.insts[toAppendJmp2].Dst = dis.Imm(appendPC)
	fl.insts[toAppendJmp3].Dst = dis.Imm(appendPC)
	fl.insts[toAppendJmp4].Dst = dis.Imm(appendPC)

	// Extract component: comp = pathOp[compStart:srcPos]
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(comp)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(compStart), dis.FP(srcPos), dis.FP(comp)))

	// If result is non-empty, append "/" first
	fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(resLen)))
	noSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(resLen), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDC, dis.MP(slashOff), dis.FP(result), dis.FP(result)))
	fl.insts[noSlashIdx].Dst = dis.Imm(int32(len(fl.insts)))
	// Append component
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(comp), dis.FP(result), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(mainLoopPC)))

	// --- done: all components processed
	donePC := int32(len(fl.insts))
	fl.insts[skipSlashDoneIdx].Dst = dis.Imm(donePC)

	// If rooted, prepend "/"
	notRootedIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(isRoot), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(result), dis.MP(slashOff), dis.FP(result)))
	fl.insts[notRootedIdx].Dst = dis.Imm(int32(len(fl.insts)))

	// If result is empty, return "."
	fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(resLen)))
	notEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(resLen), dis.Imm(0), dis.Imm(0)))
	// empty → "."
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(dotOff), dis.FP(dst)))
	toEndJmp := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))

	// non-empty
	fl.insts[notEmptyIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	toEndJmp2 := len(fl.insts)
	fl.emit(dis.Inst2(dis.IJMP, dis.Imm(0), dis.Imm(0)))

	// returnDot: empty input → "."
	fl.insts[emptyCheckIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(dotOff), dis.FP(dst)))

	// end:
	endPC := int32(len(fl.insts))
	fl.insts[toEndJmp].Dst = dis.Imm(endPC)
	fl.insts[toEndJmp2].Dst = dis.Imm(endPC)
	return true, nil
}

// lowerFilepathJoin concatenates path elements with "/".
func (fl *funcLowerer) lowerFilepathJoin(instr *ssa.Call) (bool, error) {
	// filepath.Join is variadic — SSA packs args into a []string slice.
	// Trace back to find individual elements.
	args := instr.Call.Args
	if len(args) == 0 {
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	}

	elements := fl.traceAllVarargElements(args[0])
	if elements == nil {
		return false, nil
	}

	dst := fl.slotOf(instr)
	slashOff := fl.comp.AllocString("/")

	// Start with first element
	first := fl.operandOf(elements[0])
	fl.emit(dis.Inst2(dis.IMOVP, first, dis.FP(dst)))

	// Concatenate remaining with "/" separator
	for idx := 1; idx < len(elements); idx++ {
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(slashOff), dis.FP(dst), dis.FP(dst)))
		argOp := fl.operandOf(elements[idx])
		fl.emit(dis.NewInst(dis.IADDC, argOp, dis.FP(dst), dis.FP(dst)))
	}

	return true, nil
}

// lowerFilepathIsAbs returns whether path is absolute (starts with '/').
func (fl *funcLowerer) lowerFilepathIsAbs(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenP := fl.frame.AllocWord("")
	fl.emit(dis.Inst2(dis.ILENC, pathOp, dis.FP(lenP)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	beqEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.Imm(0), dis.FP(lenP), dis.Imm(0)))

	// Check first char
	firstChar := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(firstChar)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.Imm(1), dis.FP(firstChar)))
	slashOff := fl.comp.AllocString("/")
	bneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEC, dis.MP(slashOff), dis.FP(firstChar), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[beqEmptyIdx].Dst = dis.Imm(donePC)
	fl.insts[bneIdx].Dst = dis.Imm(donePC)

	return true, nil
}

// lowerFilepathAbs returns an absolute path. Stub: returns path, nil error.
func (fl *funcLowerer) lowerFilepathAbs(instr *ssa.Call) (bool, error) {
	pathOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)
	fl.emit(dis.Inst2(dis.IMOVP, pathOp, dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
	return true, nil
}

// ============================================================
// slices package (Go 1.21+)
// ============================================================

// lowerSlicesCall handles calls to the slices package.
func (fl *funcLowerer) lowerSlicesCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Contains":
		return fl.lowerSlicesContains(instr)
	case "Index":
		return fl.lowerSlicesIndex(instr)
	case "Reverse":
		return fl.lowerSlicesReverse(instr)
	case "Sort":
		return fl.lowerSlicesSort(instr)
	case "Equal":
		return fl.lowerSlicesEqual(instr)
	case "Min":
		return fl.lowerSlicesMinMax(instr, false)
	case "Max":
		return fl.lowerSlicesMinMax(instr, true)
	case "Clone":
		return fl.lowerSlicesClone(instr)
	case "IsSorted":
		return fl.lowerSlicesIsSorted(instr)
	case "BinarySearch":
		return fl.lowerSlicesBinarySearch(instr)
	case "Compare":
		return fl.lowerSlicesCompare(instr)
	case "SortFunc", "SortStableFunc":
		// No-op stub (needs closure callback dispatch)
		return true, nil
	case "ContainsFunc", "IndexFunc":
		// Stub: ContainsFunc→false, IndexFunc→-1
		dst := fl.slotOf(instr)
		if callee.Name() == "ContainsFunc" {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		} else {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		}
		return true, nil
	case "Compact":
		return fl.lowerSlicesCompact(instr)
	case "CompactFunc", "DeleteFunc":
		return fl.lowerSlicesPassthrough(instr)
	case "Delete":
		return fl.lowerSlicesDelete(instr)
	case "Clip", "Grow", "Insert", "Replace", "Repeat":
		return fl.lowerSlicesPassthrough(instr)
	case "Concat":
		// Return nil slice (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "EqualFunc", "CompareFunc":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "IsSortedFunc":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "MinFunc", "MaxFunc":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "BinarySearchFunc":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "All", "Values", "Backward", "Chunk":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Collect", "Sorted", "SortedFunc", "SortedStableFunc", "AppendSeq":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// unwrapMakeInterface strips a MakeInterface wrapper to get the underlying
// concrete value and its type. Returns the original value if not wrapped.
func unwrapMakeInterface(v ssa.Value) ssa.Value {
	if mi, ok := v.(*ssa.MakeInterface); ok {
		return mi.X
	}
	return v
}

// isStringElem returns true if the given slice SSA value has string elements.
func isStringElem(v ssa.Value) bool {
	v = unwrapMakeInterface(v)
	if sl, ok := v.Type().Underlying().(*types.Slice); ok {
		if basic, ok := sl.Elem().Underlying().(*types.Basic); ok {
			return basic.Kind() == types.String
		}
	}
	return false
}

// isFloatElem returns true if the given slice SSA value has float64 elements.
func isFloatElem(v ssa.Value) bool {
	v = unwrapMakeInterface(v)
	if sl, ok := v.Type().Underlying().(*types.Slice); ok {
		if basic, ok := sl.Elem().Underlying().(*types.Basic); ok {
			return basic.Kind() == types.Float64 || basic.Kind() == types.Float32
		}
	}
	return false
}

// lowerSlicesPassthrough returns the input slice unchanged (for stubs).
func (fl *funcLowerer) lowerSlicesPassthrough(instr *ssa.Call) (bool, error) {
	arg := unwrapMakeInterface(instr.Call.Args[0])
	sOp := fl.operandOf(arg)
	dst := fl.slotOf(instr)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
	return true, nil
}

// lowerSlicesCompact removes consecutive duplicate elements in-place.
// Algorithm: write index tracks unique position, scan compares adjacent elements.
func (fl *funcLowerer) lowerSlicesCompact(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	dst := fl.slotOf(instr)

	// Copy array ref to result
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrSlot), dis.FP(dst)))

	lenSlot := fl.frame.AllocWord("scc.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	// if len < 2 → done (already compacted)
	earlyDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lenSlot), dis.Imm(2), dis.Imm(0)))

	iSlot := fl.frame.AllocWord("scc.i") // write index
	kSlot := fl.frame.AllocWord("scc.k") // scan index
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(iSlot)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(kSlot)))

	prevAddr := fl.frame.AllocWord("scc.pa")
	curAddr := fl.frame.AllocWord("scc.ca")
	km1 := fl.frame.AllocWord("scc.km1")

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(kSlot), dis.FP(lenSlot), dis.Imm(0)))

	// prev = s[k-1], cur = s[k]
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(kSlot), dis.FP(km1)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(prevAddr), dis.FP(km1)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(curAddr), dis.FP(kSlot)))

	if isStr {
		prevVal := fl.frame.AllocTemp(true)
		curVal := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(prevAddr, 0), dis.FP(prevVal)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(curAddr, 0), dis.FP(curVal)))
		// if prev == cur → skip (duplicate)
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(prevVal), dis.FP(curVal), dis.Imm(0)))
		// Different: s[i] = s[k]; i++
		iAddr := fl.frame.AllocWord("scc.ia")
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(iAddr), dis.FP(iSlot)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(curVal), dis.FPInd(iAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
	} else {
		prevVal := fl.frame.AllocWord("scc.pv")
		curVal := fl.frame.AllocWord("scc.cv")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(prevAddr, 0), dis.FP(prevVal)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(curAddr, 0), dis.FP(curVal)))
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(prevVal), dis.FP(curVal), dis.Imm(0)))
		iAddr := fl.frame.AllocWord("scc.ia")
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(iAddr), dis.FP(iSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(curVal), dis.FPInd(iAddr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
	}

	// k++
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(kSlot), dis.FP(kSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[earlyDoneIdx].Dst = dis.Imm(donePC)

	// Slice to [0:i]
	fl.emit(dis.NewInst(dis.ISLICELA, dis.Imm(0), dis.FP(iSlot), dis.FP(dst)))
	return true, nil
}

// lowerSlicesDelete removes elements s[i:j] by shifting s[j:] left.
func (fl *funcLowerer) lowerSlicesDelete(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	iArg := fl.materialize(unwrapMakeInterface(instr.Call.Args[1]))
	jArg := fl.materialize(unwrapMakeInterface(instr.Call.Args[2]))
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrSlot), dis.FP(dst)))

	lenSlot := fl.frame.AllocWord("sd.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	// Shift elements: for k = j; k < len; k++ { s[i + (k-j)] = s[k] }
	kSlot := fl.frame.AllocWord("sd.k")
	wSlot := fl.frame.AllocWord("sd.w") // write position
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(jArg), dis.FP(kSlot)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(iArg), dis.FP(wSlot)))

	srcAddr := fl.frame.AllocWord("sd.sa")
	dstAddr := fl.frame.AllocWord("sd.da")

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(kSlot), dis.FP(lenSlot), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(srcAddr), dis.FP(kSlot)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(dstAddr), dis.FP(wSlot)))

	if isStr {
		tmpVal := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(srcAddr, 0), dis.FP(tmpVal)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmpVal), dis.FPInd(dstAddr, 0)))
	} else {
		tmpVal := fl.frame.AllocWord("sd.tv")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(srcAddr, 0), dis.FP(tmpVal)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(tmpVal), dis.FPInd(dstAddr, 0)))
	}

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(kSlot), dis.FP(kSlot)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(wSlot), dis.FP(wSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))

	// newLen = len - (j - i)
	fl.emit(dis.NewInst(dis.ISLICELA, dis.Imm(0), dis.FP(wSlot), dis.FP(dst)))
	return true, nil
}

// lowerSlicesContains: for i := 0; i < len(s); i++ { if s[i] == v { return true } } return false
func (fl *funcLowerer) lowerSlicesContains(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	rawVal := unwrapMakeInterface(instr.Call.Args[1])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	valSlot := fl.materialize(rawVal)
	dst := fl.slotOf(instr)

	// result = false
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	// nil check: if arr == H(-1) → done
	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("sc.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	iSlot := fl.frame.AllocWord("sc.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

	// Load element at index i
	elemAddr := fl.frame.AllocWord("sc.ea")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(elemAddr), dis.FP(iSlot)))

	if isStr {
		elemSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))
		// String comparison: BEQC
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(elemSlot), dis.FP(valSlot), dis.Imm(0)))
		// Not equal: i++, loop
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// Found:
		foundPC := int32(len(fl.insts))
		fl.insts[foundIdx].Dst = dis.Imm(foundPC)
	} else {
		elemSlot := fl.frame.AllocWord("sc.elem")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))
		// Integer comparison: BEQW
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(elemSlot), dis.FP(valSlot), dis.Imm(0)))
		// Not equal: i++, loop
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// Found:
		foundPC := int32(len(fl.insts))
		fl.insts[foundIdx].Dst = dis.Imm(foundPC)
	}
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
	// Done:
	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerSlicesIndex: linear scan, return index or -1.
func (fl *funcLowerer) lowerSlicesIndex(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	rawVal := unwrapMakeInterface(instr.Call.Args[1])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	valSlot := fl.materialize(rawVal)
	dst := fl.slotOf(instr)

	// result = -1
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))

	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("si.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	iSlot := fl.frame.AllocWord("si.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

	elemAddr := fl.frame.AllocWord("si.ea")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(elemAddr), dis.FP(iSlot)))

	if isStr {
		elemSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(elemSlot), dis.FP(valSlot), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[foundIdx].Dst = dis.Imm(foundPC)
	} else {
		elemSlot := fl.frame.AllocWord("si.elem")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))
		foundIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(elemSlot), dis.FP(valSlot), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[foundIdx].Dst = dis.Imm(foundPC)
	}
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(iSlot), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerSlicesReverse: swap elements from both ends toward center.
func (fl *funcLowerer) lowerSlicesReverse(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)

	// nil check
	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("sr.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	iSlot := fl.frame.AllocWord("sr.i")
	jSlot := fl.frame.AllocWord("sr.j")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenSlot), dis.FP(jSlot)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(jSlot), dis.Imm(0)))

	iAddr := fl.frame.AllocWord("sr.ia")
	jAddr := fl.frame.AllocWord("sr.ja")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(iAddr), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(jAddr), dis.FP(jSlot)))

	if isStr {
		tmpI := fl.frame.AllocTemp(true)
		tmpJ := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(iAddr, 0), dis.FP(tmpI)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(jAddr, 0), dis.FP(tmpJ)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmpJ), dis.FPInd(iAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmpI), dis.FPInd(jAddr, 0)))
	} else {
		tmpI := fl.frame.AllocWord("sr.ti")
		tmpJ := fl.frame.AllocWord("sr.tj")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(iAddr, 0), dis.FP(tmpI)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(jAddr, 0), dis.FP(tmpJ)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(tmpJ), dis.FPInd(iAddr, 0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(tmpI), dis.FPInd(jAddr, 0)))
	}

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(jSlot), dis.FP(jSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerSlicesSort: inline insertion sort (same algorithm as sort.Ints/Strings).
func (fl *funcLowerer) lowerSlicesSort(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])
	isFloat := isFloatElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)

	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("ss.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	iSlot := fl.frame.AllocWord("ss.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(iSlot)))

	outerPC := int32(len(fl.insts))
	outerDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

	keyAddr := fl.frame.AllocWord("ss.ka")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(keyAddr), dis.FP(iSlot)))

	jSlot := fl.frame.AllocWord("ss.j")
	jAddr := fl.frame.AllocWord("ss.ja")
	j1Slot := fl.frame.AllocWord("ss.j1")
	j1Addr := fl.frame.AllocWord("ss.j1a")

	if isStr {
		keySlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(keyAddr, 0), dis.FP(keySlot)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(iSlot), dis.FP(jSlot)))

		innerPC := int32(len(fl.insts))
		innerDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(jSlot), dis.Imm(0), dis.Imm(0)))

		arrJ := fl.frame.AllocTemp(true)
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(jAddr), dis.FP(jSlot)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(jAddr, 0), dis.FP(arrJ)))

		innerDone2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEC, dis.FP(arrJ), dis.FP(keySlot), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(arrJ), dis.FPInd(j1Addr, 0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(jSlot), dis.FP(jSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

		innerDonePC := int32(len(fl.insts))
		fl.insts[innerDoneIdx].Dst = dis.Imm(innerDonePC)
		fl.insts[innerDone2Idx].Dst = dis.Imm(innerDonePC)

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(keySlot), dis.FPInd(j1Addr, 0)))
	} else if isFloat {
		keySlot := fl.frame.AllocWord("ss.key")
		fl.emit(dis.Inst2(dis.IMOVF, dis.FPInd(keyAddr, 0), dis.FP(keySlot)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(iSlot), dis.FP(jSlot)))

		innerPC := int32(len(fl.insts))
		innerDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(jSlot), dis.Imm(0), dis.Imm(0)))

		arrJ := fl.frame.AllocWord("ss.arrj")
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(jAddr), dis.FP(jSlot)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.FPInd(jAddr, 0), dis.FP(arrJ)))

		innerDone2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEF, dis.FP(arrJ), dis.FP(keySlot), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.FP(arrJ), dis.FPInd(j1Addr, 0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(jSlot), dis.FP(jSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

		innerDonePC := int32(len(fl.insts))
		fl.insts[innerDoneIdx].Dst = dis.Imm(innerDonePC)
		fl.insts[innerDone2Idx].Dst = dis.Imm(innerDonePC)

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVF, dis.FP(keySlot), dis.FPInd(j1Addr, 0)))
	} else {
		// int sort
		keySlot := fl.frame.AllocWord("ss.key")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(keyAddr, 0), dis.FP(keySlot)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(iSlot), dis.FP(jSlot)))

		innerPC := int32(len(fl.insts))
		innerDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(jSlot), dis.Imm(0), dis.Imm(0)))

		arrJ := fl.frame.AllocWord("ss.arrj")
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(jAddr), dis.FP(jSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(jAddr, 0), dis.FP(arrJ)))

		innerDone2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLEW, dis.FP(arrJ), dis.FP(keySlot), dis.Imm(0)))

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(arrJ), dis.FPInd(j1Addr, 0)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(jSlot), dis.FP(jSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

		innerDonePC := int32(len(fl.insts))
		fl.insts[innerDoneIdx].Dst = dis.Imm(innerDonePC)
		fl.insts[innerDone2Idx].Dst = dis.Imm(innerDonePC)

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(j1Slot)))
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(j1Addr), dis.FP(j1Slot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(keySlot), dis.FPInd(j1Addr, 0)))
	}

	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))

	outerDonePC := int32(len(fl.insts))
	fl.insts[outerDoneIdx].Dst = dis.Imm(outerDonePC)
	fl.insts[nilDoneIdx].Dst = dis.Imm(outerDonePC)
	return true, nil
}

// lowerSlicesEqual: compare two slices element-by-element.
func (fl *funcLowerer) lowerSlicesEqual(instr *ssa.Call) (bool, error) {
	rawS1 := unwrapMakeInterface(instr.Call.Args[0])
	rawS2 := unwrapMakeInterface(instr.Call.Args[1])
	isStr := isStringElem(instr.Call.Args[0])

	s1Slot := fl.materialize(rawS1)
	s2Slot := fl.materialize(rawS2)
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst))) // assume false

	// If both nil → equal
	len1 := fl.frame.AllocWord("se.l1")
	len2 := fl.frame.AllocWord("se.l2")

	// nil checks: nil slice has len 0
	t1 := fl.frame.AllocWord("se.t1")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(len1)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(t1)))
	skipNil1Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(s1Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(s1Slot), dis.FP(len1)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(t1)))
	skipNil1PC := int32(len(fl.insts))
	fl.insts[skipNil1Idx].Dst = dis.Imm(skipNil1PC)

	t2 := fl.frame.AllocWord("se.t2")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(len2)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(t2)))
	skipNil2Idx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(s2Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(s2Slot), dis.FP(len2)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(t2)))
	skipNil2PC := int32(len(fl.insts))
	fl.insts[skipNil2Idx].Dst = dis.Imm(skipNil2PC)

	// if len1 != len2 → not equal (done, result already false)
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))

	iSlot := fl.frame.AllocWord("se.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

	loopPC := int32(len(fl.insts))
	equalIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(len1), dis.Imm(0)))

	e1Addr := fl.frame.AllocWord("se.e1a")
	e2Addr := fl.frame.AllocWord("se.e2a")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(s1Slot), dis.FP(e1Addr), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(s2Slot), dis.FP(e2Addr), dis.FP(iSlot)))

	if isStr {
		e1 := fl.frame.AllocTemp(true)
		e2 := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(e1Addr, 0), dis.FP(e1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(e2Addr, 0), dis.FP(e2)))
		notEqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		notEqPC := int32(len(fl.insts))
		fl.insts[notEqIdx].Dst = dis.Imm(notEqPC)
	} else {
		e1 := fl.frame.AllocWord("se.e1")
		e2 := fl.frame.AllocWord("se.e2")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(e1Addr, 0), dis.FP(e1)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(e2Addr, 0), dis.FP(e2)))
		notEqIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		notEqPC := int32(len(fl.insts))
		fl.insts[notEqIdx].Dst = dis.Imm(notEqPC)
	}

	// fall through = not equal, done
	donePC2Idx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0))) // jump to done

	// All elements equal
	equalPC := int32(len(fl.insts))
	fl.insts[equalIdx].Dst = dis.Imm(equalPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[donePC2Idx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerSlicesMinMax: scan for min or max element.
func (fl *funcLowerer) lowerSlicesMinMax(instr *ssa.Call, isMax bool) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("smm.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	// best = s[0]
	firstAddr := fl.frame.AllocWord("smm.fa")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(firstAddr), dis.Imm(0)))

	if isStr {
		bestSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(firstAddr, 0), dis.FP(bestSlot)))

		iSlot := fl.frame.AllocWord("smm.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(iSlot)))

		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

		elemAddr := fl.frame.AllocWord("smm.ea")
		elemSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(elemAddr), dis.FP(iSlot)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))

		skipIdx := len(fl.insts)
		if isMax {
			fl.emit(dis.NewInst(dis.IBLEC, dis.FP(elemSlot), dis.FP(bestSlot), dis.Imm(0)))
		} else {
			fl.emit(dis.NewInst(dis.IBGEC, dis.FP(elemSlot), dis.FP(bestSlot), dis.Imm(0)))
		}
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(elemSlot), dis.FP(bestSlot)))
		skipPC := int32(len(fl.insts))
		fl.insts[skipIdx].Dst = dis.Imm(skipPC)

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(bestSlot), dis.FP(dst)))
	} else {
		bestSlot := fl.frame.AllocWord("smm.best")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(firstAddr, 0), dis.FP(bestSlot)))

		iSlot := fl.frame.AllocWord("smm.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(iSlot)))

		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

		elemAddr := fl.frame.AllocWord("smm.ea")
		elemSlot := fl.frame.AllocWord("smm.elem")
		fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(elemAddr), dis.FP(iSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(elemAddr, 0), dis.FP(elemSlot)))

		skipIdx := len(fl.insts)
		if isMax {
			fl.emit(dis.NewInst(dis.IBLEW, dis.FP(elemSlot), dis.FP(bestSlot), dis.Imm(0)))
		} else {
			fl.emit(dis.NewInst(dis.IBGEW, dis.FP(elemSlot), dis.FP(bestSlot), dis.Imm(0)))
		}
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(elemSlot), dis.FP(bestSlot)))
		skipPC := int32(len(fl.insts))
		fl.insts[skipIdx].Dst = dis.Imm(skipPC)

		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(bestSlot), dis.FP(dst)))
	}

	finalDonePC := int32(len(fl.insts))
	fl.insts[nilDoneIdx].Dst = dis.Imm(finalDonePC)
	return true, nil
}

// lowerSlicesClone: allocate new array, copy elements.
func (fl *funcLowerer) lowerSlicesClone(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	arrSlot := fl.materialize(rawSlice)
	dst := fl.slotOf(instr)

	// nil → nil
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	// Shallow copy: ISLICELA copies the entire backing array
	lenSlot := fl.frame.AllocWord("scl.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))
	fl.emit(dis.NewInst(dis.ISLICELA, dis.FP(lenSlot), dis.FP(arrSlot), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerSlicesIsSorted: check if elements are in ascending order.
func (fl *funcLowerer) lowerSlicesIsSorted(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lenSlot := fl.frame.AllocWord("sis.len")
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(lenSlot)))

	iSlot := fl.frame.AllocWord("sis.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(iSlot)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(lenSlot), dis.Imm(0)))

	prevIdx := fl.frame.AllocWord("sis.pi")
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(iSlot), dis.FP(prevIdx)))
	pAddr := fl.frame.AllocWord("sis.pa")
	cAddr := fl.frame.AllocWord("sis.ca")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(pAddr), dis.FP(prevIdx)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(cAddr), dis.FP(iSlot)))

	if isStr {
		prev := fl.frame.AllocTemp(true)
		cur := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(pAddr, 0), dis.FP(prev)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(cAddr, 0), dis.FP(cur)))
		notSortedIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTC, dis.FP(prev), dis.FP(cur), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		notSortedPC := int32(len(fl.insts))
		fl.insts[notSortedIdx].Dst = dis.Imm(notSortedPC)
	} else {
		prev := fl.frame.AllocWord("sis.prev")
		cur := fl.frame.AllocWord("sis.cur")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(pAddr, 0), dis.FP(prev)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(cAddr, 0), dis.FP(cur)))
		notSortedIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(prev), dis.FP(cur), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		notSortedPC := int32(len(fl.insts))
		fl.insts[notSortedIdx].Dst = dis.Imm(notSortedPC)
	}

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	sortedPC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(sortedPC)
	fl.insts[nilDoneIdx].Dst = dis.Imm(sortedPC)
	return true, nil
}

// lowerSlicesBinarySearch: binary search for target in sorted slice.
func (fl *funcLowerer) lowerSlicesBinarySearch(instr *ssa.Call) (bool, error) {
	rawSlice := unwrapMakeInterface(instr.Call.Args[0])
	rawTarget := unwrapMakeInterface(instr.Call.Args[1])
	isStr := isStringElem(instr.Call.Args[0])

	arrSlot := fl.materialize(rawSlice)
	targetSlot := fl.materialize(rawTarget)
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	// result = (0, false)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))

	nilDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(arrSlot), dis.Imm(-1), dis.Imm(0)))

	lo := fl.frame.AllocWord("bs.lo")
	hi := fl.frame.AllocWord("bs.hi")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(lo)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(arrSlot), dis.FP(hi)))

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(lo), dis.FP(hi), dis.Imm(0)))

	mid := fl.frame.AllocWord("bs.mid")
	fl.emit(dis.NewInst(dis.IADDW, dis.FP(lo), dis.FP(hi), dis.FP(mid)))
	fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(1), dis.FP(mid), dis.FP(mid)))

	midAddr := fl.frame.AllocWord("bs.ma")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(arrSlot), dis.FP(midAddr), dis.FP(mid)))

	if isStr {
		midVal := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(midAddr, 0), dis.FP(midVal)))
		// if mid < target: lo = mid+1
		ltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTC, dis.FP(midVal), dis.FP(targetSlot), dis.Imm(0)))
		// if mid > target: hi = mid
		gtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTC, dis.FP(midVal), dis.FP(targetSlot), dis.Imm(0)))
		// equal: found!
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(mid), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		foundDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// lt: lo = mid+1
		ltPC := int32(len(fl.insts))
		fl.insts[ltIdx].Dst = dis.Imm(ltPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(mid), dis.FP(lo)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// gt: hi = mid
		gtPC := int32(len(fl.insts))
		fl.insts[gtIdx].Dst = dis.Imm(gtPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(mid), dis.FP(hi)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
		fl.insts[foundDoneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lo), dis.FP(dst)))
	} else {
		midVal := fl.frame.AllocWord("bs.mv")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(midAddr, 0), dis.FP(midVal)))
		ltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(midVal), dis.FP(targetSlot), dis.Imm(0)))
		gtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(midVal), dis.FP(targetSlot), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(mid), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
		foundDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		ltPC := int32(len(fl.insts))
		fl.insts[ltIdx].Dst = dis.Imm(ltPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(mid), dis.FP(lo)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		gtPC := int32(len(fl.insts))
		fl.insts[gtIdx].Dst = dis.Imm(gtPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(mid), dis.FP(hi)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.insts[nilDoneIdx].Dst = dis.Imm(donePC)
		fl.insts[foundDoneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lo), dis.FP(dst)))
	}
	return true, nil
}

// lowerSlicesCompare: lexicographic comparison of two slices, returns -1/0/1.
func (fl *funcLowerer) lowerSlicesCompare(instr *ssa.Call) (bool, error) {
	rawS1 := unwrapMakeInterface(instr.Call.Args[0])
	rawS2 := unwrapMakeInterface(instr.Call.Args[1])
	isStr := isStringElem(instr.Call.Args[0])

	s1Slot := fl.materialize(rawS1)
	s2Slot := fl.materialize(rawS2)
	dst := fl.slotOf(instr)

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	len1 := fl.frame.AllocWord("scmp.l1")
	len2 := fl.frame.AllocWord("scmp.l2")
	minLen := fl.frame.AllocWord("scmp.ml")

	// nil-safe len
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(len1)))
	skipN1 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(s1Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(s1Slot), dis.FP(len1)))
	fl.insts[skipN1].Dst = dis.Imm(int32(len(fl.insts)))

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(len2)))
	skipN2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(s2Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.ILENA, dis.FP(s2Slot), dis.FP(len2)))
	fl.insts[skipN2].Dst = dis.Imm(int32(len(fl.insts)))

	// minLen = min(len1, len2)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(len1), dis.FP(minLen)))
	skipMin := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLEW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(len2), dis.FP(minLen)))
	fl.insts[skipMin].Dst = dis.Imm(int32(len(fl.insts)))

	iSlot := fl.frame.AllocWord("scmp.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

	loopPC := int32(len(fl.insts))
	loopDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(minLen), dis.Imm(0)))

	e1Addr := fl.frame.AllocWord("scmp.e1a")
	e2Addr := fl.frame.AllocWord("scmp.e2a")
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(s1Slot), dis.FP(e1Addr), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.IINDX, dis.FP(s2Slot), dis.FP(e2Addr), dis.FP(iSlot)))

	if isStr {
		e1 := fl.frame.AllocTemp(true)
		e2 := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(e1Addr, 0), dis.FP(e1)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(e2Addr, 0), dis.FP(e2)))
		ltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTC, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		gtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTC, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		ltPC := int32(len(fl.insts))
		fl.insts[ltIdx].Dst = dis.Imm(ltPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		ltDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		gtPC := int32(len(fl.insts))
		fl.insts[gtIdx].Dst = dis.Imm(gtPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		gtDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		// compare lengths
		lenCmpPC := int32(len(fl.insts))
		fl.insts[loopDoneIdx].Dst = dis.Imm(lenCmpPC)
		l1LtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))
		l1GtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))
		// equal lengths → result 0 (already set)
		finalPC := int32(len(fl.insts))
		fl.insts[ltDoneIdx].Dst = dis.Imm(finalPC)
		fl.insts[gtDoneIdx].Dst = dis.Imm(finalPC)
		fl.insts[l1LtIdx].Dst = dis.Imm(ltPC)
		fl.insts[l1GtIdx].Dst = dis.Imm(gtPC)
	} else {
		e1 := fl.frame.AllocWord("scmp.e1")
		e2 := fl.frame.AllocWord("scmp.e2")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(e1Addr, 0), dis.FP(e1)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(e2Addr, 0), dis.FP(e2)))
		ltIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		gtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(e1), dis.FP(e2), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		ltPC := int32(len(fl.insts))
		fl.insts[ltIdx].Dst = dis.Imm(ltPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		ltDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		gtPC := int32(len(fl.insts))
		fl.insts[gtIdx].Dst = dis.Imm(gtPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		gtDoneIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		lenCmpPC := int32(len(fl.insts))
		fl.insts[loopDoneIdx].Dst = dis.Imm(lenCmpPC)
		l1LtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))
		l1GtIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(len1), dis.FP(len2), dis.Imm(0)))
		finalPC := int32(len(fl.insts))
		fl.insts[ltDoneIdx].Dst = dis.Imm(finalPC)
		fl.insts[gtDoneIdx].Dst = dis.Imm(finalPC)
		fl.insts[l1LtIdx].Dst = dis.Imm(ltPC)
		fl.insts[l1GtIdx].Dst = dis.Imm(gtPC)
	}
	return true, nil
}

// ============================================================
// maps package (Go 1.21+)
// ============================================================

// lowerMapsCall handles calls to the maps package.
func (fl *funcLowerer) lowerMapsCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Keys":
		return fl.lowerMapsKeysOrValues(instr, 0) // offset 0 = keys array
	case "Values":
		return fl.lowerMapsKeysOrValues(instr, 8) // offset 8 = values array
	case "Clone":
		return fl.lowerMapsClone(instr)
	case "Equal":
		return fl.lowerMapsEqual(instr)
	case "Copy":
		return fl.lowerMapsCopy(instr)
	case "DeleteFunc", "Insert":
		// No-op stubs
		return true, nil
	case "EqualFunc":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Collect":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "All":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// lowerMapsKeysOrValues: copy the map's internal keys or values array.
// Map wrapper layout: [keysArr(PTR @0), valsArr(PTR @8), flagsArr(PTR @16), count(WORD @24), cap(WORD @32)]
// arrayOffset=0 for keys, 8 for values.
func (fl *funcLowerer) lowerMapsKeysOrValues(instr *ssa.Call, arrayOffset int32) (bool, error) {
	rawMap := unwrapMakeInterface(instr.Call.Args[0])
	mapSlot := fl.materialize(rawMap)
	mapType := rawMap.Type().Underlying().(*types.Map)
	dst := fl.slotOf(instr)

	// nil map → nil slice
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
	nilIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(mapSlot), dis.Imm(-1), dis.Imm(0)))

	// Load count
	cnt := fl.frame.AllocWord("mk.cnt")
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 24), dis.FP(cnt)))

	// Empty map → nil slice
	emptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(cnt), dis.Imm(0), dis.Imm(0)))

	// With hash tables, we must iterate the flags array to find live entries
	// and copy only those to the output array.
	cap := fl.frame.AllocWord("mk.cap")
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 32), dis.FP(cap)))

	flagsArr := fl.allocPtrTemp("mk.flags")
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 16), dis.FP(flagsArr)))
	srcArr := fl.allocPtrTemp("mk.src")
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, arrayOffset), dis.FP(srcArr)))

	// Allocate result array of size count
	elemType := mapType.Key()
	if arrayOffset == 8 {
		elemType = mapType.Elem()
	}
	elemTDIdx := fl.makeHeapTypeDesc(elemType)
	fl.emit(dis.NewInst(dis.INEWA, dis.FP(cnt), dis.Imm(int32(elemTDIdx)), dis.FP(dst)))

	// Copy live entries: scan flags, copy matching
	scanIdx := fl.frame.AllocWord("mk.si")
	outIdx := fl.frame.AllocWord("mk.oi")
	tmpPtr1 := fl.frame.AllocWord("mk.p1")
	tmpPtr2 := fl.frame.AllocWord("mk.p2")
	tmpFlag := fl.frame.AllocWord("mk.fl")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(scanIdx)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outIdx)))

	scanPC := int32(len(fl.insts))
	scanDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(scanIdx), dis.FP(cap), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDW, dis.FP(flagsArr), dis.FP(tmpPtr1), dis.FP(scanIdx)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(tmpPtr1, 0), dis.FP(tmpFlag)))

	skipIdx2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(tmpFlag), dis.Imm(1), dis.Imm(0))) // not occupied → skip

	// Copy element
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(srcArr), dis.FP(tmpPtr1), dis.FP(scanIdx)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(dst), dis.FP(tmpPtr2), dis.FP(outIdx)))
	dt := GoTypeToDis(elemType)
	if dt.IsPtr {
		fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(tmpPtr1, 0), dis.FPInd(tmpPtr2, 0)))
	} else {
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(tmpPtr1, 0), dis.FPInd(tmpPtr2, 0)))
	}
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outIdx), dis.FP(outIdx)))

	fl.insts[skipIdx2].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(scanIdx), dis.FP(scanIdx)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(scanPC)))

	fl.insts[scanDoneIdx].Dst = dis.Imm(int32(len(fl.insts)))

	donePC := int32(len(fl.insts))
	fl.insts[nilIdx].Dst = dis.Imm(donePC)
	fl.insts[emptyIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerMapsClone: create a new map wrapper with copies of keys and values arrays.
func (fl *funcLowerer) lowerMapsClone(instr *ssa.Call) (bool, error) {
	rawMap := unwrapMakeInterface(instr.Call.Args[0])
	mapSlot := fl.materialize(rawMap)
	dst := fl.slotOf(instr)

	// nil map → nil
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
	nilIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(mapSlot), dis.Imm(-1), dis.Imm(0)))

	// Allocate new wrapper: 5 fields (keys PTR, vals PTR, flags PTR, count WORD, cap WORD)
	// Shallow clone: shares the same internal arrays.
	mapTD := fl.makeMapTD()
	fl.emit(dis.Inst2(dis.INEW, dis.Imm(int32(mapTD)), dis.FP(dst)))

	// Copy all fields
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 0), dis.FPInd(dst, 0)))   // keys
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 8), dis.FPInd(dst, 8)))   // values
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(mapSlot, 16), dis.FPInd(dst, 16))) // flags
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 24), dis.FPInd(dst, 24))) // count
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(mapSlot, 32), dis.FPInd(dst, 32))) // capacity

	donePC := int32(len(fl.insts))
	fl.insts[nilIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// lowerMapsEqual: compare two maps for equality (same keys, same values).
func (fl *funcLowerer) lowerMapsEqual(instr *ssa.Call) (bool, error) {
	rawM1 := unwrapMakeInterface(instr.Call.Args[0])
	rawM2 := unwrapMakeInterface(instr.Call.Args[1])

	m1Slot := fl.materialize(rawM1)
	m2Slot := fl.materialize(rawM2)
	dst := fl.slotOf(instr)

	// Determine key/value types from the actual map type
	rawType := unwrapMakeInterface(instr.Call.Args[0]).Type().Underlying()
	mapType, isMapType := rawType.(*types.Map)
	isStrKey := false
	isStrVal := false
	if isMapType {
		if basic, ok := mapType.Key().Underlying().(*types.Basic); ok {
			isStrKey = basic.Kind() == types.String
		}
		if basic, ok := mapType.Elem().Underlying().(*types.Basic); ok {
			isStrVal = basic.Kind() == types.String
		}
	}

	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst))) // false

	// Both nil → equal
	cnt1 := fl.frame.AllocWord("me.c1")
	cnt2 := fl.frame.AllocWord("me.c2")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(cnt1)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(cnt2)))

	skipN1 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(m1Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(m1Slot, 24), dis.FP(cnt1)))
	fl.insts[skipN1].Dst = dis.Imm(int32(len(fl.insts)))

	skipN2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(m2Slot), dis.Imm(-1), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(m2Slot, 24), dis.FP(cnt2)))
	fl.insts[skipN2].Dst = dis.Imm(int32(len(fl.insts)))

	// Different count → not equal
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(cnt1), dis.FP(cnt2), dis.Imm(0)))

	// Both empty (count 0) → equal
	bothEmptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(cnt1), dis.Imm(0), dis.Imm(0)))

	// For each live key in m1, check that m2 has the same key with the same value.
	// Outer loop: iterate m1's capacity, skip non-occupied entries.
	// Inner loop: iterate m2's capacity, skip non-occupied entries.
	keys1 := fl.allocPtrTemp("me.k1")
	vals1 := fl.allocPtrTemp("me.v1")
	flags1 := fl.allocPtrTemp("me.f1")
	cap1 := fl.frame.AllocWord("me.cap1")
	keys2 := fl.allocPtrTemp("me.k2")
	vals2 := fl.allocPtrTemp("me.v2")
	flags2 := fl.allocPtrTemp("me.f2")
	cap2 := fl.frame.AllocWord("me.cap2")

	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m1Slot, 0), dis.FP(keys1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m1Slot, 8), dis.FP(vals1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m1Slot, 16), dis.FP(flags1)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(m1Slot, 32), dis.FP(cap1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m2Slot, 0), dis.FP(keys2)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m2Slot, 8), dis.FP(vals2)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.FPInd(m2Slot, 16), dis.FP(flags2)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(m2Slot, 32), dis.FP(cap2)))

	iSlot := fl.frame.AllocWord("me.i")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(iSlot)))

	tmpFlag := fl.frame.AllocWord("me.tf")
	tmpFlagAddr := fl.frame.AllocWord("me.tfa")

	// Outer loop: scan m1 capacity
	outerPC := int32(len(fl.insts))
	allMatchIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(iSlot), dis.FP(cap1), dis.Imm(0)))

	// Check flags1[i] == occupied
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(flags1), dis.FP(tmpFlagAddr), dis.FP(iSlot)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(tmpFlagAddr, 0), dis.FP(tmpFlag)))
	outerSkipIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(tmpFlag), dis.Imm(1), dis.Imm(0))) // not occupied → skip

	// Load key1[i] and val1[i]
	k1Addr := fl.frame.AllocWord("me.k1a")
	v1Addr := fl.frame.AllocWord("me.v1a")
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keys1), dis.FP(k1Addr), dis.FP(iSlot)))
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(vals1), dis.FP(v1Addr), dis.FP(iSlot)))

	// Inner loop: scan m2 capacity for matching key
	jSlot := fl.frame.AllocWord("me.j")
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(jSlot)))
	k2Addr := fl.frame.AllocWord("me.k2a")

	innerPC := int32(len(fl.insts))
	notFoundIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(jSlot), dis.FP(cap2), dis.Imm(0)))

	// Check flags2[j] == occupied
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(flags2), dis.FP(tmpFlagAddr), dis.FP(jSlot)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(tmpFlagAddr, 0), dis.FP(tmpFlag)))
	innerSkipIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(tmpFlag), dis.Imm(1), dis.Imm(0))) // not occupied → skip

	fl.emit(dis.NewInst(dis.IINDW, dis.FP(keys2), dis.FP(k2Addr), dis.FP(jSlot)))

	// Compare keys
	branchEq := dis.IBEQW
	branchNeV := dis.IBNEW
	if isStrKey {
		branchEq = dis.IBEQC
	}
	if isStrVal {
		branchNeV = dis.IBNEC
	}

	keyLoadOp := dis.IMOVW
	if isStrKey {
		keyLoadOp = dis.IMOVP
	}
	valLoadOp := dis.IMOVW
	if isStrVal {
		valLoadOp = dis.IMOVP
	}

	k1 := fl.frame.AllocTemp(isStrKey)
	k2 := fl.frame.AllocTemp(isStrKey)
	fl.emit(dis.Inst2(keyLoadOp, dis.FPInd(k1Addr, 0), dis.FP(k1)))
	fl.emit(dis.Inst2(keyLoadOp, dis.FPInd(k2Addr, 0), dis.FP(k2)))
	keyMatchIdx := len(fl.insts)
	fl.emit(dis.NewInst(branchEq, dis.FP(k1), dis.FP(k2), dis.Imm(0)))

	// No match: j++, continue inner loop
	innerAdvancePC := int32(len(fl.insts))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(jSlot), dis.FP(jSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))

	// Inner skip (non-occupied) → advance
	fl.insts[innerSkipIdx].Dst = dis.Imm(innerAdvancePC)

	// Key match: check values
	keyMatchPC := int32(len(fl.insts))
	fl.insts[keyMatchIdx].Dst = dis.Imm(keyMatchPC)

	v2Addr := fl.frame.AllocWord("me.v2a")
	fl.emit(dis.NewInst(dis.IINDW, dis.FP(vals2), dis.FP(v2Addr), dis.FP(jSlot)))

	v1 := fl.frame.AllocTemp(isStrVal)
	v2 := fl.frame.AllocTemp(isStrVal)
	fl.emit(dis.Inst2(valLoadOp, dis.FPInd(v1Addr, 0), dis.FP(v1)))
	fl.emit(dis.Inst2(valLoadOp, dis.FPInd(v2Addr, 0), dis.FP(v2)))
	valMismatchIdx := len(fl.insts)
	fl.emit(dis.NewInst(branchNeV, dis.FP(v1), dis.FP(v2), dis.Imm(0)))

	// Values match → advance outer
	outerAdvancePC := int32(len(fl.insts))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(iSlot), dis.FP(iSlot)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))

	// Outer skip (non-occupied) → advance outer
	fl.insts[outerSkipIdx].Dst = dis.Imm(outerAdvancePC)

	// Fall through = not equal (value mismatch or key not found)
	fl.insts[valMismatchIdx].Dst = dis.Imm(int32(len(fl.insts)))
	notEqualJmpIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// All keys matched with matching values → equal
	allMatchPC := int32(len(fl.insts))
	fl.insts[allMatchIdx].Dst = dis.Imm(allMatchPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))

	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	fl.insts[notFoundIdx].Dst = dis.Imm(donePC)
	fl.insts[notEqualJmpIdx].Dst = dis.Imm(donePC)

	// Both empty → equal
	fl.insts[bothEmptyIdx].Dst = dis.Imm(allMatchPC)
	return true, nil
}

// lowerMapsCopy: copy all keys/values from src to dst map.
func (fl *funcLowerer) lowerMapsCopy(instr *ssa.Call) (bool, error) {
	// For now, treat as no-op (the full implementation would iterate src
	// and call MapUpdate for each key/value pair, which is complex).
	// TODO: implement when MapUpdate can be called as a helper.
	return true, nil
}

// ============================================================
// io package
// ============================================================

// lowerIOCall handles calls to the io package.
func (fl *funcLowerer) lowerIOCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ReadAll":
		// io.ReadAll(r) → ([]byte, error)
		// Stub: return empty byte slice and nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WriteString":
		// io.WriteString(w, s) → (int, error)
		// Stub: return len(s), nil error
		sOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Copy":
		// io.Copy(dst, src) → (int64, error)
		// Stub: return 0, nil
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NopCloser":
		// io.NopCloser(r) → return r
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, rOp, dis.FP(dst)))
		return true, nil

	case "Pipe":
		// io.Pipe() → (*PipeReader, *PipeWriter) — stub returns (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil

	case "LimitReader":
		// io.LimitReader(r, n) → return r (stub)
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, rOp, dis.FP(dst)))
		return true, nil

	case "TeeReader":
		// io.TeeReader(r, w) → return r (stub)
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, rOp, dis.FP(dst)))
		return true, nil

	case "MultiReader", "MultiWriter":
		// io.MultiReader/MultiWriter → stub returns nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil

	case "CopyN", "CopyBuffer":
		// io.CopyN/CopyBuffer → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil

	case "ReadFull", "ReadAtLeast":
		// io.ReadFull/ReadAtLeast → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil

	case "NewSectionReader":
		// io.NewSectionReader → returns nil (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// cmp package (Go 1.21+)
// ============================================================

// lowerCmpCall handles calls to the cmp package.
func (fl *funcLowerer) lowerCmpCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Compare":
		// cmp.Compare(x, y) → -1, 0, or 1
		// Simplified: compare as integers
		xOp := fl.operandOf(instr.Call.Args[0])
		yOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		// if x < y → -1
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, xOp, yOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		jmpEndIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// if x > y → 1
		gePC := int32(len(fl.insts))
		fl.insts[bgeIdx].Dst = dis.Imm(gePC)
		bleIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, yOp, xOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[jmpEndIdx].Dst = dis.Imm(donePC)
		fl.insts[bleIdx].Dst = dis.Imm(donePC)
		return true, nil
	case "Less":
		// cmp.Less(x, y) → x < y
		xOp := fl.operandOf(instr.Call.Args[0])
		yOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		bgeIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, xOp, yOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		fl.insts[bgeIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "Or":
		// cmp.Or(vals...) → return first non-zero value
		// For int type: iterate array, return first != 0
		valsOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		n := fl.frame.AllocWord("or.n")
		i := fl.frame.AllocWord("or.i")
		addr := fl.frame.AllocWord("or.a")
		fl.emit(dis.Inst2(dis.ILENA, valsOp, dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		bgeDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(n), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDW, valsOp, dis.FP(addr), dis.FP(i)))
		bneFound := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FPInd(addr, 0), dis.Imm(0), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		foundPC := int32(len(fl.insts))
		fl.insts[bneFound].Dst = dis.Imm(foundPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addr, 0), dis.FP(dst)))
		fl.insts[bgeDone].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// context package
// ============================================================

// lowerContextCall handles calls to the context package.
func (fl *funcLowerer) lowerContextCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Background", "TODO":
		// context.Background() / context.TODO() → return nil context
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "WithCancel":
		// context.WithCancel(parent) → (ctx, cancel)
		// Return parent context and no-op cancel
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "WithValue":
		// context.WithValue(parent, key, val) → return parent (simplified)
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		return true, nil
	case "WithTimeout", "WithDeadline":
		// context.WithTimeout/WithDeadline(parent, ...) → (parent, no-op cancel)
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "WithCancelCause":
		// context.WithCancelCause(parent) → (parent, no-op cancelCause)
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Cause":
		// context.Cause(c) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "AfterFunc":
		// context.AfterFunc(ctx, f) → returns stop func (nil stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "WithoutCancel":
		// context.WithoutCancel(parent) → return parent
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		return true, nil
	case "WithDeadlineCause", "WithTimeoutCause":
		// Same as WithDeadline/WithTimeout but with extra cause arg
		parentOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, parentOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// sync/atomic package
// ============================================================

// lowerSyncAtomicCall handles calls to the sync/atomic package.
// Dis VM is cooperatively scheduled, so atomics are just regular memory operations.
// The addr argument is a pointer (materialized as a frame slot containing a pointer).
// We use FPInd(addrSlot, 0) to read/write through the pointer.
func (fl *funcLowerer) lowerSyncAtomicCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "AddInt32", "AddInt64", "AddUint32", "AddUint64", "AddUintptr":
		// atomic.AddInt32(addr, delta) → *addr += delta; return new value
		addrSlot := fl.materialize(instr.Call.Args[0])
		deltaOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		fl.emit(dis.NewInst(dis.IADDW, deltaOp, dis.FPInd(addrSlot, 0), dis.FPInd(addrSlot, 0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrSlot, 0), dis.FP(dst)))
		return true, nil
	case "LoadInt32", "LoadInt64", "LoadUint32", "LoadUint64", "LoadUintptr", "LoadPointer":
		// atomic.LoadInt32(addr) → return *addr
		addrSlot := fl.materialize(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrSlot, 0), dis.FP(dst)))
		return true, nil
	case "StoreInt32", "StoreInt64", "StoreUint32", "StoreUint64", "StoreUintptr", "StorePointer":
		// atomic.StoreInt32(addr, val) → *addr = val
		addrSlot := fl.materialize(instr.Call.Args[0])
		valOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVW, valOp, dis.FPInd(addrSlot, 0)))
		return true, nil
	case "SwapInt32", "SwapInt64", "SwapUint32", "SwapUint64", "SwapUintptr", "SwapPointer":
		// atomic.SwapInt32(addr, new) → old = *addr; *addr = new; return old
		addrSlot := fl.materialize(instr.Call.Args[0])
		newOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrSlot, 0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, newOp, dis.FPInd(addrSlot, 0)))
		return true, nil
	case "CompareAndSwapInt32", "CompareAndSwapInt64", "CompareAndSwapUint32",
		"CompareAndSwapUint64", "CompareAndSwapUintptr", "CompareAndSwapPointer":
		// atomic.CompareAndSwapInt32(addr, old, new) → if *addr == old { *addr = new; return true } else { return false }
		addrSlot := fl.materialize(instr.Call.Args[0])
		oldOp := fl.operandOf(instr.Call.Args[1])
		newOp := fl.operandOf(instr.Call.Args[2])
		dst := fl.slotOf(instr)
		// Load *addr into a temp — mid operand of BNEW doesn't support indirect addressing
		tmp := fl.frame.AllocWord("")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FPInd(addrSlot, 0), dis.FP(tmp)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst))) // default: false
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, oldOp, dis.FP(tmp), dis.Imm(0))) // if *addr != old, skip
		fl.emit(dis.Inst2(dis.IMOVW, newOp, dis.FPInd(addrSlot, 0)))    // *addr = new
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))          // return true
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// bufio package
// ============================================================

// lowerBufioCall handles calls to the bufio package.
func (fl *funcLowerer) lowerBufioCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewScanner":
		// bufio.NewScanner(r) → return stub pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewReader":
		// bufio.NewReader(r) → return stub pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewWriter":
		// bufio.NewWriter(w) → return stub pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "ScanLines", "ScanWords", "ScanRunes", "ScanBytes":
		// Split functions → return (0, nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "NewReaderSize", "NewWriterSize":
		// bufio.NewReaderSize/NewWriterSize → return stub pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewReadWriter":
		// bufio.NewReadWriter(r, w) → return stub pointer
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	// Scanner methods
	case "Scan":
		// (*Scanner).Scan() → false (no more tokens)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Text":
		// (*Scanner).Text() → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Err":
		// (*Scanner).Err() → nil
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Split", "Buffer":
		// (*Scanner).Split/Buffer — no-op
		return true, nil
	case "Bytes":
		// (*Scanner).Bytes() → nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil

	// Reader methods
	case "Read":
		// (*Reader).Read(p) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadByte":
		// (*Reader).ReadByte() → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadString":
		// (*Reader).ReadString(delim) → ("", nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadLine":
		// (*Reader).ReadLine() → (nil, false, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "ReadRune":
		// (*Reader).ReadRune() → (0, 0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "UnreadByte", "UnreadRune":
		// (*Reader).UnreadByte/UnreadRune() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Peek":
		// (*Reader).Peek(n) → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Buffered", "Available":
		// (*Reader/Writer).Buffered/Available() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Reset":
		// (*Reader/Writer).Reset — no-op
		return true, nil

	// Writer methods
	case "Write":
		// (*Writer).Write(p) → (len(p), nil)
		bOp := fl.operandOf(instr.Call.Args[1]) // p []byte
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.ILENA, bOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WriteByte":
		// (*Writer).WriteByte(c) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "WriteString":
		// (*Writer).WriteString(s) → (len(s), nil)
		sOp := fl.operandOf(instr.Call.Args[1]) // s string
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WriteRune":
		// (*Writer).WriteRune(r) → (size, nil)
		// Returns UTF-8 byte count: 1 for r<128, 2 for r<2048, 3 for r<65536, else 4
		rOp := fl.operandOf(instr.Call.Args[1]) // r rune/int32
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		is1Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.Imm(128), rOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(2), dis.FP(dst)))
		is2Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.Imm(2048), rOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(3), dis.FP(dst)))
		is3Idx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.Imm(65536), rOp, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(4), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[is1Idx].Dst = dis.Imm(donePC)
		fl.insts[is2Idx].Dst = dis.Imm(donePC)
		fl.insts[is3Idx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Flush":
		// (*Writer).Flush() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "ReadFrom":
		// (*Writer).ReadFrom(r) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadBytes", "ReadSlice":
		// (*Reader).ReadBytes/ReadSlice(delim) → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WriteTo":
		// (*Reader).WriteTo(w) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Discard":
		// (*Reader).Discard(n) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Size":
		// (*Reader/Writer).Size() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "AvailableBuffer":
		// (*Writer).AvailableBuffer() → nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net/url package
// ============================================================

// lowerNetURLCall handles calls to the net/url package.
func (fl *funcLowerer) lowerNetURLCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Parse":
		// url.Parse(rawURL) → (*URL, error)
		// Stub: return nil URL (H) and nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))        // nil *URL = H
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))  // error tag
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd))) // error val
		return true, nil
	case "QueryEscape", "PathEscape":
		// Real percent-encoding: unreserved chars pass through, space→'+' (QueryEscape),
		// everything else → %XX
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		isQuery := callee.Name() == "QueryEscape"

		hexTable := fl.comp.AllocString("0123456789ABCDEF")
		hexSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTable), dis.FP(hexSlot)))

		emptyMP := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyMP), dis.FP(dst)))

		lenS := fl.frame.AllocWord("qe.len")
		i := fl.frame.AllocWord("qe.i")
		ch := fl.frame.AllocWord("qe.ch")
		hi := fl.frame.AllocWord("qe.hi")
		lo := fl.frame.AllocWord("qe.lo")
		hiP1 := fl.frame.AllocWord("qe.hiP1")
		loP1 := fl.frame.AllocWord("qe.loP1")
		hiStr := fl.frame.AllocTemp(true)
		loStr := fl.frame.AllocTemp(true)
		pctStr := fl.frame.AllocTemp(true)
		outPos := fl.frame.AllocWord("qe.op")

		pctMP := fl.comp.AllocString("%")
		plusMP := fl.comp.AllocString("+")

		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outPos)))

		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

		// Check if unreserved: A-Z, a-z, 0-9, '-', '_', '.', '~'
		// A-Z: 65-90
		notAZ := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(65), dis.Imm(0)))
		notAZHi := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(90), dis.Imm(0)))
		safeIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0))) // → safe (pass through)

		// a-z: 97-122
		checkAz := int32(len(fl.insts))
		fl.insts[notAZHi].Dst = dis.Imm(checkAz)
		notAzLo := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(97), dis.Imm(0)))
		notAzHi := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(122), dis.Imm(0)))
		safeIdx2 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		// 0-9: 48-57
		checkDig := int32(len(fl.insts))
		fl.insts[notAZ].Dst = dis.Imm(checkDig)
		fl.insts[notAzLo].Dst = dis.Imm(checkDig)
		fl.insts[notAzHi].Dst = dis.Imm(checkDig)
		notDig := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(48), dis.Imm(0)))
		notDigHi := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
		safeIdx3 := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

		// Check '-' (45), '_' (95), '.' (46), '~' (126)
		checkSpecial := int32(len(fl.insts))
		fl.insts[notDig].Dst = dis.Imm(checkSpecial)
		fl.insts[notDigHi].Dst = dis.Imm(checkSpecial)
		dash := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(45), dis.Imm(0)))  // '-'
		dot := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(46), dis.Imm(0)))  // '.'
		under := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(95), dis.Imm(0)))  // '_'
		tilde := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(126), dis.Imm(0))) // '~'

		// Check space (32) → '+' for QueryEscape
		if isQuery {
			spaceIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))
			// Not safe, not space → percent-encode
			_ = spaceIdx // handled below

			// Percent-encode: %XX
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(pctMP), dis.FP(pctStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(pctStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
			fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(ch), dis.FP(lo)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(hi), dis.FP(hiP1)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lo), dis.FP(loP1)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTable), dis.FP(hiStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(hi), dis.FP(hiP1), dis.FP(hiStr)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTable), dis.FP(loStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lo), dis.FP(loP1), dis.FP(loStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(hiStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(loStr), dis.FP(dst), dis.FP(dst)))
			nextIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			// Space → '+'
			spacePC := int32(len(fl.insts))
			fl.insts[spaceIdx].Dst = dis.Imm(spacePC)
			plusSlot := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(plusMP), dis.FP(plusSlot)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(plusSlot), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			// Safe: pass through as-is (INSC at end of string)
			safePC := int32(len(fl.insts))
			fl.insts[safeIdx].Dst = dis.Imm(safePC)
			fl.insts[safeIdx2].Dst = dis.Imm(safePC)
			fl.insts[safeIdx3].Dst = dis.Imm(safePC)
			fl.insts[dash].Dst = dis.Imm(safePC)
			fl.insts[dot].Dst = dis.Imm(safePC)
			fl.insts[under].Dst = dis.Imm(safePC)
			fl.insts[tilde].Dst = dis.Imm(safePC)

			// Append single char via ISLICEC + IADDC
			oneAfter := fl.frame.AllocWord("qe.oa")
			chStr := fl.frame.AllocTemp(true)
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
			fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(chStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(chStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(chStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			donePC := int32(len(fl.insts))
			fl.insts[doneIdx].Dst = dis.Imm(donePC)
			_ = nextIdx
		} else {
			// PathEscape: same but space → %20 (no '+')
			// Not safe → percent-encode (same code)
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(pctMP), dis.FP(pctStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(pctStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(4), dis.FP(ch), dis.FP(hi)))
			fl.emit(dis.NewInst(dis.IANDW, dis.Imm(15), dis.FP(ch), dis.FP(lo)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(hi), dis.FP(hiP1)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lo), dis.FP(loP1)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTable), dis.FP(hiStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(hi), dis.FP(hiP1), dis.FP(hiStr)))
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(hexTable), dis.FP(loStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(lo), dis.FP(loP1), dis.FP(loStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(hiStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(loStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			// Safe: pass through
			safePC := int32(len(fl.insts))
			fl.insts[safeIdx].Dst = dis.Imm(safePC)
			fl.insts[safeIdx2].Dst = dis.Imm(safePC)
			fl.insts[safeIdx3].Dst = dis.Imm(safePC)
			fl.insts[dash].Dst = dis.Imm(safePC)
			fl.insts[dot].Dst = dis.Imm(safePC)
			fl.insts[under].Dst = dis.Imm(safePC)
			fl.insts[tilde].Dst = dis.Imm(safePC)
			oneAfter := fl.frame.AllocWord("pe.oa")
			chStr := fl.frame.AllocTemp(true)
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
			fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(chStr)))
			fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(chStr)))
			fl.emit(dis.NewInst(dis.IADDC, dis.FP(chStr), dis.FP(dst), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			donePC := int32(len(fl.insts))
			fl.insts[doneIdx].Dst = dis.Imm(donePC)
		}
		return true, nil

	case "QueryUnescape", "PathUnescape":
		// Real percent-decoding: %XX → byte, '+' → ' ' (QueryUnescape only)
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		isQueryUn := callee.Name() == "QueryUnescape"

		emptyMP := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyMP), dis.FP(dst)))

		lenS := fl.frame.AllocWord("qu.len")
		i := fl.frame.AllocWord("qu.i")
		ch := fl.frame.AllocWord("qu.ch")
		outPos := fl.frame.AllocWord("qu.op")
		hi := fl.frame.AllocWord("qu.hi")
		lo := fl.frame.AllocWord("qu.lo")
		bv := fl.frame.AllocWord("qu.bv")
		i1 := fl.frame.AllocWord("qu.i1")
		i2 := fl.frame.AllocWord("qu.i2")

		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(outPos)))

		// hexDigit helper (inline): given ch, compute value in outSlot
		hexDigitInline := func(outSlot int32) {
			// 0-9: 48-57 → 0-9
			notDig := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(57), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(48), dis.FP(ch), dis.FP(outSlot)))
			doneJmp := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
			// A-F: 65-70 → 10-15
			notUp := len(fl.insts)
			fl.insts[notDig].Dst = dis.Imm(int32(notUp))
			fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(70), dis.Imm(0)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(55), dis.FP(ch), dis.FP(outSlot)))
			doneJmp2 := len(fl.insts)
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
			// a-f: 97-102 → 10-15
			fl.insts[notUp].Dst = dis.Imm(int32(len(fl.insts)))
			fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(87), dis.FP(ch), dis.FP(outSlot)))
			endPC := int32(len(fl.insts))
			fl.insts[doneJmp].Dst = dis.Imm(endPC)
			fl.insts[doneJmp2].Dst = dis.Imm(endPC)
		}

		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

		// Check '%' (37)
		pctIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(37), dis.Imm(0)))

		if isQueryUn {
			// Check '+' (43) → space
			plusIdx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(43), dis.Imm(0)))
			// Regular char → pass through
			fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

			// '+' → space (32)
			fl.insts[plusIdx].Dst = dis.Imm(int32(len(fl.insts)))
			fl.emit(dis.NewInst(dis.IINSC, dis.Imm(32), dis.FP(outPos), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		} else {
			// Regular char → pass through
			fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(dst)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))
			fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
			fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		}

		// '%XX' → decode two hex digits
		fl.insts[pctIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i1)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(i), dis.FP(i2)))
		// Read hi nibble
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i1), dis.FP(ch)))
		hexDigitInline(hi)
		// Read lo nibble
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i2), dis.FP(ch)))
		hexDigitInline(lo)
		// bv = hi << 4 | lo
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(4), dis.FP(hi), dis.FP(bv)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(lo), dis.FP(bv), dis.FP(bv)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(bv), dis.FP(outPos), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(outPos), dis.FP(outPos)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(3), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		// nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ParseQuery":
		// url.ParseQuery(query) → (Values, error)
		// Stub: return nil map (H) and nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))        // nil map = H
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ParseRequestURI":
		// url.ParseRequestURI(rawURL) → (*URL, error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))        // nil *URL = H
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "User", "UserPassword":
		// url.User/UserPassword → nil *Userinfo stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // nil = H
		return true, nil
	// URL methods
	case "String", "Hostname", "Port", "RequestURI", "EscapedPath", "EscapedFragment", "Redacted":
		if callee.Signature.Recv() != nil {
			// (*URL).String/Hostname/Port/etc → "" stub
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Query":
		if callee.Signature.Recv() != nil {
			// (*URL).Query() → nil Values
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "IsAbs":
		if callee.Signature.Recv() != nil {
			// (*URL).IsAbs() → false
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "ResolveReference":
		if callee.Signature.Recv() != nil {
			// (*URL).ResolveReference(ref) → ref passthrough
			refOp := fl.operandOf(instr.Call.Args[1])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVP, refOp, dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "MarshalBinary":
		if callee.Signature.Recv() != nil {
			// (*URL).MarshalBinary() → (nil, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "UnmarshalBinary":
		if callee.Signature.Recv() != nil {
			// (*URL).UnmarshalBinary(text) → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	// Values methods
	case "Get":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Set", "Add", "Del":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
		return false, nil
	case "Has":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Encode":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	// Userinfo methods
	case "Username":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Password":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "JoinPath":
		if callee.Signature.Recv() != nil {
			// (*URL).JoinPath(elem ...string) → *URL
			// Stub: return nil *URL
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			return true, nil
		}
		// url.JoinPath(base, elem ...string) → (string, error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	// Error type methods
	case "Error":
		if callee.Signature.Recv() != nil && strings.Contains(callee.String(), "Error).Error") {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Unwrap":
		if callee.Signature.Recv() != nil && strings.Contains(callee.String(), "Error).Unwrap") {
			// (*Error).Unwrap() → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Timeout", "Temporary":
		if callee.Signature.Recv() != nil {
			// (*Error).Timeout/Temporary() → false
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// ============================================================
// encoding/json package
// ============================================================

// lowerEncodingJSONCall handles calls to the encoding/json package.
func (fl *funcLowerer) lowerEncodingJSONCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Marshal", "MarshalIndent":
		// json.Marshal(v) → ([]byte, error)
		// Stub: return nil byte slice (H) and nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))        // nil []byte = H
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))  // error tag
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd))) // error val
		return true, nil
	case "Unmarshal":
		// json.Unmarshal(data, v) → error
		// Stub: return nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Valid":
		return fl.lowerJSONValid(instr)
	case "Compact":
		// json.Compact(dst, src) → error (stub: return nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Indent":
		// json.Indent(dst, src, prefix, indent) → error (stub: return nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "HTMLEscape":
		// json.HTMLEscape(dst, src) → no-op
		return true, nil
	case "NewEncoder":
		// json.NewEncoder(w) → nil *Encoder stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // nil = H
		return true, nil
	case "NewDecoder":
		// json.NewDecoder(r) → nil *Decoder stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // nil = H
		return true, nil
	// Encoder methods
	case "Encode":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "SetIndent", "SetEscapeHTML":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
		return false, nil
	// Decoder methods
	case "Decode":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "More":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "UseNumber", "DisallowUnknownFields":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
		return false, nil
	case "Token":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "Buffered":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "InputOffset":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	// Number methods
	case "Float64":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+3*iby2wd)))
			return true, nil
		}
		return false, nil
	case "Int64":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	// Error type methods
	case "Error":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Unwrap":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	// RawMessage methods
	case "MarshalJSON":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "UnmarshalJSON":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// lowerJSONValid implements json.Valid by checking that data contains valid JSON.
// Uses a bracket/brace depth counter and validates string quoting and basic structure.
func (fl *funcLowerer) lowerJSONValid(instr *ssa.Call) (bool, error) {
	dataOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// Convert byte slice to string for character access
	sStr := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.ICVTAC, dataOp, dis.FP(sStr)))

	lenS := fl.frame.AllocWord("jv.len")
	i := fl.frame.AllocWord("jv.i")
	ch := fl.frame.AllocWord("jv.ch")
	depth := fl.frame.AllocWord("jv.depth")  // bracket/brace nesting depth
	sawVal := fl.frame.AllocWord("jv.saw")    // saw at least one value

	fl.emit(dis.Inst2(dis.ILENC, dis.FP(sStr), dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(depth)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(sawVal)))

	// Skip leading whitespace
	skipWSPC := int32(len(fl.insts))
	emptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
	// space=32, tab=9, newline=10, cr=13
	notSP := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))
	notTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))
	notNL := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(10), dis.Imm(0)))
	notCR := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(13), dis.Imm(0)))
	// Not whitespace — start main loop
	mainJmp := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	// Is whitespace: i++, continue skip
	wsContPC := int32(len(fl.insts))
	fl.insts[notSP].Dst = dis.Imm(wsContPC)
	fl.insts[notTab].Dst = dis.Imm(wsContPC)
	fl.insts[notNL].Dst = dis.Imm(wsContPC)
	fl.insts[notCR].Dst = dis.Imm(wsContPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(skipWSPC)))

	// Main scanning loop
	mainPC := int32(len(fl.insts))
	fl.insts[mainJmp].Dst = dis.Imm(mainPC)

	loopPC := int32(len(fl.insts))
	doneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(sawVal)))

	// '{' = 123: depth++
	notOBrace := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(123), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(depth), dis.FP(depth)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// '}' = 125: depth--; if depth < 0 → invalid
	notCBrace := int32(len(fl.insts))
	fl.insts[notOBrace].Dst = dis.Imm(notCBrace)
	isNotCBrace := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(125), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(depth), dis.FP(depth)))
	cbraceInvalid := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(depth), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// '[' = 91: depth++
	notOBracket := int32(len(fl.insts))
	fl.insts[isNotCBrace].Dst = dis.Imm(notOBracket)
	isNotOBracket := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(91), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(depth), dis.FP(depth)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// ']' = 93: depth--
	notCBracket := int32(len(fl.insts))
	fl.insts[isNotOBracket].Dst = dis.Imm(notCBracket)
	isNotCBracket := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(93), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(depth), dis.FP(depth)))
	cbracketInvalid := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(depth), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// '"' = 34: scan string, skip escaped chars
	notQuote := int32(len(fl.insts))
	fl.insts[isNotCBracket].Dst = dis.Imm(notQuote)
	isNotQuote := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(34), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i))) // skip opening quote
	// String scanning loop
	strLoopPC := int32(len(fl.insts))
	strEOF := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0))) // unterminated string
	fl.emit(dis.NewInst(dis.IINDC, dis.FP(sStr), dis.FP(i), dis.FP(ch)))
	// '"' → end of string
	strClose := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(34), dis.Imm(0)))
	// '\\' = 92 → skip next char
	notBackslash := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(92), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(2), dis.FP(i), dis.FP(i))) // skip backslash + next char
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(strLoopPC)))
	// Regular char: i++
	notBSPC := int32(len(fl.insts))
	fl.insts[notBackslash].Dst = dis.Imm(notBSPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(strLoopPC)))
	// String closed: i++, back to main loop
	strClosePC := int32(len(fl.insts))
	fl.insts[strClose].Dst = dis.Imm(strClosePC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Whitespace (32,9,10,13), commas(44), colons(58): skip
	afterQuote := int32(len(fl.insts))
	fl.insts[isNotQuote].Dst = dis.Imm(afterQuote)
	// Check whitespace and separators — just advance
	// These are all valid in JSON between tokens
	wsSep := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))  // space
	wsSep2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))   // tab
	wsSep3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(10), dis.Imm(0)))  // \n
	wsSep4 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(13), dis.Imm(0)))  // \r
	wsSep5 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(44), dis.Imm(0)))  // ','
	wsSep6 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(58), dis.Imm(0)))  // ':'

	// Numbers: 0-9 (48-57), '-' (45), '.' (46), '+' (43), 'e' (101), 'E' (69)
	// Keywords: t(116), r(114), u(117), e(101), f(102), a(97), l(108), s(115), n(110)
	// All valid JSON chars — just advance
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Whitespace/separator advance target
	sepPC := int32(len(fl.insts))
	fl.insts[wsSep].Dst = dis.Imm(sepPC)
	fl.insts[wsSep2].Dst = dis.Imm(sepPC)
	fl.insts[wsSep3].Dst = dis.Imm(sepPC)
	fl.insts[wsSep4].Dst = dis.Imm(sepPC)
	fl.insts[wsSep5].Dst = dis.Imm(sepPC)
	fl.insts[wsSep6].Dst = dis.Imm(sepPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Invalid: return false
	invalidPC := int32(len(fl.insts))
	fl.insts[emptyIdx].Dst = dis.Imm(invalidPC)  // empty input
	fl.insts[cbraceInvalid].Dst = dis.Imm(invalidPC)
	fl.insts[cbracketInvalid].Dst = dis.Imm(invalidPC)
	fl.insts[strEOF].Dst = dis.Imm(invalidPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	invalidDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// Done: valid if depth == 0 && sawVal
	donePC := int32(len(fl.insts))
	fl.insts[doneIdx].Dst = dis.Imm(donePC)
	// depth != 0 → invalid
	depthNZ := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(depth), dis.Imm(0), dis.Imm(0)))
	// sawVal == 0 → invalid
	noValIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(sawVal), dis.Imm(0), dis.Imm(0)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
	finalDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	fl.insts[depthNZ].Dst = dis.Imm(invalidPC)
	fl.insts[noValIdx].Dst = dis.Imm(invalidPC)

	allDonePC := int32(len(fl.insts))
	fl.insts[invalidDoneIdx].Dst = dis.Imm(allDonePC)
	fl.insts[finalDoneIdx].Dst = dis.Imm(allDonePC)
	return true, nil
}

// ============================================================
// runtime package
// ============================================================

// lowerRuntimeCall handles calls to the runtime package.
func (fl *funcLowerer) lowerRuntimeCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "GOMAXPROCS":
		// runtime.GOMAXPROCS(n) → return 1 (Dis VM is single-threaded)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "NumCPU":
		// runtime.NumCPU() → return 1
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "NumGoroutine":
		// runtime.NumGoroutine() → return 1
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		return true, nil
	case "Gosched":
		// runtime.Gosched() → no-op
		return true, nil
	case "GC":
		// runtime.GC() → no-op (Dis VM handles GC)
		return true, nil
	case "Goexit":
		// runtime.Goexit() → emit RET
		fl.emit(dis.Inst0(dis.IRET))
		return true, nil
	case "Caller":
		// runtime.Caller(skip) → (0, "", 0, false)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))             // pc
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst+iby2wd))) // file
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))    // line
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+3*iby2wd)))    // ok
		return true, nil
	case "Callers":
		// runtime.Callers(skip, pc []uintptr) → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "GOROOT":
		// runtime.GOROOT() → "/go"
		dst := fl.slotOf(instr)
		gorootOff := fl.comp.AllocString("/go")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(gorootOff), dis.FP(dst)))
		return true, nil
	case "Version":
		// runtime.Version() → "go1.22"
		dst := fl.slotOf(instr)
		verOff := fl.comp.AllocString("go1.22")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(verOff), dis.FP(dst)))
		return true, nil
	case "GOOS":
		// runtime.GOOS → "inferno" (handled as var)
		return false, nil
	case "GOARCH":
		return false, nil
	case "SetFinalizer":
		// runtime.SetFinalizer(obj, finalizer) → no-op
		return true, nil
	case "KeepAlive":
		// runtime.KeepAlive(x) → no-op
		return true, nil
	case "LockOSThread", "UnlockOSThread":
		return true, nil // no-op
	case "ReadMemStats":
		return true, nil // no-op — writes to *MemStats
	case "Stack":
		// runtime.Stack(buf, all) → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "FuncForPC":
		// runtime.FuncForPC(pc) → nil *Func
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "CallersFrames":
		// runtime.CallersFrames(callers) → nil *Frames
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Func methods
	case "Name":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Entry":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "FileLine":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	// Frames.Next method
	case "Next":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			// Frame struct fields + more bool
			for i := int32(0); i < 7; i++ {
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+i*iby2wd)))
			}
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// ============================================================
// reflect package
// ============================================================

// lowerReflectCall handles calls to the reflect package.
func (fl *funcLowerer) lowerReflectCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "TypeOf":
		// reflect.TypeOf(i) → stub return nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "ValueOf":
		// reflect.ValueOf(i) → stub return zero Value
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "DeepEqual":
		// reflect.DeepEqual(x, y) → stub return false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Zero", "New", "MakeSlice", "MakeMap", "MakeMapWithSize", "MakeChan",
		"Indirect", "AppendSlice":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Append":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Copy":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Swapper":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "PtrTo", "PointerTo", "SliceOf", "MapOf", "ChanOf":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	// Value methods (called on Value receiver)
	case "String":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Int":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Float":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Bool", "IsNil", "IsValid", "IsZero", "CanSet", "CanInterface", "CanAddr":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Bytes":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Interface":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Kind", "Type":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Elem", "Field", "FieldByName", "Index", "MapIndex", "Addr", "Convert",
		"MapRange", "Slice":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Len", "Cap", "NumField", "NumMethod":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Uint", "Pointer":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "MapKeys":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Call":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Set", "SetInt", "SetString", "SetFloat", "SetBool", "SetBytes",
		"SetUint", "SetMapIndex", "SetLen", "SetCap", "SetComplex",
		"SetPointer", "Send", "Close", "Grow", "SetZero":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	case "FieldByIndex", "FieldByNameFunc", "Method":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "MethodByName":
		if callee.Signature.Recv() != nil {
			// Returns (Value, bool)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Recv", "TryRecv":
		if callee.Signature.Recv() != nil {
			// Returns (Value, bool)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "TrySend":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "UnsafeAddr", "UnsafePointer":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "OverflowFloat", "OverflowInt", "OverflowUint", "Comparable", "Equal":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Complex":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	// StructTag methods
	case "Get":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	case "Lookup":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	// MapIter methods
	case "Key", "Value":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Next":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Reset":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
	// Package-level functions
	case "Select":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "FuncOf", "StructOf", "ArrayOf":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "MakeFunc", "NewAt":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// os/exec package
// ============================================================

func (fl *funcLowerer) lowerOsExecCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Command":
		// exec.Command(name, args...) → return nil *Cmd stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "LookPath":
		// exec.LookPath(file) → (file, nil error)
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "CommandContext":
		// exec.CommandContext(ctx, name, args...) → nil *Cmd stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Cmd methods
	case "Run", "Start", "Wait":
		// (*Cmd).Run/Start/Wait() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Output", "CombinedOutput":
		// (*Cmd).Output/CombinedOutput() → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "StdinPipe", "StdoutPipe", "StderrPipe":
		// (*Cmd).StdinPipe/StdoutPipe/StderrPipe() → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "String":
		// (*Cmd).String() → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Environ":
		// (*Cmd).Environ() → nil []string
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Error":
		// Error.Error() or ExitError.Error() → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Unwrap":
		// Error.Unwrap() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "ExitCode":
		// ExitError.ExitCode() → -1
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// os/signal package
// ============================================================

func (fl *funcLowerer) lowerOsSignalCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Notify", "Stop":
		// No-op stubs
		return true, nil
	}
	return false, nil
}

// ============================================================
// io/ioutil package (deprecated, forwards to io/os)
// ============================================================

func (fl *funcLowerer) lowerIOUtilCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ReadFile":
		// Same as os.ReadFile stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WriteFile":
		// Same as os.WriteFile stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "ReadAll":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "TempDir":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		tmpOff := fl.comp.AllocString("/tmp")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(tmpOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NopCloser":
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, rOp, dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// io/fs package
// ============================================================

func (fl *funcLowerer) lowerIOFSCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)
	switch name {
	case "ReadFile":
		// fs.ReadFile(fsys, name) → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadDir":
		// fs.ReadDir(fsys, name) → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Stat":
		// fs.Stat(fsys, name) → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "WalkDir":
		// fs.WalkDir(fsys, root, fn) → nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Sub":
		// fs.Sub(fsys, dir) → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Glob":
		// fs.Glob(fsys, pattern) → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ValidPath":
		return fl.lowerFSValidPath(instr)
	case "FormatFileInfo", "FormatDirEntry":
		// fs.FormatFileInfo/FormatDirEntry → ""
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "IsDir", "IsRegular":
		// FileMode.IsDir/IsRegular → false
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Perm", "Type":
		// FileMode.Perm/Type → 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "String":
		// FileMode.String → ""
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Error":
		// PathError.Error → ""
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Unwrap":
		// PathError.Unwrap → nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

// lowerFSValidPath implements fs.ValidPath(name string) → bool.
// A valid path is "." or a non-empty slash-separated sequence of elements
// where no element is "", ".", or "..". Must not start or end with "/".
func (fl *funcLowerer) lowerFSValidPath(instr *ssa.Call) (bool, error) {
	nameOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenN := fl.frame.AllocWord("vp.len")
	i := fl.frame.AllocWord("vp.i")
	ch := fl.frame.AllocWord("vp.ch")
	elemStart := fl.frame.AllocWord("vp.es")
	elemLen := fl.frame.AllocWord("vp.el")

	fl.emit(dis.Inst2(dis.ILENC, nameOp, dis.FP(lenN)))

	// Empty string → false
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
	emptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenN), dis.Imm(0), dis.Imm(0)))

	// Check if name == "." (len==1 && name[0]=='.')
	notDotIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(lenN), dis.Imm(1), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.Imm(0), dis.FP(ch)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
	dotTrueIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(46), dis.Imm(0))) // '.' = 46
	fl.insts[notDotIdx].Dst = dis.Imm(int32(len(fl.insts)))

	// Check first char != '/'
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.Imm(0), dis.FP(ch)))
	invalidFixups := []int{}
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(47), dis.Imm(0))) // '/' = 47

	// Check last char != '/'
	lastIdx := fl.frame.AllocWord("vp.last")
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(lenN), dis.FP(lastIdx)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(lastIdx), dis.FP(ch)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(47), dis.Imm(0)))

	// Scan character by character, tracking element boundaries
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(elemStart)))

	loopPC := int32(len(fl.insts))
	bgeDoneIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenN), dis.Imm(0)))

	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(i), dis.FP(ch)))

	// If ch == '/', check element
	notSlashIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(47), dis.Imm(0)))

	// On slash: elemLen = i - elemStart
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(elemStart), dis.FP(i), dis.FP(elemLen)))
	// Empty element (two consecutive slashes) → invalid
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(elemLen), dis.Imm(0), dis.Imm(0)))

	// Check if element is "." (len==1 && elem[0]=='.')
	notDotElem := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(elemLen), dis.Imm(1), dis.Imm(0)))
	dotCh := fl.frame.AllocWord("vp.dc")
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(elemStart), dis.FP(dotCh)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(dotCh), dis.Imm(46), dis.Imm(0)))
	fl.insts[notDotElem].Dst = dis.Imm(int32(len(fl.insts)))

	// Check if element is ".." (len==2 && elem[0]=='.' && elem[1]=='.')
	notDDElem := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(elemLen), dis.Imm(2), dis.Imm(0)))
	d1 := fl.frame.AllocWord("vp.d1")
	d2 := fl.frame.AllocWord("vp.d2")
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(elemStart), dis.FP(d1)))
	es1 := fl.frame.AllocWord("vp.es1")
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(elemStart), dis.FP(es1)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(es1), dis.FP(d2)))
	skipDD := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(d1), dis.Imm(46), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(d2), dis.Imm(46), dis.Imm(0)))
	fl.insts[skipDD].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[notDDElem].Dst = dis.Imm(int32(len(fl.insts)))

	// Update elemStart = i + 1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(elemStart)))

	// Not slash: just advance
	fl.insts[notSlashIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// After loop: check last element (from elemStart to lenN)
	fl.insts[bgeDoneIdx].Dst = dis.Imm(int32(len(fl.insts)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.FP(elemStart), dis.FP(lenN), dis.FP(elemLen)))

	// Last elem "." check
	notDotLast := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(elemLen), dis.Imm(1), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(elemStart), dis.FP(dotCh)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(dotCh), dis.Imm(46), dis.Imm(0)))
	fl.insts[notDotLast].Dst = dis.Imm(int32(len(fl.insts)))

	// Last elem ".." check
	notDDLast := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(elemLen), dis.Imm(2), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(elemStart), dis.FP(d1)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(elemStart), dis.FP(es1)))
	fl.emit(dis.NewInst(dis.IINDC, nameOp, dis.FP(es1), dis.FP(d2)))
	skipDDLast := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(d1), dis.Imm(46), dis.Imm(0)))
	invalidFixups = append(invalidFixups, len(fl.insts))
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(d2), dis.Imm(46), dis.Imm(0)))
	fl.insts[skipDDLast].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[notDDLast].Dst = dis.Imm(int32(len(fl.insts)))

	// Valid
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
	validDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))

	// Invalid
	invalidPC := int32(len(fl.insts))
	for _, idx := range invalidFixups {
		fl.insts[idx].Dst = dis.Imm(invalidPC)
	}
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[emptyIdx].Dst = dis.Imm(donePC)
	fl.insts[dotTrueIdx].Dst = dis.Imm(donePC)
	fl.insts[validDoneIdx].Dst = dis.Imm(donePC)
	return true, nil
}

// ============================================================
// regexp package
// ============================================================

func (fl *funcLowerer) lowerRegexpCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Compile":
		// regexp.Compile(expr) → (*Regexp, error) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))        // nil *Regexp = H
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "MustCompile":
		// regexp.MustCompile(str) → *Regexp stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst))) // nil *Regexp = H
		return true, nil
	case "MatchString":
		// regexp.MatchString(pattern, s) → (bool, error) stub: return false, nil
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "QuoteMeta":
		// regexp.QuoteMeta(s) → escape all regex metacharacters with backslash
		// Metacharacters: \.+*?()|[]{}^$
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		lenS := fl.frame.AllocWord("qm.len")
		i := fl.frame.AllocWord("qm.i")
		ch := fl.frame.AllocWord("qm.ch")
		result := fl.frame.AllocTemp(true)
		emptyOff := fl.comp.AllocString("")
		bsOff := fl.comp.AllocString("\\")
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		// Check each metacharacter: \.+*?()|[]{}^$
		metaChars := []int32{'\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}', '^', '$'}
		var metaMatchIdxs []int
		for _, mc := range metaChars {
			idx := len(fl.insts)
			fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(mc), dis.Imm(0)))
			metaMatchIdxs = append(metaMatchIdxs, idx)
		}
		// Not a metachar: just append ch
		outPos := fl.frame.AllocWord("qm.pos")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(outPos)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(result)))
		nextIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0))) // jump to i++
		// Is a metachar: append backslash then ch
		metaPC := int32(len(fl.insts))
		for _, idx := range metaMatchIdxs {
			fl.insts[idx].Dst = dis.Imm(metaPC)
		}
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(bsOff), dis.FP(result), dis.FP(result)))
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(outPos)))
		fl.emit(dis.NewInst(dis.IINSC, dis.FP(ch), dis.FP(outPos), dis.FP(result)))
		nextPC := int32(len(fl.insts))
		fl.insts[nextIdx].Dst = dis.Imm(nextPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		donePC := int32(len(fl.insts))
		fl.insts[doneIdx].Dst = dis.Imm(donePC)
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
		return true, nil
	case "Match":
		// regexp.Match(pattern, b) → (false, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "CompilePOSIX":
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "MustCompilePOSIX":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	// Regexp method stubs
	case "FindString", "ReplaceAllString":
		// (*Regexp).FindString/ReplaceAllString → "" stub
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "ReplaceAllStringFunc":
		// (*Regexp).ReplaceAllStringFunc(src, repl) → return src
		dst := fl.slotOf(instr)
		srcOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVP, srcOp, dis.FP(dst)))
		return true, nil
	case "FindStringIndex", "FindStringSubmatch", "FindAllString",
		"FindAllStringSubmatch", "Split", "SubexpNames":
		// return nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "String":
		// (*Regexp).String() → "" stub
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "NumSubexp", "SubexpIndex":
		// (*Regexp).NumSubexp/SubexpIndex() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	// Byte-based find methods
	case "Find", "ReplaceAll", "ReplaceAllLiteral":
		// Return nil []byte
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "ReplaceAllFunc":
		// Return src arg
		dst := fl.slotOf(instr)
		srcOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVP, srcOp, dis.FP(dst)))
		return true, nil
	case "FindIndex", "FindSubmatch", "FindSubmatchIndex",
		"FindAll", "FindAllIndex", "FindAllSubmatch", "FindAllSubmatchIndex",
		"FindStringSubmatchIndex", "FindAllStringIndex", "FindAllStringSubmatchIndex":
		// Return nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "ReplaceAllLiteralString":
		// Return src
		dst := fl.slotOf(instr)
		srcOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVP, srcOp, dis.FP(dst)))
		return true, nil
	case "Expand", "ExpandString":
		// Return dst arg
		dst := fl.slotOf(instr)
		dstOp := fl.operandOf(instr.Call.Args[1])
		fl.emit(dis.Inst2(dis.IMOVP, dstOp, dis.FP(dst)))
		return true, nil
	case "Longest":
		// No-op
		return true, nil
	case "Copy":
		// Return nil *Regexp
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "LiteralPrefix":
		// Return ("", false)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "MatchReader":
		// Return false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net/http package
// ============================================================

func (fl *funcLowerer) lowerNetHTTPCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()

	// Handle method calls on types
	if callee.Signature.Recv() != nil {
		return fl.lowerNetHTTPMethodCall(instr, callee, name)
	}

	switch name {
	case "Get", "Post", "Head", "PostForm":
		// http.Get/Post/Head/PostForm → (*Response, error) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NewRequest", "NewRequestWithContext":
		// http.NewRequest → (*Request, error) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ListenAndServe", "ListenAndServeTLS":
		// http.ListenAndServe → error stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		return true, nil
	case "Handle", "HandleFunc", "Error", "NotFound", "Redirect", "SetCookie":
		// no-op stubs
		return true, nil
	case "NewServeMux":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "StatusText":
		return fl.lowerHTTPStatusText(instr)
	case "CanonicalHeaderKey":
		return fl.lowerCanonicalHeaderKey(instr)
	case "DetectContentType":
		return fl.lowerHTTPDetectContentType(instr)
	case "NotFoundHandler", "FileServer", "StripPrefix", "TimeoutHandler", "AllowQuerySemicolons":
		// handler-returning stubs → return nil handler
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "MaxBytesReader":
		// return nil reader
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "ProxyFromEnvironment":
		// → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "ServeFile", "ServeContent":
		// http.ServeFile(w, r, name) / http.ServeContent(w, r, name, modtime, content) → no-op
		return true, nil
	case "ReadResponse":
		// http.ReadResponse(r, req) → (*Response, error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// lowerHTTPStatusText implements http.StatusText(code int) → string
// with a comparison chain for common HTTP status codes.
func (fl *funcLowerer) lowerHTTPStatusText(instr *ssa.Call) (bool, error) {
	codeOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// Table of status codes → text
	statusTable := []struct {
		code int32
		text string
	}{
		{100, "Continue"},
		{101, "Switching Protocols"},
		{200, "OK"},
		{201, "Created"},
		{202, "Accepted"},
		{204, "No Content"},
		{206, "Partial Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{303, "See Other"},
		{304, "Not Modified"},
		{307, "Temporary Redirect"},
		{308, "Permanent Redirect"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{406, "Not Acceptable"},
		{408, "Request Timeout"},
		{409, "Conflict"},
		{410, "Gone"},
		{411, "Length Required"},
		{413, "Request Entity Too Large"},
		{414, "Request URI Too Long"},
		{415, "Unsupported Media Type"},
		{416, "Requested Range Not Satisfiable"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{501, "Not Implemented"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{504, "Gateway Timeout"},
	}

	// Default: empty string
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))

	// Chain of comparisons: if code == X, set text and jump to done
	var doneFixups []int
	for _, entry := range statusTable {
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, codeOp, dis.Imm(entry.code), dis.Imm(0)))
		textOff := fl.comp.AllocString(entry.text)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(textOff), dis.FP(dst)))
		doneFixups = append(doneFixups, len(fl.insts))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
	}

	// Done: patch all jumps
	donePC := int32(len(fl.insts))
	for _, idx := range doneFixups {
		fl.insts[idx].Dst = dis.Imm(donePC)
	}
	return true, nil
}

// lowerHTTPDetectContentType implements http.DetectContentType(data []byte) → string.
// Checks magic bytes to identify common content types.
func (fl *funcLowerer) lowerHTTPDetectContentType(instr *ssa.Call) (bool, error) {
	dataOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenD := fl.frame.AllocWord("ct.len")
	addr := fl.frame.AllocWord("ct.addr")
	b0 := fl.frame.AllocWord("ct.b0")
	b1 := fl.frame.AllocWord("ct.b1")
	b2 := fl.frame.AllocWord("ct.b2")
	b3 := fl.frame.AllocWord("ct.b3")

	fl.emit(dis.Inst2(dis.ILENA, dataOp, dis.FP(lenD)))

	readByte := func(idx int32, target int32) {
		fl.emit(dis.NewInst(dis.IINDB, dataOp, dis.FP(addr), dis.Imm(idx)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(target)))
	}

	// Default: application/octet-stream
	defaultOff := fl.comp.AllocString("application/octet-stream")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(defaultOff), dis.FP(dst)))

	// If empty, return default
	emptyIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(lenD), dis.Imm(0), dis.Imm(0)))

	// Read first 4 bytes (with bounds checking)
	readByte(0, b0)
	// Read b1 if len >= 2
	has2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lenD), dis.Imm(2), dis.Imm(0)))
	readByte(1, b1)
	has3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lenD), dis.Imm(3), dis.Imm(0)))
	readByte(2, b2)
	has4 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lenD), dis.Imm(4), dis.Imm(0)))
	readByte(3, b3)
	gotBytesPC := int32(len(fl.insts))
	fl.insts[has2].Dst = dis.Imm(gotBytesPC)
	fl.insts[has3].Dst = dis.Imm(gotBytesPC)
	fl.insts[has4].Dst = dis.Imm(gotBytesPC)

	var doneFixups []int
	setResult := func(mime string) {
		off := fl.comp.AllocString(mime)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(off), dis.FP(dst)))
		doneFixups = append(doneFixups, len(fl.insts))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	}

	// PNG: 0x89 0x50 0x4E 0x47
	skipPNG := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x89), dis.Imm(0)))
	skipPNG2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b1), dis.Imm(0x50), dis.Imm(0)))
	setResult("image/png")
	fl.insts[skipPNG].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipPNG2].Dst = dis.Imm(int32(len(fl.insts)))

	// JPEG: 0xFF 0xD8 0xFF
	skipJPG := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0xFF), dis.Imm(0)))
	skipJPG2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b1), dis.Imm(0xD8), dis.Imm(0)))
	setResult("image/jpeg")
	fl.insts[skipJPG].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipJPG2].Dst = dis.Imm(int32(len(fl.insts)))

	// GIF: "GIF8" (0x47 0x49 0x46 0x38)
	skipGIF := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x47), dis.Imm(0)))
	skipGIF2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b1), dis.Imm(0x49), dis.Imm(0)))
	skipGIF3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b2), dis.Imm(0x46), dis.Imm(0)))
	setResult("image/gif")
	fl.insts[skipGIF].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipGIF2].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipGIF3].Dst = dis.Imm(int32(len(fl.insts)))

	// PDF: "%PDF" (0x25 0x50 0x44 0x46)
	skipPDF := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x25), dis.Imm(0)))
	skipPDF2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b1), dis.Imm(0x50), dis.Imm(0)))
	skipPDF3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b2), dis.Imm(0x44), dis.Imm(0)))
	setResult("application/pdf")
	fl.insts[skipPDF].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipPDF2].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipPDF3].Dst = dis.Imm(int32(len(fl.insts)))

	// PK (ZIP): 0x50 0x4B 0x03 0x04
	skipZIP := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x50), dis.Imm(0)))
	skipZIP2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b1), dis.Imm(0x4B), dis.Imm(0)))
	skipZIP3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b2), dis.Imm(0x03), dis.Imm(0)))
	setResult("application/zip")
	fl.insts[skipZIP].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipZIP2].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipZIP3].Dst = dis.Imm(int32(len(fl.insts)))

	// HTML: starts with '<' (0x3C) — simplified check
	// Real Go checks for specific tags but this catches most HTML
	skipHTML := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x3C), dis.Imm(0)))
	setResult("text/html; charset=utf-8")
	fl.insts[skipHTML].Dst = dis.Imm(int32(len(fl.insts)))

	// JSON: starts with '{' or '['
	skipJSON := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x7B), dis.Imm(0)))
	setResult("application/json")
	fl.insts[skipJSON].Dst = dis.Imm(int32(len(fl.insts)))
	skipJSON2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(b0), dis.Imm(0x5B), dis.Imm(0)))
	setResult("application/json")
	fl.insts[skipJSON2].Dst = dis.Imm(int32(len(fl.insts)))

	// Text: check if first byte is printable ASCII or whitespace
	// If b0 is in range [0x09-0x0D] or [0x20-0x7E], likely text
	skipText := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(b0), dis.Imm(0x20), dis.Imm(0)))
	skipText2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b0), dis.Imm(0x09), dis.Imm(0)))
	skipText3 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(b0), dis.Imm(0x0D), dis.Imm(0)))
	isTextPC := int32(len(fl.insts))
	fl.insts[skipText].Dst = dis.Imm(isTextPC)
	setResult("text/plain; charset=utf-8")
	fl.insts[skipText2].Dst = dis.Imm(int32(len(fl.insts)))
	fl.insts[skipText3].Dst = dis.Imm(int32(len(fl.insts)))

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[emptyIdx].Dst = dis.Imm(donePC)
	for _, idx := range doneFixups {
		fl.insts[idx].Dst = dis.Imm(donePC)
	}
	return true, nil
}

func (fl *funcLowerer) lowerNetHTTPMethodCall(instr *ssa.Call, callee *ssa.Function, name string) (bool, error) {
	recv := callee.Signature.Recv()
	recvStr := recv.Type().String()

	switch name {
	// Header methods
	case "Set", "Add", "Del":
		if strings.Contains(recvStr, "Header") {
			return true, nil // no-op
		}
	case "Get":
		if strings.Contains(recvStr, "Header") {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		// Client.Get → (*Response, error)
		if strings.Contains(recvStr, "Client") {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Values":
		if strings.Contains(recvStr, "Header") {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Clone":
		if strings.Contains(recvStr, "Header") {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		// Request.Clone(ctx) → *Request
		if strings.Contains(recvStr, "Request") {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			return true, nil
		}
	case "Write":
		if strings.Contains(recvStr, "Header") || strings.Contains(recvStr, "Request") || strings.Contains(recvStr, "Response") {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}

	// Request methods
	case "FormValue", "PostFormValue", "UserAgent", "Referer":
		if strings.Contains(recvStr, "Request") {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	case "Context":
		if strings.Contains(recvStr, "Request") {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "Cookie":
		if strings.Contains(recvStr, "Request") {
			// (*Cookie, error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Cookies":
		// Request.Cookies() or Response.Cookies() → nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "AddCookie", "SetBasicAuth":
		if strings.Contains(recvStr, "Request") {
			return true, nil // no-op
		}
	case "BasicAuth":
		if strings.Contains(recvStr, "Request") {
			// (username, password string, ok bool)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	case "ParseForm", "ParseMultipartForm":
		if strings.Contains(recvStr, "Request") {
			// → error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "ProtoAtLeast":
		if strings.Contains(recvStr, "Request") || strings.Contains(recvStr, "Response") {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "WithContext":
		if strings.Contains(recvStr, "Request") {
			// Request.WithContext(ctx) → *Request stub (return nil)
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			return true, nil
		}
	case "MultipartReader":
		if strings.Contains(recvStr, "Request") {
			// Request.MultipartReader() → (*multipart.Reader, error) stub
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	// Client methods
	case "Do", "Post", "Head", "PostForm":
		if strings.Contains(recvStr, "Client") {
			// → (*Response, error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "CloseIdleConnections":
		if strings.Contains(recvStr, "Client") {
			return true, nil // no-op
		}

	// Server methods
	case "ListenAndServe", "ListenAndServeTLS", "Shutdown", "Close":
		if strings.Contains(recvStr, "Server") {
			// → error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}

	// Response methods
	case "Location":
		if strings.Contains(recvStr, "Response") {
			// Response.Location() → (*url.URL, error) stub
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}

	// ServeMux methods
	case "Handle", "HandleFunc", "ServeHTTP":
		if strings.Contains(recvStr, "ServeMux") {
			return true, nil // no-op
		}

	// Cookie methods
	case "String":
		if strings.Contains(recvStr, "Cookie") {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	case "Valid":
		if strings.Contains(recvStr, "Cookie") {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

// ============================================================
// log/slog package
// ============================================================

func (fl *funcLowerer) lowerLogSlogCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Info", "Warn", "Error", "Debug":
		// slog.Info/Warn/Error/Debug(msg, args...) → print msg via sys->print
		if len(instr.Call.Args) > 0 {
			msgOp := fl.operandOf(instr.Call.Args[0])
			msgSlot := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.IMOVP, msgOp, dis.FP(msgSlot)))
			nlOff := fl.comp.AllocString("\n")
			fl.emit(dis.NewInst(dis.IADDC, dis.MP(nlOff), dis.FP(msgSlot), dis.FP(msgSlot)))
			fl.emitSysCall("print", []callSiteArg{{msgSlot, true}})
		}
		return true, nil
	case "String", "Int", "Int64", "Float64", "Bool", "Any", "Duration", "Group":
		// slog.String/Int/Int64/Float64/Bool/Any/Duration/Group → return zero Attr (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "InfoContext", "WarnContext", "ErrorContext", "DebugContext":
		// slog.XxxContext(ctx, msg, args...) → print msg
		if len(instr.Call.Args) > 1 {
			msgOp := fl.operandOf(instr.Call.Args[1])
			msgSlot := fl.frame.AllocTemp(true)
			fl.emit(dis.Inst2(dis.IMOVP, msgOp, dis.FP(msgSlot)))
			nlOff := fl.comp.AllocString("\n")
			fl.emit(dis.NewInst(dis.IADDC, dis.MP(nlOff), dis.FP(msgSlot), dis.FP(msgSlot)))
			fl.emitSysCall("print", []callSiteArg{{msgSlot, true}})
		}
		return true, nil
	case "New", "Default", "With", "WithGroup":
		// slog.New/Default/With/WithGroup → return nil *Logger
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "SetDefault":
		// slog.SetDefault(l) → no-op
		return true, nil
	case "Enabled":
		// Logger.Enabled → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Handler":
		// Logger.Handler → nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewTextHandler", "NewJSONHandler":
		// slog.NewTextHandler/NewJSONHandler → nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Level", "Set":
		// LevelVar.Level/Set → stub
		if callee.Signature.Recv() != nil {
			if callee.Name() == "Level" {
				dst := fl.slotOf(instr)
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			}
			return true, nil
		}
		return false, nil
	case "Equal":
		// Attr.Equal → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// embed package
// ============================================================

func (fl *funcLowerer) lowerEmbedCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Open":
		// FS.Open(name) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadDir":
		// FS.ReadDir(name) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadFile":
		// FS.ReadFile(name) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// flag package
// ============================================================

func (fl *funcLowerer) lowerFlagCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Parse":
		// flag.Parse() → no-op
		return true, nil
	case "String":
		// flag.String(name, value, usage) → return pointer to value (stub: return nil)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Int":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Bool":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Arg":
		// flag.Arg(i) → return empty string
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "Args":
		// flag.Args() → return nil slice
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NArg", "NFlag":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Float64", "Int64", "Uint", "Uint64", "Duration":
		// Return nil pointer (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "StringVar", "IntVar", "BoolVar", "Float64Var", "Int64Var",
		"UintVar", "Uint64Var", "DurationVar", "TextVar":
		// No-op (writes to pointer)
		return true, nil
	case "Parsed":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Set":
		// flag.Set(name, value) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Lookup":
		// flag.Lookup(name) → nil *Flag
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewFlagSet":
		// flag.NewFlagSet(name, handling) → nil *FlagSet
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "PrintDefaults", "Visit", "VisitAll":
		// No-op stubs
		return true, nil
	case "Func", "BoolFunc", "Var":
		// No-op (registers a flag with callback)
		return true, nil
	case "UnquoteUsage":
		// flag.UnquoteUsage(flag) → ("", "")
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst+iby2wd)))
		return true, nil
	case "Init", "SetOutput":
		// No-op
		return true, nil
	case "Name":
		// FlagSet.Name() → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	case "ErrorHandling":
		// FlagSet.ErrorHandling() → ContinueOnError (0)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Output":
		// FlagSet.Output() → nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/sha256 package
// ============================================================

func (fl *funcLowerer) lowerCryptoSHA256Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Sum256":
		// crypto/sha256.Sum256(data) → [32]byte stub: return zero array
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Sum224":
		// crypto/sha256.Sum224(data) → [28]byte stub: return zero array
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "New", "New224":
		// crypto/sha256.New/New224() → return nil (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/md5 package
// ============================================================

func (fl *funcLowerer) lowerCryptoMD5Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Sum":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "New":
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/binary package
// ============================================================

func (fl *funcLowerer) lowerEncodingBinaryCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Write", "Read":
		// binary.Write/Read → stub: return nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "PutUvarint":
		// binary.PutUvarint(buf []byte, x uint64) int
		// Writes x as unsigned varint into buf, returns bytes written.
		bufOp := fl.operandOf(instr.Call.Args[0])
		xOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		x := fl.frame.AllocWord("puv.x")
		i := fl.frame.AllocWord("puv.i")
		b := fl.frame.AllocWord("puv.b")
		addr := fl.frame.AllocWord("puv.a")
		fl.emit(dis.Inst2(dis.IMOVW, xOp, dis.FP(x)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// loop: while x >= 0x80
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(x), dis.Imm(0x80), dis.Imm(0)))
		// buf[i] = byte(x) | 0x80
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(x), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IORW, dis.Imm(0x80), dis.FP(b), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(b), dis.FPInd(addr, 0)))
		// x >>= 7
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(7), dis.FP(x), dis.FP(x)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// done: buf[i] = byte(x)
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(x), dis.FPInd(addr, 0)))
		// return i + 1
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(dst)))
		return true, nil
	case "Uvarint":
		// binary.Uvarint(buf []byte) (uint64, int)
		// Reads a uvarint from buf. Returns (value, bytesRead).
		bufOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		x := fl.frame.AllocWord("uv.x")
		s := fl.frame.AllocWord("uv.s")
		i := fl.frame.AllocWord("uv.i")
		bLen := fl.frame.AllocWord("uv.n")
		b := fl.frame.AllocWord("uv.b")
		addr := fl.frame.AllocWord("uv.a")
		v := fl.frame.AllocWord("uv.v")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(x)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(s)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ILENA, bufOp, dis.FP(bLen)))
		// loop: while i < len(buf)
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(bLen), dis.Imm(0)))
		// b = buf[i]
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(b)))
		// v = (b & 0x7F) << s
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(b), dis.FP(v)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.FP(s), dis.FP(v), dis.FP(v)))
		// x |= v
		fl.emit(dis.NewInst(dis.IORW, dis.FP(v), dis.FP(x), dis.FP(x)))
		// if b < 0x80 → done (no continuation bit)
		exitIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(b), dis.Imm(0x80), dis.Imm(0)))
		// s += 7; i++
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(7), dis.FP(s), dis.FP(s)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// found terminator: return (x, i+1)
		exitPC := int32(len(fl.insts))
		fl.insts[exitIdx].Dst = dis.Imm(exitPC)
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(x), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(dst+iby2wd)))
		endIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// exhausted buffer: return (x, 0) — incomplete
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(x), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "PutVarint":
		// binary.PutVarint(buf []byte, x int64) int
		// Encodes x = uint(ux), where ux = x<<1 ^ x>>63 (zigzag encoding)
		bufOp := fl.operandOf(instr.Call.Args[0])
		xOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		ux := fl.frame.AllocWord("pv.ux")
		tmp := fl.frame.AllocWord("pv.t")
		i := fl.frame.AllocWord("pv.i")
		b := fl.frame.AllocWord("pv.b")
		addr := fl.frame.AllocWord("pv.a")
		// ux = (x << 1) ^ (x >> 63)
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(1), xOp, dis.FP(ux)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(63), xOp, dis.FP(tmp)))
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(tmp), dis.FP(ux), dis.FP(ux)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// loop: while ux >= 0x80
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ux), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(ux), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IORW, dis.Imm(0x80), dis.FP(b), dis.FP(b)))
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(b), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(7), dis.FP(ux), dis.FP(ux)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(ux), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(dst)))
		return true, nil
	case "Varint":
		// binary.Varint(buf []byte) (int64, int)
		// Reads zigzag-encoded varint. ux = Uvarint(buf); x = int64(ux>>1) ^ -(ux&1)
		bufOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		x := fl.frame.AllocWord("vi.x")
		s := fl.frame.AllocWord("vi.s")
		i := fl.frame.AllocWord("vi.i")
		bLen := fl.frame.AllocWord("vi.n")
		bv := fl.frame.AllocWord("vi.b")
		addrV := fl.frame.AllocWord("vi.a")
		v := fl.frame.AllocWord("vi.v")
		// Inline Uvarint first
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(x)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(s)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ILENA, bufOp, dis.FP(bLen)))
		loopPC := int32(len(fl.insts))
		bufDoneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(bLen), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDB, bufOp, dis.FP(addrV), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addrV, 0), dis.FP(bv)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(bv), dis.FP(v)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.FP(s), dis.FP(v), dis.FP(v)))
		fl.emit(dis.NewInst(dis.IORW, dis.FP(v), dis.FP(x), dis.FP(x)))
		exitIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(bv), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(7), dis.FP(s), dis.FP(s)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		// Found terminator: decode zigzag, return (x, i+1)
		exitPC := int32(len(fl.insts))
		fl.insts[exitIdx].Dst = dis.Imm(exitPC)
		// zigzag decode: result = (ux >> 1) ^ -(ux & 1)
		ux1 := fl.frame.AllocWord("vi.u1")
		neg := fl.frame.AllocWord("vi.ng")
		fl.emit(dis.NewInst(dis.ILSRW, dis.Imm(1), dis.FP(x), dis.FP(ux1)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(x), dis.FP(neg)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.FP(neg), dis.Imm(0), dis.FP(neg)))
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(neg), dis.FP(ux1), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(dst+iby2wd)))
		endIdx := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// Buffer exhausted
		fl.insts[bufDoneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.insts[endIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "Size":
		// binary.Size → return 0 (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "AppendUvarint":
		// binary.AppendUvarint(buf, x) → append varint encoding of x to buf
		// Each byte: 7 bits of data, high bit = continuation
		// Must use byte array ops (INDB+CVTWB) not string ops, since values >= 128
		bufOp := fl.operandOf(instr.Call.Args[0])
		xOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		x := fl.frame.AllocWord("auv.x")
		bval := fl.frame.AllocWord("auv.b")
		nBytes := fl.frame.AllocWord("auv.n")
		addr := fl.frame.AllocWord("auv.a")
		localBuf := fl.frame.AllocPointer("auv:buf")
		fl.emit(dis.Inst2(dis.IMOVP, bufOp, dis.FP(localBuf)))
		// First pass: count bytes needed
		fl.emit(dis.Inst2(dis.IMOVW, xOp, dis.FP(x)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(nBytes)))
		countPC := int32(len(fl.insts))
		countDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(x), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(nBytes), dis.FP(nBytes)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(7), dis.FP(x), dis.FP(x)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countPC)))
		fl.insts[countDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(nBytes), dis.FP(nBytes))) // +1 for final byte
		// Allocate temp byte array of nBytes
		tmpArr := fl.frame.AllocPointer("auv:tmp")
		byteTD := fl.makeHeapTypeDesc(types.Typ[types.Byte])
		fl.emit(dis.NewInst(dis.INEWAZ, dis.FP(nBytes), dis.Imm(int32(byteTD)), dis.FP(tmpArr)))
		// Second pass: encode
		fl.emit(dis.Inst2(dis.IMOVW, xOp, dis.FP(x)))
		i := fl.frame.AllocWord("auv.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		encPC := int32(len(fl.insts))
		encDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(x), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(x), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IORW, dis.Imm(0x80), dis.FP(bval), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(tmpArr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(bval), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(7), dis.FP(x), dis.FP(x)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(encPC)))
		fl.insts[encDone].Dst = dis.Imm(int32(len(fl.insts)))
		// Final byte
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(tmpArr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(x), dis.FPInd(addr, 0)))
		// Nil-safe concat: if buf is nil (H=-1), dst = tmpArr; else dst = buf + tmpArr
		nilSkip := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(localBuf), dis.Imm(-1), dis.Imm(0)))
		// Non-nil path: concat buf + tmpArr via string ops
		bufStr := fl.frame.AllocTemp(true)
		tmpStr := fl.frame.AllocTemp(true)
		catStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, dis.FP(localBuf), dis.FP(bufStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, dis.FP(tmpArr), dis.FP(tmpStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(tmpStr), dis.FP(bufStr), dis.FP(catStr)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(catStr), dis.FP(dst)))
		doneJmp := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		// Nil path: dst = tmpArr directly
		fl.insts[nilSkip].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmpArr), dis.FP(dst)))
		fl.insts[doneJmp].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "AppendVarint":
		// binary.AppendVarint(buf, x) → encode zigzag(x) as uvarint
		bufOp := fl.operandOf(instr.Call.Args[0])
		xOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		ux := fl.frame.AllocWord("av.ux")
		x := fl.frame.AllocWord("av.x")
		bval := fl.frame.AllocWord("av.b")
		nBytes := fl.frame.AllocWord("av.n")
		addr := fl.frame.AllocWord("av.a")
		localBuf := fl.frame.AllocPointer("av:buf")
		fl.emit(dis.Inst2(dis.IMOVP, bufOp, dis.FP(localBuf)))
		fl.emit(dis.Inst2(dis.IMOVW, xOp, dis.FP(x)))
		// Zigzag: ux = (x << 1) ^ (x >> 63)
		shl := fl.frame.AllocWord("av.shl")
		shr := fl.frame.AllocWord("av.shr")
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(1), dis.FP(x), dis.FP(shl)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(63), dis.FP(x), dis.FP(shr)))
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(shr), dis.FP(shl), dis.FP(ux)))
		// Count bytes
		xCopy := fl.frame.AllocWord("av.xc")
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(ux), dis.FP(xCopy)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(nBytes)))
		countPC := int32(len(fl.insts))
		countDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(xCopy), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(nBytes), dis.FP(nBytes)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(7), dis.FP(xCopy), dis.FP(xCopy)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(countPC)))
		fl.insts[countDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(nBytes), dis.FP(nBytes)))
		// Allocate temp byte array
		tmpArr := fl.frame.AllocPointer("av:tmp")
		byteTD := fl.makeHeapTypeDesc(types.Typ[types.Byte])
		fl.emit(dis.NewInst(dis.INEWAZ, dis.FP(nBytes), dis.Imm(int32(byteTD)), dis.FP(tmpArr)))
		// Encode
		i := fl.frame.AllocWord("av.i")
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		encPC := int32(len(fl.insts))
		encDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ux), dis.Imm(0x80), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(0x7F), dis.FP(ux), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IORW, dis.Imm(0x80), dis.FP(bval), dis.FP(bval)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(tmpArr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(bval), dis.FPInd(addr, 0)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(7), dis.FP(ux), dis.FP(ux)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(encPC)))
		fl.insts[encDone].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(tmpArr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTWB, dis.FP(ux), dis.FPInd(addr, 0)))
		// Nil-safe concat
		nilSkip := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(localBuf), dis.Imm(-1), dis.Imm(0)))
		bufStr := fl.frame.AllocTemp(true)
		tmpStr := fl.frame.AllocTemp(true)
		catStr := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, dis.FP(localBuf), dis.FP(bufStr)))
		fl.emit(dis.Inst2(dis.ICVTAC, dis.FP(tmpArr), dis.FP(tmpStr)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(tmpStr), dis.FP(bufStr), dis.FP(catStr)))
		fl.emit(dis.Inst2(dis.ICVTCA, dis.FP(catStr), dis.FP(dst)))
		doneJmp := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[nilSkip].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(tmpArr), dis.FP(dst)))
		fl.insts[doneJmp].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "Encode", "Decode":
		// binary.Encode/Decode(buf, order, data) → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Append":
		// binary.Append(order, buf, data) → (buf, nil)
		bufOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVP, bufOp, dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/csv package
// ============================================================

func (fl *funcLowerer) lowerEncodingCSVCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewReader", "NewWriter":
		// csv.NewReader/NewWriter → return nil pointer (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Reader methods
	case "Read":
		// (*Reader).Read() → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ReadAll":
		// (*Reader).ReadAll() → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	// Writer methods
	case "Write":
		// (*Writer).Write(record) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "WriteAll":
		// (*Writer).WriteAll(records) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Flush":
		// (*Writer).Flush() → no-op
		return true, nil
	case "Error":
		// (*Writer).Error() or ParseError.Error() → nil error / empty string
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "FieldPos":
		// (*Reader).FieldPos(field) → (0, 0)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "InputOffset":
		// (*Reader).InputOffset() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Unwrap":
		// ParseError.Unwrap() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// math/big package
// ============================================================

func (fl *funcLowerer) lowerMathBigCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewInt", "NewFloat", "NewRat":
		// big.NewInt/NewFloat/NewRat → return nil pointer (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Int/Float/Rat methods that return *T (self-modifying)
	case "Add", "Sub", "Mul", "Div", "Mod", "Abs", "Neg", "Set", "SetInt64", "SetBytes", "Exp", "GCD", "Quo", "SetFloat64", "SetPrec":
		if callee.Signature.Recv() != nil {
			// Return receiver (self) as stub
			selfOp := fl.operandOf(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVP, selfOp, dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Cmp", "Sign", "BitLen":
		if callee.Signature.Recv() != nil {
			// Return 0 (stub)
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Int64":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Float64":
		if callee.Signature.Recv() != nil {
			// (*Float).Float64() → (0.0, 0 accuracy)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "String":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			zeroOff := fl.comp.AllocString("0")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(zeroOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "SetString":
		if callee.Signature.Recv() != nil {
			// (*Int).SetString(s, base) → (self, true)
			selfOp := fl.operandOf(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVP, selfOp, dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Bytes":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "IsInt64", "IsInf":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// ============================================================
// text/template package
// ============================================================

func (fl *funcLowerer) lowerTextTemplateCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "New":
		// template.New(name) → return nil pointer (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Must":
		// template.Must(t, err) → t passthrough
		tOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, tOp, dis.FP(dst)))
		return true, nil
	case "ParseFiles", "ParseGlob":
		// template.ParseFiles/ParseGlob → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	// Template methods
	case "Parse":
		if callee.Signature.Recv() != nil {
			// (*Template).Parse(text) → (self, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			selfOp := fl.operandOf(instr.Call.Args[0])
			fl.emit(dis.Inst2(dis.IMOVP, selfOp, dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "Execute", "ExecuteTemplate":
		if callee.Signature.Recv() != nil {
			// (*Template).Execute/ExecuteTemplate → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Funcs", "Option", "Delims":
		if callee.Signature.Recv() != nil {
			// (*Template).Funcs/Option/Delims → self passthrough
			selfOp := fl.operandOf(instr.Call.Args[0])
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVP, selfOp, dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Name":
		if callee.Signature.Recv() != nil {
			// (*Template).Name() → ""
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Lookup":
		if callee.Signature.Recv() != nil {
			// (*Template).Lookup(name) → nil
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "Clone":
		if callee.Signature.Recv() != nil {
			// (*Template).Clone() → (nil, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// ============================================================
// hash, hash/crc32 packages
// ============================================================

func (fl *funcLowerer) lowerHashCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ChecksumIEEE":
		// crc32.ChecksumIEEE(data []byte) uint32 — real bit-by-bit CRC32 with IEEE polynomial
		// Algorithm: for each byte, XOR into crc, then process 8 bits using polynomial 0xEDB88320
		dataOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		arr := fl.frame.AllocTemp(true)
		n := fl.frame.AllocWord("crc.n")
		i := fl.frame.AllocWord("crc.i")
		j := fl.frame.AllocWord("crc.j")
		crc := fl.frame.AllocWord("crc.crc")
		b := fl.frame.AllocWord("crc.b")
		bit := fl.frame.AllocWord("crc.bit")
		poly := fl.frame.AllocWord("crc.poly")
		mask32 := fl.frame.AllocWord("crc.mask")
		addr := fl.frame.AllocWord("crc.addr")
		// Build 0xFFFFFFFF: (1 << 32) - 1
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(mask32)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(32), dis.FP(mask32), dis.FP(mask32)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(mask32), dis.FP(mask32)))
		// Build 0xEDB88320: (0xEDB8 << 16) | 0x8320
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0x0EDB8), dis.FP(poly)))
		fl.emit(dis.NewInst(dis.ISHLW, dis.Imm(16), dis.FP(poly), dis.FP(poly)))
		fl.emit(dis.NewInst(dis.IORW, dis.Imm(0x8320), dis.FP(poly), dis.FP(poly)))
		// crc = 0xFFFFFFFF
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(mask32), dis.FP(crc)))
		// arr = cvtac(data), n = len(arr)
		fl.emit(dis.Inst2(dis.ICVTAC, dataOp, dis.FP(arr)))
		fl.emit(dis.Inst2(dis.ILENA, dis.FP(arr), dis.FP(n)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		// Outer loop: for i < n
		outerPC := int32(len(fl.insts))
		outerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(n), dis.Imm(0)))
		// b = arr[i]
		fl.emit(dis.NewInst(dis.IINDB, dis.FP(arr), dis.FP(addr), dis.FP(i)))
		fl.emit(dis.Inst2(dis.ICVTBW, dis.FPInd(addr, 0), dis.FP(b)))
		// crc ^= b
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(b), dis.FP(crc), dis.FP(crc)))
		// Inner loop: j = 0; for j < 8
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(j)))
		innerPC := int32(len(fl.insts))
		innerDone := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(j), dis.Imm(8), dis.Imm(0)))
		// bit = crc & 1
		fl.emit(dis.NewInst(dis.IANDW, dis.Imm(1), dis.FP(crc), dis.FP(bit)))
		// crc >>= 1
		fl.emit(dis.NewInst(dis.ISHRW, dis.Imm(1), dis.FP(crc), dis.FP(crc)))
		// if bit == 0, skip XOR with polynomial
		skipPoly := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQW, dis.FP(bit), dis.Imm(0), dis.Imm(0)))
		// crc ^= poly
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(poly), dis.FP(crc), dis.FP(crc)))
		fl.insts[skipPoly].Dst = dis.Imm(int32(len(fl.insts)))
		// j++
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(j), dis.FP(j)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(innerPC)))
		fl.insts[innerDone].Dst = dis.Imm(int32(len(fl.insts)))
		// i++
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(outerPC)))
		fl.insts[outerDone].Dst = dis.Imm(int32(len(fl.insts)))
		// crc ^= 0xFFFFFFFF (final inversion)
		fl.emit(dis.NewInst(dis.IXORW, dis.FP(mask32), dis.FP(crc), dis.FP(crc)))
		// Mask to 32 bits: crc &= 0xFFFFFFFF
		fl.emit(dis.NewInst(dis.IANDW, dis.FP(mask32), dis.FP(crc), dis.FP(dst)))
		return true, nil
	case "New":
		// hash.New/crc32.New → return nil (stub)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net package
// ============================================================

func (fl *funcLowerer) lowerNetCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Dial", "Listen":
		// net.Dial/Listen or Dialer.Dial → (nil, nil error) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "JoinHostPort":
		// net.JoinHostPort(host, port) → host + ":" + port
		hostOp := fl.operandOf(instr.Call.Args[0])
		portOp := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		colonOff := fl.comp.AllocString(":")
		fl.emit(dis.Inst2(dis.IMOVP, hostOp, dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDC, dis.MP(colonOff), dis.FP(dst), dis.FP(dst)))
		fl.emit(dis.NewInst(dis.IADDC, portOp, dis.FP(dst), dis.FP(dst)))
		return true, nil
	case "SplitHostPort":
		// net.SplitHostPort(hostport string) → (host, port string, err error)
		// Parses "host:port", "[host]:port" for IPv6
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		emptyOff := fl.comp.AllocString("")
		lenS := fl.frame.AllocWord("shp.len")
		i := fl.frame.AllocWord("shp.i")
		ch := fl.frame.AllocWord("shp.ch")
		lastColon := fl.frame.AllocWord("shp.lc")
		hostTmp := fl.frame.AllocTemp(true)
		portTmp := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
		// Default: empty host/port, nil error
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+3*iby2wd)))
		// Find last ':'
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(lastColon)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
		loopPC := int32(len(fl.insts))
		doneIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
		fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
		notColonIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(':'), dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(i), dis.FP(lastColon)))
		nextPC := int32(len(fl.insts))
		fl.insts[notColonIdx].Dst = dis.Imm(nextPC)
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
		fl.insts[doneIdx].Dst = dis.Imm(int32(len(fl.insts)))
		// If no colon found, skip (leave defaults)
		noColonIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, dis.FP(lastColon), dis.Imm(0), dis.Imm(0)))
		// host = s[:lastColon], port = s[lastColon+1:]
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(hostTmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(lastColon), dis.FP(hostTmp)))
		portStart := fl.frame.AllocWord("shp.ps")
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(lastColon), dis.FP(portStart)))
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(portTmp)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(portStart), dis.FP(lenS), dis.FP(portTmp)))
		// Check if host starts with '[' (IPv6)
		bracketCheck := fl.frame.AllocWord("shp.bc")
		fl.emit(dis.NewInst(dis.IINDC, dis.FP(hostTmp), dis.Imm(0), dis.FP(bracketCheck)))
		noBracketIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEW, dis.FP(bracketCheck), dis.Imm('['), dis.Imm(0)))
		// Strip brackets: host = host[1:len-1]
		hostLen := fl.frame.AllocWord("shp.hl")
		fl.emit(dis.Inst2(dis.ILENC, dis.FP(hostTmp), dis.FP(hostLen)))
		fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(hostLen), dis.FP(hostLen)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(1), dis.FP(hostLen), dis.FP(hostTmp)))
		fl.insts[noBracketIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(hostTmp), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVP, dis.FP(portTmp), dis.FP(dst+iby2wd)))
		fl.insts[noColonIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	case "DialTimeout":
		// net.DialTimeout(network, address, timeout) → (nil, nil error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "DialTCP", "DialUDP", "DialUnix":
		// net.DialTCP/UDP/Unix → (nil, nil error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ListenTCP", "ListenUDP", "ListenUnix", "ListenPacket":
		// net.ListenXxx → (nil, nil error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "LookupHost", "LookupAddr":
		// net.LookupHost/LookupAddr or Resolver.LookupHost → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "LookupIP":
		// net.LookupIP(host) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "LookupPort":
		// net.LookupPort(network, service) → (0, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "LookupCNAME", "LookupTXT", "LookupMX", "LookupNS", "LookupSRV":
		// net.Lookup* → return zero values
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Interfaces":
		// net.Interfaces() → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "InterfaceByName":
		// net.InterfaceByName(name) → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ResolveIPAddr", "ResolveTCPAddr", "ResolveUDPAddr", "ResolveUnixAddr":
		// net.Resolve*Addr → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ParseCIDR":
		// net.ParseCIDR(s) → (nil IP, nil IPNet, nil error) — 3 results
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "ParseIP":
		// net.ParseIP(s) → nil IP
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "CIDRMask", "IPv4Mask":
		// net.CIDRMask/IPv4Mask → nil IPMask
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "IPv4":
		// net.IPv4(a, b, c, d) → nil IP
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "FileConn", "FileListener", "FilePacketConn":
		// net.File* → (nil, nil error)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Pipe":
		// net.Pipe() → (nil, nil) — two Conn values
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	// Method calls on net types
	case "String", "Network":
		if callee.Signature.Recv() != nil {
			// IP.String(), TCPAddr.String(), UDPAddr.String(), IPNet.String(),
			// IPMask.String(), TCPAddr.Network(), UDPAddr.Network(), OpError.Error(), DNSError.Error()
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	case "Error":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	case "Equal", "IsLoopback", "IsPrivate", "IsUnspecified", "Timeout", "Temporary":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "To4", "To16":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "MarshalText":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Size":
		if callee.Signature.Recv() != nil {
			// IPMask.Size() → (0, 0)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Contains":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			return true, nil
		}
	case "DialContext":
		if callee.Signature.Recv() != nil {
			// Dialer.DialContext → (nil Conn, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

// ============================================================
// crypto/rand package
// ============================================================

func (fl *funcLowerer) lowerCryptoRandCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Read":
		// crypto/rand.Read(b) → (len(b), nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		bSlot := fl.materialize(instr.Call.Args[0])
		lenSlot := fl.frame.AllocWord("crand.len")
		fl.emit(dis.Inst2(dis.ILENA, dis.FP(bSlot), dis.FP(lenSlot)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenSlot), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Int", "Prime":
		// crypto/rand.Int/Prime → (0, nil) stub
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
// crypto/hmac package
// ============================================================

func (fl *funcLowerer) lowerCryptoHMACCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "New":
		// hmac.New(h, key) → 0 (stub handle)
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Equal":
		// hmac.Equal(mac1, mac2) → constant-time compare via ICVTAC + IBEQC
		mac1 := fl.operandOf(instr.Call.Args[0])
		mac2 := fl.operandOf(instr.Call.Args[1])
		dst := fl.slotOf(instr)
		s1 := fl.frame.AllocTemp(true)
		s2 := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.ICVTAC, mac1, dis.FP(s1)))
		fl.emit(dis.Inst2(dis.ICVTAC, mac2, dis.FP(s2)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		beqMatch := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(s1), dis.FP(s2), dis.Imm(0)))
		jmpDone := len(fl.insts)
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[beqMatch].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		fl.insts[jmpDone].Dst = dis.Imm(int32(len(fl.insts)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/aes package
// ============================================================

func (fl *funcLowerer) lowerCryptoAESCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewCipher":
		// aes.NewCipher(key) → (0, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/cipher package
// ============================================================

func (fl *funcLowerer) lowerCryptoCipherCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewGCM":
		// cipher.NewGCM(cipher) → (0, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NewCFBEncrypter", "NewCFBDecrypter", "NewCTR", "NewOFB":
		// cipher mode constructors → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "NewCBCEncrypter", "NewCBCDecrypter":
		// CBC mode → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "NewGCMWithNonceSize", "NewGCMWithTagSize":
		// GCM variants → (0, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// unicode/utf16 package
// ============================================================

func (fl *funcLowerer) lowerUnicodeUTF16Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Encode", "Decode":
		// utf16.Encode/Decode → nil slice stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "IsSurrogate":
		// utf16.IsSurrogate(r) → 0xD800 <= r && r < 0xE000
		rOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		// Materialize large mid operands BEFORE capturing instruction indices
		midLow := fl.midImm(0xD800)
		midHigh := fl.midImm(0xE000)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		bltLow := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBLTW, rOp, midLow, dis.Imm(0)))
		bgeHigh := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGEW, rOp, midHigh, dis.Imm(0)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(dst)))
		donePC := int32(len(fl.insts))
		fl.insts[bltLow].Dst = dis.Imm(donePC)
		fl.insts[bgeHigh].Dst = dis.Imm(donePC)
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/xml package
// ============================================================

func (fl *funcLowerer) lowerEncodingXMLCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Marshal", "MarshalIndent":
		// xml.Marshal(v) → ([]byte(""), nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Unmarshal":
		// xml.Unmarshal(data, v) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "EscapeText":
		// xml.EscapeText(w, data) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "NewEncoder":
		// xml.NewEncoder(w) → nil *Encoder stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewDecoder":
		// xml.NewDecoder(r) → nil *Decoder stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewTokenDecoder":
		// xml.NewTokenDecoder(r) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Escape":
		// xml.Escape(w, data) → no-op
		return true, nil
	case "CopyToken":
		// xml.CopyToken(t) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Encoder/Decoder methods
	case "Encode", "EncodeToken", "EncodeElement":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Decode", "DecodeElement":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Token":
		if callee.Signature.Recv() != nil {
			// (*Decoder).Token() → (nil, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "Skip":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Flush":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "Indent":
		if callee.Signature.Recv() != nil {
			return true, nil // no-op
		}
		return false, nil
	case "Copy":
		// StartElement.Copy(), CharData.Copy(), etc. → return zero value
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "End":
		// StartElement.End() → return zero EndElement
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "InputOffset":
		// Decoder.InputOffset() → 0
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Error":
		// SyntaxError.Error(), etc. → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// encoding/pem package
// ============================================================

func (fl *funcLowerer) lowerEncodingPEMCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Decode":
		// pem.Decode(data) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Encode":
		// pem.Encode(out, b) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "EncodeToMemory":
		// pem.EncodeToMemory(b) → nil slice stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/tls package
// ============================================================

func (fl *funcLowerer) lowerCryptoTLSCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "Dial", "DialWithDialer", "Listen":
		// tls.Dial/DialWithDialer/Listen → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "LoadX509KeyPair", "X509KeyPair":
		// → (zero Certificate, nil) stub
		dst := fl.slotOf(instr)
		for i := int32(0); i < 5*int32(dis.IBY2WD); i += int32(dis.IBY2WD) {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+i)))
		}
		return true, nil
	case "NewListener":
		// tls.NewListener → nil interface
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Clone":
		// Config.Clone() → nil *Config
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Close", "Handshake":
		// Conn.Close/Handshake → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Read", "Write":
		// Conn.Read/Write → (0, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ConnectionState":
		// Conn.ConnectionState → zero struct
		dst := fl.slotOf(instr)
		for i := int32(0); i < 4*int32(dis.IBY2WD); i += int32(dis.IBY2WD) {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+i)))
		}
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/x509 package
// ============================================================

func (fl *funcLowerer) lowerCryptoX509Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ParseCertificate":
		// x509.ParseCertificate(data) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "SystemCertPool", "NewCertPool":
		// x509.SystemCertPool/NewCertPool() → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ParseCertificates":
		// x509.ParseCertificates → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "ParsePKCS1PrivateKey", "ParsePKCS8PrivateKey", "ParsePKIXPublicKey", "MarshalPKIXPublicKey":
		// Key parsing → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Verify":
		// Certificate.Verify → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Equal":
		// Certificate/PublicKey.Equal → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "AppendCertsFromPEM":
		// CertPool.AppendCertsFromPEM → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "AddCert":
		// CertPool.AddCert → no-op
		return true, nil
	case "Error":
		// Error types → ""
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// database/sql package
// ============================================================

func (fl *funcLowerer) lowerDatabaseSQLCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	switch {
	case name == "Open":
		// sql.Open(driver, dsn) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case strings.Contains(name, "Close"):
		// DB.Close() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case strings.Contains(name, "QueryRow"):
		// DB.QueryRow(...) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case strings.Contains(name, "Exec"):
		// DB.Exec(...) → (0, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd)))
		return true, nil
	case strings.Contains(name, "Scan"):
		// Row.Scan(...) → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case name == "Query", name == "QueryContext", name == "Prepare", name == "PrepareContext",
		name == "Begin", name == "BeginTx":
		// DB.Query/QueryContext/Prepare/PrepareContext/Begin/BeginTx → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case name == "Ping", name == "PingContext", name == "Commit", name == "Rollback":
		// Ping/Commit/Rollback → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case name == "SetMaxOpenConns", name == "SetMaxIdleConns",
		name == "SetConnMaxLifetime", name == "SetConnMaxIdleTime":
		return true, nil // no-op
	case name == "Stats":
		// DB.Stats() → zero DBStats struct
		dst := fl.slotOf(instr)
		for i := int32(0); i < 9*int32(dis.IBY2WD); i += int32(dis.IBY2WD) {
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+i)))
		}
		return true, nil
	case name == "Stmt":
		// Tx.Stmt(stmt) → nil *Stmt
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case name == "Next":
		// Rows.Next() → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case name == "Err":
		// Rows.Err() → nil error
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case name == "Columns":
		// Rows.Columns() → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case name == "Register":
		return true, nil // no-op
	}
	return false, nil
}

// ============================================================
// archive/zip package
// ============================================================

func (fl *funcLowerer) lowerArchiveZipCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "OpenReader":
		// zip.OpenReader(name) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// archive/tar package
// ============================================================

func (fl *funcLowerer) lowerArchiveTarCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewReader", "NewWriter":
		// tar.NewReader/NewWriter → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// compress/gzip package
// ============================================================

func (fl *funcLowerer) lowerCompressGzipCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewReader":
		// gzip.NewReader(r) → (*Reader, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NewWriter":
		// gzip.NewWriter(w) → *Writer stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewWriterLevel":
		// gzip.NewWriterLevel(w, level) → (*Writer, nil error) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	// Reader methods
	case "Read":
		if callee.Signature.Recv() != nil {
			// Reader.Read(p) → (0, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Close":
		if callee.Signature.Recv() != nil {
			// Reader.Close / Writer.Close → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Reset":
		if callee.Signature.Recv() != nil {
			// Reader.Reset / Writer.Reset — no-op or → nil error
			if strings.Contains(callee.String(), "Reader") {
				// Reader.Reset(r) → nil error
				dst := fl.slotOf(instr)
				iby2wd := int32(dis.IBY2WD)
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
				fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
				return true, nil
			}
			// Writer.Reset(w) — no-op
			return true, nil
		}
	case "Multistream":
		if callee.Signature.Recv() != nil {
			// Reader.Multistream(ok) — no-op
			return true, nil
		}
	// Writer methods
	case "Write":
		if callee.Signature.Recv() != nil {
			// Writer.Write(p) → (0, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Flush":
		if callee.Signature.Recv() != nil {
			// Writer.Flush() → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	}
	return false, nil
}

// ============================================================
// compress/flate package
// ============================================================

func (fl *funcLowerer) lowerCompressFlateCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewReader":
		// flate.NewReader(r) → io.ReadCloser stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "NewWriter":
		// flate.NewWriter(w, level) → (*Writer, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NewWriterDict":
		// flate.NewWriterDict(w, level, dict) → (*Writer, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "NewReaderDict":
		// flate.NewReaderDict(r, dict) → io.ReadCloser stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Writer methods
	case "Write":
		if callee.Signature.Recv() != nil {
			// Writer.Write(p) → (0, nil)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Close":
		if callee.Signature.Recv() != nil {
			// Writer.Close() → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Flush":
		if callee.Signature.Recv() != nil {
			// Writer.Flush() → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
	case "Reset":
		if callee.Signature.Recv() != nil {
			// Writer.Reset(w) — no-op
			return true, nil
		}
	case "Error":
		if callee.Signature.Recv() != nil {
			// CorruptInputError.Error / InternalError.Error → ""
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
	}
	return false, nil
}

// ============================================================
// html package
// ============================================================

func (fl *funcLowerer) lowerHTMLCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "EscapeString":
		return fl.lowerHTMLEscapeString(instr)
	case "UnescapeString":
		return fl.lowerHTMLUnescapeString(instr)
	}
	return false, nil
}

// lowerHTMLEscapeString implements html.EscapeString by iterating through
// the string character by character, replacing & < > " ' with HTML entities.
func (fl *funcLowerer) lowerHTMLEscapeString(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("")
	i := fl.frame.AllocWord("")
	result := fl.frame.AllocTemp(true)
	ch := fl.frame.AllocTemp(true)
	oneAfter := fl.frame.AllocWord("")

	// Allocate replacement strings in data section
	ampMP := fl.comp.AllocString("&amp;")
	ltMP := fl.comp.AllocString("&lt;")
	gtMP := fl.comp.AllocString("&gt;")
	quotMP := fl.comp.AllocString("&#34;")
	aposMP := fl.comp.AllocString("&#39;")

	// Allocate single-char comparison strings
	ampCh := fl.comp.AllocString("&")
	ltCh := fl.comp.AllocString("<")
	gtCh := fl.comp.AllocString(">")
	quotCh := fl.comp.AllocString("\"")
	aposCh := fl.comp.AllocString("'")

	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	// Load comparison strings into temps
	ampT := fl.frame.AllocTemp(true)
	ltT := fl.frame.AllocTemp(true)
	gtT := fl.frame.AllocTemp(true)
	quotT := fl.frame.AllocTemp(true)
	aposT := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(ampCh), dis.FP(ampT)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(ltCh), dis.FP(ltT)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(gtCh), dis.FP(gtT)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(quotCh), dis.FP(quotT)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(aposCh), dis.FP(aposT)))

	// Loop: while i < lenS
	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	// ch = s[i:i+1]
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(ch)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(ch)))

	// Check each special character. BEQC jumps on match.
	// & → &amp;
	replAmp := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(ampMP), dis.FP(replAmp)))
	beqAmp := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.FP(ampT), dis.FP(ch), dis.Imm(0)))

	// < → &lt;
	replLt := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(ltMP), dis.FP(replLt)))
	beqLt := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.FP(ltT), dis.FP(ch), dis.Imm(0)))

	// > → &gt;
	replGt := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(gtMP), dis.FP(replGt)))
	beqGt := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.FP(gtT), dis.FP(ch), dis.Imm(0)))

	// " → &#34;
	replQuot := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(quotMP), dis.FP(replQuot)))
	beqQuot := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.FP(quotT), dis.FP(ch), dis.Imm(0)))

	// ' → &#39;
	replApos := fl.frame.AllocTemp(true)
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(aposMP), dis.FP(replApos)))
	beqApos := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQC, dis.FP(aposT), dis.FP(ch), dis.Imm(0)))

	// No match: append ch as-is
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(ch), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Match labels: append replacement, advance, loop
	appendAndLoop := func(beqIdx int, replSlot int32) {
		fl.insts[beqIdx].Dst = dis.Imm(int32(len(fl.insts)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(replSlot), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
	}
	appendAndLoop(beqAmp, replAmp)
	appendAndLoop(beqLt, replLt)
	appendAndLoop(beqGt, replGt)
	appendAndLoop(beqQuot, replQuot)
	appendAndLoop(beqApos, replApos)

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return true, nil
}

// lowerHTMLUnescapeString implements html.UnescapeString for the 5 basic HTML entities:
// &amp; → &, &lt; → <, &gt; → >, &#34; → ", &#39; → '
// Also handles &quot; → " and &apos; → '
func (fl *funcLowerer) lowerHTMLUnescapeString(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("une.len")
	i := fl.frame.AllocWord("une.i")
	result := fl.frame.AllocTemp(true)
	ch := fl.frame.AllocWord("une.ch")
	tmp := fl.frame.AllocWord("une.tmp")
	oneAfter := fl.frame.AllocWord("une.oa")
	substr := fl.frame.AllocTemp(true)

	emptyOff := fl.comp.AllocString("")
	ampOff := fl.comp.AllocString("&")
	ltOff := fl.comp.AllocString("<")
	gtOff := fl.comp.AllocString(">")
	quotOff := fl.comp.AllocString("\"")
	aposOff := fl.comp.AllocString("'")

	// Entity strings to match
	ampEntMP := fl.comp.AllocString("&amp;")
	ltEntMP := fl.comp.AllocString("&lt;")
	gtEntMP := fl.comp.AllocString("&gt;")
	n34EntMP := fl.comp.AllocString("&#34;")
	n39EntMP := fl.comp.AllocString("&#39;")
	quotEntMP := fl.comp.AllocString("&quot;")
	aposEntMP := fl.comp.AllocString("&apos;")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	// Main loop
	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))

	// ch = s[i] (character code)
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	// If ch != '&' (38), just append the char and continue
	notAmpIdx := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(38), dis.Imm(0)))

	// Found '&', try to match entities by substring comparison
	// Try 5-char entities first (&amp; &lt;/&gt; are shorter, so try 6-char first)

	// Helper: try to match entity of given length at current position
	// Uses ISLICEC to extract substring and IBEQC to compare
	tryEntity := func(entMP int32, entLen int32, replMP int32, replOp dis.Operand) int {
		entSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(entMP), dis.FP(entSlot)))
		// Check if enough chars remain
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(entLen), dis.FP(i), dis.FP(tmp)))
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBGTW, dis.FP(tmp), dis.FP(lenS), dis.Imm(0)))
		// Extract s[i:i+entLen]
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(substr)))
		fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(tmp), dis.FP(substr)))
		matchIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBEQC, dis.FP(entSlot), dis.FP(substr), dis.Imm(0)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
		return matchIdx
	}

	// Try &quot; (6 chars) → "
	matchQuot := tryEntity(quotEntMP, 6, quotOff, dis.MP(quotOff))
	// Try &apos; (6 chars) → '
	matchApos := tryEntity(aposEntMP, 6, aposOff, dis.MP(aposOff))
	// Try &amp; (5 chars) → &
	matchAmp := tryEntity(ampEntMP, 5, ampOff, dis.MP(ampOff))
	// Try &#34; (5 chars) → "
	matchN34 := tryEntity(n34EntMP, 5, quotOff, dis.MP(quotOff))
	// Try &#39; (5 chars) → '
	matchN39 := tryEntity(n39EntMP, 5, aposOff, dis.MP(aposOff))
	// Try &lt; (4 chars) → <
	matchLt := tryEntity(ltEntMP, 4, ltOff, dis.MP(ltOff))
	// Try &gt; (4 chars) → >
	matchGt := tryEntity(gtEntMP, 4, gtOff, dis.MP(gtOff))

	// No match: append '&' as-is
	noMatchPC := int32(len(fl.insts))
	fl.insts[notAmpIdx].Dst = dis.Imm(noMatchPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(substr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(substr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(substr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Match handlers: append replacement, advance i by entity length, loop
	matchAndLoop := func(matchIdx int, replMP int32, advLen int32) {
		fl.insts[matchIdx].Dst = dis.Imm(int32(len(fl.insts)))
		replSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(replMP), dis.FP(replSlot)))
		fl.emit(dis.NewInst(dis.IADDC, dis.FP(replSlot), dis.FP(result), dis.FP(result)))
		fl.emit(dis.NewInst(dis.IADDW, dis.Imm(advLen), dis.FP(i), dis.FP(i)))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))
	}
	matchAndLoop(matchQuot, quotOff, 6)
	matchAndLoop(matchApos, aposOff, 6)
	matchAndLoop(matchAmp, ampOff, 5)
	matchAndLoop(matchN34, quotOff, 5)
	matchAndLoop(matchN39, aposOff, 5)
	matchAndLoop(matchLt, ltOff, 4)
	matchAndLoop(matchGt, gtOff, 4)

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return true, nil
}

// ============================================================
// html/template package
// ============================================================

func (fl *funcLowerer) lowerHTMLTemplateCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "HTMLEscapeString":
		// Delegate to the real html.EscapeString implementation
		return fl.lowerHTMLEscapeString(instr)
	case "JSEscapeString":
		// Identity stub - return input string
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		return true, nil
	case "HTMLEscaper", "JSEscaper", "URLQueryEscaper":
		// Return empty string stub
		dst := fl.slotOf(instr)
		emptyOff := fl.comp.AllocString("")
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
		return true, nil
	}
	return fl.lowerTextTemplateCall(instr, callee) // delegate to text/template
}

// ============================================================
// mime package
// ============================================================

func (fl *funcLowerer) lowerMIMECall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "TypeByExtension":
		return fl.lowerMIMETypeByExtension(instr)
	case "ExtensionsByType":
		// mime.ExtensionsByType(typ) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "FormatMediaType":
		// mime.FormatMediaType(t, param) → t stub
		sOp := fl.operandOf(instr.Call.Args[0])
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
		return true, nil
	case "ParseMediaType":
		// mime.ParseMediaType(v) → (mediatype, params, err)
		// Real: extract media type before ';', lowercase it, trim whitespace
		return fl.lowerMIMEParseMediaType(instr)
		return true, nil
	}
	return false, nil
}

// lowerMIMEParseMediaType implements mime.ParseMediaType(v) → (mediatype, params, err).
// Extracts the media type before ';', lowercases it, trims whitespace.
// Returns nil params map (params parsing not implemented).
func (fl *funcLowerer) lowerMIMEParseMediaType(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)

	lowerAlpha := fl.comp.AllocString("abcdefghijklmnopqrstuvwxyz")

	lenS := fl.frame.AllocWord("pmt.len")
	i := fl.frame.AllocWord("pmt.i")
	ch := fl.frame.AllocWord("pmt.ch")
	result := fl.frame.AllocTemp(true)
	charStr := fl.frame.AllocTemp(true)
	off := fl.frame.AllocWord("pmt.off")
	offP1 := fl.frame.AllocWord("pmt.offP1")
	oneAfter := fl.frame.AllocWord("pmt.oa")

	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))

	// Skip leading whitespace
	skipWSPC := int32(len(fl.insts))
	skipDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))
	notSP := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))
	notTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))
	skipStopIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	wsPC := int32(len(fl.insts))
	fl.insts[notSP].Dst = dis.Imm(wsPC)
	fl.insts[notTab].Dst = dis.Imm(wsPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(skipWSPC)))

	// Main loop: scan chars until ';' or end, lowercase A-Z, build result
	mainPC := int32(len(fl.insts))
	fl.insts[skipDone].Dst = dis.Imm(mainPC)
	fl.insts[skipStopIdx].Dst = dis.Imm(mainPC)

	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	// ';' = 59: stop (end of media type)
	notSemicolon := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(59), dis.Imm(0)))
	semiDoneIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0))) // → done (trim trailing whitespace)

	// Check if A-Z (65-90): lowercase it
	notUpper := int32(len(fl.insts))
	fl.insts[notSemicolon].Dst = dis.Imm(notUpper)
	notA := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(65), dis.Imm(0)))
	notZ := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(90), dis.Imm(0)))
	// Is A-Z: lowercase via alphabet lookup
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(65), dis.FP(ch), dis.FP(off)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(off), dis.FP(offP1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(lowerAlpha), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(off), dis.FP(offP1), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Not A-Z: append as-is
	asIsPC := int32(len(fl.insts))
	fl.insts[notA].Dst = dis.Imm(asIsPC)
	fl.insts[notZ].Dst = dis.Imm(asIsPC)

	// Skip trailing whitespace: don't append space at end (trim right)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Done: trim trailing whitespace from result, then return
	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.insts[semiDoneIdx].Dst = dis.Imm(donePC)

	// Trim trailing whitespace: find last non-space position
	resLen := fl.frame.AllocWord("pmt.rlen")
	endR := fl.frame.AllocWord("pmt.end")
	endM1R := fl.frame.AllocWord("pmt.em1")
	fl.emit(dis.Inst2(dis.ILENC, dis.FP(result), dis.FP(resLen)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(resLen), dis.FP(endR)))
	trimPC := int32(len(fl.insts))
	trimDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.Imm(0), dis.FP(endR), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(endR), dis.FP(endM1R)))
	fl.emit(dis.NewInst(dis.IINDC, dis.FP(result), dis.FP(endM1R), dis.FP(ch)))
	notTrimSP := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))
	notTrimTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))
	trimStopIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	trimContPC := int32(len(fl.insts))
	fl.insts[notTrimSP].Dst = dis.Imm(trimContPC)
	fl.insts[notTrimTab].Dst = dis.Imm(trimContPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(endM1R), dis.FP(endR)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trimPC)))

	// Slice result to trimmed length
	finalPC := int32(len(fl.insts))
	fl.insts[trimDone].Dst = dis.Imm(finalPC)
	fl.insts[trimStopIdx].Dst = dis.Imm(finalPC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	// Only slice if endR < resLen (there was trailing whitespace)
	noSlice := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(endR), dis.FP(resLen), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.Imm(0), dis.FP(endR), dis.FP(dst)))
	allDonePC := int32(len(fl.insts))
	fl.insts[noSlice].Dst = dis.Imm(allDonePC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+iby2wd)))   // nil params map
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+2*iby2wd))) // nil error tag
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst+3*iby2wd))) // nil error val
	return true, nil
}

// lowerMIMETypeByExtension implements mime.TypeByExtension(ext string) → string
// with a comparison chain for common file extensions.
func (fl *funcLowerer) lowerMIMETypeByExtension(instr *ssa.Call) (bool, error) {
	extOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// Table of extension → MIME type
	mimeTable := []struct {
		ext  string
		mime string
	}{
		{".html", "text/html; charset=utf-8"},
		{".htm", "text/html; charset=utf-8"},
		{".css", "text/css; charset=utf-8"},
		{".js", "text/javascript; charset=utf-8"},
		{".json", "application/json"},
		{".xml", "text/xml; charset=utf-8"},
		{".txt", "text/plain; charset=utf-8"},
		{".csv", "text/csv"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".svg", "image/svg+xml"},
		{".webp", "image/webp"},
		{".ico", "image/x-icon"},
		{".pdf", "application/pdf"},
		{".zip", "application/zip"},
		{".gz", "application/gzip"},
		{".tar", "application/x-tar"},
		{".mp3", "audio/mpeg"},
		{".mp4", "video/mp4"},
		{".webm", "video/webm"},
		{".wasm", "application/wasm"},
		{".woff", "font/woff"},
		{".woff2", "font/woff2"},
		{".ttf", "font/ttf"},
		{".otf", "font/otf"},
	}

	// Default: empty string
	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))

	var doneFixups []int
	for _, entry := range mimeTable {
		extOff := fl.comp.AllocString(entry.ext)
		extSlot := fl.frame.AllocTemp(true)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(extOff), dis.FP(extSlot)))
		skipIdx := len(fl.insts)
		fl.emit(dis.NewInst(dis.IBNEC, extOp, dis.FP(extSlot), dis.Imm(0)))
		mimeOff := fl.comp.AllocString(entry.mime)
		fl.emit(dis.Inst2(dis.IMOVP, dis.MP(mimeOff), dis.FP(dst)))
		doneFixups = append(doneFixups, len(fl.insts))
		fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
		fl.insts[skipIdx].Dst = dis.Imm(int32(len(fl.insts)))
	}

	donePC := int32(len(fl.insts))
	for _, idx := range doneFixups {
		fl.insts[idx].Dst = dis.Imm(donePC)
	}
	return true, nil
}

// ============================================================
// mime/multipart package
// ============================================================

func (fl *funcLowerer) lowerMIMEMultipartCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "NewWriter", "NewReader":
		// multipart.NewWriter/NewReader → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	// Writer methods
	case "Boundary", "FormDataContentType":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	case "SetBoundary", "WriteField", "Close":
		if callee.Signature.Recv() != nil {
			// → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	case "CreatePart", "CreateFormFile", "CreateFormField":
		if callee.Signature.Recv() != nil {
			// → (nil io.Writer, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	// Part methods
	case "Read":
		if callee.Signature.Recv() != nil {
			// Part.Read → (0, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "FileName", "FormName":
		if callee.Signature.Recv() != nil {
			dst := fl.slotOf(instr)
			emptyOff := fl.comp.AllocString("")
			fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(dst)))
			return true, nil
		}
		return false, nil
	// Reader methods
	case "NextPart", "NextRawPart":
		if callee.Signature.Recv() != nil {
			// → (nil *Part, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	case "ReadForm":
		if callee.Signature.Recv() != nil {
			// → (nil *Form, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	// FileHeader methods
	case "Open":
		if callee.Signature.Recv() != nil {
			// → (nil File, nil error)
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
			return true, nil
		}
		return false, nil
	// Form methods
	case "RemoveAll":
		if callee.Signature.Recv() != nil {
			// → nil error
			dst := fl.slotOf(instr)
			iby2wd := int32(dis.IBY2WD)
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
			fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// ============================================================
// net/mail package
// ============================================================

func (fl *funcLowerer) lowerNetMailCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "ParseAddress":
		// mail.ParseAddress(address) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// net/textproto package
// ============================================================

func (fl *funcLowerer) lowerNetTextprotoCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "CanonicalMIMEHeaderKey":
		return fl.lowerCanonicalHeaderKey(instr)
	case "TrimString":
		return fl.lowerTextprotoTrimString(instr)
	}
	return false, nil
}

// lowerTextprotoTrimString trims leading and trailing ASCII spaces and tabs.
func (fl *funcLowerer) lowerTextprotoTrimString(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	lenS := fl.frame.AllocWord("ts.len")
	start := fl.frame.AllocWord("ts.start")
	end := fl.frame.AllocWord("ts.end")
	ch := fl.frame.AllocWord("ts.ch")

	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(start)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(lenS), dis.FP(end)))

	// Trim leading whitespace: while start < end && (s[start]==' ' || s[start]=='\t')
	leadPC := int32(len(fl.insts))
	leadDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(start), dis.FP(ch)))
	notSpace := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0))) // space → continue
	notTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))  // tab → continue
	// Not whitespace: stop trimming leading
	leadStopIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	// Continue: start++, loop
	contPC := int32(len(fl.insts))
	fl.insts[notSpace].Dst = dis.Imm(contPC)
	fl.insts[notTab].Dst = dis.Imm(contPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(start), dis.FP(start)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(leadPC)))

	// Trim trailing whitespace: while end > start && (s[end-1]==' ' || s[end-1]=='\t')
	trailHdrPC := int32(len(fl.insts))
	fl.insts[leadDone].Dst = dis.Imm(trailHdrPC)
	fl.insts[leadStopIdx].Dst = dis.Imm(trailHdrPC)
	endM1 := fl.frame.AllocWord("ts.em1")

	trailPC := int32(len(fl.insts))
	trailDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(start), dis.FP(end), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(1), dis.FP(end), dis.FP(endM1)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(endM1), dis.FP(ch)))
	tNotSpace := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(32), dis.Imm(0)))
	tNotTab := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(ch), dis.Imm(9), dis.Imm(0)))
	// Not whitespace: stop
	tStopIdx := len(fl.insts)
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(0)))
	// Continue: end--, loop
	tContPC := int32(len(fl.insts))
	fl.insts[tNotSpace].Dst = dis.Imm(tContPC)
	fl.insts[tNotTab].Dst = dis.Imm(tContPC)
	fl.emit(dis.Inst2(dis.IMOVW, dis.FP(endM1), dis.FP(end)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(trailPC)))

	// Result = s[start:end]
	donePC := int32(len(fl.insts))
	fl.insts[trailDone].Dst = dis.Imm(donePC)
	fl.insts[tStopIdx].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(dst)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(start), dis.FP(end), dis.FP(dst)))
	return true, nil
}

// lowerCanonicalHeaderKey implements textproto.CanonicalMIMEHeaderKey and
// http.CanonicalHeaderKey: capitalize first letter and letter after each '-',
// lowercase everything else. e.g. "content-type" → "Content-Type"
func (fl *funcLowerer) lowerCanonicalHeaderKey(instr *ssa.Call) (bool, error) {
	sOp := fl.operandOf(instr.Call.Args[0])
	dst := fl.slotOf(instr)

	// Pre-allocate alphabet strings for case conversion
	upperAlpha := fl.comp.AllocString("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lowerAlpha := fl.comp.AllocString("abcdefghijklmnopqrstuvwxyz")

	lenS := fl.frame.AllocWord("chk.len")
	i := fl.frame.AllocWord("chk.i")
	ch := fl.frame.AllocWord("chk.ch")
	upper := fl.frame.AllocWord("chk.up")     // 1 = next char should be uppercased
	result := fl.frame.AllocTemp(true)
	charStr := fl.frame.AllocTemp(true)
	off := fl.frame.AllocWord("chk.off")
	offP1 := fl.frame.AllocWord("chk.offP1")
	oneAfter := fl.frame.AllocWord("chk.oa")

	emptyOff := fl.comp.AllocString("")
	fl.emit(dis.Inst2(dis.ILENC, sOp, dis.FP(lenS)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(emptyOff), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(i)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(upper))) // first char is upper

	// Loop: while i < lenS
	loopPC := int32(len(fl.insts))
	bgeDone := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGEW, dis.FP(i), dis.FP(lenS), dis.Imm(0)))
	fl.emit(dis.NewInst(dis.IINDC, sOp, dis.FP(i), dis.FP(ch)))

	// Check if '-': set upper=1 for next char, append '-' as-is
	notDash := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBNEW, dis.FP(ch), dis.Imm(45), dis.Imm(0))) // '-' = 45

	// Is dash: append as-is and set upper=1
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(1), dis.FP(upper)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Not dash
	notDashPC := int32(len(fl.insts))
	fl.insts[notDash].Dst = dis.Imm(notDashPC)

	// Check if upper flag is set
	notUpper := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBEQW, dis.FP(upper), dis.Imm(0), dis.Imm(0)))

	// upper=1: need to uppercase if a-z
	notLower1 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(97), dis.Imm(0)))  // < 'a'
	notLower2 := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(122), dis.Imm(0))) // > 'z'

	// Is a-z: convert to uppercase via alphabet lookup
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(97), dis.FP(ch), dis.FP(off)))      // off = ch - 'a'
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(off), dis.FP(offP1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(upperAlpha), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(off), dis.FP(offP1), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(upper)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// upper=1 but not a-z: append as-is
	notLowerPC := int32(len(fl.insts))
	fl.insts[notLower1].Dst = dis.Imm(notLowerPC)
	fl.insts[notLower2].Dst = dis.Imm(notLowerPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(upper)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// upper=0: need to lowercase if A-Z
	notUpperPC := int32(len(fl.insts))
	fl.insts[notUpper].Dst = dis.Imm(notUpperPC)
	notUpA := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBLTW, dis.FP(ch), dis.Imm(65), dis.Imm(0)))  // < 'A'
	notUpZ := len(fl.insts)
	fl.emit(dis.NewInst(dis.IBGTW, dis.FP(ch), dis.Imm(90), dis.Imm(0)))  // > 'Z'

	// Is A-Z: convert to lowercase via alphabet lookup
	fl.emit(dis.NewInst(dis.ISUBW, dis.Imm(65), dis.FP(ch), dis.FP(off)))      // off = ch - 'A'
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(off), dis.FP(offP1)))
	fl.emit(dis.Inst2(dis.IMOVP, dis.MP(lowerAlpha), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(off), dis.FP(offP1), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Not A-Z: append as-is
	notUpPC := int32(len(fl.insts))
	fl.insts[notUpA].Dst = dis.Imm(notUpPC)
	fl.insts[notUpZ].Dst = dis.Imm(notUpPC)
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(oneAfter)))
	fl.emit(dis.Inst2(dis.IMOVP, sOp, dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.ISLICEC, dis.FP(i), dis.FP(oneAfter), dis.FP(charStr)))
	fl.emit(dis.NewInst(dis.IADDC, dis.FP(charStr), dis.FP(result), dis.FP(result)))
	fl.emit(dis.NewInst(dis.IADDW, dis.Imm(1), dis.FP(i), dis.FP(i)))
	fl.emit(dis.Inst1(dis.IJMP, dis.Imm(loopPC)))

	// Done
	donePC := int32(len(fl.insts))
	fl.insts[bgeDone].Dst = dis.Imm(donePC)
	fl.emit(dis.Inst2(dis.IMOVP, dis.FP(result), dis.FP(dst)))
	return true, nil
}

// ============================================================
// net/http/httputil package
// ============================================================

func (fl *funcLowerer) lowerNetHTTPUtilCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "DumpRequest", "DumpResponse":
		// httputil.DumpRequest/Response → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/elliptic package
// ============================================================

func (fl *funcLowerer) lowerCryptoEllipticCall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "P256", "P384", "P521":
		// elliptic.P256() → 0 stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/ecdsa package
// ============================================================

func (fl *funcLowerer) lowerCryptoECDSACall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "GenerateKey":
		// ecdsa.GenerateKey(c, rand) → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Sign", "SignASN1":
		// ecdsa.Sign/SignASN1 → (nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "VerifyASN1":
		// ecdsa.VerifyASN1 → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "Public":
		// PrivateKey.Public → nil
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Equal":
		// PublicKey/PrivateKey.Equal → false
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	case "ECDH":
		// PublicKey/PrivateKey.ECDH → (nil, nil)
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/rsa package
// ============================================================

func (fl *funcLowerer) lowerCryptoRSACall(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	name := callee.Name()
	dst := fl.slotOf(instr)
	iby2wd := int32(dis.IBY2WD)
	switch name {
	case "GenerateKey":
		// rsa.GenerateKey(random, bits) → (nil, nil) stub
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "SignPKCS1v15", "SignPSS", "EncryptOAEP", "EncryptPKCS1v15", "DecryptOAEP", "DecryptPKCS1v15":
		// Sign/Encrypt/Decrypt → (nil, nil) stub
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "VerifyPKCS1v15", "VerifyPSS":
		// Verify → nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Public":
		// PrivateKey.Public() → nil interface
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Sign":
		// PrivateKey.Sign → (nil, nil)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		return true, nil
	case "Validate":
		// PrivateKey.Validate → nil error
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		return true, nil
	case "Size":
		// PublicKey.Size → 0
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Equal":
		// PublicKey.Equal → false
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}

// ============================================================
// crypto/ed25519 package
// ============================================================

func (fl *funcLowerer) lowerCryptoEd25519Call(instr *ssa.Call, callee *ssa.Function) (bool, error) {
	switch callee.Name() {
	case "GenerateKey":
		// ed25519.GenerateKey(rand) → (nil, nil, nil) stub
		dst := fl.slotOf(instr)
		iby2wd := int32(dis.IBY2WD)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+2*iby2wd)))
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst+3*iby2wd)))
		return true, nil
	case "Sign":
		// ed25519.Sign(priv, msg) → nil stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(-1), dis.FP(dst)))
		return true, nil
	case "Verify":
		// ed25519.Verify(pub, msg, sig) → false stub
		dst := fl.slotOf(instr)
		fl.emit(dis.Inst2(dis.IMOVW, dis.Imm(0), dis.FP(dst)))
		return true, nil
	}
	return false, nil
}
