# Copyright 2019 Google LLC
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

from kfp import dsl


@dsl.pipeline(
    name='foo', description='A pipeline with custom output artifacts')
def foo_pipeline(bucket: str = 'foobucket'):
    """A pipeline with custom output artifacts"""

    # this op will upload metadata & metrics to input-variable:`bucket`
    op = dsl.ContainerOp("foo", 
                         image="busybox:latest", 
                         output_artifacts={'mlpipeline-metrics': '/foo-bar.json'})

    op.add_output_artifact(name='results', path='/results.csv')
    op.add_output_artifacts({'logs': '/logs'})
