# ZRP Workflow Gaps Analysis

> Generated 2026-02-18. Based on source code review of all frontend pages, backend handlers, database schema, and API routes.

---

## Priority Legend
- 🔴 **P0 — Blocker**: Workflow cannot be completed
- 🟠 **P1 — Critical**: Workflow is confusing or loses data
- 🟡 **P2 — Important**: Missing feature that users will expect
- 🟢 **P3 — Nice-to-have**: Polish and convenience

---

## 1. Parts Management

**Workflow: create part → add to inventory → use in BOM → track where-used**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 1.1 | No "Create Part" button on Parts list page | 🟠 P1 | Parts come from gitplm CSV sync only. There's no way to create a part natively in ZRP unless gitplm is configured. Users without gitplm have no path to add parts. |
| 1.2 | Part → Inventory link is manual | 🟠 P1 | Creating a part does NOT auto-create an inventory record. Users must separately go to Inventory and create a record with the same IPN. No guidance or button to do this from PartDetail. |
| 1.3 | No "Add to Inventory" action on PartDetail | 🟡 P2 | PartDetail shows stock info if it exists but has no button to create/adjust inventory. Dead-end if inventory record doesn't exist. |
| 1.4 | BOM only loads for PCA-*/ASY-* prefixed IPNs | 🟡 P2 | Hard-coded prefix check (`upperIPN.startsWith('PCA-') || upperIPN.startsWith('ASY-')`) means custom naming conventions won't show BOMs. Should be configurable or detect BOM existence. |
| 1.5 | No BOM editing in ZRP | 🟠 P1 | BOM is read-only (comes from gitplm). No way to create or edit BOMs within ZRP. |
| 1.6 | Part edit creates "pending changes" requiring ECO | 🟢 P3 | Good for controlled environments, but no way to bypass for non-critical fields (e.g., notes, location). Every edit requires an ECO workflow. |
| 1.7 | No part deletion from UI | 🟡 P2 | DELETE endpoint exists but no button in PartDetail page. |
| 1.8 | Part category/status filters missing on list page | 🟡 P2 | Parts list fetches all parts; need to verify if category/status filters exist in the list view. |
| 1.9 | No link from PartDetail to related ECOs | 🟡 P2 | Where-used shows assemblies, but there's no "ECOs affecting this part" section. |
| 1.10 | No link from PartDetail to related Documents | 🟡 P2 | Documents can reference IPNs but PartDetail doesn't show linked documents. |
| 1.11 | No link from PartDetail to open POs for this part | 🟡 P2 | Users can't see pending orders for a part from its detail page. |

---

## 2. ECO Workflow

**Workflow: create → affect parts → approve → implement → close**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 2.1 | No "close" status or transition | 🟠 P1 | ECO statuses are: draft, open, approved, implemented, rejected. There is no "closed" state. Implemented ECOs just sit there forever. No archival concept. |
| 2.2 | Draft → Open transition not enforced | 🟠 P1 | No explicit "submit for review" action. The `canApprove` check is `eco.status === 'open'`, but there's no UI button to move from draft → open. Users must manually edit the status field or know to use the API. |
| 2.3 | No "reject reason" field | 🟡 P2 | Reject action has no dialog to capture why the ECO was rejected. Critical for audit trail. |
| 2.4 | Implementing ECO doesn't actually change parts | 🟠 P1 | The "Implement ECO" button calls `api.implementECO(id)` but need to verify if part_changes are actually applied to the part CSV/database. If part changes just stay as records, the "implement" action is cosmetic only. |
| 2.5 | No ECO edit capability after creation | 🟡 P2 | ECODetail is read-only (no edit form for title, description, affected_ipns). Only actions are approve/implement/reject. |
| 2.6 | `affected_ipns` is a text field, not a relation | 🟡 P2 | Stored as comma-separated text. No validation that IPNs exist. No structured add/remove interface. |
| 2.7 | No "Reason for Change" field on create | 🟡 P2 | ECODetail renders `eco.reason` but the ECO type doesn't have a `reason` field — it will always be empty. |
| 2.8 | No mandatory fields enforcement | 🟡 P2 | Can create an ECO with empty description/title from the API. No frontend validation visible. |
| 2.9 | No email notification on status changes | 🟡 P2 | Email infrastructure exists but ECO approve/reject/implement don't appear to trigger notifications. |
| 2.10 | No document linking on ECOs | 🟡 P2 | Can't attach supporting documents to an ECO directly (only through the general attachments system). |
| 2.11 | Revision history is append-only with no edit | 🟢 P3 | Revisions can be created but not corrected if entered with errors. |

