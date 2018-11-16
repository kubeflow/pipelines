#!/usr/bin/env python3
# Copyright 2018 Google LLC
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


import kfp.dsl as dsl
import datetime


def dataflow_tf_model_analyze_op(csv_eval_path_with_predict: 'GscUri', schema: 'GcsUri[text/json]', output: 'analysis_output', step_name='analysis'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/caipe-dev/test10:frederickliu',
        arguments = [
            '--csv_eval_with_predict_path', csv_eval_path_with_predict,
            '--schema_path', schema,
            '--output', output,
        ],
    )

@dsl.pipeline(
  name='TFMA Model Agnostic Example',
  description='Example pipeline that does model analysis based on a csv file with features, predicitons, labels.'
)
def model_analysis(
    output: dsl.PipelineParam,
    project: dsl.PipelineParam,

    eval_with_predict: mlp.PipelineParam=mlp.PipelineParam(
        name='csv-eval-path-with-predict',
        value='gs://model-agnostic-test/eval_with_predict.csv'),
    schema: mlp.PipelineParam=mlp.PipelineParam(
        name='schema',
        value='gs://model-agnostic-test/schema.json')):
  analysis_output = '%s/{{workflow.name}}/analysis' % output

  analysis = dataflow_tf_model_analyze_op(eval_with_predict, schema, analysis_output)

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(model_analysis, __file__ + '.tar.gz')
