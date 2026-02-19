import { configureAxe } from "jest-axe";
import type { RenderResult } from "@testing-library/react";

/**
 * Accessibility testing utilities for ZRP
 * 
 * Uses jest-axe (axe-core) to test for WCAG 2.1 AA compliance
 */

// Configure axe with our standards
export const axe = configureAxe({
  rules: {
    // Enforce WCAG 2.1 AA standards
    "color-contrast": { enabled: true },
    region: { enabled: true },
    "label-title-only": { enabled: true },
    "landmark-one-main": { enabled: true },
    "page-has-heading-one": { enabled: true },
  },
});

/**
 * Test a rendered component for accessibility violations
 * 
 * @example
 * ```tsx
 * test("Dashboard should not have accessibility violations", async () => {
 *   const { container } = render(<Dashboard />);
 *   await expectNoA11yViolations(container);
 * });
 * ```
 */
export async function expectNoA11yViolations(container: HTMLElement) {
  const results = await axe(container);
  expect(results).toHaveNoViolations();
}

/**
 * Test a rendered component from @testing-library/react
 * 
 * @example
 * ```tsx
 * test("Parts page is accessible", async () => {
 *   const view = render(<Parts />);
 *   await testA11y(view);
 * });
 * ```
 */
export async function testA11y(renderResult: RenderResult) {
  await expectNoA11yViolations(renderResult.container);
}

/**
 * Check specific accessibility rules
 * 
 * @example
 * ```tsx
 * const results = await checkA11y(container, ["button-name", "link-name"]);
 * ```
 */
export async function checkA11y(container: HTMLElement, rules?: string[]) {
  const customAxe = rules
    ? configureAxe({
        rules: rules.reduce((acc, rule) => ({ ...acc, [rule]: { enabled: true } }), {}),
      })
    : axe;
  
  return await customAxe(container);
}

/**
 * Get human-readable summary of accessibility violations
 */
export function formatA11yViolations(violations: any[]) {
  if (violations.length === 0) return "No violations found";
  
  return violations.map(violation => 
    `${violation.id} (${violation.impact}): ${violation.description}\n` +
    `  Help: ${violation.helpUrl}\n` +
    `  Elements: ${violation.nodes.length}`
  ).join("\n\n");
}
