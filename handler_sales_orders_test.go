package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSalesOrderCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create sales order
	body := `{"customer":"Acme Corp","notes":"Test order","lines":[{"ipn":"IPN-001","description":"Widget","qty":10,"unit_price":25.50}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created SalesOrder
	json.Unmarshal(w.Body.Bytes(), &created)
	if !strings.HasPrefix(created.ID, "SO-") {
		t.Errorf("expected SO- prefix, got %s", created.ID)
	}
	if created.Status != "draft" {
		t.Errorf("expected draft, got %s", created.Status)
	}
	orderID := created.ID

	// List
	req = authedRequest("GET", "/api/v1/sales-orders", "", cookie)
	w = httptest.NewRecorder()
	handleListSalesOrders(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	var orders []SalesOrder
	json.Unmarshal(w.Body.Bytes(), &orders)
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got %d", len(orders))
	}

	// Get
	req = authedRequest("GET", "/api/v1/sales-orders/"+orderID, "", cookie)
	w = httptest.NewRecorder()
	handleGetSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}
	var fetched SalesOrder
	json.Unmarshal(w.Body.Bytes(), &fetched)
	if len(fetched.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(fetched.Lines))
	}
	if fetched.Lines[0].UnitPrice != 25.50 {
		t.Errorf("expected unit_price 25.50, got %f", fetched.Lines[0].UnitPrice)
	}

	// Update
	body = `{"customer":"Acme Corp Updated","status":"draft","notes":"updated"}`
	req = authedRequest("PUT", "/api/v1/sales-orders/"+orderID, body, cookie)
	w = httptest.NewRecorder()
	handleUpdateSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
}

func TestSalesOrderStatusFilter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create two orders with different customers
	for _, cust := range []string{"Alpha Inc", "Beta LLC"} {
		body := `{"customer":"` + cust + `","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
		req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
		w := httptest.NewRecorder()
		handleCreateSalesOrder(w, req)
		if w.Code != 200 {
			t.Fatalf("create: %d", w.Code)
		}
	}

	// Filter by status
	req := authedRequest("GET", "/api/v1/sales-orders?status=draft", "", cookie)
	w := httptest.NewRecorder()
	handleListSalesOrders(w, req)
	var orders []SalesOrder
	json.Unmarshal(w.Body.Bytes(), &orders)
	if len(orders) != 2 {
		t.Errorf("expected 2 draft orders, got %d", len(orders))
	}

	// Filter by customer
	req = authedRequest("GET", "/api/v1/sales-orders?customer=Alpha", "", cookie)
	w = httptest.NewRecorder()
	handleListSalesOrders(w, req)
	json.Unmarshal(w.Body.Bytes(), &orders)
	if len(orders) != 1 {
		t.Errorf("expected 1 order for Alpha, got %d", len(orders))
	}
}

func TestConvertQuoteToOrder(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a quote
	body := `{"customer":"Test Customer","status":"accepted","lines":[{"ipn":"IPN-001","description":"Part A","qty":5,"unit_price":100}]}`
	req := authedRequest("POST", "/api/v1/quotes", body, cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	if w.Code != 200 {
		t.Fatalf("create quote: %d %s", w.Code, w.Body.String())
	}
	var q Quote
	json.Unmarshal(w.Body.Bytes(), &q)

	// Convert to order
	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 200 {
		t.Fatalf("convert: %d %s", w.Code, w.Body.String())
	}
	var so SalesOrder
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.QuoteID != q.ID {
		t.Errorf("expected quote_id %s, got %s", q.ID, so.QuoteID)
	}
	if so.Customer != "Test Customer" {
		t.Errorf("expected customer Test Customer, got %s", so.Customer)
	}
	if len(so.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(so.Lines))
	}
	if so.Lines[0].Qty != 5 {
		t.Errorf("expected qty 5, got %d", so.Lines[0].Qty)
	}

	// Converting again should fail (409)
	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 409 {
		t.Errorf("expected 409 on duplicate convert, got %d", w.Code)
	}
}

func TestConvertDraftQuoteFails(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"customer":"Test","status":"draft","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/quotes", body, cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	var q Quote
	json.Unmarshal(w.Body.Bytes(), &q)

	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSalesOrderWorkflow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Seed inventory
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 100, 0, "A1")

	// Create order
	body := `{"customer":"Workflow Corp","lines":[{"ipn":"WIDGET-01","description":"Widget","qty":10,"unit_price":25}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	var so SalesOrder
	json.Unmarshal(w.Body.Bytes(), &so)
	id := so.ID

	// Confirm
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/confirm", "", cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.Status != "confirmed" {
		t.Errorf("expected confirmed, got %s", so.Status)
	}

	// Allocate
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("allocate: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.Status != "allocated" {
		t.Errorf("expected allocated, got %s", so.Status)
	}
	// Verify reservation
	var qtyReserved float64
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyReserved)
	if qtyReserved != 10 {
		t.Errorf("expected 10 reserved, got %.0f", qtyReserved)
	}

	// Pick
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/pick", "", cookie)
	w = httptest.NewRecorder()
	handlePickSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("pick: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.Status != "picked" {
		t.Errorf("expected picked, got %s", so.Status)
	}

	// Ship
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/ship", "", cookie)
	w = httptest.NewRecorder()
	handleShipSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("ship: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.Status != "shipped" {
		t.Errorf("expected shipped, got %s", so.Status)
	}
	// Verify inventory reduced
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyOnHand)
	if qtyOnHand != 90 {
		t.Errorf("expected 90 on hand, got %.0f", qtyOnHand)
	}
	// Verify shipment created
	if so.ShipmentID == nil {
		t.Error("expected shipment_id to be set")
	}

	// Invoice
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/invoice", "", cookie)
	w = httptest.NewRecorder()
	handleInvoiceSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("invoice: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &so)
	if so.Status != "invoiced" {
		t.Errorf("expected invoiced, got %s", so.Status)
	}
	if so.InvoiceID == nil {
		t.Error("expected invoice_id to be set")
	}
	// Verify invoice record
	var inv Invoice
	db.QueryRow("SELECT id,sales_order_id,customer,total_amount FROM invoices WHERE sales_order_id=?", id).
		Scan(&inv.ID, &inv.SalesOrderID, &inv.Customer, &inv.TotalAmount)
	if inv.TotalAmount != 250 {
		t.Errorf("expected invoice total 250, got %.2f", inv.TotalAmount)
	}
}

func TestAllocateInsufficientInventory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Seed inventory with only 5 units
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "SCARCE-01", 5, 0, "A1")

	body := `{"customer":"Test","lines":[{"ipn":"SCARCE-01","qty":10,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	var so SalesOrder
	json.Unmarshal(w.Body.Bytes(), &so)

	// Confirm first
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/confirm", "", cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, so.ID)

	// Allocate should fail
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so.ID)
	if w.Code != 400 {
		t.Errorf("expected 400 for insufficient inventory, got %d", w.Code)
	}
}

func TestSalesOrderInvalidTransition(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"customer":"Test","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	var so SalesOrder
	json.Unmarshal(w.Body.Bytes(), &so)

	// Try to allocate from draft (should fail - needs confirmed)
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so.ID)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid transition, got %d", w.Code)
	}
}
