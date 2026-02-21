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

func TestCompileHeapAllocation(t *testing.T) {
	src := []byte(`package main

type Point struct {
	X int
	Y int
}

func newPoint(x, y int) *Point {
	return &Point{X: x, Y: y}
}

func main() {
	p := newPoint(3, 4)
	println(p.X + p.Y)
}
`)
	c := New()
	m, err := c.CompileFile("heap.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have INEW for heap allocation
	hasNew := false
	// Must have CALL for newPoint
	hasCall := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.INEW {
			hasNew = true
		}
		if inst.Op == dis.ICALL {
			hasCall = true
		}
	}
	if !hasNew {
		t.Error("missing NEW instruction for heap allocation")
	}
	if !hasCall {
		t.Error("missing CALL instruction for newPoint()")
	}

	// Must have a heap type descriptor (size 16 for Point{X int, Y int})
	foundHeapTD := false
	for _, td := range m.TypeDescs {
		if td.Size == 16 {
			foundHeapTD = true
		}
	}
	if !foundHeapTD {
		t.Error("missing 16-byte type descriptor for heap Point")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("heap-alloc: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileSliceOperations(t *testing.T) {
	src := []byte(`package main

func sum(nums []int) int {
	total := 0
	for i := 0; i < len(nums); i++ {
		total += nums[i]
	}
	return total
}

func main() {
	a := []int{10, 20, 30}
	println(sum(a))
	println(len(a))
}
`)
	c := New()
	m, err := c.CompileFile("slice.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have NEWA for array creation
	hasNewa := false
	// Must have INDW for slice indexing
	hasIndw := false
	// Must have LENA for len()
	hasLena := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.INEWA {
			hasNewa = true
		}
		if inst.Op == dis.IINDW {
			hasIndw = true
		}
		if inst.Op == dis.ILENA {
			hasLena = true
		}
	}
	if !hasNewa {
		t.Error("missing NEWA instruction for slice creation")
	}
	if !hasIndw {
		t.Error("missing INDW instruction for slice indexing")
	}
	if !hasLena {
		t.Error("missing LENA instruction for len()")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("slice: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileMultipleReturnValues(t *testing.T) {
	src := []byte(`package main

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

func main() {
	q, r := divmod(17, 5)
	println(q)
	println(r)
}
`)
	c := New()
	m, err := c.CompileFile("multiret.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// divmod should write two values through REGRET
	// Check that we have DIVW and MODW in divmod
	hasDivw := false
	hasModw := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.IDIVW {
			hasDivw = true
		}
		if inst.Op == dis.IMODW {
			hasModw = true
		}
	}
	if !hasDivw {
		t.Error("missing DIVW instruction")
	}
	if !hasModw {
		t.Error("missing MODW instruction")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("multiret: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileMethodCalls(t *testing.T) {
	src := []byte(`package main

type Counter struct {
	value int
}

func (c *Counter) Inc() {
	c.value++
}

func (c *Counter) Get() int {
	return c.value
}

func main() {
	c := &Counter{value: 0}
	c.Inc()
	c.Inc()
	c.Inc()
	println(c.Get())
}
`)
	c := New()
	m, err := c.CompileFile("method.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have at least 4 type descriptors: MP + main + Get + Inc
	if len(m.TypeDescs) < 4 {
		t.Errorf("type descs = %d, want >= 4 (MP + main + Get + Inc)", len(m.TypeDescs))
	}

	// Must have 4 CALL instructions (3x Inc + 1x Get)
	callCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICALL {
			callCount++
		}
	}
	if callCount != 4 {
		t.Errorf("CALL count = %d, want 4 (3x Inc + 1x Get)", callCount)
	}

	// Must have INEW for heap-allocated Counter
	hasNew := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.INEW {
			hasNew = true
		}
	}
	if !hasNew {
		t.Error("missing NEW instruction for Counter allocation")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("method: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileSysModuleCalls(t *testing.T) {
	src := []byte(`package main

import "inferno/sys"

func main() {
	fd := sys.Fildes(1)
	sys.Fprint(fd, "hello\n")
	t1 := sys.Millisec()
	sys.Sleep(10)
	t2 := sys.Millisec()
	println(t2 - t1)
}
`)
	c := New()
	m, err := c.CompileFile("syscall.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have MFRAME + MCALL pairs for sys module calls
	mframeCount := 0
	mcallCount := 0
	iframeCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.IMFRAME {
			mframeCount++
		}
		if inst.Op == dis.IMCALL {
			mcallCount++
		}
		if inst.Op == dis.IFRAME {
			iframeCount++
		}
	}
	// fildes + sleep + millisec*2 = 4 MFRAME calls
	// fprint = 1 IFRAME call (varargs)
	if mframeCount < 4 {
		t.Errorf("MFRAME count = %d, want >= 4 (fildes + sleep + 2x millisec)", mframeCount)
	}
	if iframeCount < 1 {
		t.Errorf("IFRAME count = %d, want >= 1 (fprint is varargs)", iframeCount)
	}
	if mcallCount < 5 {
		t.Errorf("MCALL count = %d, want >= 5 (fildes + fprint + millisec + sleep + millisec)", mcallCount)
	}

	// LDT must have entries for fildes, fprint, sleep, millisec (plus print)
	if len(m.LDT) != 1 {
		t.Fatalf("LDT entries = %d, want 1", len(m.LDT))
	}
	foundNames := make(map[string]bool)
	for _, imp := range m.LDT[0] {
		foundNames[imp.Name] = true
	}
	for _, name := range []string{"print", "fildes", "fprint", "sleep", "millisec"} {
		if !foundNames[name] {
			t.Errorf("LDT missing %q import", name)
		}
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("syscall: compiled %d instructions, %d type descs, %d LDT imports, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(m.LDT[0]), len(encoded))
}

func TestCompileByteArrays(t *testing.T) {
	src := []byte(`package main

func main() {
	buf := []byte{72, 101, 108, 108, 111}
	sum := 0
	for i := 0; i < len(buf); i++ {
		sum = sum + int(buf[i])
	}
	println(sum)
}
`)
	c := New()
	m, err := c.CompileFile("bytes.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have INDB for byte indexing
	hasIndb := false
	// Must have CVTBW for byte→word conversion
	hasCvtbw := false
	// Must have CVTWB for word→byte store
	hasCvtwb := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.IINDB {
			hasIndb = true
		}
		if inst.Op == dis.ICVTBW {
			hasCvtbw = true
		}
		if inst.Op == dis.ICVTWB {
			hasCvtwb = true
		}
	}
	if !hasIndb {
		t.Error("missing INDB instruction for byte indexing")
	}
	if !hasCvtbw {
		t.Error("missing CVTBW instruction for byte→word conversion")
	}
	if !hasCvtwb {
		t.Error("missing CVTWB instruction for word→byte store")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("bytes: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileGoroutines(t *testing.T) {
	src := []byte(`package main

func worker(id int) {
	println(id)
}

func main() {
	go worker(1)
	go worker(2)
	println("done")
}
`)
	c := New()
	m, err := c.CompileFile("goroutine.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have SPAWN instructions
	spawnCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ISPAWN {
			spawnCount++
		}
	}
	if spawnCount != 2 {
		t.Errorf("SPAWN count = %d, want 2", spawnCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("goroutine: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileChannels(t *testing.T) {
	src := []byte(`package main

func worker(ch chan int) {
	ch <- 42
}

func main() {
	ch := make(chan int)
	go worker(ch)
	v := <-ch
	println(v)
}
`)
	c := New()
	m, err := c.CompileFile("channel.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Verify NEWCW, SEND, RECV instructions are present
	var hasNewcw, hasSend, hasRecv bool
	for _, inst := range m.Instructions {
		switch inst.Op {
		case dis.INEWCW:
			hasNewcw = true
		case dis.ISEND:
			hasSend = true
		case dis.IRECV:
			hasRecv = true
		}
	}
	if !hasNewcw {
		t.Error("expected NEWCW instruction")
	}
	if !hasSend {
		t.Error("expected SEND instruction")
	}
	if !hasRecv {
		t.Error("expected RECV instruction")
	}

	// Verify encode/decode round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("channel: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileSelect(t *testing.T) {
	src := []byte(`package main

func sender1(ch chan int) { ch <- 10 }
func sender2(ch chan int) { ch <- 20 }

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	go sender1(ch1)
	go sender2(ch2)
	select {
	case v := <-ch1:
		println(v)
	case v := <-ch2:
		println(v)
	}
}
`)
	c := New()
	m, err := c.CompileFile("selectrecv.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Verify ALT instruction is present
	var hasAlt bool
	for _, inst := range m.Instructions {
		if inst.Op == dis.IALT {
			hasAlt = true
		}
	}
	if !hasAlt {
		t.Error("expected ALT instruction")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("select: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileAppend(t *testing.T) {
	src := []byte(`package main

func main() {
	s := make([]int, 0)
	s = append(s, 10)
	s = append(s, 20)
	s = append(s, 30)
	println(len(s))
	println(s[0])
	println(s[1])
	println(s[2])
}
`)
	c := New()
	m, err := c.CompileFile("append.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have SLICELA for slice concatenation
	slicelaCount := 0
	// Must have NEWA for new array allocation
	newaCount := 0
	// Must have LENA for getting slice lengths
	lenaCount := 0
	for _, inst := range m.Instructions {
		if inst.Op == dis.ISLICELA {
			slicelaCount++
		}
		if inst.Op == dis.INEWA {
			newaCount++
		}
		if inst.Op == dis.ILENA {
			lenaCount++
		}
	}
	// Each append(s, elem) creates a temp slice + concatenates = 2 SLICELA per append
	if slicelaCount < 6 {
		t.Errorf("SLICELA count = %d, want >= 6 (2 per append x 3 appends)", slicelaCount)
	}
	if newaCount < 3 {
		t.Errorf("NEWA count = %d, want >= 3 (one per append)", newaCount)
	}
	if lenaCount < 6 {
		t.Errorf("LENA count = %d, want >= 6 (2 per append for old+new lengths)", lenaCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("append: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStringConversion(t *testing.T) {
	src := []byte(`package main

func main() {
	s := "hello"
	b := []byte(s)
	println(len(b))
	s2 := string(b)
	println(s2)
}
`)
	c := New()
	m, err := c.CompileFile("strconv.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have CVTCA for string→[]byte
	hasCvtca := false
	// Must have CVTAC for []byte→string
	hasCvtac := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICVTCA {
			hasCvtca = true
		}
		if inst.Op == dis.ICVTAC {
			hasCvtac = true
		}
	}
	if !hasCvtca {
		t.Error("missing CVTCA instruction for string→[]byte")
	}
	if !hasCvtac {
		t.Error("missing CVTAC instruction for []byte→string")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("strconv: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileClosure(t *testing.T) {
	src := []byte(`package main

func makeAdder(x int) func(int) int {
	return func(y int) int {
		return x + y
	}
}

func main() {
	add5 := makeAdder(5)
	println(add5(10))
	println(add5(20))
}
`)
	c := New()
	m, err := c.CompileFile("closure.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have NEW for closure struct allocation
	hasNew := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.INEW {
			hasNew = true
		}
	}
	if !hasNew {
		t.Error("missing NEW instruction for closure struct")
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("closure: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileMap(t *testing.T) {
	src := []byte(`package main

func main() {
	m := make(map[string]int)
	m["hello"] = 10
	m["world"] = 20
	v1 := m["hello"]
	v2 := m["world"]
	println(v1, v2)
	m["hello"] = 30
	v3 := m["hello"]
	println(v3)
	v4, ok := m["missing"]
	println(v4, ok)
	delete(m, "hello")
	v5 := m["hello"]
	println(v5)
}
`)
	c := New()
	m, err := c.CompileFile("maps.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Count key map opcodes: NEW (map struct), NEWA (arrays), INDW (indexing),
	// BEQC (string key comparison), SLICELA (array copy)
	var newCount, newaCount, indwCount, beqcCount, slicelaCount int
	for _, inst := range m.Instructions {
		switch inst.Op {
		case dis.INEW:
			newCount++
		case dis.INEWA:
			newaCount++
		case dis.IINDW:
			indwCount++
		case dis.IBEQC:
			beqcCount++
		case dis.ISLICELA:
			slicelaCount++
		}
	}

	if newCount < 1 {
		t.Error("expected at least 1 NEW (map struct)")
	}
	if newaCount < 6 {
		t.Errorf("expected at least 6 NEWA (key+val arrays per insert), got %d", newaCount)
	}
	if beqcCount < 3 {
		t.Errorf("expected at least 3 BEQC (string key comparisons for updates+lookups), got %d", beqcCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("maps: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileSliceSubSlice(t *testing.T) {
	src := []byte(`package main

func main() {
	s := make([]int, 0)
	s = append(s, 10)
	s = append(s, 20)
	s = append(s, 30)
	s = append(s, 40)
	t := s[1:3]
	println(len(t))
	println(t[0])
	println(t[1])
	u := s[:2]
	println(len(u))
	v := s[2:]
	println(len(v))
}
`)
	c := New()
	m, err := c.CompileFile("subslice.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have SLICEA instructions for sub-slicing
	var sliceaCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.ISLICEA {
			sliceaCount++
		}
	}
	if sliceaCount < 3 {
		t.Errorf("expected >= 3 SLICEA instructions, got %d", sliceaCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("subslice: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStringIndex(t *testing.T) {
	src := []byte(`package main

func main() {
	s := "hello"
	println(s[0])
	println(s[4])
	t := s[1:4]
	println(t)
	u := s[:3]
	println(u)
}
`)
	c := New()
	m, err := c.CompileFile("stridx.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have INDC instructions for string indexing
	var indcCount, slicecCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.IINDC {
			indcCount++
		}
		if inst.Op == dis.ISLICEC {
			slicecCount++
		}
	}
	if indcCount < 2 {
		t.Errorf("expected >= 2 INDC instructions, got %d", indcCount)
	}
	if slicecCount < 2 {
		t.Errorf("expected >= 2 SLICEC instructions, got %d", slicecCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("stridx: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileCopyCap(t *testing.T) {
	src := []byte(`package main

func main() {
	s := make([]int, 0)
	s = append(s, 1)
	s = append(s, 2)
	s = append(s, 3)
	println(cap(s))
	dst := make([]int, 5)
	n := copy(dst, s)
	println(n)
	println(dst[0])
}
`)
	c := New()
	m, err := c.CompileFile("copycap.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have LENA for cap and SLICEA for copy's sub-slicing
	var lenaCount, sliceaCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.ILENA {
			lenaCount++
		}
		if inst.Op == dis.ISLICEA {
			sliceaCount++
		}
	}
	if lenaCount < 2 {
		t.Errorf("expected >= 2 LENA instructions (cap + copy lens), got %d", lenaCount)
	}
	if sliceaCount < 1 {
		t.Errorf("expected >= 1 SLICEA instruction (copy sub-slice), got %d", sliceaCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("copycap: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileMapRange(t *testing.T) {
	src := []byte(`package main

func main() {
	m := make(map[int]int)
	m[1] = 10
	m[2] = 20
	sum := 0
	for _, v := range m {
		sum = sum + v
	}
	println(sum)
}
`)
	c := New()
	m, err := c.CompileFile("maprange.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have LENA (for loading count via map struct access) and branching
	var lenaCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.ILENA {
			lenaCount++
		}
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("maprange: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStringRange(t *testing.T) {
	src := []byte(`package main

func main() {
	s := "hi"
	sum := 0
	for _, c := range s {
		sum = sum + int(c)
	}
	println(sum)
}
`)
	c := New()
	m, err := c.CompileFile("strrange.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have INDC for character access and LENC for length
	var indcCount, lencCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.IINDC {
			indcCount++
		}
		if inst.Op == dis.ILENC {
			lencCount++
		}
	}
	if indcCount < 1 {
		t.Errorf("expected >= 1 INDC instructions, got %d", indcCount)
	}
	if lencCount < 1 {
		t.Errorf("expected >= 1 LENC instructions, got %d", lencCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("strrange: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileDefer(t *testing.T) {
	src := []byte(`package main

func greet(s string) {
	println(s)
}

func main() {
	defer greet("third")
	defer greet("second")
	defer greet("first")
	println("hello")
}
`)
	c := New()
	m, err := c.CompileFile("defer.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// 4 CALL instructions: 3 deferred greet() + 1 regular greet (the one being deferred)
	// Actually: main has 3 deferred calls (emitted at RunDefers) and the block also prints.
	// The deferred calls use ICALL. Count them.
	var callCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICALL {
			callCount++
		}
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 CALL instructions (deferred calls), got %d", callCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("defer: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileStrconv(t *testing.T) {
	src := []byte(`package main

import "strconv"

func main() {
	s := strconv.Itoa(42)
	println(s)
	n, _ := strconv.Atoi("123")
	println(n)
}
`)
	c := New()
	m, err := c.CompileFile("strconv.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have CVTWC (int→string) for Itoa
	var cvtwcCount int
	var cvtcwCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICVTWC {
			cvtwcCount++
		}
		if inst.Op == dis.ICVTCW {
			cvtcwCount++
		}
	}
	if cvtwcCount < 1 {
		t.Errorf("expected >= 1 CVTWC (Itoa), got %d", cvtwcCount)
	}
	if cvtcwCount < 1 {
		t.Errorf("expected >= 1 CVTCW (Atoi), got %d", cvtcwCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("strconv: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileRuneToString(t *testing.T) {
	src := []byte(`package main

func toChar(n int) string {
	return string(rune(n))
}

func main() {
	println(toChar(65))
	println(toChar(104))
}
`)
	c := New()
	m, err := c.CompileFile("rune2str.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must have INSC instruction (rune→string conversion)
	// toChar is a single function compiled once, so only 1 INSC
	var inscCount int
	for _, inst := range m.Instructions {
		if inst.Op == dis.IINSC {
			inscCount++
		}
	}
	if inscCount < 1 {
		t.Errorf("expected >= 1 INSC (rune→string), got %d", inscCount)
	}

	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("rune2str: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileTypeAssert(t *testing.T) {
	src := []byte(`package main

func asInt(x interface{}) int {
	return x.(int)
}

func tryString(x interface{}) (string, bool) {
	s, ok := x.(string)
	return s, ok
}

func main() {
	println(asInt(42))
	s, ok := tryString("hello")
	if ok {
		println(s)
	}
}
`)
	c := New()
	m, err := c.CompileFile("typeassert.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must compile and round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("typeassert: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileInterface(t *testing.T) {
	src := []byte(`package main

type Stringer interface {
	String() string
}

type MyInt struct {
	val int
}

func (m MyInt) String() string {
	return "myint"
}

func printIt(s Stringer) {
	println(s.String())
}

func main() {
	x := MyInt{val: 42}
	printIt(x)
}
`)
	c := New()
	m, err := c.CompileFile("iface.go", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Must compile and round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("interface: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompileFmtSprintf(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	x := 42
	s1 := fmt.Sprintf("%d", x)
	println(s1)

	name := "world"
	s2 := fmt.Sprintf("hello %s", name)
	println(s2)

	s3 := fmt.Sprintf("no verbs here")
	println(s3)

	age := 30
	s4 := fmt.Sprintf("%s is %d", name, age)
	println(s4)

	fmt.Println("hello", x)
}
`
	c := New()
	m, err := c.CompileFile("fmtsprintf.go", []byte(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Check for CVTWC (int→string)
	hasCVTWC := false
	// Check for ADDC (string concat)
	hasADDC := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.ICVTWC {
			hasCVTWC = true
		}
		if inst.Op == dis.IADDC {
			hasADDC = true
		}
	}
	if !hasCVTWC {
		t.Error("expected CVTWC instruction for Sprintf with int verb")
	}
	if !hasADDC {
		t.Error("expected ADDC instruction for Sprintf with string concat")
	}

	// Must round-trip
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}

	t.Logf("fmtsprintf: compiled %d instructions, %d type descs, %d bytes",
		len(m.Instructions), len(m.TypeDescs), len(encoded))
}

func TestCompilePanicRecover(t *testing.T) {
	src := []byte(`package main

func safeDivide(a, b int) int {
	defer func() {
		if r := recover(); r != nil {
			println("recovered")
		}
	}()
	return a / b
}

func main() {
	println(safeDivide(10, 2))
	println(safeDivide(10, 0))
}
`)
	c := New()
	m, err := c.CompileFile("panic_recover.go", src)
	if err != nil {
		t.Fatalf("compile errors: %v", err)
	}

	// Must have HASEXCEPT flag
	if m.RuntimeFlags&dis.HASEXCEPT == 0 {
		t.Error("expected HASEXCEPT flag for program with recover")
	}

	// Must have at least one handler
	if len(m.Handlers) == 0 {
		t.Fatal("expected at least one exception handler")
	}
	h := m.Handlers[0]
	if h.WildPC < 0 {
		t.Error("expected valid wildcard PC in handler")
	}
	if h.EOffset <= 0 {
		t.Error("expected positive eoff in handler")
	}

	// Must have a zero-divide check (BNEW + RAISE pattern)
	hasRaise := false
	for _, inst := range m.Instructions {
		if inst.Op == dis.IRAISE {
			hasRaise = true
			break
		}
	}
	if !hasRaise {
		t.Error("expected IRAISE instruction for zero-divide check")
	}

	// Must round-trip encode/decode
	encoded, err := m.EncodeToBytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	m2, err := dis.Decode(encoded)
	if err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}
	if len(m2.Handlers) != len(m.Handlers) {
		t.Errorf("handler count mismatch: got %d, want %d", len(m2.Handlers), len(m.Handlers))
	}

	t.Logf("panic_recover: compiled %d instructions, %d handlers, %d bytes",
		len(m.Instructions), len(m.Handlers), len(encoded))
}
