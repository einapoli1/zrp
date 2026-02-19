import { useEffect, useCallback } from "react";

type ShortcutHandler = (e: KeyboardEvent) => void;

interface ShortcutConfig {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  handler: ShortcutHandler;
  /** Don't fire when inside input/textarea */
  ignoreInputs?: boolean;
}

/**
 * Hook to register keyboard shortcuts.
 * Automatically cleans up on unmount.
 */
export function useKeyboardShortcuts(shortcuts: ShortcutConfig[]) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const isInput =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable;

      for (const shortcut of shortcuts) {
        if (shortcut.ignoreInputs !== false && isInput) continue;
        if (e.key.toLowerCase() !== shortcut.key.toLowerCase()) continue;
        if (shortcut.ctrl && !e.ctrlKey) continue;
        if (shortcut.meta && !e.metaKey) continue;
        if (shortcut.shift && !e.shiftKey) continue;
        if (!shortcut.ctrl && e.ctrlKey) continue;
        if (!shortcut.meta && e.metaKey) continue;

        e.preventDefault();
        shortcut.handler(e);
        return;
      }
    },
    [shortcuts]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
