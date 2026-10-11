#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# VERSION selects the upstream protobuf sources retained for archive decoding.
exec make -C "${DIR}" update
