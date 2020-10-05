#!/usr/bin/env bash
# This script is run by our automated unit test CodeBuild project and does all
# unit testing, linting and formatting checks required to pass tests.

set -e

./run_unit_tests.sh
./check_formatting.sh
./check_generated_specifications.sh