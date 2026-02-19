# ZRP API Documentation

Base URL: `/api/v1`

All responses are wrapped in `{"data": ...}` envelope. Paginated endpoints include `{"data": [...], "meta": {"total": N, "page": P, "limit": L}}`.

Authentication is via session cookie (set by `/auth/login`).

## Auth (outside `/api/v1`)

### POST /auth/login
```json
// Request
{"username": "admin", "password": "secret"}
// Response
{"user": {"id": 1, "username": "admin", "display_name": "Admin", "role": "admin"}}
```

### POST /auth/logout
No body required. Clears session.

### GET /auth/me
Returns current user or 401.

### POST /auth/change-password
```json
{"current_password": "old", "new_password": "new"}
```

---

## Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/dashboard` | Dashboard stats |
| GET | `/dashboard/charts` | Chart data |
| GET | `/dashboard/lowstock` | Low stock alerts |
| GET | `/dashboard/widgets` | User widget config |
| PUT | `/dashboard/widgets` | Update widget config |

### GET /dashboard
```json
// Response
{"data": {"total_parts": 150, "low_stock_alerts": 3, "active_work_orders": 5, "pending_ecos": 2, "total_inventory_value": 45000.50}}
```

## Search

### GET /search?q=query
Global search across parts, documents, ECOs, etc.

---

## Parts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/parts` | List parts (paginated) |
| POST | `/parts` | Create part |
| GET | `/parts/categories` | List categories (alias) |
| GET | `/parts/check-ipn?ipn=X` | Check IPN exists |
| GET | `/parts/{ipn}` | Get part |
| PUT | `/parts/{ipn}` | Update part |
| DELETE | `/parts/{ipn}` | Delete part |
| GET | `/parts/{ipn}/bom` | BOM tree |
| GET | `/parts/{ipn}/cost` | Cost info |
| GET | `/parts/{ipn}/where-used` | Where-used |
| GET | `/parts/{ipn}/changes` | List pending changes |
| POST | `/parts/{ipn}/changes` | Create changes |
| DELETE | `/parts/{ipn}/changes/{id}` | Delete change |
| POST | `/parts/{ipn}/changes/create-eco` | Create ECO from changes |
| GET | `/parts/{ipn}/gitplm-url` | GitPLM URL |
| GET | `/parts/{ipn}/market-pricing` | Market pricing |

### GET /parts?category=X&q=Y&page=1&limit=50
```json
// Response
{"data": [{"ipn": "RES-001", "category": "Resistors", ...}], "meta": {"total": 150, "page": 1, "limit": 50}}
```

### POST /parts
```json
// Request
{"ipn": "RES-001", "category": "Resistors", "fields": {"value": "10k", "package": "0603"}}
```

---

## Categories

| Method | Path | Description |
|--------|------|-------------|
| GET | `/categories` | List categories |
| POST | `/categories` | Create category |
| POST | `/categories/{id}/columns` | Add column |
| DELETE | `/categories/{id}/columns/{col}` | Remove column |

### POST /categories
```json
{"title": "Resistors", "prefix": "RES"}
```

---

## ECOs

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ecos` | List ECOs |
| POST | `/ecos` | Create ECO |
| GET | `/ecos/{id}` | Get ECO |
| PUT | `/ecos/{id}` | Update ECO |
| POST | `/ecos/{id}/approve` | Approve ECO |
| POST | `/ecos/{id}/implement` | Implement ECO |
| GET | `/ecos/{id}/part-changes` | ECO part changes |
| GET | `/ecos/{id}/revisions` | List revisions |
| POST | `/ecos/{id}/revisions` | Create revision |
| GET | `/ecos/{id}/revisions/{rev}` | Get revision |
| POST | `/ecos/{id}/create-pr` | Create Git PR |

---

## Documents

| Method | Path | Description |
|--------|------|-------------|
| GET | `/docs` | List documents |
| POST | `/docs` | Create document |
| GET | `/docs/{id}` | Get document |
| PUT | `/docs/{id}` | Update document |
| GET | `/docs/{id}/versions` | List versions |
| GET | `/docs/{id}/versions/{rev}` | Get version |
| GET | `/docs/{id}/diff?from=A&to=B` | Diff versions |
| POST | `/docs/{id}/release` | Release document |
| POST | `/docs/{id}/revert/{rev}` | Revert to revision |
| POST | `/docs/{id}/push` | Push to Git |
| POST | `/docs/{id}/sync` | Sync from Git |

---

## Vendors

| Method | Path | Description |
|--------|------|-------------|
| GET | `/vendors` | List vendors |
| POST | `/vendors` | Create vendor |
| GET | `/vendors/{id}` | Get vendor |
| PUT | `/vendors/{id}` | Update vendor |
| DELETE | `/vendors/{id}` | Delete vendor |

---

## Inventory

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inventory` | List inventory |
| GET | `/inventory/{ipn}` | Get item |
| GET | `/inventory/{ipn}/history` | Transaction history |
| POST | `/inventory/transact` | Create transaction |
| DELETE | `/inventory/bulk-delete` | Bulk delete items |
| POST | `/inventory/bulk-update` | Bulk update items |

