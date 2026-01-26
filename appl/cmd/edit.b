implement Edit;

#
# edit - Simple text replacement command
#
# Usage:
#   edit -f /path/to/file -old 'text to find' -new 'replacement text'
#
# Performs exact string replacement in files.
# - Finds exact match of -old text in file
# - Replaces with -new text
# - Errors if -old not found or matches multiple times (ambiguous)
# - Use -all flag to replace all occurrences
#
# Examples:
#   edit -f /tmp/test.txt -old 'hello' -new 'goodbye'
#   edit -f config.rc -old 'port=8080' -new 'port=9090'
#   edit -f src/main.b -old 'DEBUG := 0' -new 'DEBUG := 1'
#   edit -f data.txt -old 'foo' -new 'bar' -all
#

include "sys.m";
	sys: Sys;

include "draw.m";

Edit: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

stderr: ref Sys->FD;
stdout: ref Sys->FD;

usage()
{
	sys->fprint(stderr, "Usage: edit -f file -old 'old text' -new 'new text' [-all]\n");
	raise "fail:usage";
}

init(nil: ref Draw->Context, argv: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);
	stdout = sys->fildes(1);

	filepath := "";
	oldtext := "";
	newtext := "";
	replaceall := 0;
	oldset := 0;
	newset := 0;

	# Skip program name
	argv = tl argv;

	# Parse arguments
	while(argv != nil) {
		a := hd argv;
		argv = tl argv;

		if(a == "-f") {
			if(argv == nil) {
				sys->fprint(stderr, "edit: -f requires an argument\n");
				usage();
			}
			filepath = hd argv;
			argv = tl argv;
		} else if(a == "-old") {
			if(argv == nil) {
				sys->fprint(stderr, "edit: -old requires an argument\n");
				usage();
			}
			oldtext = hd argv;
			oldset = 1;
			argv = tl argv;
		} else if(a == "-new") {
			if(argv == nil) {
				sys->fprint(stderr, "edit: -new requires an argument\n");
				usage();
			}
			newtext = hd argv;
			newset = 1;
			argv = tl argv;
		} else if(a == "-all") {
			replaceall = 1;
		} else if(len a > 0 && a[0] == '-') {
			sys->fprint(stderr, "edit: unknown option %s\n", a);
			usage();
		} else {
			sys->fprint(stderr, "edit: unexpected argument %s\n", a);
			usage();
		}
	}

	if(filepath == "") {
		sys->fprint(stderr, "edit: missing -f file\n");
		usage();
	}

	if(!oldset) {
		sys->fprint(stderr, "edit: missing -old text\n");
		usage();
	}

	if(!newset) {
		sys->fprint(stderr, "edit: missing -new text\n");
		usage();
	}

	if(oldtext == "") {
		sys->fprint(stderr, "edit: -old text cannot be empty\n");
		raise "fail:empty old";
	}

	# Read the file
	content := readfile(filepath);
	if(content == nil) {
		sys->fprint(stderr, "edit: cannot read %s: %r\n", filepath);
		raise "fail:read";
	}

	# Count occurrences
	count := countoccurrences(content, oldtext);

	if(count == 0) {
		sys->fprint(stderr, "edit: '%s' not found in %s\n", oldtext, filepath);
		raise "fail:not found";
	}

	if(count > 1 && !replaceall) {
		sys->fprint(stderr, "edit: '%s' found %d times in %s (use -all to replace all)\n",
			oldtext, count, filepath);
		raise "fail:ambiguous";
	}

	# Perform replacement
	newcontent: string;
	if(replaceall) {
		newcontent = replaceallstr(content, oldtext, newtext);
	} else {
		newcontent = replaceonce(content, oldtext, newtext);
	}

	# Write back
	if(writefile(filepath, newcontent) < 0) {
		sys->fprint(stderr, "edit: cannot write %s: %r\n", filepath);
		raise "fail:write";
	}

	if(replaceall && count > 1)
		sys->print("edit: replaced %d occurrences in %s\n", count, filepath);
	else
		sys->print("edit: replaced 1 occurrence in %s\n", filepath);
}

# Read entire file into a string
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;

	content := "";
	buf := array[8192] of byte;
	while((n := sys->read(fd, buf, len buf)) > 0)
		content += string buf[0:n];

	if(n < 0)
		return nil;

	return content;
}

# Write string to file
writefile(path: string, content: string): int
{
	fd := sys->create(path, Sys->OWRITE|Sys->OTRUNC, 8r644);
	if(fd == nil)
		return -1;

	data := array of byte content;
	n := sys->write(fd, data, len data);
	if(n != len data)
		return -1;

	return 0;
}

# Count occurrences of substring in string
countoccurrences(s, sub: string): int
{
	count := 0;
	sublen := len sub;
	slen := len s;

	for(i := 0; i <= slen - sublen; i++) {
		match := 1;
		for(j := 0; j < sublen; j++) {
			if(s[i+j] != sub[j]) {
				match = 0;
				break;
			}
		}
		if(match) {
			count++;
			i += sublen - 1;  # skip past this match
		}
	}
	return count;
}

# Replace first occurrence of old with new
replaceonce(s, old, new: string): string
{
	oldlen := len old;
	slen := len s;

	for(i := 0; i <= slen - oldlen; i++) {
		match := 1;
		for(j := 0; j < oldlen; j++) {
			if(s[i+j] != old[j]) {
				match = 0;
				break;
			}
		}
		if(match) {
			return s[0:i] + new + s[i+oldlen:];
		}
	}
	return s;
}

# Replace all occurrences of old with new
replaceallstr(s, old, new: string): string
{
	result := "";
	oldlen := len old;
	slen := len s;
	i := 0;

	while(i <= slen - oldlen) {
		match := 1;
		for(j := 0; j < oldlen; j++) {
			if(s[i+j] != old[j]) {
				match = 0;
				break;
			}
		}
		if(match) {
			result += new;
			i += oldlen;
		} else {
			result[len result] = s[i];
			i++;
		}
	}

	# Add remaining characters
	if(i < slen)
		result += s[i:];

	return result;
}
