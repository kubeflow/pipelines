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

import { Node } from '@xyflow/react';
import { ArtifactFlowElementData, FlowElementDataBase } from 'src/components/graph/Constants';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';
import {
  getNodeMlmdInfo,
  PARENT_DAG_ID_KEY,
  TASK_NAME_KEY,
  updateFlowElementsState,
} from './DynamicFlow';
import { convertFlowElements, getTaskKeyFromNodeKey, NodeTypeNames } from './StaticFlow';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { load } from 'js-yaml';

describe('DynamicFlow', () => {
  describe('updateFlowElementsState', () => {
    it('update node status based on MLMD', () => {
      // Prepare MLMD objects.
      const EXECUTION_ROOT = new Execution().setId(2).setLastKnownState(Execution.State.COMPLETE);
      EXECUTION_ROOT.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue(''));
      const EXECUTION_PREPROCESS = new Execution()
        .setId(3)
        .setLastKnownState(Execution.State.COMPLETE);
      EXECUTION_PREPROCESS.getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue('preprocess'))
        .set(PARENT_DAG_ID_KEY, new Value().setIntValue(2));
      const EXECUTION_TRAIN = new Execution().setId(4).setLastKnownState(Execution.State.FAILED);
      EXECUTION_TRAIN.getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue('train'))
        .set(PARENT_DAG_ID_KEY, new Value().setIntValue(2));

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
      const yamlObject = load(v2YamlTemplateString);
      const pipelineSpec = PipelineSpec.fromJSON(yamlObject);
      const graph = convertFlowElements(pipelineSpec);

      // MLMD objects to provide node states.
      const executions: Execution[] = [EXECUTION_ROOT, EXECUTION_PREPROCESS, EXECUTION_TRAIN];
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

      const runtimeGraph = updateFlowElementsState(['root'], graph, executions, events, artifacts);
      for (let element of runtimeGraph) {
        runtimeGraph
          .filter((e) => e.id === element.id)
          .forEach((e) => {
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

    it('does not preserve React Flow hidden flags when applying MLMD state', () => {
      const rootExecution = new Execution().setId(2).setLastKnownState(Execution.State.COMPLETE);
      rootExecution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue(''));

      const preprocessExecution = new Execution()
        .setId(3)
        .setLastKnownState(Execution.State.COMPLETE);
      preprocessExecution
        .getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue('preprocess'))
        .set(PARENT_DAG_ID_KEY, new Value().setIntValue(2));

      const yamlObject = load(v2YamlTemplateString);
      const pipelineSpec = PipelineSpec.fromJSON(yamlObject);
      const graph = convertFlowElements(pipelineSpec);
      const preprocessNode = graph.find((element) => element.id === 'task.preprocess') as Node;
      (preprocessNode as Node & { hidden?: boolean }).hidden = true;
      preprocessNode.measured = { width: 123, height: 45 };

      const runtimeGraph = updateFlowElementsState(
        ['root'],
        graph,
        [rootExecution, preprocessExecution],
        [],
        [],
      );
      const updatedPreprocessNode = runtimeGraph.find(
        (element) => element.id === 'task.preprocess',
      ) as Node & { hidden?: boolean };

      expect(updatedPreprocessNode.hidden).toBeUndefined();
      expect(updatedPreprocessNode.measured).toEqual({ width: 123, height: 45 });
      expect(updatedPreprocessNode.data?.state).toEqual(preprocessExecution.getLastKnownState());
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
        data: {
          mlmdId: 1,
        },
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      const execution = new Execution();
      execution.setId(1);
      execution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue('exec'));
      const nodeMlmdInfo = getNodeMlmdInfo(elem, [execution], [], []);
      expect(nodeMlmdInfo).toEqual({ execution });
    });

    it('execution found with custom name', () => {
      const label = 'custom-label';
      const elem: Node<FlowElementDataBase> = {
        id: 'task.exec',
        data: {
          label: label,
          mlmdId: 1,
        },
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      const execution = new Execution();
      execution.setId(1);
      execution
        .getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue(getTaskKeyFromNodeKey(elem.id)));
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
        // updateFlowElementsState stamps the resolved artifact id + producer execution id
        // onto the node; the side panel resolves by those rather than by (task_name,
        // artifact_name).
        data: {
          mlmdId: 2,
          producerExecutionId: 1,
        },
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

  describe('sibling sub-DAGs producing same-named artifacts', () => {
    // Regression test for artifact nodes surfacing the wrong producer: when the same
    // component runs in two sibling sub-DAGs (differing only by an input parameter), both
    // executions share a task_name and emit an artifact with the same name. The node must
    // resolve to the producer in the DAG being viewed, not to whichever OUTPUT event was
    // processed last.
    const ROOT_EXECUTION_ID = 2;
    const SIBLING_DAG_EXECUTION_ID = 999;

    function buildExecution(id: number, taskName: string, parentDagId?: number): Execution {
      const execution = new Execution().setId(id).setLastKnownState(Execution.State.COMPLETE);
      execution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue(taskName));
      if (parentDagId !== undefined) {
        execution
          .getCustomPropertiesMap()
          .set(PARENT_DAG_ID_KEY, new Value().setIntValue(parentDagId));
      }
      return execution;
    }

    function buildOutputEvent(
      executionId: number,
      artifactId: number,
      artifactName: string,
    ): Event {
      return new Event()
        .setExecutionId(executionId)
        .setArtifactId(artifactId)
        .setType(Event.Type.OUTPUT)
        .setPath(new Event.Path().setStepsList([new Event.Path.Step().setKey(artifactName)]));
    }

    const rootExecution = buildExecution(ROOT_EXECUTION_ID, '');
    const currentDagProducer = buildExecution(3, 'preprocess', ROOT_EXECUTION_ID);
    const siblingDagProducer = buildExecution(30, 'preprocess', SIBLING_DAG_EXECUTION_ID);

    const currentDagArtifact = new Artifact().setId(1).setState(Artifact.State.LIVE);
    const siblingDagArtifact = new Artifact().setId(100).setState(Artifact.State.DELETED);

    // The sibling event is listed last on purpose: the previous name-keyed map kept the
    // last write, so before the fix the node resolved to the sibling's artifact.
    const events = [
      buildOutputEvent(3, 1, 'output_dataset_one'),
      buildOutputEvent(30, 100, 'output_dataset_one'),
    ];
    const executions = [rootExecution, currentDagProducer, siblingDagProducer];
    const artifacts = [currentDagArtifact, siblingDagArtifact];

    function buildRootGraph() {
      const yamlObject = load(v2YamlTemplateString);
      const pipelineSpec = PipelineSpec.fromJSON(yamlObject);
      return convertFlowElements(pipelineSpec);
    }

    it('stamps the current DAG artifact id onto the node', () => {
      const graph = updateFlowElementsState(
        ['root'],
        buildRootGraph(),
        executions,
        events,
        artifacts,
      );
      const artifactNode = graph.find(
        (element) => element.id === 'artifact.preprocess.output_dataset_one',
      );
      expect(artifactNode?.data.mlmdId).toEqual(currentDagArtifact.getId());
      expect(artifactNode?.data.producerExecutionId).toEqual(currentDagProducer.getId());
    });

    it('resolves the side panel to the current DAG producer', () => {
      const graph = updateFlowElementsState(
        ['root'],
        buildRootGraph(),
        executions,
        events,
        artifacts,
      );
      const artifactNode = graph.find(
        (element) => element.id === 'artifact.preprocess.output_dataset_one',
      )!;
      const nodeMlmdInfo = getNodeMlmdInfo(artifactNode, executions, events, artifacts);
      expect(nodeMlmdInfo.execution).toEqual(currentDagProducer);
      expect(nodeMlmdInfo.linkedArtifact?.artifact).toEqual(currentDagArtifact);
    });
  });

  describe('artifact with multiple OUTPUT events (same-run cache hit)', () => {
    // A cached execution republishes an existing artifact in the same run, so one artifact
    // id can have several OUTPUT events. getNodeMlmdInfo must return the producer the node
    // was stamped with, not merely the first event referencing the artifact id.
    it('resolves getNodeMlmdInfo by the stamped producer execution', () => {
      const artifact = new Artifact().setId(7).setState(Artifact.State.LIVE);
      const originalProducer = new Execution().setId(3).setLastKnownState(Execution.State.COMPLETE);
      originalProducer
        .getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue('report'));
      const cachedProducer = new Execution().setId(4).setLastKnownState(Execution.State.CACHED);
      cachedProducer
        .getCustomPropertiesMap()
        .set(TASK_NAME_KEY, new Value().setStringValue('report'));

      // Both events point at the same artifact id; the original is listed first.
      const events = [
        new Event().setExecutionId(3).setArtifactId(7).setType(Event.Type.OUTPUT),
        new Event().setExecutionId(4).setArtifactId(7).setType(Event.Type.OUTPUT),
      ];

      const elem: Node<FlowElementDataBase> = {
        id: 'artifact.report.out',
        data: { mlmdId: 7, producerExecutionId: 4 },
        type: NodeTypeNames.ARTIFACT,
        position: { x: 1, y: 2 },
      };

      const nodeMlmdInfo = getNodeMlmdInfo(elem, [originalProducer, cachedProducer], events, [
        artifact,
      ]);
      expect(nodeMlmdInfo.execution).toEqual(cachedProducer);
      expect(nodeMlmdInfo.linkedArtifact?.event.getExecutionId()).toEqual(4);
    });
  });

  describe('sibling sub-DAG output artifacts', () => {
    // A sub-DAG output artifact is produced by an inner subtask one layer below the node.
    // Two sibling sub-DAG tasks of the same component share the inner (task, artifact)
    // name, so each output node must resolve to its own sub-DAG's execution.
    function buildExecution(id: number, taskName: string, parentDagId?: number): Execution {
      const execution = new Execution().setId(id).setLastKnownState(Execution.State.COMPLETE);
      execution.getCustomPropertiesMap().set(TASK_NAME_KEY, new Value().setStringValue(taskName));
      if (parentDagId !== undefined) {
        execution
          .getCustomPropertiesMap()
          .set(PARENT_DAG_ID_KEY, new Value().setIntValue(parentDagId));
      }
      return execution;
    }

    function buildSubDagOutputNode(subDagTask: string): Node<ArtifactFlowElementData> {
      return {
        id: `artifact.${subDagTask}.report`,
        data: {
          label: `${subDagTask}.report`,
          producerSubtask: 'create_report',
          outputArtifactKey: 'report',
        },
        type: NodeTypeNames.ARTIFACT,
        position: { x: 1, y: 2 },
      };
    }

    it('scopes each output node to its own sub-DAG execution', () => {
      const root = buildExecution(1, '');
      const subDagA = buildExecution(10, 'shap_a', 1);
      const subDagB = buildExecution(11, 'shap_b', 1);
      const innerA = buildExecution(20, 'create_report', 10);
      const innerB = buildExecution(21, 'create_report', 11);

      const artifactA = new Artifact().setId(100).setState(Artifact.State.LIVE);
      const artifactB = new Artifact().setId(200).setState(Artifact.State.LIVE);

      const reportEvent = (executionId: number, artifactId: number) =>
        new Event()
          .setExecutionId(executionId)
          .setArtifactId(artifactId)
          .setType(Event.Type.OUTPUT)
          .setPath(new Event.Path().setStepsList([new Event.Path.Step().setKey('report')]));
      // innerA is listed first: pre-fix, both sibling nodes fell back to it.
      const events = [reportEvent(20, 100), reportEvent(21, 200)];
      const executions = [root, subDagA, subDagB, innerA, innerB];
      const artifacts = [artifactA, artifactB];

      const graph = updateFlowElementsState(
        ['root'],
        [buildSubDagOutputNode('shap_a'), buildSubDagOutputNode('shap_b')],
        executions,
        events,
        artifacts,
      );

      const nodeA = graph.find((element) => element.id === 'artifact.shap_a.report');
      const nodeB = graph.find((element) => element.id === 'artifact.shap_b.report');
      expect(nodeA?.data.mlmdId).toEqual(artifactA.getId());
      expect(nodeB?.data.mlmdId).toEqual(artifactB.getId());
    });
  });
});
