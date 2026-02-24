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

	// func (c *Client) Hello(localName string) error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Hello",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "localName", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) Auth(a Auth) error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Auth",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "a", authType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) StartTLS(config *tls.Config) error — simplified
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "StartTLS",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "config", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) Data() (io.WriteCloser, error) — simplified
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Data",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) Extension(ext string) (bool, string)
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Extension",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "ext", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]),
				types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))

	// func (c *Client) Reset() error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Reset",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) Noop() error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Noop",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (c *Client) Verify(addr string) error
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Verify",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "c", clientPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "addr", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func NewClient(conn net.Conn, host string) (*Client, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewClient",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "conn", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "host", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", clientPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type ServerInfo struct
	serverInfoStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "TLS", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "Auth", types.NewSlice(types.Typ[types.String]), false),
	}, nil)
	serverInfoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ServerInfo", nil),
		serverInfoStruct, nil)
	scope.Insert(serverInfoType.Obj())

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

	// type Call struct
	callStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ServiceMethod", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Args", anyType, false),
		types.NewField(token.NoPos, pkg, "Reply", anyType, false),
		types.NewField(token.NoPos, pkg, "Error", errType, false),
	}, nil)
	callType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Call", nil),
		callStruct, nil)
	scope.Insert(callType.Obj())
	callPtr := types.NewPointer(callType)

	// Client.Go(serviceMethod string, args any, reply any, done chan *Call) *Call
	clientType.AddMethod(types.NewFunc(token.NoPos, pkg, "Go",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "client", clientPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "serviceMethod", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "args", anyType),
				types.NewVar(token.NoPos, nil, "reply", anyType),
				types.NewVar(token.NoPos, nil, "done", types.NewChan(types.SendRecv, callPtr))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", callPtr)),
			false)))

	// func RegisterName(name string, rcvr any) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RegisterName",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "rcvr", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func DialHTTPPath(network, address, path string) (*Client, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DialHTTPPath",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "network", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "address", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", clientPtr),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	serverPtr := types.NewPointer(serverType)

	// Server methods
	// func (s *Server) Register(rcvr any) error
	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "Register",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "server", serverPtr),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "rcvr", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (s *Server) RegisterName(name string, rcvr any) error
	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "RegisterName",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "server", serverPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "rcvr", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", errType)),
			false)))

	// func (s *Server) HandleHTTP(rpcPath, debugPath string)
	serverType.AddMethod(types.NewFunc(token.NoPos, pkg, "HandleHTTP",
		types.NewSignatureType(
			types.NewVar(token.NoPos, nil, "server", serverPtr),
			nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "rpcPath", types.Typ[types.String]),
				types.NewVar(token.NoPos, nil, "debugPath", types.Typ[types.String])),
			nil, false)))

	// func HandleHTTP() — package-level convenience
	scope.Insert(types.NewFunc(token.NoPos, pkg, "HandleHTTP",
		types.NewSignatureType(nil, nil, nil, nil, nil, false)))

	// func Accept(lis net.Listener) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Accept",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "lis", types.Typ[types.Int])),
			nil, false)))

	// func ServeConn(conn io.ReadWriteCloser) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ServeConn",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "conn", types.Typ[types.Int])),
			nil, false)))

	// type ServerError string
	serverErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ServerError", nil),
		types.Typ[types.String], nil)
	serverErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", serverErrType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(serverErrType.Obj())

	// const DefaultRPCPath, DefaultDebugPath
	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultRPCPath", types.Typ[types.String],
		constant.MakeString("/_goRPC_")))
	scope.Insert(types.NewConst(token.NoPos, pkg, "DefaultDebugPath", types.Typ[types.String],
		constant.MakeString("/debug/rpc")))

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

	// BitString methods
	// func (b BitString) At(i int) int
	bitStringType.AddMethod(types.NewFunc(token.NoPos, pkg, "At",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "b", bitStringType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
			false)))
	// func (b BitString) RightAlign() []byte
	bitStringType.AddMethod(types.NewFunc(token.NoPos, pkg, "RightAlign",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "b", bitStringType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", byteSlice)),
			false)))

	// ObjectIdentifier methods
	// func (oi ObjectIdentifier) Equal(other ObjectIdentifier) bool
	oidType.AddMethod(types.NewFunc(token.NoPos, pkg, "Equal",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "oi", oidType),
			nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "other", oidType)),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
			false)))
	// func (oi ObjectIdentifier) String() string
	oidType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "oi", oidType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))

	// type Flag struct { ... }
	flagStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	flagType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Flag", nil),
		flagStruct, nil)
	scope.Insert(flagType.Obj())

	// type Enumerated int
	enumType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Enumerated", nil),
		types.Typ[types.Int], nil)
	scope.Insert(enumType.Obj())

	// type SyntaxError struct { Msg string }
	syntaxErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Msg", types.Typ[types.String], false),
	}, nil)
	syntaxErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "SyntaxError", nil),
		syntaxErrStruct, nil)
	syntaxErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", syntaxErrType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(syntaxErrType.Obj())

	// type StructuralError struct { Msg string }
	structuralErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Msg", types.Typ[types.String], false),
	}, nil)
	structuralErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "StructuralError", nil),
		structuralErrStruct, nil)
	structuralErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "e", structuralErrType),
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false)))
	scope.Insert(structuralErrType.Obj())

	// Tag constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagBoolean", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagInteger", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagBitString", types.Typ[types.Int], constant.MakeInt64(3)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagOctetString", types.Typ[types.Int], constant.MakeInt64(4)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagNULL", types.Typ[types.Int], constant.MakeInt64(5)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagOID", types.Typ[types.Int], constant.MakeInt64(6)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagEnum", types.Typ[types.Int], constant.MakeInt64(10)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagUTF8String", types.Typ[types.Int], constant.MakeInt64(12)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagSequence", types.Typ[types.Int], constant.MakeInt64(16)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagSet", types.Typ[types.Int], constant.MakeInt64(17)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagPrintableString", types.Typ[types.Int], constant.MakeInt64(19)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagIA5String", types.Typ[types.Int], constant.MakeInt64(22)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagUTCTime", types.Typ[types.Int], constant.MakeInt64(23)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagGeneralizedTime", types.Typ[types.Int], constant.MakeInt64(24)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "TagBMPString", types.Typ[types.Int], constant.MakeInt64(30)))

	// Class constants
	scope.Insert(types.NewConst(token.NoPos, pkg, "ClassUniversal", types.Typ[types.Int], constant.MakeInt64(0)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ClassApplication", types.Typ[types.Int], constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ClassContextSpecific", types.Typ[types.Int], constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "ClassPrivate", types.Typ[types.Int], constant.MakeInt64(3)))

	// var NullRawValue, NullBytes
	scope.Insert(types.NewVar(token.NoPos, pkg, "NullRawValue", rawValueType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "NullBytes", byteSlice))

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

