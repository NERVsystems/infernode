implement NsConstruct;

#
# nsconstruct.b - Namespace construction for Veltro agents
#
# CAPABILITY MODEL:
# 1. Agent starts with namespace given by user/system (tools + paths + shellcmds)
# 2. Sub-agent gets subset: tools' ⊆ tools, paths' ⊆ paths, shellcmds' ⊆ shellcmds
# 3. You can only grant what you have
#
# Implementation uses FORKNS (copy parent's namespace) then restricts:
#   - Tool isolation: new tools9p serves only granted tools
#   - Shell isolation: /dis contains only granted shell commands
#   - Path isolation: tools check caps.paths before operations
#

include "sys.m";
	sys: Sys;
	FORKNS, NEWPGRP: import Sys;

include "draw.m";

include "nsconstruct.m";

init()
{
	sys = load Sys Sys->PATH;
}

# Capture essential paths - not needed with FORKNS model
captureessentials(): ref Essentials
{
	return ref Essentials("/dis", "/dev", "/module");
}

# Bind essentials - not needed with FORKNS model
# Child inherits parent's namespace; restrictions come from:
#   1. New tools9p with subset of tools
#   2. Path checking in tool execution
bindessentials(nil: ref Essentials): string
{
	return nil;
}

# Mount tools - handled by spawn.b directly
mounttools(nil: list of string): string
{
	return nil;
}

# Create directories - not needed with FORKNS model
mkdirs(): string
{
	return nil;
}

# Construct namespace - not used with FORKNS model
# Kept for interface compatibility
construct(nil: ref Essentials, nil: ref Capabilities): string
{
	return nil;
}
