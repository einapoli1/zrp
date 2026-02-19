# ZRP API Quick Reference

Quick reference for all ZRP API endpoints. For detailed examples, see [API_GUIDE.md](API_GUIDE.md).

## Authentication

All `/api/v1/*` endpoints require authentication via:
- **Session cookie** (`zrp_session`) from `/auth/login`, OR
- **Bearer token** (`Authorization: Bearer zrp_XXXXX`)

## Base URL

```
http://localhost:9000
```

## Auth Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/auth/login` | User login | No |
| POST | `/auth/logout` | User logout | Yes |
| GET | `/auth/me` | Get current user | Yes |
| POST | `/auth/change-password` | Change password | Yes |

## Core Endpoints

### Parts

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/parts` | List parts (paginated) | parts:read |
| POST | `/api/v1/parts` | Create part | parts:write |
| GET | `/api/v1/parts/categories` | List categories | parts:read |
| GET | `/api/v1/parts/check-ipn?ipn=X` | Check if IPN exists | parts:read |
| GET | `/api/v1/parts/{ipn}` | Get part details | parts:read |
| PUT | `/api/v1/parts/{ipn}` | Update part | parts:write |
| DELETE | `/api/v1/parts/{ipn}` | Delete part | parts:delete |
| GET | `/api/v1/parts/{ipn}/bom` | Get BOM tree | parts:read |
| GET | `/api/v1/parts/{ipn}/cost` | Get cost breakdown | parts:read |
| GET | `/api/v1/parts/{ipn}/where-used` | Where-used analysis | parts:read |
| GET | `/api/v1/parts/{ipn}/gitplm-url` | Get GitPLM URL | parts:read |
| GET | `/api/v1/parts/{ipn}/market-pricing` | Get market pricing | parts:read |

**Query Parameters (GET /parts):**
- `category` - Filter by category
- `q` - Search query
- `lifecycle_stage` - Filter by stage (design/prototype/production/obsolete)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 50)

### Part Changes & ECOs

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/parts/{ipn}/changes` | List pending changes | parts:read |
| POST | `/api/v1/parts/{ipn}/changes` | Create pending change | parts:write |
| DELETE | `/api/v1/parts/{ipn}/changes/{id}` | Delete change | parts:write |
| POST | `/api/v1/parts/{ipn}/changes/create-eco` | Create ECO from changes | ecos:write |
| GET | `/api/v1/part-changes` | List all pending changes | parts:read |

### Categories

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/categories` | List categories | parts:read |
| POST | `/api/v1/categories` | Create category | parts:write |
| POST | `/api/v1/categories/{id}/columns` | Add custom column | parts:write |
| DELETE | `/api/v1/categories/{id}/columns/{col}` | Delete column | parts:write |

### Inventory

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/inventory` | List inventory | inventory:read |
| POST | `/api/v1/inventory/transact` | Add/remove inventory | inventory:write |
| GET | `/api/v1/inventory/{id}` | Get inventory item | inventory:read |
| GET | `/api/v1/inventory/{id}/history` | Transaction history | inventory:read |
| POST | `/api/v1/inventory/bulk` | Bulk create inventory | inventory:write |
| DELETE | `/api/v1/inventory/bulk-delete` | Bulk delete inventory | inventory:delete |
| POST | `/api/v1/inventory/bulk-update` | Bulk update inventory | inventory:write |

**Query Parameters (GET /inventory):**
- `ipn` - Filter by part IPN
- `location` - Filter by location
- `low_stock` - Show low stock items (true/false)
- `page`, `limit` - Pagination

