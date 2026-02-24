package compiler

// stdlib_packages3.go — additional stdlib package type stubs.

import (
	"go/constant"
	"go/token"
	"go/types"
)

func buildEncodingBase32Package() *types.Package {
	pkg := types.NewPackage("encoding/base32", "base32")
	scope := pkg.Scope()

	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Encoding struct { ... }
	encStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	encType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Encoding", nil),
		encStruct, nil)
	scope.Insert(encType.Obj())
	encPtr := types.NewPointer(encType)

	// var StdEncoding, HexEncoding *Encoding
	scope.Insert(types.NewVar(token.NoPos, pkg, "StdEncoding", encPtr))
	scope.Insert(types.NewVar(token.NoPos, pkg, "HexEncoding", encPtr))

	// func NewEncoding(encoder string) *Encoding
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewEncoding",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "encoder", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", encPtr)),
			false)))

	// Methods on *Encoding
	errType := types.Universe.Lookup("error").Type()
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "EncodeToString",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "src", byteSlice)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "DecodeString",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", byteSlice),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "Encode",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "dst", byteSlice),
				types.NewVar(token.NoPos, nil, "src", byteSlice)),
			nil, false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "Decode",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "dst", byteSlice),
				types.NewVar(token.NoPos, nil, "src", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "EncodedLen",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "DecodedLen",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))
	encType.AddMethod(types.NewFunc(token.NoPos, pkg, "WithPadding",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "enc", encPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "padding", types.Typ[types.Rune])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", encType)),
			false)))

	// const NoPadding rune = -1
	scope.Insert(types.NewConst(token.NoPos, pkg, "NoPadding", types.Typ[types.Rune], constant.MakeInt64(-1)))
	// const StdPadding rune = '='
	scope.Insert(types.NewConst(token.NoPos, pkg, "StdPadding", types.Typ[types.Rune], constant.MakeInt64('=')))

	pkg.MarkComplete()
	return pkg
}

func buildCryptoDESPackage() *types.Package {
	pkg := types.NewPackage("crypto/des", "des")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type KeySizeError int
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "KeySizeError",
		types.NewNamed(types.NewTypeName(token.NoPos, pkg, "KeySizeError", nil),
			types.Typ[types.Int], nil)))

	// const BlockSize = 8
	scope.Insert(types.NewConst(token.NoPos, pkg, "BlockSize", types.Typ[types.Int], constant.MakeInt64(8)))

	// func NewCipher(key []byte) (cipher.Block, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewCipher",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "key", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewTripleDESCipher(key []byte) (cipher.Block, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewTripleDESCipher",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "key", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildCryptoRC4Package() *types.Package {
	pkg := types.NewPackage("crypto/rc4", "rc4")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Cipher struct { ... }
	cipherStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "s", types.Typ[types.Int], false),
	}, nil)
	cipherType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Cipher", nil),
		cipherStruct, nil)
	scope.Insert(cipherType.Obj())
	cipherPtr := types.NewPointer(cipherType)

	// func NewCipher(key []byte) (*Cipher, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewCipher",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "key", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", cipherPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// (*Cipher).XORKeyStream(dst, src []byte)
	cipherType.AddMethod(types.NewFunc(token.NoPos, pkg, "XORKeyStream",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", cipherPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "dst", byteSlice),
				types.NewVar(token.NoPos, nil, "src", byteSlice)),
			nil, false)))

	// (*Cipher).Reset()
	cipherType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", cipherPtr),
			nil, nil, nil, nil, false)))

	pkg.MarkComplete()
	return pkg
}

