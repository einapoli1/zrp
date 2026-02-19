package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitDocsConfig struct {
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch"`
	Token   string `json:"token"`
}

func getGitDocsConfig() GitDocsConfig {
	var cfg GitDocsConfig
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_repo_url'").Scan(&cfg.RepoURL)
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_branch'").Scan(&cfg.Branch)
	db.QueryRow("SELECT value FROM app_settings WHERE key='git_docs_token'").Scan(&cfg.Token)
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	return cfg
}

func handleGetGitDocsSettings(w http.ResponseWriter, r *http.Request) {
	cfg := getGitDocsConfig()
	// Don't expose full token
	if cfg.Token != "" {
		cfg.Token = "***"
	}
	jsonResp(w, cfg)
}

func handlePutGitDocsSettings(w http.ResponseWriter, r *http.Request) {
	var cfg GitDocsConfig
	if err := decodeBody(r, &cfg); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	upsertSetting := func(key, value string) {
		db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=?", key, value, value)
	}
	upsertSetting("git_docs_repo_url", cfg.RepoURL)
	upsertSetting("git_docs_branch", cfg.Branch)
	if cfg.Token != "***" && cfg.Token != "" {
		upsertSetting("git_docs_token", cfg.Token)
	}
	logAudit(db, getUsername(r), "updated", "settings", "git-docs", "Updated git docs settings")
	jsonResp(w, map[string]string{"status": "ok"})
}

func gitDocsRepoPath() string {
	return filepath.Join("docs-repo")
}

func ensureGitRepo(cfg GitDocsConfig) error {
	repoPath := gitDocsRepoPath()
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repo exists, pull
		cmd := exec.Command("git", "-C", repoPath, "pull", "origin", cfg.Branch)
		cmd.Env = append(os.Environ(), gitAuthEnv(cfg)...)
		cmd.Run() // best effort
		return nil
	}
	// Clone
	repoURL := injectTokenInURL(cfg.RepoURL, cfg.Token)
	cmd := exec.Command("git", "clone", "--branch", cfg.Branch, repoURL, repoPath)
	cmd.Env = append(os.Environ(), gitAuthEnv(cfg)...)
	return cmd.Run()
}

func gitAuthEnv(cfg GitDocsConfig) []string {
	if cfg.Token != "" {
		return []string{"GIT_ASKPASS=echo", "GIT_TERMINAL_PROMPT=0"}
	}
	return nil
}

func injectTokenInURL(repoURL, token string) string {
	if token == "" {
		return repoURL
	}
	// For https URLs, inject token as password
	if strings.HasPrefix(repoURL, "https://") {
		return strings.Replace(repoURL, "https://", "https://oauth2:"+token+"@", 1)
	}
	return repoURL
}

func docFilePath(category, docID, title string) string {
	cat := category
	if cat == "" {
		cat = "general"
	}
	// Sanitize title for filename
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, title)
	return filepath.Join(cat, docID+"-"+safe+".md")
}

func handlePushDocToGit(w http.ResponseWriter, r *http.Request, docID string) {
	cfg := getGitDocsConfig()
	if cfg.RepoURL == "" {
		jsonErr(w, "git docs repo not configured", 400)
		return
	}

	var d Document
	err := db.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", docID).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	if err := ensureGitRepo(cfg); err != nil {
		jsonErr(w, "failed to setup git repo: "+err.Error(), 500)
		return
	}

	repoPath := gitDocsRepoPath()
	filePath := docFilePath(d.Category, d.ID, d.Title)
	fullPath := filepath.Join(repoPath, filePath)

	// Create directory
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	// Write file
	if err := os.WriteFile(fullPath, []byte(d.Content), 0644); err != nil {
		jsonErr(w, "failed to write file: "+err.Error(), 500)
		return
	}

	// Git add, commit, push
	commitMsg := fmt.Sprintf("Update %s to Rev %s", d.Title, d.Revision)
	cmds := [][]string{
		{"git", "-C", repoPath, "add", filePath},
		{"git", "-C", repoPath, "commit", "-m", commitMsg, "--allow-empty"},
		{"git", "-C", repoPath, "push", "origin", cfg.Branch},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), gitAuthEnv(cfg)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			jsonErr(w, fmt.Sprintf("git error: %s: %s", err, string(out)), 500)
			return
		}
	}

	logAudit(db, getUsername(r), "pushed", "document", docID, "Pushed "+docID+" to git")
	jsonResp(w, map[string]string{"status": "pushed", "file": filePath})
}

