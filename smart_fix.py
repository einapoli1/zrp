#!/usr/bin/env python3
import re

with open('integration_workflow_test.go.bak4', 'r') as f:
    lines = f.readlines()

# Track which test function we're in and whether we've seen a makeRequest declaration yet
in_new_test = False
has_declared_resp_body = False
current_test = None

result_lines = []

for i, line in enumerate(lines):
    # Detect start of new test functions (after line 805)
    if i >= 805 and line.startswith('func TestIntegration_'):
        in_new_test = True
        has_declared_resp_body = False
        current_test = line.strip()
    
    # If we're in a new test function
    if in_new_test and 'client.makeRequest' in line:
        # First makeRequest call - use :=
        if not has_declared_resp_body:
            if '\tresp, body = client.makeRequest' in line:
                line = line.replace('\tresp, body = client.makeRequest', '\tresp, body := client.makeRequest')
                has_declared_resp_body = True
            elif '\t_, body = client.makeRequest' in line:
                line = line.replace('\t_, body = client.makeRequest', '\t_, body := client.makeRequest')
                has_declared_resp_body = True
        # Subsequent calls - keep =
        # (no change needed)
    
    result_lines.append(line)

with open('integration_workflow_test.go', 'w') as f:
    f.writelines(result_lines)

print(f"Fixed {len(result_lines)} lines")