func buildSyscallPackage() *types.Package {
	pkg := types.NewPackage("syscall", "syscall")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()

	// type Errno uintptr — implements error
	errnoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Errno", nil),
		types.Typ[types.Uintptr], nil)
	scope.Insert(errnoType.Obj())

	// type Signal int
	signalType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Signal", nil),
		types.Typ[types.Int], nil)
	scope.Insert(signalType.Obj())

	// type SysProcAttr struct { ... }
	sysProcAttrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Chroot", types.Typ[types.String], false),
	}, nil)
	sysProcAttrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "SysProcAttr", nil),
		sysProcAttrStruct, nil)
	scope.Insert(sysProcAttrType.Obj())

	// Error constants
	scope.Insert(types.NewVar(token.NoPos, pkg, "EINVAL", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ENOENT", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "EEXIST", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "EPERM", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "EACCES", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "EAGAIN", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ENOSYS", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ENOTDIR", errnoType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "EISDIR", errnoType))

	// Signal constants
	scope.Insert(types.NewVar(token.NoPos, pkg, "SIGINT", signalType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "SIGTERM", signalType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "SIGKILL", signalType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "SIGHUP", signalType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "SIGPIPE", signalType))

	// Constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_RDONLY", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_WRONLY", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_RDWR", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_CREAT", types.Typ[types.Int], constant.MakeInt64(0100)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_TRUNC", types.Typ[types.Int], constant.MakeInt64(01000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_APPEND", types.Typ[types.Int], constant.MakeInt64(02000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_EXCL", types.Typ[types.Int], constant.MakeInt64(0200)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_SYNC", types.Typ[types.Int], constant.MakeInt64(04010000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_NONBLOCK", types.Typ[types.Int], constant.MakeInt64(04000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "O_CLOEXEC", types.Typ[types.Int], constant.MakeInt64(02000000)))

	scope.Insert(types.NewConst(token.NoPos, pkg, "STDIN_FILENO", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "STDOUT_FILENO", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "STDERR_FILENO", types.Typ[types.Int], constant.MakeInt64(2)))

	// func Getenv(key string) (value string, found bool)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getenv",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "key", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "value", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "found", types.Typ[types.Bool])),
			false)))

	// func Getpid() int
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getpid",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getuid",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Getgid",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	// func Exit(code int)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Exit",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "code", types.Typ[types.Int])),
			nil, false)))

	// func Open(path string, mode int, perm uint32) (fd int, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "mode", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "perm", types.Typ[types.Uint32])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fd", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func Close(fd int) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "fd", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Read(fd int, p []byte) (n int, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fd", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func Write(fd int, p []byte) (n int, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fd", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildUnsafePackage() *types.Package {
	pkg := types.NewPackage("unsafe", "unsafe")
	scope := pkg.Scope()

	// type Pointer *ArbitraryType — modeled as uintptr
	pointerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Pointer", nil),
		types.Typ[types.Uintptr], nil)
	scope.Insert(pointerType.Obj())

	// func Sizeof(x ArbitraryType) uintptr
	anyType := types.NewInterfaceType(nil, nil)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Sizeof",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uintptr])),
			false)))

	// func Offsetof(x ArbitraryType) uintptr
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Offsetof",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uintptr])),
			false)))

	// func Alignof(x ArbitraryType) uintptr
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Alignof",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uintptr])),
			false)))

	// func Add(ptr Pointer, len IntegerType) Pointer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Add",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ptr", pointerType),
				types.NewVar(token.NoPos, pkg, "len", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", pointerType)),
			false)))

	// func Slice(ptr *ArbitraryType, len IntegerType) []ArbitraryType
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Slice",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ptr", types.NewPointer(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "len", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))

	// func String(ptr *byte, len IntegerType) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ptr", types.NewPointer(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "len", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func SliceData(slice []ArbitraryType) *ArbitraryType
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SliceData",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "slice", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(types.Typ[types.Byte]))),
			false)))

	// func StringData(str string) *byte
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StringData",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "str", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(types.Typ[types.Byte]))),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildMathCmplxPackage() *types.Package {
	pkg := types.NewPackage("math/cmplx", "cmplx")
	scope := pkg.Scope()

	c128 := types.Typ[types.Complex128]
	f64 := types.Typ[types.Float64]

	// Unary complex functions: complex128 → complex128
	for _, name := range []string{"Sqrt", "Exp", "Log", "Sin", "Cos", "Tan",
		"Asin", "Acos", "Atan", "Sinh", "Cosh", "Tanh",
		"Conj", "Log10", "Log2"} {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "x", c128)),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", c128)),
				false)))
	}

	// complex128 → float64
	for _, name := range []string{"Abs", "Phase"} {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "x", c128)),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", f64)),
				false)))
	}

	// func Polar(x complex128) (r, θ float64)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Polar",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", c128)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", f64),
				types.NewVar(token.NoPos, pkg, "theta", f64)),
			false)))

	// func Rect(r, θ float64) complex128
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Rect",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "r", f64),
				types.NewVar(token.NoPos, pkg, "theta", f64)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", c128)),
			false)))

	// func Pow(x, y complex128) complex128
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pow",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x", c128),
				types.NewVar(token.NoPos, pkg, "y", c128)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", c128)),
			false)))

	// func Inf() complex128
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Inf",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", c128)),
			false)))

	// func NaN() complex128
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NaN",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", c128)),
			false)))

	// func IsNaN(x complex128) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsNaN",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", c128)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsInf(x complex128) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsInf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", c128)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildNetSMTPPackage() *types.Package {
	pkg := types.NewPackage("net/smtp", "smtp")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()

	// type Auth interface { ... }
	authIface := types.NewInterfaceType(nil, nil)
	authIface.Complete()
	authType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Auth", nil),
		authIface, nil)
	scope.Insert(authType.Obj())

	// type Client struct { ... }
	clientStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	clientType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Client", nil),
		clientStruct, nil)
	scope.Insert(clientType.Obj())
	clientPtr := types.NewPointer(clientType)

	// func SendMail(addr string, a Auth, from string, to []string, msg []byte) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "SendMail",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "addr", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "a", authType),
				types.NewVar(token.NoPos, pkg, "from", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "to", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, pkg, "msg", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func PlainAuth(identity, username, password, host string) Auth
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PlainAuth",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "identity", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "username", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "password", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "host", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", authType)),
			false)))

	// func CRAMMD5Auth(username, secret string) Auth
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CRAMMD5Auth",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "username", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "secret", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", authType)),
			false)))

	// func Dial(addr string) (*Client, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dial",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "addr", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", clientPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Client methods
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Mail",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "from", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Rcpt",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "to", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Quit",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildNetRPCPackage() *types.Package {
	pkg := types.NewPackage("net/rpc", "rpc")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()

	// type Client struct { ... }
	clientStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	clientType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Client", nil),
		clientStruct, nil)
	scope.Insert(clientType.Obj())
	clientPtr := types.NewPointer(clientType)

	// type Server struct { ... }
	serverStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	serverType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Server", nil),
		serverStruct, nil)
	scope.Insert(serverType.Obj())

	// var DefaultServer *Server
	scope.Insert(types.NewVar(token.NoPos, pkg, "DefaultServer", types.NewPointer(serverType)))
	// var ErrShutdown error
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrShutdown", errType))

	anyType := types.NewInterfaceType(nil, nil)

	// func Dial(network, address string) (*Client, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dial",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", clientPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func DialHTTP(network, address string) (*Client, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DialHTTP",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", clientPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func NewServer() *Server
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewServer",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(serverType))),
			false)))

	// func Register(rcvr any) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Register",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "rcvr", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Client.Call(serviceMethod string, args any, reply any) error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Call",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "client", clientPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "serviceMethod", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "args", anyType),
				types.NewVar(token.NoPos, nil, "reply", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// Client.Close() error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "client", clientPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildTextTemplateParsePackage() *types.Package {
	pkg := types.NewPackage("text/template/parse", "parse")
	scope := pkg.Scope()

	// type Node interface { ... }
	nodeIface := types.NewInterfaceType(nil, nil)
	nodeIface.Complete()
	nodeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Node", nil),
		nodeIface, nil)
	scope.Insert(nodeType.Obj())

	// type Tree struct { ... }
	treeStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
	}, nil)
	treeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Tree", nil),
		treeStruct, nil)
	scope.Insert(treeType.Obj())

	// type NodeType int
	nodeTypeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NodeType", nil),
		types.Typ[types.Int], nil)
	scope.Insert(nodeTypeType.Obj())

	// NodeType constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "NodeText", nodeTypeType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NodeAction", nodeTypeType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NodeList", nodeTypeType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NodePipe", nodeTypeType, constant.MakeInt64(3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "NodeTemplate", nodeTypeType, constant.MakeInt64(4)))

	pkg.MarkComplete()
	return pkg
}