func buildDatabaseSQLDriverPackage() *types.Package {
	pkg := types.NewPackage("database/sql/driver", "driver")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()
	anyType := types.NewInterfaceType(nil, nil)

	// type Value interface{}
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Value", anyType))

	// type NamedValue struct
	namedValueStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Ordinal", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Value", anyType, false),
	}, nil)
	namedValueType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NamedValue", nil),
		namedValueStruct, nil)
	scope.Insert(namedValueType.Obj())

	// type IsolationLevel int
	isolationType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "IsolationLevel", nil),
		types.Typ[types.Int], nil)
	scope.Insert(isolationType.Obj())

	// type TxOptions struct
	txOptsStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Isolation", isolationType, false),
		types.NewField(token.NoPos, pkg, "ReadOnly", types.Typ[types.Bool], false),
	}, nil)
	txOptsType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "TxOptions", nil),
		txOptsStruct, nil)
	scope.Insert(txOptsType.Obj())

	// Interfaces: Driver, Conn, Stmt, Rows, Tx, Result, etc.
	for _, name := range []string{
		"Driver", "DriverContext", "Conn", "ConnPrepareContext",
		"ConnBeginTx", "Stmt", "StmtExecContext", "StmtQueryContext",
		"Rows", "RowsNextResultSet", "Tx", "Result",
		"Execer", "ExecerContext", "Queryer", "QueryerContext",
		"Pinger", "SessionResetter", "Validator", "Connector",
		"RowsColumnTypeScanType", "RowsColumnTypeDatabaseTypeName",
		"RowsColumnTypeLength", "RowsColumnTypeNullable",
		"RowsColumnTypePrecisionScale",
		"ValueConverter", "Valuer",
	} {
		iface := types.NewInterfaceType(nil, nil)
		iface.Complete()
		scope.Insert(types.NewTypeName(token.NoPos, pkg, name, iface))
	}

	// type NotNull, Null structs
	for _, name := range []string{"NotNull", "Null"} {
		s := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "Converter", anyType, false),
		}, nil)
		t := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), s, nil)
		scope.Insert(t.Obj())
	}

	// var Int32, String, Bool, DefaultParameterConverter
	for _, name := range []string{"Int32", "String", "Bool", "DefaultParameterConverter"} {
		scope.Insert(types.NewVar(token.NoPos, pkg, name, anyType))
	}

	// var ErrSkip, ErrBadConn, ErrRemoveArgument error
	for _, name := range []string{"ErrSkip", "ErrBadConn", "ErrRemoveArgument"} {
		scope.Insert(types.NewVar(token.NoPos, pkg, name, errType))
	}

	// func IsScanValue(v Value) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsScanValue",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsValue(v any) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsValue",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "v", anyType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

