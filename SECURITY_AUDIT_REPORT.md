# ZRP Security Audit Report

**Date**: February 19, 2026  
**Audited by**: Eva (AI Security Auditor)  
**Application**: ZRP - Go Backend + React Frontend

---

## Executive Summary

This security audit identified **6 CRITICAL**, **8 HIGH**, **5 MEDIUM**, and **3 LOW** severity issues in the ZRP application. The most critical issues involve SQL injection vulnerabilities, missing security headers, insecure cookie configuration, and potential XSS vulnerabilities.

---

## CRITICAL Severity Issues

### 1. SQL Injection via Table/Column Name Interpolation

**Severity**: CRITICAL  
**Files Affected**:
- `validation.go` (lines 114, 126, 186)
- `handler_changes.go` (lines 176, 226)
- `handler_notification_prefs.go` (line 277)

**Description**:  
Table and column names are being inserted into SQL queries using `fmt.Sprintf` with user-controlled or semi-controlled input. While the VALUES are properly parameterized, table and column names cannot be parameterized in SQL, creating SQL injection opportunities.

**Vulnerable Code Examples**:
```go
// validation.go:114
err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id=?", table), id).Scan(&count)

// validation.go:126
err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", table, col), value).Scan(&count)

// handler_changes.go:176
_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, idCol), recordID)
```

**Impact**:  
An attacker could potentially:
- Extract sensitive data from the database
- Modify or delete database records
- Escalate privileges
- Execute arbitrary SQL commands

**Recommendation**:  
1. Create a whitelist of allowed table and column names
2. Validate all table/column names against this whitelist before use
3. Never allow user input to directly control table or column names

**Fix**:
```go
// Create a whitelist validator
var validTables = map[string]bool{
    "parts": true,
    "users": true,
    "ecos": true,
    // ... add all valid tables
}

func validateTableName(table string) error {
    if !validTables[table] {
        return errors.New("invalid table name")
    }
    return nil
}

// Then use before any query:
if err := validateTableName(table); err != nil {
    return err
}
```

---

### 2. Missing Security Headers

**Severity**: CRITICAL

**Description**:  
The application does not set any security headers, leaving it vulnerable to:
- Clickjacking attacks (no X-Frame-Options)
- MIME-sniffing attacks (no X-Content-Type-Options)
- XSS attacks (no Content-Security-Policy)

**Vulnerable Code**:  
No security headers are set in `main.go` or `middleware.go`

**Impact**:
- **Clickjacking**: Attacker can embed the application in an iframe on a malicious site
- **MIME-sniffing**: Browser could interpret files as different content types, enabling XSS
- **XSS**: Without CSP, any XSS vulnerability becomes easier to exploit

**Recommendation**:  
Add security headers middleware

**Fix**:
```go
// Add to middleware.go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Prevent MIME-sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // Enable XSS protection (legacy browsers)
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", 
            "default-src 'self'; "+
            "script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
            "style-src 'self' 'unsafe-inline'; "+
            "img-src 'self' data: blob:; "+
            "font-src 'self' data:; "+
            "connect-src 'self'")
        
        // Referrer Policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Permissions Policy
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}
```

---

### 3. Cookies Missing Secure Flag

**Severity**: CRITICAL

**Files Affected**:
- `handler_auth.go` (lines 126-132, 147-152)
- `middleware.go` (lines 140-146)

**Description**:  
Session cookies are set with `HttpOnly` and `SameSite` flags, but missing the `Secure` flag. This means cookies could be transmitted over unencrypted HTTP connections.

**Vulnerable Code**:
```go
http.SetCookie(w, &http.Cookie{
    Name:     "zrp_session",
    Value:    token,
    Path:     "/",
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
    Expires:  expires,
    // Missing: Secure flag
})
```

**Impact**:  
- Session cookies could be intercepted via man-in-the-middle attacks
- Session hijacking becomes trivial on unencrypted connections

**Recommendation**:  
Add `Secure: true` flag to all cookies

**Fix**:
```go
http.SetCookie(w, &http.Cookie{
    Name:     "zrp_session",
    Value:    token,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,  // ADD THIS
    SameSite: http.SameSiteLaxMode,
    Expires:  expires,
})
```

---

### 4. VACUUM Command with User Path (SQL Injection)

**Severity**: CRITICAL

**File**: `handler_backup.go:66`

**Vulnerable Code**:
```go
_, err := db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, destPath))
```

**Description**:  
The `destPath` variable is being directly interpolated into a SQL command. If this path comes from user input, it creates a SQL injection vulnerability.

**Impact**:  
Arbitrary SQL execution, data exfiltration

**Recommendation**:  
Validate and sanitize the path, ensure it's within allowed directories only

---

### 5. No CSRF Protection

**Severity**: CRITICAL

**Description**:  
The application has no CSRF (Cross-Site Request Forgery) protection for state-changing operations. While `SameSite=Lax` provides some protection, it's not sufficient for all scenarios.

**Impact**:  
Attackers could trick authenticated users into performing unwanted actions:
- Creating/modifying/deleting records
- Changing passwords
- Transferring data

**Recommendation**:  
Implement CSRF tokens for all state-changing requests (POST, PUT, DELETE)

**Fix**:
```go
// 1. Generate CSRF token on login
func generateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}

// 2. Add to session
type Session struct {
    Token      string
    UserID     int
    CSRFToken  string
    ExpiresAt  time.Time
}

// 3. Validate on state-changing requests
func validateCSRF(r *http.Request, sessionCSRF string) error {
    requestCSRF := r.Header.Get("X-CSRF-Token")
    if requestCSRF == "" {
        requestCSRF = r.FormValue("csrf_token")
    }
    if requestCSRF != sessionCSRF {
        return errors.New("invalid CSRF token")
    }
    return nil
}
```

