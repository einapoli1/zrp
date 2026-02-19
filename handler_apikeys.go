package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type APIKey struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	KeyPrefix string  `json:"key_prefix"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	LastUsed  *string `json:"last_used"`
	ExpiresAt *string `json:"expires_at"`
	Enabled   int     `json:"enabled"`
}

type CreateAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func generateAPIKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "zrp_" + hex.EncodeToString(b), nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, key_prefix, created_by, created_at, last_used, expires_at, enabled FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var lastUsed, expiresAt *string
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.CreatedBy, &k.CreatedAt, &lastUsed, &expiresAt, &k.Enabled); err != nil {
			continue
		}
		k.LastUsed = lastUsed
		k.ExpiresAt = expiresAt
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []APIKey{}
	}
	jsonResp(w, keys)
}

func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}
	if req.Name == "" {
		jsonErr(w, "Name is required", 400)
		return
	}

	key, err := generateAPIKey()
	if err != nil {
		jsonErr(w, "Failed to generate key", 500)
		return
	}

	keyHash := hashAPIKey(key)
	keyPrefix := key[:12] // "zrp_" + first 8 hex chars

	var expiresAt interface{}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		expiresAt = *req.ExpiresAt
	}

	result, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, expires_at) VALUES (?, ?, ?, ?)`,
		req.Name, keyHash, keyPrefix, expiresAt)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         id,
		"name":       req.Name,
		"key":        key,
		"full_key":   key, // For frontend compatibility
		"key_prefix": keyPrefix,
		"created_by": "admin", // TODO: get from session
		"created_at": time.Now().Format(time.RFC3339),
		"enabled":    1,
		"status":     "active",
		"message":    "Store this key securely. It will not be shown again.",
	})
}

func handleDeleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	res, err := db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "API key not found", 404)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func handleToggleAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Enabled int `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "Invalid body", 400)
		return
	}
	_, err := db.Exec("UPDATE api_keys SET enabled = ? WHERE id = ?", body.Enabled, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// validateBearerToken checks an Authorization: Bearer token against the DB.
// Returns true if valid (and updates last_used).
func validateBearerToken(token string) bool {
	if !strings.HasPrefix(token, "zrp_") {
		return false
	}
	keyHash := hashAPIKey(token)
	var id int
	var enabled int
	var expiresAt *string
	err := db.QueryRow("SELECT id, enabled, expires_at FROM api_keys WHERE key_hash = ?", keyHash).Scan(&id, &enabled, &expiresAt)
	if err != nil || enabled == 0 {
		return false
	}
	if expiresAt != nil && *expiresAt != "" {
		exp, err := time.Parse("2006-01-02T15:04:05Z", *expiresAt)
		if err != nil {
			exp, err = time.Parse("2006-01-02", *expiresAt)
		}
		if err == nil && time.Now().After(exp) {
			return false
		}
	}
	db.Exec("UPDATE api_keys SET last_used = ? WHERE id = ?", time.Now().Format("2006-01-02 15:04:05"), id)
	return true
}

func init() {
	// Register the api_keys table creation in a post-init hook
	_ = fmt.Sprintf("api_keys handler loaded")
}
