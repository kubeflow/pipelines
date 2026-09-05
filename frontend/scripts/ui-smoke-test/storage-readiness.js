// Deployment availability only proves that the S3 listener and bucket exist, not that the
// volume server can accept data. Exercise the same bucket used by the smoke fixtures.
const STORAGE_READINESS_TIMEOUT_MS = 180000;
const STORAGE_READINESS_SCRIPT = `
set -eu
[ -n "$accesskey" ] || { echo 'Missing artifact-store access key' >&2; exit 1; }
[ -n "$secretkey" ] || { echo 'Missing artifact-store secret key' >&2; exit 1; }
url="http://127.0.0.1:8333/mlpipeline/.ui-smoke-readiness-$$"
payload="ui-smoke-storage-ready-$$"
request() {
  curl --fail --silent --show-error --connect-timeout 2 --max-time 4 \\
    --aws-sigv4 aws:amz:us-east-1:s3 --user "$accesskey:$secretkey" "$@" "$url"
}
cleanup() { request -X DELETE >/dev/null 2>&1 || true; }
trap cleanup EXIT
attempt=0
while [ "$attempt" -lt 12 ]; do
  attempt=$((attempt + 1))
  if request -X PUT --data "$payload" >/dev/null && \\
     actual=$(request) && [ "$actual" = "$payload" ] && \\
     request -X DELETE >/dev/null; then
    trap - EXIT
    exit 0
  fi
  sleep 2
done
echo 'SeaweedFS failed its S3 write/read/delete probe; check writable volumes and free disk space.' >&2
df -h /data >&2 || true
exit 1
`.trim();

module.exports = { STORAGE_READINESS_SCRIPT, STORAGE_READINESS_TIMEOUT_MS };
