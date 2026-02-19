# ZRP Mobile Responsiveness Audit Report

**Date:** 2026-02-19  
**Auditor:** Eva (AI Assistant)  
**Scope:** Frontend mobile/tablet responsiveness across 59 pages

---

## Executive Summary

ZRP is a desktop-first PLM application with limited mobile optimization. This audit identifies critical issues preventing usable mobile/tablet experiences and provides targeted fixes for the most impactful improvements.

### Audit Methodology
- Reviewed component library (shadcn/ui with Tailwind CSS)
- Analyzed table components (ConfigurableTable, ui/table)
- Examined navigation (AppLayout, Sidebar)
- Reviewed form patterns across key pages
- Identified breakpoint requirements (320px, 375px, 768px, 1024px)

---

## Current State Analysis

### ✅ What's Working
1. **Tailwind CSS Foundation**: Project uses Tailwind CSS with responsive utilities
2. **Sidebar Component**: Has mobile sheet/drawer support via `useIsMobile` hook
3. **shadcn/ui Components**: Modern component library with some mobile patterns
4. **Table Wrapper**: Base table component has `overflow-auto` wrapper

### ❌ Critical Issues Found

#### 1. **Tables - Major Usability Issues**
**Component**: `ConfigurableTable.tsx`, `ui/table.tsx`

**Problems:**
- ✗ No responsive card view for mobile
- ✗ Horizontal scroll works but hard to discover
- ✗ Column resize handles unusable on touch (1px width)
- ✗ Dense content (p-2 padding) too small for touch
- ✗ Settings dropdown tiny (h-6 w-6 button)
- ✗ Sort indicators not touch-friendly

**Impact**: Tables power 30+ pages - this affects parts lists, inventory, procurement, vendors, etc.

#### 2. **Forms - Stacking Issues**
**Examples**: Parts create dialog, vendor forms, ECO forms

**Problems:**
- ✗ Labels/inputs often inline on desktop without mobile override
- ✗ Multi-column layouts don't collapse on mobile
- ✗ Touch targets < 44px for checkboxes/small buttons
- ✗ Dialogs can overflow viewport on small screens

#### 3. **Navigation - Partially Mobile Ready**
**Component**: `AppLayout.tsx`, `ui/sidebar.tsx`

**Problems:**
- ⚠️ Sidebar converts to sheet on mobile (GOOD)
- ✗ Trigger button may be hidden on some layouts
- ✗ Navigation items readable but could be larger touch targets

#### 4. **Typography - Inconsistent Sizing**
**Location**: `index.css`, component classes

**Problems:**
- ✗ Some text as small as `text-sm` (14px) - hard to read on mobile
- ✗ Table cells use default small sizing
- ⚠️ Base font size OK but headings could be optimized

#### 5. **Modals/Dialogs - Overflow Risk**
**Component**: `ui/dialog.tsx`

**Problems:**
- ✗ Large dialogs may not fit on mobile viewport
- ✗ No max-height with scroll handling
- ✗ Action buttons may not stack on narrow screens

---

## Page-Specific Audit

### High-Traffic Pages (Priority 1)

#### Parts List (`/parts`)
- ✗ Table with 10+ columns - very wide
- ✗ Filter dropdowns functional but cramped
- ✗ Search bar OK
- ✗ Create part dialog has multi-field form (needs stacking)

#### Dashboard (`/`)
- ⚠️ Grid layout - needs breakpoint adjustment
- ✗ Cards probably don't stack gracefully
- ✗ Widget content may overflow

#### Inventory (`/inventory`)
- ✗ Similar table issues as Parts
- ✗ Barcode scanner component - needs mobile testing

#### Procurement (`/purchase-orders`)
- ✗ Complex table with inline actions
- ✗ Multi-step creation flow - needs mobile flow

#### Vendors (`/vendors`)
- ✗ Table with contact info columns (wide)
- ✗ Detail page likely has multi-column layout

### Medium Priority

#### ECOs, NCRs, CAPAs, RFQs
- All use ConfigurableTable - inherit same issues
- Form dialogs need individual review

#### Documents, Testing, Devices
- Similar patterns - table + detail views
- Mobile usability tied to table fixes

### Lower Priority

#### Reports, Audit Log
- Read-only views
- Acceptable with horizontal scroll
- Could benefit from simplified mobile layout

---

## Responsive Breakpoints Analysis

### Current Tailwind Config
```js
screens: {
  "2xl": "1400px",
}
```

**Missing breakpoints:** sm (640px), md (768px), lg (1024px), xl (1280px)

**Recommendation:** Use Tailwind defaults + custom 2xl:
```js
screens: {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  '2xl': '1400px',
}
```

---

## Top 10 Critical Fixes (Prioritized)

