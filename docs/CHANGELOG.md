# Changelog

All notable changes to ZRP are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/).

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
