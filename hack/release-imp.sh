#!/bin/bash
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

echo "Usage: update kubeflow/pipelines/VERSION to new version tag by"
echo '`echo -n "\$VERSION" > VERSION` first, then run this script.'
echo "Please use the above command to make sure the file doesn't have extra"
echo "line endings."

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
REPO_ROOT="$DIR/.."
TAG_NAME="$(cat $REPO_ROOT/VERSION)"

if [[ -z "$TAG_NAME" ]]; then
  echo "ERROR: $REPO_ROOT/VERSION is empty" >&2
  exit 1
fi

"$DIR/check-release-needed-tools.sh"

pushd "$REPO_ROOT"
npm ci
npm run changelog
popd
# Change github issue/PR references like #123 to real urls in markdown.
# The issues must have a " " or a "(" before it to avoid already converted issues like [\#123](url...).
sed -i.bak -e 's|\([ (]\)#\([0-9]\+\)|\1[\\#\2](https://github.com/kubeflow/pipelines/issues/\2)|g' "$REPO_ROOT/CHANGELOG.md"

"$REPO_ROOT/components/release-in-place.sh" $TAG_NAME
"$REPO_ROOT/manifests/gcp_marketplace/hack/release.sh" $TAG_NAME
"$REPO_ROOT/manifests/kustomize/hack/release.sh" $TAG_NAME
"$REPO_ROOT/sdk/hack/release.sh" $TAG_NAME
"$REPO_ROOT/backend/api/generate_api.sh"
"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"
