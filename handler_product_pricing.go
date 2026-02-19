package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type ProductPricing struct {
	ID            int     `json:"id"`
	ProductIPN    string  `json:"product_ipn"`
	PricingTier   string  `json:"pricing_tier"`
	MinQty        int     `json:"min_qty"`
	MaxQty        int     `json:"max_qty"`
	UnitPrice     float64 `json:"unit_price"`
	Currency      string  `json:"currency"`
	EffectiveDate string  `json:"effective_date"`
	ExpiryDate    string  `json:"expiry_date,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type CostAnalysis struct {
	ID             int     `json:"id"`
	ProductIPN     string  `json:"product_ipn"`
	BOMCost        float64 `json:"bom_cost"`
	LaborCost      float64 `json:"labor_cost"`
	OverheadCost   float64 `json:"overhead_cost"`
	TotalCost      float64 `json:"total_cost"`
	MarginPct      float64 `json:"margin_pct"`
	LastCalculated string  `json:"last_calculated"`
	CreatedAt      string  `json:"created_at"`
}

type CostAnalysisWithPricing struct {
	CostAnalysis
	SellingPrice float64 `json:"selling_price"`
}

type BulkPriceUpdate struct {
	IDs             []int   `json:"ids"`
	AdjustmentType  string  `json:"adjustment_type"`  // "percentage" or "absolute"
	AdjustmentValue float64 `json:"adjustment_value"`
}

func handleListProductPricing(w http.ResponseWriter, r *http.Request) {
	ipnFilter := r.URL.Query().Get("product_ipn")
	tierFilter := r.URL.Query().Get("pricing_tier")

	query := `SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE 1=1`
	var args []interface{}
	if ipnFilter != "" {
		query += " AND product_ipn = ?"
		args = append(args, ipnFilter)
	}
	if tierFilter != "" {
		query += " AND pricing_tier = ?"
		args = append(args, tierFilter)
	}
	query += " ORDER BY product_ipn, pricing_tier, min_qty"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []ProductPricing
	for rows.Next() {
		var p ProductPricing
		rows.Scan(&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
			&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
			&p.CreatedAt, &p.UpdatedAt)
		items = append(items, p)
	}
	if items == nil {
		items = []ProductPricing{}
	}
	jsonResp(w, items)
}

func handleGetProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	var p ProductPricing
	err := db.QueryRow(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE id = ?`, id).Scan(
		&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
		&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		jsonErr(w, "pricing not found", 404)
		return
	}
	jsonResp(w, p)
}

