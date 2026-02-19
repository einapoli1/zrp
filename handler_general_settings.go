package main

import (
	"encoding/json"
	"net/http"
)

type GeneralSettings struct {
	AppName        string `json:"app_name"`
	CompanyName    string `json:"company_name"`
	CompanyAddress string `json:"company_address"`
	Currency       string `json:"currency"`
	DateFormat     string `json:"date_format"`
}

var generalSettingsKeys = []string{
	"app_name", "company_name", "company_address", "currency", "date_format",
}

var generalSettingsDefaults = map[string]string{
	"app_name":        "ZRP",
	"company_name":    "",
	"company_address": "",
	"currency":        "USD",
	"date_format":     "YYYY-MM-DD",
}

func handleGetGeneralSettings(w http.ResponseWriter, r *http.Request) {
	s := GeneralSettings{
		AppName:    generalSettingsDefaults["app_name"],
		Currency:   generalSettingsDefaults["currency"],
		DateFormat: generalSettingsDefaults["date_format"],
	}

	for _, key := range generalSettingsKeys {
		var val string
		err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_"+key).Scan(&val)
		if err != nil {
			continue
		}
		switch key {
		case "app_name":
			s.AppName = val
		case "company_name":
			s.CompanyName = val
		case "company_address":
			s.CompanyAddress = val
		case "currency":
			s.Currency = val
		case "date_format":
			s.DateFormat = val
		}
	}

	jsonResp(w, s)
}

func handlePutGeneralSettings(w http.ResponseWriter, r *http.Request) {
	var s GeneralSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	vals := map[string]string{
		"app_name":        s.AppName,
		"company_name":    s.CompanyName,
		"company_address": s.CompanyAddress,
		"currency":        s.Currency,
		"date_format":     s.DateFormat,
	}

	for key, val := range vals {
		_, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, "general_"+key, val)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonResp(w, s)
}