func buildEncodingASN1Package() *types.Package {
	pkg := types.NewPackage("encoding/asn1", "asn1")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	anyType := types.NewInterfaceType(nil, nil)

	// func Marshal(val any) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Marshal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "val", anyType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", byteSlice),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Unmarshal(b []byte, val any) (rest []byte, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Unmarshal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "b", byteSlice),
				types.NewVar(token.NoPos, pkg, "val", anyType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "rest", byteSlice),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// func MarshalWithParams(val any, params string) ([]byte, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MarshalWithParams",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "val", anyType),
				types.NewVar(token.NoPos, pkg, "params", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", byteSlice),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func UnmarshalWithParams(b []byte, val any, params string) (rest []byte, err error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "UnmarshalWithParams",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "b", byteSlice),
				types.NewVar(token.NoPos, pkg, "val", anyType),
				types.NewVar(token.NoPos, pkg, "params", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "rest", byteSlice),
				types.NewVar(token.NoPos, pkg, "err", errType)),
			false)))

	// type ObjectIdentifier []int
	oidType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ObjectIdentifier", nil),
		types.NewSlice(types.Typ[types.Int]), nil)
	scope.Insert(oidType.Obj())

	// type RawValue struct { ... }
	rawValueStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Class", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Tag", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Bytes", byteSlice, false),
	}, nil)
	rawValueType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "RawValue", nil),
		rawValueStruct, nil)
	scope.Insert(rawValueType.Obj())

	// type BitString struct { ... }
	bitStringStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Bytes", byteSlice, false),
		types.NewField(token.NoPos, pkg, "BitLength", types.Typ[types.Int], false),
	}, nil)
	bitStringType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "BitString", nil),
		bitStringStruct, nil)
	scope.Insert(bitStringType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildCryptoX509PkixPackage() *types.Package {
	pkg := types.NewPackage("crypto/x509/pkix", "pkix")
	scope := pkg.Scope()

	// type Name struct { ... }
	nameStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Country", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "Organization", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "CommonName", types.Typ[types.String], false),
	}, nil)
	nameType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Name", nil),
		nameStruct, nil)
	scope.Insert(nameType.Obj())

	// Name.String() method
	nameType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "n", nameType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))

	// type AlgorithmIdentifier struct { ... }
	algStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Algorithm", types.NewSlice(types.Typ[types.Int]), false),
	}, nil)
	algType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "AlgorithmIdentifier", nil),
		algStruct, nil)
	scope.Insert(algType.Obj())

	pkg.MarkComplete()
	return pkg
}

