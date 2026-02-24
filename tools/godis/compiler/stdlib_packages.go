package compiler

// stdlib_packages.go — additional standard library package stubs.
// These are buildXxxPackage() functions called from stubImporter.Import().

import (
	"go/constant"
	"go/token"
	"go/types"
)

func buildCryptoSHA512Package() *types.Package {
	pkg := types.NewPackage("crypto/sha512", "sha512")
	scope := pkg.Scope()

	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int], constant.MakeInt64(64)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size256", types.Typ[types.Int], constant.MakeInt64(32)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BlockSize", types.Typ[types.Int], constant.MakeInt64(128)))

	// func Sum512(data []byte) [64]byte — simplified as returning []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum512",
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

func buildCryptoSubtlePackage() *types.Package {
	pkg := types.NewPackage("crypto/subtle", "subtle")
	scope := pkg.Scope()

	// func ConstantTimeCompare(x, y []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ConstantTimeCompare",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "y", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func ConstantTimeSelect(v, x, y int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ConstantTimeSelect",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "v", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "y", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func ConstantTimeEq(x, y int32) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ConstantTimeEq",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int32]),
				types.NewVar(token.NoPos, pkg, "y", types.Typ[types.Int32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func XORBytes(dst, x, y []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "XORBytes",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "x", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "y", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildEncodingGobPackage() *types.Package {
	pkg := types.NewPackage("encoding/gob", "gob")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anyType := types.Universe.Lookup("any").Type()

	// type Encoder struct
	encStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	encType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Encoder", nil),
		encStruct, nil)
	scope.Insert(encType.Obj())
	encPtr := types.NewPointer(encType)

	// type Decoder struct
	decStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	decType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Decoder", nil),
		decStruct, nil)
	scope.Insert(decType.Obj())
	decPtr := types.NewPointer(decType)

	// func NewEncoder(w io.Writer) *Encoder — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewEncoder",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", encPtr)),
			false)))

	// func NewDecoder(r io.Reader) *Decoder — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewDecoder",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", decPtr)),
			false)))

	// func (enc *Encoder) Encode(e any) error
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "e", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func (dec *Decoder) Decode(e any) error
	decType.AddMethod(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "dec", decPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "e", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Register(value any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Register",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "value", anyType)),
			nil, false)))

	// func RegisterName(name string, value any)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RegisterName",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "value", anyType)),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildEncodingASCII85Package() *types.Package {
	pkg := types.NewPackage("encoding/ascii85", "ascii85")
	scope := pkg.Scope()

	// func Encode(dst, src []byte) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "src", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func MaxEncodedLen(n int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MaxEncodedLen",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildContainerListPackage() *types.Package {
	pkg := types.NewPackage("container/list", "list")
	scope := pkg.Scope()
	anyType := types.Universe.Lookup("any").Type()

	// type Element struct { Value any }
	elemStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Value", anyType, false),
	}, nil)
	elemType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Element", nil),
		elemStruct, nil)
	scope.Insert(elemType.Obj())
	elemPtr := types.NewPointer(elemType)

	// type List struct
	listStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "len", types.Typ[types.Int], false),
	}, nil)
	listType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "List", nil),
		listStruct, nil)
	scope.Insert(listType.Obj())
	listPtr := types.NewPointer(listType)

	// func New() *List
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", listPtr)),
			false)))

	// func (l *List) PushBack(v any) *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "PushBack",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) PushFront(v any) *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "PushFront",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) Len() int
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "Len",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func (l *List) Front() *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "Front",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) Remove(e *Element) any
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "Remove",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "e", elemPtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anyType)),
			false)))

	// func (e *Element) Next() *Element
	elemType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", elemPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildContainerRingPackage() *types.Package {
	pkg := types.NewPackage("container/ring", "ring")
	scope := pkg.Scope()

	ringStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Value", types.Universe.Lookup("any").Type(), false),
	}, nil)
	ringType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Ring", nil),
		ringStruct, nil)
	scope.Insert(ringType.Obj())
	ringPtr := types.NewPointer(ringType)

	// func New(n int) *Ring
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	// func (r *Ring) Len() int
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Len",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func (r *Ring) Next() *Ring
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildContainerHeapPackage() *types.Package {
	pkg := types.NewPackage("container/heap", "heap")
	scope := pkg.Scope()
	anyType := types.Universe.Lookup("any").Type()

	// func Init(h Interface) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Init",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "h", anyType)),
			nil, false)))

	// func Push(h Interface, x any) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Push",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "h", anyType),
				types.NewVar(token.NoPos, pkg, "x", anyType)),
			nil, false)))

	// func Pop(h Interface) any — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pop",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "h", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", anyType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildImagePackage() *types.Package {
	pkg := types.NewPackage("image", "image")
	scope := pkg.Scope()

	// type Point struct { X, Y int }
	pointStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Y", types.Typ[types.Int], false),
	}, nil)
	pointType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Point", nil),
		pointStruct, nil)
	scope.Insert(pointType.Obj())

	// func Pt(X, Y int) Point
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pt",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "X", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "Y", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))

	// type Rectangle struct { Min, Max Point }
	rectStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Min", pointType, false),
		types.NewField(token.NoPos, pkg, "Max", pointType, false),
	}, nil)
	rectType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Rectangle", nil),
		rectStruct, nil)
	scope.Insert(rectType.Obj())

	// func Rect(x0, y0, x1, y1 int) Rectangle
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Rect",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x0", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "y0", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "x1", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "y1", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))

	// type RGBA struct
	rgbaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Pix", types.NewSlice(types.Typ[types.Uint8]), false),
		types.NewField(token.NoPos, pkg, "Stride", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Rect", rectType, false),
	}, nil)
	rgbaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "RGBA", nil),
		rgbaStruct, nil)
	scope.Insert(rgbaType.Obj())

	// func NewRGBA(r Rectangle) *RGBA
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewRGBA",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(rgbaType))),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildImageColorPackage() *types.Package {
	pkg := types.NewPackage("image/color", "color")
	scope := pkg.Scope()

	// type RGBA struct { R, G, B, A uint8 }
	rgbaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "R", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "G", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "B", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "A", types.Typ[types.Uint8], false),
	}, nil)
	rgbaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "RGBA", nil),
		rgbaStruct, nil)
	scope.Insert(rgbaType.Obj())

	// var White, Black, Transparent, Opaque
	scope.Insert(types.NewVar(token.NoPos, pkg, "White", rgbaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Black", rgbaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Transparent", rgbaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Opaque", rgbaType))

	pkg.MarkComplete()
	return pkg
}

func buildImagePNGPackage() *types.Package {
	pkg := types.NewPackage("image/png", "png")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Encode(w io.Writer, m image.Image) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "m", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

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

func buildImageJPEGPackage() *types.Package {
	pkg := types.NewPackage("image/jpeg", "jpeg")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Encode(w io.Writer, m image.Image, o *Options) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "m", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "o", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Decode(r io.Reader) (image.Image, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultQuality", types.Typ[types.Int], constant.MakeInt64(75)))

	pkg.MarkComplete()
	return pkg
}

func buildDebugBuildInfoPackage() *types.Package {
	pkg := types.NewPackage("debug/buildinfo", "buildinfo")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type BuildInfo struct { GoVersion string; Path string }
	infoStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GoVersion", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
	}, nil)
	infoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "BuildInfo", nil),
		infoStruct, nil)
	scope.Insert(infoType.Obj())

	// func ReadFile(name string) (*BuildInfo, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(infoType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoASTPackage() *types.Package {
	pkg := types.NewPackage("go/ast", "ast")
	scope := pkg.Scope()

	// type File struct { Name *Ident }
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.Int], false),
	}, nil)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		fileStruct, nil)
	scope.Insert(fileType.Obj())

	// type Node interface — simplified
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Node", types.Typ[types.Int]))

	pkg.MarkComplete()
	return pkg
}