func buildGoDocPackage() *types.Package {
	pkg := types.NewPackage("go/doc", "doc")
	scope := pkg.Scope()

	// type Package struct
	pkgStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "ImportPath", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Doc", types.Typ[types.String], false),
	}, nil)
	pkgType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Package", nil),
		pkgStruct, nil)
	scope.Insert(pkgType.Obj())

	// type Type, Func, Value, Note structs
	for _, name := range []string{"Type", "Func", "Value", "Note"} {
		s := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
			types.NewField(token.NoPos, pkg, "Doc", types.Typ[types.String], false),
		}, nil)
		t := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), s, nil)
		scope.Insert(t.Obj())
	}

	// type Mode int
	modeType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Mode", nil),
		types.Typ[types.Int], nil)
	scope.Insert(modeType.Obj())
	scope.Insert(types.NewConst(token.NoPos, pkg, "AllDecls", modeType, constant.MakeInt64(1)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "AllMethods", modeType, constant.MakeInt64(2)))
	scope.Insert(types.NewConst(token.NoPos, pkg, "PreserveAST", modeType, constant.MakeInt64(4)))

	// func New(...) *Package — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "New",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "pkg_", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "importPath", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "mode", modeType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(pkgType))),
			false)))

	// func Synopsis(text string) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Synopsis",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "text", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	// func ToHTML / ToText — no-op stubs
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ToHTML",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "text", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "words", types.Typ[types.Int])),
			nil, false)))

	scope.Insert(types.NewFunc(token.NoPos, pkg, "ToText",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "w", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "text", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "indent", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "preIndent", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "width", types.Typ[types.Int])),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// net/netip
// ============================================================