func handleCreateProductPricing(w http.ResponseWriter, r *http.Request) {
	var p ProductPricing
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if p.ProductIPN == "" {
		jsonErr(w, "product_ipn required", 400)
		return
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.PricingTier == "" {
		p.PricingTier = "standard"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency, effective_date, expiry_date, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ProductIPN, p.PricingTier, p.MinQty, p.MaxQty, p.UnitPrice, p.Currency,
		p.EffectiveDate, p.ExpiryDate, p.Notes, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	p.ID = int(id)
	p.CreatedAt = now
	p.UpdatedAt = now
	jsonResp(w, p)
}

func handleUpdateProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	// Check exists
	var existing ProductPricing
	err := db.QueryRow(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE id = ?`, id).Scan(
		&existing.ID, &existing.ProductIPN, &existing.PricingTier, &existing.MinQty, &existing.MaxQty,
		&existing.UnitPrice, &existing.Currency, &existing.EffectiveDate, &existing.ExpiryDate, &existing.Notes,
		&existing.CreatedAt, &existing.UpdatedAt)
	if err != nil {
		jsonErr(w, "pricing not found", 404)
		return
	}

	var update map[string]interface{}
	if err := decodeBody(r, &update); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	if v, ok := update["product_ipn"].(string); ok {
		existing.ProductIPN = v
	}
	if v, ok := update["pricing_tier"].(string); ok {
		existing.PricingTier = v
	}
	if v, ok := update["min_qty"].(float64); ok {
		existing.MinQty = int(v)
	}
	if v, ok := update["max_qty"].(float64); ok {
		existing.MaxQty = int(v)
	}
	if v, ok := update["unit_price"].(float64); ok {
		existing.UnitPrice = v
	}
	if v, ok := update["currency"].(string); ok {
		existing.Currency = v
	}
	if v, ok := update["effective_date"].(string); ok {
		existing.EffectiveDate = v
	}
	if v, ok := update["expiry_date"].(string); ok {
		existing.ExpiryDate = v
	}
	if v, ok := update["notes"].(string); ok {
		existing.Notes = v
	}

	now := time.Now().UTC().Format(time.RFC3339)
	existing.UpdatedAt = now
	_, err = db.Exec(`UPDATE product_pricing SET product_ipn=?, pricing_tier=?, min_qty=?, max_qty=?, unit_price=?, currency=?, effective_date=?, expiry_date=?, notes=?, updated_at=? WHERE id=?`,
		existing.ProductIPN, existing.PricingTier, existing.MinQty, existing.MaxQty,
		existing.UnitPrice, existing.Currency, existing.EffectiveDate, existing.ExpiryDate,
		existing.Notes, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, existing)
}

func handleDeleteProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	res, err := db.Exec("DELETE FROM product_pricing WHERE id = ?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "not found", 404)
		return
	}
	jsonResp(w, map[string]string{"status": "deleted"})
}

func handleListCostAnalysis(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT ca.id, ca.product_ipn, ca.bom_cost, ca.labor_cost, ca.overhead_cost,
		ca.total_cost, ca.margin_pct, ca.last_calculated, ca.created_at,
		COALESCE((SELECT pp.unit_price FROM product_pricing pp WHERE pp.product_ipn = ca.product_ipn AND pp.pricing_tier = 'standard' ORDER BY pp.effective_date DESC LIMIT 1), 0) as selling_price
		FROM cost_analysis ca ORDER BY ca.product_ipn`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []CostAnalysisWithPricing
	for rows.Next() {
		var c CostAnalysisWithPricing
		rows.Scan(&c.ID, &c.ProductIPN, &c.BOMCost, &c.LaborCost, &c.OverheadCost,
			&c.TotalCost, &c.MarginPct, &c.LastCalculated, &c.CreatedAt, &c.SellingPrice)
		items = append(items, c)
	}
	if items == nil {
		items = []CostAnalysisWithPricing{}
	}
	jsonResp(w, items)
}

func handleCreateCostAnalysis(w http.ResponseWriter, r *http.Request) {
	var c CostAnalysis
	if err := decodeBody(r, &c); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if c.ProductIPN == "" {
		jsonErr(w, "product_ipn required", 400)
		return
	}
	c.TotalCost = c.BOMCost + c.LaborCost + c.OverheadCost

	// Calculate margin from standard pricing
	var sellingPrice float64
	db.QueryRow(`SELECT unit_price FROM product_pricing WHERE product_ipn = ? AND pricing_tier = 'standard' ORDER BY effective_date DESC LIMIT 1`, c.ProductIPN).Scan(&sellingPrice)
	if sellingPrice > 0 {
		c.MarginPct = ((sellingPrice - c.TotalCost) / sellingPrice) * 100
	}

	now := time.Now().UTC().Format(time.RFC3339)
	c.LastCalculated = now
	c.CreatedAt = now

	// Upsert
	res, err := db.Exec(`INSERT INTO cost_analysis (product_ipn, bom_cost, labor_cost, overhead_cost, total_cost, margin_pct, last_calculated, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_ipn) DO UPDATE SET bom_cost=excluded.bom_cost, labor_cost=excluded.labor_cost,
		overhead_cost=excluded.overhead_cost, total_cost=excluded.total_cost, margin_pct=excluded.margin_pct,
		last_calculated=excluded.last_calculated`,
		c.ProductIPN, c.BOMCost, c.LaborCost, c.OverheadCost, c.TotalCost, c.MarginPct, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	c.ID = int(id)
	jsonResp(w, c)
}

func handleProductPricingHistory(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := db.Query(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE product_ipn = ? ORDER BY created_at DESC`, ipn)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []ProductPricing
	for rows.Next() {
		var p ProductPricing
		rows.Scan(&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
			&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
			&p.CreatedAt, &p.UpdatedAt)
		items = append(items, p)
	}
	if items == nil {
		items = []ProductPricing{}
	}
	jsonResp(w, items)
}

func handleBulkUpdateProductPricing(w http.ResponseWriter, r *http.Request) {
	var req BulkPriceUpdate
	if err := decodeBody(r, &req); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(req.IDs) == 0 {
		jsonErr(w, "ids required", 400)
		return
	}

	updated := 0
	for _, id := range req.IDs {
		var currentPrice float64
		err := db.QueryRow("SELECT unit_price FROM product_pricing WHERE id = ?", id).Scan(&currentPrice)
		if err != nil {
			continue
		}
		var newPrice float64
		switch req.AdjustmentType {
		case "percentage":
			newPrice = currentPrice * (1 + req.AdjustmentValue/100)
		case "absolute":
			newPrice = currentPrice + req.AdjustmentValue
		default:
			jsonErr(w, "adjustment_type must be 'percentage' or 'absolute'", 400)
			return
		}
		// Round to 2 decimal places
		newPrice = float64(int(newPrice*100+0.5)) / 100
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = db.Exec("UPDATE product_pricing SET unit_price = ?, updated_at = ? WHERE id = ?", newPrice, now, id)
		if err == nil {
			updated++
		}
	}
	jsonResp(w, map[string]interface{}{
		"updated": updated,
		"total":   len(req.IDs),
	})
}

// Helper used by routes
func parsePricingID(parts []string, idx int) string {
	if idx < len(parts) {
		return parts[idx]
	}
	return ""
}

func pricingIDStr(id int) string {
	return strconv.Itoa(id)
}

var _ = fmt.Sprintf // keep import