---

## 3. Procurement (RFQ → PO → Receive → Inspect)

**Workflow: RFQ → compare quotes → award → PO → receive → inspect → to inventory**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 3.1 | RFQ → PO award works, but PO → Receive → Inventory is fragile | 🟠 P1 | PO receive endpoint exists (`/pos/{id}/receive`), but need to verify it creates receiving inspections AND updates inventory. Multiple handoff points with no visual workflow. |
| 3.2 | No "Create PO from RFQ" confirmation with PO details | 🟡 P2 | Award action creates PO silently — user gets an alert() with PO ID but no link to navigate to it. |
| 3.3 | Receiving inspection → inventory flow unclear | 🟠 P1 | After inspection passes, does qty_passed automatically go into inventory? Or is that another manual step? The flow likely requires manual inventory adjustment. |
| 3.4 | No PO line-level receiving | 🟡 P2 | PO receive appears to be at the PO level, not per-line. Can't partially receive individual line items across multiple deliveries. |
| 3.5 | No three-way match (PO vs Receipt vs Invoice) | 🟡 P2 | No invoice concept exists anywhere in the system. |
| 3.6 | RFQ email generation copies to clipboard only | 🟢 P3 | No actual email sending — just generates text. Would need email settings integration to actually send RFQ to vendors. |
| 3.7 | No vendor performance tracking | 🟡 P2 | No metrics on delivery time, quality rate, or price competitiveness across vendors. |
| 3.8 | PO expected_date is a single field | 🟢 P3 | No per-line expected dates. Real POs often have different delivery dates per line. |
| 3.9 | No PO approval workflow | 🟡 P2 | POs go from draft → ordered with no approval step. No spending limits or approval chains. |
| 3.10 | Receiving page has no link back to originating PO | 🟡 P2 | Inspection records reference `po_id` and `po_line_id` but UI may not link back clearly. |
| 3.11 | No reorder automation | 🟡 P2 | Inventory has reorder_point and reorder_qty fields but no automated PO generation when stock hits reorder point. Dashboard shows "low stock" count but no action. |

---

## 4. Manufacturing (Work Orders)

**Workflow: create WO → check BOM/shortages → kit materials → build → test → complete**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 4.1 | No material kitting / reservation step | 🟠 P1 | `qty_reserved` exists in inventory but nothing in the WO flow actually reserves materials. No "Kit Materials" action. BOM check is read-only. |
| 4.2 | No serial number generation during WO | 🟠 P1 | `wo_serials` table exists, but WorkOrderDetail has NO serial management UI. Can't assign, track, or manage serials for units being built. |
| 4.3 | No test integration from WO | 🟡 P2 | No link from WorkOrderDetail to Testing page. Can't record test results for WO serials. No "Test" step in the workflow. |
| 4.4 | Status transitions not enforced | 🟡 P2 | Can jump from "open" directly to "completed" — no requirement to go through in_progress first. |
| 4.5 | Completing WO doesn't update inventory | 🟠 P1 | When WO is completed, finished goods should be added to inventory and consumed materials deducted. This likely doesn't happen automatically. |
| 4.6 | No WO yield/scrap tracking | 🟡 P2 | If 100 units are ordered but only 95 pass, there's no way to record scrap or yield. |
| 4.7 | Generate PO from WO is single-vendor only | 🟢 P3 | The "Generate PO" dialog only allows selecting one vendor for all shortages. In practice, different parts come from different vendors. |
| 4.8 | No work instructions or routing | 🟡 P2 | No way to attach assembly instructions, step-by-step procedures, or routing information to a WO. |
| 4.9 | No time tracking | 🟢 P3 | `started_at` and `completed_at` capture wall-clock time but no labor hours tracking. |
| 4.10 | Print Traveler exists but no customization | 🟢 P3 | WorkOrderPrint page exists but content/format is fixed. |
| 4.11 | No WO-to-WO dependency | 🟢 P3 | Can't express that WO-002 depends on WO-001 completing first. |

