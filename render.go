package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var baseTemplateFiles = []string{
	"templates/layout.html",
	"templates/partials/sidebar.html",
	"templates/partials/header.html",
	"templates/partials/toast.html",
	"templates/partials/pagination.html",
	"templates/partials/modal.html",
	"templates/partials/empty-state.html",
}

var templateFuncs template.FuncMap

type PageData struct {
	Title        string
	ActiveNav    string
	User         *UserFull
	Dashboard    DashboardData
	Parts        []Part
	Part         Part
	HasBOM       bool
	Cost         *PartCostInfo
	ECOs         []ECO
	ECO          ECO
	AffectedIPNs []string
	Categories   []Category
	Query        string
	Category     string
	Status       string
	Total        int
	Pagination   PaginationData
	Activities   []AuditEntry
	// Quotes module
	Quotes       []Quote
	Quote        Quote
	// Calendar module
	CalendarData CalendarPageData
	Year         int
	Month        int
	MonthName    string
	// NCR module  
	NCRs         []NCR
	NCR          NCR
	// Inventory
	Inventory     []InventoryItem
	InventoryItem InventoryItem
	Transactions  []InventoryTransaction
	LowStock      bool
	// Procurement
	POs       []PurchaseOrder
	PO        PurchaseOrder
	Vendors   []Vendor
	Vendor    Vendor
	WorkOrders []WorkOrder
}

type PartCostInfo struct {
	LastUnitPrice float64
	BOMCost       float64
}

type PaginationData struct {
	Page        int
	TotalPages  int
	Total       int
	Start       int
	End         int
	BaseURL     string
	ExtraParams template.HTML
	Target      string
	Pages       []int
}

type EmptyStateData struct {
	Message     string
	ActionURL   string
	ActionLabel string
}

type CalendarPageData struct {
	Weeks [][]CalendarDay
}

type CalendarDay struct {
	Day     int
	InMonth bool
	Today   bool
	Events  []CalendarEvent
}

func initTemplates() {
	templateFuncs = template.FuncMap{
		"badge": func(status string) template.HTML {
			return template.HTML(fmt.Sprintf(`<span class="badge badge-%s">%s</span>`, status, status))
		},
		"formatDate": func(s string) string {
			t, err := time.Parse("2006-01-02 15:04:05", s)
			if err != nil {
				t2, err2 := time.Parse(time.RFC3339, s)
				if err2 != nil { return s }
				t = t2
			}
			return t.Format("Jan 2, 2006")
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n { return s }
			return s[:n] + "\xe2\x80\xa6"
		},
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"deref": func(s *string) string {
			if s == nil { return "" }
			return *s
		},
		"fieldVal": func(fields map[string]string, keys ...string) string {
			for _, k := range keys {
				if v, ok := fields[k]; ok && v != "" { return v }
				if v, ok := fields[strings.ToLower(k)]; ok && v != "" { return v }
			}
			return ""
		},
		"minus": func(a, b int) int { return a - b },
		"plus":  func(a, b int) int { return a + b },
	}
	log.Println("Template functions initialized")
}

func pageTemplate(pageFiles ...string) *template.Template {
	files := append([]string{}, baseTemplateFiles...)
	files = append(files, pageFiles...)
	t, err := template.New("").Funcs(templateFuncs).ParseFiles(files...)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		return nil
	}
	return t
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func render(w http.ResponseWriter, pageFiles []string, templateName string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := pageTemplate(pageFiles...)
	if t == nil { http.Error(w, "Template error", 500); return }
	if err := t.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template error (%s): %v", templateName, err)
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func renderPartial(w http.ResponseWriter, pageFiles []string, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := pageTemplate(pageFiles...)
	if t == nil { http.Error(w, "Template error", 500); return }
	if err := t.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Partial template error (%s): %v", templateName, err)
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func makePagination(page, total, limit int, baseURL, target string, extraParams string) PaginationData {
	totalPages := (total + limit - 1) / limit
	if totalPages < 1 { totalPages = 1 }
	start := (page-1)*limit + 1
	end := start + limit - 1
	if end > total { end = total }
	if start > total { start = total }
	var pages []int
	for i := 1; i <= totalPages && i <= 10; i++ { pages = append(pages, i) }
	return PaginationData{
		Page: page, TotalPages: totalPages, Total: total,
		Start: start, End: end, BaseURL: baseURL, Target: target,
		ExtraParams: template.HTML(extraParams), Pages: pages,
	}
}
