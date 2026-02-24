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
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// Resetter interface
	resetterIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Reset",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "r", types.Typ[types.Int]),
					types.NewVar(token.NoPos, nil, "dict", byteSlice)),
				types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
				false)),
	}, nil)
	resetterIface.Complete()
	resetterType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Resetter", nil),
		resetterIface, nil)
	scope.Insert(resetterType.Obj())

	// type Writer struct
	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(writerType.Obj())
	writerPtr := types.NewPointer(writerType)

	writerRecv := types.NewVar(token.NoPos, nil, "z", writerPtr)
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(writerRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "err", errType)),
			false)))
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(writerRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Flush",
		types.NewSignatureType(writerRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(writerRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "w", types.Typ[types.Int])),
			nil, false)))

	// func NewReader(r io.Reader) (io.ReadCloser, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewWriter(w io.Writer) *Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", writerPtr)),
			false)))

	scope.Insert(types.NewConst(token.NoPos, pkg, "NoCompression", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BestSpeed", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BestCompression", types.Typ[types.Int], constant.MakeInt64(9)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultCompression", types.Typ[types.Int], constant.MakeInt64(-1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "HuffmanOnly", types.Typ[types.Int], constant.MakeInt64(-2)))

	// func NewWriterLevel(w io.Writer, level int) (*Writer, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriterLevel",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "level", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", writerPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewReaderDict(r io.Reader, dict []byte) (io.ReadCloser, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReaderDict",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "dict", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewWriterLevelDict(w io.Writer, level int, dict []byte) (*Writer, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriterLevelDict",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "level", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "dict", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", writerPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// var ErrChecksum, ErrDictionary, ErrHeader error
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrChecksum", errType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrDictionary", errType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrHeader", errType))

	pkg.MarkComplete()
	return pkg
}

func buildCompressBzip2Package() *types.Package {
	pkg := types.NewPackage("compress/bzip2", "bzip2")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func NewReader(r io.Reader) io.Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// type StructuralError string
	structuralErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "StructuralError", nil),
		types.Typ[types.String], nil)
	structuralErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", structuralErrType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(structuralErrType.Obj())
	_ = errType

	pkg.MarkComplete()
	return pkg
}

func buildCompressLzwPackage() *types.Package {
	pkg := types.NewPackage("compress/lzw", "lzw")
	scope := pkg.Scope()

	// type Order int
	orderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Order", nil), types.Typ[types.Int], nil)
	scope.Insert(orderType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "LSB", orderType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MSB", orderType, constant.MakeInt64(1)))

	// func NewReader(r io.Reader, order Order, litWidth int) io.ReadCloser
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "order", orderType),
				types.NewVar(token.NoPos, pkg, "litWidth", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// func NewWriter(w io.Writer, order Order, litWidth int) io.WriteCloser
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "order", orderType),
				types.NewVar(token.NoPos, pkg, "litWidth", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildHashFNVPackage() *types.Package {
	pkg := types.NewPackage("hash/fnv", "fnv")
	scope := pkg.Scope()

	// All New* functions return hash.Hash (simplified as interface)
	hashIface := types.NewInterfaceType(nil, nil)
	for _, name := range []string{"New32", "New32a", "New64", "New64a", "New128", "New128a"} {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", hashIface)),
				false)))
	}

	pkg.MarkComplete()
	return pkg
}