func buildCryptoDSAPackage() *types.Package {
	pkg := types.NewPackage("crypto/dsa", "dsa")
	scope := pkg.Scope()

	// type ParameterSizes int
	paramSizesType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ParameterSizes", nil),
		types.Typ[types.Int], nil)
	scope.Insert(paramSizesType.Obj())

	scope.Insert(types.NewConst(token.NoPos, pkg, "L1024N160", paramSizesType, constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "L2048N224", paramSizesType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "L2048N256", paramSizesType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "L3072N256", paramSizesType, constant.MakeInt64(3)))

	// type PublicKey struct { ... }
	pubStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Y", types.Typ[types.Int], false),
	}, nil)
	pubType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "PublicKey", nil),
		pubStruct, nil)
	scope.Insert(pubType.Obj())

	// type PrivateKey struct { ... }
	privStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.Int], false),
	}, nil)
	privType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "PrivateKey", nil),
		privStruct, nil)
	scope.Insert(privType.Obj())

	// var ErrInvalidPublicKey error
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrInvalidPublicKey",
		types.Universe.Lookup("error").Type()))

	pkg.MarkComplete()
	return pkg
}

func buildNetRPCJSONRPCPackage() *types.Package {
	pkg := types.NewPackage("net/rpc/jsonrpc", "jsonrpc")
	scope := pkg.Scope()

	errType := types.Universe.Lookup("error").Type()

	// func Dial(network, address string) (*rpc.Client, error) — simplified
	clientStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	clientType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Client", nil),
		clientStruct, nil)
	_ = clientType

	scope.Insert(types.NewFunc(token.NoPos, pkg, "Dial",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildCryptoPackage() *types.Package {
	pkg := types.NewPackage("crypto", "crypto")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	// type Hash uint
	hashType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Hash", nil),
		types.Typ[types.Uint], nil)
	scope.Insert(hashType.Obj())

	scope.Insert(types.NewConst(token.NoPos, pkg, "MD4", hashType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "MD5", hashType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA1", hashType, constant.MakeInt64(3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA224", hashType, constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA256", hashType, constant.MakeInt64(5)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA384", hashType, constant.MakeInt64(6)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA512", hashType, constant.MakeInt64(7)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA512_224", hashType, constant.MakeInt64(12)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA512_256", hashType, constant.MakeInt64(13)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "SHA3_256", hashType, constant.MakeInt64(11)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BLAKE2b_256", hashType, constant.MakeInt64(17)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "BLAKE2b_512", hashType, constant.MakeInt64(19)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "RIPEMD160", hashType, constant.MakeInt64(20)))

	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Available",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "h", hashType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "Size",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "h", hashType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))
	hashType.AddMethod(types.NewFunc(token.NoPos, pkg, "HashFunc",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "h", hashType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", hashType)), false)))

	signerIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Public",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewInterfaceType(nil, nil))), false)),
		types.NewFunc(token.NoPos, pkg, "Sign",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "rand", types.NewInterfaceType(nil, nil)),
					types.NewVar(token.NoPos, nil, "digest", byteSlice),
					types.NewVar(token.NoPos, nil, "opts", types.NewInterfaceType(nil, nil))),
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "", byteSlice),
					types.NewVar(token.NoPos, nil, "", errType)), false)),
	}, nil)
	signerIface.Complete()
	signerType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Signer", nil), signerIface, nil)
	scope.Insert(signerType.Obj())

	scope.Insert(types.NewTypeName(token.NoPos, pkg, "PrivateKey", types.NewInterfaceType(nil, nil)))
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "PublicKey", types.NewInterfaceType(nil, nil)))
	_ = signerType
	pkg.MarkComplete()
	return pkg
}

