# ZRP API Reference

Base URL: `http://localhost:9000/api/v1/`

All requests and responses use `Content-Type: application/json`. No authentication is required (bind to localhost or use a reverse proxy for access control).

## Response Format

Successful responses wrap data in an envelope:

```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1, "limit": 50 }
}
```

`meta` is included only for paginated list endpoints. Error responses:

```json
{ "error": "not found" }
```

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Invalid request body |
| 404 | Resource not found |
| 500 | Server error |
| 501 | Not implemented (e.g., CSV write operations) |

---

## Dashboard

### GET /dashboard

Returns summary KPIs.

```bash
curl http://localhost:9000/api/v1/dashboard
```

```json
{
  "open_ecos": 2,
  "low_stock": 1,
  "open_pos": 1,
  "active_wos": 1,
  "open_ncrs": 1,
  "open_rmas": 1,
  "total_parts": 150,
  "total_devices": 3
}
```

Note: The dashboard response is **not** wrapped in the `{data}` envelope — it returns the object directly.

---

## Parts

Parts are read from gitplm CSV files on disk. Write operations return 501.

### GET /parts

List parts with optional filtering and pagination.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `category` | string | Filter by category name |
| `q` | string | Full-text search across IPN and all fields |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 50) |

```bash
curl "http://localhost:9000/api/v1/parts?category=capacitors&q=murata&page=1&limit=20"
```

```json
{
  "data": [
    {
      "ipn": "CAP-001-0001",
      "fields": {
        "IPN": "CAP-001-0001",
        "Manufacturer": "Murata",
        "MPN": "GRM188R71C104KA01",
        "Value": "100nF",
        "_category": "capacitors"
      }
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20 }
}
```

### GET /parts/{ipn}

Get a single part by IPN.

```bash
curl http://localhost:9000/api/v1/parts/CAP-001-0001
```

### POST /parts → 501

### PUT /parts/{ipn} → 501

### DELETE /parts/{ipn} → 501

Parts are managed through gitplm CSV files. Edit the CSVs directly.

---

## Categories

### GET /categories

List all part categories with column schemas and part counts.

```bash
curl http://localhost:9000/api/v1/categories
```

```json
{
  "data": [
    { "id": "capacitors", "name": "capacitors", "count": 45, "columns": ["IPN", "MPN", "Manufacturer", "Value", "Package"] }
  ]
}
```

### POST /categories/{id}/columns

Add a column to a category (stub — not yet implemented for CSV backend).

```bash
curl -X POST http://localhost:9000/api/v1/categories/capacitors/columns \
  -d '{"name": "Voltage Rating"}'
```

### DELETE /categories/{id}/columns/{colName}

Remove a column (stub).

---

## ECOs

### GET /ecos

List all ECOs, optionally filtered by status.

**Query Parameters:** `status` (draft, review, approved, implemented, rejected)

```bash
curl http://localhost:9000/api/v1/ecos?status=draft
```

```json
{
  "data": [
    {
      "id": "ECO-2026-001",
      "title": "Update power supply capacitor",
      "description": "Replace C12 with higher voltage rating",
      "status": "draft",
      "priority": "high",
      "affected_ipns": "[\"CAP-001-0001\"]",
      "created_by": "engineer",
      "created_at": "2026-02-17 20:46:55",
      "updated_at": "2026-02-17 20:46:55",
      "approved_at": null,
      "approved_by": null
    }
  ]
}
```

### GET /ecos/{id}

```bash
curl http://localhost:9000/api/v1/ecos/ECO-2026-001
```

### POST /ecos

Create an ECO. Auto-generates ID as `ECO-{YEAR}-{NNN}`.

```bash
curl -X POST http://localhost:9000/api/v1/ecos \
  -d '{
    "title": "Change resistor value",
    "description": "R15 should be 10K not 1K",
    "priority": "normal",
    "affected_ipns": "[\"RES-001-0001\"]"
  }'
```

**Fields:** `title` (required), `description`, `status` (default: draft), `priority` (default: normal), `affected_ipns` (JSON string array)

### PUT /ecos/{id}

Update an ECO. Send all fields.

```bash
curl -X PUT http://localhost:9000/api/v1/ecos/ECO-2026-001 \
  -d '{
    "title": "Updated title",
    "description": "Updated desc",
    "status": "review",
    "priority": "high",
    "affected_ipns": "[]"
  }'
```

### POST /ecos/{id}/approve

Sets status to `approved`, records approver and timestamp.

```bash
curl -X POST http://localhost:9000/api/v1/ecos/ECO-2026-001/approve
```

### POST /ecos/{id}/implement

Sets status to `implemented`.

```bash
curl -X POST http://localhost:9000/api/v1/ecos/ECO-2026-001/implement
```

---

## Documents

### GET /docs

List all documents, ordered by creation date descending.

```bash
curl http://localhost:9000/api/v1/docs
```

