# ZRP Test Coverage Gap Analysis — Master List

**Current: 670 passing tests across 32 files**
**Generated: 2026-02-18**

---

## 🐛 ACTUAL BUGS FOUND (not just test gaps)

| # | Component | Bug | Severity |
|---|-----------|-----|----------|
| 1 | **Firmware.tsx** | Pause/Play buttons in campaign table rows have NO onClick handlers — buttons do nothing | Critical |
| 2 | **FirmwareDetail.tsx** | Pause Campaign, Start Campaign, Retry Failed buttons have NO onClick handlers | Critical |
| 3 | **FirmwareDetail.tsx** | Polling useEffect has stale closure — `campaign` in interval callback is always from previous render | Critical |
| 4 | **Firmware.tsx** | Pause/Play buttons lack `e.stopPropagation()` — clicking them also triggers row navigation | Medium |
| 5 | **AppLayout.tsx** | Command palette `onSelect` has TODO comment — selecting an item does nothing | Medium |
| 6 | **4 pages use hardcoded mock data** | Users, APIKeys, EmailSettings, Audit don't call the API — they have internal mocks instead of using `api.*` methods | Architectural |

**Recommendation: Fix all 6 bugs before writing more tests.**

---

## 🔴 CRITICAL GAPS (56 total) — SHOULD ADD

These are tests for core user workflows that have zero coverage.

### API & Infrastructure (4)
| Component | Gap | Verdict |
|-----------|-----|---------|
| ApiClient (`src/lib/api.ts`) | ~50 API methods, request helper, error handling — zero direct tests | **ADD** — foundational layer |
| ApiClient | Query parameter construction | **ADD** |
| Router (App.tsx) | No integration test verifying routes render correct pages | **ADD** |
| Auth/permissions | No tests verify role-based access anywhere | **ADD** — after RBAC is enforced |

### Form Submissions Never Exercised (12)
| Component | Gap | Verdict |
|-----------|-----|---------|
| ECOs.tsx | Create form submission — `api.createECO` never called with payload | **ADD** |
| ECOs.tsx | Create form validation — required fields not tested | **ADD** |
| Procurement.tsx | Create PO — fill vendor + line items and submit | **ADD** |
| WorkOrderDetail.tsx | Change Status — select status and click Update | **ADD** |
| WorkOrderDetail.tsx | Generate PO — select vendor and submit | **ADD** |
| PODetail.tsx | Receive Items — enter quantities and submit | **ADD** |
| PODetail.tsx | Change Status — select and submit | **ADD** |
| Users.tsx | Create user end-to-end | **ADD** |
| Users.tsx | Edit user end-to-end | **ADD** |
| Testing.tsx | Create test record end-to-end | **ADD** |
| Documents.tsx | Create document form submission | **ADD** |
| Documents.tsx | Form validation (required fields) | **ADD** |

### Feature Workflows Untested (14)
| Component | Gap | Verdict |
|-----------|-----|---------|
| Inventory.tsx | Checkbox select/deselect individual items | **ADD** |
| Inventory.tsx | Select all checkbox | **ADD** |
| Inventory.tsx | Delete Selected button appears after selection | **ADD** |
| Inventory.tsx | Bulk delete calls API with confirm dialog | **ADD** |
| Devices.tsx | Import CSV dialog — open, select file, click Import, see results | **ADD** |
| Devices.tsx | Import error handling — error result display | **ADD** |
| Documents.tsx | File upload via drag-and-drop | **ADD** |
| Documents.tsx | File upload via file input | **ADD** |
| Documents.tsx | `handleFileUpload` execution — actual upload flow | **ADD** |
| NCRDetail.tsx | Create ECO checkbox (NCR→ECO workflow) | **ADD** |
| NCRDetail.tsx | `create_eco` sent in update payload | **ADD** |
| EmailSettings.tsx | Disable email notifications hides sections | **ADD** |
| EmailSettings.tsx | SMTP preset application (Gmail/Outlook/SendGrid auto-fill) | **ADD** |
| EmailSettings.tsx | Save settings flow | **ADD** |

### Error Handling (10)
| Component | Gap | Verdict |
|-----------|-----|---------|
| Dashboard.tsx | No error UI for users on API failure | **ADD** (also a UX bug) |
| Users.tsx | Error handling for fetch/create/edit | **ADD** |
| APIKeys.tsx | Error handling for fetch/create/revoke | **ADD** |
| EmailSettings.tsx | Error handling for save and test email | **ADD** |
| Quotes.tsx | API fetch error + create error | **ADD** |
| QuoteDetail.tsx | Fetch error + save error + PDF export error | **ADD** |
| RMAs.tsx | Fetch error + create error | **ADD** |
| RMADetail.tsx | Fetch error + save error | **ADD** |
| Audit.tsx | Fetch error | **ADD** |
| Testing.tsx | Fetch error + create error | **ADD** |

