#!/dis/sh.dis
# Test script for temp file slot reclamation fix
# This tests that removing OEXCL allows stale file reuse
#
# NOTE: This test requires:
#   - A display (xenith needs a GUI)
#   - /tmp to be writable
#

load std

echo '=== Temp File Slot Reclamation Test ==='
echo ''

# Check if display is available by checking for draw device
if {! ftest -d /dev/draw} {
	raise 'skip:no display available (headless mode)'
}

# Check if xenith is available
if {! ftest -f /dis/xenith.dis} {
	raise 'skip:xenith not available'
}

# Get username
user=`{cat /dev/user}
if {~ $user ''} {
	raise 'skip:cannot determine user'
}
echo 'User:' $user

# Extract first 4 chars for prefix
userprefix=`{echo $user | sed 's/^\(....\).*/\1/'}
if {~ $userprefix ''} {
	userprefix=$user
}

# Clean any existing temp files first
echo 'Cleaning existing temp files...'
rm /tmp/*xenith >[2] /dev/null
echo 'Done.'
echo ''

# Test 1: Create stale files for all 26 slots to simulate exhaustion
echo '=== Test 1: Simulating temp file exhaustion ==='
echo 'Creating 26 stale files (A-Z) to fill all slots...'

# Use a fake PID
fakepid=99999

for letter in A B C D E F G H I J K L M N O P Q R S T U V W X Y Z {
    stalefile='/tmp/'^$letter^$fakepid^'.'^$userprefix^'xenith'
    echo 'stale data' > $stalefile >[2] /dev/null
}

# Verify files were created
count=`{ls /tmp/*xenith >[2] /dev/null | wc -l}
echo 'Stale files created:' $count
echo ''

# Test 2: Try to create a temp file using disk module
echo '=== Test 2: Testing temp file creation with stale files present ==='
echo 'Launching xenith (will exit after 2 seconds)...'
echo ''

# Run xenith in background and kill after timeout
xenith &
xpid=$apid
sleep 2

failed=0

# Check if xenith is still running
if {ftest -e /prog/$xpid/status} {
    echo 'SUCCESS: xenith started successfully!'
    echo 'This confirms the fix works - stale temp files were reclaimed.'
    # Kill xenith
    echo kill > /prog/$xpid/ctl >[2] /dev/null
} {
    echo 'FAILURE: xenith failed to start'
    echo 'Check if the OEXCL fix was applied correctly.'
    failed=1
}

echo ''
echo '=== Test Complete ==='

# Cleanup
rm /tmp/*xenith >[2] /dev/null

if {~ $failed 1} {
	raise 'fail:tempfile slot test failed'
} {}
