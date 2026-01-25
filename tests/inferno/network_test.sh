#!/dis/sh.dis
#
# TCP/IP Network Stack Tests for InferNode
# Runs inside the emulator to test network functionality
#

load std

echo '=========================================='
echo 'TCP/IP Network Stack Tests'
echo '=========================================='
echo ''

failed=0

echo 'Test 1: Check network devices exist'
if {ftest -d /net} {
	echo 'PASS: net device exists'
} {
	echo 'FAIL: net device missing'
	failed=1
}

if {ftest -d /net/tcp} {
	echo 'PASS: tcp device exists'
} {
	echo 'FAIL: tcp device missing'
	failed=1
}

echo ''
echo 'Test 2: Can we allocate a TCP connection?'
if {ftest -f /net/tcp/clone} {
	connid=`{cat /net/tcp/clone}
	if {! ~ $connid ''} {
		echo 'PASS: TCP clone returned connection ID:' $connid
	} {
		echo 'FAIL: TCP clone did not return valid ID'
		failed=1
	}
} {
	echo 'FAIL: TCP clone not accessible'
	failed=1
}

echo ''
echo 'Test 3: Check DNS resolution'
if {ftest -d /net/dns} {
	echo 'PASS: DNS device exists'
} {
	echo 'SKIP: DNS device not configured (optional)'
}

echo ''
echo '=========================================='
echo 'Network Tests Complete'
echo '=========================================='

if {~ $failed 1} {
	raise 'fail:network tests failed'
} {}
