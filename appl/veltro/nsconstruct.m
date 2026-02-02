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
# Security Model (v2):
#   Parent prepares sandbox directory with restrictive permissions
#   Child: NEWPGRP -> FORKNS -> NEWENV -> NEWFD -> NODEVS -> chdir -> NEWNS
#   Result: Child sees only what parent explicitly bound
#

NsConstruct: module {
	PATH: con "/dis/veltro/nsconstruct.dis";

	# LLM configuration for a child agent
	LLMConfig: adt {
		model:       string;   # Model name (e.g., "gpt-4")
		temperature: real;     # 0.0 - 1.0
		system:      string;   # System prompt (parent-controlled)
	};

	# Mount point permissions for sandbox
	Mountpoints: adt {
		srv:   int;    # 0 = no /srv (default for untrusted)
		net:   int;    # 0 = no /net (default for untrusted)
		prog:  int;    # 0 = no /prog (always 0 for untrusted)
	};

	# MCP provider configuration (for mc9p integration)
	MCProvider: adt {
		name:     string;          # Provider name ("http", "fs", "search")
		domains:  list of string;  # Domains to grant within provider
		netgrant: int;             # 1 = provider has /net access
	};

	# Capabilities to grant to a child agent
	Capabilities: adt {
		tools:       list of string;       # Tool names to include ("read", "list")
		paths:       list of string;       # File paths to expose
		shellcmds:   list of string;       # Shell commands for exec ("cat", "ls")
		llmconfig:   ref LLMConfig;        # Child's LLM settings
		fds:         list of int;          # Explicit FD keep-list
		mountpoints: ref Mountpoints;      # What to mount in sandbox
		sandboxid:   string;               # Validated unique identifier
		trusted:     int;                  # 0 = untrusted (no shell, no exec)
		mcproviders: list of ref MCProvider;  # mc9p providers to spawn
		memory:      int;                  # 1 = enable agent memory
	};

	# Essential paths captured before NEWNS for binding into child namespace
	Essentials: adt {
		dis:    string;   # Path to /dis
		dev:    string;   # Path to /dev
		moddir: string;   # Path to module definitions
	};

	# Bind operation record for audit trail
	BindRecord: adt {
		src:   string;   # Source path
		dst:   string;   # Destination path
		flags: int;      # Bind flags used
	};

	# Initialize the module
	init: fn();

	# Validate sandbox ID - returns error string or nil on success
	# Rejects path traversal attacks, invalid characters, length issues
	validatesandboxid: fn(id: string): string;

	# Generate a unique sandbox ID
	gensandboxid: fn(): string;

	# Prepare sandbox directory structure (called by parent before spawn)
	# Creates sandbox at /tmp/.veltro/sandbox/{id}/ with restrictive permissions
	# Returns error string or nil on success
	preparesandbox: fn(caps: ref Capabilities): string;

	# Clean up sandbox directory after child exits
	cleanupsandbox: fn(sandboxid: string);

	# Emit audit log of namespace bindings
	# Writes to /tmp/.veltro/audit/{sandboxid}.ns
	emitauditlog: fn(sandboxid: string, binds: list of ref BindRecord);

	# Get sandbox path for a given ID
	sandboxpath: fn(sandboxid: string): string;

	# Verify ownership of a path (stat check before bind)
	verifyownership: fn(path: string): string;

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
