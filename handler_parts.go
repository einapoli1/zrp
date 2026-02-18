package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// gitplm CSV format: first row is headers, IPN is derived from filename/category
// Category dirs contain CSV files with parts data

func loadPartsFromDir() (map[string][]Part, map[string][]string, error) {
	categories := make(map[string][]Part)
	schemas := make(map[string][]string)

	if partsDir == "" {
		return categories, schemas, nil
	}

	entries, err := os.ReadDir(partsDir)
	if err != nil {
		return categories, schemas, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			catDir := filepath.Join(partsDir, entry.Name())
			csvFiles, _ := filepath.Glob(filepath.Join(catDir, "*.csv"))
			catName := strings.ToLower(entry.Name())
			for _, csvFile := range csvFiles {
				parts, cols, err := readCSV(csvFile, catName)
				if err != nil {
					continue
				}
				categories[catName] = append(categories[catName], parts...)
				if len(cols) > len(schemas[catName]) {
					schemas[catName] = cols
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".csv") {
			catName := strings.TrimSuffix(entry.Name(), ".csv")
			catName = strings.ToLower(catName)
			parts, cols, err := readCSV(filepath.Join(partsDir, entry.Name()), catName)
			if err != nil {
				continue
			}
			categories[catName] = append(categories[catName], parts...)
			schemas[catName] = cols
		}
	}
	return categories, schemas, nil
}

func readCSV(path string, category string) ([]Part, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) < 2 {
		return nil, nil, fmt.Errorf("empty csv")
	}

	headers := records[0]
	var parts []Part
	for _, row := range records[1:] {
		fields := make(map[string]string)
		ipn := ""
		for i, h := range headers {
			if i < len(row) {
				fields[h] = row[i]
				hl := strings.ToLower(h)
				if hl == "ipn" || hl == "part_number" || hl == "pn" {
					ipn = row[i]
				}
			}
		}
		fields["_category"] = category
		if ipn == "" {
			// Try to derive from filename
			ipn = fields[headers[0]]
		}
		if ipn != "" {
			parts = append(parts, Part{IPN: ipn, Fields: fields})
		}
	}
	return parts, headers, nil
}

func handleListParts(w http.ResponseWriter, r *http.Request) {
	cats, _, _ := loadPartsFromDir()
	category := r.URL.Query().Get("category")
	q := strings.ToLower(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 { page = 1 }
	if limit < 1 { limit = 50 }

	var all []Part
	if category != "" {
		all = cats[category]
	} else {
		for _, p := range cats {
			all = append(all, p...)
		}
	}

	// Search filter
	if q != "" {
		var filtered []Part
		for _, p := range all {
			if strings.Contains(strings.ToLower(p.IPN), q) {
				filtered = append(filtered, p)
				continue
			}
			for _, v := range p.Fields {
				if strings.Contains(strings.ToLower(v), q) {
					filtered = append(filtered, p)
					break
				}
			}
		}
		all = filtered
	}

	sort.Slice(all, func(i, j int) bool { return all[i].IPN < all[j].IPN })
	total := len(all)
	start := (page - 1) * limit
	if start > total { start = total }
	end := start + limit
	if end > total { end = total }

	if all == nil { all = []Part{} }
	jsonRespMeta(w, all[start:end], total, page, limit)
}

func handleGetPart(w http.ResponseWriter, r *http.Request, ipn string) {
	cats, _, _ := loadPartsFromDir()
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				jsonResp(w, p)
				return
			}
		}
	}
	jsonErr(w, "part not found", 404)
}

func handleCreatePart(w http.ResponseWriter, r *http.Request) {
	// For now, parts are read-only from CSVs
	jsonErr(w, "creating parts via API not yet supported — edit CSVs directly", 501)
}

func handleUpdatePart(w http.ResponseWriter, r *http.Request, ipn string) {
	jsonErr(w, "updating parts via API not yet supported — edit CSVs directly", 501)
}

func handleDeletePart(w http.ResponseWriter, r *http.Request, ipn string) {
	jsonErr(w, "deleting parts via API not yet supported — edit CSVs directly", 501)
}

func handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, schemas, _ := loadPartsFromDir()
	var result []Category
	for name, parts := range cats {
		cols := schemas[name]
		if cols == nil { cols = []string{} }
		result = append(result, Category{ID: name, Name: name, Count: len(parts), Columns: cols})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	jsonResp(w, result)
}

func handleAddColumn(w http.ResponseWriter, r *http.Request, catID string) {
	var body struct{ Name string `json:"name"` }
	if err := decodeBody(r, &body); err != nil || body.Name == "" {
		jsonErr(w, "name required", 400)
		return
	}
	// Would need to modify CSV files - stub for now
	jsonResp(w, map[string]string{"status": "column add not yet implemented for CSV backend"})
}

func handleDeleteColumn(w http.ResponseWriter, r *http.Request, catID, colName string) {
	jsonResp(w, map[string]string{"status": "column delete not yet implemented for CSV backend"})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	d := DashboardData{}
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE status NOT IN ('implemented','rejected')").Scan(&d.OpenECOs)
	db.QueryRow("SELECT COUNT(*) FROM inventory WHERE qty_on_hand <= reorder_point AND reorder_point > 0").Scan(&d.LowStock)
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE status NOT IN ('received','cancelled')").Scan(&d.OpenPOs)
	db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE status IN ('open','in_progress')").Scan(&d.ActiveWOs)
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE status NOT IN ('resolved','closed')").Scan(&d.OpenNCRs)
	db.QueryRow("SELECT COUNT(*) FROM rmas WHERE status NOT IN ('closed')").Scan(&d.OpenRMAs)
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&d.TotalDevices)

	// Count parts from CSV
	cats, _, _ := loadPartsFromDir()
	for _, p := range cats { d.TotalParts += len(p) }

	json.NewEncoder(w).Encode(d)
}