### POST /inventory/transact
```json
{"ipn": "RES-001", "type": "receive", "qty": 100, "reference": "PO-001", "notes": "Received from vendor"}
```

### DELETE /inventory/bulk-delete
```json
{"ipns": ["RES-001", "RES-002"]}
```

---

## Purchase Orders

| Method | Path | Description |
|--------|------|-------------|
| GET | `/pos` | List POs |
| POST | `/pos` | Create PO |
| GET | `/pos/{id}` | Get PO |
| PUT | `/pos/{id}` | Update PO |
| POST | `/pos/{id}/receive` | Receive PO |
| POST | `/pos/generate` | Generate PO from work order |

### POST /pos/generate
```json
// Request
{"wo_id": "WO-001", "vendor_id": "V-001"}
// Response
{"data": {"po_id": "PO-005", "lines": 3}}
```

---

## Receiving/Inspection

| Method | Path | Description |
|--------|------|-------------|
| GET | `/receiving` | List inspections |
| POST | `/receiving/{id}/inspect` | Inspect received items |

---

## Work Orders

| Method | Path | Description |
|--------|------|-------------|
| GET | `/workorders` | List work orders |
| POST | `/workorders` | Create work order |
| GET | `/workorders/{id}` | Get work order |
| PUT | `/workorders/{id}` | Update work order |
| GET | `/workorders/{id}/bom` | BOM with shortage analysis |
| GET | `/workorders/{id}/pdf` | Download PDF |
| POST | `/workorders/bulk-update` | Bulk update |

---

## Tests

| Method | Path | Description |
|--------|------|-------------|
| GET | `/tests` | List test records |
| POST | `/tests` | Create test record |
| GET | `/tests/{idOrSerial}` | Get by numeric ID (single) or serial (list) |

### GET /tests/42
Returns single test record with id=42.

### GET /tests/SN-12345
Returns all test records for serial number "SN-12345".

---

