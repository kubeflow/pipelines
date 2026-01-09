# Optional developer convenience commands for Kubeflow Pipelines
#
# These recipes are simple wrappers around existing Make targets.
# They are entirely optional and are not used by CI.

# Use a POSIX shell for consistency across platforms
set shell := ["sh", "-cu"]

# Default: show available recipes
default:
    @echo "Available recipes (run 'just <name>'):" \
    && just --list

# ---- High-level commands ----

# NOTE:
# - There is intentionally no generic `build` or `test` recipe.
# - The backend `all` target builds multiple Docker images and must not be
#   wired to a generic `just build`.

# Explicitly build all backend container images (backend/Makefile: all -> image_all)
backend-images:
    make -C backend all

# Backend Go unit tests for the v2 engine (backend/src/v2/Makefile: test)
backend-test:
    make -C backend/src/v2 test

# ---- Backend helpers ----

# Create a local standalone Kind cluster with KFP deployed
kind-standalone:
    make -C backend kind-cluster-agnostic

# Create a local Kind cluster for API server development
kind-dev:
    make -C backend dev-kind-cluster

# Backend lint and format (backend/Makefile)
backend-lint:
    make -C backend lint

backend-format:
    make -C backend format

backend-lint-format:
    make -C backend lint-and-format

# Backend integration webhook tests
backend-webhook-test:
    make -C backend/test/integration test-webhook

# ---- SDK / API / platform helpers ----

# Build API v2alpha1 Go and Python artifacts (api/Makefile: all)
api-protos:
    make -C api all

# Build Kubernetes platform Go and Python artifacts (kubernetes_platform/Makefile: all)
kubernetes-platform-protos:
    make -C kubernetes_platform all

# Build SDK Python distribution artifacts (sdk/Makefile: python)
sdk-build:
    make -C sdk python

# ---- Repo-wide tooling ----

# Install Ginkgo test runner into ./bin (root Makefile)
ginkgo:
    make ginkgo

# Check that generated files are up to date (root Makefile)
check-diff:
    make check-diff
