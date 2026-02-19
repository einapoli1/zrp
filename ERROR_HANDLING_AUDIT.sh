#!/bin/bash
# Comprehensive error handling audit script for ZRP
# This script scans both frontend and backend for error handling gaps

echo "🔍 ZRP ERROR HANDLING AUDIT"
echo "=============================="
echo ""

# Frontend Audit
echo "📱 FRONTEND AUDIT"
echo "----------------"
echo ""

# Count pages with/without try-catch
total_pages=$(find frontend/src/pages -name "*.tsx" -not -name "*.test.tsx" | wc -l | tr -d ' ')
pages_with_trycatch=$(grep -l "try" frontend/src/pages/*.tsx 2>/dev/null | grep -v ".test.tsx" | wc -l | tr -d ' ')
pages_without_trycatch=$((total_pages - pages_with_trycatch))

echo "📊 Page Error Handling Coverage:"
echo "  Total pages: $total_pages"
echo "  With try/catch: $pages_with_trycatch"
echo "  WITHOUT try/catch: $pages_without_trycatch"
echo ""

# List pages without try/catch
echo "⚠️  Pages missing try/catch blocks:"
grep -L "try" frontend/src/pages/*.tsx 2>/dev/null | grep -v ".test.tsx" | sed 's|frontend/src/pages/||' | sed 's/.tsx//' | while read page; do
  echo "  - $page"
done
echo ""

# Check for inconsistent error display
echo "🎨 Error Display Patterns:"
echo "  Using toast: $(grep -l "toast.error\|toast(" frontend/src/pages/*.tsx 2>/dev/null | wc -l | tr -d ' ') pages"
echo "  Using error state: $(grep -l "setError\|error\]\s*=" frontend/src/pages/*.tsx 2>/dev/null | wc -l | tr -d ' ') pages"
echo "  Silent failures (catch without user feedback): $(grep -A3 "catch" frontend/src/pages/*.tsx 2>/dev/null | grep -c "console.error")"
echo ""

# Check for missing error messages in catch blocks
echo "🔇 Silent Catch Blocks (no user feedback):"
for file in frontend/src/pages/*.tsx; do
  if grep -q "catch" "$file"; then
    # Check if there's a toast or setError after catch
    if ! grep -A5 "catch" "$file" | grep -q "toast\|setError"; then
      basename "$file" .tsx
    fi
  fi
done | head -10
echo ""

# Backend Audit
echo "🔧 BACKEND AUDIT"
echo "---------------"
echo ""

# Find handlers that ignore errors
echo "⚠️  Handlers ignoring returned errors (using _):"
grep -n "_, _ :=\|_, err :=" handler_*.go 2>/dev/null | grep -v "if err" | head -10
echo ""

# Find handlers without validation
echo "📋 Handlers without validation:"
for file in handler_*.go; do
  handler_count=$(grep -c "^func handle" "$file" 2>/dev/null || echo 0)
  validation_count=$(grep -c "ValidationErrors\|requireField\|validateEnum" "$file" 2>/dev/null || echo 0)
  if [ "$handler_count" -gt 0 ] && [ "$validation_count" -eq 0 ]; then
    echo "  - $file ($handler_count handlers, 0 validation)"
  fi
done
echo ""

# Find jsonErr calls with generic messages
echo "🔍 Generic Error Messages (exposing internals):"
grep -n "jsonErr.*err.Error()" handler_*.go 2>/dev/null | head -10
echo ""

# Find DELETE operations without proper error handling
echo "🗑️  DELETE operations:"
grep -n "DELETE\|handleDelete" handler_*.go | head -10
echo ""

# API mutation operations without try/catch
echo "🔄 API Mutation Calls (DELETE/POST/PUT) in Frontend:"
echo ""
echo "  DELETE calls without try/catch:"
for file in frontend/src/pages/*.tsx; do
  if grep -q "api.delete\|api.bulk.*Delete\|\.delete(" "$file"; then
    if ! grep -B10 "api.delete\|api.bulk.*Delete\|\.delete(" "$file" | grep -q "try"; then
      basename "$file" .tsx
    fi
  fi
done | head -10
echo ""

echo "  POST/PUT calls without try/catch:"
for file in frontend/src/pages/*.tsx; do
  if grep -q "api.create\|api.update\|api\..*Post\|api\..*Put" "$file"; then
    if ! grep -B10 "api.create\|api.update" "$file" | grep -q "try"; then
      basename "$file" .tsx
    fi
  fi
done | head -10
echo ""

# Check for ErrorBoundary usage
echo "🛡️  Error Boundary Coverage:"
error_boundary_imports=$(grep -l "ErrorBoundary" frontend/src/**/*.tsx 2>/dev/null | wc -l | tr -d ' ')
echo "  Files importing ErrorBoundary: $error_boundary_imports"
echo ""

# Summary statistics
echo "📈 SUMMARY"
echo "----------"
echo "  Frontend pages without error handling: $pages_without_trycatch / $total_pages"
echo "  Backend handlers with ignored errors: $(grep -c "_, _ :=\|_, err :=" handler_*.go 2>/dev/null | head -1)"
echo "  Generic error messages in backend: $(grep -c "jsonErr.*err.Error()" handler_*.go 2>/dev/null)"
echo ""

echo "✅ Audit complete!"
