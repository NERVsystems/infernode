package compiler

// stdlib_packages2.go — more standard library package stubs.

import (
	"go/constant"
	"go/token"
	"go/types"
)

func buildCryptoSHA1Package() *types.Package {
	pkg := types.NewPackage("crypto/sha1", "sha1")
	scope := pkg.Scope()

	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int], constant.MakeInt64(20)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BlockSize", types.Typ[types.Int], constant.MakeInt64(64)))

	// func Sum(data []byte) [20]byte — simplified as []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func New() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildCompressZlibPackage() *types.Package {
	pkg := types.NewPackage("compress/zlib", "zlib")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	readerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	readerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Resetter", nil),
		readerStruct, nil)
	scope.Insert(readerType.Obj())

	// func NewReader(r io.Reader) (io.ReadCloser, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewWriter(w io.Writer) *Writer — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	scope.Insert(types.NewConst(token.NoPos, pkg, "NoCompression", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BestSpeed", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BestCompression", types.Typ[types.Int], constant.MakeInt64(9)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultCompression", types.Typ[types.Int], constant.MakeInt64(-1)))

	pkg.MarkComplete()
	return pkg
}

func buildCompressBzip2Package() *types.Package {
	pkg := types.NewPackage("compress/bzip2", "bzip2")
	scope := pkg.Scope()

	// func NewReader(r io.Reader) io.Reader — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildCompressLzwPackage() *types.Package {
	pkg := types.NewPackage("compress/lzw", "lzw")
	scope := pkg.Scope()

	// Order type
	scope.Insert(types.NewConst(token.NoPos, pkg, "LSB", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MSB", types.Typ[types.Int], constant.MakeInt64(1)))

	// func NewReader(r io.Reader, order Order, litWidth int) io.ReadCloser — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "order", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "litWidth", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NewWriter(w io.Writer, order Order, litWidth int) io.WriteCloser — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "order", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "litWidth", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildHashFNVPackage() *types.Package {
	pkg := types.NewPackage("hash/fnv", "fnv")
	scope := pkg.Scope()

	// func New32() hash.Hash32 — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New32",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New32a",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New64",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New64a",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New128",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New128a",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildHashMaphashPackage() *types.Package {
	pkg := types.NewPackage("hash/maphash", "maphash")
	scope := pkg.Scope()

	hashStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "seed", types.Typ[types.Uint64], false),
	}, nil)
	hashType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Hash", nil),
		hashStruct, nil)
	scope.Insert(hashType.Obj())

	// func MakeSeed() Seed — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeSeed",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func Bytes(seed Seed, b []byte) uint64 — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Bytes",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "seed", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "b", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func String(seed Seed, s string) uint64 — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "seed", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildImageDrawPackage() *types.Package {
	pkg := types.NewPackage("image/draw", "draw")
	scope := pkg.Scope()

	// Op type
	scope.Insert(types.NewConst(token.NoPos, pkg, "Over", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Src", types.Typ[types.Int], constant.MakeInt64(1)))

	// func Draw(dst Image, r Rectangle, src Image, sp Point, op Op) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Draw",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "src", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "sp", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "op", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildImageGIFPackage() *types.Package {
	pkg := types.NewPackage("image/gif", "gif")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Decode(r io.Reader) (image.Image, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildExpvarPackage() *types.Package {
	pkg := types.NewPackage("expvar", "expvar")
	scope := pkg.Scope()

	// func NewInt(name string) *Int — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NewString(name string) *String — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Publish(name string, v Var) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Publish",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "v", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildLogSyslogPackage() *types.Package {
	pkg := types.NewPackage("log/syslog", "syslog")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(writerType.Obj())

	// func New(priority Priority, tag string) (*Writer, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "priority", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "tag", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(writerType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Priority constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_EMERG", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_ALERT", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_CRIT", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_ERR", types.Typ[types.Int], constant.MakeInt64(3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_WARNING", types.Typ[types.Int], constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_NOTICE", types.Typ[types.Int], constant.MakeInt64(5)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_INFO", types.Typ[types.Int], constant.MakeInt64(6)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_DEBUG", types.Typ[types.Int], constant.MakeInt64(7)))

	pkg.MarkComplete()
	return pkg
}

func buildIndexSuffixarrayPackage() *types.Package {
	pkg := types.NewPackage("index/suffixarray", "suffixarray")
	scope := pkg.Scope()

	indexStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	indexType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Index", nil),
		indexStruct, nil)
	scope.Insert(indexType.Obj())

	// func New(data []byte) *Index
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(indexType))),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoPrinterPackage() *types.Package {
	pkg := types.NewPackage("go/printer", "printer")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Fprint(output io.Writer, fset *token.FileSet, node any) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "output", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "fset", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "node", types.Universe.Lookup("any").Type())),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoBuildPackage() *types.Package {
	pkg := types.NewPackage("go/build", "build")
	scope := pkg.Scope()

	contextStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GOARCH", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOOS", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOROOT", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOPATH", types.Typ[types.String], false),
	}, nil)
	contextType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Context", nil),
		contextStruct, nil)
	scope.Insert(contextType.Obj())

	// var Default Context
	scope.Insert(types.NewVar(token.NoPos, pkg, "Default", contextType))

	pkg.MarkComplete()
	return pkg
}

