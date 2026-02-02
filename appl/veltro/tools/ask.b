implement ToolAsk;

#
# ask - User prompt tool for Veltro agent
#
# Asks the user a question and returns their response.
# Works via console I/O or Xenith dialog depending on context.
#
# Usage:
#   ask <question>              # Simple question
#   ask -c <choice1,choice2...> <question>  # Multiple choice
#
# Examples:
#   ask "What is the target directory?"
#   ask -c "yes,no" "Do you want to continue?"
#   ask -c "small,medium,large" "What size?"
#
# Returns the user's response as a string.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "string.m";
	str: String;

include "../tool.m";

ToolAsk: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

# Console for input
cons: ref Sys->FD;
consout: ref Sys->FD;

init(): string
{
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "cannot load Sys";
	bufio = load Bufio Bufio->PATH;
	if(bufio == nil)
		return "cannot load Bufio";
	str = load String String->PATH;
	if(str == nil)
		return "cannot load String";

	# Open console for input/output
	cons = sys->open("/dev/cons", Sys->OREAD);
	consout = sys->open("/dev/cons", Sys->OWRITE);

	return nil;
}

name(): string
{
	return "ask";
}

doc(): string
{
	return "Ask - Prompt user for input\n\n" +
		"Usage:\n" +
		"  ask <question>                     # Free-form input\n" +
		"  ask -c <choice1,choice2> <question> # Multiple choice\n\n" +
		"Arguments:\n" +
		"  question - The question to ask the user\n" +
		"  -c       - Comma-separated list of valid choices\n\n" +
		"Examples:\n" +
		"  ask \"What is the target directory?\"\n" +
		"  ask -c \"yes,no\" \"Do you want to continue?\"\n" +
		"  ask -c \"small,medium,large\" \"What size?\"\n\n" +
		"Returns the user's response as a string.";
}

exec(args: string): string
{
	if(sys == nil || cons == nil)
		init();

	if(cons == nil || consout == nil)
		return "error: cannot open console for user input";

	args = strip(args);
	if(args == "")
		return "error: usage: ask <question> | ask -c <choices> <question>";

	choices: list of string;
	question: string;

	# Check for -c flag (choices)
	if(len args > 3 && args[0:3] == "-c ") {
		rest := strip(args[3:]);

		# Extract choices (first word, comma-separated)
		(choicestr, remainder) := splitfirst(rest);
		(nil, choicelist) := sys->tokenize(choicestr, ",");
		for(; choicelist != nil; choicelist = tl choicelist)
			choices = hd choicelist :: choices;

		# Reverse to maintain order
		rev: list of string;
		for(; choices != nil; choices = tl choices)
			rev = hd choices :: rev;
		choices = rev;

		question = stripquotes(strip(remainder));
	} else {
		question = stripquotes(args);
	}

	if(question == "")
		return "error: no question provided";

	# Display question
	if(choices != nil) {
		# Multiple choice prompt
		choicestr := "";
		i := 1;
		c: list of string;
		for(c = choices; c != nil; c = tl c) {
			choicestr += sys->sprint("  %d) %s\n", i, hd c);
			i++;
		}
		prompt := sys->sprint("\n%s\n%s> ", question, choicestr);
		sys->fprint(consout, "%s", prompt);

		# Read response
		response := readline();
		if(response == "")
			return "error: no response received";

		# Check if numeric choice
		if(response[0] >= '1' && response[0] <= '9') {
			idx := int response - 1;
			i = 0;
			for(c = choices; c != nil; c = tl c) {
				if(i == idx)
					return hd c;
				i++;
			}
		}

		# Check if matches a choice directly
		lresponse := str->tolower(response);
		for(c = choices; c != nil; c = tl c) {
			if(str->tolower(hd c) == lresponse)
				return hd c;
		}

		# Invalid choice
		validopts := "";
		for(c = choices; c != nil; c = tl c) {
			if(validopts != "")
				validopts += ", ";
			validopts += hd c;
		}
		return sys->sprint("error: invalid choice '%s' (valid: %s)", response, validopts);
	} else {
		# Free-form prompt
		sys->fprint(consout, "\n%s\n> ", question);

		# Read response
		response := readline();
		if(response == "")
			return "error: no response received";

		return response;
	}
}

# Read a line from console
readline(): string
{
	if(cons == nil)
		return "";

	buf := array[1024] of byte;
	n := sys->read(cons, buf, len buf);
	if(n <= 0)
		return "";

	line := string buf[0:n];

	# Trim trailing newline
	if(len line > 0 && line[len line - 1] == '\n')
		line = line[:len line - 1];

	return line;
}

# Strip leading/trailing whitespace
strip(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n'))
		j--;
	if(i >= j)
		return "";
	return s[i:j];
}

# Strip surrounding quotes
stripquotes(s: string): string
{
	if(len s < 2)
		return s;
	if((s[0] == '"' && s[len s - 1] == '"') ||
	   (s[0] == '\'' && s[len s - 1] == '\''))
		return s[1:len s - 1];
	return s;
}

# Split on first whitespace
splitfirst(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], strip(s[i:]));
	}
	return (s, "");
}
