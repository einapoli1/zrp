#!/usr/bin/env python3

# Read original clean file (lines 1-805)
with open('integration_workflow_test.go.orig', 'r') as f:
    orig_lines = f.readlines()

# Read backup with new tests (lines 806+)
with open('integration_workflow_test.go.bak4', 'r') as f:
    bak_lines = f.readlines()

# Start with original
result = orig_lines.copy()

# Add new tests from line 806 onwards
new_test_lines = bak_lines[805:]

# Fix variable declarations in new tests
current_func = None
resp_declared = False
body_declared = False

for i, line in enumerate(new_test_lines):
    # Detect new test function
    if line.startswith('func TestIntegration_'):
        current_func = line.split('(')[0].strip()
        resp_declared = False
        body_declared = False
    
    # Handle makeRequest calls
    if current_func and 'client.makeRequest' in line:
        # Determine if we need resp or just body
        # Look ahead in the function to see if resp is ever used
        needs_resp = False
        for j in range(i, min(i+100, len(new_test_lines))):  # Look ahead 100 lines
            if new_test_lines[j].startswith('func '):  # Hit next function
                break
            if '\tresp.' in new_test_lines[j] or 'resp.Status' in new_test_lines[j]:
                needs_resp = True
                break
        
        # Fix the declaration
        if '\tresp, body = client.makeRequest' in line or '\t_, body = client.makeRequest' in line:
            if not body_declared:
                # First occurrence
                if needs_resp and not resp_declared:
                    line = line.replace('\t_, body = client', '\tresp, body := client')
                    line = line.replace('\tresp, body = client', '\tresp, body := client')
                    resp_declared = True
                    body_declared = True
                elif not needs_resp:
                    line = line.replace('\tresp, body = client', '\t_, body := client')
                    line = line.replace('\t_, body = client', '\t_, body := client')
                    body_declared = True
                else:  # resp already declared, just need body
                    line = line.replace('\t_, body = client', '\t_, body := client')
                    line = line.replace('\tresp, body = client', '\tresp, body := client')
                    body_declared = True
            # else: subsequent calls, keep as `=`
    
    new_test_lines[i] = line

# Combine
result.extend(new_test_lines)

with open('integration_workflow_test.go', 'w') as f:
    f.writelines(result)

print(f"Created file with {len(result)} lines")
