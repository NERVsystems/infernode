#
# subagent.m - Interface for Veltro sub-agent loop
#
# A lightweight agent loop designed to run inside sandboxes.
# Unlike veltro.b, this module:
#   - Uses pre-loaded tool modules directly (no tools9p)
#   - Receives system prompt as parameter (no /lib/veltro/ access)
#   - Survives NEWNS by pre-loading dependencies
#
# NOTE: Include tool.m before including this file
#

SubAgent: module {
	PATH: con "/dis/veltro/subagent.dis";

	# Must be called BEFORE NEWNS while /dis/lib paths exist
	# Loads Bufio, String modules
	# Returns error string or nil on success
	init: fn(): string;

	# Run agent loop with pre-loaded tools
	# task: the task to accomplish
	# tools: list of pre-loaded Tool modules
	# toolnames: list of tool name strings (for namespace discovery)
	# systemprompt: system prompt from parent (session already configured)
	# llmaskfd: file descriptor for session's /n/llm/<id>/ask (survives NEWNS)
	# maxsteps: maximum agent steps (typically 50)
	# Returns final result string
	#
	# NOTE: Session is already created and configured by spawn.b with model,
	# temperature, thinking, and system prompt. This function just uses the ask fd.
	runloop: fn(task: string, tools: list of Tool,
	             toolnames: list of string,
	             systemprompt: string,
	             llmaskfd: ref Sys->FD,
	             maxsteps: int): string;
};
