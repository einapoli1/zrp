# Accessibility Guidelines for ZRP Frontend

## Overview

This document provides accessibility (a11y) standards and best practices for ZRP frontend development. All new features and components must meet **WCAG 2.1 Level AA** compliance.

---

## 🎯 Core Principles

1. **Perceivable** - Information must be presentable to users in ways they can perceive
2. **Operable** - Interface components must be operable by all users
3. **Understandable** - Information and operation must be understandable
4. **Robust** - Content must be robust enough to work with assistive technologies

---

## 📋 Checklist for New Components

Before merging any new component or page:

- [ ] All interactive elements are keyboard accessible
- [ ] All form inputs have associated labels
- [ ] All images have alt text (or aria-hidden if decorative)
- [ ] All icon-only buttons have aria-label
- [ ] Color is not the only means of conveying information
- [ ] Focus indicators are visible on all interactive elements
- [ ] Headings follow proper hierarchy (h1 → h2 → h3)
- [ ] Page title updates on navigation
- [ ] Error messages are announced to screen readers
- [ ] Loading states are announced
- [ ] Accessibility tests pass (use `testA11y()` helper)

---

## 🔧 Common Patterns & Solutions

### 1. Form Fields

✅ **Good:** Use the FormField component with proper labels

```tsx
import { FormField } from "@/components/FormField";
import { Input } from "@/components/ui/input";

<FormField 
  label="Part Number" 
  htmlFor="ipn"
  required
  error={errors.ipn?.message}
  description="Format: XXX-NNNNN"
>
  <Input id="ipn" {...register("ipn")} />
</FormField>
```

❌ **Bad:** Input without label association

```tsx
<div>
  <span>Part Number</span>
  <input type="text" />
</div>
```

---

### 2. Icon-Only Buttons

✅ **Good:** Always include aria-label

```tsx
<Button 
  variant="ghost" 
  size="icon" 
  aria-label="Delete item"
>
  <Trash2 className="h-4 w-4" aria-hidden="true" />
</Button>
```

❌ **Bad:** Icon button without label

```tsx
<Button variant="ghost" size="icon">
  <Trash2 className="h-4 w-4" />
</Button>
```

**Pro tip:** Add `aria-hidden="true"` to decorative icons inside labeled buttons.

---

### 3. Links vs Buttons

**Use `<Link>` or `<a>` for navigation:**

```tsx
import { Link } from "react-router-dom";

<Link 
  to="/parts/ABC-123" 
  className="text-blue-600 hover:underline focus-visible:ring-2"
>
  ABC-123
</Link>
```

**Use `<Button>` for actions:**

```tsx
<Button onClick={handleSubmit}>
  Save Changes
</Button>
```

❌ **Never do this:**

```tsx
// DON'T: Non-interactive element with onClick
<span onClick={() => navigate("/parts")}>Parts</span>

// DON'T: Div styled as button
<div className="button" onClick={handleClick}>Click me</div>
```

---

### 4. Page Titles

Use the `usePageTitle` hook in every page component:

```tsx
import { usePageTitle } from "@/hooks/usePageTitle";

function PartsPage() {
  usePageTitle("Parts");
  // ... rest of component
}
```

This:
- Updates `document.title` to "ZRP | Parts"
- Announces page change to screen readers
- Improves navigation history clarity

---

### 5. Dialogs & Modals

Radix UI Dialog components have good built-in accessibility:

```tsx
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

<Dialog open={isOpen} onOpenChange={setIsOpen}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Delete Part</DialogTitle>
      <DialogDescription>
        This action cannot be undone. Are you sure?
      </DialogDescription>
    </DialogHeader>
    {/* Content */}
  </DialogContent>
</Dialog>
```

**Always include:**
- `DialogTitle` (required for screen readers)
- `DialogDescription` (context for the modal)
- Dialog automatically traps focus and returns focus on close

---

### 6. Tables

Use `ConfigurableTable` with proper labels:

```tsx
<ConfigurableTable
  tableName="parts"
  columns={columns}
  data={parts}
  rowKey={(row) => row.ipn}
  ariaLabel="Parts inventory table"
  caption="List of all parts in inventory"
/>
```

For custom tables:

```tsx
<table aria-label="Purchase orders">
  <caption className="sr-only">List of purchase orders</caption>
  <thead>
    <tr>
      <th scope="col">PO Number</th>
      <th scope="col">Vendor</th>
      <th scope="col" aria-sort="ascending">Date</th>
    </tr>
  </thead>
  <tbody>
    {/* rows */}
  </tbody>
</table>
```

---

### 7. Loading States

Always announce loading states:

```tsx
import { LoadingState } from "@/components/LoadingState";

{loading && <LoadingState />}
```

The LoadingState component includes:
- Spinner with `role="status"`
- `aria-live="polite"` region
- Screen reader text: "Loading..."

---

### 8. Error Messages

Use `role="alert"` for important errors:

```tsx
{error && (
  <div className="text-destructive text-sm" role="alert">
    {error}
  </div>
)}
```

