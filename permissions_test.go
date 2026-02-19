package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupPermTestDB(t *testing.T) {
	t.Helper()
	if err := initDB("file::memory:?cache=shared"); err != nil {
		t.Fatal("initDB:", err)
	}
	seedDB()
	if err := initPermissionsTable(); err != nil {
		t.Fatal("initPermissionsTable:", err)
	}
}

func TestSeedDefaultPermissions(t *testing.T) {
	setupPermTestDB(t)

	// Admin should have all permissions
	for _, mod := range AllModules {
		for _, act := range AllActions {
			if !HasPermission("admin", mod, act) {
				t.Errorf("admin should have %s:%s", mod, act)
			}
		}
	}

	// User should have everything except admin module
	for _, mod := range AllModules {
		for _, act := range AllActions {
			if mod == ModuleAdmin {
				if HasPermission("user", mod, act) {
					t.Errorf("user should NOT have %s:%s", mod, act)
				}
			} else {
				if !HasPermission("user", mod, act) {
					t.Errorf("user should have %s:%s", mod, act)
				}
			}
		}
	}

	// Readonly should only have view
	for _, mod := range AllModules {
		if !HasPermission("readonly", mod, ActionView) {
			t.Errorf("readonly should have %s:view", mod)
		}
		for _, act := range []string{ActionCreate, ActionEdit, ActionDelete, ActionApprove} {
			if HasPermission("readonly", mod, act) {
				t.Errorf("readonly should NOT have %s:%s", mod, act)
			}
		}
	}
}

func TestMapAPIPathToPermission(t *testing.T) {
	tests := []struct {
		path       string
		method     string
		wantMod    string
		wantAction string
	}{
		{"parts", "GET", ModuleParts, ActionView},
		{"parts", "POST", ModuleParts, ActionCreate},
		{"parts/123", "PUT", ModuleParts, ActionEdit},
		{"parts/123", "DELETE", ModuleParts, ActionDelete},
		{"ecos", "GET", ModuleECOs, ActionView},
		{"ecos/123/approve", "POST", ModuleECOs, ActionApprove},
		{"ecos/123/implement", "POST", ModuleECOs, ActionApprove},
		{"inventory", "GET", ModuleInventory, ActionView},
		{"inventory/transact", "POST", ModuleInventory, ActionCreate},
		{"users", "GET", ModuleAdmin, ActionView},
		{"users", "POST", ModuleAdmin, ActionCreate},
		{"apikeys", "GET", ModuleAdmin, ActionView},
		{"api-keys", "DELETE", ModuleAdmin, ActionDelete},
		{"email/config", "PUT", ModuleAdmin, ActionEdit},
		{"settings/email", "GET", ModuleAdmin, ActionView},
		{"settings/general", "PUT", ModuleAdmin, ActionEdit},
		{"workorders", "POST", ModuleWorkOrders, ActionCreate},
		{"ncrs", "GET", ModuleNCRs, ActionView},
		{"rmas", "POST", ModuleRMAs, ActionCreate},
		{"vendors", "PUT", ModuleVendors, ActionEdit},
		{"pos", "GET", ModulePOs, ActionView},
		{"quotes", "POST", ModuleQuotes, ActionCreate},
		{"pricing", "GET", ModulePricing, ActionView},
		{"devices", "GET", ModuleDevices, ActionView},
		{"campaigns", "POST", ModuleFirmware, ActionCreate},
		{"shipments", "GET", ModuleShipments, ActionView},
		{"field-reports", "POST", ModuleFieldReports, ActionCreate},
		{"rfqs", "GET", ModuleRFQs, ActionView},
		{"reports/inventory-valuation", "GET", ModuleReports, ActionView},
		{"tests", "POST", ModuleTesting, ActionCreate},
		{"docs", "GET", ModuleDocuments, ActionView},
		{"docs/123/approve", "POST", ModuleDocuments, ActionApprove},
		{"categories", "POST", ModuleParts, ActionCreate},
		{"receiving", "GET", ModuleInventory, ActionView},
		{"prices", "POST", ModulePricing, ActionCreate},

		// Passthrough routes
		{"dashboard", "GET", "", ""},
		{"search", "GET", "", ""},
		{"scan/123", "GET", "", ""},
		{"audit", "GET", "", ""},
		{"calendar", "GET", "", ""},
		{"notifications", "GET", "", ""},
		{"config", "GET", "", ""},
		{"attachments", "POST", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			mod, act := mapAPIPathToPermission(tt.path, tt.method)
			if mod != tt.wantMod || act != tt.wantAction {
				t.Errorf("got (%q, %q), want (%q, %q)", mod, act, tt.wantMod, tt.wantAction)
			}
		})
	}
}