func buildGoTokenPackage() *types.Package {
	pkg := types.NewPackage("go/token", "token")
	scope := pkg.Scope()

	// type FileSet struct
	fsetStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	fsetType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FileSet", nil),
		fsetStruct, nil)
	scope.Insert(fsetType.Obj())

	// func NewFileSet() *FileSet
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFileSet",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(fsetType))),
			false)))

	// type Pos int
	posType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Pos", nil),
		types.Typ[types.Int], nil)
	scope.Insert(posType.Obj())

	scope.Insert(types.NewConst(token.NoPos, pkg, "NoPos", posType, constant.MakeInt64(0)))

	pkg.MarkComplete()
	return pkg
}

func buildGoParserPackage() *types.Package {
	pkg := types.NewPackage("go/parser", "parser")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Mode uint — simplified
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Mode", types.Typ[types.Uint]))

	// func ParseFile(fset *token.FileSet, filename string, src any, mode Mode) (*ast.File, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fset", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "filename", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "src", types.Universe.Lookup("any").Type()),
				types.NewVar(token.NoPos, pkg, "mode", types.Typ[types.Uint])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoFormatPackage() *types.Package {
	pkg := types.NewPackage("go/format", "format")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Source(src []byte) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Source",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "src", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildNetHTTPCookiejarPackage() *types.Package {
	pkg := types.NewPackage("net/http/cookiejar", "cookiejar")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	jarStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	jarType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Jar", nil),
		jarStruct, nil)
	scope.Insert(jarType.Obj())

	// func New(o *Options) (*Jar, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "o", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(jarType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildNetHTTPPprofPackage() *types.Package {
	pkg := types.NewPackage("net/http/pprof", "pprof")
	scope := pkg.Scope()

	// func Index(w http.ResponseWriter, r *http.Request) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Index",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildOsUserPackage() *types.Package {
	pkg := types.NewPackage("os/user", "user")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	userStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Uid", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Gid", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Username", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "HomeDir", types.Typ[types.String], false),
	}, nil)
	userType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "User", nil),
		userStruct, nil)
	scope.Insert(userType.Obj())
	userPtr := types.NewPointer(userType)

	// func Current() (*User, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Current",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", userPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Lookup(username string) (*User, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "username", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", userPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildRegexpSyntaxPackage() *types.Package {
	pkg := types.NewPackage("regexp/syntax", "syntax")
	scope := pkg.Scope()

	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Flags", types.Typ[types.Uint16]))

	scope.Insert(types.NewConst(token.NoPos, pkg, "Perl", types.Typ[types.Uint16], constant.MakeInt64(0xD2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "POSIX", types.Typ[types.Uint16], constant.MakeInt64(0)))

	pkg.MarkComplete()
	return pkg
}

