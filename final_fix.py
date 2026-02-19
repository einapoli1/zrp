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

# Fix variable declarations in new tests only
has_declared = {}
for i, line in enumerate(new_test_lines):
    # Detect new test function
    if line.startswith('func TestIntegration_'):
        current_func = line.split('(')[0].strip()
        has_declared[current_func] = False
    
    # Fix first makeRequest in each function
    if 'client.makeRequest' in line and current_func:
        if not has_declared.get(current_func, False):
            # First occurrence - use :=
            if '\tresp, body = client.makeRequest' in line:
                line = line.replace('\tresp, body = client.makeRequest', '\tresp, body := client.makeRequest')
                has_declared[current_func] = True
            elif '\t_, body = client.makeRequest' in line:
                line = line.replace('\t_, body = client.makeRequest', '\t_, body := client.makeRequest')
                has_declared[current_func] = True
        # Else keep as = (assignment)
    
    new_test_lines[i] = line

# Combine
result.extend(new_test_lines)

with open('integration_workflow_test.go', 'w') as f:
    f.writelines(result)

print(f"Created file with {len(result)} lines ({len(orig_lines)} original + {len(new_test_lines)} new)")
