# Contributing to ZRP

## Project Structure

```
zrp/
├── main.go              # Server entry point, routing, JSON helpers
├── db.go                # Database init, migrations, seed data, ID generation
├── types.go             # All struct types
├── middleware.go         # CORS + logging middleware
├── handler_*.go         # One file per module (14 files)
├── static/
│   ├── index.html       # SPA shell (sidebar, routing, shared UI helpers)
│   └── modules/*.js     # One JS file per module (15 files)
├── go.mod
└── go.sum
```

## How to Add a New Module

Example: adding a "Suppliers" module.

### 1. Define the type in `types.go`

```go
type Supplier struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Region    string `json:"region"`
    CreatedAt string `json:"created_at"`
}
```

### 2. Add the table in `db.go`

Add a `CREATE TABLE IF NOT EXISTS` statement to the `tables` slice in `runMigrations()`:

```go
`CREATE TABLE IF NOT EXISTS suppliers (
    id TEXT PRIMARY KEY, name TEXT NOT NULL,
    region TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

### 3. Create `handler_suppliers.go`

```go
package main

import "net/http"

func handleListSuppliers(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id,name,COALESCE(region,''),created_at FROM suppliers ORDER BY name")
    if err != nil { jsonErr(w, err.Error(), 500); return }
    defer rows.Close()
    var items []Supplier
    for rows.Next() {
        var s Supplier
        rows.Scan(&s.ID, &s.Name, &s.Region, &s.CreatedAt)
        items = append(items, s)
    }
    if items == nil { items = []Supplier{} }
    jsonResp(w, items)
}
// ... handleGetSupplier, handleCreateSupplier, etc.
```

### 4. Add routes in `main.go`

In the `/api/v1/` handler switch:

```go
case parts[0] == "suppliers" && len(parts) == 1 && r.Method == "GET":
    handleListSuppliers(w, r)
case parts[0] == "suppliers" && len(parts) == 1 && r.Method == "POST":
    handleCreateSupplier(w, r)
```

### 5. Create `static/modules/suppliers.js`

```javascript
window.module_suppliers = {
  render: async (container) => {
    const res = await api('GET', 'suppliers');
    const items = res.data || [];
    container.innerHTML = `
      <div class="card">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold">Suppliers</h2>
          <button class="btn btn-primary" onclick="window._suppliersNew()">+ New</button>
        </div>
        <!-- table or list here -->
      </div>
    `;
  }
};
```

### 6. Add to `index.html`

Add a sidebar link:

```html
<a class="sidebar-link" data-route="suppliers" onclick="navigate('suppliers')">🏢 Suppliers</a>
```

Add to `moduleNames`:

```javascript
const moduleNames = { ..., suppliers: 'Suppliers' };
```

## Code Conventions

- **Go:** Standard library only (no web frameworks). One handler file per module.
- **SQL:** Use `COALESCE` for nullable columns to avoid nil scan issues.
- **IDs:** Use `nextID(prefix, table, digits)` for auto-generated IDs.
- **Null handling:** Use `sql.NullString` + `sp()` helper for nullable string fields.
- **JS:** Vanilla JS, no build step. Each module is a self-contained `window.module_{name}` object.
- **CSS:** Tailwind via CDN. Use the utility classes defined in `index.html` (`badge`, `btn`, `card`, `input`, `label`, `table-row`).
- **Responses:** Always use `jsonResp()` / `jsonRespMeta()` / `jsonErr()`. Never write raw responses.
- **Empty arrays:** Always initialize nil slices to empty arrays before JSON encoding to avoid `null` in responses.

## Testing

### Go Tests

```bash
go test ./...
```

### Manual Testing

```bash
# Start the server
go run . -pmDir /path/to/parts

# Test an endpoint
curl http://localhost:9000/api/v1/dashboard | jq
```

## Submitting Changes

1. Fork and create a feature branch
2. Make your changes
3. Ensure `go build` succeeds with no errors
4. Test your changes manually
5. Submit a pull request with a clear description of what changed and why
