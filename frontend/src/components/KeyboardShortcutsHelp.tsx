import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { useHotkeys } from 'react-hotkeys-hook';
import { useState } from 'react';
import { Keyboard } from 'lucide-react';

interface Shortcut {
  key: string;
  description: string;
}

interface KeyboardShortcutsHelpProps {
  shortcuts: Shortcut[];
}

export function KeyboardShortcutsHelp({ shortcuts }: KeyboardShortcutsHelpProps) {
  const [open, setOpen] = useState(false);

  // Global ? shortcut to show help
  useHotkeys('shift+/', () => setOpen(true), { preventDefault: true });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Keyboard className="h-5 w-5" />
            Keyboard Shortcuts
          </DialogTitle>
          <DialogDescription>
            Use these shortcuts to navigate and perform actions quickly
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3 py-4">
          {shortcuts.map((shortcut, index) => (
            <div key={index} className="flex items-center justify-between gap-4">
              <span className="text-sm text-muted-foreground">{shortcut.description}</span>
              <kbd className="pointer-events-none inline-flex h-7 select-none items-center gap-1 rounded border bg-muted px-2 font-mono text-sm font-medium text-muted-foreground opacity-100">
                {shortcut.key}
              </kbd>
            </div>
          ))}
          <div className="flex items-center justify-between gap-4 border-t pt-3 mt-3">
            <span className="text-sm text-muted-foreground">Show this help</span>
            <kbd className="pointer-events-none inline-flex h-7 select-none items-center gap-1 rounded border bg-muted px-2 font-mono text-sm font-medium text-muted-foreground opacity-100">
              ?
            </kbd>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
