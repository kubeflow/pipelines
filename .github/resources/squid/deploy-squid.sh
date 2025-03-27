#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"

build_image() {
  docker build --progress=plain -t "registry.domain.local/squid:test" -f ${C_DIR}/Containerfile .
  kind --name kfp load docker-image registry.domain.local/squid:test
}

build_image

kubectl apply -k manifests