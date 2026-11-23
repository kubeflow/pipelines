"""Construction of release package, workflow and container commands."""

import os
from pathlib import Path

from .github_api import REPO
from .metadata import ReleaseMetadata


def image_workflow_command(
    metadata: ReleaseMetadata,
    source_sha: str | None = None,
    workflow_ref: str | None = None,
    set_latest: bool = False,
) -> list[str]:
    """Build command to trigger image-builds-release.yml workflow.

    Args:
      metadata: Release metadata for version and branch info.

    Returns:
      Command list for gh workflow run with appropriate parameters.
    """
    return [
        'gh',
        'workflow',
        'run',
        'image-builds-release.yml',
        '--repo',
        REPO,
        '--ref',
        workflow_ref or metadata.release_branch,
        '-f',
        f'src_branch={source_sha or metadata.release_branch}',
        '-f',
        f'target_tag={metadata.tag}',
        '-f',
        'overwrite_imgs=false',
        '-f',
        f'set_latest={str(set_latest).lower()}',
        '-f',
        'add_sha_tag=true',
        '-f',
        'dry_run=false',
    ]


def sdk_workflow_command(
    metadata: ReleaseMetadata,
    packages: str = 'all',
    workflow_ref: str | None = None,
) -> list[str]:
    """Build command to trigger publish-packages.yml workflow.

    Args:
      metadata: Release metadata for tag and branch info.

    Returns:
      Command list for gh workflow run with appropriate parameters.
    """
    return [
        'gh',
        'workflow',
        'run',
        'publish-packages.yml',
        '--repo',
        REPO,
        '--ref',
        workflow_ref or metadata.release_branch,
        '-f',
        f'tag={metadata.sdk_tag}',
        '-f',
        f'packages={packages}',
        '-f',
        'dry_run=false',
    ]


def kfp_requirements_command() -> list[str]:
    """Build command for regenerating sdk/python requirements.

    Returns:
      Command list for pip-compile with local unpublished package dists.
    """
    return [
        'bash',
        '-c',
        """
GIT_ROOT="$(git rev-parse --show-toplevel)"
pip-compile --no-emit-find-links --no-header --no-emit-index-url requirements.in \
  --find-links="$GIT_ROOT/api/v2alpha1/python/dist" \
  --find-links="$GIT_ROOT/backend/api/v2beta1/python_http_client/dist" --output-file=requirements.txt
""".strip(),
    ]


def kubernetes_requirements_command() -> list[str]:
    """Build command for regenerating kubernetes_platform/python requirements.

    Returns:
      Command list for pip-compile with local unpublished package dists.
    """
    return [
        'bash',
        '-c',
        """
GIT_ROOT="$(git rev-parse --show-toplevel)"
pip-compile --no-emit-find-links --no-header --no-emit-index-url requirements.in \
  --find-links="$GIT_ROOT/sdk/python/dist" \
  --find-links="$GIT_ROOT/backend/api/v2beta1/python_http_client/dist" \
  --find-links="$GIT_ROOT/api/v2alpha1/python/dist" --output-file=requirements.txt
""".strip(),
    ]


def _release_git_mounts(root: Path) -> list[str]:
    """Expose linked-worktree metadata at the paths recorded by Git."""
    git_file = root / '.git'
    if not git_file.is_file():
        return []
    prefix = 'gitdir: '
    content = git_file.read_text().strip()
    if not content.startswith(prefix):
        raise ValueError(f'Invalid Git worktree metadata: {git_file}')
    git_dir = (root / content[len(prefix) :]).resolve()
    common_file = git_dir / 'commondir'
    common_dir = (
        (git_dir / common_file.read_text().strip()).resolve()
        if common_file.is_file()
        else git_dir
    )
    # Git's worktree backlink and common object directory use host paths.
    # Both mounts are read-only; generated files use the canonical writable mount.
    mounts = []
    for path in dict.fromkeys((common_dir, root.resolve())):
        mounts.extend(
            ['--mount', f'type=bind,source={path},target={path},readonly']
        )
    return mounts