func buildNetNetipPackage() *types.Package {
	pkg := types.NewPackage("net/netip", "netip")
	scope := pkg.Scope()

	// type Addr struct { ... }
	addrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "hi", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "lo", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "z", types.Typ[types.Int], false),
	}, nil)
	addrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Addr", nil),
		addrStruct, nil)
	scope.Insert(addrType.Obj())

	// type AddrPort struct { ... }
	addrPortStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ip", addrType, false),
		types.NewField(token.NoPos, pkg, "port", types.Typ[types.Uint16], false),
	}, nil)
	addrPortType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "AddrPort", nil),
		addrPortStruct, nil)
	scope.Insert(addrPortType.Obj())

	// type Prefix struct { ... }
	prefixStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "ip", addrType, false),
		types.NewField(token.NoPos, pkg, "bits", types.Typ[types.Int], false),
	}, nil)
	prefixType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Prefix", nil),
		prefixStruct, nil)
	scope.Insert(prefixType.Obj())

	errType := types.Universe.Lookup("error").Type()

	// Addr constructors
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddrFrom4",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "addr", types.NewArray(types.Typ[types.Byte], 4))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddrFrom16",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "addr", types.NewArray(types.Typ[types.Byte], 16))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddrFromSlice",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "slice", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", addrType),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MustParseAddr",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseAddr",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", addrType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IPv4Unspecified",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IPv6Unspecified",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IPv6LinkLocalAllNodes",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IPv6Loopback",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))

	// Addr methods
	addrMethods := []struct{ name string; ret *types.Tuple }{
		{"IsValid", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"Is4", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"Is6", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"Is4In6", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsLoopback", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsMulticast", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsPrivate", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsGlobalUnicast", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsLinkLocalUnicast", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsLinkLocalMulticast", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsInterfaceLocalMulticast", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"IsUnspecified", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool]))},
		{"BitLen", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]))},
		{"Zone", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]))},
		{"String", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]))},
		{"StringExpanded", types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]))},
	}
	for _, m := range addrMethods {
		addrType.AddMethod(types.NewFunc(token.NoPos, pkg, m.name,
			types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
				nil, m.ret, false)))
	}
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "As4",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewArray(types.Typ[types.Byte], 4))),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "As16",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewArray(types.Typ[types.Byte], 16))),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "AsSlice",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Unmap",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "WithZone",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "zone", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "MarshalText",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "MarshalBinary",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Prev",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Next",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Prefix",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "b", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", prefixType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Compare",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ip2", addrType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))
	addrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Less",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ip2", addrType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// AddrPort constructors
	scope.Insert(types.NewFunc(token.NoPos, pkg, "AddrPortFrom",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ip", addrType),
				types.NewVar(token.NoPos, pkg, "port", types.Typ[types.Uint16])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrPortType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MustParseAddrPort",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrPortType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParseAddrPort",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", addrPortType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// AddrPort methods
	addrPortType.AddMethod(types.NewFunc(token.NoPos, pkg, "Addr",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrPortType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)), false)))
	addrPortType.AddMethod(types.NewFunc(token.NoPos, pkg, "Port",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrPortType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint16])), false)))
	addrPortType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsValid",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrPortType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	addrPortType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", addrPortType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))

	// Prefix constructors
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PrefixFrom",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ip", addrType),
				types.NewVar(token.NoPos, pkg, "bits", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", prefixType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "MustParsePrefix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", prefixType)),
			false)))
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ParsePrefix",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "s", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", prefixType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Prefix methods
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "Addr",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", addrType)), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "Bits",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsValid",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "Contains",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ip", addrType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "Overlaps",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "o", prefixType)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "Masked",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", prefixType)), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "String",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))
	prefixType.AddMethod(types.NewFunc(token.NoPos, pkg, "IsSingleIP",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", prefixType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])), false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// iter
// ============================================================

func buildIterPackage() *types.Package {
	pkg := types.NewPackage("iter", "iter")
	scope := pkg.Scope()

	// type Seq[V any] func(yield func(V) bool) — simplified as func(func(int) bool)
	seqType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Seq", nil),
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "yield",
				types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
					types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
					false))),
			nil, false), nil)
	scope.Insert(seqType.Obj())

	// type Seq2[K, V any] func(yield func(K, V) bool) — simplified as func(func(int, int) bool)
	seq2Type := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Seq2", nil),
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "yield",
				types.NewSignatureType(nil, nil, nil,
					types.NewTuple(
						types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
						types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
					types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
					false))),
			nil, false), nil)
	scope.Insert(seq2Type.Obj())

	// func Pull[V any](seq Seq[V]) (next func() (V, bool), stop func()) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pull",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "seq", seqType)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "next",
					types.NewSignatureType(nil, nil, nil, nil,
						types.NewTuple(
							types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
							types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
						false)),
				types.NewVar(token.NoPos, pkg, "stop",
					types.NewSignatureType(nil, nil, nil, nil, nil, false))),
			false)))

	// func Pull2[K, V any](seq Seq2[K, V]) (next func() (K, V, bool), stop func()) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Pull2",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "seq", seq2Type)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "next",
					types.NewSignatureType(nil, nil, nil, nil,
						types.NewTuple(
							types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
							types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
							types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
						false)),
				types.NewVar(token.NoPos, pkg, "stop",
					types.NewSignatureType(nil, nil, nil, nil, nil, false))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// unique
// ============================================================