func TestPermissionBasedRBAC(t *testing.T) {
	setupPermTestDB(t)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	rbac := requireRBAC(okHandler)

	tests := []struct {
		name       string
		method     string
		path       string
		role       string
		wantStatus int
	}{
		// Admin: full access
		{"admin GET parts", "GET", "/api/v1/parts", "admin", 200},
		{"admin POST parts", "POST", "/api/v1/parts", "admin", 200},
		{"admin GET users", "GET", "/api/v1/users", "admin", 200},
		{"admin POST users", "POST", "/api/v1/users", "admin", 200},
		{"admin PUT email config", "PUT", "/api/v1/email/config", "admin", 200},

		// User: CRUD on business objects, no admin endpoints
		{"user GET parts", "GET", "/api/v1/parts", "user", 200},
		{"user POST parts", "POST", "/api/v1/parts", "user", 200},
		{"user PUT ecos", "PUT", "/api/v1/ecos/123", "user", 200},
		{"user DELETE parts", "DELETE", "/api/v1/parts/123", "user", 200},
		{"user GET users DENIED", "GET", "/api/v1/users", "user", 403},
		{"user POST users DENIED", "POST", "/api/v1/users", "user", 403},
		{"user PUT users DENIED", "PUT", "/api/v1/users/1", "user", 403},
		{"user GET apikeys DENIED", "GET", "/api/v1/apikeys", "user", 403},
		{"user POST apikeys DENIED", "POST", "/api/v1/apikeys", "user", 403},
		{"user DELETE apikeys DENIED", "DELETE", "/api/v1/apikeys/1", "user", 403},
		{"user GET email config DENIED", "GET", "/api/v1/email/config", "user", 403},
		{"user PUT email config DENIED", "PUT", "/api/v1/email/config", "user", 403},
		{"user POST email test DENIED", "POST", "/api/v1/email/test", "user", 403},
		{"user GET settings email DENIED", "GET", "/api/v1/settings/email", "user", 403},
		{"user PUT settings email DENIED", "PUT", "/api/v1/settings/email", "user", 403},
		{"user GET email-log allowed", "GET", "/api/v1/email-log", "user", 200},

		// Readonly: view only
		{"readonly GET parts", "GET", "/api/v1/parts", "readonly", 200},
		{"readonly GET users", "GET", "/api/v1/users", "readonly", 200},
		{"readonly POST parts DENIED", "POST", "/api/v1/parts", "readonly", 403},
		{"readonly PUT ecos DENIED", "PUT", "/api/v1/ecos/1", "readonly", 403},
		{"readonly DELETE parts DENIED", "DELETE", "/api/v1/parts/1", "readonly", 403},

		// No role (Bearer token): full access
		{"no role GET", "GET", "/api/v1/parts", "", 200},
		{"no role POST users", "POST", "/api/v1/users", "", 200},

		// Non-API paths pass through
		{"non-api path", "GET", "/auth/login", "readonly", 200},

		// Passthrough routes
		{"user GET dashboard", "GET", "/api/v1/dashboard", "user", 200},
		{"readonly GET dashboard", "GET", "/api/v1/dashboard", "readonly", 200},
		{"user GET search", "GET", "/api/v1/search", "user", 200},
		{"user GET audit", "GET", "/api/v1/audit", "user", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.method, tt.path, tt.role)
			w := httptest.NewRecorder()
			rbac.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestCustomRolePermissions(t *testing.T) {
	setupPermTestDB(t)

	// Create a custom "viewer_plus" role with view + create on parts only
	perms := []Permission{
		{Role: "viewer_plus", Module: ModuleParts, Action: ActionView},
		{Role: "viewer_plus", Module: ModuleParts, Action: ActionCreate},
	}
	if err := setRolePermissions(db, "viewer_plus", perms); err != nil {
		t.Fatal(err)
	}

	// Should have parts:view and parts:create
	if !HasPermission("viewer_plus", ModuleParts, ActionView) {
		t.Error("should have parts:view")
	}
	if !HasPermission("viewer_plus", ModuleParts, ActionCreate) {
		t.Error("should have parts:create")
	}
	// Should NOT have parts:edit or ecos:view
	if HasPermission("viewer_plus", ModuleParts, ActionEdit) {
		t.Error("should NOT have parts:edit")
	}
	if HasPermission("viewer_plus", ModuleECOs, ActionView) {
		t.Error("should NOT have ecos:view")
	}

	// Test via middleware
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	rbac := requireRBAC(okHandler)

	// viewer_plus GET parts → 200
	req := makeRequest("GET", "/api/v1/parts", "viewer_plus")
	w := httptest.NewRecorder()
	rbac.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("viewer_plus GET parts: got %d, want 200", w.Code)
	}

	// viewer_plus POST parts → 200
	req = makeRequest("POST", "/api/v1/parts", "viewer_plus")
	w = httptest.NewRecorder()
	rbac.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("viewer_plus POST parts: got %d, want 200", w.Code)
	}

	// viewer_plus PUT parts → 403
	req = makeRequest("PUT", "/api/v1/parts/123", "viewer_plus")
	w = httptest.NewRecorder()
	rbac.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("viewer_plus PUT parts: got %d, want 403", w.Code)
	}

	// viewer_plus GET ecos → 403
	req = makeRequest("GET", "/api/v1/ecos", "viewer_plus")
	w = httptest.NewRecorder()
	rbac.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("viewer_plus GET ecos: got %d, want 403", w.Code)
	}
}

