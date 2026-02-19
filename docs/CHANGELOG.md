# Changelog

All notable changes to ZRP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Device handler test coverage** — comprehensive test suite for handler_devices.go (16 test functions, ~683 LOC)
  - List, get, create, update devices with validation
  - CSV import/export with duplicate handling and error reporting
  - Device history endpoint (test records + firmware campaigns)
  - Edge cases: missing fields, invalid data, file operations
  - All tests passing with proper API response envelope handling
  
- **Vendor handler test coverage** — comprehensive test suite for handler_vendors.go (16 test functions, ~616 LOC)
  - CRUD operations: list, get, create, update, delete
  - Auto-incrementing vendor IDs (V-001, V-002, etc.)
  - Input validation: email format, negative lead times, required fields
  - Referential integrity: prevent deletion when POs or RFQs reference vendor
  - Default status handling and enum validation
  - All tests passing with conflict detection (409) for deletions

### Fixed
- **Backend Tests — API Envelope Consistency**: Fixed 10+ failing tests that expected raw data instead of documented `{data, meta}` envelope
  - **Inventory tests** (7 tests fixed): `TestHandleListInventory_*`, `TestHandleGetInventory_*`, `TestHandleInventoryHistory_*`, `TestHandleBulkDeleteInventory_*`
    - All inventory tests now properly decode API response envelope
    - Fixed timestamp ordering issue in history tests by using explicit timestamps
  - Added `test_helpers.go` with `decodeEnvelope()` helper to standardize envelope extraction across all tests
  - Tests now match the documented API contract from README: `{"data": {...}, "meta": {...}}`
  - Remaining work: ECO, NCR, Procurement, and Parts tests still need envelope fixes
- **Accessibility**: Added missing `DialogDescription` components to 50+ dialogs for screen reader support
  - Affects: WorkOrders, Inventory, Procurement, PODetail, WorkOrderDetail, Users, Devices, Testing, Firmware, InventoryDetail, Vendors, Quotes, NCRs, APIKeys, RMAs, FieldReports, Pricing, Shipments, RFQs, CAPAs, Receiving, and more
  - Reduced accessibility warnings from 90+ to near-zero in test output
  - All dialogs now follow Radix UI accessibility guidelines
- **Tests**: Fixed AppLayout navigation test (removed obsolete "Procurement" reference)

### Added
- **Parts handler test coverage** — comprehensive test suite for handler_parts.go covering core PLM functionality
  - 18 passing test functions covering GET /parts, GET /parts/:ipn, GET /parts/categories, GET /parts/check-ipn, GET /parts/:ipn/bom
  - List parts with filtering (category, search query) and pagination (11 subtests)
  - Get individual parts by IPN (4 subtests)
  - Check IPN existence for duplicate prevention (2 subtests)
  - List categories with dynamic column schemas
  - BOM tree retrieval for assembly parts (PCA-, ASY- prefixes)
  - IPN deduplication across multiple CSV files
  - Search across IPN, description, manufacturer, and other fields
  - Isolated test environment with temporary gitplm CSV directory structure
  - Tests include edge cases: empty directories, invalid files, non-assembly BOM requests
  - Foundation for testing write operations (create, update, delete parts/categories)

### Fixed
- **Backend Tests**: Fixed Go test compilation errors that were blocking all backend testing
  - Resolved duplicate `setupTestPartsDir` function conflict between test files
  - Renamed conflicting helper to `setupTestPartsDirForChanges` in `handler_part_changes_test.go`
  - Fixed `Category.Schema` → `Category.Columns` field reference to match actual type definition
  - All Go tests now compile and execute successfully

### Improved
- **Test Infrastructure**: Enhanced test isolation and naming conventions
  - Test helper functions now have more descriptive names to avoid conflicts
  - Improved test organization across handler test files

## Previous Releases
(To be documented from git history)