```json
{
  "data": [
    {
      "id": "DOC-2026-001",
      "title": "Assembly Procedure - Z1000",
      "category": "procedure",
      "ipn": "",
      "revision": "B",
      "status": "approved",
      "content": "# Assembly Procedure\n\nStep 1: Place PCB...",
      "file_path": "",
      "created_by": "engineer",
      "created_at": "2026-02-17 20:46:55",
      "updated_at": "2026-02-17 20:46:55"
    }
  ]
}
```

### GET /docs/{id}

### POST /docs

```bash
curl -X POST http://localhost:9000/api/v1/docs \
  -d '{
    "title": "Test Procedure",
    "category": "procedure",
    "ipn": "PCB-001-0001",
    "content": "# Test Steps\n\n1. Power on..."
  }'
```

**Fields:** `title` (required), `category`, `ipn`, `revision` (default: A), `status` (default: draft), `content`, `file_path`

### PUT /docs/{id}

### POST /docs/{id}/approve

Sets status to `approved`.

---

## Vendors

### GET /vendors

List all vendors, ordered by name.

```json
{
  "data": [
    {
      "id": "V-001",
      "name": "DigiKey",
      "website": "https://digikey.com",
      "contact_name": "Sales Team",
      "contact_email": "sales@digikey.com",
      "contact_phone": "",
      "notes": "",
      "status": "preferred",
      "lead_time_days": 3,
      "created_at": "2026-02-17 20:46:55"
    }
  ]
}
```

### GET /vendors/{id}

### POST /vendors

```bash
curl -X POST http://localhost:9000/api/v1/vendors \
  -d '{"name": "Arrow Electronics", "website": "https://arrow.com", "status": "active", "lead_time_days": 7}'
```

**Fields:** `name` (required), `website`, `contact_name`, `contact_email`, `contact_phone`, `notes`, `status` (default: active), `lead_time_days`

ID is auto-generated as `V-{NNN}`.

### PUT /vendors/{id}

### DELETE /vendors/{id}

```bash
curl -X DELETE http://localhost:9000/api/v1/vendors/V-003
```

---

## Inventory

### GET /inventory

List all inventory items. Use `low_stock=true` to filter items at or below reorder point.

```bash
curl "http://localhost:9000/api/v1/inventory?low_stock=true"
```

```json
{
  "data": [
    {
      "ipn": "RES-001-0001",
      "qty_on_hand": 25,
      "qty_reserved": 0,
      "location": "Bin B-03",
      "reorder_point": 100,
      "reorder_qty": 500,
      "updated_at": "2026-02-17 20:46:55"
    }
  ]
}
```

### GET /inventory/{ipn}

### POST /inventory/transact

Create an inventory transaction. Supported types: `receive`, `issue`, `return`, `adjust`.

- `receive` / `return`: adds qty to on-hand
- `issue`: subtracts qty from on-hand
- `adjust`: sets on-hand to the given qty

```bash
curl -X POST http://localhost:9000/api/v1/inventory/transact \
  -d '{"ipn": "CAP-001-0001", "type": "receive", "qty": 500, "reference": "PO-2026-0003", "notes": "Restock"}'
```

### GET /inventory/{ipn}/history

Transaction history for a specific IPN, ordered by date descending.

```bash
curl http://localhost:9000/api/v1/inventory/CAP-001-0001/history
```

---

## Purchase Orders

### GET /pos

List all POs, ordered by creation date descending.

### GET /pos/{id}

Returns PO with line items.

```json
{
  "data": {
    "id": "PO-2026-0001",
    "vendor_id": "V-001",
    "status": "received",
    "notes": "Capacitor order",
    "created_at": "2026-02-17 20:46:55",
    "expected_date": "2026-03-01",
    "received_at": null,
    "lines": [
      {
        "id": 1,
        "po_id": "PO-2026-0001",
        "ipn": "CAP-001-0001",
        "mpn": "GRM188R71C104KA01",
        "manufacturer": "Murata",
        "qty_ordered": 1000,
        "qty_received": 1000,
        "unit_price": 0.02,
        "notes": ""
      }
    ]
  }
}
```

### POST /pos

Create a PO with line items.

```bash
curl -X POST http://localhost:9000/api/v1/pos \
  -d '{
    "vendor_id": "V-001",
    "expected_date": "2026-04-01",
    "notes": "Quarterly restock",
    "lines": [
      {"ipn": "CAP-001-0001", "mpn": "GRM188R71C104KA01", "manufacturer": "Murata", "qty_ordered": 500, "unit_price": 0.02}
    ]
  }'
```

### PUT /pos/{id}

Update PO header (vendor, status, notes, expected date).

### POST /pos/{id}/receive

Receive line items. Automatically updates inventory and creates transactions. Marks PO as `partial` or `received` based on fulfillment.

```bash
curl -X POST http://localhost:9000/api/v1/pos/PO-2026-0002/receive \
  -d '{"lines": [{"id": 2, "qty": 250}]}'
```

---

## Work Orders

### GET /workorders

### GET /workorders/{id}

### POST /workorders

```bash
curl -X POST http://localhost:9000/api/v1/workorders \
  -d '{"assembly_ipn": "PCB-001-0001", "qty": 25, "priority": "high", "notes": "Rush order"}'
```

