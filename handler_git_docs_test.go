package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHandleGetGitDocsSettings_Default(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/settings/git-docs", nil)
	w := httptest.NewRecorder()

	handleGetGitDocsSettings(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	
	cfgBytes, _ := json.Marshal(resp.Data)
	var cfg GitDocsConfig
	json.Unmarshal(cfgBytes, &cfg)

	if cfg.Branch != "main" {
		t.Errorf("Expected default branch 'main', got '%s'", cfg.Branch)
	}
}

func TestHandleGetGitDocsSettings_HidesToken(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Set a token
	db.Exec("INSERT INTO app_settings (key, value) VALUES ('git_docs_token', 'secret123')")

	req := httptest.NewRequest("GET", "/api/settings/git-docs", nil)
	w := httptest.NewRecorder()

	handleGetGitDocsSettings(w, req)

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	
	cfgBytes, _ := json.Marshal(resp.Data)
	var cfg GitDocsConfig
	json.Unmarshal(cfgBytes, &cfg)

	if cfg.Token != "***" {
		t.Errorf("Expected token to be masked as '***', got '%s'", cfg.Token)
	}
	
	// Branch should still default to "main"
	if cfg.Branch != "main" {
		t.Errorf("Expected default branch 'main', got '%s'", cfg.Branch)
	}
}

func TestHandlePutGitDocsSettings_Success(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	payload := GitDocsConfig{
		RepoURL: "https://github.com/test/docs",
		Branch:  "develop",
		Token:   "newtoken456",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlePutGitDocsSettings(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify saved to database
	var repoURL, branch, token string
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_repo_url'").Scan(&repoURL)
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_branch'").Scan(&branch)
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_token'").Scan(&token)

	if repoURL != "https://github.com/test/docs" {
		t.Errorf("Repo URL not saved correctly: %s", repoURL)
	}
	if branch != "develop" {
		t.Errorf("Branch not saved correctly: %s", branch)
	}
	if token != "newtoken456" {
		t.Errorf("Token not saved correctly: %s", token)
	}
}

func TestHandlePutGitDocsSettings_InvalidJSON(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlePutGitDocsSettings(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandlePutGitDocsSettings_EmptyRepoURL(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	payload := GitDocsConfig{
		RepoURL: "",
		Branch:  "main",
		Token:   "token",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlePutGitDocsSettings(w, req)

	// Should still accept empty repo URL (allows disabling)
	if w.Code != 200 {
		t.Fatalf("Expected 200 for empty repo URL, got %d", w.Code)
	}
}

func TestHandlePutGitDocsSettings_PreserveTokenOnMasked(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Set initial token
	db.Exec("INSERT INTO app_settings (key, value) VALUES ('git_docs_token', 'original_secret')")

	// Update with masked token (should preserve original)
	payload := GitDocsConfig{
		RepoURL: "https://github.com/test/docs",
		Branch:  "main",
		Token:   "***",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlePutGitDocsSettings(w, req)

	// Check token was NOT updated (preserved)
	var token string
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_token'").Scan(&token)

	if token != "original_secret" {
		t.Errorf("Expected token to be preserved when masked '***' sent, got: %s", token)
	}
}

func TestHandlePushDocToGit_NoConfig(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Create a test document
	db.Exec(`INSERT INTO documents (id, title, content, file_path) VALUES (1, 'Test Doc', 'Content', 'test.md')`)

	req := httptest.NewRequest("POST", "/api/docs/1/push-git", nil)
	w := httptest.NewRecorder()

	handlePushDocToGit(w, req, "1")

	if w.Code != 400 {
		t.Errorf("Expected 400 when git not configured, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "git docs not configured" {
		t.Errorf("Expected 'git docs not configured' error, got: %v", response)
	}
}

func TestHandlePushDocToGit_DocumentNotFound(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Configure git
	db.Exec("INSERT INTO app_settings (key, value) VALUES ('git_docs_repo_url', 'https://github.com/test/docs')")

	req := httptest.NewRequest("POST", "/api/docs/999/push-git", nil)
	w := httptest.NewRecorder()

	handlePushDocToGit(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected 404 for non-existent document, got %d", w.Code)
	}
}

func TestGetGitDocsConfig_DefaultBranch(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO app_settings (key, value) VALUES ('git_docs_repo_url', 'https://github.com/test/repo')")
	// No branch set

	cfg := getGitDocsConfig()

	if cfg.Branch != "main" {
		t.Errorf("Expected default branch 'main', got %s", cfg.Branch)
	}
	if cfg.RepoURL != "https://github.com/test/repo" {
		t.Errorf("Expected repo URL, got %s", cfg.RepoURL)
	}
}

func TestGitDocsSettings_MultipleUpdates(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// First update
	payload1 := GitDocsConfig{
		RepoURL: "https://github.com/test/docs1",
		Branch:  "develop",
		Token:   "token1",
	}
	body1, _ := json.Marshal(payload1)
	req1 := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handlePutGitDocsSettings(w1, req1)

	// Second update (should overwrite)
	payload2 := GitDocsConfig{
		RepoURL: "https://github.com/test/docs2",
		Branch:  "staging",
		Token:   "token2",
	}
	body2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handlePutGitDocsSettings(w2, req2)

	// Verify latest values
	var repoURL, branch string
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_repo_url'").Scan(&repoURL)
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_branch'").Scan(&branch)

	if repoURL != "https://github.com/test/docs2" {
		t.Errorf("Expected updated repo URL, got %s", repoURL)
	}
	if branch != "staging" {
		t.Errorf("Expected updated branch, got %s", branch)
	}
}

func TestGitDocsSettings_WhitespaceHandling(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	payload := GitDocsConfig{
		RepoURL: "  https://github.com/test/docs  ",
		Branch:  "  main  ",
		Token:   "  token  ",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/settings/git-docs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlePutGitDocsSettings(w, req)

	// Values should be saved as-is (trimming should be done by validation if needed)
	var repoURL string
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_repo_url'").Scan(&repoURL)

	// Current implementation doesn't trim - this documents that behavior
	if repoURL != "  https://github.com/test/docs  " {
		t.Errorf("Expected whitespace preserved (or add trim validation), got: '%s'", repoURL)
	}
}
