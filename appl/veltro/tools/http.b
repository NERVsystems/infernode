implement ToolHttp;

#
# http - HTTP client tool for Veltro agent
#
# Performs HTTP requests and returns response body.
# Requires /net access (only available to trusted agents with net grant).
#
# HTTP uses Inferno's /net for direct TCP connections.
# HTTPS uses host-side curl via /cmd device — requires curl on host.
# DNS resolution uses host-side getent via /cmd for hostname lookups.
#
# Usage:
#   http GET <url>                    # GET request
#   http POST <url> <body>            # POST request
#   http PUT <url> <body>             # PUT request
#   http DELETE <url>                 # DELETE request
#   http HEAD <url>                   # HEAD request (headers only)
#
# Examples:
#   http GET http://example.com/api
#   http GET https://api.github.com/
#   http POST http://localhost:8080/data '{"key": "value"}'
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

ToolHttp: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

# Default HTTP port
HTTP_PORT: con "80";
HTTPS_PORT: con "443";

# Maximum response size (1MB)
MAX_RESPONSE: con 1024 * 1024;

# Response timeout (30 seconds)
TIMEOUT: con 30000;

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
	return nil;
}

name(): string
{
	return "http";
}

doc(): string
{
	return "Http - HTTP/HTTPS client\n\n" +
		"Usage:\n" +
		"  http GET <url>              # GET request\n" +
		"  http POST <url> <body>      # POST request\n" +
		"  http PUT <url> <body>       # PUT request\n" +
		"  http DELETE <url>           # DELETE request\n" +
		"  http HEAD <url>             # HEAD request\n\n" +
		"Arguments:\n" +
		"  url  - Full URL (http:// or https://)\n" +
		"  body - Request body (for POST/PUT)\n\n" +
		"Examples:\n" +
		"  http GET http://example.com/api\n" +
		"  http GET https://api.github.com/\n" +
		"  http POST http://localhost:8080/data '{\"key\": \"value\"}'\n\n" +
		"HTTP uses direct TCP. HTTPS uses host-side curl via /cmd.\n" +
		"Hostnames are resolved automatically via host DNS.";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	# Parse arguments
	(n, argv) := sys->tokenize(args, " \t");
	if(n < 2)
		return "error: usage: http <METHOD> <url> [body]";

	method := str->toupper(hd argv);
	argv = tl argv;
	url := hd argv;
	argv = tl argv;

	body := "";
	if(argv != nil) {
		# Join remaining args as body
		for(; argv != nil; argv = tl argv) {
			if(body != "")
				body += " ";
			body += hd argv;
		}
		body = stripquotes(body);
	}

	# Validate method
	case method {
	"GET" or "POST" or "PUT" or "DELETE" or "HEAD" or "PATCH" =>
		;
	* =>
		return "error: unsupported HTTP method: " + method;
	}

	# Parse URL
	(scheme, host, port, path, err) := parseurl(url);
	if(err != nil)
		return "error: " + err;

	# Connect
	if(scheme == "https") {
		# HTTPS via host-side curl
		return curlrequest(method, host, port, path, body);
	} else {
		# Plain HTTP — resolve hostname first
		(resolvedhost, rerr) := resolve(host);
		if(rerr != nil)
			return "error: " + rerr;
		addr := sys->sprint("tcp!%s!%s", resolvedhost, port);
		(ok, conn) := sys->dial(addr, nil);
		if(ok < 0)
			return sys->sprint("error: cannot connect to %s: %r", addr);

		return dorequest(conn.dfd, method, host, path, body);
	}
}

# Run a host command via /cmd device and capture output
runcmd(cmd: string): (string, string)
{
	(ok, nil) := sys->stat("/cmd");
	if(ok < 0)
		return (nil, "requires /cmd device");
	cmdctl := sys->open("/cmd/clone", Sys->ORDWR);
	if(cmdctl == nil)
		return (nil, sys->sprint("cannot open /cmd/clone: %r"));
	buf := array[32] of byte;
	n := sys->read(cmdctl, buf, len buf);
	if(n <= 0)
		return (nil, "cannot read cmd slot");
	cmdnum := string buf[0:n];
	datapath := "/cmd/" + cmdnum + "/data";
	data := sys->open(datapath, Sys->ORDWR);
	if(data == nil)
		return (nil, sys->sprint("cannot open %s: %r", datapath));
	fullcmd := "exec " + cmd;
	if(sys->fprint(cmdctl, "%s", fullcmd) < 0)
		return (nil, sys->sprint("cannot exec command: %r"));
	if(sys->fprint(cmdctl, "start") < 0)
		return (nil, sys->sprint("cannot start command: %r"));
	# Read all output
	output := "";
	readbuf := array[8192] of byte;
	while((n = sys->read(data, readbuf, len readbuf)) > 0)
		output += string readbuf[0:n];
	return (output, nil);
}

# Check if string is already an IP address (dotted decimal)
isipaddr(s: string): int
{
	dots := 0;
	for(i := 0; i < len s; i++) {
		if(s[i] == '.')
			dots++;
		else if(s[i] < '0' || s[i] > '9')
			return 0;
	}
	return dots == 3;
}

