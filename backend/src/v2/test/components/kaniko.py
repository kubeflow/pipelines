# Copyright 2021 The Kubeflow Authors
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

from pathlib import Path

from kfp import compiler
from kfp import dsl

SHELL_COMMAND = r'''dockerfile="$0"
context_uri="$1"
context_artifact_path="$2"
context_sub_path="$3"
destination="$4"
digest_output_path="$5"
cache="$6"
cache_ttl="$7"
context=""
if [[ "${context_uri}" != "" ]]; then
  context="${context_uri}"
else
  context="dir://${context_artifact_path}"
fi
digest_file="$(mktemp)"
# Reference: https://github.com/GoogleContainerTools/kaniko
# snapshotMode and use-new-run added because: https://github.com/GoogleContainerTools/kaniko/issues/1333
/kaniko/executor \
    --dockerfile "${dockerfile}" \
    --context "${context}" \
    ${context_sub_path:+ --context-sub-path "${context_sub_path}"} \
    --destination "${destination}" \
    --digest-file "${digest_file}" \
    --cache="${cache}" \
    --cache-ttl="${cache_ttl}"
mkdir -p "$(dirname "${digest_output_path}")"
digest="$(cat $digest_file)"
echo "${destination}@${digest}" > "${digest_output_path}"
'''


@dsl.container_component
def kaniko(
    dockerfile: str,
    destination: str,
    digest: dsl.Output[dsl.Artifact],
    context_uri: str = '',
    context_artifact: dsl.Input[dsl.Artifact] = None,
    context_sub_path: str = '',
    cache: str = 'true',
    cache_ttl: str = '24h',
):
    """Build an image from context_uri or context_artifact and write its
    URI/digest."""
    # The debug image supplies sh; this pin avoids known later-version OOMs.
    # https://github.com/GoogleContainerTools/kaniko/issues/1680
    return dsl.ContainerSpec(
        image='gcr.io/kaniko-project/executor:v1.3.0-debug',
        command=[
            'sh',
            '-exc',
            SHELL_COMMAND,
            dockerfile,
            context_uri,
            dsl.IfPresentPlaceholder(
                input_name='context_artifact',
                then=context_artifact.path,
                else_='',
            ),
            context_sub_path,
            destination,
            digest.path,
            cache,
            cache_ttl,
        ],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        kaniko, package_path=str(Path(__file__).with_suffix('.yaml')))