func buildUniquePackage() *types.Package {
	pkg := types.NewPackage("unique", "unique")
	scope := pkg.Scope()

	// type Handle[T comparable] struct { ... } — simplified as struct with value
	handleStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "value", types.Typ[types.Int], false),
	}, nil)
	handleType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Handle", nil),
		handleStruct, nil)
	scope.Insert(handleType.Obj())

	// func Make[T comparable](value T) Handle[T] — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Make",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", handleType)),
			false)))

	// Handle.Value() T
	handleType.AddMethod(types.NewFunc(token.NoPos, pkg, "Value",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", handleType), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// testing/quick
// ============================================================

func buildTestingQuickPackage() *types.Package {
	pkg := types.NewPackage("testing/quick", "quick")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Config struct { ... }
	configStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "MaxCount", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "MaxCountScale", types.Typ[types.Float64], false),
		types.NewField(token.NoPos, pkg, "Rand", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Values", types.Typ[types.Int], false),
	}, nil)
	configType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Config", nil),
		configStruct, nil)
	scope.Insert(configType.Obj())

	// type CheckError struct { ... }
	checkErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Count", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "In", types.NewSlice(types.NewInterfaceType(nil, nil)), false),
	}, nil)
	checkErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "CheckError", nil),
		checkErrStruct, nil)
	scope.Insert(checkErrType.Obj())
	checkErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(checkErrType)), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))

	// type CheckEqualError struct { ... }
	checkEqErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "CheckError", checkErrType, true),
		types.NewField(token.NoPos, pkg, "Out1", types.NewSlice(types.NewInterfaceType(nil, nil)), false),
		types.NewField(token.NoPos, pkg, "Out2", types.NewSlice(types.NewInterfaceType(nil, nil)), false),
	}, nil)
	checkEqErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "CheckEqualError", nil),
		checkEqErrStruct, nil)
	scope.Insert(checkEqErrType.Obj())
	checkEqErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(checkEqErrType)), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))

	// func Check(f any, config *Config) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Check",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "f", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "config", types.NewPointer(configType))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func CheckEqual(f, g any, config *Config) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "CheckEqual",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "f", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "g", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "config", types.NewPointer(configType))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Value(t reflect.Type, rand *rand.Rand) (reflect.Value, bool) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Value",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "t", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "rand", types.Typ[types.Int])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// testing/slogtest
// ============================================================

func buildTestingSlogtestPackage() *types.Package {
	pkg := types.NewPackage("testing/slogtest", "slogtest")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func Run(t *testing.T, newHandler func(*testing.T) slog.Handler, opts ...Option) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Run",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "t", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "newHandler", types.Typ[types.Int])),
			nil, false)))

	// func TestHandler(h slog.Handler, results func() []map[string]any) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "TestHandler",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "h", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "results", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// go/build/constraint
// ============================================================

func buildGoBuildConstraintPackage() *types.Package {
	pkg := types.NewPackage("go/build/constraint", "constraint")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Expr interface { ... }
	exprIface := types.NewInterfaceType(nil, nil)
	exprIface.Complete()
	exprType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Expr", nil),
		exprIface, nil)
	scope.Insert(exprType.Obj())

	// type TagExpr struct { Tag string }
	tagExprStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Tag", types.Typ[types.String], false),
	}, nil)
	tagExprType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "TagExpr", nil),
		tagExprStruct, nil)
	scope.Insert(tagExprType.Obj())

	// type NotExpr struct { X Expr }
	notExprStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", exprIface, false),
	}, nil)
	notExprType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NotExpr", nil),
		notExprStruct, nil)
	scope.Insert(notExprType.Obj())

	// type AndExpr struct { X, Y Expr }
	andExprStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", exprIface, false),
		types.NewField(token.NoPos, pkg, "Y", exprIface, false),
	}, nil)
	andExprType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "AndExpr", nil),
		andExprStruct, nil)
	scope.Insert(andExprType.Obj())

	// type OrExpr struct { X, Y Expr }
	orExprStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", exprIface, false),
		types.NewField(token.NoPos, pkg, "Y", exprIface, false),
	}, nil)
	orExprType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "OrExpr", nil),
		orExprStruct, nil)
	scope.Insert(orExprType.Obj())

	// type SyntaxError struct { ... }
	syntaxErrStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Offset", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Err", types.Typ[types.String], false),
	}, nil)
	syntaxErrType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "SyntaxError", nil),
		syntaxErrStruct, nil)
	scope.Insert(syntaxErrType.Obj())
	syntaxErrType.AddMethod(types.NewFunc(token.NoPos, pkg, "Error",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(syntaxErrType)), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])), false)))

	// func Parse(line string) (Expr, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Parse",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "line", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", exprIface),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func IsGoBuild(line string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsGoBuild",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "line", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func IsPlusBuild(line string) bool
	scope.Insert(types.NewFunc(token.NoPos, pkg, "IsPlusBuild",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "line", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	// func PlusBuildLines(x Expr) ([]string, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "PlusBuildLines",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", exprIface)),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.String])),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func GoVersion(x Expr) string
	scope.Insert(types.NewFunc(token.NoPos, pkg, "GoVersion",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "x", exprIface)),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// go/doc/comment
// ============================================================

func buildGoDocCommentPackage() *types.Package {
	pkg := types.NewPackage("go/doc/comment", "comment")
	scope := pkg.Scope()

	// type Doc struct { Content []Block; Links []*LinkDef }
	docStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Content", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Links", types.Typ[types.Int], false),
	}, nil)
	docType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Doc", nil),
		docStruct, nil)
	scope.Insert(docType.Obj())

	// type Parser struct { ... }
	parserStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Words", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "LookupPackage", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "LookupSym", types.Typ[types.Int], false),
	}, nil)
	parserType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Parser", nil),
		parserStruct, nil)
	scope.Insert(parserType.Obj())

	// Parser.Parse(text string) *Doc
	parserType.AddMethod(types.NewFunc(token.NoPos, pkg, "Parse",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(parserType)), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "text", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(docType))),
			false)))

	// type Printer struct { ... }
	printerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "DocLinkURL", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "DocLinkBaseURL", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "HeadingLevel", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "HeadingID", types.Typ[types.Int], false),
	}, nil)
	printerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Printer", nil),
		printerStruct, nil)
	scope.Insert(printerType.Obj())

	// Printer methods
	for _, m := range []string{"HTML", "Markdown", "Text", "Comment"} {
		printerType.AddMethod(types.NewFunc(token.NoPos, pkg, m,
			types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(printerType)), nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, pkg, "d", types.NewPointer(docType))),
				types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(types.Typ[types.Byte]))),
				false)))
	}

	// type LinkDef, DocLink, etc.
	linkDefStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Text", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "URL", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Used", types.Typ[types.Bool], false),
	}, nil)
	linkDefType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "LinkDef", nil),
		linkDefStruct, nil)
	scope.Insert(linkDefType.Obj())

	// DefaultLookupPackage
	scope.Insert(types.NewFunc(token.NoPos, pkg, "DefaultLookupPackage",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "name", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Bool])),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// go/importer
