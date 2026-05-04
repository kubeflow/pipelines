#!/usr/bin/env bash
# Verifies that all Dockerfiles using a golang base image
# specify a Go version consistent with the root go.mod.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

GOMOD_VERSION=$(grep -E '^go [0-9]' "$REPO_ROOT/go.mod" | awk '{print $2}' || true)

if [[ -z "$GOMOD_VERSION" ]]; then
    echo "ERROR: Could not extract Go version from go.mod" >&2
    exit 1
fi

echo "go.mod Go version: $GOMOD_VERSION"

ERRORS=0
CHECKED=0
FOUND=0

while IFS= read -r dockerfile; do
    relative="${dockerfile#"$REPO_ROOT"/}"
    FOUND=$((FOUND + 1))
    while IFS= read -r line; do
        docker_version=$(echo "$line" | sed -E 's/.*FROM[[:space:]]+(--[^[:space:]]+[[:space:]]+)*golang:([0-9]+\.[0-9]+(\.[0-9]+)?).*/\2/')

        if [[ ! "$docker_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
            echo "ERROR: Could not parse Go version from line in $relative: $line" >&2
            ERRORS=$((ERRORS + 1))
            continue
        fi

        CHECKED=$((CHECKED + 1))

        if [[ "$docker_version" != "$GOMOD_VERSION" ]]; then
            echo "MISMATCH: $relative has Go $docker_version, but go.mod requires $GOMOD_VERSION" >&2
            ERRORS=$((ERRORS + 1))
        else
            echo "  OK: $relative (Go $docker_version)"
        fi
    done < <(grep -iE '^FROM[[:space:]]+(--[^[:space:]]+[[:space:]]+)*golang:' "$dockerfile" || true)
done < <(cd "$REPO_ROOT" && (git ls-files '*Dockerfile*' | xargs grep -liE 'FROM[[:space:]]+(--[^[:space:]]+[[:space:]]+)*golang:' | sed "s|^|$REPO_ROOT/|") || true)

echo ""

if [[ $FOUND -eq 0 ]]; then
    echo "ERROR: No Dockerfiles with 'FROM golang:' found." >&2
    exit 1
fi

if [[ $CHECKED -eq 0 ]]; then
    echo "ERROR: Found $FOUND Dockerfile(s) with 'FROM golang:', but could not parse any Go version." >&2
    exit 1
fi

if [[ $ERRORS -gt 0 ]]; then
    echo "FAILED: $ERRORS error(s) found when checking 'FROM golang:' stages against go.mod ($GOMOD_VERSION)." >&2
    exit 1
fi

echo "PASSED: All $CHECKED 'FROM golang:' stage(s) use Go $GOMOD_VERSION, matching go.mod."
