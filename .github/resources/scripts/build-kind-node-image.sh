#!/usr/bin/env bash

set -euo pipefail

version=${1:?Usage: build-kind-node-image.sh vX.Y.Z}
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Expected a Kubernetes release version such as v1.34.13" >&2
  exit 1
fi
case "$(uname -m)" in
  x86_64) architecture=amd64 ;;
  aarch64|arm64) architecture=arm64 ;;
  *) echo "Unsupported host architecture" >&2; exit 1 ;;
esac

download_dir=$(mktemp -d)
trap 'rm -rf "$download_dir"' EXIT
archive="kubernetes-server-linux-${architecture}.tar.gz"
url="https://dl.k8s.io/${version}/${archive}"
curl --fail --location --retry 3 --retry-all-errors --connect-timeout 30 --max-time 600 \
  "$url" -o "$download_dir/$archive"
curl --fail --location --retry 3 --retry-all-errors --connect-timeout 30 --max-time 60 \
  "$url.sha512" -o "$download_dir/$archive.sha512"
(
  cd "$download_dir"
  printf '%s  %s\n' "$(cat "$archive.sha512")" "$archive" | shasum -a 512 --check --status
)

# Package the released binaries and control-plane images; do not compile Kubernetes.
kind build node-image --type file "$download_dir/$archive" --image "kindest/node:$version"