// ============================================================

func buildGoImporterPackage() *types.Package {
	pkg := types.NewPackage("go/importer", "importer")
	scope := pkg.Scope()

	// func Default() types.Importer — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Default",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// func For(compiler string, lookup Lookup) types.Importer — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "For",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "compiler", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "lookup", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// func ForCompiler(fset *token.FileSet, compiler string, lookup Lookup) types.Importer — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ForCompiler",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "fset", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "compiler", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "lookup", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// mime/quotedprintable
// ============================================================

func buildMimeQuotedprintablePackage() *types.Package {
	pkg := types.NewPackage("mime/quotedprintable", "quotedprintable")
	scope := pkg.Scope()

	// type Reader struct { ... }
	readerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	readerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Reader", nil),
		readerStruct, nil)
	scope.Insert(readerType.Obj())

	// func NewReader(r io.Reader) *Reader
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewReader",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(readerType))),
			false)))

	// Reader.Read(p []byte) (int, error)
	errType := types.Universe.Lookup("error").Type()
	readerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(readerType)), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// type Writer struct { Binary bool }
	writerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Binary", types.Typ[types.Bool], false),
	}, nil)
	writerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Writer", nil),
		writerStruct, nil)
	scope.Insert(writerType.Obj())

	// func NewWriter(w io.Writer) *Writer
	scope.Insert(types.NewFunc(token.NoPos, pkg, "NewWriter",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(writerType))),
			false)))

	// Writer.Write(p []byte) (int, error)
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Write",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(writerType)), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "p", types.NewSlice(types.Typ[types.Byte]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Writer.Close() error
	writerType.AddMethod(types.NewFunc(token.NoPos, pkg, "Close",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(writerType)), nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// net/http/httptrace
// ============================================================

func buildNetHTTPHttptracePackage() *types.Package {
	pkg := types.NewPackage("net/http/httptrace", "httptrace")
	scope := pkg.Scope()

	// type ClientTrace struct { ... }
	clientTraceStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "GetConn", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "GotConn", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "PutIdleConn", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "GotFirstResponseByte", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Got100Continue", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Got1xxResponse", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "DNSStart", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "DNSDone", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "ConnectStart", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "ConnectDone", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "TLSHandshakeStart", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "TLSHandshakeDone", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "WroteHeaderField", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "WroteHeaders", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Wait100Continue", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "WroteRequest", types.Typ[types.Int], false),
	}, nil)
	clientTraceType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ClientTrace", nil),
		clientTraceStruct, nil)
	scope.Insert(clientTraceType.Obj())

	// type GotConnInfo struct { ... }
	gotConnInfoStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Conn", types.NewInterfaceType(nil, nil), false),
		types.NewField(token.NoPos, pkg, "Reused", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "WasIdle", types.Typ[types.Bool], false),
		types.NewField(token.NoPos, pkg, "IdleTime", types.Typ[types.Int64], false),
	}, nil)
	gotConnInfoType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "GotConnInfo", nil),
		gotConnInfoStruct, nil)
	scope.Insert(gotConnInfoType.Obj())

	// type DNSStartInfo struct { Host string }
	dnsStartStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Host", types.Typ[types.String], false),
	}, nil)
	dnsStartType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "DNSStartInfo", nil),
		dnsStartStruct, nil)
	scope.Insert(dnsStartType.Obj())

	// type DNSDoneInfo struct { Addrs []net.IPAddr; Err error }
	dnsDoneStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Addrs", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Err", types.Universe.Lookup("error").Type(), false),
		types.NewField(token.NoPos, pkg, "Coalesced", types.Typ[types.Bool], false),
	}, nil)
	dnsDoneType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "DNSDoneInfo", nil),
		dnsDoneStruct, nil)
	scope.Insert(dnsDoneType.Obj())

	// type WroteRequestInfo struct { Err error }
	wroteReqStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Err", types.Universe.Lookup("error").Type(), false),
	}, nil)
	wroteReqType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "WroteRequestInfo", nil),
		wroteReqStruct, nil)
	scope.Insert(wroteReqType.Obj())

	// func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WithClientTrace",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "ctx", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "trace", types.NewPointer(clientTraceType))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil))),
			false)))

	// func ContextClientTrace(ctx context.Context) *ClientTrace
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ContextClientTrace",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "ctx", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewPointer(clientTraceType))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// net/http/cgi
// ============================================================

