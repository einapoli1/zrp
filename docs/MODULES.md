# ZRP Modules Guide

## Dashboard

The landing page. Shows eight KPI cards:

| Card | What it shows |
|------|---------------|
| Open ECOs | ECOs not yet implemented or rejected |
| Low Stock | Items at or below reorder point |
| Open POs | Purchase orders not received or cancelled |
| Active WOs | Work orders that are open or in progress |
| Open NCRs | Quality issues not yet resolved |
| Open RMAs | Customer returns still being processed |
| Total Parts | Count of all parts in the gitplm database |
| Total Devices | Count of registered field devices |

Click any card to jump to that module.

---

## Parts (PLM)

**What it's for:** Browsing and searching your parts database. Parts are managed externally in gitplm CSV files — ZRP reads them and provides a searchable web interface.

**Key concepts:**
- **IPN** (Internal Part Number) — your unique identifier for each part
- **Category** — groups parts by type (capacitors, resistors, connectors, etc.)
- **Fields** — each category has its own columns from the CSV headers

**Common workflows:**

1. **Find a part:** Type in the search box or filter by category. Search matches against IPN and all field values.
2. **View details:** Click a part row to see all its fields.
3. **Edit a part:** Edit the CSV file directly in the gitplm directory, then refresh ZRP.

**Integration:** Parts IPNs are referenced by Inventory, Work Orders, Purchase Orders, Documents, NCRs, and Quotes.

---

## ECOs (Engineering Change Orders)

**What it's for:** Tracking proposed changes to parts, assemblies, or processes before they're implemented.

**Status workflow:** `draft` → `review` → `approved` → `implemented`

**Common workflows:**

1. **Propose a change:** Create a new ECO with a title, description, priority, and list of affected IPNs.
2. **Submit for review:** Update the status to `review`.
3. **Approve:** Click the approve action. Records who approved and when.
4. **Implement:** After changes are made, mark as implemented.

**Integration:** ECOs reference affected IPNs from the Parts database. Dashboard shows count of open ECOs.

---

## Documents

**What it's for:** Managing revision-controlled engineering documents — assembly procedures, test specs, drawings, process instructions.

**Key concepts:**
- **Revision** — letter revision (A, B, C...) incremented on each release
- **Category** — procedure, spec, drawing, etc.
- **Content** — markdown text stored in the document record

**Common workflows:**

1. **Create a document:** Give it a title, category, optional IPN link, and write the content.
2. **Edit and revise:** Update content and bump the revision letter.
3. **Approve:** Move from draft to approved when reviewed.

---

## Inventory

**What it's for:** Tracking how much stock you have of each part, where it's stored, and when to reorder.

**Key concepts:**
- **Qty On Hand** — physical count in stock
- **Qty Reserved** — allocated to work orders
- **Reorder Point** — when on-hand drops to this level, reorder
- **Transaction History** — every receive, issue, return, and adjustment is logged

**Common workflows:**

1. **Check stock levels:** View the inventory list. Filter by `low_stock` to see items needing reorder.
2. **Receive parts:** When a PO shipment arrives, use the PO receive function (automatically updates inventory).
3. **Issue parts:** Create an inventory transaction of type `issue` when consuming parts for a work order.
4. **Cycle count:** Use `adjust` transaction type to correct counts.

**Integration:** Automatically updated when POs are received. Referenced by Work Order BOM checks.

---

## Purchase Orders

**What it's for:** Ordering parts from vendors and tracking deliveries.

**Status workflow:** `draft` → `sent` → `partial` → `received`

**Key concepts:**
- **Line items** — each PO has one or more lines, each specifying an IPN, MPN, quantity, and price
- **Partial receiving** — you can receive a portion of an order; status auto-updates

**Common workflows:**

1. **Create a PO:** Select a vendor, add line items with part numbers and quantities.
2. **Send to vendor:** Update status to `sent`.
3. **Receive shipment:** Use the receive action with quantities for each line. Inventory updates automatically.

**Integration:** Links to Vendors. Receiving creates Inventory transactions.

---

## Vendors

**What it's for:** Keeping a directory of your suppliers with contact information and lead times.

**Key concepts:**
- **Status** — `active` (current supplier), `preferred` (primary supplier), `inactive` (no longer used)
- **Lead Time** — expected days from order to delivery

**Common workflows:**

1. **Add a vendor:** Enter company name, website, contact details, and expected lead time.
2. **Reference in POs:** When creating purchase orders, select from your vendor list.

---

## Work Orders