### Work Orders

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/workorders` | List work orders | workorders:read |
| POST | `/api/v1/workorders` | Create work order | workorders:write |
| GET | `/api/v1/workorders/{id}` | Get work order | workorders:read |
| PUT | `/api/v1/workorders/{id}` | Update work order | workorders:write |
| GET | `/api/v1/workorders/{id}/pdf` | Download PDF | workorders:read |
| GET | `/api/v1/workorders/{id}/bom` | Get BOM for WO | workorders:read |
| POST | `/api/v1/workorders/bulk` | Bulk create WOs | workorders:write |
| POST | `/api/v1/workorders/bulk-update` | Bulk update WOs | workorders:write |

**Query Parameters (GET /workorders):**
- `status` - Filter by status
- `ipn` - Filter by part
- `assigned_to` - Filter by assignee
- `page`, `limit` - Pagination

**Work Order Statuses:** `planned`, `released`, `in-progress`, `completed`, `cancelled`

### Purchase Orders

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/pos` | List purchase orders | pos:read |
| POST | `/api/v1/pos` | Create PO | pos:write |
| GET | `/api/v1/pos/{id}` | Get PO | pos:read |
| PUT | `/api/v1/pos/{id}` | Update PO | pos:write |
| POST | `/api/v1/pos/{id}/receive` | Receive items | pos:write |
| POST | `/api/v1/pos/generate-from-wo` | Generate PO from WO | pos:write |

**Query Parameters (GET /pos):**
- `status` - Filter by status
- `vendor_id` - Filter by vendor
- `page`, `limit` - Pagination

**PO Statuses:** `draft`, `sent`, `acknowledged`, `received`, `closed`

### Receiving/Inspection

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/receiving` | List receiving records | pos:read |
| POST | `/api/v1/receiving/{id}/inspect` | Perform inspection | pos:write |

### ECOs (Engineering Change Orders)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/ecos` | List ECOs | ecos:read |
| POST | `/api/v1/ecos` | Create ECO | ecos:write |
| GET | `/api/v1/ecos/{id}` | Get ECO | ecos:read |
| PUT | `/api/v1/ecos/{id}` | Update ECO | ecos:write |
| POST | `/api/v1/ecos/{id}/approve` | Approve ECO | ecos:approve |
| POST | `/api/v1/ecos/{id}/implement` | Implement ECO | ecos:approve |
| GET | `/api/v1/ecos/{id}/part-changes` | List affected part changes | ecos:read |
| GET | `/api/v1/ecos/{id}/revisions` | List ECO revisions | ecos:read |
| POST | `/api/v1/ecos/{id}/revisions` | Create ECO revision | ecos:write |
| GET | `/api/v1/ecos/{id}/revisions/{rev}` | Get ECO revision | ecos:read |
| POST | `/api/v1/ecos/{id}/create-pr` | Create GitPLM PR | ecos:write |
| POST | `/api/v1/ecos/bulk` | Bulk create ECOs | ecos:write |

**Query Parameters (GET /ecos):**
- `status` - Filter by status
- `priority` - Filter by priority
- `page`, `limit` - Pagination

**ECO Statuses:** `draft`, `submitted`, `approved`, `implemented`, `rejected`

### Documents

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/docs` | List documents | docs:read |
| POST | `/api/v1/docs` | Create document | docs:write |
| GET | `/api/v1/docs/{id}` | Get document | docs:read |
| PUT | `/api/v1/docs/{id}` | Update document | docs:write |
| POST | `/api/v1/docs/{id}/approve` | Approve document | docs:approve |
| POST | `/api/v1/docs/{id}/release` | Release document | docs:approve |
| GET | `/api/v1/docs/{id}/versions` | List versions | docs:read |
| GET | `/api/v1/docs/{id}/versions/{v}` | Get specific version | docs:read |
| GET | `/api/v1/docs/{id}/diff?from=X&to=Y` | Compare versions | docs:read |
| POST | `/api/v1/docs/{id}/revert/{version}` | Revert to version | docs:write |
| POST | `/api/v1/docs/{id}/push` | Push to Git | docs:write |
| POST | `/api/v1/docs/{id}/sync` | Sync from Git | docs:write |

### Vendors

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/vendors` | List vendors | vendors:read |
| POST | `/api/v1/vendors` | Create vendor | vendors:write |
| GET | `/api/v1/vendors/{id}` | Get vendor | vendors:read |
| PUT | `/api/v1/vendors/{id}` | Update vendor | vendors:write |
| DELETE | `/api/v1/vendors/{id}` | Delete vendor | vendors:delete |

