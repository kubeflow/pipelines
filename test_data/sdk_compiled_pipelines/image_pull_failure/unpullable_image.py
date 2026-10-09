# Copyright 2026 The Kubeflow Authors
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
"""A pipeline whose only task uses an image that can never be pulled.

The registry host is under the reserved ``.invalid`` TLD (RFC 2606), so the
pull fails with ErrImagePull / ImagePullBackOff on any cluster without
depending on an external registry. The persistence agent's image pull
failure handling is expected to terminate the run after its grace period.
"""

from kfp import compiler
from kfp import dsl

UNPULLABLE_IMAGE = 'registry.invalid/kubeflow/pipelines-unpullable:does-not-exist'


@dsl.container_component
def unpullable():
    return dsl.ContainerSpec(
        image=UNPULLABLE_IMAGE,
        command=['sh', '-c'],
        args=['echo this container never starts'],
    )


@dsl.pipeline(name='image-pull-failure')
def image_pull_failure():
    unpullable().set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=image_pull_failure,
        package_path=__file__.replace('.py', '.yaml'),
    )
