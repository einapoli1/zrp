#!/usr/bin/env python3
"""
Fix integration_workflow_test.go compilation errors.
The original 4 tests (lines 1-805) should not be modified.
The new 8 tests (lines 806+) need fixes for variable declarations.
"""

with open('integration_workflow_test.go', 'r') as f:
    lines = f.readlines()

# Only modify lines 806 onwards (index 805+)
for i in range(805, len(lines)):
    line = lines[i]
    
    # Fix: Change `, body = client.makeRequest` to `, body := client.makeRequest`
    # ONLY when it's the first occurrence in a test function (declaration)
    if '\t_, body = client.makeRequest' in line:
        lines[i] = line.replace('\t_, body = client.makeRequest', '\t_, body := client.makeRequest')
    
    # Fix: Change `resp, body = client.makeRequest` to `resp, body := client.makeRequest` 
    # ONLY when it's the first occurrence in a test function (declaration)
    elif '\tresp, body = client.makeRequest' in line and '(:=' not in line:
        # Check if this looks like a first declaration (previous few lines should have test start)
        # This is a heuristic - look back 20 lines for "func Test"
        is_first_in_func = False
        for j in range(max(0, i-20), i):
            if 'func TestIntegration_' in lines[j]:
                # Check if there's already a := between func declaration and current line
                has_declaration = False
                for k in range(j, i):
                    if 'client.makeRequest' in lines[k] and ':=' in lines[k]:
                        has_declaration = True
                        break
                if not has_declaration:
                    is_first_in_func = True
                break
        
        if is_first_in_func:
            lines[i] = line.replace('\tresp, body = client.makeRequest', '\tresp, body := client.makeRequest')

with open('integration_workflow_test.go', 'w') as f:
    f.writelines(lines)

print(f"Fixed integration_workflow_test.go ({len(lines)} lines)")
