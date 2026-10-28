// Deployment availability only proves that the S3 listener and bucket exist, not that the
// volume server can accept data. Both the fixture bucket and filer collection need space.
const STORAGE_READINESS_TIMEOUT_MS = 360000;
const STORAGE_READINESS_SCRIPT = `
set -eu
[ -n "$accesskey" ] || { echo 'Missing artifact-store access key' >&2; exit 1; }
[ -n "$secretkey" ] || { echo 'Missing artifact-store secret key' >&2; exit 1; }
payload="ui-smoke-storage-ready-$$"
request() {
  if [ "$port" = 8333 ]; then
    set -- --aws-sigv4 aws:amz:us-east-1:s3 --user "$accesskey:$secretkey" "$@"
  fi
  curl --fail --silent --show-error --connect-timeout 2 --max-time 4 \\
    "$@" "$url"
}
cleanup() { request -X DELETE >/dev/null 2>&1 || true; }
for port in 8333 8888; do
  if [ "$port" = 8333 ]; then prefix=mlpipeline/; else prefix=; fi
  url="http://127.0.0.1:$port/$prefix.ui-smoke-readiness-$$"
  trap cleanup EXIT
  attempt=0
  ready=false
  while [ "$attempt" -lt 12 ]; do
    attempt=$((attempt + 1))
    if request -X PUT --data "$payload" >/dev/null && \\
       actual=$(request) && [ "$actual" = "$payload" ] && \\
       request -X DELETE >/dev/null; then
      ready=true
      break
    fi
    sleep 2
  done
  if [ "$ready" != true ]; then
    echo "SeaweedFS port $port failed write/read/delete; check writable volumes and free disk space." >&2
    df -h /data >&2 || true
    exit 1
  fi
  trap - EXIT
done
`.trim();

module.exports = { STORAGE_READINESS_SCRIPT, STORAGE_READINESS_TIMEOUT_MS };
