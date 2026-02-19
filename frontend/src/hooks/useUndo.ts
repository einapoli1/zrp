import { useCallback } from 'react';
import { toast } from 'sonner';
import { api } from '../lib/api';

interface UndoOptions {
  entityType: string;
  entityId: string;
  action: string;
  description?: string;
}

/**
 * Hook for undo functionality on destructive actions.
 * Shows a toast with an "Undo" button for 5 seconds after a destructive action.
 */
export function useUndo() {
  const showUndoToast = useCallback(
    (undoId: number, options: UndoOptions) => {
      const desc = options.description || `${options.action} ${options.entityType} ${options.entityId}`;
      toast(desc, {
        duration: 5000,
        action: {
          label: 'Undo',
          onClick: async () => {
            try {
              await api.performUndo(undoId);
              toast.success(`Restored ${options.entityType} ${options.entityId}`);
            } catch {
              toast.error('Undo failed');
            }
          },
        },
      });
    },
    []
  );

  /**
   * Wraps a destructive API call. If the response contains undo_id,
   * shows a toast with an Undo button.
   */
  const withUndo = useCallback(
    async <T extends Record<string, unknown>>(
      apiCall: () => Promise<T>,
      options: Omit<UndoOptions, 'action'> & { action?: string }
    ): Promise<T> => {
      const result = await apiCall();
      const undoId = (result as Record<string, unknown>)?.undo_id as number | undefined;
      if (undoId) {
        showUndoToast(undoId, {
          action: options.action || 'Deleted',
          ...options,
        });
      }
      return result;
    },
    [showUndoToast]
  );

  return { showUndoToast, withUndo };
}
