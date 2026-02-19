package main

import (
	"net/http"
	"strings"
)

// GitPLMConfig holds the gitplm-ui integration settings
type GitPLMConfig struct {
	BaseURL string `json:"base_url"`
}

// GitPLMURLResponse is the response for the gitplm URL endpoint
type GitPLMURLResponse struct {
	URL       string `json:"url"`
	Configured bool  `json:"configured"`
}

func handleGetGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	var baseURL string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = 'gitplm_base_url'").Scan(&baseURL)
	if err != nil {
		jsonResp(w, GitPLMConfig{BaseURL: ""})
		return
	}
	jsonResp(w, GitPLMConfig{BaseURL: baseURL})
}

func handleUpdateGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	var cfg GitPLMConfig
	if err := decodeBody(r, &cfg); err != nil {
		jsonErr(w, "invalid request body", 400)
		return
	}

	// Trim trailing slash
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	_, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('gitplm_base_url', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, cfg.BaseURL)
	if err != nil {
		jsonErr(w, "failed to save setting", 500)
		return
	}
	jsonResp(w, cfg)
}

func handleGetGitPLMURL(w http.ResponseWriter, r *http.Request, ipn string) {
	var baseURL string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = 'gitplm_base_url'").Scan(&baseURL)
	if err != nil || baseURL == "" {
		jsonResp(w, GitPLMURLResponse{URL: "", Configured: false})
		return
	}
	url := baseURL + "/parts/" + ipn
	jsonResp(w, GitPLMURLResponse{URL: url, Configured: true})
}