---

## 5. Quality (NCR → CAPA)

**Workflow: NCR creation → investigation → CAPA → verification → close**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 5.1 | NCR → CAPA link is URL-based, not a true relation | 🟠 P1 | "Create CAPA from NCR" navigates to `/capas?from_ncr={id}` — this passes data via URL params. If the CAPA list page doesn't parse these params and auto-populate, it's a dead end. |
| 5.2 | NCR has no "created_by" field | 🟡 P2 | NCR table has no `created_by` column. Can't track who reported the non-conformance. |
| 5.3 | NCR defect_type is free-text | 🟡 P2 | Should be a dropdown with configurable defect categories for consistent reporting. |
| 5.4 | CAPA approval is cosmetic | 🟠 P1 | "Approve as QE" just sets `approved_by_qe: "QE Approved"` — hardcoded string, not actual user identity. No RBAC enforcement. Anyone can click either approval button. |
| 5.5 | CAPA status not auto-advanced on approval | 🟡 P2 | Getting both approvals doesn't auto-advance CAPA status. Status must be manually changed. |
| 5.6 | No effectiveness verification deadline | 🟡 P2 | CAPA has `due_date` but no separate verification due date. |
| 5.7 | NCR → ECO creation via URL navigation, not API | 🟡 P2 | "Create ECO from NCR" navigates to `/ecos?from_ncr={id}` — same URL-param pattern. ECO list page may not handle this. |
| 5.8 | No NCR photo/evidence attachment UI | 🟡 P2 | General attachment API exists but NCRDetail has no attachment upload section. Critical for quality evidence. |
| 5.9 | NCR "Create ECO from corrective action" checkbox exists but unclear if it works | 🟡 P2 | The checkbox sets `create_eco: true` in formData, but need to verify the backend handles this flag in the NCR update handler. |
| 5.10 | No NCR metrics/trends dashboard | 🟡 P2 | Reports page has ncr-summary but no Pareto charts, trending, or defect analysis. |
| 5.11 | CAPA has no attachment support visible in UI | 🟡 P2 | Can't attach evidence of corrective action or verification. |

---

## 6. Document Control

**Workflow: create → revise → approve → release → link to parts/ECOs**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 6.1 | No "approved" → "released" distinction in workflow | 🟡 P2 | Document statuses are draft/released/approved/obsolete, but the UI only has a "Release" button on draft docs. No separate approval step before release. |
| 6.2 | Document approve endpoint exists but no UI button | 🟠 P1 | API route `/docs/{id}/approve` exists but DocumentDetail only shows "Release" button for draft status. The approve action is unreachable from the UI. |
| 6.3 | No document-to-part linking UI | 🟡 P2 | Document has an `ipn` field but it's not prominently displayed or linked. No "linked parts" section on DocumentDetail. No "linked documents" on PartDetail. |
| 6.4 | No document-to-ECO linking UI | 🟡 P2 | Document versions have an `eco_id` field, but there's no way to select/link an ECO when creating a version. |
| 6.5 | Content is plain text only | 🟡 P2 | Document content is rendered in `<pre>` tags. No markdown rendering, no rich text editor. Fine for simple docs but limiting. |
| 6.6 | No file upload for documents | 🟡 P2 | Document has a `file_path` field but DocumentDetail shows no file upload/download functionality. Only plain text content. |
| 6.7 | Revision auto-increment not visible | 🟢 P3 | Documents have `revision` field but creating a new version's revision logic is unclear from the frontend. |
| 6.8 | No document templates | 🟢 P3 | Users start from blank. Common templates (test procedure, assembly instruction, etc.) would help. |
| 6.9 | No check-out/check-in or edit locking | 🟡 P2 | Multiple users could edit the same document simultaneously with no conflict resolution. |
| 6.10 | Obsolete documents can't be superseded | 🟢 P3 | No "superseded by" reference when obsoleting a document. |

