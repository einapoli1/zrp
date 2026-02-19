import { useEffect } from "react";

/**
 * Custom hook to update page title and announce page changes to screen readers
 * 
 * @param title - The page title (will be prefixed with "ZRP | ")
 * @param announce - Whether to announce the page change via aria-live region (default: true)
 * 
 * @example
 * ```tsx
 * function PartsPage() {
 *   usePageTitle("Parts");
 *   // Page title is now "ZRP | Parts"
 *   // Screen readers announce: "Parts"
 * }
 * ```
 */
export function usePageTitle(title: string, announce: boolean = true) {
  useEffect(() => {
    // Update document title
    const fullTitle = `ZRP | ${title}`;
    document.title = fullTitle;

    // Announce page change to screen readers
    if (announce) {
      // Create or update aria-live region for page announcements
      let announcer = document.getElementById("page-title-announcer");
      if (!announcer) {
        announcer = document.createElement("div");
        announcer.id = "page-title-announcer";
        announcer.setAttribute("role", "status");
        announcer.setAttribute("aria-live", "polite");
        announcer.setAttribute("aria-atomic", "true");
        announcer.className = "sr-only";
        document.body.appendChild(announcer);
      }
      
      // Clear and re-announce to ensure screen readers pick it up
      announcer.textContent = "";
      // Use setTimeout to ensure the change is detected
      setTimeout(() => {
        announcer!.textContent = `Navigated to ${title}`;
      }, 100);
    }

    // Cleanup: restore default title on unmount
    return () => {
      document.title = "ZRP";
    };
  }, [title, announce]);
}
