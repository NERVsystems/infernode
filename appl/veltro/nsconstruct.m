#
# nsconstruct.m - Namespace construction for Veltro agents
#
# The heart of Veltro's security model. Constructs a namespace from nothing.
# A child's namespace can only be equal to or smaller than its parent's - never larger.
#
# This is guaranteed because:
# 1. NEWNS creates an empty namespace - child starts with nothing
# 2. Parent binds from its own resources - can't grant what you don't have
# 3. No way to expand - no "unbind and get more" operation
# 4. Dynamic loading restricted - can only load from bound paths
#

NsConstruct: module {
	PATH: con "/dis/veltro/nsconstruct.dis";

	# LLM configuration for a child agent
	LLMConfig: adt {
		model:       string;   # Model name (e.g., "gpt-4")
		temperature: real;     # 0.0 - 1.0
		system:      string;   # System prompt (parent-controlled)
	};

	# Capabilities to grant to a child agent
	Capabilities: adt {
		tools:     list of string;    # Tool names to include ("read", "list")
		paths:     list of string;    # File paths to expose
		shellcmds: list of string;    # Shell commands for exec ("cat", "ls")
		llmconfig: ref LLMConfig;     # Child's LLM settings
	};

	# Essential paths captured before NEWNS for binding into child namespace
	Essentials: adt {
		dis:    string;   # Path to /dis
		dev:    string;   # Path to /dev
		moddir: string;   # Path to module definitions
	};

	# Initialize the module
	init: fn();

	# Capture essential paths from current namespace (call before NEWNS)
	captureessentials: fn(): ref Essentials;

	# Construct a new namespace with given capabilities
	# Returns nil on success, error string on failure
	# Must be called in a spawned process - uses NEWNS
	construct: fn(ess: ref Essentials, caps: ref Capabilities): string;

	# Bind essential runtime paths into current namespace
	# Called after NEWNS with captured essentials
	bindessentials: fn(ess: ref Essentials): string;

	# Start and mount tools9p with only the specified tools
	mounttools: fn(tools: list of string): string;

	# Create directory structure for agent
	mkdirs: fn(): string;
};
