import re

with open('handler_reports_test.go', 'r') as f:
    content = f.read()

# Pattern 1: var report WOThroughputReport\n\tjson.NewDecoder(w.Body).Decode(&report)
pattern1 = r'(\tvar report WOThroughputReport)\n(\t+json\.NewDecoder\(w\.Body\)\.Decode\(&report\))'
replacement1 = r'\1\n\2\n\n\treportData, _ := json.Marshal(resp.Data)\n\tjson.Unmarshal(reportData, &report)'

# Actually, let's fix this more carefully - we need to first add the APIResponse decode, then the report unmarshal

# Find all instances where we decode directly into a report struct
patterns = [
    (r'(\t)(var report WOThroughputReport)\n(\t+)(json\.NewDecoder\(w\.Body\)\.Decode\(&report\))',
     r'\1var resp APIResponse\n\1\4(&resp)\n\n\1reportData, _ := json.Marshal(resp.Data)\n\1\2\n\1json.Unmarshal(reportData, &report)'),
    
    (r'(\t)(var report NCRSummaryReport)\n(\t+)(json\.NewDecoder\(w\.Body\)\.Decode\(&report\))',
     r'\1var resp APIResponse\n\1\4(&resp)\n\n\1reportData, _ := json.Marshal(resp.Data)\n\1\2\n\1json.Unmarshal(reportData, &report)'),
    
    (r'(\t)(var report LowStockItem)\n(\t+)(json\.NewDecoder\(w\.Body\)\.Decode\(&report\))',
     r'\1var resp APIResponse\n\1\4(&resp)\n\n\1reportData, _ := json.Marshal(resp.Data)\n\1\2\n\1json.Unmarshal(reportData, &report)'),
]

for pattern, repl in patterns:
    content = re.sub(pattern, repl, content)

with open('handler_reports_test.go', 'w') as f:
    f.write(content)

print("Fixed test decoders")