---

## 7. Device Management

**Workflow: register → assign firmware campaigns → track field reports → RMA**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 7.1 | No link from Device to Field Reports | 🟠 P1 | DeviceDetail shows RMAs but NOT field reports. Field reports have `device_serial` field but no query from device page. |
| 7.2 | No link from Device to Test Records | 🟡 P2 | "View Test History" button navigates to `/testing?device={serial}` but Testing page may not filter by this param. |
| 7.3 | Firmware campaign assignment is indirect | 🟡 P2 | DeviceDetail has "View Firmware Campaigns" link, but no way to see which campaigns target this specific device or enroll it. |
| 7.4 | No device lifecycle tracking | 🟡 P2 | Device status (active/inactive/maintenance/retired) is manual. No automated status changes based on events (e.g., auto-set "maintenance" when RMA is opened). |
| 7.5 | Device → RMA creation via URL params | 🟡 P2 | "Create RMA" navigates to `/rmas?device={serial}`. RMA list page may not auto-populate the serial number from this param. |
| 7.6 | No firmware version history on device | 🟡 P2 | Only shows current firmware_version. No log of previous versions or when upgrades occurred. |
| 7.7 | No device configuration tracking | 🟢 P3 | No way to store per-device configuration, calibration data, or custom settings. |
| 7.8 | Campaign progress/status per-device exists but is buried | 🟢 P3 | `campaign_devices` table tracks per-device status but DeviceDetail doesn't surface this. |
| 7.9 | No device genealogy | 🟢 P3 | Can't trace which WO produced a device, what components were used, etc. |

---

## 8. Quotes / Sales

**Workflow: create quote → send to customer → convert to order → ship → invoice**

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 8.1 | No order concept exists | 🔴 P0 | Quote can be "accepted" but there is NO sales order, work order generation, or fulfillment flow after acceptance. Complete dead end. |
| 8.2 | No "Send to Customer" functionality | 🟠 P1 | Quote can be set to "sent" status manually, but there's no email integration to actually send the quote PDF to a customer. |
| 8.3 | No invoice module | 🔴 P0 | No invoicing capability at all. No invoice table, no invoice generation, no AR tracking. |
| 8.4 | No shipping integration from quotes | 🟠 P1 | Shipments module exists but has no link to quotes. Can't "fulfill" a quote/order by creating a shipment. |
| 8.5 | Quote PDF export may not work | 🟡 P2 | `api.exportQuotePDF(id)` is called but need to verify the backend generates actual PDFs (vs. just returning data). |
| 8.6 | Part cost lookup uses `part.cost` which may be 0 | 🟡 P2 | Margin calculations depend on `getPartCost()` which looks for `part.cost` from the parts list. If parts don't have cost data, margins show as 100%. |
| 8.7 | No customer database | 🟡 P2 | Quote "customer" is a free-text string. No customer master data, contact info, or address management. |
| 8.8 | No quote versioning | 🟡 P2 | Editing a quote overwrites the previous version. No revision history for quotes. |
| 8.9 | No discount/markup capabilities | 🟢 P3 | Line items have unit_price only. No percentage discount, volume pricing, or terms. |
| 8.10 | No quote → WO generation | 🟠 P1 | Accepted quote should trigger WO creation for build-to-order products. No such integration exists. |
| 8.11 | IPN in quote lines is free-text | 🟡 P2 | No autocomplete or validation against actual parts database. |

