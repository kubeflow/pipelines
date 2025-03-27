#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"

kubectl apply -k ${C_DIR}/manifests
