# ZRP UI Guidelines

## Design System

ZRP uses **shadcn/ui** components with **Tailwind CSS** and supports **light/dark mode**.

---

## Color Usage

### ✅ Do: Use CSS Variables (dark-mode safe)
```tsx
className="text-foreground"           // Primary text
className="text-muted-foreground"     // Secondary text
className="bg-background"             // Page background
className="bg-card"                   // Card background
className="bg-muted"                  // Subtle background
className="bg-destructive/5"          // Light destructive tint (e.g., low stock rows)
className="text-destructive"          // Error/danger text
className="border-border"             // Default border (automatic)
```

### ❌ Don't: Use hardcoded Tailwind colors in UI
```tsx
// BAD — breaks in dark mode:
className="bg-red-50 text-gray-900 border-gray-200"

// OK — only in semantic badges/status indicators:
className="bg-red-500"  // Status dot color (same in both modes)
```

---

## Page Layout Pattern

Every page follows this structure:

```tsx
<div className="space-y-6">
  {/* Header: title + description + primary action */}
  <div className="flex justify-between items-start">
    <div>
      <h1 className="text-3xl font-bold tracking-tight">Page Title</h1>
      <p className="text-muted-foreground">Brief description.</p>
    </div>
    <Button><Plus className="h-4 w-4 mr-2" />Create</Button>
  </div>

  {/* Optional: Summary cards */}
  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">...</div>

  {/* Main content */}
  <Card>
    <CardHeader><CardTitle>Section</CardTitle></CardHeader>
    <CardContent>...</CardContent>
  </Card>
</div>
```

---

## Loading States

### Initial page load
Use `<TablePageSkeleton />` or `<DetailPageSkeleton />` from `components/PageSkeleton.tsx`:

```tsx
import { TablePageSkeleton } from "../components/PageSkeleton";

if (loading) return <TablePageSkeleton />;
```

### Async operations (buttons)
Disable the button and show loading text:

```tsx
<Button disabled={saving}>
  {saving ? "Saving..." : "Save"}
</Button>
```

### Inline loading
Use `<LoadingSpinner />` from `components/LoadingSpinner.tsx` for centered spinners.

---

## Feedback: Toast Notifications

Always use `toast` from `sonner` for user feedback:

```tsx
import { toast } from "sonner";

// Success
toast.success("Part created successfully");

// Error (catch blocks)
toast.error(error.message || "Failed to create part");

// Info
toast.info("Reset to defaults — click Save to apply");
```

**Rule:** Every `catch` block that affects the user should call `toast.error()`.

---

## Destructive Actions: Confirmation Dialogs

**Never use `window.confirm()`.**

Use `<ConfirmDialog />` from `components/ConfirmDialog.tsx`:

```tsx
import { ConfirmDialog } from "../components/ConfirmDialog";

<ConfirmDialog
  open={deleteConfirmOpen}
  onOpenChange={setDeleteConfirmOpen}
  title="Delete Item"
  description="This action cannot be undone."
  confirmLabel="Delete"
  variant="destructive"
  onConfirm={handleDelete}
/>
```

---

## Empty States

Use `<EmptyState />` from `components/PageSkeleton.tsx` for rich empty states:

```tsx
import { EmptyState } from "../components/PageSkeleton";

<EmptyState
  icon={Package}
  title="No parts found"
  description="Create your first part to get started."
  action={<Button>Create Part</Button>}
/>
```

For tables, pass `emptyMessage` to `<ConfigurableTable />`.

---

## Tables

Prefer `<ConfigurableTable />` for all list pages. It provides:
- Column visibility toggle
- Column reordering & resize
- Sorting
- Persistent column preferences
- Empty state
- Checkbox selection via `leadingColumn`

---

## Keyboard Shortcuts

Use the `useKeyboardShortcuts` hook:

```tsx
import { useKeyboardShortcuts } from "../hooks/useKeyboardShortcuts";

useKeyboardShortcuts([
  { key: "n", handler: () => setCreateDialogOpen(true) },
  { key: "Escape", handler: () => navigate(-1), ignoreInputs: false },
]);
```

Global shortcuts (already handled):
- `⌘K` — Command palette / search
- `Ctrl+Z` — Undo (via global undo hook)

---

## Responsive Design

### Breakpoints
- **375px** — Mobile phone
- **768px** — Tablet
- **1024px** — Desktop

### Patterns
```tsx
// Grid: stack on mobile, spread on desktop
className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"

// Hide on mobile
className="hidden sm:inline"

// Full-width buttons on mobile
className="w-full sm:w-auto"
```

### Sidebar
The sidebar collapses automatically on mobile via `SidebarProvider`. Use `SidebarTrigger` for the hamburger menu.

---

## Button Conventions

| Action | Variant | Example |
|--------|---------|---------|
| Primary action | `default` | Create, Save, Submit |
| Secondary action | `outline` | Cancel, Filter, Export |
| Destructive action | `destructive` | Delete, Remove |
| Icon-only action | `ghost` size=`icon` | Settings gear, notifications |
| Table row action | `ghost` size=`sm` | MoreHorizontal menu |

Always include an icon with text for primary actions:
```tsx
<Button><Plus className="h-4 w-4 mr-2" />Create Part</Button>
```

---

## Form Patterns

- Labels above inputs (not inline)
- Required fields marked with `*` in the label
- Error messages inline below the field: `<p className="text-sm text-destructive mt-1">{error}</p>`
- Dialog forms use `<DialogFooter>` with Cancel (outline) + Submit (default)
- Use `onSubmit` with `<form>` for Enter-to-submit support

---

## Dark Mode

The dark mode toggle is in the sidebar footer. All components **must** work in both modes.

**Checklist:**
- [ ] No hardcoded `bg-white`, `text-gray-*`, `border-gray-*`
- [ ] Use CSS variable-based classes (`bg-background`, `text-foreground`, etc.)
- [ ] Status badge colors (red/green/yellow dots) are intentionally the same in both modes
- [ ] Test by toggling the moon/sun icon in sidebar