### NCRs (Non-Conformance Reports)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/ncrs` | List NCRs | ncrs:read |
| POST | `/api/v1/ncrs` | Create NCR | ncrs:write |
| GET | `/api/v1/ncrs/{id}` | Get NCR | ncrs:read |
| PUT | `/api/v1/ncrs/{id}` | Update NCR | ncrs:write |
| POST | `/api/v1/ncrs/{id}/create-capa` | Create CAPA from NCR | capas:write |
| POST | `/api/v1/ncrs/{id}/create-eco` | Create ECO from NCR | ecos:write |
| POST | `/api/v1/ncrs/bulk` | Bulk create NCRs | ncrs:write |

**NCR Severities:** `critical`, `major`, `minor`  
**NCR Statuses:** `open`, `investigating`, `resolved`, `closed`

### CAPAs (Corrective/Preventive Actions)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/capas` | List CAPAs | capas:read |
| POST | `/api/v1/capas` | Create CAPA | capas:write |
| GET | `/api/v1/capas/{id}` | Get CAPA | capas:read |
| PUT | `/api/v1/capas/{id}` | Update CAPA | capas:write |
| GET | `/api/v1/capas/dashboard` | CAPA dashboard | capas:read |

**CAPA Types:** `corrective`, `preventive`  
**CAPA Statuses:** `open`, `in-progress`, `completed`, `verified`, `closed`

### Tests

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/tests` | List tests | tests:read |
| POST | `/api/v1/tests` | Create test | tests:write |
| GET | `/api/v1/tests/{id}` | Get test | tests:read |

### Devices

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/devices` | List devices | devices:read |
| POST | `/api/v1/devices` | Create device | devices:write |
| GET | `/api/v1/devices/{id}` | Get device | devices:read |
| PUT | `/api/v1/devices/{id}` | Update device | devices:write |
| GET | `/api/v1/devices/{id}/history` | Device history | devices:read |
| POST | `/api/v1/devices/bulk` | Bulk create | devices:write |
| POST | `/api/v1/devices/bulk-update` | Bulk update | devices:write |
| GET | `/api/v1/devices/export` | Export devices | devices:read |
| POST | `/api/v1/devices/import` | Import devices | devices:write |

