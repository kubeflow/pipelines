#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: bash regenerate_catalogs.sh PINNED_SOURCE_CHECKOUT OUTPUT_DIR" >&2
  exit 2
fi

fixture_source=$(realpath "$1")
fixture_output=$(realpath "$2")
fixture_repo=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
fixture_data="$fixture_repo/backend/src/apiserver/migration/mlmd/testdata"
test -f "$fixture_source/go.mod"
test -f "$fixture_data/legacy_source.json"
test -d "$fixture_output"

fixture_password=$(openssl rand -hex 16)
mysql_container=
cleanup() {
  if [[ -n "$mysql_container" ]]; then podman stop "$mysql_container" >/dev/null; fi
}
trap cleanup EXIT

mysql_container=$(podman run --rm -d -p 127.0.0.1:13306:3306 \
  -e MYSQL_ROOT_PASSWORD="$fixture_password" -e MYSQL_ROOT_HOST=% \
  docker.io/library/mysql@sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b)
ready=false
for attempt in {1..60}; do
  if podman exec "$mysql_container" mysqladmin ping --protocol=tcp -h127.0.0.1 -uroot -p"$fixture_password" --silent >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo "disposable database containers did not become ready" >&2
  exit 1
fi

(
  cd "$fixture_source"
  KFP_MLMD_FIXTURE_PASSWORD="$fixture_password" go run "$fixture_data/generate_legacy_schema.go" \
    -metadata="$fixture_data/legacy_source.json" > "$fixture_output/legacy_schema_mysql.json"
)
(
  cd "$fixture_repo"
  KFP_MLMD_SCHEMA_FIXTURE_DIR="$fixture_output" \
    KFP_MLMD_CAPTURE_PARTIAL_DIR="$fixture_output" \
    KFP_MLMD_MYSQL_TEST_DSN="root:${fixture_password}@tcp(127.0.0.1:13306)/?parseTime=true" \
    go test ./backend/src/apiserver/client_manager -run '^TestMLMDLegacyStartup$' -count=1
)

echo "Captured two MySQL catalog fixtures in $fixture_output"
