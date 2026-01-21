implement Hello;

include "sys.m";
	sys: Sys;
include "draw.m";

Hello: module
{
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

init(ctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	sys->print("Hello from 64-bit Inferno!\n");
	sys->print("Arguments: ");
	for(a := args; a != nil; a = tl a)
		sys->print("%s ", hd a);
	sys->print("\n");
}
