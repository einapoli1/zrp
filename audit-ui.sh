#!/bin/bash

# UI Pattern Audit Script for ZRP
# Checks all page files for key UI polish patterns

PAGES_DIR="frontend/src/pages"
OUTPUT="ui-audit-report.md"

echo "# ZRP UI Audit Report" > "$OUTPUT"
echo "Generated: $(date)" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "## Summary" >> "$OUTPUT"
echo "" >> "$OUTPUT"

# Initialize counters
total=0
has_loading=0
has_empty=0
has_error=0
has_form_field=0
has_labels=0
has_responsive=0

echo "| Page | Loading | Empty | Error | FormField | Labels | Responsive | Score |" >> "$OUTPUT"
echo "|------|---------|-------|-------|-----------|--------|------------|-------|" >> "$OUTPUT"

for file in "$PAGES_DIR"/*.tsx; do
  # Skip test files
  if [[ "$file" == *".test.tsx" ]]; then
    continue
  fi
  
  filename=$(basename "$file")
  total=$((total + 1))
  
  # Check for patterns
  loading=0
  empty=0
  error=0
  formfield=0
  labels=0
  responsive=0
  
  # LoadingState usage
  if grep -q "LoadingState" "$file" || grep -q "loading.*?" "$file"; then
    loading=2
    has_loading=$((has_loading + 1))
  elif grep -q "Loading\.\.\." "$file"; then
    loading=1
  fi
  
  # EmptyState usage
  if grep -q "EmptyState" "$file"; then
    empty=2
    has_empty=$((has_empty + 1))
  elif grep -q "No.*found\|empty" "$file"; then
    empty=1
  fi
  
  # ErrorState or error handling
  if grep -q "ErrorState" "$file"; then
    error=2
    has_error=$((has_error + 1))
  elif grep -q "onRetry\|try.*catch\|toast\.error" "$file"; then
    error=1
  fi
  
  # FormField usage
  if grep -q "FormField" "$file"; then
    formfield=2
    has_form_field=$((has_form_field + 1))
  elif grep -q "Label.*htmlFor" "$file"; then
    formfield=1
  fi
  
  # Label association (accessibility)
  if grep -q 'htmlFor=\|aria-label=\|aria-labelledby=' "$file"; then
    labels=2
    has_labels=$((has_labels + 1))
  elif grep -q '<Label' "$file"; then
    labels=1
  fi
  
  # Responsive design patterns
  if grep -q 'sm:\|md:\|lg:\|xl:\|grid-cols-' "$file"; then
    responsive=2
    has_responsive=$((has_responsive + 1))
  elif grep -q 'flex.*wrap' "$file"; then
    responsive=1
  fi
  
  # Calculate score (out of 12)
  score=$((loading + empty + error + formfield + labels + responsive))
  
  # Format row
  echo "| $filename | $loading | $empty | $error | $formfield | $labels | $responsive | **$score/12** |" >> "$OUTPUT"
done

echo "" >> "$OUTPUT"
echo "## Statistics" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "- **Total Pages**: $total" >> "$OUTPUT"
echo "- **Pages with LoadingState**: $has_loading ($((has_loading * 100 / total))%)" >> "$OUTPUT"
echo "- **Pages with EmptyState**: $has_empty ($((has_empty * 100 / total))%)" >> "$OUTPUT"
echo "- **Pages with ErrorState**: $has_error ($((has_error * 100 / total))%)" >> "$OUTPUT"
echo "- **Pages with FormField**: $has_form_field ($((has_form_field * 100 / total))%)" >> "$OUTPUT"
echo "- **Pages with proper Labels**: $has_labels ($((has_labels * 100 / total))%)" >> "$OUTPUT"
echo "- **Pages with Responsive Design**: $has_responsive ($((has_responsive * 100 / total))%)" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "## Scoring Guide" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "- **2**: Pattern fully implemented" >> "$OUTPUT"
echo "- **1**: Partial implementation" >> "$OUTPUT"
echo "- **0**: Missing" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "**Total Score Range**: 0-12 per page" >> "$OUTPUT"

echo "✅ Audit complete! Report saved to $OUTPUT"
cat "$OUTPUT"
