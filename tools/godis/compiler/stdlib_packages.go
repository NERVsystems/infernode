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

	// const Size224 = 28
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size224", types.Typ[types.Int], constant.MakeInt64(28)))
	// const Size384 = 48
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size384", types.Typ[types.Int], constant.MakeInt64(48)))

	// func Sum384(data []byte) [48]byte — simplified as returning []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum384",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func Sum512_224(data []byte) [28]byte — simplified as returning []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum512_224",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func Sum512_256(data []byte) [32]byte — simplified as returning []byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sum512_256",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func New() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func New384() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New384",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func New512_224() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New512_224",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func New512_256() hash.Hash — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New512_256",
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

	// func (enc *Encoder) EncodeValue(value reflect.Value) error — simplified to any
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "EncodeValue",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "value", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func (dec *Decoder) DecodeValue(value reflect.Value) error — simplified to any
	decType.AddMethod(types.NewFunc(token.NoPos, pkg, "DecodeValue",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "dec", decPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "value", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type CommonType struct { Name string; Id typeId }
	commonStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
	}, nil)
	commonType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "CommonType", nil),
		commonStruct, nil)
	scope.Insert(commonType.Obj())

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

	// func (e *Element) Prev() *Element
	elemType.AddMethod(types.NewFunc(token.NoPos, pkg, "Prev",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", elemPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) Back() *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "Back",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) Init() *List
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "Init",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", listPtr)),
			false)))

	// func (l *List) InsertBefore(v any, mark *Element) *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "InsertBefore",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "v", anyType),
				types.NewVar(token.NoPos, pkg, "mark", elemPtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) InsertAfter(v any, mark *Element) *Element
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "InsertAfter",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "v", anyType),
				types.NewVar(token.NoPos, pkg, "mark", elemPtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", elemPtr)),
			false)))

	// func (l *List) MoveToFront(e *Element)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "MoveToFront",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "e", elemPtr)),
			nil, false)))

	// func (l *List) MoveToBack(e *Element)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "MoveToBack",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "e", elemPtr)),
			nil, false)))

	// func (l *List) MoveBefore(e, mark *Element)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "MoveBefore",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "e", elemPtr),
				types.NewVar(token.NoPos, pkg, "mark", elemPtr)),
			nil, false)))

	// func (l *List) MoveAfter(e, mark *Element)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "MoveAfter",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "e", elemPtr),
				types.NewVar(token.NoPos, pkg, "mark", elemPtr)),
			nil, false)))

	// func (l *List) PushBackList(other *List)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "PushBackList",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "other", listPtr)),
			nil, false)))

	// func (l *List) PushFrontList(other *List)
	listType.AddMethod(types.NewFunc(token.NoPos, pkg, "PushFrontList",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "l", listPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "other", listPtr)),
			nil, false)))

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

	// func (r *Ring) Prev() *Ring
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Prev",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	// func (r *Ring) Move(n int) *Ring
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Move",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	// func (r *Ring) Link(s *Ring) *Ring
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Link",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", ringPtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	// func (r *Ring) Unlink(n int) *Ring
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Unlink",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", ringPtr)),
			false)))

	// func (r *Ring) Do(f func(any))
	ringType.AddMethod(types.NewFunc(token.NoPos, pkg, "Do",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", ringPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f",
				types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("any").Type())),
					nil, false))),
			nil, false)))

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

	// func Fix(h Interface, i int) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "h", anyType),
				types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int])),
			nil, false)))

	// func Remove(h Interface, i int) any — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Remove",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "h", anyType),
				types.NewVar(token.NoPos, pkg, "i", types.Typ[types.Int])),
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

	rgbaPtr := types.NewPointer(rgbaType)

	// RGBA methods: Bounds, ColorModel, At, Set, RGBAAt, SetRGBA
	// func (p *RGBA) Bounds() Rectangle
	rgbaType.AddMethod(types.NewFunc(token.NoPos, pkg, "Bounds",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", rgbaPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))

	// func (p *RGBA) SubImage(r Rectangle) Image — simplified to any
	rgbaType.AddMethod(types.NewFunc(token.NoPos, pkg, "SubImage",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", rgbaPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// type NRGBA struct { Pix, Stride, Rect }
	nrgbaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Pix", types.NewSlice(types.Typ[types.Uint8]), false),
		types.NewField(token.NoPos, pkg, "Stride", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Rect", rectType, false),
	}, nil)
	nrgbaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NRGBA", nil),
		nrgbaStruct, nil)
	scope.Insert(nrgbaType.Obj())

	// func NewNRGBA(r Rectangle) *NRGBA
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewNRGBA",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(nrgbaType))),
			false)))

	// type Gray struct
	grayStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Pix", types.NewSlice(types.Typ[types.Uint8]), false),
		types.NewField(token.NoPos, pkg, "Stride", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Rect", rectType, false),
	}, nil)
	grayType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Gray", nil),
		grayStruct, nil)
	scope.Insert(grayType.Obj())

	// func NewGray(r Rectangle) *Gray
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewGray",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(grayType))),
			false)))

	// type Alpha struct
	alphaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Pix", types.NewSlice(types.Typ[types.Uint8]), false),
		types.NewField(token.NoPos, pkg, "Stride", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Rect", rectType, false),
	}, nil)
	alphaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Alpha", nil),
		alphaStruct, nil)
	scope.Insert(alphaType.Obj())

	// func NewAlpha(r Rectangle) *Alpha
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewAlpha",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(alphaType))),
			false)))

	// type Uniform struct { C color.Color } — simplified
	uniformStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "C", types.Typ[types.Int], false),
	}, nil)
	uniformType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Uniform", nil),
		uniformStruct, nil)
	scope.Insert(uniformType.Obj())

	// func NewUniform(c color.Color) *Uniform — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewUniform",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "c", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(uniformType))),
			false)))

	// type Config struct { ColorModel, Width, Height }
	configStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ColorModel", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Width", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Height", types.Typ[types.Int], false),
	}, nil)
	configType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Config", nil),
		configStruct, nil)
	scope.Insert(configType.Obj())

	errType := types.Universe.Lookup("error").Type()

	// func DecodeConfig(r io.Reader) (Config, string, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DecodeConfig",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", configType),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Decode(r io.Reader) (Image, string, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func RegisterFormat(name, magic string, decode, decodeConfig func) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RegisterFormat",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "magic", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "decode", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "decodeConfig", types.Typ[types.Int])),
			nil, false)))

	// Point methods
	pointPtr := types.NewPointer(pointType)
	_ = pointPtr
	// func (p Point) Add(q Point) Point
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "Add",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "q", pointType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))
	// func (p Point) Sub(q Point) Point
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sub",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "q", pointType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))
	// func (p Point) Mul(k int) Point
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "Mul",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "k", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))
	// func (p Point) Div(k int) Point
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "Div",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "k", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))
	// func (p Point) In(r Rectangle) bool
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "In",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (p Point) Eq(q Point) bool
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "Eq",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "q", pointType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (p Point) String() string
	pointType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", pointType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// Rectangle methods
	// func (r Rectangle) Dx() int
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Dx",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	// func (r Rectangle) Dy() int
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Dy",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	// func (r Rectangle) Size() Point
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Size",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointType)),
			false)))
	// func (r Rectangle) Empty() bool
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Empty",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (r Rectangle) Eq(s Rectangle) bool
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Eq",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (r Rectangle) Overlaps(s Rectangle) bool
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Overlaps",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (r Rectangle) In(s Rectangle) bool
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "In",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	// func (r Rectangle) Intersect(s Rectangle) Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Intersect",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) Union(s Rectangle) Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Union",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", rectType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) Add(p Point) Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Add",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", pointType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) Sub(p Point) Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Sub",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", pointType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) Inset(n int) Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Inset",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) Canon() Rectangle
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "Canon",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", rectType)),
			false)))
	// func (r Rectangle) String() string
	rectType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "r", rectType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// var ZP Point (deprecated but still used)
	scope.Insert(types.NewVar(token.NoPos, pkg, "ZP", pointType))
	// var ZR Rectangle (deprecated but still used)
	scope.Insert(types.NewVar(token.NoPos, pkg, "ZR", rectType))

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

	// type NRGBA struct { R, G, B, A uint8 }
	nrgbaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "R", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "G", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "B", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, pkg, "A", types.Typ[types.Uint8], false),
	}, nil)
	nrgbaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NRGBA", nil),
		nrgbaStruct, nil)
	scope.Insert(nrgbaType.Obj())

	// type Gray struct { Y uint8 }
	grayStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Y", types.Typ[types.Uint8], false),
	}, nil)
	grayType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Gray", nil),
		grayStruct, nil)
	scope.Insert(grayType.Obj())

	// type Alpha struct { A uint8 }
	alphaStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "A", types.Typ[types.Uint8], false),
	}, nil)
	alphaType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Alpha", nil),
		alphaStruct, nil)
	scope.Insert(alphaType.Obj())

	// type RGBA64 struct { R, G, B, A uint16 }
	rgba64Struct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "R", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "G", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "B", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, pkg, "A", types.Typ[types.Uint16], false),
	}, nil)
	rgba64Type := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "RGBA64", nil),
		rgba64Struct, nil)
	scope.Insert(rgba64Type.Obj())

	// Color interface (opaque)
	colorIface := types.NewInterfaceType(nil, nil)
	colorIface.Complete()
	colorType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Color", nil),
		colorIface, nil)
	scope.Insert(colorType.Obj())

	// Model interface (opaque)
	modelIface := types.NewInterfaceType(nil, nil)
	modelIface.Complete()
	modelType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Model", nil),
		modelIface, nil)
	scope.Insert(modelType.Obj())

	// Model vars
	scope.Insert(types.NewVar(token.NoPos, pkg, "RGBAModel", modelType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "RGBA64Model", modelType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "NRGBAModel", modelType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "GrayModel", modelType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "AlphaModel", modelType))

	// func ModelFunc(f func(Color) Color) Model — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ModelFunc",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "f", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", modelType)),
			false)))

	// var White, Black, Transparent, Opaque
	scope.Insert(types.NewVar(token.NoPos, pkg, "White", rgbaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Black", rgbaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Transparent", alphaType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "Opaque", alphaType))

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

	// Pos type (simplified as int, same underlying type as token.Pos)
	posType := types.Typ[types.Int]

	// type Node interface { Pos() token.Pos; End() token.Pos }
	nodeIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
		types.NewFunc(token.NoPos, pkg, "End",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
	}, nil)
	nodeIface.Complete()
	nodeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Node", nil), nodeIface, nil)
	scope.Insert(nodeType.Obj())

	// type Expr interface { Node; exprNode() }
	exprIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
		types.NewFunc(token.NoPos, pkg, "End",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
	}, nil)
	exprIface.Complete()
	exprType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Expr", nil), exprIface, nil)
	scope.Insert(exprType.Obj())

	// type Stmt interface { Node; stmtNode() }
	stmtIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
		types.NewFunc(token.NoPos, pkg, "End",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
	}, nil)
	stmtIface.Complete()
	stmtType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Stmt", nil), stmtIface, nil)
	scope.Insert(stmtType.Obj())

	// type Decl interface { Node; declNode() }
	declIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
		types.NewFunc(token.NoPos, pkg, "End",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
	}, nil)
	declIface.Complete()
	declType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Decl", nil), declIface, nil)
	scope.Insert(declType.Obj())

	// type Spec interface (same shape as Node)
	specIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Pos",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
		types.NewFunc(token.NoPos, pkg, "End",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)),
	}, nil)
	specIface.Complete()
	specType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Spec", nil), specIface, nil)
	scope.Insert(specType.Obj())

	// type Object struct { Kind ObjKind; Name string; Decl interface{}; Data interface{}; Type interface{} }
	objKindType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ObjKind", nil), types.Typ[types.Int], nil)
	scope.Insert(objKindType.Obj())
	for i, name := range []string{"Bad", "Pkg", "Con", "Typ", "Var", "Fun", "Lbl"} {
		scope.Insert(types.NewConst(token.NoPos, pkg, name, objKindType, constant.MakeInt64(int64(i))))
	}

	anyType := types.NewInterfaceType(nil, nil)
	objectStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Kind", objKindType, false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Decl", anyType, false),
		types.NewField(token.NoPos, pkg, "Data", anyType, false),
		types.NewField(token.NoPos, pkg, "Type", anyType, false),
	}, nil)
	objectType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Object", nil), objectStruct, nil)
	scope.Insert(objectType.Obj())
	objectPtr := types.NewPointer(objectType)

	// type Ident struct { NamePos token.Pos; Name string; Obj *Object }
	identStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "NamePos", posType, false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Obj", objectPtr, false),
	}, nil)
	identType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Ident", nil), identStruct, nil)
	scope.Insert(identType.Obj())
	identPtr := types.NewPointer(identType)

	// type Scope struct { Outer *Scope; Objects map[string]*Object }
	scopeObjType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Scope", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(scopeObjType.Obj())
	scopePtr := types.NewPointer(scopeObjType)
	scopeRecv := types.NewVar(token.NoPos, nil, "s", scopePtr)
	scopeObjType.AddMethod(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(scopeRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", objectPtr)), false)))
	scopeObjType.AddMethod(types.NewFunc(token.NoPos, pkg, "Insert",
		types.NewSignatureType(scopeRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "obj", objectPtr)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", objectPtr)), false)))
	scopeObjType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(scopeRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewScope",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "outer", scopePtr)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", scopePtr)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewObj",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "kind", objKindType),
				types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", objectPtr)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewIdent",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", identPtr)), false)))

	// type BasicLit struct { ValuePos token.Pos; Kind token.Token; Value string }
	basicLitStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ValuePos", posType, false),
		types.NewField(token.NoPos, pkg, "Kind", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.String], false),
	}, nil)
	basicLitType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "BasicLit", nil), basicLitStruct, nil)
	scope.Insert(basicLitType.Obj())

	// type CommentGroup struct { List []*Comment }
	commentStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Slash", posType, false),
		types.NewField(token.NoPos, pkg, "Text", types.Typ[types.String], false),
	}, nil)
	commentType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Comment", nil), commentStruct, nil)
	scope.Insert(commentType.Obj())

	commentGroupStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "List", types.NewSlice(types.NewPointer(commentType)), false),
	}, nil)
	commentGroupType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "CommentGroup", nil), commentGroupStruct, nil)
	scope.Insert(commentGroupType.Obj())
	commentGroupPtr := types.NewPointer(commentGroupType)
	commentGroupType.AddMethod(types.NewFunc(token.NoPos, pkg, "Text",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "g", commentGroupPtr), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// type FieldList struct { Opening token.Pos; List []*Field; Closing token.Pos }
	fieldStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Names", types.NewSlice(identPtr), false),
		types.NewField(token.NoPos, pkg, "Type", exprType, false),
		types.NewField(token.NoPos, pkg, "Tag", types.NewPointer(basicLitType), false),
		types.NewField(token.NoPos, pkg, "Comment", commentGroupPtr, false),
	}, nil)
	fieldType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Field", nil), fieldStruct, nil)
	scope.Insert(fieldType.Obj())
	fieldListStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Opening", posType, false),
		types.NewField(token.NoPos, pkg, "List", types.NewSlice(types.NewPointer(fieldType)), false),
		types.NewField(token.NoPos, pkg, "Closing", posType, false),
	}, nil)
	fieldListType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FieldList", nil), fieldListStruct, nil)
	scope.Insert(fieldListType.Obj())
	fieldListPtr := types.NewPointer(fieldListType)
	fieldListType.AddMethod(types.NewFunc(token.NoPos, pkg, "NumFields",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "f", fieldListPtr), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))

	// type FuncType struct { Func token.Pos; Params, Results *FieldList }
	funcTypeStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Func", posType, false),
		types.NewField(token.NoPos, pkg, "Params", fieldListPtr, false),
		types.NewField(token.NoPos, pkg, "Results", fieldListPtr, false),
	}, nil)
	funcTypeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FuncType", nil), funcTypeStruct, nil)
	scope.Insert(funcTypeType.Obj())

	// type FuncDecl struct { Doc *CommentGroup; Recv *FieldList; Name *Ident; Type *FuncType; Body *BlockStmt }
	blockStmtType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "BlockStmt", nil),
		types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "Lbrace", posType, false),
			types.NewField(token.NoPos, pkg, "List", types.NewSlice(stmtType), false),
			types.NewField(token.NoPos, pkg, "Rbrace", posType, false),
		}, nil), nil)
	scope.Insert(blockStmtType.Obj())

	funcDeclStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Doc", commentGroupPtr, false),
		types.NewField(token.NoPos, pkg, "Recv", fieldListPtr, false),
		types.NewField(token.NoPos, pkg, "Name", identPtr, false),
		types.NewField(token.NoPos, pkg, "Type", types.NewPointer(funcTypeType), false),
		types.NewField(token.NoPos, pkg, "Body", types.NewPointer(blockStmtType), false),
	}, nil)
	funcDeclType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FuncDecl", nil), funcDeclStruct, nil)
	scope.Insert(funcDeclType.Obj())

	// type GenDecl struct { Doc *CommentGroup; TokPos token.Pos; Tok token.Token; Specs []Spec }
	genDeclStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Doc", commentGroupPtr, false),
		types.NewField(token.NoPos, pkg, "TokPos", posType, false),
		types.NewField(token.NoPos, pkg, "Tok", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Lparen", posType, false),
		types.NewField(token.NoPos, pkg, "Specs", types.NewSlice(specType), false),
		types.NewField(token.NoPos, pkg, "Rparen", posType, false),
	}, nil)
	genDeclType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "GenDecl", nil), genDeclStruct, nil)
	scope.Insert(genDeclType.Obj())

	// type ImportSpec struct { Doc *CommentGroup; Name *Ident; Path *BasicLit; Comment *CommentGroup }
	importSpecStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Doc", commentGroupPtr, false),
		types.NewField(token.NoPos, pkg, "Name", identPtr, false),
		types.NewField(token.NoPos, pkg, "Path", types.NewPointer(basicLitType), false),
		types.NewField(token.NoPos, pkg, "Comment", commentGroupPtr, false),
	}, nil)
	importSpecType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ImportSpec", nil), importSpecStruct, nil)
	scope.Insert(importSpecType.Obj())

	// type File struct
	fileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Doc", commentGroupPtr, false),
		types.NewField(token.NoPos, pkg, "Package", posType, false),
		types.NewField(token.NoPos, pkg, "Name", identPtr, false),
		types.NewField(token.NoPos, pkg, "Decls", types.NewSlice(declType), false),
		types.NewField(token.NoPos, pkg, "Scope", scopePtr, false),
		types.NewField(token.NoPos, pkg, "Imports", types.NewSlice(types.NewPointer(importSpecType)), false),
		types.NewField(token.NoPos, pkg, "Unresolved", types.NewSlice(identPtr), false),
		types.NewField(token.NoPos, pkg, "Comments", types.NewSlice(commentGroupPtr), false),
	}, nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "File", nil), fileStruct, nil)
	scope.Insert(fileType.Obj())

	// type Package struct { Name string; Scope *Scope; Imports map[string]*Object; Files map[string]*File }
	packageStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Scope", scopePtr, false),
		types.NewField(token.NoPos, pkg, "Files", types.NewMap(types.Typ[types.String], types.NewPointer(fileType)), false),
	}, nil)
	packageType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Package", nil), packageStruct, nil)
	scope.Insert(packageType.Obj())

	// Helper functions
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Inspect",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "node", nodeType),
				types.NewVar(token.NoPos, nil, "f", types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "node", nodeType)),
					types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false))),
			nil, false)))

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Walk",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "v", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "node", nodeType)),
			nil, false)))

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Print",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "x", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())), false)))

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Fprint",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "w", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "x", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, nil, "f", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())), false)))

	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsExported",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))

	// type Visitor interface { Visit(node Node) (w Visitor) }
	visitorIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Visit",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "node", nodeType)),
				types.NewTuple(types.NewVar(token.NoPos, nil, "w", types.NewInterfaceType(nil, nil))), false)),
	}, nil)
	visitorIface.Complete()
	visitorType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Visitor", nil), visitorIface, nil)
	scope.Insert(visitorType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildGoTokenPackage() *types.Package {
	pkg := types.NewPackage("go/token", "token")
	scope := pkg.Scope()

	// type Token int
	tokenType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Token", nil),
		types.Typ[types.Int], nil)
	scope.Insert(tokenType.Obj())
	tokenType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "tok", tokenType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	tokenType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsLiteral",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "tok", tokenType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	tokenType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsOperator",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "tok", tokenType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	tokenType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsKeyword",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "tok", tokenType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	tokenType.AddMethod(types.NewFunc(token.NoPos, pkg, "Precedence",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "tok", tokenType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))

	// Token constants
	for i, name := range []string{
		"ILLEGAL", "EOF", "COMMENT",
		"IDENT", "INT", "FLOAT", "IMAG", "CHAR", "STRING",
		"ADD", "SUB", "MUL", "QUO", "REM",
		"AND", "OR", "XOR", "SHL", "SHR", "AND_NOT",
		"ADD_ASSIGN", "SUB_ASSIGN", "MUL_ASSIGN", "QUO_ASSIGN", "REM_ASSIGN",
		"LAND", "LOR", "ARROW", "INC", "DEC",
		"EQL", "LSS", "GTR", "ASSIGN", "NOT",
		"NEQ", "LEQ", "GEQ", "DEFINE", "ELLIPSIS",
		"LPAREN", "LBRACK", "LBRACE", "COMMA", "PERIOD",
		"RPAREN", "RBRACK", "RBRACE", "SEMICOLON", "COLON",
		"BREAK", "CASE", "CHAN", "CONST", "CONTINUE",
		"DEFAULT", "DEFER", "ELSE", "FALLTHROUGH", "FOR",
		"FUNC", "GO", "GOTO", "IF", "IMPORT",
		"INTERFACE", "MAP", "PACKAGE", "RANGE", "RETURN",
		"SELECT", "STRUCT", "SWITCH", "TYPE", "VAR",
	} {
		scope.Insert(types.NewConst(token.NoPos, pkg, name, tokenType, constant.MakeInt64(int64(i))))
	}

	// func Lookup(ident string) Token
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ident", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", tokenType)), false)))

	// type Pos int
	posType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Pos", nil),
		types.Typ[types.Int], nil)
	scope.Insert(posType.Obj())
	posType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsValid",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", posType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NoPos", posType, constant.MakeInt64(0)))

	// type Position struct
	positionStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Filename", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Line", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Column", types.Typ[types.Int], false),
	}, nil)
	positionType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Position", nil),
		positionStruct, nil)
	scope.Insert(positionType.Obj())
	positionType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsValid",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "pos", positionType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	positionType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "pos", positionType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// type File struct (opaque)
	fileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "File", nil),
		types.NewStruct(nil, nil), nil)
	scope.Insert(fileType.Obj())
	filePtr := types.NewPointer(fileType)
	fileRecv := types.NewVar(token.NoPos, nil, "f", filePtr)
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Name",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Base",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Size",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "LineCount",
		types.NewSignatureType(fileRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Pos",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "offset", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", posType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Offset",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", posType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Position",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", posType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", positionType)), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Line",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", posType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	fileType.AddMethod(types.NewFunc(token.NoPos, pkg, "AddLine",
		types.NewSignatureType(fileRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "offset", types.Typ[types.Int])),
			nil, false)))

	// type FileSet struct (opaque)
	fsetType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "FileSet", nil),
		types.NewStruct(nil, nil), nil)
	scope.Insert(fsetType.Obj())
	fsetPtr := types.NewPointer(fsetType)
	fsetRecv := types.NewVar(token.NoPos, nil, "s", fsetPtr)

	// func NewFileSet() *FileSet
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewFileSet",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", fsetPtr)),
			false)))

	// FileSet methods
	fsetType.AddMethod(types.NewFunc(token.NoPos, pkg, "AddFile",
		types.NewSignatureType(fsetRecv, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "filename", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "base", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "size", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", filePtr)), false)))
	fsetType.AddMethod(types.NewFunc(token.NoPos, pkg, "Position",
		types.NewSignatureType(fsetRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", posType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", positionType)), false)))
	fsetType.AddMethod(types.NewFunc(token.NoPos, pkg, "File",
		types.NewSignatureType(fsetRecv, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "p", posType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", filePtr)), false)))
	fsetType.AddMethod(types.NewFunc(token.NoPos, pkg, "Base",
		types.NewSignatureType(fsetRecv, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoParserPackage() *types.Package {
	pkg := types.NewPackage("go/parser", "parser")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anyType := types.Universe.Lookup("any").Type()

	// type Mode uint
	modeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Mode", nil), types.Typ[types.Uint], nil)
	scope.Insert(modeType.Obj())

	// Mode constants
	for i, name := range []string{"PackageClauseOnly", "ImportsOnly", "ParseComments",
		"Trace", "DeclarationErrors", "SpuriousErrors", "SkipObjectResolution", "AllErrors"} {
		val := int64(1 << i)
		if name == "AllErrors" {
			val = 1 << 5 // same as SpuriousErrors
		}
		scope.Insert(types.NewConst(token.NoPos, pkg, name, modeType, constant.MakeInt64(val)))
	}

	// Opaque fset and file types (cross-package references simplified)
	fsetType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "fset", nil), types.NewStruct(nil, nil), nil)
	fileType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "file", nil), types.NewStruct(nil, nil), nil)

	// func ParseFile(fset *token.FileSet, filename string, src any, mode Mode) (*ast.File, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseFile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fset", types.NewPointer(fsetType)),
				types.NewVar(token.NoPos, pkg, "filename", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "src", anyType),
				types.NewVar(token.NoPos, pkg, "mode", modeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(fileType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ParseDir(fset *token.FileSet, path string, filter func(fs.FileInfo) bool, mode Mode) (map[string]*ast.Package, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseDir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fset", types.NewPointer(fsetType)),
				types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "filter", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "mode", modeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil))),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ParseExprFrom(fset *token.FileSet, filename string, src any, mode Mode) (ast.Expr, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseExprFrom",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fset", types.NewPointer(fsetType)),
				types.NewVar(token.NoPos, pkg, "filename", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "src", anyType),
				types.NewVar(token.NoPos, pkg, "mode", modeType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ParseExpr(x string) (ast.Expr, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseExpr",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoFormatPackage() *types.Package {
	pkg := types.NewPackage("go/format", "format")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// func Source(src []byte) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Source",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "src", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", byteSlice),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Node(dst io.Writer, fset *token.FileSet, node interface{}) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Node",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "dst", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "fset", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "node", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
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

	// func LookupId(uid string) (*User, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LookupId",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "uid", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", userPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// User.GroupIds() ([]string, error)
	userType.AddMethod(types.NewFunc(token.NoPos, pkg, "GroupIds",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "u", userPtr),
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type Group struct
	groupStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Gid", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
	}, nil)
	groupType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Group", nil),
		groupStruct, nil)
	scope.Insert(groupType.Obj())
	groupPtr := types.NewPointer(groupType)

	// func LookupGroup(name string) (*Group, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LookupGroup",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", groupPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func LookupGroupId(gid string) (*Group, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "LookupGroupId",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "gid", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", groupPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type UnknownUserError string
	unknownUserType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "UnknownUserError", nil),
		types.Typ[types.String], nil)
	unknownUserType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", unknownUserType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(unknownUserType.Obj())

	// type UnknownUserIdError int
	unknownUserIdType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "UnknownUserIdError", nil),
		types.Typ[types.Int], nil)
	unknownUserIdType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", unknownUserIdType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(unknownUserIdType.Obj())

	// type UnknownGroupError string
	unknownGroupType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "UnknownGroupError", nil),
		types.Typ[types.String], nil)
	unknownGroupType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", unknownGroupType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(unknownGroupType.Obj())

	// type UnknownGroupIdError string
	unknownGroupIdType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "UnknownGroupIdError", nil),
		types.Typ[types.String], nil)
	unknownGroupIdType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", unknownGroupIdType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(unknownGroupIdType.Obj())

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

	// Module type
	moduleStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Version", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Sum", types.Typ[types.String], false),
	}, nil)
	moduleType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Module", nil),
		moduleStruct, nil)
	scope.Insert(moduleType.Obj())

	// BuildSetting type
	buildSettingStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Key", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Value", types.Typ[types.String], false),
	}, nil)
	buildSettingType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "BuildSetting", nil),
		buildSettingStruct, nil)
	scope.Insert(buildSettingType.Obj())

	// BuildInfo type
	buildInfoStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GoVersion", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Main", moduleType, false),
		types.NewField(token.NoPos, pkg, "Deps", types.NewSlice(types.NewPointer(moduleType)), false),
		types.NewField(token.NoPos, pkg, "Settings", types.NewSlice(buildSettingType), false),
	}, nil)
	buildInfoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "BuildInfo", nil),
		buildInfoStruct, nil)
	scope.Insert(buildInfoType.Obj())
	buildInfoPtr := types.NewPointer(buildInfoType)

	// BuildInfo.String() method
	buildInfoType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "bi", buildInfoPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

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

	// func SetMaxStack(bytes int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetMaxStack",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "bytes", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func SetMaxThreads(threads int) int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetMaxThreads",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "threads", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func SetPanicOnFault(enabled bool) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetPanicOnFault",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "enabled", types.Typ[types.Bool])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func SetTraceback(level string)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetTraceback",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "level", types.Typ[types.String])),
			nil, false)))

	// func ReadBuildInfo() (*BuildInfo, bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadBuildInfo",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", buildInfoPtr),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// GCStats type
	gcStatsStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "LastGC", types.Typ[types.Int64], false),
		types.NewField(token.NoPos, pkg, "NumGC", types.Typ[types.Int64], false),
		types.NewField(token.NoPos, pkg, "PauseTotal", types.Typ[types.Int64], false),
	}, nil)
	gcStatsType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "GCStats", nil),
		gcStatsStruct, nil)
	scope.Insert(gcStatsType.Obj())

	// func ReadGCStats(stats *GCStats)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ReadGCStats",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "stats", types.NewPointer(gcStatsType))),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildRuntimePprofPackage() *types.Package {
	pkg := types.NewPackage("runtime/pprof", "pprof")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// Profile type
	profileStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	profileType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Profile", nil),
		profileStruct, nil)
	scope.Insert(profileType.Obj())
	profilePtr := types.NewPointer(profileType)

	// Profile.Name() string
	profileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Name",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "p", profilePtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// Profile.Count() int
	profileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Count",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "p", profilePtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// Profile.Add(value any, skip int)
	profileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Add",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "p", profilePtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "skip", types.Typ[types.Int])),
			nil, false)))

	// Profile.Remove(value any)
	profileType.AddMethod(types.NewFunc(token.NoPos, pkg, "Remove",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "p", profilePtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int])),
			nil, false)))

	// Profile.WriteTo(w io.Writer, debug int) error
	profileType.AddMethod(types.NewFunc(token.NoPos, pkg, "WriteTo",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "p", profilePtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "debug", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Lookup(name string) *Profile
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", profilePtr)),
			false)))

	// func NewProfile(name string) *Profile
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewProfile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", profilePtr)),
			false)))

	// func Profiles() []*Profile
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Profiles",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(profilePtr))),
			false)))

	// func StartCPUProfile(w io.Writer) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StartCPUProfile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func StopCPUProfile()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StopCPUProfile",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func WriteHeapProfile(w io.Writer) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteHeapProfile",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func SetGoroutineLabels(ctx context.Context)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SetGoroutineLabels",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ctx", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildTextScannerPackage() *types.Package {
	pkg := types.NewPackage("text/scanner", "scanner")
	scope := pkg.Scope()

	// Position type
	posStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Filename", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Line", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Column", types.Typ[types.Int], false),
	}, nil)
	posType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Position", nil),
		posStruct, nil)
	scope.Insert(posType.Obj())

	// Position.IsValid() bool
	posType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsValid",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "pos", types.NewPointer(posType)),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// Position.String() string
	posType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "pos", posType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// Scanner type
	scannerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Mode", types.Typ[types.Uint], false),
		types.NewField(token.NoPos, pkg, "Whitespace", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "Position", posType, true), // embedded
		types.NewField(token.NoPos, pkg, "IsIdentRune", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Error", types.Typ[types.Int], false),
	}, nil)
	scannerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Scanner", nil),
		scannerStruct, nil)
	scope.Insert(scannerType.Obj())
	scannerPtr := types.NewPointer(scannerType)

	// Scanner.Init(src io.Reader) *Scanner
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Init",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "src", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", scannerPtr)),
			false)))

	// Scanner.Scan() rune
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Scan",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// Scanner.Peek() rune
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Peek",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// Scanner.Next() rune
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int32])),
			false)))

	// Scanner.TokenText() string
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "TokenText",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// Scanner.Pos() Position
	scannerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Pos",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "s", scannerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", posType)),
			false)))

	// func TokenString(tok rune) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TokenString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "tok", types.Typ[types.Int32])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// Constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "EOF", types.Typ[types.Int32], constant.MakeInt64(-1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Ident", types.Typ[types.Int32], constant.MakeInt64(-2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Int", types.Typ[types.Int32], constant.MakeInt64(-3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Float", types.Typ[types.Int32], constant.MakeInt64(-4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Char", types.Typ[types.Int32], constant.MakeInt64(-5)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "String", types.Typ[types.Int32], constant.MakeInt64(-6)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "RawString", types.Typ[types.Int32], constant.MakeInt64(-7)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Comment", types.Typ[types.Int32], constant.MakeInt64(-8)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanIdents", types.Typ[types.Uint], constant.MakeUint64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanInts", types.Typ[types.Uint], constant.MakeUint64(8)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanFloats", types.Typ[types.Uint], constant.MakeUint64(16)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanChars", types.Typ[types.Uint], constant.MakeUint64(32)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanStrings", types.Typ[types.Uint], constant.MakeUint64(64)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanRawStrings", types.Typ[types.Uint], constant.MakeUint64(128)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanComments", types.Typ[types.Uint], constant.MakeUint64(256)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SkipComments", types.Typ[types.Uint], constant.MakeUint64(512)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "GoTokens", types.Typ[types.Uint], constant.MakeUint64(1012)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "GoWhitespace", types.Typ[types.Uint64], constant.MakeUint64(1<<'\t'|1<<'\n'|1<<'\r'|1<<' ')))

	pkg.MarkComplete()
	return pkg
}

