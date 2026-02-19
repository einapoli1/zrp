#!/bin/bash
# UI Consistency Audit Script for ZRP

PAGES_DIR="frontend/src/pages"
OUTPUT="ui-consistency-audit.md"

echo "# ZRP UI Consistency Audit Report" > $OUTPUT
echo "**Date:** $(date '+%Y-%m-%d %H:%M:%S')" >> $OUTPUT
echo "" >> $OUTPUT

# Count pages
PAGE_COUNT=$(find $PAGES_DIR -name "*.tsx" ! -name "*.test.tsx" | wc -l)
echo "**Total Pages Analyzed:** $PAGE_COUNT" >> $OUTPUT
echo "" >> $OUTPUT

echo "## Audit Criteria" >> $OUTPUT
echo "1. Loading states (useQuery isLoading, skeleton/spinner)" >> $OUTPUT
echo "2. Empty states (no data messages)" >> $OUTPUT
echo "3. Error handling (toast, inline, try-catch)" >> $OUTPUT
echo "4. Button styles (variant consistency)" >> $OUTPUT
echo "5. Form layouts (spacing, validation)" >> $OUTPUT
echo "" >> $OUTPUT

echo "## Page Analysis" >> $OUTPUT
echo "" >> $OUTPUT

for page in $(find $PAGES_DIR -name "*.tsx" ! -name "*.test.tsx" | sort); do
  filename=$(basename $page)
  echo "### $filename" >> $OUTPUT
  
  # Check for loading states
  has_loading=$(grep -c "isLoading\|isPending\|Skeleton\|Spinner\|Loading" $page || echo 0)
  
  # Check for empty states
  has_empty=$(grep -c "No data\|Empty\|no.*found\|length === 0" $page || echo 0)
  
  # Check for error handling
  has_error=$(grep -c "isError\|error\|catch\|toast.*error\|Error" $page || echo 0)
  
  # Check for buttons
  has_buttons=$(grep -c "<Button\|<button" $page || echo 0)
  
  # Check for forms
  has_forms=$(grep -c "<form\|useForm\|FormField" $page || echo 0)
  
  echo "- Loading: $has_loading | Empty: $has_empty | Error: $has_error | Buttons: $has_buttons | Forms: $has_forms" >> $OUTPUT
  
  # Flag potential issues
  if [ $has_loading -eq 0 ] && grep -q "useQuery\|useMutation" $page; then
    echo "  - ⚠️ **Missing loading state** (has query but no loading check)" >> $OUTPUT
  fi
  
  if [ $has_empty -eq 0 ] && grep -q "map(\|\.length" $page; then
    echo "  - ⚠️ **Missing empty state** (renders lists but no empty check)" >> $OUTPUT
  fi
  
  if [ $has_error -eq 0 ] && grep -q "useQuery\|useMutation\|fetch\|axios" $page; then
    echo "  - ⚠️ **Missing error handling** (has data fetching but no error check)" >> $OUTPUT
  fi
  
  echo "" >> $OUTPUT
done

echo "## Summary" >> $OUTPUT
echo "" >> $OUTPUT
echo "Audit complete. Review flagged issues above." >> $OUTPUT
