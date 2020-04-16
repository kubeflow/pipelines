#!/bin/bash

# This script is to build Kustomize files for code reviews.
# The built files is not suggested to be used for installation.

kubectl kustomize cluster-scoped-resources/ > gen/cluster-scoped-resources.yaml
kubectl kustomize env/dev/ > gen/dev.yaml
kubectl kustomize env/gcp-dev/ > gen/gcp-dev.yaml
kubectl kustomize env/gcp/ > gen/gcp.yaml