#!/usr/bin/env bash
# This script ensures that all of our code adheres to the black formatting
# standard.

set -e

# Change to the component base directory (assuming we are in unit_tests)
pushd ../../

black --check .

popd

# Change to the samples directory (assuming we are in unit_tests)

pushd ../../../../../samples/contrib/aws-samples/

black --check .

popd