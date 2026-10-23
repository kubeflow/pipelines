#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0.
set -euo pipefail

: "${MINIO_BINARY:?Set MINIO_BINARY to an executable MinIO server built from the documented release}"
if [[ ! -x "$MINIO_BINARY" ]]; then
  echo "MINIO_BINARY must identify an executable MinIO server" >&2
  exit 1
fi
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
server_dir="$script_dir/../server"
test_dir="$(mktemp -d)"
seaweed_container=""
minio_pid=""
cleanup() {
  if [[ -n "$minio_pid" ]]; then
    kill "$minio_pid" 2>/dev/null || true
    wait "$minio_pid" 2>/dev/null || true
  fi
  if [[ -n "$seaweed_container" ]]; then docker rm -f "$seaweed_container" >/dev/null; fi
  rm -rf "$test_dir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

seaweed_container="$(docker run -d --rm -p 127.0.0.1::8333 --entrypoint sh \
  ghcr.io/chrislusf/seaweedfs:4.48 \
  -c 'printf "%s" '\''{"identities":[{"name":"contract","credentials":[{"accessKey":"kfp-contract","secretKey":"kfp-contract-secret"}],"actions":["Admin","Read","Write","List","Tagging"]}]}'\'' > /tmp/s3.json; exec weed server -dir=/data -s3 -s3.port=8333 -s3.config=/tmp/s3.json -volume.max=100')"
seaweed_address="$(docker port "$seaweed_container" 8333/tcp)"
minio_port="$(node -e 'const s=require("node:net").createServer(); s.listen(0,"127.0.0.1",()=>{process.stdout.write(String(s.address().port));s.close()})')"
MINIO_ROOT_USER=kfp-contract MINIO_ROOT_PASSWORD=kfp-contract-secret MINIO_BROWSER=off MINIO_SITE_REGION=contract-region \
  "$MINIO_BINARY" server --address "127.0.0.1:$minio_port" "$test_dir/data" >"$test_dir/minio.log" 2>&1 &
minio_pid=$!
export KFP_SEAWEEDFS_ENDPOINT="http://$seaweed_address"
export KFP_MINIO_ENDPOINT="http://127.0.0.1:$minio_port"
for attempt in {1..60}; do
  if curl --silent --output /dev/null "$KFP_SEAWEEDFS_ENDPOINT" && \
    curl --fail --silent --output /dev/null "$KFP_MINIO_ENDPOINT/minio/health/ready"; then
    break
  fi
  if [[ "$attempt" == 60 ]]; then
    docker logs "$seaweed_container" >&2
    cat "$test_dir/minio.log" >&2
    echo "Storage contract services failed to become ready" >&2
    exit 1
  fi
  sleep 1
done
"$MINIO_BINARY" --version
docker exec "$seaweed_container" weed version
cd "$server_dir"
node node_modules/vitest/vitest.mjs run artifact-store.contract.test.ts
