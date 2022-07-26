#!/bin/bash
#
# Copyright 2022 The Kubeflow Authors
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

# creates a GitHub release _draft_ of the KFP SDK using the current git hash
# finish the process using the GitHub UI to publish the draft

# example usage:
# > ./sdk/hack/github_release.sh 1.8.13

set -ex

TAG_NAME=$1
TARGET_HASH=$(git rev-parse HEAD)

# substrings indicating pre-release status
ALPHA="a"
BETA="b"
RC="rc"
echo $BETA
if [[ "$TAG_NAME" == *"$ALPHA"* ]] || [[ "$TAG_NAME" == *"$BETA"* ]] || [[ "$TAG_NAME" == *"$RC"* ]]; then
  PRE_STATUS="--prerelease"
else
  PRE_STATUS=""
fi


TITLE="KFP SDK v${TAG_NAME}"
RELEASE_NOTES_LINK="https://github.com/kubeflow/pipelines/blob/${TARGET_HASH}/sdk/RELEASE.md"
REPO=kubeflow/pipelines
NOTES="Release of the KFP SDK **only**."$'\n'$'\n'"To install the KFP SDK:"$'\n'"\`\`\`bash"$'\n'"pip install kfp==${TAG_NAME}"$'\n'"\`\`\`"$'\n'"For changelog, see [release notes](${RELEASE_NOTES_LINK})."

gh release create $TAG_NAME --target $TARGET_HASH --repo $REPO --title "${TITLE}" --notes "${NOTES}" $PRE_STATUS --draft
