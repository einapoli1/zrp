# ZRP Frontend - Quick Wins Checklist

**Total Time Estimate**: 6.5 hours  
**Impact**: Improves consistency across 40+ pages  
**Difficulty**: Low (copy-paste existing patterns)

---

## 1. Replace Inline Empty States (90 min)

### Files to Update (6 files)

#### NCRs.tsx
**Line 146-150** - Replace with:
```tsx
{ncrs.length === 0 && (
  <EmptyState
    icon={AlertTriangle}
    title="No NCRs found"
    description="Create your first Non-Conformance Report to track quality issues"
    action={
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create NCR
        </Button>
      </DialogTrigger>
    }
  />
)}
```

#### Quotes.tsx
**Line ~165** - Replace table empty row with:
```tsx
{quotes.length === 0 && (
  <EmptyState
    icon={FileText}
    title="No quotes found"
    description="Create your first quote to send to customers"
    action={
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Quote
        </Button>
      </DialogTrigger>
    }
  />
)}
```

#### RMAs.tsx
**Line ~165** - Replace with:
```tsx
{rmas.length === 0 && (
  <EmptyState
    icon={RotateCcw}
    title="No RMAs found"
    description="Create an RMA to manage device returns and warranty claims"
    action={
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create RMA
        </Button>
      </DialogTrigger>
    }
  />
)}
```

#### Firmware.tsx
**Line ~220** - Replace with:
```tsx
{campaigns.length === 0 && (
  <EmptyState
    icon={Cpu}
    title="No firmware campaigns"
    description="Create a campaign to update firmware across your device fleet"
    action={
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Campaign
        </Button>
      </DialogTrigger>
    }
  />
)}
```

#### Devices.tsx
**Line ~300** - Replace with:
```tsx
{devices.length === 0 && (
  <EmptyState
    icon={Smartphone}
    title="No devices registered"
    description="Import devices from CSV or create them individually"
    action={
      <div className="flex gap-2">
        <DialogTrigger asChild>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Device
          </Button>
        </DialogTrigger>
        <Button variant="outline" onClick={() => setImportDialogOpen(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Import CSV
        </Button>
      </div>
    }
  />
)}
```

#### SalesOrders.tsx
**Line 75-79** - Replace with:
```tsx
{orders.length === 0 && (
  <EmptyState
    icon={ShoppingCart}
    title="No sales orders found"
    description="Sales orders are created when quotes are accepted"
  />
)}
```

---

## 2. Replace Custom Loading Spinners (60 min)

### Files to Update (6 files)

#### Replace this pattern:
```tsx
if (loading) {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
        <p className="mt-2 text-muted-foreground">Loading...</p>
      </div>
    </div>
  );
}
```

#### With this:
```tsx
if (loading) {
  return <LoadingState variant="spinner" message="Loading [resource name]..." />;
}
```

**Files**:
1. NCRs.tsx (line ~77)
2. Quotes.tsx (line ~125)
3. RMAs.tsx (line ~67)
4. Firmware.tsx (line ~115)
5. Devices.tsx (line ~150)
6. SalesOrders.tsx (line ~43 - change to LoadingState)

**Don't forget to add import**:
```tsx
import { LoadingState } from "../components/LoadingState";
```

---

## 3. Add Export Buttons (80 min)

### Pattern to Copy (from Parts.tsx)
```tsx
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
import { Download } from "lucide-react";

// In page header actions:
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="outline">
      <Download className="h-4 w-4 mr-2" />
      Export
    </Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent>
    <DropdownMenuItem onClick={() => handleExport('csv')}>
      Export as CSV
    </DropdownMenuItem>
    <DropdownMenuItem onClick={() => handleExport('xlsx')}>
      Export as Excel
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>

// Handler function:
const handleExport = (format: 'csv' | 'xlsx') => {
  const params = new URLSearchParams();
  params.set('format', format);
  // Add any active filters
  if (searchQuery) params.set('search', searchQuery);
  if (statusFilter !== 'all') params.set('status', statusFilter);
  
  window.location.href = `/api/v1/[resource]/export?${params.toString()}`;
  toast.success(`Exporting [resource] as ${format.toUpperCase()}`);
};
```

### Files to Update
1. **NCRs.tsx** - Add export dropdown next to "Create NCR"
2. **RMAs.tsx** - Add export dropdown next to "Create RMA"
3. **Quotes.tsx** - Add export dropdown next to "Create Quote"
4. **Firmware.tsx** - Add export dropdown next to "Create Campaign"

---

## 4. Add Required Field Indicators (75 min)

### Pattern
Replace:
```tsx
<Label htmlFor="field">Field Name</Label>
```

With:
```tsx
<Label htmlFor="field">Field Name {required && <span className="text-destructive">*</span>}</Label>
```

Or simpler:
```tsx
<Label htmlFor="field">Field Name *</Label>
```

### Files with Forms (15 forms × 5 min each)
1. Parts.tsx - Create Part dialog
2. Inventory.tsx - Quick Receive dialog
3. ECOs.tsx - Create ECO dialog
4. NCRs.tsx - Create NCR form
5. WorkOrders.tsx - Create WO form
6. Quotes.tsx - Create Quote form
7. RMAs.tsx - Create RMA form
8. Devices.tsx - Create Device form
9. Firmware.tsx - Create Campaign form
10. Vendors.tsx - Create Vendor form
11. Procurement.tsx - Create PO form
12. RFQs.tsx - Create RFQ form
13. SalesOrders.tsx - Convert quote form (if exists)
14. Shipments.tsx - Create shipment form
15. Documents.tsx - Upload document form

