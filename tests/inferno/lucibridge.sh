#!/dis/sh.dis
load std

# Mount llm9p (must be running on host with -backend echo)
mkdir -p /n/llm
mount -A tcp!127.0.0.1!5640 /n/llm

# Start UI server
luciuisrv
sleep 1

# Create activity for the bridge
echo 'activity create BridgeTest' > /mnt/ui/ctl
echo activity created:
cat /mnt/ui/activity/0/label

# Start bridge in background
lucibridge -v -a 0 &
sleep 1

# Send human input (writes to conversation/input, bridge picks it up)
echo 'What is the meaning of life?' > /mnt/ui/activity/0/conversation/input
sleep 2

# Check conversation messages
echo msg 0:
cat /mnt/ui/activity/0/conversation/0
echo msg 1:
cat /mnt/ui/activity/0/conversation/1

echo PASS
