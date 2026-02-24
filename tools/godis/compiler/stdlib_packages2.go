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
	scope.Insert(types.NewConst(token.NoPos, pkg, "HuffmanOnly", types.Typ[types.Int], constant.MakeInt64(-2)))

	// func NewWriterLevel(w io.Writer, level int) (*Writer, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriterLevel",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "level", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewReaderDict(r io.Reader, dict []byte) (io.ReadCloser, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReaderDict",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "dict", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewWriterLevelDict(w io.Writer, level int, dict []byte) (*Writer, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriterLevelDict",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "level", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "dict", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

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

	// type Op int
	opType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Op", nil),
		types.Typ[types.Int], nil)
	scope.Insert(opType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "Over", opType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Src", opType, constant.MakeInt64(1)))

	// Image interface (opaque — extends image.Image with Set)
	imgIface := types.NewInterfaceType(nil, nil)
	imgIface.Complete()
	imgType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Image", nil),
		imgIface, nil)
	scope.Insert(imgType.Obj())

	// Drawer interface (opaque)
	drawerIface := types.NewInterfaceType(nil, nil)
	drawerIface.Complete()
	drawerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Drawer", nil),
		drawerIface, nil)
	scope.Insert(drawerType.Obj())

	// Quantizer interface (opaque)
	quantizerIface := types.NewInterfaceType(nil, nil)
	quantizerIface.Complete()
	quantizerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Quantizer", nil),
		quantizerIface, nil)
	scope.Insert(quantizerType.Obj())

	// var FloydSteinberg Drawer
	scope.Insert(types.NewVar(token.NoPos, pkg, "FloydSteinberg", drawerType))

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

	// func DrawMask(dst Image, r, src, sp, mask, mp, op) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DrawMask",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "src", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "sp", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "mask", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "mp", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "op", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildImageGIFPackage() *types.Package {
	pkg := types.NewPackage("image/gif", "gif")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type GIF struct
	gifStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Image", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Delay", types.NewSlice(types.Typ[types.Int]), false),
		types.NewField(token.NoPos, pkg, "LoopCount", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Disposal", types.NewSlice(types.Typ[types.Byte]), false),
		types.NewField(token.NoPos, pkg, "BackgroundIndex", types.Typ[types.Byte], false),
	}, nil)
	gifType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "GIF", nil),
		gifStruct, nil)
	scope.Insert(gifType.Obj())
	gifPtr := types.NewPointer(gifType)

	// func Decode(r io.Reader) (*image.Paletted, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func DecodeAll(r io.Reader) (*GIF, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DecodeAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", gifPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Encode(w io.Writer, m *image.Paletted, o *Options) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "m", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "o", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func EncodeAll(w io.Writer, g *GIF) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "EncodeAll",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "g", gifPtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Disposal constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "DisposalNone", types.Typ[types.Byte], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DisposalBackground", types.Typ[types.Byte], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DisposalPrevious", types.Typ[types.Byte], constant.MakeInt64(3)))

	pkg.MarkComplete()
	return pkg
}

func buildExpvarPackage() *types.Package {
	pkg := types.NewPackage("expvar", "expvar")
	scope := pkg.Scope()

	// Var interface (opaque)
	varIface := types.NewInterfaceType(nil, nil)
	varIface.Complete()
	varType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Var", nil),
		varIface, nil)
	scope.Insert(varType.Obj())

	// func NewInt(name string) *Int — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewInt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func NewFloat(name string) *Float — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFloat",
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

	// func NewMap(name string) *Map — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewMap",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Get(name string) Var
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Get",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", varType)),
			false)))

	// func Publish(name string, v Var) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Publish",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "v", varType)),
			nil, false)))

	// func Do(f func(KeyValue)) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Do",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Int])),
			nil, false)))

	// func Handler() http.Handler — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Handler",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// type KeyValue struct
	kvStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Key", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Value", varType, false),
	}, nil)
	kvType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "KeyValue", nil),
		kvStruct, nil)
	scope.Insert(kvType.Obj())

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

	// type Priority int
	priorityType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Priority", nil),
		types.Typ[types.Int], nil)
	scope.Insert(priorityType.Obj())

	// Priority constants - severity
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_EMERG", priorityType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_ALERT", priorityType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_CRIT", priorityType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_ERR", priorityType, constant.MakeInt64(3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_WARNING", priorityType, constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_NOTICE", priorityType, constant.MakeInt64(5)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_INFO", priorityType, constant.MakeInt64(6)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_DEBUG", priorityType, constant.MakeInt64(7)))

	// Priority constants - facility
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_KERN", priorityType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_USER", priorityType, constant.MakeInt64(8)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_MAIL", priorityType, constant.MakeInt64(16)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_DAEMON", priorityType, constant.MakeInt64(24)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_AUTH", priorityType, constant.MakeInt64(32)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_SYSLOG", priorityType, constant.MakeInt64(40)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_LOCAL0", priorityType, constant.MakeInt64(128)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "LOG_LOCAL7", priorityType, constant.MakeInt64(184)))

	// func Dial(network, raddr string, priority Priority, tag string) (*Writer, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dial",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "raddr", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "priority", priorityType),
				types.NewVar(token.NoPos, pkg, "tag", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(writerType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Writer methods
	writerPtr := types.NewPointer(writerType)

	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "w", writerPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "b", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "w", writerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	for _, name := range []string{"Emerg", "Alert", "Crit", "Err", "Warning", "Notice", "Info", "Debug"} {
		writerType.AddMethod(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(types.NewVar(token.NoPos, nil, "w", writerPtr),
				nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "m", types.Typ[types.String])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
				false)))
	}

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
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Server struct
	serverStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "URL", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Listener", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "TLS", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Config", types.Typ[types.Int], false),
	}, nil)
	serverType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Server", nil),
		serverStruct, nil)
	scope.Insert(serverType.Obj())
	serverPtr := types.NewPointer(serverType)

	// func NewServer(handler http.Handler) *Server
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewServer",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "handler", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", serverPtr)),
			false)))

	// func NewTLSServer(handler http.Handler) *Server
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewTLSServer",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "handler", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", serverPtr)),
			false)))

	// func NewUnstartedServer(handler http.Handler) *Server
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewUnstartedServer",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "handler", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", serverPtr)),
			false)))

	// Server methods
	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil, nil, false)))

	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "CloseClientConnections",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil, nil, false)))

	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Start",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil, nil, false)))

	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "StartTLS",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil, nil, false)))

	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Client",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Certificate",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", serverPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	// type ResponseRecorder struct
	recorderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Code", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "HeaderMap", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Body", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Flushed", types.Typ[types.Bool], false),
	}, nil)
	recorderType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ResponseRecorder", nil),
		recorderStruct, nil)
	scope.Insert(recorderType.Obj())
	recorderPtr := types.NewPointer(recorderType)

	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewRecorder",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", recorderPtr)),
			false)))

	// ResponseRecorder methods
	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Header",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))

	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "buf", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteString",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "str", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteHeader",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "code", types.Typ[types.Int])),
			nil, false)))

	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Flush",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil, nil, nil, false)))

	recorderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Result",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "rw", recorderPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
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

	// DefaultRemoteAddr constant
	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultRemoteAddr",
		types.Typ[types.String], constant.MakeString("1.2.3.4")))

	pkg.MarkComplete()
	return pkg
}

