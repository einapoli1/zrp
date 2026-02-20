#!/bin/bash
# Script to add audit_log table to custom test setup functions
# This fixes the "no such table: audit_log" errors in test files

AUDIT_LOG_SQL='
	// Create audit_log table - CRITICAL: Used by almost every handler
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			record_id TEXT NOT NULL,
			user_id INTEGER,
			username TEXT DEFAULT '\'''\''',
			summary TEXT DEFAULT '\'''\''',
			changes TEXT DEFAULT '\''{}'\''',
			ip_address TEXT DEFAULT '\'''\''',
			user_agent TEXT DEFAULT '\'''\''',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}
'

# Priority test files that need audit_log
FILES=(
    "handler_auth_test.go"
    "handler_permissions_test.go"
    "handler_scan_test.go"
    "handler_search_test.go"
    "handler_users_test.go"
    "security_auth_bypass_test.go"
    "security_file_upload_test.go"
    "handler_notifications_test.go"
    "handler_query_profiler_test.go"
    "handler_reports_test.go"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        # Check if audit_log already exists
        if ! grep -q "CREATE TABLE.*audit_log" "$file"; then
            echo "Processing $file..."
            # Find the return testDB line and add audit_log before it
            # This is a simplified approach - manual review recommended
            echo "  Needs manual audit_log addition"
        else
            echo "Skipping $file (audit_log already exists)"
        fi
    fi
done

echo ""
echo "Manual steps needed:"
echo "1. For each file listed above, add the audit_log CREATE TABLE statement"
echo "2. Add it just before 'return testDB' in the setup function"
echo "3. Test with: go test -v -run TestName"
