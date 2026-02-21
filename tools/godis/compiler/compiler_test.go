package compiler

import (
	"testing"

	"github.com/NERVsystems/infernode/tools/godis/dis"
)

func TestCompileHelloWorld(t *testing.T) {
	src := []byte(`package main

func main() {
	println("hello, infernode")
}
`)
	c := New()
	m, err := c.CompileFile("hello.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Verify module structure
	if m.Name != "Hello" {
		t.Errorf("name = %q, want %q", m.Name, "Hello")
	}
	if m.Magic != dis.XMAGIC {
		t.Errorf("magic = %d, want %d", m.Magic, dis.XMAGIC)
	}
	if m.RuntimeFlags != dis.HASLDT {
		t.Errorf("flags = 0x%x, want 0x%x", m.RuntimeFlags, dis.HASLDT)
	}

	// Must have instructions
	if len(m.Instructions) < 5 {
		t.Errorf("instructions = %d, want >= 5", len(m.Instructions))
	}

	// First instruction must be LOAD (loading the Sys module)
	if m.Instructions[0].Op != dis.ILOAD {
		t.Errorf("inst[0].op = %s, want load", m.Instructions[0].Op)
	}

	// Last instruction must be RET
	last := m.Instructions[len(m.Instructions)-1]
	if last.Op != dis.IRET {
		t.Errorf("last inst = %s, want ret", last.Op)
	}

	// Must have at least 2 type descriptors (MP + init frame)
	if len(m.TypeDescs) < 2 {
		t.Errorf("type descs = %d, want >= 2", len(m.TypeDescs))
	}

	// Must have the init link with correct signature
	if len(m.Links) != 1 {
		t.Fatalf("links = %d, want 1", len(m.Links))
	}
	if m.Links[0].Name != "init" {
		t.Errorf("link name = %q, want %q", m.Links[0].Name, "init")
	}
	if m.Links[0].Sig != 0x4244b354 {
		t.Errorf("link sig = 0x%x, want 0x4244b354", m.Links[0].Sig)
	}

	// Must have LDT with print import
	if len(m.LDT) != 1 || len(m.LDT[0]) == 0 {
		t.Fatalf("LDT entries: got %v, want 1 entry with imports", len(m.LDT))
	}
	found := false
	for _, imp := range m.LDT[0] {
		if imp.Name == "print" {
			found = true
			if imp.Sig != 0xac849033 {
				t.Errorf("print sig = 0x%x, want 0xac849033", imp.Sig)
			}
		}
	}
	if !found {
		t.Error("LDT missing print import")
	}

	// Must have data section with the hello string
	foundStr := false
	for _, d := range m.Data {
		if d.Kind == dis.DEFS && d.Str == "hello, infernode" {
			foundStr = true
		}
	}
	if !foundStr {
		t.Error("data section missing 'hello, infernode' string")
	}

	// Must round-trip encode/decode
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	m2, err := dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	reencoded, err := m2.EncodeToBytes()
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if len(encoded) != len(reencoded) {
		t.Errorf("round-trip size: %d -> %d", len(encoded), len(reencoded))
	}
	for i := range encoded {
		if encoded[i] != reencoded[i] {
			t.Errorf("round-trip mismatch at byte %d: 0x%02x != 0x%02x", i, encoded[i], reencoded[i])
			break
		}
	}

	t.Logf("compiled %d instructions, %d type descs, %d data items, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(m.Data), len(encoded))
}

func TestCompileArithmetic(t *testing.T) {
	src := []byte(`package main

func main() {
	x := 40
	y := 2
	println(x + y)
}
`)
	c := New()
	m, err := c.CompileFile("arith.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if len(m.Instructions) < 5 {
		t.Errorf("too few instructions: %d", len(m.Instructions))
	}

	// Verify it encodes correctly
	_, err = m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs", len(m.Instructions), len(m.TypeDescs))
}

func TestCompileLocalFunctionCall(t *testing.T) {
	src := []byte(`package main

func add(a, b int) int {
	return a + b
}

func main() {
	result := add(40, 2)
	println(result)
}
`)
	c := New()
	m, err := c.CompileFile("funcall.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have type descriptors for MP + main frame + add frame + call-site TDs
	if len(m.TypeDescs) < 3 {
		t.Errorf("type descs = %d, want >= 3 (MP + main + add)", len(m.TypeDescs))
	}

	// Must have IFRAME and CALL instructions for the local call
	hasFrame := false
	hasCall := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.IFRAME {
			hasFrame = true
		}
		if inst.Op == dis.ICALL {
			hasCall = true
		}
	}
	if !hasFrame {
		t.Error("missing IFRAME instruction for local call")
	}
	if !hasCall {
		t.Error("missing CALL instruction for local call")
	}

	// The add function must have a RET that writes through REGRET
	// Look for movw ... 0(32(fp)) pattern (indirect write to REGRET)
	hasReturnWrite := false
	for _, inst := range m.Instructions {
		if (inst.Op == dis.IMOVW || inst.Op == dis.IMOVP) && inst.Dst.IsIndirect() {
			if inst.Dst.Val == 32 && inst.Dst.Ind == 0 {
				hasReturnWrite = true
			}
		}
	}
	if !hasReturnWrite {
		t.Error("add() missing return value write through REGRET (0(32(fp)))")
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	m2, err := dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	reencoded, err := m2.EncodeToBytes()
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if len(encoded) != len(reencoded) {
		t.Errorf("round-trip size: %d -> %d", len(encoded), len(reencoded))
	}

	// Print instruction listing for debugging
	for i, inst := range m.Instructions {
		t.Logf("  [%3d] %s", i, inst.String())
	}
	for i, td := range m.TypeDescs {
		t.Logf("  td[%d]: id=%d size=%d map=%v", i, td.ID, td.Size, td.Map)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileVoidFunctionCall(t *testing.T) {
	src := []byte(`package main

func greet(name string) {
	println("hello", name)
}

func main() {
	greet("world")
}
`)
	c := New()
	m, err := c.CompileFile("greet.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must compile without errors and have instructions
	if len(m.Instructions) < 5 {
		t.Errorf("too few instructions: %d", len(m.Instructions))
	}

	// Must have CALL for calling greet
	hasCall := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICALL {
			hasCall = true
		}
	}
	if !hasCall {
		t.Error("missing CALL instruction for greet()")
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompilePhiElimination(t *testing.T) {
	// This program has a phi node: x gets different values depending on the branch.
	// The phi elimination must insert MOVs in each predecessor block.
	src := []byte(`package main

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	println(abs(-7))
	println(abs(3))
}
`)
	c := New()
	m, err := c.CompileFile("phi.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must compile and encode correctly
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Print instruction listing for debugging
	for i, inst := range m.Instructions {
		t.Logf("  [%3d] %s", i, inst.String())
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileConditionalValue(t *testing.T) {
	// Tests proper phi elimination where a variable gets different values
	// from different control flow paths
	src := []byte(`package main

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	println(max(10, 20))
}
`)
	c := New()
	m, err := c.CompileFile("max.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileForLoop(t *testing.T) {
	src := []byte(`package main

func loop(n int) int {
	sum := 0
	i := 0
	for i < n {
		sum = sum + i
		i = i + 1
	}
	return sum
}

func main() {
	println(loop(5))
	println(loop(10))
}
`)
	c := New()
	m, err := c.CompileFile("loop.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStringOperations(t *testing.T) {
	src := []byte(`package main

func classify(s string) int {
	if s == "hello" {
		return 1
	}
	if s == "world" {
		return 2
	}
	return 0
}

func longer(a, b string) string {
	if len(a) > len(b) {
		return a
	}
	return b
}

func main() {
	println(classify("hello"))
	println(classify("world"))
	println(classify("other"))
	println(longer("hi", "hello"))
}
`)
	c := New()
	m, err := c.CompileFile("strings.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have BEQC for string comparison
	hasBeqc := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.IBEQC {
			hasBeqc = true
		}
	}
	if !hasBeqc {
		t.Error("missing BEQC instruction for string comparison")
	}

	// Must have LENC for len()
	hasLenc := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.ILENC {
			hasLenc = true
		}
	}
	if !hasLenc {
		t.Error("missing LENC instruction for string len()")
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStringConcatenation(t *testing.T) {
	src := []byte(`package main

func greet(first, last string) string {
	return first + " " + last
}

func main() {
	msg := greet("hello", "world")
	println(msg)
}
`)
	c := New()
	m, err := c.CompileFile("strcat.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have ADDC for string concatenation
	addcCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.IADDC {
			addcCount++
		}
	}
	if addcCount < 2 {
		t.Errorf("ADDC count = %d, want >= 2 (two concatenations)", addcCount)
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileMultipleFunctionCalls(t *testing.T) {
	src := []byte(`package main

func double(x int) int {
	return x + x
}

func square(x int) int {
	return x * x
}

func main() {
	a := double(5)
	b := square(3)
	println(a + b)
}
`)
	c := New()
	m, err := c.CompileFile("multi.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have at least 4 type descriptors: MP + main + double + square
	if len(m.TypeDescs) < 4 {
		t.Errorf("type descs = %d, want >= 4", len(m.TypeDescs))
	}

	// Must have 2 CALL instructions (for double and square)
	callCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICALL {
			callCount++
		}
	}
	if callCount != 2 {
		t.Errorf("CALL count = %d, want 2", callCount)
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileGlobalVariable(t *testing.T) {
	src := []byte(`package main

var counter int

func increment() {
	counter = counter + 1
}

func main() {
	increment()
	increment()
	increment()
	println(counter)
}
`)
	c := New()
	m, err := c.CompileFile("global.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have LEA instructions for loading global addresses from MP
	leaCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ILEA {
			leaCount++
		}
	}
	if leaCount < 1 {
		t.Error("missing LEA instruction for global variable access")
	}

	// Global storage should be in module data (MP), increasing data size
	if m.DataSize < 24 {
		t.Errorf("data size = %d, want >= 24 (must include global storage)", m.DataSize)
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStructFieldAccess(t *testing.T) {
	src := []byte(`package main

type Point struct {
	X int
	Y int
}

func main() {
	var p Point
	p.X = 3
	p.Y = 4
	println(p.X + p.Y)
}
`)
	c := New()
	m, err := c.CompileFile("point.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have LEA for struct field addresses
	leaCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ILEA {
			leaCount++
		}
	}
	if leaCount < 3 {
		t.Errorf("LEA count = %d, want >= 3 (alloc base + 2 field accesses minimum)", leaCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("struct: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStructByValue(t *testing.T) {
	src := []byte(`package main

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func area(r Rect) int {
	return r.Width * r.Height
}

func main() {
	var r Rect
	r.X = 10
	r.Y = 20
	r.Width = 30
	r.Height = 40
	println(area(r))
}
`)
	c := New()
	m, err := c.CompileFile("rect.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have CALL for area()
	hasCall := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICALL {
			hasCall = true
		}
	}
	if !hasCall {
		t.Error("missing CALL instruction for area()")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("struct-by-value: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}
