/**
 * Accessibility testing setup for Vitest
 * 
 * This file should be imported in vitest.setup.ts or test setup files
 */

import { toHaveNoViolations } from "jest-axe";
import { expect } from "vitest";

// Extend Vitest's expect with jest-axe matchers
expect.extend(toHaveNoViolations);

// Add custom matchers type declarations
declare module "vitest" {
  interface Assertion {
    toHaveNoViolations(): void;
  }
  interface AsymmetricMatchersContaining {
    toHaveNoViolations(): void;
  }
}

// Suppress axe-core console warnings in tests (optional)
// Uncomment if axe-core is too noisy in test output
// const originalWarn = console.warn;
// console.warn = (...args: any[]) => {
//   if (args[0]?.includes?.("axe-core")) return;
//   originalWarn(...args);
// };