func buildNetHTTPCgiPackage() *types.Package {
	pkg := types.NewPackage("net/http/cgi", "cgi")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Handler struct { ... }
	handlerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Path", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Root", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Dir", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Env", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "InheritEnv", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "Logger", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Args", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(token.NoPos, pkg, "Stderr", types.NewInterfaceType(nil, nil), false),
	}, nil)
	handlerType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Handler", nil),
		handlerStruct, nil)
	scope.Insert(handlerType.Obj())

	// Handler.ServeHTTP(rw http.ResponseWriter, req *http.Request) — simplified
	handlerType.AddMethod(types.NewFunc(token.NoPos, pkg, "ServeHTTP",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(handlerType)), nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "rw", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "req", types.Typ[types.Int])),
			nil, false)))

	// func Request() (*http.Request, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Request",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func RequestFromMap(params map[string]string) (*http.Request, error) — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "RequestFromMap",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "params",
				types.NewMap(types.Typ[types.String], types.Typ[types.String]))),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func Serve(handler http.Handler) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Serve",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "handler", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// net/http/fcgi
// ============================================================

func buildNetHTTPFcgiPackage() *types.Package {
	pkg := types.NewPackage("net/http/fcgi", "fcgi")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// var ErrRequestAborted, ErrConnClosed
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrRequestAborted", errType))
	scope.Insert(types.NewVar(token.NoPos, pkg, "ErrConnClosed", errType))

	// func Serve(l net.Listener, handler http.Handler) error — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Serve",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "l", types.NewInterfaceType(nil, nil)),
				types.NewVar(token.NoPos, pkg, "handler", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ProcessEnv(r *http.Request) map[string]string — simplified
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ProcessEnv",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "",
				types.NewMap(types.Typ[types.String], types.Typ[types.String]))),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// image/color/palette
// ============================================================

