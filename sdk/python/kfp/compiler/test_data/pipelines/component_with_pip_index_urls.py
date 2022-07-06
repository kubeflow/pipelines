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

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component(
    pip_index_urls=['https://pypi.org/simple'], packages_to_install=['yapf'])
def component_op():
    import yapf
    print(dir(yapf))


@dsl.pipeline(name='v2-component-pip-index-urls')
def pipeline():
    component_op()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