### Firmware Campaigns

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/campaigns` | List campaigns | devices:read |
| POST | `/api/v1/campaigns` | Create campaign | devices:write |
| GET | `/api/v1/campaigns/{id}` | Get campaign | devices:read |
| PUT | `/api/v1/campaigns/{id}` | Update campaign | devices:write |
| POST | `/api/v1/campaigns/{id}/launch` | Launch campaign | devices:write |
| GET | `/api/v1/campaigns/{id}/progress` | Campaign progress | devices:read |
| GET | `/api/v1/campaigns/{id}/stream` | Progress stream (SSE) | devices:read |
| GET | `/api/v1/campaigns/{id}/devices` | Campaign devices | devices:read |
| POST | `/api/v1/campaigns/{id}/devices/{dev}/mark` | Mark device status | devices:write |

**Note:** `/api/v1/firmware/*` endpoints are aliases for campaigns

### Shipments

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/shipments` | List shipments | shipments:read |
| POST | `/api/v1/shipments` | Create shipment | shipments:write |
| GET | `/api/v1/shipments/{id}` | Get shipment | shipments:read |
| PUT | `/api/v1/shipments/{id}` | Update shipment | shipments:write |
| POST | `/api/v1/shipments/{id}/ship` | Mark as shipped | shipments:write |
| POST | `/api/v1/shipments/{id}/deliver` | Mark as delivered | shipments:write |
| GET | `/api/v1/shipments/{id}/pack-list` | Packing list PDF | shipments:read |

### RMAs (Return Merchandise Authorizations)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/rmas` | List RMAs | rmas:read |
| POST | `/api/v1/rmas` | Create RMA | rmas:write |
| GET | `/api/v1/rmas/{id}` | Get RMA | rmas:read |
| PUT | `/api/v1/rmas/{id}` | Update RMA | rmas:write |
| POST | `/api/v1/rmas/bulk` | Bulk create RMAs | rmas:write |

### Quotes

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/quotes` | List quotes | quotes:read |
| POST | `/api/v1/quotes` | Create quote | quotes:write |
| GET | `/api/v1/quotes/{id}` | Get quote | quotes:read |
| PUT | `/api/v1/quotes/{id}` | Update quote | quotes:write |
| GET | `/api/v1/quotes/{id}/pdf` | Download PDF | quotes:read |
| GET | `/api/v1/quotes/{id}/cost` | Cost analysis | quotes:read |

### RFQs (Request for Quotation)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/rfqs` | List RFQs | rfqs:read |
| POST | `/api/v1/rfqs` | Create RFQ | rfqs:write |
| GET | `/api/v1/rfqs/{id}` | Get RFQ | rfqs:read |
| PUT | `/api/v1/rfqs/{id}` | Update RFQ | rfqs:write |
| DELETE | `/api/v1/rfqs/{id}` | Delete RFQ | rfqs:delete |
| POST | `/api/v1/rfqs/{id}/send` | Send RFQ to vendors | rfqs:write |
| POST | `/api/v1/rfqs/{id}/award` | Award RFQ | rfqs:write |
| POST | `/api/v1/rfqs/{id}/award-lines` | Award per line | rfqs:write |
| GET | `/api/v1/rfqs/{id}/compare` | Compare vendor quotes | rfqs:read |
| POST | `/api/v1/rfqs/{id}/quotes` | Submit vendor quote | rfqs:write |
| PUT | `/api/v1/rfqs/{id}/quotes/{qid}` | Update vendor quote | rfqs:write |
| POST | `/api/v1/rfqs/{id}/close` | Close RFQ | rfqs:write |
| GET | `/api/v1/rfqs/{id}/email` | Email body preview | rfqs:read |
| GET | `/api/v1/rfq-dashboard` | RFQ dashboard | rfqs:read |

### Sales Orders

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/sales-orders` | List sales orders | sales:read |
| POST | `/api/v1/sales-orders` | Create sales order | sales:write |
| GET | `/api/v1/sales-orders/{id}` | Get sales order | sales:read |
| PUT | `/api/v1/sales-orders/{id}` | Update sales order | sales:write |
| POST | `/api/v1/sales-orders/{id}/confirm` | Confirm order | sales:write |
| POST | `/api/v1/sales-orders/{id}/allocate` | Allocate inventory | sales:write |
| POST | `/api/v1/sales-orders/{id}/pick` | Pick items | sales:write |
| POST | `/api/v1/sales-orders/{id}/ship` | Ship order | sales:write |
| POST | `/api/v1/sales-orders/{id}/create-invoice` | Create invoice | sales:write |

### Invoices

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/invoices` | List invoices | invoices:read |
| POST | `/api/v1/invoices` | Create invoice | invoices:write |
| GET | `/api/v1/invoices/{id}` | Get invoice | invoices:read |
| PUT | `/api/v1/invoices/{id}` | Update invoice | invoices:write |
| POST | `/api/v1/invoices/{id}/send` | Send invoice email | invoices:write |
| POST | `/api/v1/invoices/{id}/mark-paid` | Mark as paid | invoices:write |
| GET | `/api/v1/invoices/{id}/pdf` | Download PDF | invoices:read |

### Pricing

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/prices/{ipn}` | List prices for part | prices:read |
| POST | `/api/v1/prices` | Create price | prices:write |
| DELETE | `/api/v1/prices/{id}` | Delete price | prices:delete |
| GET | `/api/v1/prices/{ipn}/trend` | Price trend analysis | prices:read |
| GET | `/api/v1/pricing` | List product pricing | pricing:read |
| POST | `/api/v1/pricing` | Create product pricing | pricing:write |
| GET | `/api/v1/pricing/{id}` | Get product pricing | pricing:read |
| PUT | `/api/v1/pricing/{id}` | Update product pricing | pricing:write |
| DELETE | `/api/v1/pricing/{id}` | Delete product pricing | pricing:delete |
| GET | `/api/v1/pricing/analysis` | List cost analyses | pricing:read |
| POST | `/api/v1/pricing/analysis` | Create cost analysis | pricing:write |
| POST | `/api/v1/pricing/bulk-update` | Bulk update pricing | pricing:write |
| GET | `/api/v1/pricing/history/{ipn}` | Pricing history | pricing:read |

### Field Reports

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/field-reports` | List field reports | field-reports:read |
| POST | `/api/v1/field-reports` | Create field report | field-reports:write |
| GET | `/api/v1/field-reports/{id}` | Get field report | field-reports:read |
| PUT | `/api/v1/field-reports/{id}` | Update field report | field-reports:write |
| DELETE | `/api/v1/field-reports/{id}` | Delete field report | field-reports:delete |
| POST | `/api/v1/field-reports/{id}/create-ncr` | Create NCR from report | ncrs:write |

## Admin & System

### Users

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/users` | List users | Admin only |
| POST | `/api/v1/users` | Create user | Admin only |
| PUT | `/api/v1/users/{id}` | Update user | Admin only |
| DELETE | `/api/v1/users/{id}` | Delete user | Admin only |
| PUT | `/api/v1/users/{id}/password` | Reset password | Admin only |

### API Keys

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/apikeys` | List API keys | Admin only |
| POST | `/api/v1/apikeys` | Create API key | Admin only |
| PUT | `/api/v1/apikeys/{id}` | Enable/disable key | Admin only |
| DELETE | `/api/v1/apikeys/{id}` | Delete API key | Admin only |
| POST | `/api/v1/apikeys/{id}/revoke` | Revoke API key | Admin only |

**Note:** Both `/api/v1/apikeys` and `/api/v1/api-keys` paths are supported

### Permissions (RBAC)

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/permissions` | List all permissions | Admin only |
| GET | `/api/v1/permissions/modules` | List modules | Admin only |
| GET | `/api/v1/permissions/me` | My permissions | Any authenticated |
| PUT | `/api/v1/permissions/{role}` | Set role permissions | Admin only |

### Backups

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| POST | `/api/v1/admin/backup` | Create backup | Admin only |
| GET | `/api/v1/admin/backups` | List backups | Admin only |
| GET | `/api/v1/admin/backups/{filename}` | Download backup | Admin only |
| DELETE | `/api/v1/admin/backups/{filename}` | Delete backup | Admin only |
| POST | `/api/v1/admin/restore` | Restore from backup | Admin only |

### Settings

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/settings/general` | Get general settings | Admin only |
| PUT | `/api/v1/settings/general` | Update settings | Admin only |
| GET | `/api/v1/settings/gitplm` | Get GitPLM config | Admin only |
| PUT | `/api/v1/settings/gitplm` | Update GitPLM config | Admin only |
| GET | `/api/v1/settings/git-docs` | Get Git docs settings | Admin only |
| PUT | `/api/v1/settings/git-docs` | Update Git docs settings | Admin only |
| GET | `/api/v1/settings/email` | Get email config | Admin only |
| PUT | `/api/v1/settings/email` | Update email config | Admin only |
| POST | `/api/v1/settings/email/test` | Test email | Admin only |
| GET | `/api/v1/settings/distributors` | Get distributor settings | Admin only |
| POST | `/api/v1/settings/digikey` | Update Digi-Key settings | Admin only |
| POST | `/api/v1/settings/mouser` | Update Mouser settings | Admin only |

### Email

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/email/config` | Get email config | Admin only |
| PUT | `/api/v1/email/config` | Update email config | Admin only |
| POST | `/api/v1/email/test` | Send test email | Admin only |
| GET | `/api/v1/email/subscriptions` | Get subscriptions | Any authenticated |
| PUT | `/api/v1/email/subscriptions` | Update subscriptions | Any authenticated |
| GET | `/api/v1/email-log` | Email log | Admin only |

### Notifications

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/notifications` | List notifications | Any authenticated |
| POST | `/api/v1/notifications/{id}/read` | Mark as read | Any authenticated |
| GET | `/api/v1/notifications/preferences` | Get preferences | Any authenticated |
| PUT | `/api/v1/notifications/preferences` | Update preferences | Any authenticated |
| PUT | `/api/v1/notifications/preferences/{type}` | Update single pref | Any authenticated |
| GET | `/api/v1/notifications/types` | List notification types | Any authenticated |

### Attachments

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| POST | `/api/v1/attachments` | Upload attachment | Any authenticated |
| GET | `/api/v1/attachments` | List attachments | Any authenticated |
| GET | `/api/v1/attachments/{id}/download` | Download attachment | Any authenticated |
| DELETE | `/api/v1/attachments/{id}` | Delete attachment | Any authenticated |

### Audit & Change History

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/audit` | Audit log | Admin only |
| GET | `/api/v1/changes/recent` | Recent changes | Any authenticated |
| POST | `/api/v1/changes/{id}` | Undo change | Any authenticated |
| GET | `/api/v1/undo` | List undo records | Any authenticated |
| POST | `/api/v1/undo/{id}` | Perform undo | Any authenticated |

### Debug / Monitoring

| Method | Endpoint | Description | Permissions |
|--------|----------|-------------|-------------|
| GET | `/api/v1/debug/query-stats` | Query profiler stats | Admin only |
| GET | `/api/v1/debug/slow-queries` | Slow query log | Admin only |
| GET | `/api/v1/debug/all-queries` | All queries log | Admin only |
| POST | `/api/v1/debug/query-reset` | Reset profiler | Admin only |

## Utility Endpoints

### Search & Discovery

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/search?q=query` | Global search | Yes |
| GET | `/api/v1/scan/{code}` | Barcode/QR lookup | Yes |
| GET | `/api/v1/calendar` | Calendar view | Yes |

### Dashboard & Reporting

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/dashboard` | Dashboard stats | Yes |
| GET | `/api/v1/dashboard/charts` | Chart data | Yes |
| GET | `/api/v1/dashboard/lowstock` | Low stock alerts | Yes |
| GET | `/api/v1/dashboard/widgets` | User widget config | Yes |
| PUT | `/api/v1/dashboard/widgets` | Update widgets | Yes |

### Reports

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/reports/inventory-valuation` | Inventory valuation | Yes |
| GET | `/api/v1/reports/open-ecos` | Open ECOs report | Yes |
| GET | `/api/v1/reports/wo-throughput` | WO throughput | Yes |
| GET | `/api/v1/reports/low-stock` | Low stock report | Yes |
| GET | `/api/v1/reports/ncr-summary` | NCR summary | Yes |

### Config

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/config` | Public config | No |
| GET | `/healthz` | Health check | No |

### WebSocket

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| WS | `/api/v1/ws` | WebSocket connection | Yes (cookie) |

## Common Query Parameters

### Pagination
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 50, max: 100)

### Filtering
- `q` - Search query
- `status` - Status filter
- `category` - Category filter
- `ipn` - Part IPN filter

### Sorting
- `sort` - Sort field
- `order` - Sort direction (asc/desc)

## HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created |
| 400 | Bad Request | Invalid request |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |

## Error Codes

- `UNAUTHORIZED` - Authentication required
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `DUPLICATE_IPN` - Part already exists
- `INVALID_BOM` - BOM validation failed
- `INSUFFICIENT_STOCK` - Not enough inventory
- `VALIDATION_ERROR` - Request validation failed
- `RATE_LIMITED` - Too many requests

## Example cURL Commands

### Authenticate with API Key
```bash
curl http://localhost:9000/api/v1/parts \
  -H "Authorization: Bearer zrp_your_api_key_here"
```

### Create a Part
```bash
curl -X POST http://localhost:9000/api/v1/parts \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "ipn": "RES-001",
    "category": "Resistors",
    "fields": {"value": "10k", "package": "0603"}
  }'
```

### Search Parts
```bash
curl "http://localhost:9000/api/v1/parts?category=Resistors&q=10k&page=1&limit=25" \
  -H "Authorization: Bearer YOUR_KEY"
```

### Create Work Order
```bash
curl -X POST http://localhost:9000/api/v1/workorders \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "ipn": "WIDGET-001",
    "quantity": 100,
    "priority": "high",
    "due_date": "2024-03-15"
  }'
```

---

**For detailed examples and workflows, see [API_GUIDE.md](API_GUIDE.md)**  
**For complete API specification, see [api-spec.yaml](api-spec.yaml)**

**Last Updated:** 2024-02-19