func TestPermissionAPIEndpoints(t *testing.T) {
	setupPermTestDB(t)

	// Test GET /api/v1/permissions
	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()
	handleListPermissions(w, req)
	if w.Code != 200 {
		t.Fatalf("list permissions: got %d", w.Code)
	}
	var resp struct {
		Data []Permission `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data) == 0 {
		t.Error("expected permissions in response")
	}

	// Test GET /api/v1/permissions?role=readonly
	req = httptest.NewRequest("GET", "/api/v1/permissions?role=readonly", nil)
	w = httptest.NewRecorder()
	handleListPermissions(w, req)
	json.NewDecoder(w.Body).Decode(&resp)
	for _, p := range resp.Data {
		if p.Role != "readonly" {
			t.Errorf("expected only readonly perms, got role=%s", p.Role)
		}
		if p.Action != ActionView {
			t.Errorf("readonly should only have view, got %s", p.Action)
		}
	}

	// Test GET /api/v1/permissions/modules
	req = httptest.NewRequest("GET", "/api/v1/permissions/modules", nil)
	w = httptest.NewRecorder()
	handleListModules(w, req)
	if w.Code != 200 {
		t.Fatalf("list modules: got %d", w.Code)
	}

	// Test GET /api/v1/permissions/me
	req = httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), ctxRole, "user")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handleMyPermissions(w, req)
	if w.Code != 200 {
		t.Fatalf("my permissions: got %d", w.Code)
	}

	// Test PUT /api/v1/permissions/readonly — add create on parts
	body := `{"permissions":[{"module":"parts","action":"view"},{"module":"parts","action":"create"},{"module":"ecos","action":"view"}]}`
	req = httptest.NewRequest("PUT", "/api/v1/permissions/readonly", bytes.NewBufferString(body))
	w = httptest.NewRecorder()
	handleSetPermissions(w, req, "readonly")
	if w.Code != 200 {
		t.Fatalf("set permissions: got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify
	if !HasPermission("readonly", ModuleParts, ActionCreate) {
		t.Error("readonly should now have parts:create")
	}
	if !HasPermission("readonly", ModuleECOs, ActionView) {
		t.Error("readonly should have ecos:view")
	}
	if HasPermission("readonly", ModuleInventory, ActionView) {
		t.Error("readonly should NOT have inventory:view anymore")
	}
}

func TestPermissionSetInvalidInput(t *testing.T) {
	setupPermTestDB(t)

	// Invalid module
	body := `{"permissions":[{"module":"bogus","action":"view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test_role", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handleSetPermissions(w, req, "test_role")
	if w.Code != 400 {
		t.Errorf("invalid module: got %d, want 400", w.Code)
	}

	// Invalid action
	body = `{"permissions":[{"module":"parts","action":"bogus"}]}`
	req = httptest.NewRequest("PUT", "/api/v1/permissions/test_role", bytes.NewBufferString(body))
	w = httptest.NewRecorder()
	handleSetPermissions(w, req, "test_role")
	if w.Code != 400 {
		t.Errorf("invalid action: got %d, want 400", w.Code)
	}
}