**Fields:** `assembly_ipn` (required), `qty` (default: 1), `status` (default: open), `priority` (default: normal), `notes`

### PUT /workorders/{id}

Setting status to `in_progress` auto-sets `started_at`. Setting to `completed` auto-sets `completed_at`.

### GET /workorders/{id}/bom

BOM availability check. Returns inventory levels for all tracked components.

```json
{
  "data": {
    "assembly_ipn": "PCB-001-0001",
    "wo_qty": 10,
    "bom": [
      { "ipn": "CAP-001-0001", "qty_required": 10, "qty_on_hand": 500, "available": true }
    ]
  }
}
```

---

## Test Records

### GET /tests

List all test records, ordered by date descending.

### GET /tests/{serial_number}

Get all test records for a specific serial number.

```bash
curl http://localhost:9000/api/v1/tests/SN-001
```

### POST /tests

```bash
curl -X POST http://localhost:9000/api/v1/tests \
  -d '{
    "serial_number": "SN-005",
    "ipn": "PCB-001-0001",
    "firmware_version": "1.3.0",
    "test_type": "factory",
    "result": "pass",
    "measurements": "{\"voltage\":12.05,\"current\":0.48}"
  }'
```

**Fields:** `serial_number` (required), `ipn` (required), `result` (required: pass/fail), `firmware_version`, `test_type`, `measurements` (JSON string), `notes`

---

## NCRs

### GET /ncrs

### GET /ncrs/{id}

### POST /ncrs

```bash
curl -X POST http://localhost:9000/api/v1/ncrs \
  -d '{
    "title": "Cold solder joint on J1",
    "description": "Pin 3 of J1 has insufficient solder",
    "ipn": "PCB-001-0001",
    "serial_number": "SN-005",
    "defect_type": "workmanship",
    "severity": "major"
  }'
```

**Fields:** `title` (required), `description`, `ipn`, `serial_number`, `defect_type`, `severity` (default: minor), `status` (default: open)

### PUT /ncrs/{id}

Update NCR. Setting status to `resolved` or `closed` auto-sets `resolved_at`.

**Additional fields for update:** `root_cause`, `corrective_action`

---

## Devices

### GET /devices

### GET /devices/{serial_number}

### POST /devices

```bash
curl -X POST http://localhost:9000/api/v1/devices \
  -d '{
    "serial_number": "SN-010",
    "ipn": "PCB-001-0001",
    "firmware_version": "1.3.0",
    "customer": "NewCorp",
    "location": "Data Center C",
    "install_date": "2026-02-17"
  }'
```

### PUT /devices/{serial_number}

### GET /devices/{serial_number}/history

Returns combined test records and firmware campaign participation for a device.

```json
{
  "data": {
    "tests": [ ... ],
    "campaigns": [ ... ]
  }
}
```

---

## Firmware Campaigns

### GET /campaigns

### GET /campaigns/{id}

### POST /campaigns

```bash
curl -X POST http://localhost:9000/api/v1/campaigns \
  -d '{"name": "v1.4.0 Feature Release", "version": "1.4.0", "category": "public", "notes": "New telemetry features"}'
```

**Fields:** `name` (required), `version` (required), `category` (default: public), `status` (default: draft), `target_filter`, `notes`

### PUT /campaigns/{id}

### POST /campaigns/{id}/launch

Enrolls all active devices and sets campaign to `active`.

```bash
curl -X POST http://localhost:9000/api/v1/campaigns/FW-2026-001/launch
```

```json
{ "data": { "launched": true, "devices_added": 2 } }
```

### GET /campaigns/{id}/progress

```bash
curl http://localhost:9000/api/v1/campaigns/FW-2026-001/progress
```

```json
{ "data": { "total": 2, "pending": 2, "sent": 0, "updated": 0, "failed": 0 } }
```

---

## RMAs

### GET /rmas

### GET /rmas/{id}

### POST /rmas

```bash
curl -X POST http://localhost:9000/api/v1/rmas \
  -d '{"serial_number": "SN-010", "customer": "NewCorp", "reason": "Overheating", "defect_description": "Unit shuts down after 30 min"}'
```

### PUT /rmas/{id}

Setting status to `received` auto-sets `received_at`. Setting to `closed` or `shipped` auto-sets `resolved_at`.

**Additional fields:** `resolution`

---

## Quotes

### GET /quotes

### GET /quotes/{id}

Returns quote with line items.

### POST /quotes

```bash
curl -X POST http://localhost:9000/api/v1/quotes \
  -d '{
    "customer": "MegaCorp",
    "valid_until": "2026-06-01",
    "notes": "100 unit order",
    "lines": [
      {"ipn": "PCB-001-0001", "description": "Z1000 Power Module", "qty": 100, "unit_price": 139.99}
    ]
  }'
```

### PUT /quotes/{id}

### GET /quotes/{id}/cost

Cost rollup — calculates line totals and grand total.

```json
{
  "data": {
    "lines": [
      { "ipn": "PCB-001-0001", "description": "Z1000 Power Module", "qty": 50, "unit_price": 149.99, "line_total": 7499.50 }
    ],
    "total": 7499.50
  }
}
```
