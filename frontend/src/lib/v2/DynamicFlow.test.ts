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
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';
import { getNodeMlmdInfo, TASK_NAME_KEY, updateFlowElementsState } from './DynamicFlow';
import { convertFlowElements, NodeTypeNames } from './StaticFlow';
import { Node } from 'react-flow-renderer';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { Step } from '@material-ui/core';

describe('DynamicFlow', () => {
  describe('updateFlowElementsState', () => {
    it('update node status based on MLMD', () => {
      // Prepare MLMD objects.
      const EXECUTION_PREPROCESS = new Execution()
        .setId(3)
        .setLastKnownState(Execution.State.COMPLETE);
      EXECUTION_PREPROCESS.getCustomPropertiesMap().set(
        TASK_NAME_KEY,
        new Value().setStringValue('preprocess'),
      );
      const EXECUTION_TRAIN = new Execution().setId(4).setLastKnownState(Execution.State.FAILED);
      EXECUTION_TRAIN.getCustomPropertiesMap().set(
        TASK_NAME_KEY,
        new Value().setStringValue('train'),
      );

      const ARTIFACT_OUTPUT_DATA_ONE = new Artifact().setId(1).setState(Artifact.State.LIVE);
      const ARTIFACT_OUTPUT_DATA_TWO = new Artifact().setId(2).setState(Artifact.State.PENDING);
      const ARTIFACT_MODEL = new Artifact().setId(3).setState(Artifact.State.DELETED);

      const EVENT_PREPROCESS_OUTPUT_DATA_ONE = new Event()
        .setExecutionId(3)
        .setArtifactId(1)
        .setType(Event.Type.OUTPUT)
        .setPath(
          new Event.Path().setStepsList([new Event.Path.Step().setKey('output_dataset_one')]),
        );
      const EVENT_PREPROCESS_OUTPUT_DATA_TWO = new Event()
        .setExecutionId(3)
        .setArtifactId(2)
        .setType(Event.Type.OUTPUT)
        .setPath(
          new Event.Path().setStepsList([new Event.Path.Step().setKey('output_dataset_two_path')]),
        );
      const EVENT_OUTPUT_DATA_ONE_TRAIN = new Event().setExecutionId(4).setArtifactId(1);
      const EVENT_OUTPUT_DATA_TWO_TRAIN = new Event().setExecutionId(4).setArtifactId(2);
      const EVENT_TRAIN_MODEL = new Event()
        .setExecutionId(4)
        .setArtifactId(3)
        .setType(Event.Type.OUTPUT)
        .setPath(new Event.Path().setStepsList([new Event.Path.Step().setKey('model')]));

      // Converts to static graph first, its type is Elements<any>.
      const jsonObject = TWO_STEP_PIPELINE;
      const message = ml_pipelines.PipelineSpec.fromObject(jsonObject);
      const buffer = ml_pipelines.PipelineSpec.encode(message).finish();
      const pipelineSpec = PipelineSpec.deserializeBinary(buffer);
      const graph = convertFlowElements(pipelineSpec);

      // MLMD objects to provide node states.
      const executions: Execution[] = [EXECUTION_PREPROCESS, EXECUTION_TRAIN];
      const events: Event[] = [
        EVENT_PREPROCESS_OUTPUT_DATA_ONE,
        EVENT_PREPROCESS_OUTPUT_DATA_TWO,
        EVENT_OUTPUT_DATA_ONE_TRAIN,
        EVENT_OUTPUT_DATA_TWO_TRAIN,
        EVENT_TRAIN_MODEL,
      ];
      const artifacts: Artifact[] = [
        ARTIFACT_OUTPUT_DATA_ONE,
        ARTIFACT_OUTPUT_DATA_TWO,
        ARTIFACT_MODEL,
      ];

      updateFlowElementsState(graph, executions, events, artifacts);
      for (let element of graph) {
        graph
          .filter(e => e.id === element.id)
          .forEach(e => {
            if (e.id === 'task.preprocess') {
              expect(e.data.state).toEqual(EXECUTION_PREPROCESS.getLastKnownState());
            } else if (e.id === 'task.train') {
              expect(e.data.state).toEqual(EXECUTION_TRAIN.getLastKnownState());
            } else if (e.id === 'artifact.preprocess.output_dataset_one') {
              expect(e.data.state).toEqual(ARTIFACT_OUTPUT_DATA_ONE.getState());
            } else if (e.id === 'artifact.preprocess.output_dataset_two_path') {
              expect(e.data.state).toEqual(ARTIFACT_OUTPUT_DATA_TWO.getState());
            } else if (e.id === 'artifact.train.model') {
              expect(e.data.state).toEqual(ARTIFACT_MODEL.getState());
            }
          });
      }
    });
  });

  describe('getNodeMlmdInfo', () => {
    it('execution not exist', () => {
      const elem: Node<FlowElementDataBase> = {
        id: 'task.exec',
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      const nodeMlmdInfo = getNodeMlmdInfo(elem, [], [], []);
      expect(nodeMlmdInfo).toEqual({});
    });

    it('execution found', () => {
      const elem: Node<FlowElementDataBase> = {
        id: 'task.exec',
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      const execution = new Execution();
      execution.setId(1);
      execution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue('exec'));
      const nodeMlmdInfo = getNodeMlmdInfo(elem, [execution], [], []);
      expect(nodeMlmdInfo).toEqual({ execution });
    });

    it('artifact not exist', () => {
      const elem: Node<FlowElementDataBase> = {
        id: 'artifact.exec.arti',
        type: NodeTypeNames.ARTIFACT,
        position: { x: 1, y: 2 },
      };

      const nodeMlmdInfo = getNodeMlmdInfo(elem, [], [], []);
      expect(nodeMlmdInfo).toEqual({});
    });

    it('artifact found', () => {
      const elem: Node<FlowElementDataBase> = {
        id: 'artifact.exec.arti',
        type: NodeTypeNames.ARTIFACT,
        position: { x: 1, y: 2 },
      };

      const execution = new Execution();
      execution.setId(1);
      execution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue('exec'));

      const artifact = new Artifact();
      artifact.setId(2);

      const event = new Event();
      event.setExecutionId(1);
      event.setArtifactId(2);
      event.setType(Event.Type.OUTPUT);
      event.setPath(new Event.Path().setStepsList([new Event.Path.Step().setKey('arti')]));

      const nodeMlmdInfo = getNodeMlmdInfo(elem, [execution], [event], [artifact]);
      expect(nodeMlmdInfo).toEqual({ execution, linkedArtifact: { event, artifact } });
    });
  });
});