**What it's for:** Tracking production runs — building assemblies from components.

**Status workflow:** `open` → `in_progress` → `completed`

**Key concepts:**
- **Assembly IPN** — the product being built
- **Quantity** — how many units to produce
- **BOM** — Bill of Materials showing required components and availability

**Common workflows:**

1. **Plan production:** Create a WO specifying the assembly and quantity.
2. **Check materials:** Use the BOM endpoint to verify all components are in stock.
3. **Start production:** Update status to `in_progress` (auto-records start time).
4. **Complete:** Update status to `completed` (auto-records completion time).

**Integration:** BOM check queries Inventory. Assembly IPN references Parts.

---

## Test Records

**What it's for:** Recording factory test results for individual units identified by serial number.

**Key concepts:**
- **Serial Number** — unique identifier for the unit under test
- **Result** — `pass` or `fail`
- **Measurements** — JSON object with measured values (voltage, current, etc.)
- **Test Type** — factory, burn-in, final, etc.

**Common workflows:**

1. **Record a test:** Submit serial number, IPN, firmware version, result, and measurements.
2. **Review history:** Look up all test records for a serial number to see pass/fail trends.

**Integration:** Referenced by Device history. Serial numbers link to Work Order serials.

---

## NCRs (Non-Conformance Reports)

**What it's for:** Documenting and tracking quality issues — defects found during manufacturing, inspection, or in the field.

**Status workflow:** `open` → `investigating` → `resolved` → `closed`

**Key concepts:**
- **Defect Type** — workmanship, component, design
- **Severity** — minor, major, critical
- **Root Cause** — why the defect occurred
- **Corrective Action** — what was done to prevent recurrence

**Common workflows:**

1. **Report a defect:** Create an NCR with title, description, affected IPN/serial, defect type, and severity.
2. **Investigate:** Update with root cause analysis.
3. **Resolve:** Document the corrective action and mark resolved.

**Integration:** Links to specific IPNs and serial numbers. Dashboard shows open NCR count.

---

## Device Registry

**What it's for:** Tracking deployed devices in the field — which customer has which serial number, what firmware it's running, and where it's located.

**Key concepts:**
- **Status** — `active` (deployed), `inactive`, `rma` (returned), `decommissioned`
- **History** — combined view of test records and firmware campaign participation

**Common workflows:**

1. **Register a device:** After production and testing, register the serial number with customer and location info.
2. **Track firmware:** View current firmware version for each device.
3. **View history:** See full lifecycle — factory tests, firmware updates, campaigns.

**Integration:** Firmware campaigns target active devices. Test records and RMAs reference serial numbers.

---

## Firmware Campaigns

**What it's for:** Managing OTA firmware rollouts to deployed devices.

**Status workflow:** `draft` → `active` → `completed`

**Key concepts:**
- **Campaign** — defines a target firmware version
- **Launch** — enrolls all active devices and begins rollout
- **Progress** — tracks each device: pending, sent, updated, failed

**Common workflows:**

1. **Create a campaign:** Specify the firmware version and any notes.
2. **Launch:** Automatically enrolls all active devices.
3. **Monitor progress:** Check how many devices have updated vs. pending/failed.

**Integration:** Targets devices from the Device Registry. Device history shows campaign participation.

---

## RMAs (Return Merchandise Authorization)

**What it's for:** Processing customer returns — from initial complaint through diagnosis and resolution.

**Status workflow:** `open` → `received` → `diagnosing` → `repaired` → `shipped` → `closed`

**Common workflows:**

1. **Open an RMA:** Record the serial number, customer, reason, and defect description.
2. **Receive unit:** Mark as received when the physical unit arrives.
3. **Diagnose:** Update defect findings.
4. **Resolve:** Document the resolution (repair, replace, refund) and ship back.

**Integration:** Links to Device Registry by serial number. Dashboard shows open RMA count.

---

## Quotes

**What it's for:** Creating and tracking customer quotes with itemized pricing.

**Status workflow:** `draft` → `sent` → `accepted` / `declined` / `expired`

**Key concepts:**
- **Line items** — each line has an IPN, description, quantity, and unit price
- **Cost rollup** — endpoint calculates line totals and grand total

**Common workflows:**

1. **Create a quote:** Select customer, add line items with quantities and prices, set validity date.
2. **Send to customer:** Update status to `sent`.
3. **Track outcome:** Mark as accepted, declined, or expired.

**Integration:** Line items reference IPNs from Parts. Cost rollup provides pricing summary.