### 1. **Add Responsive Table Card View** (CRITICAL)
- Create `<ResponsiveTable>` wrapper component
- Desktop: Use existing ConfigurableTable
- Mobile (< 768px): Render as stacked cards
- **Impact**: Fixes 30+ pages immediately

### 2. **Increase Touch Targets** (HIGH)
- Buttons minimum `h-10` (40px) on mobile
- Table row click targets full height
- Checkbox/radio inputs with larger tap area
- **Classes**: Add `md:h-8` overrides where space-constrained

### 3. **Fix Table Settings Dropdown** (HIGH)
- Current: 24x24px button
- Mobile: 44x44px touch target
- **Fix**: `h-6 w-6 md:h-6 md:w-6` → `h-11 w-11 md:h-6 md:w-6`

### 4. **Stack Form Labels Vertically** (HIGH)
- Audit `FormField` component
- Add mobile-first stacking: `flex flex-col md:flex-row`
- **Impact**: All create/edit dialogs

### 5. **Improve Dialog Mobile Behavior** (MEDIUM)
- Add `max-h-[90vh] overflow-y-auto` to dialog content
- Stack action buttons: `flex-col sm:flex-row`
- Increase padding on mobile

### 6. **Add Horizontal Scroll Indicators** (MEDIUM)
- Tables need visual cue for scrollability
- Add fade gradient on edges
- Or add "Scroll →" hint

### 7. **Optimize Mobile Font Sizes** (MEDIUM)
- Ensure minimum 16px for body text (prevents zoom)
- Table cells: `text-sm md:text-sm` → `text-base md:text-sm`
- Buttons: Ensure readable labels

### 8. **Collapsible Sidebar Enhancement** (LOW)
- Already has mobile sheet
- Ensure trigger is always visible
- Add swipe-to-open gesture (if not present)

### 9. **Responsive Grid Layouts** (MEDIUM)
- Dashboard widgets: `grid-cols-1 md:grid-cols-2 lg:grid-cols-3`
- Card grids across app need breakpoints

### 10. **Test Modal Stacking** (LOW)
- Nested dialogs on mobile
- Ensure z-index management
- Prevent body scroll when modal open

---

## Implementation Plan

### Phase 1: Foundation (30 min)
1. Add responsive breakpoints to `tailwind.config.js`
2. Create `hooks/use-mobile.ts` utility (may exist)
3. Update button base styles for touch targets

### Phase 2: Tables (1 hour)
1. Create `ResponsiveTable` wrapper component
2. Implement card view for mobile
3. Update ConfigurableTable touch targets
4. Test on Parts, Inventory pages

### Phase 3: Forms (45 min)
1. Audit and fix `FormField` component
2. Update dialog widths and padding
3. Stack button groups on mobile
4. Test create/edit flows

### Phase 4: Layout & Polish (30 min)
1. Fix dashboard grid
2. Improve font sizing
3. Add scroll indicators
4. Test navigation flow

### Phase 5: Testing & Documentation (30 min)
1. Test at 320px, 375px, 768px, 1024px
2. Document mobile support status
3. Add mobile testing to CI (optional)

**Total Estimated Time:** 3-4 hours

---

## Success Criteria Checklist

- [ ] All tables usable on mobile (scroll or cards)
- [ ] Touch targets minimum 44px
- [ ] Forms stack labels/inputs vertically on mobile
- [ ] Dialogs fit on screen with scroll
- [ ] Navigation accessible on all screen sizes
- [ ] Font sizes readable (16px minimum)
- [ ] Build succeeds, no regressions
- [ ] Top 10 pages tested at all breakpoints

---

## Mobile Testing Strategy

### Manual Testing
1. **Chrome DevTools**: Device toolbar (iPhone SE, iPad)
2. **Responsive Mode**: Drag to test breakpoints
3. **Touch Emulation**: Enable to test touch targets

### Automated Testing (Future)
- Playwright mobile viewports
- Accessibility checks with axe
- Visual regression testing

### Real Device Testing (Recommended)
- iPhone (iOS Safari)
- Android phone (Chrome)
- iPad/tablet
- Test in portrait and landscape

---

## Notes for Future Work

1. **Progressive Enhancement**: Consider offline support for mobile field workers
2. **Barcode Scanner**: Ensure camera permissions work on mobile browsers
3. **Print Styles**: Already have print.css - verify mobile print
4. **Performance**: Code splitting and lazy loading already implemented
5. **PWA**: Consider making ZRP installable for mobile users

---

## Current Build Error

**Issue**: `ConfigurableTable.tsx:282` - TypeScript syntax error
**Status**: Needs investigation - appears to be related to JSX/TSX parsing
**Priority**: CRITICAL - blocks build

**Next Step**: Review line 282 and surrounding code for syntax issues before implementing responsive changes.
