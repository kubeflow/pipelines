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

from kfp.v2 import dsl
from kfp.v2.dsl import Output, SlicedClassificationMetrics
import kfp.v2.compiler as compiler


@dsl.component
def simple_metric_op(sliced_eval_metrics: Output[SlicedClassificationMetrics]):
    sliced_eval_metrics._sliced_metrics = {}
    sliced_eval_metrics.load_confusion_matrix(
        'a slice', categories=['cat1', 'cat2'], matrix=[[1, 0], [2, 4]])


@dsl.pipeline(name='test-sliced-metric')
def my_pipeline():
    m_op = simple_metric_op()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))