func buildHashAdler32Package() *types.Package {
	pkg := types.NewPackage("hash/adler32", "adler32")
	scope := pkg.Scope()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Checksum",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint32])), false)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int], constant.MakeInt64(4)))
	pkg.MarkComplete()
	return pkg
}

func buildHashCRC64Package() *types.Package {
	pkg := types.NewPackage("hash/crc64", "crc64")
	scope := pkg.Scope()
	tableType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Table", nil),
		types.NewArray(types.Typ[types.Uint64], 256), nil)
	scope.Insert(tableType.Obj())
	tablePtr := types.NewPointer(tableType)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "tab", tablePtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeTable",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "poly", types.Typ[types.Uint64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", tablePtr)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Checksum",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "data", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "tab", tablePtr)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])), false)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "Size", types.Typ[types.Int], constant.MakeInt64(8)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ISO", types.Typ[types.Uint64], constant.MakeUint64(0xD800000000000000)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ECMA", types.Typ[types.Uint64], constant.MakeUint64(0x42F0E1EBA9EA3693)))
	pkg.MarkComplete()
	return pkg
}

func buildEncodingPackage() *types.Package {
	pkg := types.NewPackage("encoding", "encoding")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])

	for _, name := range []string{"BinaryMarshaler", "TextMarshaler"} {
		method := "MarshalBinary"
		if name == "TextMarshaler" {
			method = "MarshalText"
		}
		iface := types.NewInterfaceType([]*types.Func{
			types.NewFunc(token.NoPos, pkg, method,
				types.NewSignatureType(nil, nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice),
						types.NewVar(token.NoPos, nil, "", errType)), false)),
		}, nil)
		iface.Complete()
		t := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), iface, nil)
		scope.Insert(t.Obj())
	}
	for _, name := range []string{"BinaryUnmarshaler", "TextUnmarshaler"} {
		method := "UnmarshalBinary"
		if name == "TextUnmarshaler" {
			method = "UnmarshalText"
		}
		iface := types.NewInterfaceType([]*types.Func{
			types.NewFunc(token.NoPos, pkg, method,
				types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "data", byteSlice)),
					types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)), false)),
		}, nil)
		iface.Complete()
		t := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), iface, nil)
		scope.Insert(t.Obj())
	}
	pkg.MarkComplete()
	return pkg
}

