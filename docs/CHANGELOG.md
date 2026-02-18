# Changelog

All notable changes to ZRP are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/).

## [0.2.0] - 2026-02-18

### Added
- **Authentication** — session-based login/logout with bcrypt password hashing, 24-hour session tokens, and role-based access control (admin, user, readonly)
- **User Management** — admin panel for creating, editing, deactivating users and resetting passwords
- **API Keys** — generate Bearer tokens for programmatic access with optional expiration, enable/disable toggle, and automatic last-used tracking
- **Readonly Role Enforcement** — readonly users can view all data but POST/PUT/DELETE requests return 403
- **Notifications** — automatic notification generation for low stock, overdue work orders (>7 days), aging NCRs (>14 days), and new RMAs; bell icon with unread count; mark-as-read
- **File Attachments** — upload files (up to 32MB) to any module record; file serving at `/files/`; delete with disk cleanup
- **Audit Log** — all create/update/delete/bulk operations logged with username, action, module, record ID, and summary; filterable by module, user, and date range
- **Global Search** — search across parts, ECOs, work orders, devices, NCRs, POs, and quotes simultaneously from the top bar
- **Calendar View** — monthly calendar showing WO due dates (blue), PO expected deliveries (green), and quote expirations (orange)
- **Dashboard Charts** — ECOs by status, work orders by status, and top inventory items by value
- **Dashboard Low Stock Panel** — dedicated view of items below reorder point
- **Bulk Operations** — multi-select with checkboxes for ECOs (approve/implement/reject/delete), work orders (complete/cancel/delete), NCRs (close/resolve/delete), devices (decommission/delete), RMAs (close/delete), and inventory (delete)
- **Work Order PDF Traveler** — printable HTML traveler with assembly info, BOM table, and sign-off section
- **Quote PDF** — printable HTML quote with customer info, line items, subtotal, terms
- **WO BOM Shortage Highlighting** — color-coded status (ok/low/shortage) for each BOM component
- **Parts BOM & Cost Rollup** — view Bill of Materials and cost breakdown for assembly IPNs
- **ECO→Parts Enrichment** — affected IPNs enriched with part details when viewing an ECO
- **NCR→ECO Auto-Link** — ECOs can reference an NCR ID for traceability
- **Generate PO from WO Shortages** — automatically create draft POs for work order BOM shortages
- **Device Import/Export** — CSV import (upsert on serial number) and export for the device registry
- **Campaign SSE Stream** — real-time Server-Sent Events for firmware campaign progress monitoring
- **Campaign Device Marking** — mark individual devices as updated or failed during rollout
- **Campaign Device Listing** — view all devices enrolled in a campaign with status
- **IPN Autocomplete** — suggested IPNs when entering part numbers in inventory and other modules
- **Dark Mode** — toggle with localStorage persistence
- **Keyboard Shortcuts** — `/` or `Ctrl+K` for search, `n` for new record, `Escape` to close, `?` for help

### Changed
- All API endpoints now require authentication (session cookie or Bearer token), except `/auth/*`, `/static/*`, `/files/*`
- API responses include 401/403 status codes for auth failures
- CORS headers include Authorization in allowed headers

## [0.1.0] - 2026-02-18

### Added
- Single-binary Go server with embedded SQLite (WAL mode)
- SPA frontend with Tailwind CSS and hash-based routing
- **Dashboard** with 8 KPI cards (open ECOs, low stock, active WOs, open NCRs/RMAs, etc.)
- **Parts (PLM)** — read-only browsing of gitplm CSV files with category filtering, full-text search, pagination
- **ECOs** — engineering change order lifecycle (draft → review → approved → implemented)
- **Documents** — revision-controlled documents with approval workflow
- **Inventory** — per-IPN stock tracking with reorder points, locations, and full transaction history
- **Purchase Orders** — PO lifecycle with line items and partial receiving (auto-updates inventory)
- **Vendors** — supplier directory with contacts, lead times, and status
- **Work Orders** — production tracking with BOM availability checks
- **Test Records** — factory test results with serial numbers, measurements, and pass/fail
- **NCRs** — non-conformance reports with defect classification, root cause, and corrective actions
- **Device Registry** — field device tracking with firmware versions, customers, and history
- **Firmware Campaigns** — OTA rollout management with per-device progress tracking
- **RMAs** — return processing from complaint through resolution
- **Quotes** — customer quotes with line items and cost rollup
- CORS support for cross-origin API access
- Request logging middleware
- Seed data for all modules (demo-ready out of the box)
- Auto-generated IDs with year prefix (ECO-2026-001, PO-2026-0001, etc.)
