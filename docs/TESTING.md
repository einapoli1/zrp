# ZRP Frontend Testing Documentation

## Overview

- **Test Framework:** Vitest 3.2.4 + React Testing Library
- **E2E Framework:** Playwright 1.58.2
- **Total Vitest Tests:** 1,136 (68 test files)
- **All tests passing:** ✅

## Test Structure

### Unit/Integration Tests (Vitest)

```
frontend/src/
├── pages/          # 58 test files — one per page component
├── layouts/        # AppLayout.test.tsx
├── components/     # BarcodeScanner, BulkEditDialog, ConfigurableTable
├── contexts/       # WebSocketContext
├── hooks/          # useBarcodeScanner, useGitPLM, useUndo, useWebSocket
└── lib/            # api.test.ts
```

### E2E Tests (Playwright)

```
frontend/e2e/
├── app.spec.ts     # Navigation, CRUD, dark mode, global search
└── smoke.spec.ts   # Basic smoke tests
```

## Running Tests

```bash
# All Vitest tests
cd frontend && npx vitest run

# Watch mode
npx vitest

# Single file
npx vitest run src/pages/Parts.test.tsx

# E2E (requires running dev server + backend)
npx playwright test
```

## Test Coverage by Page Component

| Page | Test File | Tests | Key Coverage |
|------|-----------|-------|-------------|
| APIKeys | ✅ | Render, CRUD | |
| Audit | ✅ | Render, filters | |
| Backups | ✅ | Render, operations | |
| CAPADetail | ✅ | Render, status | |
| CAPAs | ✅ | List, filters | |
| Calendar | ✅ | Render, events | |
| Dashboard | ✅ | Stats, charts | |
| DeviceDetail | ✅ | Render, status | |
| Devices | ✅ | List, CRUD | |
| DistributorSettings | ✅ | Load, save Digikey/Mouser, error handling |
| DocumentDetail | ✅ | Load, status, versions | |
| Documents | ✅ | List, filters | |
| ECODetail | ✅ | Render, workflow | |
| ECOs | ✅ | List, create | |
| EmailLog | ✅ | Render | |
| EmailPreferences | ✅ | Render, save | |
| EmailSettings | ✅ | Render, save | |
| FieldReportDetail | ✅ | Load, status changes, error handling |
| FieldReports | ✅ | List, filters | |
| Firmware | ✅ | List, campaigns | |
| FirmwareDetail | ✅ | Render, status | |
| GitDocsSettings | ✅ | Load, save, form fields, error handling |
| GitPLMSettings | ✅ | Render, save | |
| Inventory | ✅ | List, operations | |
| InventoryDetail | ✅ | Render, stock | |
| Login | ✅ | Form, auth | |
| NCRDetail | ✅ | Render, workflow | |
| NCRs | ✅ | List, filters | |
| NotificationPreferences | ✅ | Render, save | |
| PODetail | ✅ | Render, operations | |
| POPrint | ✅ | Render, print | |
| PartDetail | ✅ | Render, BOM, cost | |
| Parts | ✅ | List, CRUD, filters, categories | |
| Permissions | ✅ | Load roles, module display |
| Pricing | ✅ | Render | |
| Procurement | ✅ | Render | |
| QuoteDetail | ✅ | Render, operations | |
| Quotes | ✅ | List, CRUD | |
| RFQDetail | ✅ | Load, status, compare |
| RFQs | ✅ | List, create, empty state, status badges |
| RMADetail | ✅ | Render, workflow | |
| RMAs | ✅ | List, filters | |
| Receiving | ✅ | Render | |
| Reports | ✅ | Render | |
| Scan | ✅ | Render | |
| Settings | ✅ | Render, save | |
| ShipmentDetail | ✅ | Render, operations | |
| ShipmentPrint | ✅ | Render, print | |
| Shipments | ✅ | List, filters | |
| Testing | ✅ | Render | |
| UndoHistory | ✅ | Render | |
| Users | ✅ | List, CRUD | |
| VendorDetail | ✅ | Render, operations | |
| Vendors | ✅ | List, CRUD, filters | |
| WorkOrderDetail | ✅ | Render, workflow | |
| WorkOrderPrint | ✅ | Render, print | |
| WorkOrders | ✅ | List, CRUD | |

## Testing Patterns

### Mock Setup (standard pattern)

```tsx
// 1. Define mock functions at top level (no imports!)
const mockGetItems = vi.fn();

// 2. vi.mock with wrapper functions (hoisted above imports)
vi.mock("../lib/api", () => ({
  api: {
    getItems: (...args: any[]) => mockGetItems(...args),
  },
}));

// 3. Import component AFTER vi.mock
import MyComponent from "./MyComponent";

// 4. Set default mock values in beforeEach
beforeEach(() => {
  vi.clearAllMocks();
  mockGetItems.mockResolvedValue([...]);
});
```

**Important:** `vi.mock` is hoisted to the top of the file. Never reference imported values inside the factory function. Use the wrapper pattern (`(...args) => mockFn(...args)`) instead.

### Common Mocks

- **useIsMobile:** Mock as `() => false` for desktop sidebar rendering
- **PermissionsContext:** Mock `canView: () => true` for full nav access
- **sonner toast:** Mock as `{ success: vi.fn(), error: vi.fn() }`
- **react-router-dom:** Use `vi.importActual` + override `useParams`/`useNavigate`

### Test Utils

Custom render wrapper at `src/test/test-utils.tsx` includes:
- `BrowserRouter` 
- `WebSocketProvider`

Mock data at `src/test/mocks.ts` provides fixtures for all major entities.

## Fixes Applied (2026-02-18)

1. **AppLayout tests (9 failures → 0):** Added mocks for `api.getMe()`, `useIsMobile`, and `PermissionsContext` 
2. **7 missing page test files created:** DistributorSettings, DocumentDetail, FieldReportDetail, GitDocsSettings, Permissions, RFQDetail, RFQs
3. **Test count: 1073 (9 failing) → 1136 (0 failing)**