func buildRuntimeDebugPackage() *types.Package {
	pkg := types.NewPackage("runtime/debug", "debug")
	scope := pkg.Scope()

	// func Stack() []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Stack",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func PrintStack()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PrintStack",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func FreeOSMemory()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "FreeOSMemory",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func SetGCPercent(percent int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetGCPercent",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "percent", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func ReadBuildInfo() (*BuildInfo, bool) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadBuildInfo",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildRuntimePprofPackage() *types.Package {
	pkg := types.NewPackage("runtime/pprof", "pprof")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func StartCPUProfile(w io.Writer) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StartCPUProfile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func StopCPUProfile()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StopCPUProfile",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildTextScannerPackage() *types.Package {
	pkg := types.NewPackage("text/scanner", "scanner")
	scope := pkg.Scope()

	scannerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	scannerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Scanner", nil),
		scannerStruct, nil)
	scope.Insert(scannerType.Obj())

	scope.Insert(types.NewConst(token.NoPos, pkg, "EOF", types.Typ[types.Int], constant.MakeInt64(-1)))

	pkg.MarkComplete()
	return pkg
}

func buildTextTabwriterPackage() *types.Package {
	pkg := types.NewPackage("text/tabwriter", "tabwriter")
	scope := pkg.Scope()

	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(writerType.Obj())
	writerPtr := types.NewPointer(writerType)

	// func NewWriter(output io.Writer, minwidth, tabwidth, padding int, padchar byte, flags uint) *Writer — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "output", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "minwidth", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "tabwidth", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "padding", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "padchar", types.Typ[types.Byte]),
				types.NewVar(token.NoPos, pkg, "flags", types.Typ[types.Uint])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", writerPtr)),
			false)))

	pkg.MarkComplete()
	return pkg
}
