#0
	frame	$6,80(fp)
	movp	792(mp),64(80(fp))
	movp	688(mp),72(80(fp))
	movp	64(fp),80(80(fp))
	lea	72(fp),32(80(fp))
	mcall	80(fp),$1,808(mp)
	raise	648(mp)
	load	8(mp),$5,808(mp)
	load	520(mp),$1,592(mp)
	bnew	592(mp),720(mp),$13
#10
	frame	$5,112(fp)
	movp	520(mp),64(112(fp))
	call	112(fp),$0
	load	536(mp),$3,768(mp)
	bnew	768(mp),720(mp),$18
	frame	$5,112(fp)
	movp	536(mp),64(112(fp))
	call	112(fp),$0
	load	544(mp),$4,800(mp)
	bnew	800(mp),720(mp),$23
#20
	frame	$5,112(fp)
	movp	544(mp),64(112(fp))
	call	112(fp),$0
	mframe	808(mp),$0,112(fp)
	movw	$2,64(112(fp))
	lea	792(mp),32(112(fp))
	mcall	112(fp),$0,808(mp)
	mframe	808(mp),$0,128(fp)
	movw	$1,64(128(fp))
	lea	120(fp),32(128(fp))
#30
	mcall	128(fp),$0,808(mp)
	mframe	592(mp),$1,112(fp)
	movp	120(fp),64(112(fp))
	movp	720(mp),120(fp)
	movw	$1,72(112(fp))
	lea	744(mp),32(112(fp))
	mcall	112(fp),$1,592(mp)
	movw	$0,96(fp)
	movw	$0,784(mp)
	movw	$0,88(fp)
#40
	load	512(mp),$0,80(fp)
	bnew	720(mp),80(fp),$45
	frame	$5,128(fp)
	movp	512(mp),64(128(fp))
	call	128(fp),$0
	mframe	80(fp),$1,128(fp)
	movp	72(fp),64(128(fp))
	mcall	128(fp),$1,80(fp)
	mframe	80(fp),$2,112(fp)
	lea	104(fp),32(112(fp))
#50
	mcall	112(fp),$2,80(fp)
	movw	104(fp),128(fp)
	beqw	128(fp),$0,$98
	case	104(fp),96(mp)
	addw	$1,672(mp)
	load	528(mp),$2,608(mp)
	bnew	608(mp),720(mp),$60
	frame	$5,128(fp)
	movp	528(mp),64(128(fp))
	call	128(fp),$0
#60
	mframe	608(mp),$1,128(fp)
	lea	736(mp),32(128(fp))
	mcall	128(fp),$1,608(mp)
	jmp	$48
	addw	$1,752(mp)
	jmp	$48
	addw	$1,760(mp)
	jmp	$48
	addw	$1,616(mp)
	jmp	$48
#70
	addw	$1,640(mp)
	jmp	$48
	addw	$1,704(mp)
	jmp	$48
	addw	$1,728(mp)
	jmp	$48
	addw	$1,776(mp)
	jmp	$48
	addw	$1,824(mp)
	jmp	$48
#80
	addw	$1,832(mp)
	jmp	$48
	movw	$3,784(mp)
	jmp	$48
	movw	$16,88(fp)
	jmp	$48
	movw	$32,96(fp)
	jmp	$48
	addw	$1,568(mp)
	jmp	$48
#90
	addw	$1,560(mp)
	jmp	$48
	frame	$4,112(fp)
	movp	792(mp),64(112(fp))
	movp	840(mp),72(112(fp))
	lea	128(fp),32(112(fp))
	mcall	112(fp),$1,808(mp)
	raise	656(mp)
	mframe	80(fp),$0,128(fp)
	lea	72(fp),32(128(fp))
#100
	mcall	128(fp),$0,80(fp)
	movp	720(mp),80(fp)
	bnew	$0,728(mp),$109
	beqw	$0,824(mp),$110
	beqw	$0,832(mp),$107
	movw	$1,784(mp)
	jmp	$110
	movw	$2,784(mp)
	jmp	$110
	movw	$4,784(mp)