func handleSyncDocFromGit(w http.ResponseWriter, r *http.Request, docID string) {
	cfg := getGitDocsConfig()
	if cfg.RepoURL == "" {
		jsonErr(w, "git docs repo not configured", 400)
		return
	}

	var d Document
	err := db.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", docID).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	if err := ensureGitRepo(cfg); err != nil {
		jsonErr(w, "failed to setup git repo: "+err.Error(), 500)
		return
	}

	repoPath := gitDocsRepoPath()
	filePath := docFilePath(d.Category, d.ID, d.Title)
	fullPath := filepath.Join(repoPath, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		jsonErr(w, "file not found in git repo", 404)
		return
	}

	username := getUsername(r)
	snapshotDocumentVersion(docID, "Before git sync", username, nil)

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE documents SET content=?, updated_at=? WHERE id=?", string(content), now, docID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "synced", "document", docID, "Synced "+docID+" from git")
	handleGetDoc(w, r, docID)
}

func handleCreateECOPR(w http.ResponseWriter, r *http.Request, ecoID string) {
	cfg := getGitDocsConfig()
	if cfg.RepoURL == "" {
		jsonErr(w, "git docs repo not configured", 400)
		return
	}

	var eco ECO
	err := db.QueryRow("SELECT id,title,COALESCE(description,''),status,COALESCE(affected_ipns,'') FROM ecos WHERE id=?", ecoID).
		Scan(&eco.ID, &eco.Title, &eco.Description, &eco.Status, &eco.AffectedIPNs)
	if err != nil {
		jsonErr(w, "ECO not found", 404)
		return
	}

	if err := ensureGitRepo(cfg); err != nil {
		jsonErr(w, "failed to setup git repo: "+err.Error(), 500)
		return
	}

	repoPath := gitDocsRepoPath()
	branchName := "eco/" + ecoID

	// Create branch
	exec.Command("git", "-C", repoPath, "checkout", cfg.Branch).Run()
	exec.Command("git", "-C", repoPath, "pull", "origin", cfg.Branch).Run()
	exec.Command("git", "-C", repoPath, "checkout", "-b", branchName).Run()

	// Find and commit all documents linked via affected IPNs
	var ipns []string
	json.Unmarshal([]byte(eco.AffectedIPNs), &ipns)

	var affectedDocs []string
	for _, ipn := range ipns {
		rows, _ := db.Query("SELECT id,title,COALESCE(category,''),revision,COALESCE(content,'') FROM documents WHERE ipn=?", ipn)
		if rows == nil {
			continue
		}
		for rows.Next() {
			var id, title, cat, rev, content string
			rows.Scan(&id, &title, &cat, &rev, &content)
			filePath := docFilePath(cat, id, title)
			fullPath := filepath.Join(repoPath, filePath)
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(content), 0644)
			exec.Command("git", "-C", repoPath, "add", filePath).Run()
			affectedDocs = append(affectedDocs, fmt.Sprintf("- %s (%s) Rev %s", title, id, rev))
		}
		rows.Close()
	}

	if len(affectedDocs) == 0 {
		// Cleanup branch
		exec.Command("git", "-C", repoPath, "checkout", cfg.Branch).Run()
		exec.Command("git", "-C", repoPath, "branch", "-D", branchName).Run()
		jsonErr(w, "no documents found for ECO's affected IPNs", 400)
		return
	}

	commitMsg := fmt.Sprintf("ECO %s: %s\n\n%s\n\nAffected documents:\n%s",
		ecoID, eco.Title, eco.Description, strings.Join(affectedDocs, "\n"))
	exec.Command("git", "-C", repoPath, "commit", "-m", commitMsg, "--allow-empty").Run()

	cmd := exec.Command("git", "-C", repoPath, "push", "origin", branchName)
	cmd.Env = append(os.Environ(), gitAuthEnv(cfg)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		jsonErr(w, fmt.Sprintf("push failed: %s: %s", err, string(out)), 500)
		exec.Command("git", "-C", repoPath, "checkout", cfg.Branch).Run()
		return
	}

	// Switch back to main
	exec.Command("git", "-C", repoPath, "checkout", cfg.Branch).Run()

	logAudit(db, getUsername(r), "created-pr", "eco", ecoID, "Created PR branch "+branchName)
	jsonResp(w, map[string]string{
		"status": "created",
		"branch": branchName,
	})
}
