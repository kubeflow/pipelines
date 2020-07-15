#!/usr/bin/env bash
# This script ensures that all of our code adheres to the black formatting
# standard.

# Change to the component base directory (assuming we are in unit_tests)
pushd ../../

black --check .

popd