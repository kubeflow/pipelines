#!/bin/sh
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

set -eu
if command -v uv >/dev/null 2>&1; then
    exec uv "$@"
fi

# Published images may predate the bundled uv binary. Keep source checkouts
# usable with those images while the updated generator image is being published.
: "${HOME:?HOME must be set for the uv bootstrap}"
python3 -m pip install --user --break-system-packages uv==0.13.5
exec "$HOME/.local/bin/uv" "$@"
