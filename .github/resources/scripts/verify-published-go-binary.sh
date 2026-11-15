#!/bin/sh
# Verify the compiled artifact before BuildKit can export a publication image.
set -eu
binary=$1
expected_sha=$2
actual_sha=$(git rev-parse HEAD)
if [ "$actual_sha" != "$expected_sha" ]; then
  echo 'Go publication source differs from the requested commit' >&2
  exit 1
fi
# Go records UTC RFC3339 timestamps; Git supports the same format without date(1).
source_time=$(TZ=UTC git show -s --date=format-local:%Y-%m-%dT%H:%M:%SZ --format=%cd HEAD)
info=$(go version -m "$binary")
printf '%s\n' "$info"
for setting in 'vcs=git' "vcs.revision=$expected_sha" "vcs.time=$source_time" 'vcs.modified=false'; do
  if ! printf '%s\n' "$info" | grep -Fqx "$(printf '\tbuild\t%s' "$setting")"; then
    echo "Go publication binary lacks expected $setting" >&2
    exit 1
  fi
done