---

## 9. Cross-Module Integration Gaps

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 9.1 | URL-param based linking pattern is fragile | 🟠 P1 | Multiple workflows pass data via URL query params (NCR→ECO, NCR→CAPA, Device→RMA). Target pages likely don't parse these params, making the "Create X from Y" actions dead ends. |
| 9.2 | No global search results page | 🟡 P2 | Search API exists but results may not link to all entity types effectively. |
| 9.3 | No dashboard customization | 🟢 P3 | Dashboard shows fixed set of counts. No ability to add charts, recent activity, or custom widgets. |
| 9.4 | Attachments not surfaced on detail pages | 🟡 P2 | Attachment API exists (upload, list, download, delete) but most detail pages (NCR, ECO, WO, Device) don't show attachment UI. |
| 9.5 | Calendar integration is basic | 🟢 P3 | Calendar page exists but unclear what events it shows. No PO due dates, WO deadlines, quote expiry dates, or ECO review dates. |
| 9.6 | No activity feed / audit trail on detail pages | 🟡 P2 | Audit log exists as a separate page but individual records don't show their change history inline. |
| 9.7 | Notifications exist but aren't triggered | 🟡 P2 | Notification infrastructure exists (table, API, preferences) but events don't appear to create notifications. |
| 9.8 | No "Related Items" pattern | 🟡 P2 | Each entity exists in isolation. A part should show related ECOs, POs, WOs, NCRs, Documents, etc. Only PartDetail shows where-used. |

---

## 10. UX / UI Gaps

| # | Gap | Priority | Details |
|---|-----|----------|---------|
| 10.1 | alert() used for user feedback | 🟠 P1 | Multiple pages use `alert()` for success/error messages (RFQ award, git push, PR creation). Should use toast notifications. |
| 10.2 | No confirmation dialogs for destructive actions | 🟡 P2 | RFQ delete has `confirm()` but ECO reject has no confirmation. Inconsistent patterns. |
| 10.3 | No loading states on action buttons | 🟡 P2 | Many save/action buttons don't show loading indicators (e.g., FieldReportDetail status changes). |
| 10.4 | No empty state guidance | 🟡 P2 | When a module has no data, most just show "No X found." Should guide users: "Create your first X" with a prominent CTA. |
| 10.5 | Inconsistent edit patterns | 🟡 P2 | Some pages use inline editing (NCR, Device), some use modals (WO status), some use separate edit mode (Quote). Should standardize. |
| 10.6 | No breadcrumb navigation | 🟢 P3 | Only "Back to X" buttons. No hierarchical breadcrumbs showing context. |
| 10.7 | No keyboard shortcuts | 🟢 P3 | No Ctrl+S to save, no keyboard navigation. |
| 10.8 | Error handling is console.error only | 🟠 P1 | Most API errors are caught and logged to console with no user-visible feedback. Users may think actions succeeded when they failed. |

---

## Summary: Top 10 Priorities

1. **🔴 No sales order / fulfillment flow** (8.1) — Quote acceptance is a complete dead end
2. **🔴 No invoicing** (8.3) — Can't bill customers
3. **🟠 ECO implement doesn't apply part changes** (2.4) — Core change management may be cosmetic
4. **🟠 No material kitting/reservation in WOs** (4.1) — Manufacturing can't reserve inventory
5. **🟠 WO completion doesn't update inventory** (4.5) — Finished goods not tracked
6. **🟠 No serial management UI in WOs** (4.2) — Can't track units being built
7. **🟠 Draft→Open ECO transition missing** (2.2) — Can't submit ECOs for review
8. **🟠 URL-param linking is likely broken** (9.1) — Cross-module "create from" actions are dead ends
9. **🟠 No native part creation without gitplm** (1.1) — Standalone users can't add parts
10. **🟠 Error handling shows nothing to users** (10.8) — Silent failures across the app
