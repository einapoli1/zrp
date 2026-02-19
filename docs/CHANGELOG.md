# Changelog

All notable changes to ZRP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
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