func buildHashMaphashPackage() *types.Package {
	pkg := types.NewPackage("hash/maphash", "maphash")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Seed struct (opaque)
	seedType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Seed", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(seedType.Obj())

	// func MakeSeed() Seed
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeSeed",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", seedType)),
			false)))

	// type Hash struct (opaque)
	hashType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Hash", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(hashType.Obj())
	hashPtr := types.NewPointer(hashType)
	hashRecv := types.NewVar(token.NoPos, nil, "h", hashPtr)

	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(hashRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "b", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteByte",
		types.NewSignatureType(hashRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "b", types.Typ[types.Byte])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteString",
		types.NewSignatureType(hashRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sum64",
		types.NewSignatureType(hashRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Uint64])), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Seed",
		types.NewSignatureType(hashRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", seedType)), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "SetSeed",
		types.NewSignatureType(hashRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "seed", seedType)),
			nil, false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(hashRecv, nil, nil, nil, nil, false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Size",
		types.NewSignatureType(hashRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "BlockSize",
		types.NewSignatureType(hashRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sum",
		types.NewSignatureType(hashRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "b", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice)), false)))

	// func Bytes(seed Seed, b []byte) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Bytes",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "seed", seedType),
				types.NewVar(token.NoPos, pkg, "b", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])),
			false)))

	// func String(seed Seed, s string) uint64
	scope.Insert(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "seed", seedType),
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
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Index struct (opaque)
	indexType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Index", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(indexType.Obj())
	indexPtr := types.NewPointer(indexType)
	indexRecv := types.NewVar(token.NoPos, nil, "x", indexPtr)

	// func New(data []byte) *Index
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", indexPtr)),
			false)))

	// Index.Bytes() int
	indexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Bytes",
		types.NewSignatureType(indexRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))

	// Index.Lookup(s []byte, n int) []int
	indexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(indexRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "s", byteSlice),
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Int]))), false)))

	// Index.FindAllIndex(r *regexp.Regexp, n int) [][]int — simplified
	indexType.AddMethod(types.NewFunc(token.NoPos, pkg, "FindAllIndex",
		types.NewSignatureType(indexRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "r", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewSlice(types.NewSlice(types.Typ[types.Int])))), false)))

	// Index.Read/Write
	indexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(indexRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	indexType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(indexRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoPrinterPackage() *types.Package {
	pkg := types.NewPackage("go/printer", "printer")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anyType := types.Universe.Lookup("any").Type()

	// type Mode uint
	modeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Mode", nil), types.Typ[types.Uint], nil)
	scope.Insert(modeType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "RawFormat", modeType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TabIndent", modeType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "UseSpaces", modeType, constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SourcePos", modeType, constant.MakeInt64(8)))

	// type Config struct { Mode Mode; Tabwidth int; Indent int }
	configStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Mode", modeType, false),
		types.NewField(token.NoPos, pkg, "Tabwidth", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Indent", types.Typ[types.Int], false),
	}, nil)
	configType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Config", nil), configStruct, nil)
	scope.Insert(configType.Obj())
	configPtr := types.NewPointer(configType)

	// Config.Fprint(output io.Writer, fset *token.FileSet, node any) error
	configType.AddMethod(types.NewFunc(token.NoPos, pkg, "Fprint",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "cfg", configPtr), nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "output", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "node", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func Fprint(output io.Writer, fset *token.FileSet, node any) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "output", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "node", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoBuildPackage() *types.Package {
	pkg := types.NewPackage("go/build", "build")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Package struct
	buildPkgStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Dir", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "ImportComment", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Doc", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "ImportPath", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Root", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "SrcRoot", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "PkgRoot", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "BinDir", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Goroot", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "PkgObj", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GoFiles", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "CgoFiles", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "IgnoredGoFiles", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "TestGoFiles", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "XTestGoFiles", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "Imports", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "TestImports", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "XTestImports", types.NewSlice(types.Typ[types.String]), false),
	}, nil)
	buildPkgType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Package", nil), buildPkgStruct, nil)
	scope.Insert(buildPkgType.Obj())

	// type Context struct
	contextStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GOARCH", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOOS", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOROOT", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GOPATH", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "CgoEnabled", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "Compiler", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "BuildTags", types.NewSlice(types.Typ[types.String]), false),
	}, nil)
	contextType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Context", nil), contextStruct, nil)
	scope.Insert(contextType.Obj())
	contextPtr := types.NewPointer(contextType)
	contextRecv := types.NewVar(token.NoPos, nil, "ctxt", contextPtr)

	// var Default Context
	scope.Insert(types.NewVar(token.NoPos, pkg, "Default", contextType))

	// Context.ImportDir(dir string, mode ImportMode) (*Package, error)
	importModeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ImportMode", nil), types.Typ[types.Uint], nil)
	scope.Insert(importModeType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "FindOnly", importModeType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "AllowBinary", importModeType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ImportComment", importModeType, constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "IgnoreVendor", importModeType, constant.MakeInt64(8)))

	contextType.AddMethod(types.NewFunc(token.NoPos, pkg, "Import",
		types.NewSignatureType(contextRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "srcDir", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "mode", importModeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewPointer(buildPkgType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	contextType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportDir",
		types.NewSignatureType(contextRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "dir", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "mode", importModeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewPointer(buildPkgType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func Import(path, srcDir string, mode ImportMode) (*Package, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Import",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "srcDir", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "mode", importModeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(buildPkgType)),
				types.NewVar(token.NoPos, pkg, "", errType)), false)))

	// func ImportDir(dir string, mode ImportMode) (*Package, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ImportDir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "mode", importModeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(buildPkgType)),
				types.NewVar(token.NoPos, pkg, "", errType)), false)))

	// type NoGoError struct { Dir string }
	noGoErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Dir", types.Typ[types.String], false),
	}, nil)
	noGoErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "NoGoError", nil), noGoErrStruct, nil)
	noGoErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", types.NewPointer(noGoErrType)), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(noGoErrType.Obj())

	// type MultiplePackageError struct
	multiPkgErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Dir", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Packages", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "Files", types.NewSlice(types.Typ[types.String]), false),
	}, nil)
	multiPkgErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "MultiplePackageError", nil), multiPkgErrStruct, nil)
	multiPkgErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", types.NewPointer(multiPkgErrType)), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(multiPkgErrType.Obj())

	// func IsLocalImport(path string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsLocalImport",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoTypesPackage() *types.Package {
	pkg := types.NewPackage("go/types", "types")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Type interface { Underlying() Type; String() string }
	typeIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Underlying",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)),
		types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
	}, nil)
	typeIface.Complete()
	typeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Type", nil), typeIface, nil)
	scope.Insert(typeType.Obj())

	// type Object interface { Name() string; Type() Type; Pos() token.Pos; Id() string; Parent() *Scope; Exported() bool; Pkg() *Package; String() string }
	objectIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Name",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
		types.NewFunc(token.NoPos, pkg, "Type",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", typeType)), false)),
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)),
		types.NewFunc(token.NoPos, pkg, "Id",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
		types.NewFunc(token.NoPos, pkg, "Exported",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)),
		types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
	}, nil)
	objectIface.Complete()
	objectType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Object", nil), objectIface, nil)
	scope.Insert(objectType.Obj())

	// type Package struct (opaque)
	pkgType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Package", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(pkgType.Obj())
	pkgPtr := types.NewPointer(pkgType)
	pkgRecv := types.NewVar(token.NoPos, nil, "pkg", pkgPtr)
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Path",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Name",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Scope",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Imports",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewSlice(pkgPtr))), false)))
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "Complete",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	pkgType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(pkgRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// type Info struct
	infoStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Types", types.NewMap(types.NewInterfaceType(nil, nil), types.NewInterfaceType(nil, nil)), false),
		types.NewField(token.NoPos, pkg, "Defs", types.NewMap(types.NewInterfaceType(nil, nil), objectType), false),
		types.NewField(token.NoPos, pkg, "Uses", types.NewMap(types.NewInterfaceType(nil, nil), objectType), false),
	}, nil)
	infoType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Info", nil), infoStruct, nil)
	scope.Insert(infoType.Obj())

	// type Config struct
	configStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GoVersion", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Error", types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "err", errType)), nil, false), false),
		types.NewField(token.NoPos, pkg, "Importer", types.NewInterfaceType(nil, nil), false),
	}, nil)
	configType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Config", nil), configStruct, nil)
	scope.Insert(configType.Obj())
	configPtr := types.NewPointer(configType)
	configType.AddMethod(types.NewFunc(token.NoPos, pkg, "Check",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "conf", configPtr), nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "files", types.NewSlice(types.NewInterfaceType(nil, nil))),
				types.NewVar(token.NoPos, nil, "info", types.NewPointer(infoType))),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", pkgPtr),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// type Sizes interface { Alignof(T Type) int64; Offsetsof(fields []*Var) []int64; Sizeof(T Type) int64 }
	sizesIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Alignof",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "T", typeType)),
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64])), false)),
		types.NewFunc(token.NoPos, pkg, "Sizeof",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "T", typeType)),
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64])), false)),
	}, nil)
	sizesIface.Complete()
	sizesType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Sizes", nil), sizesIface, nil)
	scope.Insert(sizesType.Obj())

	// func SizesFor(compiler, arch string) Sizes
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SizesFor",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "compiler", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "arch", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", sizesType)), false)))

	// type Error struct { Fset *token.FileSet; Pos token.Pos; Msg string; Soft bool }
	typesErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Fset", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Pos", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Msg", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Soft", types.Typ[types.Bool], false),
	}, nil)
	typesErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Error", nil), typesErrStruct, nil)
	typesErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "err", typesErrType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(typesErrType.Obj())

	// type Importer interface { Import(path string) (*Package, error) }
	importerIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Import",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "path", types.Typ[types.String])),
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "", pkgPtr),
					types.NewVar(token.NoPos, nil, "", errType)), false)),
	}, nil)
	importerIface.Complete()
	importerType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Importer", nil), importerIface, nil)
	scope.Insert(importerType.Obj())

	// func ExprString(x ast.Expr) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ExprString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))

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
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// Type aliases for ELF header fields
	for _, name := range []string{"Class", "Data", "OSABI", "Type", "Machine"} {
		t := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), types.Typ[types.Int], nil)
		t.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(types.NewVar(token.NoPos, nil, "i", t), nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
		scope.Insert(t.Obj())
	}

	// Section type enums
	sectionTypeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SectionType", nil), types.Typ[types.Uint32], nil)
	scope.Insert(sectionTypeType.Obj())
	sectionFlagType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SectionFlag", nil), types.Typ[types.Uint32], nil)
	scope.Insert(sectionFlagType.Obj())
	progTypeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ProgType", nil), types.Typ[types.Int], nil)
	scope.Insert(progTypeType.Obj())
	progFlagType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ProgFlag", nil), types.Typ[types.Uint32], nil)
	scope.Insert(progFlagType.Obj())
	symBindType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SymBind", nil), types.Typ[types.Int], nil)
	scope.Insert(symBindType.Obj())
	symTypeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SymType", nil), types.Typ[types.Int], nil)
	scope.Insert(symTypeType.Obj())
	symVisType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SymVis", nil), types.Typ[types.Int], nil)
	scope.Insert(symVisType.Obj())

	// type SectionHeader struct
	sectionHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Type", sectionTypeType, false),
		types.NewField(token.NoPos, pkg, "Flags", sectionFlagType, false),
		types.NewField(token.NoPos, pkg, "Addr", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Size", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Link", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Info", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Addralign", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Entsize", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "FileSize", types.Typ[types.Uint64], false),
	}, nil)
	sectionHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SectionHeader", nil), sectionHeaderStruct, nil)
	scope.Insert(sectionHeaderType.Obj())

	// type Section struct { SectionHeader; ... }
	sectionStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "SectionHeader", sectionHeaderType, true),
	}, nil)
	sectionType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Section", nil), sectionStruct, nil)
	scope.Insert(sectionType.Obj())
	sectionPtr := types.NewPointer(sectionType)
	sectionRecv := types.NewVar(token.NoPos, nil, "s", sectionPtr)
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Data",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice), types.NewVar(token.NoPos, nil, "", errType)), false)))
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))

	// type Symbol struct
	symbolStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Info", types.Typ[types.Byte], false),
		types.NewField(token.NoPos, pkg, "Other", types.Typ[types.Byte], false),
		types.NewField(token.NoPos, pkg, "Section", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Size", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Version", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Library", types.Typ[types.String], false),
	}, nil)
	symbolType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Symbol", nil), symbolStruct, nil)
	scope.Insert(symbolType.Obj())

	// type Prog struct
	progHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Type", progTypeType, false),
		types.NewField(token.NoPos, pkg, "Flags", progFlagType, false),
		types.NewField(token.NoPos, pkg, "Off", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Vaddr", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Paddr", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Filesz", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Memsz", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Align", types.Typ[types.Uint64], false),
	}, nil)
	progHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ProgHeader", nil), progHeaderStruct, nil)
	scope.Insert(progHeaderType.Obj())

	progStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ProgHeader", progHeaderType, true),
	}, nil)
	progType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Prog", nil), progStruct, nil)
	scope.Insert(progType.Obj())

	// type FileHeader struct
	fileHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Class", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Data", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Version", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "OSABI", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "ABIVersion", types.Typ[types.Byte], false),
		types.NewField(token.NoPos, pkg, "Type", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Machine", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Entry", types.Typ[types.Uint64], false),
	}, nil)
	fileHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FileHeader", nil), fileHeaderStruct, nil)
	scope.Insert(fileHeaderType.Obj())

	// type File struct
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "FileHeader", fileHeaderType, true),
		types.NewField(token.NoPos, pkg, "Sections", types.NewSlice(sectionPtr), false),
		types.NewField(token.NoPos, pkg, "Progs", types.NewSlice(types.NewPointer(progType)), false),
	}, nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "File", nil), fileStruct, nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	fileRecv := types.NewVar(token.NoPos, nil, "f", filePtr)
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Section",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", sectionPtr)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Symbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(symbolType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "DynamicSymbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(symbolType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedSymbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.NewInterfaceType(nil, nil))),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedLibraries",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "DWARF",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func Open(name string) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewFile(r io.ReaderAt) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Some ELF constants
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"ELFCLASS32", 1}, {"ELFCLASS64", 2},
		{"ELFDATA2LSB", 1}, {"ELFDATA2MSB", 2},
		{"ET_NONE", 0}, {"ET_REL", 1}, {"ET_EXEC", 2}, {"ET_DYN", 3}, {"ET_CORE", 4},
		{"EM_386", 3}, {"EM_ARM", 40}, {"EM_X86_64", 62}, {"EM_AARCH64", 183}, {"EM_RISCV", 243},
		{"SHT_NULL", 0}, {"SHT_PROGBITS", 1}, {"SHT_SYMTAB", 2}, {"SHT_STRTAB", 3},
		{"SHT_NOTE", 7}, {"SHT_NOBITS", 8}, {"SHT_DYNSYM", 11},
		{"SHF_WRITE", 1}, {"SHF_ALLOC", 2}, {"SHF_EXECINSTR", 4},
		{"PT_NULL", 0}, {"PT_LOAD", 1}, {"PT_DYNAMIC", 2}, {"PT_INTERP", 3}, {"PT_NOTE", 4},
		{"STB_LOCAL", 0}, {"STB_GLOBAL", 1}, {"STB_WEAK", 2},
		{"STT_NOTYPE", 0}, {"STT_OBJECT", 1}, {"STT_FUNC", 2}, {"STT_SECTION", 3}, {"STT_FILE", 4},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, types.Typ[types.Int], constant.MakeInt64(c.val)))
	}

	// FormatError
	formatErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FormatError", nil),
		types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "Msg", types.Typ[types.String], false),
		}, nil), nil)
	formatErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", types.NewPointer(formatErrType)), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(formatErrType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildDebugDwarfPackage() *types.Package {
	pkg := types.NewPackage("debug/dwarf", "dwarf")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Tag uint32
	tagType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Tag", nil), types.Typ[types.Uint32], nil)
	scope.Insert(tagType.Obj())
	tagType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "t", tagType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// type Attr uint32
	attrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Attr", nil), types.Typ[types.Uint32], nil)
	scope.Insert(attrType.Obj())
	attrType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "a", attrType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// Some common tag/attr constants
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"TagCompileUnit", 0x11}, {"TagSubprogram", 0x2e}, {"TagVariable", 0x34},
		{"TagFormalParameter", 0x05}, {"TagMember", 0x0d}, {"TagBaseType", 0x24},
		{"TagStructType", 0x13}, {"TagTypedef", 0x16}, {"TagPointerType", 0x0f},
		{"AttrName", 0x03}, {"AttrType", 0x49}, {"AttrByteSize", 0x0b},
		{"AttrLocation", 0x02}, {"AttrLowpc", 0x11}, {"AttrHighpc", 0x12},
		{"AttrLanguage", 0x13}, {"AttrCompDir", 0x1b},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, types.Typ[types.Int], constant.MakeInt64(c.val)))
	}

	// type Offset uint32
	offsetType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Offset", nil), types.Typ[types.Uint32], nil)
	scope.Insert(offsetType.Obj())

	// type Field struct { Attr Attr; Val interface{}; Class Class }
	fieldStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Attr", attrType, false),
		types.NewField(token.NoPos, pkg, "Val", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Class", types.Typ[types.Int], false),
	}, nil)
	fieldType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Field", nil), fieldStruct, nil)
	scope.Insert(fieldType.Obj())

	// type Entry struct
	entryStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Offset", offsetType, false),
		types.NewField(token.NoPos, pkg, "Tag", tagType, false),
		types.NewField(token.NoPos, pkg, "Children", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "Field", types.NewSlice(fieldType), false),
	}, nil)
	entryType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Entry", nil), entryStruct, nil)
	scope.Insert(entryType.Obj())
	entryPtr := types.NewPointer(entryType)
	entryRecv := types.NewVar(token.NoPos, nil, "e", entryPtr)
	entryType.AddMethod(types.NewFunc(token.NoPos, pkg, "Val",
		types.NewSignatureType(entryRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "a", attrType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))
	entryType.AddMethod(types.NewFunc(token.NoPos, pkg, "AttrField",
		types.NewSignatureType(entryRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "a", attrType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(fieldType))), false)))

	// type Reader struct (opaque)
	readerType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Reader", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(readerType.Obj())
	readerPtr := types.NewPointer(readerType)
	readerRecv := types.NewVar(token.NoPos, nil, "r", readerPtr)
	readerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(readerRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", entryPtr), types.NewVar(token.NoPos, nil, "", errType)), false)))
	readerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Seek",
		types.NewSignatureType(readerRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "off", offsetType)), nil, false)))
	readerType.AddMethod(types.NewFunc(token.NoPos, pkg, "SkipChildren",
		types.NewSignatureType(readerRecv, nil, nil, nil, nil, false)))

	// type LineReader struct (opaque)
	lineReaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "LineReader", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(lineReaderType.Obj())

	// type LineEntry struct
	lineEntryStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Address", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "File", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Line", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Column", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "IsStmt", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "EndSequence", types.Typ[types.Bool], false),
	}, nil)
	lineEntryType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "LineEntry", nil), lineEntryStruct, nil)
	scope.Insert(lineEntryType.Obj())

	lineReaderPtr := types.NewPointer(lineReaderType)
	lineReaderRecv := types.NewVar(token.NoPos, nil, "r", lineReaderPtr)
	lineReaderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(lineReaderRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "entry", types.NewPointer(lineEntryType))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	lineReaderType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(lineReaderRecv, nil, nil, nil, nil, false)))

	// type Type interface (simplified)
	typeIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Common",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)),
		types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
		types.NewFunc(token.NoPos, pkg, "Size",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64])), false)),
	}, nil)
	typeIface.Complete()
	dwarfTypeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Type", nil), typeIface, nil)
	scope.Insert(dwarfTypeType.Obj())

	// type Data struct (opaque)
	dataType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Data", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(dataType.Obj())
	dataPtr := types.NewPointer(dataType)
	dataRecv := types.NewVar(token.NoPos, nil, "d", dataPtr)
	dataType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reader",
		types.NewSignatureType(dataRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", readerPtr)), false)))
	dataType.AddMethod(types.NewFunc(token.NoPos, pkg, "Type",
		types.NewSignatureType(dataRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "off", offsetType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", dwarfTypeType), types.NewVar(token.NoPos, nil, "", errType)), false)))
	dataType.AddMethod(types.NewFunc(token.NoPos, pkg, "LineReader",
		types.NewSignatureType(dataRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "cu", entryPtr)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", lineReaderPtr), types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func New(abbrev, aranges, frame, info, line, pubnames, ranges, str []byte) (*Data, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "abbrev", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "aranges", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "frame", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "info", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "line", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "pubnames", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "ranges", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "str", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", dataPtr), types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugPEPackage() *types.Package {
	pkg := types.NewPackage("debug/pe", "pe")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type FileHeader struct
	fileHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Machine", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "NumberOfSections", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "TimeDateStamp", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "PointerToSymbolTable", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "NumberOfSymbols", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "SizeOfOptionalHeader", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "Characteristics", types.Typ[types.Uint16], false),
	}, nil)
	fileHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FileHeader", nil), fileHeaderStruct, nil)
	scope.Insert(fileHeaderType.Obj())

	// type SectionHeader32 struct
	sectionHeader32Struct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.NewArray(types.Typ[types.Uint8], 8), false),
		types.NewField(token.NoPos, pkg, "VirtualSize", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "VirtualAddress", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "SizeOfRawData", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "PointerToRawData", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Characteristics", types.Typ[types.Uint32], false),
	}, nil)
	sectionHeader32Type := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "SectionHeader32", nil), sectionHeader32Struct, nil)
	scope.Insert(sectionHeader32Type.Obj())

	// type Section struct
	sectionStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "VirtualSize", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "VirtualAddress", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Size", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Characteristics", types.Typ[types.Uint32], false),
	}, nil)
	sectionType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Section", nil), sectionStruct, nil)
	scope.Insert(sectionType.Obj())
	sectionPtr := types.NewPointer(sectionType)
	sectionRecv := types.NewVar(token.NoPos, nil, "s", sectionPtr)
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Data",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))

	// type Symbol struct
	symbolStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "SectionNumber", types.Typ[types.Int16], false),
		types.NewField(token.NoPos, pkg, "Type", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "StorageClass", types.Typ[types.Uint8], false),
	}, nil)
	symbolType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Symbol", nil), symbolStruct, nil)
	scope.Insert(symbolType.Obj())

	// type File struct
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "FileHeader", fileHeaderType, true),
		types.NewField(token.NoPos, pkg, "OptionalHeader", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Sections", types.NewSlice(sectionPtr), false),
		types.NewField(token.NoPos, pkg, "Symbols", types.NewSlice(types.NewPointer(symbolType)), false),
	}, nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "File", nil), fileStruct, nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	fileRecv := types.NewVar(token.NoPos, nil, "f", filePtr)
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Section",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", sectionPtr)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "DWARF",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedSymbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedLibraries",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func Open(name string) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewFile(r io.ReaderAt) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Machine constants
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"IMAGE_FILE_MACHINE_UNKNOWN", 0}, {"IMAGE_FILE_MACHINE_AM33", 0x1d3},
		{"IMAGE_FILE_MACHINE_AMD64", 0x8664}, {"IMAGE_FILE_MACHINE_ARM", 0x1c0},
		{"IMAGE_FILE_MACHINE_ARM64", 0xaa64}, {"IMAGE_FILE_MACHINE_I386", 0x14c},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, types.Typ[types.Uint16], constant.MakeInt64(c.val)))
	}

	pkg.MarkComplete()
	return pkg
}

