#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"

build_image() {
  docker build --progress=plain -t "${REGISTRY}/squid:latest" -f ${C_DIR}/Containerfile .
  docker push "${REGISTRY}/squid:latest"
}

build_image

kubectl apply -k manifests