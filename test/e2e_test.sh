#!/usr/bin/env bash
set -euo pipefail

# E2E Test Harness for Phase 4 Implementation
# Builds binaries, runs integration tests, reports results

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Phase 4 E2E Test Harness ==="
echo "Project root: $PROJECT_ROOT"
echo

# Step 1: Build prismctl
echo "[1/5] Building prismctl..."
cd "$PROJECT_ROOT"
go build -o bin/prismctl ./cmd/prismctl
if [[ -f bin/prismctl ]]; then
  echo "✓ prismctl built successfully"
else
  echo "✗ Failed to build prismctl"
  exit 1
fi
echo

# Step 2: Build test-prism fixture
echo "[2/5] Building test-prism fixture..."
cd "$PROJECT_ROOT/test/fixtures"
go build -o test-prism test_prism.go
if [[ -f test-prism ]]; then
  echo "✓ test-prism built successfully"
else
  echo "✗ Failed to build test-prism"
  exit 1
fi
echo

# Step 3: Add test-prism to PATH
export PATH="$PROJECT_ROOT/test/fixtures:$PATH"
echo "[3/5] Added test-prism to PATH"
echo

# Step 4: Run integration tests
echo "[4/5] Running integration tests..."
cd "$PROJECT_ROOT/test/integration"

# Create go.mod for integration tests
if [[ ! -f go.mod ]]; then
  cat > go.mod <<EOF
module github.com/starbased-co/shine/test/integration

go 1.23

require (
	golang.org/x/sys v0.29.0
	golang.org/x/term v0.28.0
)
EOF
  go mod tidy
fi

echo
echo "--- Running TestMultiPrismLifecycle ---"
if go test -v -run TestMultiPrismLifecycle -timeout 30s; then
  echo "✓ TestMultiPrismLifecycle PASSED"
  LIFECYCLE_RESULT="PASS"
else
  echo "✗ TestMultiPrismLifecycle FAILED"
  LIFECYCLE_RESULT="FAIL"
fi

echo
echo "--- Running TestHotSwap ---"
if go test -v -run TestHotSwap -timeout 30s; then
  echo "✓ TestHotSwap PASSED"
  HOTSWAP_RESULT="PASS"
else
  echo "✗ TestHotSwap FAILED"
  HOTSWAP_RESULT="FAIL"
fi

echo
echo "--- Running TestBackgroundProcessing ---"
if go test -v -run TestBackgroundProcessing -timeout 30s; then
  echo "✓ TestBackgroundProcessing PASSED"
  BACKGROUND_RESULT="PASS"
else
  echo "✗ TestBackgroundProcessing FAILED"
  BACKGROUND_RESULT="FAIL"
fi

echo
echo "--- Running TestSIGWINCH ---"
if go test -v -run TestSIGWINCH -timeout 30s; then
  echo "✓ TestSIGWINCH PASSED"
  SIGWINCH_RESULT="PASS"
else
  echo "✗ TestSIGWINCH FAILED"
  SIGWINCH_RESULT="FAIL"
fi

echo
echo "--- Running TestStress ---"
if go test -v -run TestStress -timeout 60s; then
  echo "✓ TestStress PASSED"
  STRESS_RESULT="PASS"
else
  echo "✗ TestStress FAILED"
  STRESS_RESULT="FAIL"
fi

echo
echo "[5/5] Test Results Summary"
echo "=========================="
echo "TestMultiPrismLifecycle:   $LIFECYCLE_RESULT"
echo "TestHotSwap:               $HOTSWAP_RESULT"
echo "TestBackgroundProcessing:  $BACKGROUND_RESULT"
echo "TestSIGWINCH:              $SIGWINCH_RESULT"
echo "TestStress:                $STRESS_RESULT"
echo

# Calculate overall result
if [[ "$LIFECYCLE_RESULT" == "PASS" ]] && \
   [[ "$HOTSWAP_RESULT" == "PASS" ]] && \
   [[ "$BACKGROUND_RESULT" == "PASS" ]] && \
   [[ "$SIGWINCH_RESULT" == "PASS" ]] && \
   [[ "$STRESS_RESULT" == "PASS" ]]; then
  echo "=== ALL TESTS PASSED ==="
  exit 0
else
  echo "=== SOME TESTS FAILED ==="
  exit 1
fi
