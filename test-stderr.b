implement Test;

include "sys.m";
	sys: Sys;
include "draw.m";

Test: module
{
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

init(ctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	sys->fprint(sys->fildes(2), "STDERR: Hello from 64-bit Inferno!\n");
	sys->fprint(sys->fildes(1), "STDOUT: Hello from 64-bit Inferno!\n");
	sys->print("PRINT: Hello from 64-bit Inferno!\n");
	raise "fail:done";  # Force exit
}
