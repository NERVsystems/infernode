typedef struct{char *name; long sig; void (*fn)(void*); int size; int np; uchar map[16];} Runtab;
Runtab Loadermodtab[]={
	"compile",0xffffffff9af56bb1,Loader_compile,80,2,{0x0,0x80,},
	"dnew",0x62a7cf80,Loader_dnew,80,2,{0x0,0x40,},
	"ext",0xffffffffcd936c80,Loader_ext,96,2,{0x0,0x80,},
	"ifetch",0xfffffffffb64be19,Loader_ifetch,72,2,{0x0,0x80,},
	"link",0xffffffffe2473595,Loader_link,72,2,{0x0,0x80,},
	"newmod",0x6de26f71,Loader_newmod,104,2,{0x0,0x98,},
	"tdesc",0xffffffffb933ef75,Loader_tdesc,72,2,{0x0,0x80,},
	"tnew",0xffffffffc1d58785,Loader_tnew,88,2,{0x0,0xa0,},
	0
};
#define Loadermodlen	8