#110
	orw	88(fp),96(fp),128(fp)
	orw	128(fp),784(mp)
	bnew	720(mp),72(fp),$118
	movp	720(mp),120(fp)
	consp	72(mp),120(fp)
	movp	120(fp),72(fp)
	movp	720(mp),120(fp)
	addw	$1,752(mp)
	beqw	720(mp),72(fp),$124
	frame	$17,128(fp)
#120
	headp	72(fp),64(128(fp))
	call	128(fp),$131
	tail	72(fp),72(fp)
	jmp	$118
	frame	$10,128(fp)
	call	128(fp),$196
	mframe	592(mp),$0,112(fp)
	movp	744(mp),64(112(fp))
	lea	128(fp),32(112(fp))
	mcall	112(fp),$0,592(mp)
#130
	ret	
	mframe	808(mp),$3,120(fp)
	movp	64(fp),64(120(fp))
	lea	240(fp),32(120(fp))
	mcall	120(fp),$3,808(mp)
	movw	240(fp),112(fp)
	movmp	248(fp),$8,136(fp)
	movp	720(mp),248(fp)
	movp	720(mp),256(fp)
	movp	720(mp),264(fp)
#140
	movp	720(mp),272(fp)
	bnew	$-1,112(fp),$149
	frame	$6,128(fp)
	movp	792(mp),64(128(fp))
	movp	696(mp),72(128(fp))
	movp	64(fp),80(128(fp))
	lea	120(fp),32(128(fp))
	mcall	128(fp),$1,808(mp)
	ret	
	bnew	$0,616(mp),$152
#150
	andw	488(mp),192(fp),128(fp)
	bnew	128(fp),$0,$175
	bnew	$0,624(mp),$155
	newa	$30,$8,632(mp)
	jmp	$162
	lena	632(mp),128(fp)
	bnew	128(fp),624(mp),$162
	mulw	$2,624(mp),128(fp)
	newa	128(fp),$8,88(fp)
	slicela	632(mp),$0,88(fp)
#160
	movp	88(fp),632(mp)
	movp	720(mp),88(fp)
	mframe	800(mp),$0,128(fp)
	movp	64(fp),64(128(fp))
	movp	504(mp),72(128(fp))
	lea	72(fp),32(128(fp))
	mcall	128(fp),$0,800(mp)
	beqc	720(mp),72(fp),$170
	addc	80(fp),72(fp),136(fp)
	orw	480(mp),232(fp)
#170
	movw	624(mp),128(fp)
	addw	$1,624(mp)
	indx	632(mp),128(fp),128(fp)
	movmp	136(fp),$8,0(128(fp))
	ret	
	frame	$10,128(fp)
	call	128(fp),$196
	mframe	768(mp),$0,128(fp)
	movp	64(fp),64(128(fp))
	movw	784(mp),72(128(fp))
#180
	lea	96(fp),32(128(fp))
	mcall	128(fp),$0,768(mp)
	blew	$0,104(fp),$190
	frame	$6,120(fp)
	movp	792(mp),64(120(fp))
	movp	680(mp),72(120(fp))
	movp	64(fp),80(120(fp))
	lea	128(fp),32(120(fp))
	mcall	120(fp),$1,808(mp)
	ret	
#190
	frame	$11,128(fp)
	movp	64(fp),64(128(fp))
	movp	96(fp),72(128(fp))
	slicea	$0,104(fp),72(128(fp))
	call	128(fp),$286
	ret	
	bnew	$0,624(mp),$198
	ret	
	newa	624(mp),$1,72(fp)
	movw	$0,64(fp)
#200
	blew	624(mp),64(fp),$209
	indl	72(fp),104(fp),64(fp)
	new	$8,112(fp)
	indx	632(mp),96(fp),64(fp)
	movmp	0(96(fp)),$8,0(112(fp))
	movp	112(fp),0(104(fp))
	movp	720(mp),112(fp)
	addw	$1,64(fp)
	jmp	$200
	mframe	768(mp),$1,96(fp)
#210
	movp	72(fp),64(96(fp))
	movw	784(mp),72(96(fp))
	lea	80(fp),32(96(fp))
	mcall	96(fp),$1,768(mp)
	frame	$11,96(fp)
	movp	720(mp),64(96(fp))
	movp	80(fp),72(96(fp))
	slicea	$0,88(fp),72(96(fp))
	call	96(fp),$286
	movw	$0,624(mp)
