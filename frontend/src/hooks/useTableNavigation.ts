import { useHotkeys } from 'react-hotkeys-hook';
import { useCallback, useState } from 'react';

interface UseTableNavigationOptions {
  onNavigate?: (index: number) => void;
  rowCount: number;
  onRowClick?: (index: number) => void;
}

export function useTableNavigation({ 
  rowCount, 
  onRowClick,
  onNavigate 
}: UseTableNavigationOptions) {
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const handleNavigateDown = useCallback(() => {
    const newIndex = selectedIndex < rowCount - 1 ? selectedIndex + 1 : selectedIndex;
    setSelectedIndex(newIndex);
    onNavigate?.(newIndex);
  }, [selectedIndex, rowCount, onNavigate]);

  const handleNavigateUp = useCallback(() => {
    const newIndex = selectedIndex > 0 ? selectedIndex - 1 : 0;
    setSelectedIndex(newIndex);
    onNavigate?.(newIndex);
  }, [selectedIndex, onNavigate]);

  const handleEnter = useCallback(() => {
    if (selectedIndex >= 0 && selectedIndex < rowCount) {
      onRowClick?.(selectedIndex);
    }
  }, [selectedIndex, rowCount, onRowClick]);

  // j = down, k = up, Enter = select
  useHotkeys('j', handleNavigateDown, { 
    enableOnFormTags: false,
    preventDefault: true 
  }, [handleNavigateDown]);

  useHotkeys('k', handleNavigateUp, { 
    enableOnFormTags: false,
    preventDefault: true 
  }, [handleNavigateUp]);

  useHotkeys('enter', handleEnter, { 
    enableOnFormTags: false,
    preventDefault: true 
  }, [handleEnter]);

  return {
    selectedIndex,
    setSelectedIndex,
  };
}