func buildImageColorPalettePackage() *types.Package {
	pkg := types.NewPackage("image/color/palette", "palette")
	scope := pkg.Scope()

	// Palette type = []color.Color — simplified as []interface{}
	paletteType := types.NewSlice(types.NewInterfaceType(nil, nil))

	// var Plan9 []color.Color
	scope.Insert(types.NewVar(token.NoPos, pkg, "Plan9", paletteType))
	// var WebSafe []color.Color
	scope.Insert(types.NewVar(token.NoPos, pkg, "WebSafe", paletteType))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// runtime/metrics
// ============================================================

func buildRuntimeMetricsPackage() *types.Package {
	pkg := types.NewPackage("runtime/metrics", "metrics")
	scope := pkg.Scope()

	// type Description struct { ... }
	descStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Description", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Kind", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg, "Cumulative", types.Typ[types.Bool], false),
	}, nil)
	descType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Description", nil),
		descStruct, nil)
	scope.Insert(descType.Obj())

	// type ValueKind int
	valueKindType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "ValueKind", nil),
		types.Typ[types.Int], nil)
	scope.Insert(valueKindType.Obj())

	// ValueKind constants
	for i, name := range []string{"KindBad", "KindUint64", "KindFloat64", "KindFloat64Histogram"} {
		scope.Insert(types.NewConst(token.NoPos, pkg, name, valueKindType, constant.MakeInt64(int64(i))))
	}

	// type Value struct { ... }
	valueStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "kind", valueKindType, false),
		types.NewField(token.NoPos, pkg, "scalar", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, pkg, "pointer", types.Typ[types.Int], false),
	}, nil)
	valueType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Value", nil),
		valueStruct, nil)
	scope.Insert(valueType.Obj())

	valueType.AddMethod(types.NewFunc(token.NoPos, pkg, "Kind",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", valueType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", valueKindType)), false)))
	valueType.AddMethod(types.NewFunc(token.NoPos, pkg, "Uint64",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", valueType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Uint64])), false)))
	valueType.AddMethod(types.NewFunc(token.NoPos, pkg, "Float64",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", valueType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Float64])), false)))
	valueType.AddMethod(types.NewFunc(token.NoPos, pkg, "Float64Histogram",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", valueType), nil, nil,
			nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int])), false)))

	// type Float64Histogram struct { ... }
	histStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Counts", types.NewSlice(types.Typ[types.Uint64]), false),
		types.NewField(token.NoPos, pkg, "Buckets", types.NewSlice(types.Typ[types.Float64]), false),
	}, nil)
	histType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Float64Histogram", nil),
		histStruct, nil)
	scope.Insert(histType.Obj())

	// type Sample struct { Name string; Value Value }
	sampleStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Name", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Value", valueType, false),
	}, nil)
	sampleType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Sample", nil),
		sampleStruct, nil)
	scope.Insert(sampleType.Obj())

	// func All() []Description
	scope.Insert(types.NewFunc(token.NoPos, pkg, "All",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.NewSlice(descType))),
			false)))

	// func Read(m []Sample)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Read",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "m", types.NewSlice(sampleType))),
			nil, false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// runtime/coverage
// ============================================================

func buildRuntimeCoveragePackage() *types.Package {
	pkg := types.NewPackage("runtime/coverage", "coverage")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// func WriteCountersDir(dir string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteCountersDir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteCounters(w io.Writer) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteCounters",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteMetaDir(dir string) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteMetaDir",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "dir", types.Typ[types.String])),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func WriteMeta(w io.Writer) error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "WriteMeta",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "w", types.NewInterfaceType(nil, nil))),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// func ClearCounters() error
	scope.Insert(types.NewFunc(token.NoPos, pkg, "ClearCounters",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}

// ============================================================
// plugin
// ============================================================

func buildPluginPackage() *types.Package {
	pkg := types.NewPackage("plugin", "plugin")
	scope := pkg.Scope()
	errType := types.Universe.Lookup("error").Type()

	// type Plugin struct { ... }
	pluginStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "data", types.Typ[types.Int], false),
	}, nil)
	pluginType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Plugin", nil),
		pluginStruct, nil)
	scope.Insert(pluginType.Obj())

	// func Open(path string) (*Plugin, error)
	scope.Insert(types.NewFunc(token.NoPos, pkg, "Open",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "path", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.NewPointer(pluginType)),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	// Plugin.Lookup(symName string) (Symbol, error)
	symbolType := types.NewInterfaceType(nil, nil)
	symbolType.Complete()
	scope.Insert(types.NewTypeName(token.NoPos, pkg, "Symbol", symbolType))

	pluginType.AddMethod(types.NewFunc(token.NoPos, pkg, "Lookup",
		types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", types.NewPointer(pluginType)), nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, pkg, "symName", types.Typ[types.String])),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", symbolType),
				types.NewVar(token.NoPos, pkg, "", errType)),
			false)))

	pkg.MarkComplete()
	return pkg
}