func buildTextTabwriterPackage() *types.Package {
	pkg := types.NewPackage("text/tabwriter", "tabwriter")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

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

	// Writer.Init(output io.Writer, minwidth, tabwidth, padding int, padchar byte, flags uint) *Writer
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Init",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "w", writerPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "output", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "minwidth", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "tabwidth", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "padding", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "padchar", types.Typ[types.Byte]),
				types.NewVar(token.NoPos, pkg, "flags", types.Typ[types.Uint])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", writerPtr)),
			false)))

	// Writer.Write(buf []byte) (n int, err error)
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "w", writerPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "buf", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Writer.Flush() error
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Flush",
		types.NewSignatureType(
			types.NewVar(token.NoPos, pkg, "w", writerPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Constants for flags
	scope.Insert(types.NewConst(token.NoPos, pkg, "FilterHTML", types.Typ[types.Uint], constant.MakeUint64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "StripEscape", types.Typ[types.Uint], constant.MakeUint64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "AlignRight", types.Typ[types.Uint], constant.MakeUint64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DiscardEmptyColumns", types.Typ[types.Uint], constant.MakeUint64(8)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TabIndent", types.Typ[types.Uint], constant.MakeUint64(16)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Debug", types.Typ[types.Uint], constant.MakeUint64(32)))

	// Escape constant
	scope.Insert(types.NewConst(token.NoPos, pkg, "Escape", types.Typ[types.Byte], constant.MakeInt64(0xff)))

	pkg.MarkComplete()
	return pkg
}
