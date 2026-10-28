#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

normalize_node_version() {
  printf '%s\n' "${1#v}"
}

DEFAULT_BASE_COMMIT="dbc2319f4"
BASE_COMMIT="${1:-$DEFAULT_BASE_COMMIT}"
if [[ -z "$BASE_COMMIT" ]]; then
  echo "Usage: $0 <base-commit>"
  echo "Environment knobs:"
  echo "  BASE_WORKTREE   (default: ../pipelines-vite-baseline)"
  echo "  BASE_PORT       (default: 3010)"
  echo "  CURRENT_PORT    (default: 3020)"
  echo "  MOCK_PORT       (default: 3001)"
  echo "  USE_MOCK        (default: 1)"
  echo "  SKIP_INSTALL    (default: 0)"
  echo "  ROUTES          (default: frontend/scripts/visual-compare.routes.json)"
  echo "  BASE_ROUTES    (default: ROUTES; use a compatible manifest for older baselines)"
  echo "  FIXED_TIME      (default: 2026-09-26T12:00:00.000Z; empty uses the real clock)"
  exit 1
fi

BASE_WORKTREE="${BASE_WORKTREE:-$ROOT/../pipelines-vite-baseline}"
BASE_PORT="${BASE_PORT:-3010}"
CURRENT_PORT="${CURRENT_PORT:-3020}"
MOCK_PORT="${MOCK_PORT:-3001}"
USE_MOCK="${USE_MOCK:-1}"
SKIP_INSTALL="${SKIP_INSTALL:-0}"
ROUTES="${ROUTES:-$ROOT/frontend/scripts/visual-compare.routes.json}"
# Resolve caller-supplied routes before npm changes its working directory.
if [[ "$ROUTES" != /* ]]; then
  ROUTES="$PWD/$ROUTES"
fi
BASE_ROUTES="${BASE_ROUTES:-$ROUTES}"
if [[ "$BASE_ROUTES" != /* ]]; then
  BASE_ROUTES="$PWD/$BASE_ROUTES"
fi
FIXED_TIME="${FIXED_TIME-2026-09-26T12:00:00.000Z}"
clock_args=()
if [[ -n "$FIXED_TIME" ]]; then
  clock_args=(--fixed-time "$FIXED_TIME")
fi
OUT_DIR="$ROOT/frontend/.visual"

pids=()
cleanup() {
  for pid in "${pids[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

resolve_node_version() {
  local dir="$1"
  if [[ -f "$dir/frontend/.nvmrc" ]]; then
    local version
    version="$(tr -d '\r\n' < "$dir/frontend/.nvmrc")" || return $?
    normalize_node_version "$version"
  else
    # Historical checkouts without a pin use the current checkout's version.
    echo "$CURRENT_NODE_VERSION"
  fi
}

ensure_node_version() {
  local version="$1"
  if command -v fnm >/dev/null 2>&1; then
    fnm install "$version" >/dev/null
  fi
}

node_bin_dir() {
  local version="$1"
  local fnm_dir="${FNM_DIR:-$HOME/.local/share/fnm}"
  local dir="$fnm_dir/node-versions/v$version/installation/bin"
  if [[ -x "$dir/node" ]]; then
    echo "$dir"
  fi
}

run_with_node() {
  local dir="$1"
  shift
  local version
  version="$(resolve_node_version "$dir")" || return $?
  ensure_node_version "$version" || return $?
  local bin_dir
  bin_dir="$(node_bin_dir "$version")" || return $?
  if [[ -n "$bin_dir" ]]; then
    PATH="$bin_dir:$PATH" "$@"
  else
    "$@"
  fi
}

wait_url() {
  local url="$1"
  local label="$2"
  local log_path="${3:-}"
  local attempts=0
  until curl -sS --fail "$url" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [[ "$attempts" -ge 60 ]]; then
      echo "Timed out waiting for $label at $url"
      if [[ -n "$log_path" && -f "$log_path" ]]; then
        echo "---- $label log (tail) ----"
        tail -n 200 "$log_path" || true
        echo "---------------------------"
      fi
      exit 1
    fi
    sleep 1
  done
}

# Initialize the capture/report toolchain before launching any servers. Setup
# failures are not capture failures and cannot produce useful report diagnostics.
prepare_current_node() {
  local version
  version="$(tr -d '\r\n' < "$ROOT/frontend/.nvmrc")" || return $?
  version="$(normalize_node_version "$version")"
  ensure_node_version "$version" || return $?
  local bin_dir
  bin_dir="$(node_bin_dir "$version")" || return $?
  CURRENT_NODE_VERSION="$version"
  CURRENT_NODE_PATH="$PATH"
  if [[ -n "$bin_dir" ]]; then
    CURRENT_NODE_PATH="$bin_dir:$PATH"
  fi
  local actual_version
  actual_version="$(PATH="$CURRENT_NODE_PATH" node -p 'process.versions.node')" || return $?
  if [[ ! "$actual_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] ||
    (( BASH_REMATCH[1] < 24 || (BASH_REMATCH[1] == 24 && BASH_REMATCH[2] < 2) )); then
    setup_error="Node >=24.2.0 is required for capture/report; found $actual_version."
    return 1
  fi
}
setup_error=""
setup_status=0
prepare_current_node || setup_status=$?
if [[ "$setup_status" -ne 0 ]]; then
  echo "Node toolchain setup failed (exit $setup_status); comparison not started.${setup_error:+ $setup_error}" >&2
  exit "$setup_status"
fi

run_current_node() {
  PATH="$CURRENT_NODE_PATH" "$@"
}

mkdir -p "$OUT_DIR"

if [[ ! -e "$BASE_WORKTREE/.git" ]]; then
  git -C "$ROOT" worktree add "$BASE_WORKTREE" "$BASE_COMMIT"
else
  git -C "$BASE_WORKTREE" checkout "$BASE_COMMIT"
fi

if [[ "$SKIP_INSTALL" != "1" ]]; then
  run_current_node npm --prefix "$ROOT/frontend" ci
  run_with_node "$BASE_WORKTREE" npm --prefix "$BASE_WORKTREE/frontend" ci
fi

if [[ "$USE_MOCK" == "1" ]]; then
  run_current_node npm --prefix "$ROOT/frontend" run mock:api >"$OUT_DIR/mock-api.log" 2>&1 &
  pids+=("$!")
fi

BASELINE_LOG="$OUT_DIR/cra-baseline.log"
CURRENT_LOG="$OUT_DIR/vite-current.log"

if grep -q '"start": "vite"' "$BASE_WORKTREE/frontend/package.json"; then
  BASELINE_LOG="$OUT_DIR/vite-baseline.log"
  run_with_node "$BASE_WORKTREE" npm --prefix "$BASE_WORKTREE/frontend" run start -- --port "$BASE_PORT" >"$BASELINE_LOG" 2>&1 &
else
  PORT="$BASE_PORT" run_with_node "$BASE_WORKTREE" npm --prefix "$BASE_WORKTREE/frontend" run start >"$BASELINE_LOG" 2>&1 &
fi
pids+=("$!")

run_current_node npm --prefix "$ROOT/frontend" run start -- --port "$CURRENT_PORT" >"$CURRENT_LOG" 2>&1 &
pids+=("$!")

wait_url "http://localhost:$BASE_PORT" "baseline dev server" "$BASELINE_LOG"
wait_url "http://localhost:$CURRENT_PORT" "current dev server" "$CURRENT_LOG"

# Capture failures still produce useful screenshots and diagnostics. Run both sides
# and build the report before propagating any failure to the caller.
comparison_status=0
run_comparison_step() {
  local label="$1"
  shift
  local status=0
  run_current_node npm --prefix "$ROOT/frontend" run "$@" || status=$?
  if [[ "$status" -ne 0 ]]; then
    echo "$label failed (exit $status); continuing to collect comparison diagnostics." >&2
    comparison_status=1
  fi
}

run_comparison_step "Baseline capture" visual:baseline -- --base-url "http://localhost:$BASE_PORT" --routes "$BASE_ROUTES" --out-dir "$OUT_DIR/baseline" ${clock_args[@]+"${clock_args[@]}"}
run_comparison_step "Current capture" visual:current -- --base-url "http://localhost:$CURRENT_PORT" --routes "$ROUTES" --out-dir "$OUT_DIR/current" ${clock_args[@]+"${clock_args[@]}"}
run_comparison_step "Visual diff/report" visual:diff -- \
  --baseline-dir "$OUT_DIR/baseline" --current-dir "$OUT_DIR/current" \
  --diff-dir "$OUT_DIR/diff" --side-by-side-dir "$OUT_DIR/side-by-side" \
  --report "$OUT_DIR/report.html"

echo "Comparison output: $OUT_DIR"
exit "$comparison_status"
