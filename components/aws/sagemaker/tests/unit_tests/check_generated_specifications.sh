#!/usr/bin/env bash
# This script ensures that all of our generated specifications are up to date.

set -e

# Change to the component base directory (assuming we are in unit_tests)
pushd ../../

# Image and tag will be ignored when comparing specifications
PYTHONPATH=. ./common/generate_components.py --tag arbitrary-tag --check True

exit_status=$?
test $exit_status -eq 0 && echo "🙌  Generated specifications matched existing specification files" || exit 1

popd