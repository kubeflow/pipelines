#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

source_version=$(gh api --paginate 'repos/kubeflow/pipelines/releases?per_page=100' |
  jq -ser --arg target '' -f .github/resources/scripts/profile-upgrade-source.jq)
git fetch --no-tags --depth=1 https://github.com/kubeflow/pipelines.git "refs/tags/$source_version"
source_commit=$(git rev-parse 'FETCH_HEAD^{commit}')
echo "Deploying Kubeflow Pipelines $source_version ($source_commit)"

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
git archive "$source_commit" manifests .github/resources test_data third_party/argo | tar -x -C "$stage"
cd "$stage"
overlay=.github/resources/manifests/multiuser/cache-disabled

# Rewrite the frontend override before rendering so Kustomize uses the
# actual image in the immutable signing-key Job's ConfigMap hash. After
# rendering, resolve the remaining images, including images in env values.
python3 - "$source_version" "$overlay" <<'PY'
from pathlib import Path
import re
import subprocess
import sys

version, overlay = sys.argv[1:]
aliases = {
    "apiserver": "api-server",
    "scheduledworkflow": "scheduled-workflow-controller",
    "persistenceagent": "persistence-agent",
}
pattern = re.compile(r"kind-registry:5000/([a-z0-9-]+):(?:ci|latest)\b|ghcr\.io/kubeflow/kfp-(frontend):dummy\b")
refs = {}

def image(name):
    if name not in refs:
        refs[name] = f"ghcr.io/kubeflow/{name}:{version}"
        print(refs[name])
    return refs[name]

frontend = image("kfp-frontend")
image_name, tag = frontend.rsplit(":", 1)
override = f"    newName: {image_name}\n    newTag: {tag}"
config = Path(overlay) / "kustomization.yaml"
text, count = re.subn(
    r"(?<=  - name: ghcr\.io/kubeflow/kfp-frontend\n)"
    r"    newName: kind-registry:5000/frontend\n    newTag: latest",
    lambda _: override,
    config.read_text(),
)
if count != 1:
    raise ValueError("Expected one frontend image override in the multi-user CI overlay")
config.write_text(text)

manifest = subprocess.check_output(
    ["kubectl", "kustomize", "--load-restrictor", "LoadRestrictionsNone", overlay],
    text=True,
)
def resolve(match):
    alias = match.group(1) or match.group(2)
    return image("kfp-" + aliases.get(alias, alias))

manifest = pattern.sub(resolve, manifest)
if "kind-registry:" in manifest or ":dummy" in manifest:
    raise ValueError("Unresolved test image in rendered multi-user overlay")
if not {"kfp-driver", "kfp-launcher", "kfp-api-server", "kfp-frontend"} <= refs.keys():
    raise ValueError("Missing required KFP image in multi-user overlay")
(Path(overlay) / "resources.yaml").write_text(manifest)
PY

# Reuse the release's own install and readiness sequence on the same cluster.
cat > "$overlay/kustomization.yaml" <<'YAML'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - resources.yaml
YAML
bash .github/resources/scripts/deploy-kfp.sh --multi-user --cache-disabled --db-type mysql
