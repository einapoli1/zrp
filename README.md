# ZRP — Zonit Resource Planning

A single-binary ERP system for hardware electronics manufacturing. Go backend, vanilla JS frontend, SQLite database. No dependencies to deploy — just run the binary.

```
┌──────────────────────────────────────────────┐
│  ZRP                                    ERP  │
├──────────────┬───────────────────────────────┤
│ ▸ Dashboard  │  Dashboard                    │
│              │  ┌──────┐┌──────┐┌──────┐     │
│ ENGINEERING  │  │Open  ││Low   ││Active│     │
│ ▸ Parts      │  │ECOs  ││Stock ││WOs   │     │
│ ▸ ECOs       │  │  2   ││  1   ││  1   │     │
│ ▸ Documents  │  └──────┘└──────┘└──────┘     │
│              │  ┌──────┐┌──────┐┌──────┐     │
│ SUPPLY CHAIN │  │Open  ││Open  ││Total │     │
│ ▸ Inventory  │  │NCRs  ││RMAs  ││Parts │     │
│ ▸ POs        │  │  1   ││  1   ││ 150  │     │
│ ▸ Vendors    │  └──────┘└──────┘└──────┘     │
│              │                                │
│ MANUFACTURING│  Welcome to ZRP               │
│ ▸ Work Orders│  Zonit Resource Planning —     │
│ ▸ Testing    │  your complete ERP for         │
│ ▸ NCRs       │  hardware manufacturing.       │
│              │                                │
│ FIELD        │                                │
│ ▸ Devices    │                                │
│ ▸ Firmware   │                                │
│ ▸ RMAs       │                                │
│              │                                │
│ SALES        │                                │
│ ▸ Quotes     │                                │
└──────────────┴───────────────────────────────┘
```

## Features

- **Parts (PLM)** — Browse and search parts from gitplm CSV files, organized by category
- **ECOs** — Engineering Change Orders with draft → review → approved → implemented workflow
- **Documents** — Revision-controlled procedures, specs, and drawings linked to parts
- **Inventory** — Real-time stock tracking with reorder points, locations, and transaction history
- **Purchase Orders** — PO lifecycle from draft to received, with line items and partial receiving
- **Vendors** — Supplier management with contacts, lead times, and status tracking
- **Work Orders** — Production work orders tied to assembly IPNs with BOM availability checks
- **Test Records** — Factory test results with pass/fail, measurements, and firmware versions
- **NCRs** — Non-Conformance Reports with defect classification, root cause, and corrective actions
- **Device Registry** — Deployed device tracking with serial numbers, customers, and firmware versions
- **Firmware Campaigns** — OTA firmware rollouts targeting active devices with progress tracking
- **RMAs** — Return Merchandise Authorization with full lifecycle from open to resolution
- **Quotes** — Customer quotes with line items, pricing, and cost rollup
- **Dashboard** — At-a-glance KPIs: open ECOs, low stock, active WOs, open NCRs/RMAs

## Quick Start

```bash
git clone https://github.com/zonit/zrp.git
cd zrp
go build -o zrp .
./zrp --pmDir /path/to/gitplm/parts/database
# Open http://localhost:9000
```

## Requirements

- Go 1.24+ (build only — the binary is self-contained)
- No external database required (uses embedded SQLite)
- Tested on macOS (arm64) and Linux (amd64)

## Installation

```bash
# From source
git clone https://github.com/zonit/zrp.git
cd zrp
go build -o zrp .

# Or install directly
go install github.com/zonit/zrp@latest
```

## Configuration