def release_version_bump_command(
    root: Path,
    release_branch: str,
    image_tag: str | None = None,
) -> list[str]:
    """Build command that runs release version updates in the release
    container."""
    image_tag = image_tag or release_branch
    repo_path = '/go/src/github.com/kubeflow/pipelines'
    script = f"""
set -e
REPO_ROOT={repo_path}
TAG_NAME="$(cat "$REPO_ROOT/VERSION")"
if [ -z "$TAG_NAME" ]; then
  echo "ERROR: $REPO_ROOT/VERSION is empty" >&2
  exit 1
fi
normalize_node_version() {{
  printf '%s\\n' "${{1#v}}"
}}
REQUIRED_NODE_VERSION="$(normalize_node_version "$(tr -d '\\r\\n' < "$REPO_ROOT/frontend/.nvmrc")")"
echo "The following tools are needed when releasing KFP:"
echo "node==${{REQUIRED_NODE_VERSION}}"
which node >/dev/null || (echo "node not found in PATH, recommend install via https://github.com/nvm-sh/nvm#installing-and-updating" && exit 1)
ACTUAL_NODE_VERSION="$(normalize_node_version "$(node -v)")"
test "${{ACTUAL_NODE_VERSION}}" = "${{REQUIRED_NODE_VERSION}}" || (echo "node version should match ${{REQUIRED_NODE_VERSION}}" && exit 1)
echo "jq>=1.6"
which jq >/dev/null || (echo "jq not found in PATH" && exit 1)
which yq >/dev/null || (echo "yq not found in PATH" && exit 1)
# The target checkout, not the release tool, determines the manifest syntax.
if grep -Eq '^[[:space:]]*yq[[:space:]]+(w|write)[[:space:]]' "$REPO_ROOT/manifests/kustomize/hack/release.sh"; then
  yq --version | grep -E '^yq version 3[.]' || (echo "Target release scripts require mikefarah/yq version 3.x" && exit 1)
else
  yq --version | grep -E '^yq [(]https://github.com/mikefarah/yq/[)] version v4[.]' || (echo "Target release scripts require mikefarah/yq version 4.x" && exit 1)
fi
echo "java>=11"
which java >/dev/null || (echo "java not found in PATH" && exit 1)
echo "python>3"
which python3 >/dev/null || (echo "python not found in PATH" && exit 1)
python3 -c "import setuptools" || (echo "setuptools should be installed in python" && exit 1)
echo "go"
which go >/dev/null || (echo "go not found in PATH" && exit 1)
go_path=$(go env GOPATH)
echo "$PATH" | grep "${{go_path}}/bin" >/dev/null || (echo "\\$GOPATH/bin: ${{go_path}}/bin should be in PATH" && exit 1)
echo "All tools installed"
cd "$REPO_ROOT"
"$REPO_ROOT/manifests/kustomize/hack/release.sh" "$TAG_NAME"
if compgen -G "$REPO_ROOT/backend/api/v1beta1/*.proto" >/dev/null; then
  API_VERSION=v1beta1 "$REPO_ROOT/backend/api/hack/generator.sh"
fi
if [ -f "$REPO_ROOT/backend/api/v2/run.proto" ] && \\
    grep -Eq '^[[:space:]]*package[[:space:]]+kubeflow[.]pipelines[.]backend[.]api[.]v2[[:space:]]*;' "$REPO_ROOT/backend/api/v2/run.proto"; then
  export API_VERSION=v2
else
  export API_VERSION=v2beta1
fi
"$REPO_ROOT/backend/api/hack/generator.sh"
"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"
if [ -f "$REPO_ROOT/api/Makefile" ]; then
  make -C "$REPO_ROOT/api" generate
fi
if [ -f "$REPO_ROOT/kubernetes_platform/Makefile" ]; then
  make -C "$REPO_ROOT/kubernetes_platform" generate-go-in-container
fi
""".strip()
    return [
        'docker',
        'run',
        '--rm',
        '--user',
        f'{os.getuid()}:{os.getgid()}',
        '--mount',
        f'type=bind,source={root},target={repo_path}',
        *_release_git_mounts(root),
        f'ghcr.io/kubeflow/kfp-release:{image_tag}',
        '/bin/bash',
        '-c',
        script,
    ]


def kfp_kubernetes_docs_build_command(
    root: Path, release_branch: str
) -> list[str]:
    """Build command that regenerates the kfp-kubernetes Python package without
    Make."""
    repo_path = '/go/src/github.com/kubeflow/pipelines'
    script = f'''
set -e
cd "{repo_path}"
# Legacy release checkouts do not contain backend/api/hack/uv.sh.
UV=uv
if ! command -v "$UV" >/dev/null 2>&1; then
  python3 -m pip install --user --break-system-packages uv==0.13.5
  UV="$HOME/.local/bin/uv"
fi
rm -rf kubernetes_platform/python/build
rm -rf kubernetes_platform/python/dist
rm -rf kubernetes_platform/python/kfp_kubernetes.egg-info
rm -f kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py
rm -rf /tmp/protobuf-src
mkdir -p api/v2alpha1/google/rpc
wget -qO api/v2alpha1/google/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/fecd7d35f46753b45bf4519f6342495a181740c9/google/rpc/status.proto
git clone --depth 1 --branch v26.0 https://github.com/protocolbuffers/protobuf.git /tmp/protobuf-src
mkdir -p api/v2alpha1/google/protobuf
cp /tmp/protobuf-src/src/google/protobuf/*.proto api/v2alpha1/google/protobuf/
cd "{repo_path}/kubernetes_platform/python"
python3 generate_proto.py
if [ -f pyproject.toml ]; then
  "$UV" build --package kfp-kubernetes --out-dir dist
else
  "$UV" build --out-dir dist .
fi
'''.strip()
    return [
        'docker',
        'run',
        '--rm',
        '--user',
        f'{os.getuid()}:{os.getgid()}',
        '-e',
        'HOME=/tmp',
        '--mount',
        f'type=bind,source={root},target={repo_path}',
        f'ghcr.io/kubeflow/kfp-api-generator:{release_branch}',
        '/bin/bash',
        '-c',
        script,
    ]
