#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track overall success
FAILED=0

echo "🔍 ZRP Build Verification Script"
echo "================================="
echo ""

# Function to report results
report_step() {
    local step_name="$1"
    local status="$2"
    
    if [ "$status" -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $step_name: PASSED"
    else
        echo -e "${RED}✗${NC} $step_name: FAILED"
        FAILED=1
    fi
}

# 1. Backend Build
echo "🔨 Building Go backend..."
if go build ./... 2>&1; then
    report_step "Backend Build" 0
else
    report_step "Backend Build" 1
fi

# 2. Backend Tests
echo ""
echo "🧪 Running Go tests..."
if go test ./... 2>&1; then
    report_step "Backend Tests" 0
else
    report_step "Backend Tests" 1
fi

# 3. Frontend Tests
echo ""
echo "🧪 Running frontend tests..."
cd frontend
if npx vitest run 2>&1; then
    report_step "Frontend Tests" 0
else
    report_step "Frontend Tests" 1
fi

# 4. Frontend Build (TypeScript compilation + Vite)
echo ""
echo "🔨 Building frontend (TypeScript + Vite)..."
if npm run build 2>&1; then
    report_step "Frontend Build" 0
else
    report_step "Frontend Build" 1
fi

cd ..

# Final report
echo ""
echo "================================="
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some checks failed. Fix errors before committing.${NC}"
    exit 1
fi
