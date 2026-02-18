# ZRP Architecture

## System Overview

```
                 ┌─────────────────────────────────────────┐
                 │              Browser (SPA)               │
                 │  index.html + /static/modules/*.js       │
                 │  Tailwind CSS · hash-based routing       │
                 └────────────────┬────────────────────────┘
                                  │ HTTP (JSON)
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Go HTTP Server                           │
│                                                                 │
│  main.go        — mux routing, response helpers                 │
│  middleware.go   — CORS headers, request logging                │
│  handler_*.go   — one file per module                           │
│                                                                 │
│  ┌──────────────────────┐    ┌──────────────────────────┐       │
│  │   SQLite (WAL mode)  │    │  gitplm CSV Files (disk) │       │
│  │   zrp.db             │    │  ~/parts/database/       │       │
│  │   17 tables          │    │  read on every request   │       │
│  └──────────────────────┘    └──────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Request Lifecycle

1. Browser makes `fetch()` call to `/api/v1/{resource}`
2. `middleware.go` `logging()` wrapper: sets CORS headers, handles OPTIONS preflight, logs method/path/duration
3. `main.go` route switch matches path segments and HTTP method
4. Handler function in `handler_{module}.go` executes:
   - Decodes JSON body (for POST/PUT)
   - Queries SQLite or reads CSV files
   - Returns JSON via `jsonResp()`, `jsonRespMeta()`, or `jsonErr()`
5. Response sent to browser

## Database Schema

All tables use SQLite. The database runs in WAL (Write-Ahead Logging) mode for concurrent read performance. Tables are created via `runMigrations()` in `db.go` on startup.

### ecos
Engineering Change Orders.

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | Auto-generated: `ECO-{YEAR}-{NNN}` |
| title | TEXT NOT NULL | Short description |
| description | TEXT | Full details |
| status | TEXT | draft, review, approved, implemented, rejected |
| priority | TEXT | low, normal, high |
| affected_ipns | TEXT | JSON array of IPN strings |
| created_by | TEXT | Default: "engineer" |
| created_at | DATETIME | Auto-set |
| updated_at | DATETIME | Auto-set |
| approved_at | DATETIME | Set on approval |
| approved_by | TEXT | Set on approval |

### documents

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `DOC-{YEAR}-{NNN}` |
| title | TEXT NOT NULL | Document title |
| category | TEXT | procedure, spec, drawing, etc. |
| ipn | TEXT | Linked part number |
| revision | TEXT | Letter revision (A, B, C...) |
| status | TEXT | draft, approved |
| content | TEXT | Markdown content |
| file_path | TEXT | External file reference |
| created_by | TEXT | Default: "engineer" |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### vendors

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `V-{NNN}` |
| name | TEXT NOT NULL | Company name |
| website | TEXT | URL |
| contact_name | TEXT | |
| contact_email | TEXT | |
| contact_phone | TEXT | |
| notes | TEXT | |
| status | TEXT | active, preferred, inactive |
| lead_time_days | INTEGER | Default: 0 |
| created_at | DATETIME | |

### inventory

| Column | Type | Description |
|--------|------|-------------|
| ipn | TEXT PK | Part number |
| qty_on_hand | REAL | Current stock |
| qty_reserved | REAL | Allocated to WOs |
| location | TEXT | Bin/shelf location |
| reorder_point | REAL | Low stock threshold |
| reorder_qty | REAL | Suggested reorder amount |
| updated_at | DATETIME | |

### inventory_transactions

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| ipn | TEXT NOT NULL | Part number |
| type | TEXT NOT NULL | receive, issue, return, adjust |
| qty | REAL NOT NULL | Positive for receive/return, negative for issue |
| reference | TEXT | PO or WO number |
| notes | TEXT | |
| created_at | DATETIME | |

### purchase_orders

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `PO-{YEAR}-{NNNN}` |
| vendor_id | TEXT | FK to vendors |
| status | TEXT | draft, sent, partial, received, cancelled |
| notes | TEXT | |
| created_at | DATETIME | |
| expected_date | DATE | |
| received_at | DATETIME | Set when fully received |

### po_lines

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| po_id | TEXT NOT NULL | FK to purchase_orders |
| ipn | TEXT NOT NULL | Part number |
| mpn | TEXT | Manufacturer part number |
| manufacturer | TEXT | |
| qty_ordered | REAL | |
| qty_received | REAL | Default: 0, incremented on receive |
| unit_price | REAL | |
| notes | TEXT | |

### work_orders

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `WO-{YEAR}-{NNNN}` |
| assembly_ipn | TEXT NOT NULL | Assembly being built |
| qty | INTEGER | Default: 1 |
| status | TEXT | open, in_progress, completed |
| priority | TEXT | low, normal, high |
| notes | TEXT | |
| created_at | DATETIME | |
| started_at | DATETIME | Auto-set when status → in_progress |
| completed_at | DATETIME | Auto-set when status → completed |

### wo_serials

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| wo_id | TEXT NOT NULL | FK to work_orders |
| serial_number | TEXT NOT NULL UNIQUE | |
| status | TEXT | building, testing, passed |
| notes | TEXT | |

### test_records

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| serial_number | TEXT NOT NULL | Device under test |
| ipn | TEXT NOT NULL | Part number |
| firmware_version | TEXT | |
| test_type | TEXT | factory, burn-in, final, etc. |
| result | TEXT NOT NULL | pass, fail |
| measurements | TEXT | JSON object with measured values |
| notes | TEXT | |
| tested_by | TEXT | Default: "operator" |
| tested_at | DATETIME | |

### ncrs

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `NCR-{YEAR}-{NNN}` |
| title | TEXT NOT NULL | |
| description | TEXT | |
| ipn | TEXT | Affected part |
| serial_number | TEXT | Affected unit |
| defect_type | TEXT | workmanship, component, design |
| severity | TEXT | minor, major, critical |
| status | TEXT | open, investigating, resolved, closed |
| root_cause | TEXT | |
| corrective_action | TEXT | |
| created_at | DATETIME | |
| resolved_at | DATETIME | Auto-set on resolve/close |

### devices

| Column | Type | Description |
|--------|------|-------------|
| serial_number | TEXT PK | |
| ipn | TEXT NOT NULL | Product type |
| firmware_version | TEXT | Current firmware |
| customer | TEXT | |
| location | TEXT | Physical location |
| status | TEXT | active, inactive, rma, decommissioned |
| install_date | DATE | |
| last_seen | DATETIME | |
| notes | TEXT | |
| created_at | DATETIME | |

### firmware_campaigns

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `FW-{YEAR}-{NNN}` |
| name | TEXT NOT NULL | Campaign name |
| version | TEXT NOT NULL | Target firmware version |
| category | TEXT | public, beta, internal |
| status | TEXT | draft, active, completed |
| target_filter | TEXT | Device filter criteria |
| notes | TEXT | |
| created_at | DATETIME | |
| started_at | DATETIME | Set on launch |
| completed_at | DATETIME | |

### campaign_devices

| Column | Type | Description |
|--------|------|-------------|
| campaign_id | TEXT | Composite PK |
| serial_number | TEXT | Composite PK |
| status | TEXT | pending, sent, updated, failed |
| updated_at | DATETIME | |

### rmas

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `RMA-{YEAR}-{NNN}` |
| serial_number | TEXT NOT NULL | |
| customer | TEXT | |
| reason | TEXT | Customer-stated reason |
| status | TEXT | open, received, diagnosing, repaired, shipped, closed |
| defect_description | TEXT | |
| resolution | TEXT | What was done |
| created_at | DATETIME | |
| received_at | DATETIME | |
| resolved_at | DATETIME | |

### quotes

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT PK | `Q-{YEAR}-{NNN}` |
| customer | TEXT NOT NULL | |
| status | TEXT | draft, sent, accepted, declined, expired |
| notes | TEXT | |
| created_at | DATETIME | |
| valid_until | DATE | |
| accepted_at | DATETIME | |

### quote_lines

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| quote_id | TEXT NOT NULL | FK to quotes |
| ipn | TEXT NOT NULL | |
| description | TEXT | |
| qty | INTEGER | |
| unit_price | REAL | |
| notes | TEXT | |

## Module System

### Backend (Go)

Each module has one handler file (`handler_{module}.go`) containing all HTTP handler functions for that resource. Handlers are registered in `main.go` via a single route switch on path segments.

Pattern for each resource:
- `handleList{Resource}` — GET collection
- `handleGet{Resource}` — GET single by ID
- `handleCreate{Resource}` — POST new
- `handleUpdate{Resource}` — PUT by ID
- Additional action endpoints (approve, receive, launch, etc.)

ID generation uses `nextID(prefix, table, digits)` in `db.go` — queries the table for the highest existing ID in the current year and increments.

### Frontend (Vanilla JS)

Each module is a JS file in `static/modules/` that registers itself as `window.module_{name}`. Each module exports a `render(container)` function that:

1. Fetches data from the API
2. Renders HTML into the provided container element
3. Registers click handlers for CRUD operations using `showModal()` and `api()` helpers

The SPA shell (`index.html`) handles:
- Hash-based routing (`#/parts`, `#/ecos`, etc.)
- Lazy-loading module JS files on first navigation
- Sidebar navigation with active state
- Shared helpers: `api()`, `toast()`, `badge()`, `showModal()`, `getModalValues()`

## gitplm Integration

Parts data lives in a directory of CSV files, specified by the `-pmDir` flag. The directory structure:

```
parts/database/
├── capacitors/
│   └── capacitors.csv
├── resistors/
│   └── resistors.csv
├── connectors.csv
└── ...
```

Each CSV has headers in the first row. The `IPN` (or `part_number` or `pn`) column is used as the unique identifier. A `_category` field is injected from the directory/filename.

**Important:** CSV files are re-read from disk on every API request. No caching. This means edits to CSV files are reflected immediately without restarting the server.

Parts are read-only through the ZRP API. To modify parts, edit the CSV files directly (typically through gitplm's own tooling, which commits changes to git).