### Navigation/Interaction (8)
| Component | Gap | Verdict |
|-----------|-----|---------|
| Parts.tsx | Row click navigation | **ADD** |
| Parts.tsx | Pagination button clicks (Next/Previous) | **ADD** |
| AppLayout.tsx | Active nav link highlighting at different routes | **ADD** |
| AppLayout.tsx | Navigation links actually navigate | **ADD** |
| Audit.tsx | Entity type filter | **ADD** |
| Audit.tsx | User filter | **ADD** |
| Audit.tsx | Pagination (>20 entries) | **ADD** |
| Vendors.tsx | Edit vendor flow (open, pre-populate, submit) | **ADD** |

### Other Critical (8)
| Component | Gap | Verdict |
|-----------|-----|---------|
| Vendors.tsx | Edit dialog form pre-population | **ADD** |
| VendorDetail.tsx | Purchase Orders tab click & table rendering | **ADD** |
| VendorDetail.tsx | PO table row content (links, badges, totals) | **ADD** |
| Dashboard.tsx | 30-second auto-refresh interval + cleanup on unmount | **ADD** |
| InventoryDetail.tsx | Transaction type selection (only "receive" tested) | **ADD** |
| PartDetail.tsx | BOM part click navigates | **ADD** |
| EmailSettings.tsx | Send test email flow + result display | **ADD** |
| Testing.tsx | Form validation (required fields) | **ADD** |

---

## 🟡 MEDIUM GAPS (99 total) — PRIORITIZED

### Should Add (high-value medium) — 35
| Component | Gap |
|-----------|-----|
| Parts.tsx | Category filter calls API with category param |
| Parts.tsx | Reset button clears all filters |
| Parts.tsx | Create part error handling (dialog stays open) |
| Parts.tsx | Create part with all fields (numeric parsing) |
| Parts.tsx | `displayParts` field extraction fallback logic |
| PartDetail.tsx | BOM tree expand/collapse |
| PartDetail.tsx | ASY- prefix also triggers BOM fetch |
| PartDetail.tsx | Non-assembly IPN does NOT show BOM section |
| PartDetail.tsx | `fetchCost` error handling |
| PartDetail.tsx | IPN URL decoding for special chars |
| Inventory.tsx | Quick Receive IPN autocomplete dropdown |
| Inventory.tsx | Quick Receive selecting from autocomplete |
| Inventory.tsx | Quick Receive disabled state |
| Inventory.tsx | Low stock highlighting (bg-red-50 class) |
| Inventory.tsx | Available qty calculation correctness |
| Inventory.tsx | Dropdown menu per row (View Details, Quick Receive) |
| ECOs.tsx | Create form error handling |
| NCRs.tsx | Create form error handling |
| NCRs.tsx | Dialog closes and form resets after create |
| NCRs.tsx | New NCR prepended to list |
| NCRDetail.tsx | Edit mode: severity/status select interaction |
| NCRDetail.tsx | Save error handling |
| NCRDetail.tsx | Checkbox visibility conditions |
| WorkOrders.tsx | Priority select interaction in create form |
| WorkOrders.tsx | Quantity field edge cases (0, negative, NaN) |
| WorkOrderDetail.tsx | Status change error path |
| WorkOrderDetail.tsx | Generate PO error path |
| WorkOrderDetail.tsx | "cancelled" status — no Change Status button |
| Procurement.tsx | Line item IPN autocomplete |
| Procurement.tsx | Line item field updates |
| Procurement.tsx | Remove line item click |
| PODetail.tsx | Receive Items button disabled with no qty |
| PODetail.tsx | "partial" status — Receive Items shown |
| Calendar.tsx | Today's date highlighting |
| Calendar.tsx | More than 2 events overflow indicator |

### Nice to Have (lower-value medium) — 64
Deferred — these cover form resets, cancel buttons, badge variants, conditional renders, loading states, and edge cases that are less likely to catch real bugs.

---

## 🟢 LOW GAPS (98 total) — SKIP

These are CSS class assertions, skeleton loaders, fallback text for null fields, uncommon status variants, date formatting, and other low-ROI items. Not worth the maintenance cost.

---

## SUMMARY

| Category | Count | Action |
|----------|-------|--------|
| Bugs to fix | 6 | Fix immediately |
| Critical test gaps | 56 | Add all |
| High-value medium gaps | 35 | Add most |
| Lower-value medium gaps | 64 | Cherry-pick |
| Low gaps | 98 | Skip |
| **Estimated new tests needed** | **~150-200** | To reach ~850-870 total |

### Recommended Order of Attack
1. **Fix the 6 bugs first** (firmware buttons, command palette, stale closure, hardcoded mocks)
2. **API client tests** — foundational, catches integration issues
3. **Form submission tests** — 12 core workflows never exercised end-to-end
4. **Feature workflow tests** — bulk operations, CSV import, file upload, NCR→ECO
5. **Error handling tests** — universal gap across all components
6. **Navigation/interaction tests** — row clicks, pagination, filters
7. **High-value medium gaps** — as time allows
