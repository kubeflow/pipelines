#!/bin/bash

set -euo pipefail

# Python dependencies are managed by uv in the workspace lockfile.
cd ../..
uv lock
