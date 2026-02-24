#!/bin/bash
go test -v -run 'TestHandlerQueryConsistency' 2>&1 | grep -A 100 "Expected ~50+" || echo "Test passed or different error"
