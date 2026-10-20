#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
# Requires Docker, openssl, frontend/server dependencies and boto3. All identities,
# signing keys, bucket data, and containers are disposable; no AWS account is used.
"${PYTHON:-python3}" -m unittest discover -s "$repo_root/test/artifact-proxy" -p test_external_storage_acceptance.py
"${PYTHON:-python3}" -m unittest discover -s "$repo_root/test/seaweedfs" -p external_storage_fixture_test.py
"${PYTHON:-python3}" "$repo_root/test/seaweedfs/external_storage_fixture.py" "$@"
