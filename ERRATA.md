# Known Limitations

## Parts are Read-Only
Parts data comes from gitplm CSV files and cannot be created, updated, or deleted through the ZRP API. The POST/PUT/DELETE endpoints for parts return 501. Edit the CSV files directly.

## CSV Re-read on Every Request
Parts and categories are loaded from disk on every API call. For large parts databases (10,000+ parts), this may cause noticeable latency. Caching is planned.

## No Authentication
There is no built-in authentication or authorization. Deploy behind a reverse proxy with auth, or bind to localhost only.

## No File Upload
The Documents module stores markdown content in the database but does not support file uploads. The `file_path` field is for reference only.

## Simplified BOM
The Work Order BOM endpoint returns all tracked inventory items rather than a true bill of materials for the assembly. Proper BOM support requires gitplm BOM data integration.

## No Real Firmware Delivery
Firmware campaigns track status but don't actually push firmware to devices. The campaign launch enrolls devices and sets them to "pending" — actual OTA delivery requires integration with your device management system.

## Single-User Model
All actions are attributed to default users ("engineer", "operator"). There is no multi-user support, user accounts, or audit trail by user.

## No Pagination on Most List Endpoints
Only the Parts endpoint supports pagination (`page` and `limit` parameters). Other list endpoints return all records.

## CORS Wide Open
`Access-Control-Allow-Origin: *` is set on all responses. Restrict this in production via reverse proxy.
