package main

import (
	"errors"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ValidTableNames is a whitelist of allowed table names
var ValidTableNames = map[string]bool{
	"parts":                  true,
	"ecos":                   true,
	"users":                  true,
	"sessions":               true,
	"api_keys":               true,
	"categories":             true,
	"custom_columns":         true,
	"work_orders":            true,
	"purchase_orders":        true,
	"po_lines":               true,
	"receiving":              true,
	"inventory":              true,
	"inventory_transactions": true,
	"ncrs":                   true,
	"capas":                  true,
	"rmas":                   true,
	"devices":                true,
	"campaigns":              true,
	"campaign_devices":       true,
	"shipments":              true,
	"quotes":                 true,
	"docs":                   true,
	"doc_versions":           true,
	"vendors":                true,
	"prices":                 true,
	"email_config":           true,
	"email_log":              true,
	"notifications":          true,
	"notification_prefs":     true,
	"attachments":            true,
	"undo_log":               true,
	"part_changes":           true,
	"changes":                true,
	"permissions":            true,
	"role_permissions":       true,
	"rfqs":                   true,
	"rfq_quotes":             true,
	"product_pricing":        true,
	"cost_analysis":          true,
	"sales_orders":           true,
	"invoices":               true,
	"field_reports":          true,
	"saved_searches":         true,
	"search_history":         true,
	"password_history":       true,
	"password_reset_tokens":  true,
}

// ValidColumnNames is a whitelist of commonly used column names
var ValidColumnNames = map[string]bool{
	"id":                      true,
	"ipn":                     true,
	"mpn":                     true,
	"description":             true,
	"category":                true,
	"status":                  true,
	"created_at":              true,
	"updated_at":              true,
	"created_by":              true,
	"updated_by":              true,
	"name":                    true,
	"email":                   true,
	"username":                true,
	"role":                    true,
	"active":                  true,
	"title":                   true,
	"content":                 true,
	"quantity":                true,
	"price":                   true,
	"vendor_id":               true,
	"part_id":                 true,
	"eco_id":                  true,
	"user_id":                 true,
	"device_id":               true,
	"campaign_id":             true,
	"shipment_id":             true,
	"quote_id":                true,
	"doc_id":                  true,
	"po_id":                   true,
	"wo_id":                   true,
	"ncr_id":                  true,
	"capa_id":                 true,
	"rma_id":                  true,
	"invoice_id":              true,
	"sales_order_id":          true,
	"field_report_id":         true,
	"revision":                true,
	"approved":                true,
	"approved_by":             true,
	"approved_at":             true,
	"failed_login_attempts":   true,
	"locked_until":            true,
}

// ValidateTableName checks if a table name is in the whitelist
func ValidateTableName(table string) error {
	if !ValidTableNames[table] {
		return errors.New("invalid table name")
	}
	return nil
}

// ValidateColumnName checks if a column name is in the whitelist
func ValidateColumnName(column string) error {
	if !ValidColumnNames[column] {
		return errors.New("invalid column name")
	}
	return nil
}

// SanitizeIdentifier ensures an identifier contains only safe characters
// This is a defense-in-depth measure even with whitelisting
func SanitizeIdentifier(identifier string) (string, error) {
	// Only allow alphanumeric, underscore
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validPattern.MatchString(identifier) {
		return "", errors.New("invalid identifier format")
	}
	return identifier, nil
}

// ValidateAndSanitizeTable validates and sanitizes a table name
func ValidateAndSanitizeTable(table string) (string, error) {
	sanitized, err := SanitizeIdentifier(table)
	if err != nil {
		return "", err
	}
	if err := ValidateTableName(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

// ValidateAndSanitizeColumn validates and sanitizes a column name
func ValidateAndSanitizeColumn(column string) (string, error) {
	sanitized, err := SanitizeIdentifier(column)
	if err != nil {
		return "", err
	}
	if err := ValidateColumnName(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

// ValidatePasswordStrength checks password complexity
func ValidatePasswordStrength(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString
		hasSpecial = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>_\-+=]`).MatchString
	)

	checks := 0
	if hasUpper(password) {
		checks++
	}
	if hasLower(password) {
		checks++
	}
	if hasNumber(password) {
		checks++
	}
	if hasSpecial(password) {
		checks++
	}

	if checks < 3 {
		return errors.New("password must contain at least 3 of: uppercase, lowercase, numbers, special characters")
	}

	return nil
}

// Password history and reset token functions

var (
	ErrPasswordReused = errors.New("password was recently used, please choose a different password")
	ErrInvalidToken   = errors.New("invalid or expired token")
)

// CheckPasswordHistory verifies a password hasn't been used recently
func CheckPasswordHistory(userID int, newPassword string) error {
	// Get last 5 passwords from history
	rows, err := db.Query(`
		SELECT password_hash FROM password_history 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT 5`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var oldHash string
		if err := rows.Scan(&oldHash); err != nil {
			continue
		}

		// Check if new password matches old password
		if bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(newPassword)) == nil {
			return ErrPasswordReused
		}
	}

	return nil
}

// AddPasswordHistory adds a password to the user's history
func AddPasswordHistory(userID int, passwordHash string) error {
	_, err := db.Exec(
		"INSERT INTO password_history (user_id, password_hash, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		userID, passwordHash)
	return err
}

// GeneratePasswordResetToken creates a reset token valid for 1 hour
func GeneratePasswordResetToken(username string) (string, error) {
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return "", err
	}

	token := generateToken() // Reuse existing token generator
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = db.Exec(
		"INSERT INTO password_reset_tokens (token, user_id, created_at, expires_at, used) VALUES (?, ?, CURRENT_TIMESTAMP, ?, 0)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidatePasswordResetToken checks if a token is valid and not expired
func ValidatePasswordResetToken(token string) (bool, int) {
	var userID int
	var expiresAt string

	err := db.QueryRow(
		"SELECT user_id, expires_at FROM password_reset_tokens WHERE token = ? AND used = 0",
		token).Scan(&userID, &expiresAt)
	if err != nil {
		return false, 0
	}

	// Parse expiration time
	expires, err := time.Parse("2006-01-02 15:04:05", expiresAt)
	if err != nil {
		return false, 0
	}

	// Check if expired
	if time.Now().After(expires) {
		return false, 0
	}

	return true, userID
}

// ResetPasswordWithToken resets a password using a valid reset token
func ResetPasswordWithToken(token, newPassword string) error {
	valid, userID := ValidatePasswordResetToken(token)
	if !valid {
		return ErrInvalidToken
	}

	// Validate new password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// Check password history
	if err := CheckPasswordHistory(userID, newPassword); err != nil {
		return err
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), userID)
	if err != nil {
		return err
	}

	// Mark token as used
	_, err = db.Exec("UPDATE password_reset_tokens SET used = 1 WHERE token = ?", token)
	if err != nil {
		return err
	}

	// Add to password history
	AddPasswordHistory(userID, string(hash))

	return nil
}

// Account lockout functions

const (
	MaxFailedLoginAttempts = 10
	AccountLockoutDuration = 15 * time.Minute
)

// IncrementFailedLoginAttempts increments the failed login counter
func IncrementFailedLoginAttempts(username string) error {
	_, err := db.Exec(`
		UPDATE users 
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE 
		        WHEN failed_login_attempts + 1 >= ? THEN datetime('now', '+15 minutes')
		        ELSE locked_until 
		    END
		WHERE username = ?`, MaxFailedLoginAttempts, username)
	return err
}

// ResetFailedLoginAttempts resets the failed login counter after successful login
func ResetFailedLoginAttempts(username string) error {
	_, err := db.Exec(`
		UPDATE users 
		SET failed_login_attempts = 0, locked_until = NULL 
		WHERE username = ?`, username)
	return err
}

// IsAccountLocked checks if an account is currently locked
func IsAccountLocked(username string) (bool, error) {
	var lockedUntil *string
	err := db.QueryRow("SELECT locked_until FROM users WHERE username = ?", username).Scan(&lockedUntil)
	if err != nil {
		return false, err
	}

	if lockedUntil == nil {
		return false, nil
	}

	// Try multiple time formats (SQLite can return different formats)
	var lockTime time.Time
	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // SQLite default
		time.RFC3339Nano,
	}
	
	var parseErr error
	for _, format := range formats {
		lockTime, parseErr = time.Parse(format, *lockedUntil)
		if parseErr == nil {
			break
		}
	}
	
	if parseErr != nil {
		// Could not parse time, assume not locked
		return false, nil
	}

	// If lock time is in the future, account is locked
	if time.Now().Before(lockTime) {
		return true, nil
	}

	// Lock has expired, clear it
	ResetFailedLoginAttempts(username)
	return false, nil
}
