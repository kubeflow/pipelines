// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as TWO_STEP_PIPELINE from 'src/data/test/mock_lightweight_python_functions_v2_pipeline.json';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { ml_pipelines } from 'src/generated/pipeline_spec/pbjs_ml_pipelines';
import { testBestPractices } from 'src/TestUtils';
import { convertFlowElements } from './StaticFlow';

testBestPractices();
describe('StaticFlow', () => {
  it('converts simple pipeline with element ids to graph', () => {
    const jsonObject = TWO_STEP_PIPELINE;

    const message = ml_pipelines.PipelineSpec.fromObject(jsonObject);
    const buffer = ml_pipelines.PipelineSpec.encode(message).finish();
    const pipelineSpec = PipelineSpec.deserializeBinary(buffer);

    const graph = convertFlowElements(pipelineSpec);
    // If the static flow logic gets update, inspect result with the console log result below.
    console.log(graph);
    for (let element of graph) {
      const index = [
        'task.preprocess',
        'task.train',
        'artifact.preprocess.output_dataset_one',
        'artifact.preprocess.output_dataset_two_path',
        'artifact.train.model',
        'outedge.preprocess.output_dataset_one',
        'inedge.output_dataset_two_path.train',
        'outedge.preprocess.output_dataset_two_path',
        'inedge.output_dataset_one.train',
        'outedge.train.model',
        'paramedge.preprocess.train',
      ].findIndex(x => x === element.id);
      expect(index > -1).toBeTruthy();
    }
  });
});
