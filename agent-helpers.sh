#!/bin/bash

# Robust agent control functions for cmux
# Usage: source this file before using agent functions

AGENT_WAITSEC=10
AGENT_MAXRETRIES=5
AGENT_RETRYDELAY=5

wait_for_session() {
    local surface=$1
    local retries=0
    
    while [ $retries -lt $AGENT_MAXRETRIES ]; do
        # Check if surface shows ready prompt (not "Unable to connect")
        local screen=$(cmux read-screen --surface $surface --scrollback --lines 5 2>/dev/null)
        
        if echo "$screen" | grep -q "Ask anything"; then
            echo "✓ Session ready on $surface"
            return 0
        fi
        
        if echo "$screen" | grep -q "Unable to connect"; then
            echo "⏳ Session not ready on $surface (retry $((retries+1))/$AGENT_MAXRETRIES)"
            retries=$((retries+1))
            sleep $AGENT_RETRYDELAY
        else
            # Check for other ready indicators
            if echo "$screen" | grep -qE "(big-pickle|OpenCode)"; then
                echo "✓ Session appears ready on $surface"
                return 0
            fi
        fi
    done
    
    echo "✗ Session failed to connect on $surface after $AGENT_MAXRETRIES attempts"
    return 1
}

send_to_agent() {
    local surface=$1
    local message=$2
    
    # First wait for session
    if ! wait_for_session "$surface"; then
        return 1
    fi
    
    # Now send the message
    cmux send --surface "$surface" "$message" && cmux send-key --surface "$surface" Enter
    return $?
}

send_and_wait() {
    local surface=$1
    local message=$2
    local wait_time=$3
    
    send_to_agent "$surface" "$message"
    sleep $wait_time
}

open_session() {
    local surface=$1
    local workdir=${2:-.}
    
    echo "Opening session on $surface..."
    cmux send --surface "$surface" "cd $workdir && opencode ." && cmux send-key --surface "$surface" Enter
    wait_for_session "$surface"
}

get_agent_status() {
    local surface=$1
    local screen=$(cmux read-screen --surface $surface --scrollback --lines 10 2>/dev/null)
    
    if echo "$screen" | grep -q "Unable to connect"; then
        echo "DISCONNECTED"
    elif echo "$screen" | grep -qE "(big-pickle|OpenCode|Ask anything)"; then
        echo "READY"
    else
        echo "UNKNOWN"
    fi
}

list_agents() {
    echo "Available agents:"
    cmux tree --all | grep -E "(surface:|workspace:)" | head -30
}

echo "✓ Agent control functions loaded"
echo "  Commands: wait_for_session, send_to_agent, send_and_wait, open_session, get_agent_status, list_agents"