func buildGoTypesPackage() *types.Package {
	pkg := types.NewPackage("go/types", "types")
	scope := pkg.Scope()

	// type Package struct
	pkgStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "path", types.Typ[types.String], false),
	}, nil)
	pkgType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Package", nil),
		pkgStruct, nil)
	scope.Insert(pkgType.Obj())

	// type Object interface — simplified
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Object", types.Typ[types.Int]))

	// type Type interface — simplified
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Type", types.Typ[types.Int]))

	pkg.MarkComplete()
	return pkg
}

func buildNetHTTPTestPackage() *types.Package {
	pkg := types.NewPackage("net/http/httptest", "httptest")
	scope := pkg.Scope()

	// type Server struct { URL string }
	serverStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "URL", types.Typ[types.String], false),
	}, nil)
	serverType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Server", nil),
		serverStruct, nil)
	scope.Insert(serverType.Obj())
	serverPtr := types.NewPointer(serverType)

	// func NewServer(handler http.Handler) *Server — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewServer",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "handler", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", serverPtr)),
			false)))

	// func NewRecorder() *ResponseRecorder — simplified
	recorderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Code", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Body", types.Typ[types.Int], false),
	}, nil)
	recorderType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ResponseRecorder", nil),
		recorderStruct, nil)
	scope.Insert(recorderType.Obj())

	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewRecorder",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(recorderType))),
			false)))

	// func NewRequest(method, target string, body io.Reader) *http.Request — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewRequest",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "method", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "target", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "body", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func (s *Server) Close()
	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil, nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildTestingFstestPackage() *types.Package {
	pkg := types.NewPackage("testing/fstest", "fstest")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func TestFS(fsys fs.FS, expected ...string) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TestFS",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fsys", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "expected", types.NewSlice(types.Typ[types.String]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			true)))

	pkg.MarkComplete()
	return pkg
}

func buildTestingIotestPackage() *types.Package {
	pkg := types.NewPackage("testing/iotest", "iotest")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func ErrReader(err error) io.Reader — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ErrReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func TestReader(r io.Reader, content []byte) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TestReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "content", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// debug/* packages — minimal stubs

func buildDebugElfPackage() *types.Package {
	pkg := types.NewPackage("debug/elf", "elf")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(fileType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugDwarfPackage() *types.Package {
	pkg := types.NewPackage("debug/dwarf", "dwarf")
	scope := pkg.Scope()

	dataStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	dataType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Data", nil),
		dataStruct, nil)
	scope.Insert(dataType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildDebugPEPackage() *types.Package {
	pkg := types.NewPackage("debug/pe", "pe")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(fileType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugMachoPackage() *types.Package {
	pkg := types.NewPackage("debug/macho", "macho")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(fileType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugGosymPackage() *types.Package {
	pkg := types.NewPackage("debug/gosym", "gosym")
	scope := pkg.Scope()

	tableStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	tableType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Table", nil),
		tableStruct, nil)
	scope.Insert(tableType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildDebugPlan9objPackage() *types.Package {
	pkg := types.NewPackage("debug/plan9obj", "plan9obj")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(fileType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}