## NCRs

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ncrs` | List NCRs |
| POST | `/ncrs` | Create NCR |
| GET | `/ncrs/{id}` | Get NCR |
| PUT | `/ncrs/{id}` | Update NCR |

---

## Devices

| Method | Path | Description |
|--------|------|-------------|
| GET | `/devices` | List devices |
| POST | `/devices` | Create device |
| GET | `/devices/{serial}` | Get device |
| PUT | `/devices/{serial}` | Update device |
| GET | `/devices/{serial}/history` | Device history |
| POST | `/devices/import` | Import CSV |
| GET | `/devices/export` | Export CSV |
| POST | `/devices/bulk-update` | Bulk update |

---

## Firmware Campaigns

| Method | Path | Description |
|--------|------|-------------|
| GET | `/firmware` | List campaigns |
| POST | `/firmware` | Create campaign |
| GET | `/firmware/{id}` | Get campaign |
| PUT | `/firmware/{id}` | Update campaign |
| GET | `/firmware/{id}/devices` | Campaign devices |

> **Note:** Backend also supports `/campaigns/*` as the canonical path. The `/firmware/*` paths are aliases for frontend compatibility.

---

## Shipments

| Method | Path | Description |
|--------|------|-------------|
| GET | `/shipments` | List shipments |
| POST | `/shipments` | Create shipment |
| GET | `/shipments/{id}` | Get shipment |
| PUT | `/shipments/{id}` | Update shipment |
| POST | `/shipments/{id}/ship` | Mark shipped |
| POST | `/shipments/{id}/deliver` | Mark delivered |
| GET | `/shipments/{id}/pack-list` | Get pack list |

---

## RMAs

| Method | Path | Description |
|--------|------|-------------|
| GET | `/rmas` | List RMAs |
| POST | `/rmas` | Create RMA |
| GET | `/rmas/{id}` | Get RMA |
| PUT | `/rmas/{id}` | Update RMA |

---

## Quotes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/quotes` | List quotes |
| POST | `/quotes` | Create quote |
| GET | `/quotes/{id}` | Get quote |
| PUT | `/quotes/{id}` | Update quote |
| GET | `/quotes/{id}/pdf` | Export PDF |

---

## Attachments

| Method | Path | Description |
|--------|------|-------------|
| POST | `/attachments` | Upload (multipart) |
| GET | `/attachments` | List attachments |
| GET | `/attachments/{id}/download` | Download file |
| DELETE | `/attachments/{id}` | Delete attachment |

---

## Users

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users` | List users |
| POST | `/users` | Create user |
| PUT | `/users/{id}` | Update user |
| DELETE | `/users/{id}` | Delete user |

---

## API Keys

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api-keys` | List keys |
| POST | `/api-keys` | Create key |
| POST | `/api-keys/{id}/revoke` | Revoke key |

---

## Settings

| Method | Path | Description |
|--------|------|-------------|
| GET/PUT | `/settings/general` | General settings |
| GET/PUT | `/settings/gitplm` | GitPLM config |
| GET/PUT | `/settings/git-docs` | Git docs config |
| POST | `/settings/digikey` | DigiKey settings |
| POST | `/settings/mouser` | Mouser settings |
| GET | `/settings/distributors` | All distributor settings |

---

## Email

| Method | Path | Description |
|--------|------|-------------|
| GET/PUT | `/email/config` | Email config |
| POST | `/email/test` | Send test email |
| GET/PUT | `/email/subscriptions` | Email subscriptions |
| GET | `/email-log` | Email log |

---

## Notifications

| Method | Path | Description |
|--------|------|-------------|
| GET | `/notifications` | List notifications |
| POST | `/notifications/{id}/read` | Mark as read |
| GET | `/notifications/types` | Notification types |
| GET/PUT | `/notifications/preferences` | Preferences |
| PUT | `/notifications/preferences/{type}` | Single pref |

---

## RFQs

| Method | Path | Description |
|--------|------|-------------|
| GET | `/rfqs` | List RFQs |
| POST | `/rfqs` | Create RFQ |
| GET | `/rfqs/{id}` | Get RFQ |
| PUT | `/rfqs/{id}` | Update RFQ |
| DELETE | `/rfqs/{id}` | Delete RFQ |
| POST | `/rfqs/{id}/send` | Send to vendors |
| POST | `/rfqs/{id}/award` | Award to vendor |
| POST | `/rfqs/{id}/award-lines` | Award per line |
| GET | `/rfqs/{id}/compare` | Compare quotes |
| POST | `/rfqs/{id}/quotes` | Create quote |
| PUT | `/rfqs/{id}/quotes/{qid}` | Update quote |
| POST | `/rfqs/{id}/close` | Close RFQ |
| GET | `/rfqs/{id}/email` | Email template |
| GET | `/rfq-dashboard` | RFQ dashboard |

---

## Product Pricing

| Method | Path | Description |
|--------|------|-------------|
| GET | `/pricing` | List pricing |
| POST | `/pricing` | Create pricing |
| GET | `/pricing/{id}` | Get pricing |
| PUT | `/pricing/{id}` | Update pricing |
| DELETE | `/pricing/{id}` | Delete pricing |
| GET | `/pricing/analysis` | Cost analysis |
| POST | `/pricing/analysis` | Create analysis |
| POST | `/pricing/bulk-update` | Bulk update |
| GET | `/pricing/history/{ipn}` | Price history |

---

## Reports

| Method | Path | Description |
|--------|------|-------------|
| GET | `/reports/inventory-valuation` | Inventory valuation |
| GET | `/reports/open-ecos` | Open ECOs |
| GET | `/reports/wo-throughput` | WO throughput |
| GET | `/reports/low-stock` | Low stock |
| GET | `/reports/ncr-summary` | NCR summary |

---

## Calendar

### GET /calendar?year=2026&month=2
Returns calendar events for the specified month.

---

## Changes & Undo

| Method | Path | Description |
|--------|------|-------------|
| GET | `/changes/recent` | Recent changes |
| POST | `/changes/{id}` | Undo change |
| GET | `/undo` | List undo entries |
| POST | `/undo/{id}` | Perform undo |

---

## Admin / Backups

| Method | Path | Description |
|--------|------|-------------|
| POST | `/admin/backup` | Create backup |
| GET | `/admin/backups` | List backups |
| GET | `/admin/backups/{filename}` | Download backup |
| DELETE | `/admin/backups/{filename}` | Delete backup |
| POST | `/admin/restore` | Restore backup |

---

## Field Reports

| Method | Path | Description |
|--------|------|-------------|
| GET | `/field-reports` | List field reports |
| POST | `/field-reports` | Create field report |
| GET | `/field-reports/{id}` | Get field report |
| PUT | `/field-reports/{id}` | Update field report |
| DELETE | `/field-reports/{id}` | Delete field report |
| POST | `/field-reports/{id}/create-ncr` | Create NCR from report |
