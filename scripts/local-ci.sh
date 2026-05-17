#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"

log() {
  printf '\n=== %s ===\n' "$1"
}

run() {
  printf '+ %s\n' "$*"
  "$@"
}

resolve_base() {
  if [ -n "${CI_BASE:-}" ]; then
    printf '%s\n' "$CI_BASE"
    return
  fi
  if git -C "$ROOT_DIR" rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    printf 'HEAD~1\n'
    return
  fi
  printf '0000000000000000000000000000000000000000\n'
}

changed_go_files() {
  local base="$1"
  if [ "$base" = "0000000000000000000000000000000000000000" ]; then
    git -C "$ROOT_DIR" diff --name-only HEAD -- '*.go' | grep -v '_test.go' || true
  else
    git -C "$ROOT_DIR" diff --name-only "$base" HEAD -- '*.go' | grep -v '_test.go' || true
  fi
}

check_backend_incremental_coverage() {
  local base="$1"
  local changed
  changed="$(changed_go_files "$base")"

  if [ -z "$changed" ]; then
    echo "No Go source files changed, skipping coverage check."
    return
  fi

  echo "=== Changed files ==="
  echo "$changed"
  echo ""
  echo "=== Per-file coverage ==="

  local fail=0
  while IFS= read -r file; do
    [ -z "$file" ] && continue
    local backend_file="${file#backend/}"
    local cov
    cov="$(
      go tool cover -func="$BACKEND_DIR/coverage.out" | grep "/$backend_file:" | awk '
        {
          pct=$NF
          sub(/%/, "", pct)
          sum+=pct
          n++
        }
        END {
          if (n > 0) printf "%.1f", sum / n
          else print "N/A"
        }
      '
    )"

    if [ "$cov" = "N/A" ]; then
      echo "  $file -> no coverage data (missing tests)"
      fail=1
      continue
    fi

    if awk "BEGIN { exit !($cov >= $COVERAGE_THRESHOLD) }"; then
      echo "  $file -> ${cov}% OK"
    else
      echo "  $file -> ${cov}% FAIL (need >=${COVERAGE_THRESHOLD}%)"
      fail=1
    fi
  done <<< "$changed"

  if [ "$fail" -eq 1 ]; then
    echo ""
    echo "Incremental coverage check FAILED."
    exit 1
  fi

  echo ""
  echo "All changed files meet ${COVERAGE_THRESHOLD}% coverage threshold."
}

run_backend_tests_with_coverage() {
  local output
  output="$(mktemp)"
  if (cd "$BACKEND_DIR" && go test ./... -coverprofile=coverage.out -covermode=atomic) >"$output" 2>&1; then
    cat "$output"
    rm -f "$output"
    return
  fi

  cat "$output"
  if ! grep -q 'no such tool "covdata"' "$output"; then
    rm -f "$output"
    exit 1
  fi
  rm -f "$output"

  echo ""
  echo "Local Go toolchain is missing covdata; retrying packages that have test files."
  echo "Remote CI still runs: go test ./... -coverprofile=coverage.out -covermode=atomic"

  local packages
  packages="$(cd "$BACKEND_DIR" && go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep . || true)"
  if [ -z "$packages" ]; then
    echo "No backend packages with tests found."
    exit 1
  fi

  (
    cd "$BACKEND_DIR"
    # shellcheck disable=SC2086
    go test $packages -coverprofile=coverage.out -covermode=atomic
  )
}

main() {
  local base
  base="$(resolve_base)"

  log "Backend: tests with coverage"
  printf '+ go test ./... -coverprofile=coverage.out -covermode=atomic\n'
  run_backend_tests_with_coverage

  log "Backend: incremental coverage gate"
  echo "Coverage base: $base"
  check_backend_incremental_coverage "$base"

  log "Backend: build"
  (
    cd "$BACKEND_DIR"
    run go build ./...
  )

  log "Frontend: install dependencies"
  (
    cd "$FRONTEND_DIR"
    run npm ci
  )

  log "Frontend: lint"
  (
    cd "$FRONTEND_DIR"
    run npm run lint
  )

  log "Frontend: unit tests"
  (
    cd "$FRONTEND_DIR"
    run npm test
  )

  log "Frontend: coverage gate"
  (
    cd "$FRONTEND_DIR"
    run npm run test:coverage
  )

  log "Frontend: build"
  (
    cd "$FRONTEND_DIR"
    run npm run build
  )

  log "Local CI passed"
}

main "$@"
