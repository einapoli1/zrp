#!/bin/bash
cd ~/.openclaw/workspace/zrp
echo "Running notification list endpoint tests..."
go test -v -run "^TestHandleListNotifications_(All|UnreadOnly|Empty|Limit)$" 2>&1