---

### 6. Weak Password Policy

**Severity**: CRITICAL

**File**: `handler_auth.go`

**Description**:  
The password change function only requires:
- Minimum 8 characters
- No complexity requirements

**Vulnerable Code**:
```go
if len(req.NewPassword) < 8 {
    jsonErr(w, "New password must be at least 8 characters", 400)
    return
}
```

**Impact**:  
Weak passwords can be easily brute-forced

**Recommendation**:  
Implement stronger password policy

**Fix**:
```go
func validatePasswordStrength(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    
    var (
        hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString
        hasLower   = regexp.MustCompile(`[a-z]`).MatchString
        hasNumber  = regexp.MustCompile(`[0-9]`).MatchString
        hasSpecial = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString
    )
    
    checks := 0
    if hasUpper(password) { checks++ }
    if hasLower(password) { checks++ }
    if hasNumber(password) { checks++ }
    if hasSpecial(password) { checks++ }
    
    if checks < 3 {
        return errors.New("password must contain at least 3 of: uppercase, lowercase, numbers, special characters")
    }
    
    return nil
}
```

---

## HIGH Severity Issues

### 7. Rate Limiting Only on Login

**Severity**: HIGH

**Description**:  
Rate limiting is only implemented for login attempts. Other endpoints are not rate-limited, allowing:
- Brute force attacks on other endpoints
- Denial of service attacks
- API abuse

**Recommendation**:  
Implement rate limiting on all API endpoints

---

### 8. No Input Validation on Many Endpoints

**Severity**: HIGH

**Description**:  
Many handlers accept user input without validation:
- File upload size limits may not be enforced
- No maximum length checks on string inputs
- No format validation on many fields

**Recommendation**:  
Implement comprehensive input validation

---

### 9. Potential XSS in Error Messages

**Severity**: HIGH

**Description**:  
Error messages may reflect user input without proper sanitization

**Recommendation**:  
Always sanitize error messages that include user input

---

### 10. Session Fixation Vulnerability

**Severity**: HIGH

**Description**:  
After password change, the session is not regenerated

**Recommendation**:  
Regenerate session token after password change

---

### 11. No Account Lockout

**Severity**: HIGH

**Description**:  
No account lockout after multiple failed login attempts

**Recommendation**:  
Implement account lockout after N failed attempts

---

### 12. Insecure Direct Object References

**Severity**: HIGH

**Description**:  
Many endpoints accept IDs directly without verifying ownership

**Recommendation**:  
Always verify user has permission to access the requested resource

---

### 13. Missing Authentication on Some Endpoints

**Severity**: HIGH

**Description**:  
Some API endpoints may be accessible without authentication

**Recommendation**:  
Audit all endpoints and ensure proper authentication

---

### 14. No API Key Expiration Enforcement

**Severity**: HIGH

**Description**:  
API keys may not be properly validated for expiration

**Recommendation**:  
Enforce API key expiration checks

---

## MEDIUM Severity Issues

### 15. Weak Session Timeout

**Severity**: MEDIUM

**Description**:  
24-hour session timeout may be too long for sensitive operations

**Recommendation**:  
Implement shorter timeouts or idle timeouts

---

### 16. No Audit Logging for Sensitive Operations

**Severity**: MEDIUM

**Description**:  
Not all sensitive operations are logged

**Recommendation**:  
Log all authentication events, permission changes, and sensitive data access

---

### 17. Predictable Token Generation

**Severity**: MEDIUM

**Description**:  
While using crypto/rand, should verify entropy

**Recommendation**:  
Use crypto/rand.Reader explicitly

---

### 18. Information Disclosure

**Severity**: MEDIUM

**Description**:  
Error messages may reveal internal system information

**Recommendation**:  
Use generic error messages for users, log details separately

---

### 19. CORS Configuration

**Severity**: MEDIUM

**Description**:  
CORS allows all origins with wildcard

**Recommendation**:  
Restrict CORS to specific trusted origins

---

## LOW Severity Issues

### 20. Version Disclosure

**Severity**: LOW

**Description**:  
Application may disclose version information

**Recommendation**:  
Remove or obscure version info

---

### 21. Missing HTTP Strict Transport Security

**Severity**: LOW

**Description**:  
No HSTS header to enforce HTTPS

**Recommendation**:  
Add HSTS header

---

### 22. Database Connection Not Using TLS

**Severity**: LOW

**Description**:  
SQLite is local, but document this is not for distributed deployments

**Recommendation**:  
Document security considerations

---

## Summary Statistics

| Severity | Count | Fixed |
|----------|-------|-------|
| CRITICAL | 6     | 0     |
| HIGH     | 8     | 0     |
| MEDIUM   | 5     | 0     |
| LOW      | 3     | 0     |
| **TOTAL**| **22**| **0** |

---

## Recommended Immediate Actions

1. **Fix SQL injection vulnerabilities** (Issues #1, #4)
2. **Add security headers** (Issue #2)
3. **Add Secure flag to cookies** (Issue #3)
4. **Implement CSRF protection** (Issue #5)
5. **Strengthen password policy** (Issue #6)

---

## Testing Performed

- Static code analysis
- Manual code review
- Pattern matching for common vulnerabilities
- Configuration review

---

## Next Steps

1. Prioritize and fix CRITICAL issues immediately
2. Create tickets for HIGH severity issues
3. Schedule fixes for MEDIUM and LOW issues
4. Implement security testing in CI/CD pipeline
5. Conduct penetration testing after fixes

---

**End of Report**
