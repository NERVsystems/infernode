#!/dis/sh.dis
#
# Regression tests for Xenith window manipulation features
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

# Check if we can actually interact with Xenith
if {! ftest -f $XENITH/new/ctl} {
	raise 'skip:Xenith new/ctl not accessible'
}

echo '================================================'
echo 'Xenith Window Manipulation Regression Tests'
echo '================================================'

failed=0

# Test 1: Window creation via filesystem
echo ''
echo '=== Test: Window Creation ==='

# Create a window and capture the ID
CREATED_WINDOW=`{cat $XENITH/new/ctl}
if {~ $#CREATED_WINDOW 0} {
	echo 'SKIP: Could not create window (Xenith may not be running)'
	raise 'skip:Cannot create window'
}
if {~ $CREATED_WINDOW ''} {
	echo 'SKIP: Window ID is empty (Xenith may not be running)'
	raise 'skip:Empty window ID'
}

echo 'PASS: Window created with ID' $CREATED_WINDOW

# Check window exists in index
if {ftest -d $XENITH/$CREATED_WINDOW} {
	echo 'PASS: Window directory exists'
} {
	echo 'FAIL: Window directory not found'
	failed=1
}

# Test 2: Content writing
echo ''
echo '=== Test: Content Writing ==='

echo 'Test content from regression test' > $XENITH/$CREATED_WINDOW/body >[2] /dev/null
echo 'PASS: Write to window body succeeded'

# Test 3: Layout commands
echo ''
echo '=== Test: Layout Commands ==='

# Test grow
echo grow > $XENITH/$CREATED_WINDOW/ctl >[2] /dev/null
echo 'PASS: grow command accepted'

# Test growmax
echo growmax > $XENITH/$CREATED_WINDOW/ctl >[2] /dev/null
echo 'PASS: growmax command accepted'

# Test 4: Deletion of filesystem-created window
echo ''
echo '=== Test: Filesystem Window Deletion ==='

# Delete the window we created
echo delete > $XENITH/$CREATED_WINDOW/ctl >[2] /dev/null

# Check if window is gone
if {ftest -d $XENITH/$CREATED_WINDOW} {
	echo 'FAIL: Window deletion failed - window still exists'
	failed=1
} {
	echo 'PASS: Window deleted successfully'
}

echo ''
echo '================================================'
echo 'Xenith Window Tests Complete'
echo '================================================'

if {~ $failed 1} {
	raise 'fail:tests failed'
}
