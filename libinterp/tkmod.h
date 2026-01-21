typedef struct{char *name; long sig; void (*fn)(void*); int size; int np; uchar map[16];} Runtab;
Runtab Tkmodtab[]={
	"cmd",0x1ee9697,Tk_cmd,80,2,{0x0,0xc0,},
	"color",0xffffffffc6935858,Tk_color,72,2,{0x0,0x80,},
	"getimage",0xffffffff80bea378,Tk_getimage,80,2,{0x0,0xc0,},
	"keyboard",0xffffffff8671bae6,Tk_keyboard,80,2,{0x0,0x80,},
	"namechan",0x35182638,Tk_namechan,88,2,{0x0,0xe0,},
	"pointer",0x21188625,Tk_pointer,104,2,{0x0,0x80,},
	"putimage",0x2dc55622,Tk_putimage,96,2,{0x0,0xf0,},
	"quote",0xffffffffb2cd7190,Tk_quote,72,2,{0x0,0x80,},
	"rect",0x683e6bae,Tk_rect,88,2,{0x0,0xc0,},
	"toplevel",0xffffffff96ab1cc9,Tk_toplevel,80,2,{0x0,0xc0,},
	0
};
#define Tkmodlen	10
