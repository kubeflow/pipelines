#!/usr/bin/env bash
set -euo pipefail

# Build venv with required packages
VENV=".venv"
PYTHON_VENV="${VENV}/bin/python"
python -m venv $VENV
$PYTHON_VENV -m pip install -U pip
$PYTHON_VENV -m pip install --require-hashes -r requirements.txt
$PYTHON_VENV -m pip install pytest requests
$PYTHON_VENV runtime_smoke.py sync.py

# Run tests
AWS_EC2_METADATA_DISABLED=true $PYTHON_VENV -m pytest ./test_sync.py
