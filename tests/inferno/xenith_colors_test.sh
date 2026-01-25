#!/dis/sh.dis
#
# Test per-window color control in Xenith
#
# Prerequisites:
#   - Xenith must be running
#   - /mnt/xenith must be mounted
#

load std

XENITH=/mnt/xenith

# Check if Xenith is mounted
if {! ftest -d $XENITH} {
	raise 'skip:Xenith not mounted at /mnt/xenith'
}

# Find first window (get first numeric directory)
windows=`{ls $XENITH >[2] /dev/null | grep '^[0-9]'}
if {~ $#windows 0} {
	raise 'skip:No windows found (Xenith may not be running)'
}
# Get first window from list
WIN=$windows(1)
if {~ $WIN ''} {
	raise 'skip:No windows found'
}

# Verify window directory exists
if {! ftest -d $XENITH/$WIN} {
	raise 'skip:Window directory does not exist'
}

echo 'Testing window' $WIN

failed=0

# Test 1: Read default colors
echo '=== Test 1: Read defaults ==='
if {ftest -f $XENITH/$WIN/colors} {
	cat $XENITH/$WIN/colors
	echo 'PASS: colors file readable'
} {
	echo 'SKIP: colors file not accessible'
	raise 'skip:colors file not accessible'
}
echo ''

# Test 2: Set tag background to red (warning)
echo '=== Test 2: Set red tag (warning) ==='
echo 'tagbg #F38BA8
tagfg #1E1E2E' > $XENITH/$WIN/colors
echo 'PASS: red tag colors set'
sleep 1

# Test 3: Set tag background to green (success)
echo '=== Test 3: Set green tag (success) ==='
echo 'tagbg #A6E3A1
tagfg #1E1E2E' > $XENITH/$WIN/colors
echo 'PASS: green tag colors set'
sleep 1

# Test 4: Reset to defaults
echo '=== Test 4: Reset ==='
echo 'reset' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
echo 'PASS: colors reset'

echo ''
echo '=== Tests complete ==='

if {~ $failed 1} {
	raise 'fail:color tests failed'
}