### Required Fields to Mark
- **IPN fields** - always required
- **Title/Name fields** - always required
- **Quantity fields** - required for transactions
- **Status/Category selects** - usually required
- **Customer/Vendor fields** - required for orders

---

## 5. Fix TypeScript Errors (90 min)

### AdvancedSearch.tsx (5 errors)
**Lines 122, 131, 140, 215, 228**

Issues:
- `api.get()` doesn't exist - replace with proper API method
- `api.post()` doesn't exist - replace with proper API method
- Unused variable `getFieldType`

Fixes:
```tsx
// Replace api.get() with:
const data = await api.getCategories(); // or appropriate method

// Replace api.post() with:
const result = await api.createSavedSearch(searchData);

// Remove unused variable:
// Delete line with: const getFieldType = ...
```

### Audit.tsx (9 errors)
**Lines 80, 99, 101-103, 252, 254, 269**

Issues:
- Possible `undefined` values not handled
- Array type mismatch

Fixes:
```tsx
// Line 80: Add filter
setSelectedUsers(logs.map(log => log.user).filter((u): u is string => u !== undefined));

// Lines 99-103: Add undefined checks
if (log.user) userSet.add(log.user);
if (log.entity_type) typeSet.add(log.entity_type);
if (log.entity_id) idSet.add(log.entity_id);
if (log.details) detailSet.add(log.details);

// Line 252: Add undefined check
{log.timestamp ? new Date(log.timestamp) : new Date()}

// Line 254: Add undefined check
{log.entity_type || 'Unknown'}

// Line 269: Add undefined check
{log.entity_type || 'Unknown'}
```

### Backups.tsx (1 error)
**Line 6**: Remove unused import
```tsx
// Remove: import { Label } from ...
```

### DocumentDetail.tsx (2 errors)
**Line 8**: Remove unused import
```tsx
// Remove: import { Label } from ...
```

**Line 348**: Import missing component
```tsx
import { Skeleton } from "../components/ui/skeleton";
```

### RFQDetail.tsx (1 error)
**Line 15**: Remove unused import
```tsx
// Remove: import { FormField } from ...
```

---

## Checklist Progress

### Empty States (6 files)
- [ ] NCRs.tsx
- [ ] Quotes.tsx
- [ ] RMAs.tsx
- [ ] Firmware.tsx
- [ ] Devices.tsx
- [ ] SalesOrders.tsx

### Loading States (6 files)
- [ ] NCRs.tsx
- [ ] Quotes.tsx
- [ ] RMAs.tsx
- [ ] Firmware.tsx
- [ ] Devices.tsx
- [ ] SalesOrders.tsx

### Export Buttons (4 files)
- [ ] NCRs.tsx
- [ ] RMAs.tsx
- [ ] Quotes.tsx
- [ ] Firmware.tsx

### Required Indicators (15 files)
- [ ] Parts.tsx
- [ ] Inventory.tsx
- [ ] ECOs.tsx
- [ ] NCRs.tsx
- [ ] WorkOrders.tsx
- [ ] Quotes.tsx
- [ ] RMAs.tsx
- [ ] Devices.tsx
- [ ] Firmware.tsx
- [ ] Vendors.tsx
- [ ] Procurement.tsx
- [ ] RFQs.tsx
- [ ] SalesOrders.tsx
- [ ] Shipments.tsx
- [ ] Documents.tsx

### TypeScript Errors (5 files)
- [ ] AdvancedSearch.tsx (6 errors)
- [ ] Audit.tsx (9 errors)
- [ ] Backups.tsx (1 error)
- [ ] DocumentDetail.tsx (2 errors)
- [ ] RFQDetail.tsx (1 error)

---

## Verification

After completing all quick wins, verify:

```bash
# 1. TypeScript compiles without errors
npm run build

# 2. No unused imports
npm run lint

# 3. Test in browser
npm run dev

# 4. Check key pages:
- /parts (empty state when no parts)
- /inventory (loading state on initial load)
- /ecos (export button visible)
- /ncrs (create dialog has required indicators)
```

---

## Git Commit Messages

Suggested commit structure:

```bash
git commit -m "refactor: standardize empty states across 6 pages

- Replace inline empty state markup with EmptyState component
- Add action buttons to empty states
- Improve consistency and accessibility

Pages updated: NCRs, Quotes, RMAs, Firmware, Devices, SalesOrders"
```

```bash
git commit -m "refactor: replace custom loading spinners with LoadingState

- Use shared LoadingState component for consistency
- Remove duplicate spinner implementations
- Standardize loading messages

Pages updated: NCRs, Quotes, RMAs, Firmware, Devices, SalesOrders"
```

```bash
git commit -m "feat: add export functionality to 4 pages

- Add CSV/Excel export dropdown to NCRs, RMAs, Quotes, Firmware
- Match export pattern from Parts and Inventory pages
- Improve feature parity across modules"
```

```bash
git commit -m "feat: add required field indicators to all forms

- Mark required fields with asterisk (*)
- Improve form accessibility and UX
- Consistent with web standards

Forms updated: 15 across all modules"
```

```bash
git commit -m "fix: resolve TypeScript errors

- Fix undefined checks in Audit.tsx
- Update AdvancedSearch API calls
- Remove unused imports
- Fix type mismatches

Files: AdvancedSearch.tsx, Audit.tsx, Backups.tsx, DocumentDetail.tsx, RFQDetail.tsx"
```

---

**End of Quick Wins Guide**