# Resolve hostname to IP via host DNS
resolve(hostname: string): (string, string)
{
	if(isipaddr(hostname))
		return (hostname, nil);
	cmd := "/bin/sh -c 'getent hosts " + hostname + " | head -1 | awk \"{print \\$1}\"'";
	(output, err) := runcmd(cmd);
	if(err != nil)
		return (nil, "DNS: " + err + " (use IP address instead)");
	ip := strip(output);
	if(ip == "")
		return (nil, sys->sprint("DNS resolution failed for %s", hostname));
	return (ip, nil);
}

# Shell-quote a string for safe inclusion in sh -c commands
shellquote(s: string): string
{
	result := "'";
	for(i := 0; i < len s; i++) {
		if(s[i] == '\'')
			result += "'\\''";
		else
			result[len result] = s[i];
	}
	result += "'";
	return result;
}

# HTTPS via host-side curl
curlrequest(method, host, port, path, body: string): string
{
	url := "https://" + host;
	if(port != HTTPS_PORT)
		url += ":" + port;
	url += path;

	cmd := "/bin/sh -c 'curl -s -X " + method;
	if(body != "")
		cmd += " -H \"Content-Type: application/json\" -d " + shellquote(body);
	cmd += " " + shellquote(url) + "'";

	(output, err) := runcmd(cmd);
	if(err != nil)
		return "error: HTTPS request failed: " + err;
	if(output == "")
		return "error: empty response from " + url;
	return output;
}

# Strip leading/trailing whitespace
strip(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r'))
		j--;
	if(i >= j)
		return "";
	return s[i:j];
}

# Perform HTTP request
dorequest(fd: ref Sys->FD, method, host, path, body: string): string
{
	# Build request
	if(path == "")
		path = "/";

	request := sys->sprint("%s %s HTTP/1.1\r\nHost: %s\r\n", method, path, host);
	request += "Connection: close\r\n";
	request += "User-Agent: Veltro/1.0\r\n";

	if(body != "") {
		request += sys->sprint("Content-Length: %d\r\n", len body);
		request += "Content-Type: application/json\r\n";
	}

	request += "\r\n";
	if(body != "")
		request += body;

	# Send request
	reqbytes := array of byte request;
	if(sys->write(fd, reqbytes, len reqbytes) < 0)
		return sys->sprint("error: write failed: %r");

	# Read response
	response := "";
	buf := array[8192] of byte;
	total := 0;

	while(total < MAX_RESPONSE) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		response += string buf[0:n];
		total += n;
	}

	if(response == "")
		return "error: empty response";

	# Parse response
	(status, headers, rbody) := parseresponse(response);
	if(status == "")
		return "error: invalid HTTP response";

	# Check status
	statuscode := 0;
	for(i := 0; i < len status && status[i] != ' '; i++)
		;
	if(i < len status) {
		for(j := i+1; j < len status && status[j] >= '0' && status[j] <= '9'; j++)
			statuscode = statuscode * 10 + (status[j] - '0');
	}

	# For HEAD, return headers
	if(method == "HEAD")
		return headers;

	# For error status, include status line
	if(statuscode >= 400)
		return sys->sprint("error: HTTP %d\n%s", statuscode, rbody);

	return rbody;
}

# Parse URL into components
parseurl(url: string): (string, string, string, string, string)
{
	scheme := "http";
	port := HTTP_PORT;
	i: int;

	# Check scheme
	if(len url > 8 && str->tolower(url[0:8]) == "https://") {
		scheme = "https";
		port = HTTPS_PORT;
		url = url[8:];
	} else if(len url > 7 && str->tolower(url[0:7]) == "http://") {
		url = url[7:];
	} else {
		return ("", "", "", "", "invalid URL: must start with http:// or https://");
	}

	# Find path
	path := "/";
	for(i = 0; i < len url; i++) {
		if(url[i] == '/') {
			path = url[i:];
			url = url[0:i];
			break;
		}
	}

	# Find port
	host := url;
	for(i = 0; i < len url; i++) {
		if(url[i] == ':') {
			host = url[0:i];
			port = url[i+1:];
			break;
		}
	}

	if(host == "")
		return ("", "", "", "", "invalid URL: no host");

	return (scheme, host, port, path, nil);
}

# Parse HTTP response
parseresponse(response: string): (string, string, string)
{
	# Find status line
	statusend := 0;
	for(; statusend < len response; statusend++) {
		if(response[statusend] == '\n')
			break;
	}
	if(statusend == 0)
		return ("", "", "");

	status := response[0:statusend];
	if(len status > 0 && status[len status - 1] == '\r')
		status = status[0:len status - 1];

	# Find headers end (blank line)
	headersend := statusend + 1;
	for(; headersend < len response - 1; headersend++) {
		if(response[headersend] == '\n' &&
		   (response[headersend+1] == '\n' || response[headersend+1] == '\r'))
			break;
	}

	headers := "";
	if(headersend > statusend + 1)
		headers = response[statusend+1:headersend];

	# Find body start
	bodystart := headersend + 1;
	if(bodystart < len response && response[bodystart] == '\r')
		bodystart++;
	if(bodystart < len response && response[bodystart] == '\n')
		bodystart++;

	body := "";
	if(bodystart < len response)
		body = response[bodystart:];

	return (status, headers, body);
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