All configuration is via CLI flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-pmDir` | `""` | Path to gitplm parts database directory (CSV files) |
| `-port` | `9000` | HTTP port to listen on |
| `-db` | `zrp.db` | Path to SQLite database file |

Example:
```bash
./zrp -pmDir ~/Documents/Zonit/zonit-dev/parts/database -port 8080 -db /var/data/zrp.db
```

## Running

```bash
./zrp --pmDir /path/to/parts
```

Open [http://localhost:9000](http://localhost:9000) in your browser. The database is created automatically on first run with seed data.

## Modules

### Dashboard
Summary cards showing open ECOs, low stock items, open POs, active work orders, open NCRs, open RMAs, total parts, and total devices. Click any card to jump to that module.

### Parts (PLM)
Reads parts data from gitplm CSV files on disk. Supports category filtering, full-text search across all fields, and pagination. Parts are read-only through the API — edit the CSV files directly and they're reflected immediately.

### ECOs (Engineering Change Orders)
Track proposed changes to parts and assemblies. Workflow: `draft` → `review` → `approved` → `implemented`. Each ECO can reference affected IPNs and tracks who approved it and when.

### Documents
Revision-controlled documentation: procedures, specs, test plans. Each document has a category, optional IPN link, revision letter, and markdown content. Supports `draft` → `approved` workflow.

### Inventory
Per-IPN stock tracking with quantities on hand, reserved quantities, bin locations, and reorder points. Transaction history records every receive, issue, return, and adjustment. Automatically updated when POs are received.

### Purchase Orders
Full PO lifecycle: `draft` → `sent` → `partial` → `received`. Each PO has line items with IPN, MPN, manufacturer, quantities, and unit pricing. Receiving a PO automatically creates inventory transactions and updates stock levels.

### Vendors
Supplier database with contact info, website, lead time in days, and status (`active`, `preferred`, `inactive`).

### Work Orders
Production tracking for assemblies. Each WO specifies an assembly IPN, quantity, and priority. Status flow: `open` → `in_progress` → `completed`. BOM availability endpoint checks inventory against required components.

### Test Records
Factory test results by serial number. Records include IPN, firmware version, test type, pass/fail result, JSON measurements, and tester identity. Queryable by serial number to see full test history.

### NCRs (Non-Conformance Reports)
Quality issue tracking with defect type classification (workmanship, component, design), severity levels (minor, major, critical), root cause analysis, and corrective actions. Links to specific IPNs and serial numbers.

### Device Registry
Field-deployed device tracking. Each device has a serial number, IPN, firmware version, customer, location, and status. History endpoint aggregates test records and firmware campaign participation.

### Firmware Campaigns
Manage OTA firmware rollouts. Create a campaign targeting a firmware version, launch it to automatically enroll all active devices, and track progress (pending, sent, updated, failed).

### RMAs
Return processing from customer complaint through diagnosis to resolution. Tracks serial number, customer, reason, defect description, and resolution. Status flow: `open` → `received` → `diagnosing` → `repaired` → `shipped` → `closed`.

### Quotes
Customer quotes with line items. Each line has an IPN, description, quantity, and unit price. Cost rollup endpoint calculates line totals and grand total. Status: `draft` → `sent` → `accepted`/`declined`/`expired`.

## API

All endpoints are under `/api/v1/`. Requests and responses use JSON. The standard response envelope is:

```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1, "limit": 50 }
}
```

See [docs/API.md](docs/API.md) for the complete API reference.

## Development

### Project Structure

```
zrp/
├── main.go              # HTTP server, routing, response helpers
├── db.go                # SQLite init, migrations, seed data, ID generation
├── types.go             # All Go struct types and API response types
├── middleware.go         # CORS and request logging middleware
├── handler_parts.go     # Parts + Categories + Dashboard handlers
├── handler_eco.go       # ECO handlers
├── handler_docs.go      # Document handlers
├── handler_vendors.go   # Vendor handlers
├── handler_inventory.go # Inventory + transaction handlers
├── handler_procurement.go # PO handlers with receiving logic
├── handler_workorders.go  # Work order + BOM handlers
├── handler_testing.go   # Test record handlers
├── handler_ncr.go       # NCR handlers
├── handler_devices.go   # Device registry + history handlers
├── handler_firmware.go  # Firmware campaign handlers
├── handler_rma.go       # RMA handlers
├── handler_quotes.go    # Quote + cost rollup handlers
├── handler_costing.go   # BOM cost rollup (shared logic)
├── static/
│   ├── index.html       # SPA shell with Tailwind CSS
│   └── modules/         # One JS file per module
│       ├── dashboard.js
│       ├── parts.js
│       ├── eco.js
│       ├── docs.js
│       ├── inventory.js
│       ├── procurement.js
│       ├── vendors.js
│       ├── workorders.js
│       ├── testing.js
│       ├── ncr.js
│       ├── devices.js
│       ├── firmware.js
│       ├── rma.js
│       ├── quotes.js
│       └── costing.js
└── zrp.db               # SQLite database (auto-created)
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o zrp .
```

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## License

MIT