#220
	movp	720(mp),632(mp)
	ret	
	movw	$0,128(fp)
	movw	$0,136(fp)
	movw	$0,144(fp)
	movw	$0,152(fp)
	movw	$0,160(fp)
	movw	$0,168(fp)
	movw	$0,176(fp)
	movw	$0,88(fp)
#230
	lena	64(fp),104(fp)
	blew	104(fp),88(fp),$281
	indl	64(fp),104(fp),88(fp)
	movp	0(104(fp)),80(fp)
	beqw	$0,776(mp),$243
	addl	80(80(fp)),80(mp),96(fp)
	divl	88(mp),96(fp)
	cvtlc	96(fp),112(fp)
	lenc	112(fp),72(fp)
	movp	720(mp),112(fp)
#240
	movw	72(fp),104(fp)
	blew	104(fp),176(fp),$243
	movw	72(fp),176(fp)
	beqw	$0,704(mp),$249
	lenc	24(80(fp)),120(fp)
	addw	120(fp),$2,72(fp)
	movw	72(fp),104(fp)
	blew	104(fp),160(fp),$249
	movw	72(fp),160(fp)
	beqw	$0,760(mp),$256
#250
	cvtwc	40(80(fp)),112(fp)
	lenc	112(fp),72(fp)
	movp	720(mp),112(fp)
	movw	72(fp),120(fp)
	blew	120(fp),128(fp),$256
	movw	72(fp),128(fp)
	beqw	$0,672(mp),$278
	andw	96(80(fp)),496(mp),104(fp)
	cvtwc	104(fp),112(fp)
	lenc	112(fp),72(fp)
#260
	movp	720(mp),112(fp)
	movw	72(fp),120(fp)
	blew	120(fp),136(fp),$264
	movw	72(fp),136(fp)
	lenc	8(80(fp)),72(fp)
	movw	72(fp),120(fp)
	blew	120(fp),144(fp),$268
	movw	72(fp),144(fp)
	lenc	16(80(fp)),72(fp)
	movw	72(fp),120(fp)
#270
	blew	120(fp),152(fp),$272
	movw	72(fp),152(fp)
	cvtlc	80(80(fp)),112(fp)
	lenc	112(fp),72(fp)
	movp	720(mp),112(fp)
	movw	72(fp),120(fp)
	blew	120(fp),168(fp),$278
	movw	72(fp),168(fp)
	movp	720(mp),80(fp)
	addw	$1,88(fp)
#280
	jmp	$230
	new	$2,112(fp)
	movm	128(fp),$56,0(112(fp))
	movp	112(fp),0(32(fp))
	movp	720(mp),112(fp)
	ret	
	frame	$15,96(fp)
	movp	72(fp),64(96(fp))
	lea	88(fp),32(96(fp))
	call	96(fp),$222
#290
	movw	$0,80(fp)
	lena	72(fp),96(fp)
	blew	96(fp),80(fp),$305
	frame	$16,96(fp)
	movp	64(fp),64(96(fp))
	indl	72(fp),112(fp),80(fp)
	movp	0(112(fp)),104(fp)
	movp	0(104(fp)),72(96(fp))
	movp	720(mp),104(fp)
	indl	72(fp),112(fp),80(fp)
#300
	movp	0(112(fp)),80(96(fp))
	movp	88(fp),88(96(fp))
	call	96(fp),$306
	addw	$1,80(fp)
	jmp	$291
	ret	
	beqw	$0,776(mp),$320
	frame	$5,152(fp)
	movp	16(mp),64(152(fp))
	movw	48(88(fp)),72(152(fp))
