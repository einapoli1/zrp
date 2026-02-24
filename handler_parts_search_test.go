package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPartSearch tests the part search API endpoint
func TestPartSearch(t *testing.T) {
	// Create temporary parts directory with test data
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test category with parts
	createTestPartCSV(t, tmpDir, "resistors.csv", [][]string{
		{"IPN", "description", "manufacturer", "value"},
		{"RES-001", "10K Resistor", "Yageo", "10K"},
		{"RES-002", "1K Resistor", "Vishay", "1K"},
	})

	createTestPartCSV(t, tmpDir, "capacitors.csv", [][]string{
		{"IPN", "description", "manufacturer", "mpn"},
		{"CAP-100", "100nF Ceramic Cap", "Murata", "GRM188R71H104KA93"},
		{"CAP-101", "22uF Electrolytic", "Nichicon", "UWT1H220MCL1GS"},
	})

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectedIPNs   []string
		checkSubstring bool // if true, check if IPN contains substring
	}{
		{
			name:          "search by exact IPN",
			query:         "RES-001",
			expectedCount: 1,
			expectedIPNs:  []string{"RES-001"},
		},
		{
			name:           "search by partial IPN",
			query:          "RES",
			expectedCount:  2,
			expectedIPNs:   []string{"RES-001", "RES-002"},
			checkSubstring: true,
		},
		{
			name:           "search by case-insensitive IPN",
			query:          "res-001",
			expectedCount:  1,
			expectedIPNs:   []string{"RES-001"},
			checkSubstring: true,
		},
		{
			name:           "search by manufacturer",
			query:          "Yageo",
			expectedCount:  1,
			expectedIPNs:   []string{"RES-001"},
			checkSubstring: true,
		},
		{
			name:           "search by MPN",
			query:          "GRM188R71H104KA93",
			expectedCount:  1,
			expectedIPNs:   []string{"CAP-100"},
			checkSubstring: true,
		},
		{
			name:           "search by description keyword",
			query:          "Resistor",
			expectedCount:  2,
			expectedIPNs:   []string{"RES-001", "RES-002"},
			checkSubstring: true,
		},
		{
			name:          "search with empty query returns all",
			query:         "",
			expectedCount: 4,
		},
		{
			name:          "search for non-existent part",
			query:         "DOESNOTEXIST-999",
			expectedCount: 0,
			expectedIPNs:  []string{},
		},
		{
			name:           "search by value field",
			query:          "10K",
			expectedCount:  1,
			expectedIPNs:   []string{"RES-001"},
			checkSubstring: true,
		},
		{
			name:           "search by partial manufacturer (case-insensitive)",
			query:          "murata",
			expectedCount:  1,
			expectedIPNs:   []string{"CAP-100"},
			checkSubstring: true,
		},
		{
			name:           "search with special characters in query",
			query:          "CAP-",
			expectedCount:  2,
			checkSubstring: true,
		},
		{
			name:           "search across multiple categories",
			query:          "Cap",
			expectedCount:  2,
			checkSubstring: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/parts?q="+tt.query, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleListParts)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response struct {
				Data []Part `json:"data"`
				Meta struct {
					Total int `json:"total"`
				} `json:"meta"`
			}

			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectedCount > 0 && len(response.Data) != tt.expectedCount {
				t.Errorf("expected %d parts, got %d", tt.expectedCount, len(response.Data))
			}

			// Verify specific IPNs are present if specified
			if len(tt.expectedIPNs) > 0 {
				foundIPNs := make(map[string]bool)
				for _, part := range response.Data {
					foundIPNs[part.IPN] = true
				}

				for _, expectedIPN := range tt.expectedIPNs {
					if !foundIPNs[expectedIPN] {
						t.Errorf("expected IPN %s not found in results", expectedIPN)
					}
				}
			}
		})
	}
}

// TestPartSearchByCategory tests filtering parts by category
func TestPartSearchByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	createTestPartCSV(t, tmpDir, "resistors.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
		{"RES-002", "1K Resistor"},
	})

	createTestPartCSV(t, tmpDir, "capacitors.csv", [][]string{
		{"IPN", "description"},
		{"CAP-001", "10uF Capacitor"},
		{"CAP-002", "22uF Capacitor"},
	})

	tests := []struct {
		name          string
		category      string
		expectedCount int
	}{
		{
			name:          "filter by resistors category",
			category:      "resistors",
			expectedCount: 2,
		},
		{
			name:          "filter by capacitors category",
			category:      "capacitors",
			expectedCount: 2,
		},
		{
			name:          "no category filter returns all",
			category:      "",
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/parts"
			if tt.category != "" {
				url += "?category=" + tt.category
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleListParts)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response struct {
				Data []Part `json:"data"`
			}

			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(response.Data) != tt.expectedCount {
				t.Errorf("expected %d parts, got %d", tt.expectedCount, len(response.Data))
			}
		})
	}
}

// TestPartSearchPagination tests pagination of part search results
func TestPartSearchPagination(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create 25 test parts
	parts := [][]string{{"IPN", "description"}}
	for i := 1; i <= 25; i++ {
		parts = append(parts, []string{
			"RES-" + padInt(i, 3),
			"Test Resistor " + padInt(i, 3),
		})
	}
	createTestPartCSV(t, tmpDir, "resistors.csv", parts)

	tests := []struct {
		name          string
		page          string
		limit         string
		expectedCount int
	}{
		{
			name:          "first page with default limit (50)",
			page:          "1",
			limit:         "",
			expectedCount: 25,
		},
		{
			name:          "page 1 with limit 10",
			page:          "1",
			limit:         "10",
			expectedCount: 10,
		},
		{
			name:          "page 2 with limit 10",
			page:          "2",
			limit:         "10",
			expectedCount: 10,
		},
		{
			name:          "page 3 with limit 10",
			page:          "3",
			limit:         "10",
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/parts?"
			if tt.page != "" {
				url += "page=" + tt.page + "&"
			}
			if tt.limit != "" {
				url += "limit=" + tt.limit
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleListParts)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response struct {
				Data []Part `json:"data"`
				Meta struct {
					Total int `json:"total"`
					Page  int `json:"page"`
					Limit int `json:"limit"`
				} `json:"meta"`
			}

			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(response.Data) != tt.expectedCount {
				t.Errorf("expected %d parts on page, got %d", tt.expectedCount, len(response.Data))
			}

			if response.Meta.Total != 25 {
				t.Errorf("expected total of 25 parts, got %d", response.Meta.Total)
			}
		})
	}
}

// Helper function to create test part CSV files
func createTestPartCSV(t *testing.T, dir, filename string, rows [][]string) {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	content := ""
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				content += ","
			}
			content += cell
		}
		content += "\n"
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}
}

// Helper to pad integers for sorting
func padInt(i, width int) string {
	s := ""
	for j := 0; j < width; j++ {
		s += "0"
	}
	s += string(rune('0' + i))
	return s[len(s)-width:]
}