func buildGoConstantPackage() *types.Package {
	pkg := types.NewPackage("go/constant", "constant")
	scope := pkg.Scope()
	kindType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Kind", nil), types.Typ[types.Int], nil)
	scope.Insert(kindType.Obj())
	for _, kv := range []struct {
		name string
		val  int64
	}{{"Unknown", 0}, {"Bool", 1}, {"String", 2}, {"Int", 3}, {"Float", 4}, {"Complex", 5}} {
		scope.Insert(types.NewConst(token.NoPos, pkg, kv.name, kindType, constant.MakeInt64(kv.val)))
	}
	valueIface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Kind",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", kindType)), false)),
		types.NewFunc(token.NoPos, pkg, "String",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
		types.NewFunc(token.NoPos, pkg, "ExactString",
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
	}, nil)
	valueIface.Complete()
	valueType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Value", nil), valueIface, nil)
	scope.Insert(valueType.Obj())
	for _, name := range []string{"MakeBool"} {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "b", types.Typ[types.Bool])),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueType)), false)))
	}
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeString",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueType)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeInt64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueType)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MakeFloat64",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Float64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueType)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "BoolVal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", valueType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "StringVal",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", valueType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int64Val",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", valueType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, pkg, "exact", types.Typ[types.Bool])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64Val",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", valueType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64]),
				types.NewVar(token.NoPos, pkg, "exact", types.Typ[types.Bool])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Compare",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "x_", valueType),
				types.NewVar(token.NoPos, pkg, "op", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "y_", valueType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	pkg.MarkComplete()
	return pkg
}

func buildRuntimeTracePackage() *types.Package {
	pkg := types.NewPackage("runtime/trace", "trace")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Start",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Stop",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsEnabled",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	taskType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Task", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(taskType.Obj())
	regionType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Region", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(regionType.Obj())
	pkg.MarkComplete()
	return pkg
}

func buildCryptoECDHPackage() *types.Package {
	pkg := types.NewPackage("crypto/ecdh", "ecdh")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	curveType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Curve", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(curveType.Obj())
	for _, name := range []string{"P256", "P384", "P521", "X25519"} {
		scope.Insert(types.NewFunc(token.NoPos, pkg, name,
			types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", curveType)), false)))
	}
	privType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "PrivateKey", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(privType.Obj())
	privPtr := types.NewPointer(privType)
	pubType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "PublicKey", nil), types.NewStruct(nil, nil), nil)
	scope.Insert(pubType.Obj())
	pubPtr := types.NewPointer(pubType)
	privType.AddMethod(types.NewFunc(token.NoPos, pkg, "PublicKey",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "k", privPtr), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", pubPtr)), false)))
	privType.AddMethod(types.NewFunc(token.NoPos, pkg, "Bytes",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "k", privPtr), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice)), false)))
	privType.AddMethod(types.NewFunc(token.NoPos, pkg, "ECDH",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "k", privPtr), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "remote", pubPtr)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice), types.NewVar(token.NoPos, nil, "", errType)), false)))
	pubType.AddMethod(types.NewFunc(token.NoPos, pkg, "Bytes",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "k", pubPtr), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice)), false)))
	pkg.MarkComplete()
	return pkg
}

func buildGoScannerPackage() *types.Package {
	pkg := types.NewPackage("go/scanner", "scanner")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Error struct { Pos token.Position; Msg string }
	errorStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Msg", types.Typ[types.String], false),
	}, nil)
	errorType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Error", nil), errorStruct, nil)
	scope.Insert(errorType.Obj())
	errorType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", errorType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)))

	// type ErrorList []*Error
	errorListType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ErrorList", nil),
		types.NewSlice(types.NewPointer(errorType)), nil)
	scope.Insert(errorListType.Obj())
	errorListType.AddMethod(types.NewFunc(token.NoPos, pkg, "Len",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "p", errorListType), nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])), false)))

	// type Scanner struct { ErrorCount int }
	scannerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ErrorCount", types.Typ[types.Int], false),
	}, nil)
	scannerType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Scanner", nil), scannerStruct, nil)
	scope.Insert(scannerType.Obj())

	// type Mode uint
	modeType := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Mode", nil), types.Typ[types.Uint], nil)
	scope.Insert(modeType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "ScanComments", modeType, constant.MakeInt64(1)))

	// ErrorHandler type alias
	_ = errType

	pkg.MarkComplete()
	return pkg
}

func buildMathRandV2Package() *types.Package {
	pkg := types.NewPackage("math/rand/v2", "rand")
	scope := pkg.Scope()
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IntN",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int64",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Int64N",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int64])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int64])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Uint32",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint32])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Uint64",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float32",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float32])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Float64",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "N",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])), false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Shuffle",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "swap", types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int]),
						types.NewVar(token.NoPos, nil, "j", types.Typ[types.Int])), nil, false))),
			nil, false)))
	pkg.MarkComplete()
	return pkg
}