For form validation, associate errors with inputs:

```tsx
<Input 
  id="email"
  aria-invalid={!!errors.email}
  aria-describedby={errors.email ? "email-error" : undefined}
/>
{errors.email && (
  <p id="email-error" className="text-destructive" role="alert">
    {errors.email.message}
  </p>
)}
```

---

### 9. Skip Navigation

The main layout includes a skip-to-content link. No action needed in page components.

Users can press Tab immediately after page load to reveal the link.

---

### 10. Keyboard Navigation

All interactive elements must be keyboard accessible:

**Tab order:**
- Should follow visual order
- Skip over non-interactive decorative elements
- Include all buttons, links, form fields

**Custom interactive elements need:**

```tsx
<div
  role="button"
  tabIndex={0}
  onClick={handleClick}
  onKeyDown={(e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      handleClick();
    }
  }}
  aria-label="Action description"
>
  Custom Button
</div>
```

**But prefer native elements when possible!**

---

## 🧪 Testing Accessibility

### Unit Tests

Use the provided a11y testing utilities:

```tsx
import { describe, it } from "vitest";
import { render, testA11y } from "@/test/test-utils";
import "@/test/setup-a11y";

describe("MyComponent", () => {
  it("should not have accessibility violations", async () => {
    const view = render(<MyComponent />);
    await testA11y(view);
  });
});
```

### Manual Testing

**Keyboard navigation:**
1. Tab through all interactive elements
2. Verify focus indicators are visible
3. Test Enter/Space on custom buttons
4. Ensure focus doesn't get trapped (except modals)

**Screen reader:**
- macOS: VoiceOver (Cmd + F5)
- Windows: NVDA (free) or JAWS
- Verify all content is announced properly

**Browser DevTools:**
- Chrome: Install [axe DevTools](https://www.deque.com/axe/devtools/)
- Run automated scan on each page
- Fix all "Critical" and "Serious" issues

---

## 🎨 Color Contrast

All text must meet WCAG AA contrast ratios:

- **Normal text:** 4.5:1 minimum
- **Large text (18pt+):** 3:1 minimum
- **UI components:** 3:1 minimum

**Tailwind classes that meet contrast:**

```tsx
// Good contrast pairs
className="bg-primary text-primary-foreground"
className="bg-destructive text-destructive-foreground"
className="bg-muted text-foreground"

// Check custom colors with:
// https://webaim.org/resources/contrastchecker/
```

---

## 🚫 Common Mistakes

### 1. `onClick` on Non-Interactive Elements

```tsx
// ❌ BAD
<div onClick={handleClick}>Click me</div>
<span onClick={handleClick}>Link</span>

// ✅ GOOD
<button onClick={handleClick}>Click me</button>
<Link to="/page">Link</Link>
```

### 2. Missing Form Labels

```tsx
// ❌ BAD
<input type="text" placeholder="Email" />

// ✅ GOOD
<label htmlFor="email">Email</label>
<input id="email" type="text" />
```

### 3. Icon-Only Buttons Without Labels

```tsx
// ❌ BAD
<Button><X /></Button>

// ✅ GOOD
<Button aria-label="Close dialog">
  <X aria-hidden="true" />
</Button>
```

### 4. Using Color Alone

```tsx
// ❌ BAD (color is only indicator)
<span className="text-red-600">Error</span>

// ✅ GOOD (icon + color)
<span className="text-red-600 flex items-center gap-1">
  <AlertTriangle className="h-4 w-4" aria-hidden="true" />
  <span role="alert">Error</span>
</span>
```

### 5. Auto-Playing Media

```tsx
// ❌ BAD
<video autoPlay src="demo.mp4" />

// ✅ GOOD
<video src="demo.mp4" controls>
  <track kind="captions" src="captions.vtt" />
</video>
```

---

## 📚 Resources

### Official Standards
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)

### Tools
- [axe DevTools](https://www.deque.com/axe/devtools/) - Browser extension
- [WAVE](https://wave.webaim.org/extension/) - Browser extension
- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)

### Libraries
- [Radix UI](https://www.radix-ui.com/) - Accessible component primitives (already in use)
- [jest-axe](https://github.com/nickcolley/jest-axe) - Automated testing (configured)

### Learning
- [a11y Project](https://www.a11yproject.com/) - Accessibility checklist
- [Inclusive Components](https://inclusive-components.design/) - Accessible patterns

---

## 🤝 Getting Help

1. **Before coding:** Check this guide for the pattern you need
2. **During review:** Run `npm test` to catch a11y issues
3. **Questions:** Ask in #frontend or tag @accessibility-champions

---

## 📝 Changelog

| Date | Change |
|------|--------|
| 2026-02-19 | Initial guidelines created |
| 2026-02-19 | Added page title hook pattern |
| 2026-02-19 | Added table accessibility standards |

---

**Remember:** Accessibility is not a feature—it's a requirement. Building accessible interfaces makes ZRP better for everyone. 🚀
