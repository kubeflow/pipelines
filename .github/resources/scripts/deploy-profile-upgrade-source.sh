#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

source_commit=2bfc482594dc125e66746b836d49387173539b4e
source_version=2.18.1
digest_root=
if [[ $# == 0 ]]; then
  ref=$source_commit
  if ! git cat-file -e "$ref^{commit}" 2>/dev/null; then
    git fetch --no-tags --depth=1 https://github.com/kubeflow/pipelines.git "$ref"
  fi
  echo "Deploying Kubeflow Pipelines $source_version ($ref)"
elif [[ $# == 2 && $1 == --candidate && $2 == /* && -d $2 ]]; then
  ref=$(git rev-parse HEAD)
  digest_root=$2
  : "${GITHUB_REPOSITORY_OWNER:?required for candidate image digests}"
  echo "Deploying candidate $ref"
else
  echo "Usage: $0 [--candidate ABSOLUTE_DIGEST_DIRECTORY]" >&2
  exit 2
fi

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
git archive "$ref" manifests .github/resources test_data third_party/argo | tar -x -C "$stage"
cd "$stage"
overlay=.github/resources/manifests/multiuser/cache-disabled
kubectl kustomize --load-restrictor LoadRestrictionsNone "$overlay" > rendered.yaml

# The CI overlay also embeds images in ConfigMap templates and launcher/driver
# environment variables, which the standard Kustomize image transformer skips.
python3 - "$digest_root" "$source_version" "$overlay/resources.yaml" <<'PY'
import os
from pathlib import Path
import re
import sys

root, version, output = sys.argv[1:]
manifest = Path("rendered.yaml").read_text()
aliases = {
    "apiserver": "api-server",
    "scheduledworkflow": "scheduled-workflow-controller",
    "persistenceagent": "persistence-agent",
}
pattern = re.compile(r"kind-registry:5000/([a-z0-9-]+):(?:ci|latest)\b|ghcr\.io/kubeflow/kfp-(frontend):dummy\b")
refs = {}

def resolve(match):
    alias = match.group(1) or match.group(2)
    name = "kfp-" + aliases.get(alias, alias)
    if name not in refs:
        if root:
            files = list((Path(root) / f"digests-{name}-amd64").iterdir())
            if len(files) != 1 or not files[0].is_file() or not re.fullmatch("[0-9a-f]{64}", files[0].name):
                raise ValueError(f"Expected one AMD64 digest file for {name}")
            owner = os.environ["GITHUB_REPOSITORY_OWNER"].lower()
            refs[name] = f"ghcr.io/{owner}/{name}@sha256:{files[0].name}"
        else:
            refs[name] = f"ghcr.io/kubeflow/{name}:{version}"
        print(refs[name])
    return refs[name]

manifest = pattern.sub(resolve, manifest)
if "kind-registry:" in manifest or ":dummy" in manifest:
    raise ValueError("Unresolved test image in rendered multi-user overlay")
if not {"kfp-driver", "kfp-launcher", "kfp-api-server", "kfp-frontend"} <= refs.keys():
    raise ValueError("Missing required KFP image in multi-user overlay")
Path(output).write_text(manifest)
PY

# Reuse the release's own install and readiness sequence on the same cluster.
cat > "$overlay/kustomization.yaml" <<'YAML'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - resources.yaml
YAML
bash .github/resources/scripts/deploy-kfp.sh --multi-user --cache-disabled --db-type mysql