#310
	addl	80(80(fp)),80(mp),120(fp)
	divl	88(mp),120(fp),80(152(fp))
	lea	144(fp),32(152(fp))
	mcall	152(fp),$2,808(mp)
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	144(fp),72(136(fp))
	movp	720(mp),144(fp)
	lea	128(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
#320
	beqw	$0,704(mp),$342
	frame	$4,128(fp)
	movp	576(mp),64(128(fp))
	movp	24(80(fp)),72(128(fp))
	lea	144(fp),32(128(fp))
	mcall	128(fp),$2,808(mp)
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	144(fp),72(136(fp))
	movp	720(mp),144(fp)
#330
	lea	152(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
	lenc	24(80(fp)),152(fp)
	addw	152(fp),$2,112(fp)
	blew	32(88(fp)),112(fp),$342
	mframe	592(mp),$2,136(fp)
	movp	744(mp),64(136(fp))
	movw	$32,72(136(fp))
	lea	152(fp),32(136(fp))
	mcall	136(fp),$2,592(mp)
#340
	addw	$1,112(fp)
	jmp	$334
	beqw	$0,760(mp),$357
	frame	$9,128(fp)
	movp	40(mp),64(128(fp))
	movl	32(80(fp)),72(128(fp))
	movw	0(88(fp)),80(128(fp))
	movw	40(80(fp)),88(128(fp))
	movw	48(80(fp)),96(128(fp))
	lea	144(fp),32(128(fp))
#350
	mcall	128(fp),$2,808(mp)
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	144(fp),72(136(fp))
	movp	720(mp),144(fp)
	lea	152(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
	beqw	$0,568(mp),$372
	movw	$67108864,48(fp)
	andw	56(80(fp)),48(fp),152(fp)
#360
	beqw	152(fp),$0,$367
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	816(mp),72(136(fp))
	lea	152(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
	jmp	$372
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	64(mp),72(136(fp))
#370
	lea	152(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
	movp	72(fp),96(fp)
	andw	96(80(fp)),480(mp),112(fp)
	andw	496(mp),96(80(fp))
	beqw	$0,752(mp),$388
	beqw	$0,112(fp),$386
	mframe	800(mp),$0,152(fp)
	movp	0(80(fp)),64(152(fp))
	movp	504(mp),72(152(fp))
#380
	lea	184(fp),32(152(fp))
	mcall	152(fp),$0,800(mp)
	movp	192(fp),96(fp)
	movp	720(mp),184(fp)
	movp	720(mp),192(fp)
	jmp	$398
	movp	0(80(fp)),96(fp)
	jmp	$398
	beqc	720(mp),64(fp),$398
	lenc	64(fp),136(fp)
#390
	subw	$1,136(fp)
	indc	64(fp),136(fp),152(fp)
	bnew	152(fp),$47,$395
	addc	96(fp),64(fp),96(fp)
	jmp	$398
	addc	504(mp),64(fp),144(fp)
	addc	96(fp),144(fp),96(fp)
	movp	720(mp),144(fp)
	beqw	$0,560(mp),$405
	frame	$3,152(fp)
#400
	movp	80(fp),64(152(fp))
	lea	144(fp),32(152(fp))
	call	152(fp),$478
	addc	144(fp),96(fp)
	movp	720(mp),144(fp)
	beqw	$0,672(mp),$472
	movw	72(80(fp)),104(fp)
	beqw	$0,832(mp),$409
	movw	64(80(fp)),104(fp)
	beqw	$0,640(mp),$438
#410
	frame	$7,168(fp)
	movw	56(80(fp)),64(168(fp))
	lea	160(fp),32(168(fp))
	call	168(fp),$488
	frame	$13,128(fp)
	movp	24(mp),64(128(fp))
	movp	160(fp),72(128(fp))
	movp	720(mp),160(fp)
	movw	88(80(fp)),80(128(fp))
	movw	8(88(fp)),88(128(fp))
#420
	movw	96(80(fp)),96(128(fp))
	subw	16(88(fp)),$0,104(128(fp))
	movp	8(80(fp)),112(128(fp))
	subw	24(88(fp)),$0,120(128(fp))
	movp	16(80(fp)),128(128(fp))
	movw	40(88(fp)),136(128(fp))
	movl	80(80(fp)),144(128(fp))
	movw	104(fp),152(128(fp))
	movp	96(fp),160(128(fp))
	lea	144(fp),32(128(fp))
#430
	mcall	128(fp),$2,808(mp)
	mframe	592(mp),$3,136(fp)
	movp	744(mp),64(136(fp))
	movp	144(fp),72(136(fp))
	movp	720(mp),144(fp)
	lea	152(fp),32(136(fp))
	mcall	136(fp),$3,592(mp)
	ret	
	frame	$7,128(fp)
	movw	56(80(fp)),64(128(fp))
#440
	lea	144(fp),32(128(fp))
	call	128(fp),$488
	mframe	608(mp),$0,128(fp)
	movw	736(mp),64(128(fp))
	movw	104(fp),72(128(fp))
	lea	176(fp),32(128(fp))
	mcall	128(fp),$0,608(mp)
	frame	$14,136(fp)
	movp	32(mp),64(136(fp))
	movp	144(fp),72(136(fp))
#450
	movp	720(mp),144(fp)
	movw	88(80(fp)),80(136(fp))
	movw	8(88(fp)),88(136(fp))
	movw	96(80(fp)),96(136(fp))
	subw	16(88(fp)),$0,104(136(fp))
	movp	8(80(fp)),112(136(fp))
	subw	24(88(fp)),$0,120(136(fp))
	movp	16(80(fp)),128(136(fp))
	movw	40(88(fp)),136(136(fp))
	movl	80(80(fp)),144(136(fp))
#460
	movp	176(fp),152(136(fp))
	movp	720(mp),176(fp)
	movp	96(fp),160(136(fp))
	lea	160(fp),32(136(fp))
	mcall	136(fp),$2,808(mp)
	mframe	592(mp),$3,152(fp)
	movp	744(mp),64(152(fp))
	movp	160(fp),72(152(fp))
	movp	720(mp),160(fp)
	lea	168(fp),32(152(fp))
#470
	mcall	152(fp),$3,592(mp)
	ret	
	mframe	592(mp),$3,152(fp)
	movp	744(mp),64(152(fp))
	addc	0(mp),96(fp),72(152(fp))
	lea	168(fp),32(152(fp))
	mcall	152(fp),$3,592(mp)
	ret	
	andw	48(64(fp)),$128,72(fp)
	beqw	72(fp),$0,$482
#480
	movp	504(mp),0(32(fp))
	ret	
	andw	56(64(fp)),$73,72(fp)
	beqw	72(fp),$0,$486
	movp	48(mp),0(32(fp))
	ret	
	movp	720(mp),0(32(fp))
	ret	
	andw	488(mp),64(fp),80(fp)
	beqw	80(fp),$0,$492
#490
	movp	600(mp),72(fp)
	jmp	$501
	andw	480(mp),64(fp),80(fp)
	beqw	80(fp),$0,$496
	movp	584(mp),72(fp)
	jmp	$501
	andw	$134217728,64(fp),80(fp)
	beqw	80(fp),$0,$500
	movp	552(mp),72(fp)
	jmp	$501
#500
	movp	56(mp),72(fp)
	andw	472(mp),64(fp),80(fp)
	beqw	80(fp),$0,$505
	addc	664(mp),72(fp)
	jmp	$506
	addc	56(mp),72(fp)
	shrw	$6,64(fp),80(fp)
	andw	$7,80(fp)
	indl	712(mp),80(fp),80(fp)
	movp	0(80(fp)),88(fp)
#510
	shrw	$3,64(fp),80(fp)
	andw	$7,80(fp)
	indl	712(mp),80(fp),80(fp)
	addc	0(80(fp)),88(fp)
	andw	$7,64(fp),80(fp)
	indl	712(mp),80(fp),80(fp)
	addc	0(80(fp)),88(fp)
	addc	88(fp),72(fp)
	movp	720(mp),88(fp)
	movp	72(fp),0(32(fp))
#520
	ret	
	entry	7, 12
	desc	$0,848,"ffc0000000000001fcf977649e40"
	desc	$1,8,"80"
	desc	$2,56,""
	desc	$3,80,"0080"
	desc	$4,80,"00c0"
	desc	$5,88,"0080"
	desc	$6,88,"00e0"
	desc	$7,96,"0050"
	desc	$8,104,"f0"
	desc	$9,104,"0080"
	desc	$10,120,"0062"
	desc	$11,120,"00d4"
	desc	$12,136,"00e1"
	desc	$13,168,"00c288"
	desc	$14,168,"00c298"
	desc	$15,184,"00a2"
	desc	$16,200,"00f82b80"
	desc	$17,352,"00f87801e0"
	var	@mp,848
	string	@mp+0,"\n"
	string	@mp+8,"$Sys"
	string	@mp+16,"%*bd "
	string	@mp+24,"%s %c %*d %*s %*s %*bud %d %s\n"
	string	@mp+32,"%s %c %*d %*s %*s %*bud %s %s\n"
	string	@mp+40,"(%.16bux %*ud %.2ux) "
	string	@mp+48,"*"
	string	@mp+56,"-"
	string	@mp+64,"- "
	string	@mp+72,"."
	long	@mp+80,1023 # 00000000000003ff
	long	@mp+88,1024 # 0000000000000400
	word	@mp+96,15,70,71,90,84,85,88,99,100,84,100,101,68,101,102,70,107,108,76,108,109,54,109,110,72,110,111,74,112,113,64,113,114,66,114,115,86,115,116,82,116,117,78,117,118,80,92
	word	@mp+472,536870912
	word	@mp+480,1073741824
	word	@mp+488,-2147483648
	word	@mp+496,-1073741825
	string	@mp+504,"/"
	string	@mp+512,"/dis/lib/arg.dis"
	string	@mp+520,"/dis/lib/bufio.dis"
	string	@mp+528,"/dis/lib/daytime.dis"
	string	@mp+536,"/dis/lib/readdir.dis"
	string	@mp+544,"/dis/lib/string.dis"
	string	@mp+552,"A"
	word	@mp+560,0
	word	@mp+568,0
	string	@mp+576,"[%s] "
	string	@mp+584,"a"
	string	@mp+600,"d"
	word	@mp+616,0
	word	@mp+640,0
	string	@mp+648,"fail:bad module"
	string	@mp+656,"fail:usage"
	string	@mp+664,"l"
	word	@mp+672,0
	string	@mp+680,"ls: Readdir: %s: %r\n"
	string	@mp+688,"ls: cannot load %s: %r\n"
	string	@mp+696,"ls: stat %s: %r\n"
	word	@mp+704,0
	array	@mp+712,$1,8
	indir	@mp+712,0
	string	@mp+0,"---"
	string	@mp+8,"--x"
	string	@mp+16,"-w-"
	string	@mp+24,"-wx"
	string	@mp+32,"r--"
	string	@mp+40,"r-x"
	string	@mp+48,"rw-"
	string	@mp+56,"rwx"
	apop
	word	@mp+728,0
	word	@mp+752,0
	word	@mp+760,0
	word	@mp+776,0
	string	@mp+816,"t "
	word	@mp+824,0
	word	@mp+832,0
	string	@mp+840,"usage: ls [-delmnpqrstucFT] [files]\n"
	module	Ls
	link	12,7,0x4244b354,"init"
	ldts	@ldt,6
	word	@ldt+0,3
	ext	@ldt+8,0x57f77f4f,"argv"
	ext	@ldt+24,0x2e01144a,"init"
	ext	@ldt+40,0x616977e8,"opt"
	word	@ldt+1,4
	ext	@ldt+16,0xffffffffd9e8365b,"Iobuf.flush"
	ext	@ldt+40,0x2c386517,"fopen"
	ext	@ldt+56,0xffffffff8b27d66b,"Iobuf.putc"
	ext	@ldt+80,0x80831eb,"Iobuf.puts"
	word	@ldt+2,2
	ext	@ldt+16,0x591ff366,"filet"
	ext	@ldt+32,0x616977e8,"now"
	word	@ldt+3,2
	ext	@ldt+16,0x156cb1b,"init"
	ext	@ldt+32,0x6470e606,"sortdir"
	word	@ldt+4,1
	ext	@ldt+16,0x49e5d97d,"splitstrr"
	word	@ldt+5,4
	ext	@ldt+16,0x1478f993,"fildes"
	ext	@ldt+32,0xfffffffff46486c8,"fprint"
	ext	@ldt+48,0x4c0624b6,"sprint"
	ext	@ldt+64,0x319328dd,"stat"
	source	"/appl/cmd/ls.b"