func buildDebugMachoPackage() *types.Package {
	pkg := types.NewPackage("debug/macho", "macho")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Cpu uint32
	cpuType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Cpu", nil), types.Typ[types.Uint32], nil)
	scope.Insert(cpuType.Obj())
	cpuType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "i", cpuType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"Cpu386", 7}, {"CpuAmd64", 0x01000007}, {"CpuArm", 12}, {"CpuArm64", 0x0100000c}, {"CpuPpc", 18}, {"CpuPpc64", 0x01000012},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, cpuType, constant.MakeInt64(c.val)))
	}

	// type Type uint32
	typeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Type", nil), types.Typ[types.Uint32], nil)
	scope.Insert(typeType.Obj())
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"TypeObj", 1}, {"TypeExec", 2}, {"TypeDylib", 6}, {"TypeBundle", 8},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, typeType, constant.MakeInt64(c.val)))
	}

	// type Section struct
	sectionStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Seg", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Addr", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Size", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Align", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Reloff", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Nreloc", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Flags", types.Typ[types.Uint32], false),
	}, nil)
	sectionType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Section", nil), sectionStruct, nil)
	scope.Insert(sectionType.Obj())
	sectionPtr := types.NewPointer(sectionType)
	sectionRecv := types.NewVar(token.NoPos, nil, "s", sectionPtr)
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Data",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))

	// type Symbol struct
	symbolStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Type", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "Sect", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "Desc", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.Uint64], false),
	}, nil)
	symbolType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Symbol", nil), symbolStruct, nil)
	scope.Insert(symbolType.Obj())

	// type Segment struct
	segmentStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Addr", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Memsz", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Filesz", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Maxprot", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Prot", types.Typ[types.Uint32], false),
	}, nil)
	segmentType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Segment", nil), segmentStruct, nil)
	scope.Insert(segmentType.Obj())

	// type FileHeader struct
	fileHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Magic", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Cpu", cpuType, false),
		types.NewField(token.NoPos, pkg, "SubCpu", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Type", typeType, false),
		types.NewField(token.NoPos, pkg, "Flags", types.Typ[types.Uint32], false),
	}, nil)
	fileHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FileHeader", nil), fileHeaderStruct, nil)
	scope.Insert(fileHeaderType.Obj())

	// type File struct
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "FileHeader", fileHeaderType, true),
		types.NewField(token.NoPos, pkg, "Sections", types.NewSlice(sectionPtr), false),
		types.NewField(token.NoPos, pkg, "Symtab", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Loads", types.NewSlice(types.NewInterfaceType(nil, nil)), false),
	}, nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "File", nil), fileStruct, nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	fileRecv := types.NewVar(token.NoPos, nil, "f", filePtr)
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Section",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", sectionPtr)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Segment",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(segmentType))), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "DWARF",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedSymbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "ImportedLibraries",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func Open(name string) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewFile(r io.ReaderAt) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Magic constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "Magic32", types.Typ[types.Uint32], constant.MakeUint64(0xfeedface)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Magic64", types.Typ[types.Uint32], constant.MakeUint64(0xfeedfacf)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MagicFat", types.Typ[types.Uint32], constant.MakeUint64(0xcafebabe)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugGosymPackage() *types.Package {
	pkg := types.NewPackage("debug/gosym", "gosym")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Sym struct { Value uint64; Type byte; Name string; GoType uint64; Func *Func }
	// Forward declare Func
	funcType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Func", nil), types.NewStruct(nil, nil), nil)

	symStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Type", types.Typ[types.Byte], false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "GoType", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Func", types.NewPointer(funcType), false),
	}, nil)
	symType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Sym", nil), symStruct, nil)
	scope.Insert(symType.Obj())

	// type Obj struct { Funcs []Func; Paths []Sym }
	objStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Funcs", types.NewSlice(funcType), false),
	}, nil)
	objType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Obj", nil), objStruct, nil)
	scope.Insert(objType.Obj())

	// Now set up Func struct properly
	funcStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Sym", symType, true),
		types.NewField(token.NoPos, pkg, "End", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Obj", types.NewPointer(objType), false),
	}, nil)
	funcType.SetUnderlying(funcStruct)
	scope.Insert(funcType.Obj())

	// type Table struct
	tableStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Syms", types.NewSlice(symType), false),
		types.NewField(token.NoPos, pkg, "Funcs", types.NewSlice(funcType), false),
		types.NewField(token.NoPos, pkg, "Files", types.NewMap(types.Typ[types.String], types.NewPointer(objType)), false),
		types.NewField(token.NoPos, pkg, "Objs", types.NewSlice(objType), false),
	}, nil)
	tableType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Table", nil), tableStruct, nil)
	scope.Insert(tableType.Obj())
	tablePtr := types.NewPointer(tableType)
	tableRecv := types.NewVar(token.NoPos, nil, "t", tablePtr)
	tableType.AddMethod(types.NewFunc(token.NoPos, pkg, "PCToFunc",
		types.NewSignatureType(tableRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "pc", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(funcType))), false)))
	tableType.AddMethod(types.NewFunc(token.NoPos, pkg, "PCToLine",
		types.NewSignatureType(tableRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "pc", types.Typ[types.Uint64])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", types.NewPointer(funcType))), false)))
	tableType.AddMethod(types.NewFunc(token.NoPos, pkg, "LineToPC",
		types.NewSignatureType(tableRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "file", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "line", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Uint64]),
				types.NewVar(token.NoPos, nil, "", types.NewPointer(funcType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	tableType.AddMethod(types.NewFunc(token.NoPos, pkg, "LookupSym",
		types.NewSignatureType(tableRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(symType))), false)))
	tableType.AddMethod(types.NewFunc(token.NoPos, pkg, "LookupFunc",
		types.NewSignatureType(tableRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(funcType))), false)))

	// type LineTable struct (opaque)
	lineTableType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "LineTable", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(lineTableType.Obj())

	// func NewTable(symtab []byte, pcln *LineTable) (*Table, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewTable",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "symtab", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "pcln", types.NewPointer(lineTableType))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", tablePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewLineTable(data []byte, text uint64) *LineTable
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewLineTable",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "text", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(lineTableType))),
			false)))

	// type UnknownFileError string
	unknownFileErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "UnknownFileError", nil), types.Typ[types.String], nil)
	unknownFileErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", unknownFileErrType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(unknownFileErrType.Obj())

	// type UnknownLineError struct
	unknownLineErrType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "UnknownLineError", nil),
		types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "File", types.Typ[types.String], false),
			types.NewField(token.NoPos, pkg, "Line", types.Typ[types.Int], false),
		}, nil), nil)
	unknownLineErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", types.NewPointer(unknownLineErrType)), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(unknownLineErrType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildDebugPlan9objPackage() *types.Package {
	pkg := types.NewPackage("debug/plan9obj", "plan9obj")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type FileHeader struct
	fileHeaderStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Magic", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Bss", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Entry", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "PtrSize", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "LoadAddress", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "HdrSize", types.Typ[types.Uint64], false),
	}, nil)
	fileHeaderType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FileHeader", nil), fileHeaderStruct, nil)
	scope.Insert(fileHeaderType.Obj())

	// type Section struct
	sectionStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Size", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Uint32], false),
	}, nil)
	sectionType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Section", nil), sectionStruct, nil)
	scope.Insert(sectionType.Obj())
	sectionPtr := types.NewPointer(sectionType)
	sectionRecv := types.NewVar(token.NoPos, nil, "s", sectionPtr)
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Data",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, nil, "", errType)), false)))
	sectionType.AddMethod(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(sectionRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)))

	// type Sym struct
	symStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Type", types.Typ[types.Rune], false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
	}, nil)
	symType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Sym", nil), symStruct, nil)
	scope.Insert(symType.Obj())

	// type File struct
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "FileHeader", fileHeaderType, true),
		types.NewField(token.NoPos, pkg, "Sections", types.NewSlice(sectionPtr), false),
	}, nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "File", nil), fileStruct, nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	fileRecv := types.NewVar(token.NoPos, nil, "f", filePtr)
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Section",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", sectionPtr)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Symbols",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.NewSlice(symType)),
				types.NewVar(token.NoPos, nil, "", errType)), false)))

	// func Open(name string) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewFile(r io.ReaderAt) (*File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", filePtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Magic constants
	for _, c := range []struct {
		name string
		val  int64
	}{
		{"Magic386", 0x01EB}, {"MagicAMD64", 0x8A97}, {"MagicARM", 0x0104},
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, c.name, types.Typ[types.Uint32], constant.MakeInt64(c.val)))
	}

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
