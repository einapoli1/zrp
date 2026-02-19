#!/usr/bin/env python3

with open('integration_workflow_test.go', 'r') as f:
    lines = f.readlines()

output_lines = []
i = 0
while i < len(lines):
    line = lines[i]
    output_lines.append(line)
    
    # Check if this is a test function declaration
    if line.startswith('func TestIntegration_') and '(t *testing.T)' in line:
        # Look ahead for the pattern we want to fix
        if i + 5 < len(lines):
            # Expected pattern:
            # func TestIntegration_...
            # \tif testing.Short() {
            # \t\tt.Skip(...)
            # \t}
            # (blank line)
            # \tclient := newTestClient(t)
            # \ttimestamp := time.Now().UnixNano()
            # (blank line or other line)
            
            # Check if lines i+5 and i+6 match the client/timestamp pattern
            if (i + 6 < len(lines) and
                '\tclient := newTestClient(t)' in lines[i + 5] and
                '\ttimestamp := time.Now().UnixNano()' in lines[i + 6]):
                
                # Check if the next line already has our variable declarations
                if i + 7 < len(lines) and 'var resp *http.Response' in lines[i + 7]:
                    # Already fixed, skip
                    pass
                else:
                    # Add lines up to and including timestamp line
                    for j in range(i + 1, i + 7):
                        output_lines.append(lines[j])
                    # Insert our variable declarations
                    output_lines.append('\tvar resp *http.Response\n')
                    output_lines.append('\tvar body []byte\n')
                    # Move index forward
                    i = i + 7
                    continue
    
    i += 1

# Write output
with open('integration_workflow_test.go', 'w') as f:
    f.writelines(output_lines)

print("Fixed test functions")

# Now fix `, body :=` to `, body =`
with open('integration_workflow_test.go', 'r') as f:
    content = f.read()

content = content.replace('_, body :=', '_, body =')

with open('integration_workflow_test.go', 'w') as f:
    f.write(content)

print("Fixed body assignments")
