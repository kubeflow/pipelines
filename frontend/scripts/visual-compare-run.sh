#!/usr/bin/env bash
set -euo pipefail

DEFAULT_BASE_COMMIT="dbc2319f4"
DEFAULT_NODE_VERSION="${DEFAULT_NODE_VERSION:-22.19.0}"
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
  exit 1
fi

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BASE_WORKTREE="${BASE_WORKTREE:-$ROOT/../pipelines-vite-baseline}"
BASE_PORT="${BASE_PORT:-3010}"
CURRENT_PORT="${CURRENT_PORT:-3020}"
MOCK_PORT="${MOCK_PORT:-3001}"
USE_MOCK="${USE_MOCK:-1}"
SKIP_INSTALL="${SKIP_INSTALL:-0}"
ROUTES="${ROUTES:-$ROOT/frontend/scripts/visual-compare.routes.json}"
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
    cat "$dir/frontend/.nvmrc"
  else
    echo "$DEFAULT_NODE_VERSION"
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
  version="$(resolve_node_version "$dir")"
  ensure_node_version "$version"
  local bin_dir
  bin_dir="$(node_bin_dir "$version")"
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

mkdir -p "$OUT_DIR"

if [[ ! -e "$BASE_WORKTREE/.git" ]]; then
  git -C "$ROOT" worktree add "$BASE_WORKTREE" "$BASE_COMMIT"
else
  git -C "$BASE_WORKTREE" checkout "$BASE_COMMIT"
fi

if [[ "$SKIP_INSTALL" != "1" ]]; then
  run_with_node "$ROOT" npm --prefix "$ROOT/frontend" ci
  run_with_node "$BASE_WORKTREE" npm --prefix "$BASE_WORKTREE/frontend" ci
fi

if [[ "$USE_MOCK" == "1" ]]; then
  run_with_node "$ROOT" npm --prefix "$ROOT/frontend" run mock:api >"$OUT_DIR/mock-api.log" 2>&1 &
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

run_with_node "$ROOT" npm --prefix "$ROOT/frontend" run start -- --port "$CURRENT_PORT" >"$CURRENT_LOG" 2>&1 &
pids+=("$!")

wait_url "http://localhost:$BASE_PORT" "baseline dev server" "$BASELINE_LOG"
wait_url "http://localhost:$CURRENT_PORT" "current dev server" "$CURRENT_LOG"

run_with_node "$ROOT" npm --prefix "$ROOT/frontend" run visual:baseline -- --base-url "http://localhost:$BASE_PORT" --routes "$ROUTES"
run_with_node "$ROOT" npm --prefix "$ROOT/frontend" run visual:current -- --base-url "http://localhost:$CURRENT_PORT" --routes "$ROUTES"
run_with_node "$ROOT" npm --prefix "$ROOT/frontend" run visual:diff

echo "Side-by-side PNGs: $ROOT/frontend/.visual/side-by-side"
