#0
	load	0(mp),$0,48(mp)
	mframe	48(mp),$0,104(fp)
	movw	$2,64(104(fp))
	lea	96(fp),32(104(fp))
	mcall	104(fp),$0,48(mp)
	frame	$2,88(fp)
	movp	96(fp),64(88(fp))
	movp	40(mp),96(fp)
	movp	16(mp),72(88(fp))
	lea	80(fp),32(88(fp))
#10
	mcall	88(fp),$1,48(mp)
	mframe	48(mp),$0,80(fp)
	movw	$1,64(80(fp))
	lea	96(fp),32(80(fp))
	mcall	80(fp),$0,48(mp)
	frame	$2,88(fp)
	movp	96(fp),64(88(fp))
	movp	40(mp),96(fp)
	movp	24(mp),72(88(fp))
	lea	104(fp),32(88(fp))
#20
	mcall	88(fp),$1,48(mp)
	frame	$1,88(fp)
	movp	8(mp),64(88(fp))
	lea	104(fp),32(88(fp))
	mcall	88(fp),$2,48(mp)
	raise	32(mp)
	entry	0, 3
	desc	$0,56,"fe"
	desc	$1,72,"0080"
	desc	$2,80,"00c0"
	desc	$3,112,"00c8"
	var	@mp,56
	string	@mp+0,"$Sys"
	string	@mp+8,"PRINT: Hello from 64-bit Inferno!\n"
	string	@mp+16,"STDERR: Hello from 64-bit Inferno!\n"
	string	@mp+24,"STDOUT: Hello from 64-bit Inferno!\n"
	string	@mp+32,"fail:done"
	module	Test
	link	3,0,0x4244b354,"init"
	ldts	@ldt,1
	word	@ldt+0,3
	ext	@ldt+8,0x1478f993,"fildes"
	ext	@ldt+24,0xfffffffff46486c8,"fprint"
	ext	@ldt+40,0xffffffffac849033,"print"
	source	"/test-stderr.b"