func buildTestingFstestPackage() *types.Package {
	pkg := types.NewPackage("testing/fstest", "fstest")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type MapFile struct
	mapFileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Data", types.NewSlice(types.Typ[types.Byte]), false),
		types.NewField(token.NoPos, pkg, "Mode", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "ModTime", types.Typ[types.Int64], false),
		types.NewField(token.NoPos, pkg, "Sys", types.NewInterfaceType(nil, nil), false),
	}, nil)
	mapFileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "MapFile", nil),
		mapFileStruct, nil)
	scope.Insert(mapFileType.Obj())

	// type MapFS map[string]*MapFile
	mapFSType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "MapFS", nil),
		types.NewMap(types.Typ[types.String], types.NewPointer(mapFileType)), nil)
	scope.Insert(mapFSType.Obj())

	// MapFS.Open(name string) (fs.File, error) — simplified
	mapFSType.AddMethod(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "fsys", mapFSType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// MapFS.ReadFile(name string) ([]byte, error)
	mapFSType.AddMethod(types.NewFunc(token.NoPos, pkg, "ReadFile",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "fsys", mapFSType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// MapFS.Stat(name string) (fs.FileInfo, error)
	mapFSType.AddMethod(types.NewFunc(token.NoPos, pkg, "Stat",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "fsys", mapFSType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// MapFS.ReadDir(name string) ([]fs.DirEntry, error)
	mapFSType.AddMethod(types.NewFunc(token.NoPos, pkg, "ReadDir",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "fsys", mapFSType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Int])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// MapFS.Sub(dir string) (fs.FS, error)
	mapFSType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sub",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "fsys", mapFSType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

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
	readerType := types.Typ[types.Int] // simplified io.Reader

	// func ErrReader(err error) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ErrReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "err", errType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func TestReader(r io.Reader, content []byte) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TestReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", readerType),
				types.NewVar(token.NoPos, pkg, "content", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func HalfReader(r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "HalfReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func DataErrReader(r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DataErrReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func OneByteReader(r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "OneByteReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func TimeoutReader(r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TimeoutReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func TruncateWriter(w io.Writer, n int64) io.Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TruncateWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", readerType),
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func NewReadLogger(prefix string, r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReadLogger",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "r", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// func NewWriteLogger(prefix string, w io.Writer) io.Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriteLogger",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "prefix", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "w", readerType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", readerType)),
			false)))

	// var ErrTimeout error
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrTimeout", errType))

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

func buildSyncErrgroupPackage() *types.Package {
	pkg := types.NewPackage("sync/errgroup", "errgroup")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Group struct
	groupType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Group", nil), types.NewStruct(nil, nil), nil)
	groupPtr := types.NewPointer(groupType)
	scope.Insert(groupType.Obj())

	groupRecv := types.NewVar(token.NoPos, pkg, "g", groupPtr)

	// func (g *Group) Go(f func() error)
	groupType.AddMethod(types.NewFunc(token.NoPos, pkg, "Go",
		types.NewSignatureType(groupRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f",
				types.NewSignatureType(nil, nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
					false))),
			nil, false)))

	// func (g *Group) Wait() error
	groupType.AddMethod(types.NewFunc(token.NoPos, pkg, "Wait",
		types.NewSignatureType(groupRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (g *Group) SetLimit(n int)
	groupType.AddMethod(types.NewFunc(token.NoPos, pkg, "SetLimit",
		types.NewSignatureType(groupRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			nil, false)))

	// func (g *Group) TryGo(f func() error) bool
	groupType.AddMethod(types.NewFunc(token.NoPos, pkg, "TryGo",
		types.NewSignatureType(groupRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f",
				types.NewSignatureType(nil, nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
					false))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
			false)))

	// func WithContext(ctx context.Context) (*Group, context.Context)
	ctxType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WithContext",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ctx", ctxType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", groupPtr),
				types.NewVar(token.NoPos, pkg, "", ctxType)),
			false)))

	pkg.MarkComplete()
	return pkg
}
