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

import { load } from 'js-yaml';
import { Node } from '@xyflow/react';
import {
  ArtifactArtifactType,
  PipelineTaskTaskState,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { convertFlowElements, NodeTypeNames } from './StaticFlow';
import {
  buildRuntimeFlowContext,
  convertSubDagToRuntimeFlowElements,
  getNodeRuntimeInfo,
  getTaskRuntimeLayers,
  reconcileRuntimeFlowElements,
  updateFlowElementsState,
} from './DynamicFlow';

const rootTask: V2beta1PipelineTask = {
  task_id: 'root-task',
  name: 'root',
  type: PipelineTaskTaskType.ROOT,
  state: PipelineTaskTaskState.RUNNING,
};

describe('DynamicFlow', () => {
  describe('updateFlowElementsState', () => {
    it('preserves the spec identity when the loop runtime display name is generic', () => {
      const graph = updateFlowElementsState(
        ['root'],
        [
          {
            id: 'task.parallel-loop',
            type: NodeTypeNames.SUB_DAG,
            position: { x: 0, y: 0 },
            data: { label: 'parallel-loop' },
          },
        ],
        [
          rootTask,
          {
            task_id: 'loop-task',
            parent_task_id: rootTask.task_id,
            name: 'parallel-loop',
            display_name: 'Loop',
            type: PipelineTaskTaskType.LOOP,
          },
        ],
      );
      expect(graph[0].data?.label).toBe('parallel-loop (Loop)');
    });

    it('updates task and artifact nodes from hydrated task data', () => {
      const preprocessTask: V2beta1PipelineTask = {
        task_id: 'preprocess-task',
        parent_task_id: rootTask.task_id,
        name: 'preprocess',
        display_name: 'Preprocess data',
        state: PipelineTaskTaskState.SUCCEEDED,
        outputs: {
          artifacts: [
            {
              artifact_key: 'output_dataset_one',
              artifacts: [
                {
                  artifact_id: 'artifact-one',
                  name: 'dataset-one',
                  type: ArtifactArtifactType.Dataset,
                },
              ],
            },
            {
              artifact_key: 'output_dataset_two_path',
              artifacts: [
                {
                  artifact_id: 'artifact-two',
                  name: 'dataset-two',
                  type: ArtifactArtifactType.Dataset,
                },
              ],
            },
          ],
        },
      };
      const trainTask: V2beta1PipelineTask = {
        task_id: 'train-task',
        parent_task_id: rootTask.task_id,
        name: 'train',
        state: PipelineTaskTaskState.FAILED,
        outputs: {
          artifacts: [
            {
              artifact_key: 'model',
              artifacts: [
                {
                  artifact_id: 'model-artifact',
                  name: 'model',
                  type: ArtifactArtifactType.Model,
                },
              ],
            },
          ],
        },
      };

      const yamlObject = load(v2YamlTemplateString);
      const graph = convertFlowElements(PipelineSpec.fromJSON(yamlObject));
      const runtimeGraph = updateFlowElementsState(['root'], graph, [
        rootTask,
        preprocessTask,
        trainTask,
      ]);

      expect(runtimeGraph.find((element) => element.id === 'task.preprocess')?.data).toMatchObject({
        label: 'Preprocess data',
        state: PipelineTaskTaskState.SUCCEEDED,
        taskId: 'preprocess-task',
      });
      expect(runtimeGraph.find((element) => element.id === 'task.train')?.data).toMatchObject({
        state: PipelineTaskTaskState.FAILED,
        taskId: 'train-task',
      });
      expect(
        runtimeGraph.find((element) => element.id === 'artifact.preprocess.output_dataset_one')
          ?.data,
      ).toMatchObject({ hasArtifact: true });
      expect(
        runtimeGraph.find((element) => element.id === 'artifact.train.model')?.data,
      ).toMatchObject({ hasArtifact: true });
    });

    it('does not preserve React Flow hidden flags when applying task state', () => {
      const preprocessTask: V2beta1PipelineTask = {
        task_id: 'preprocess-task',
        parent_task_id: rootTask.task_id,
        name: 'preprocess',
        state: PipelineTaskTaskState.SUCCEEDED,
      };
      const yamlObject = load(v2YamlTemplateString);
      const graph = convertFlowElements(PipelineSpec.fromJSON(yamlObject));
      const preprocessNode = graph.find((element) => element.id === 'task.preprocess') as Node;
      (preprocessNode as Node & { hidden?: boolean }).hidden = true;
      preprocessNode.measured = { width: 123, height: 45 };

      const runtimeGraph = updateFlowElementsState(['root'], graph, [rootTask, preprocessTask]);
      const updatedPreprocessNode = runtimeGraph.find(
        (element) => element.id === 'task.preprocess',
      ) as Node & { hidden?: boolean };

      expect(updatedPreprocessNode.hidden).toBeUndefined();
      expect(updatedPreprocessNode.measured).toEqual({ width: 123, height: 45 });
      expect(updatedPreprocessNode.data?.state).toEqual(PipelineTaskTaskState.SUCCEEDED);
    });
  });

  describe('getNodeRuntimeInfo', () => {
    it('finds a task from the current runtime layer', () => {
      const element: Node<FlowElementDataBase> = {
        id: 'task.exec',
        data: { label: 'custom-label' },
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };
      const task: V2beta1PipelineTask = {
        task_id: 'task-id',
        parent_task_id: rootTask.task_id,
        name: 'exec',
      };

      expect(getNodeRuntimeInfo(element, [rootTask, task], ['root'])).toEqual({ task });
    });

    it('finds an output artifact group on its producing task', () => {
      const element: Node<FlowElementDataBase> = {
        id: 'artifact.exec.output',
        data: { label: 'output' },
        type: NodeTypeNames.ARTIFACT,
        position: { x: 1, y: 2 },
      };
      const artifactGroup = {
        artifact_key: 'output',
        artifacts: [{ artifact_id: 'artifact-id', name: 'output' }],
      };
      const task: V2beta1PipelineTask = {
        task_id: 'task-id',
        parent_task_id: rootTask.task_id,
        name: 'exec',
        outputs: { artifacts: [artifactGroup] },
      };

      expect(getNodeRuntimeInfo(element, [rootTask, task], ['root'])).toEqual({
        task,
        artifactGroup,
      });
    });

    it('filters loop body tasks by selected iteration', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const iterationZero: V2beta1PipelineTask = {
        task_id: 'body-0',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type_attributes: { iteration_index: '0' },
      };
      const iterationOne: V2beta1PipelineTask = {
        task_id: 'body-1',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type_attributes: { iteration_index: '1' },
      };
      const element: Node<FlowElementDataBase> = {
        id: 'task.body',
        data: { label: 'body' },
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      expect(
        getNodeRuntimeInfo(
          element,
          [rootTask, loopTask, iterationZero, iterationOne],
          ['root', 'loop', 'loop.1'],
        ),
      ).toEqual({ task: iterationOne });
    });

    it('descends into the named task from the selected loop iteration', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const iterationZeroDag: V2beta1PipelineTask = {
        task_id: 'body-0',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '0' },
      };
      const iterationOneDag: V2beta1PipelineTask = {
        task_id: 'body-1',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '1' },
      };
      const iterationZeroChild: V2beta1PipelineTask = {
        task_id: 'child-0',
        parent_task_id: iterationZeroDag.task_id,
        name: 'exec',
      };
      const iterationOneChild: V2beta1PipelineTask = {
        task_id: 'child-1',
        parent_task_id: iterationOneDag.task_id,
        name: 'exec',
      };
      const element: Node<FlowElementDataBase> = {
        id: 'task.exec',
        data: { label: 'exec' },
        type: NodeTypeNames.EXECUTION,
        position: { x: 1, y: 2 },
      };

      expect(
        getNodeRuntimeInfo(
          element,
          [
            rootTask,
            loopTask,
            iterationZeroDag,
            iterationOneDag,
            iterationZeroChild,
            iterationOneChild,
          ],
          ['root', 'loop', 'loop.1', 'body'],
        ),
      ).toEqual({ task: iterationOneChild });
    });

    it('does not attribute a synthetic ParallelFor iteration group to an arbitrary body task', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const iterationOneBodyA: V2beta1PipelineTask = {
        task_id: 'body-a-1',
        parent_task_id: loopTask.task_id,
        name: 'body-a',
        type: PipelineTaskTaskType.RUNTIME,
        type_attributes: { iteration_index: '1' },
      };
      const iterationOneBodyB: V2beta1PipelineTask = {
        task_id: 'body-b-1',
        parent_task_id: loopTask.task_id,
        name: 'body-b',
        type: PipelineTaskTaskType.RUNTIME,
        type_attributes: { iteration_index: '1' },
      };
      const element: Node<FlowElementDataBase> = {
        id: 'task.loop.1',
        data: { label: 'loop.1' },
        type: NodeTypeNames.SUB_DAG,
        position: { x: 1, y: 2 },
      };

      expect(
        getNodeRuntimeInfo(
          element,
          [rootTask, loopTask, iterationOneBodyA, iterationOneBodyB],
          ['root', 'loop'],
        ),
      ).toEqual({});
    });
  });

  describe('getTaskRuntimeLayers', () => {
    it('resolves nested DAG and ParallelFor iteration layers from task ancestry', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const iterationDag: V2beta1PipelineTask = {
        task_id: 'body-1',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '1' },
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        parent_task_id: iterationDag.task_id,
        name: 'train',
        type: PipelineTaskTaskType.RUNTIME,
      };

      expect(
        getTaskRuntimeLayers(nestedTask, [rootTask, loopTask, iterationDag, nestedTask]),
      ).toEqual(['root', 'loop', 'loop.1', 'body']);
    });

    it('falls back to scope_path when parent task records are unavailable', () => {
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        scope_path: 'root.outer.inner.train',
      };

      expect(getTaskRuntimeLayers(nestedTask, [nestedTask])).toEqual(['root', 'outer', 'inner']);
    });

    it('restores the ParallelFor iteration layer when scope_path ancestry is unavailable', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        name: 'loop',
        scope_path: 'root.loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'body-task',
        scope_path: 'root.loop.body.train',
        type_attributes: { iteration_index: '2' },
      };
      const bodyTask: V2beta1PipelineTask = {
        task_id: 'body-task',
        parent_task_id: 'loop-task',
        name: 'body',
        scope_path: 'root.loop.body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [loopTask, bodyTask, nestedTask])).toEqual([
        'root',
        'loop',
        'loop.2',
        'body',
      ]);
    });

    it('does not manufacture nested ParallelFor indices from one leaf iteration', () => {
      const outerLoop: V2beta1PipelineTask = {
        task_id: 'outer-loop',
        name: 'outer',
        scope_path: 'root.outer',
        type: PipelineTaskTaskType.LOOP,
      };
      const innerLoop: V2beta1PipelineTask = {
        task_id: 'inner-loop',
        name: 'inner',
        scope_path: 'root.outer.inner',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'missing-parent',
        scope_path: 'root.outer.inner.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [outerLoop, innerLoop, nestedTask])).toBeUndefined();
    });

    it('does not manufacture a nested loop path when only one loop context is visible', () => {
      const innerLoop: V2beta1PipelineTask = {
        task_id: 'inner-loop',
        name: 'inner',
        scope_path: 'root.outer.inner',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'missing-parent',
        scope_path: 'root.outer.inner.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [innerLoop, nestedTask])).toBeUndefined();
    });

    it('does not choose an arbitrary same-name loop when scope ancestry is ambiguous', () => {
      const loopA: V2beta1PipelineTask = {
        task_id: 'loop-a',
        name: 'loop',
        scope_path: 'root.loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const loopB: V2beta1PipelineTask = {
        task_id: 'loop-b',
        name: 'loop',
        scope_path: 'root.loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'missing-parent',
        scope_path: 'root.loop.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [loopA, loopB, nestedTask])).toBeUndefined();
    });

    it('matches same-name contexts by their exact runtime scope', () => {
      const wrongBody: V2beta1PipelineTask = {
        task_id: 'wrong-body',
        name: 'body',
        scope_path: 'root.other.body',
        type: PipelineTaskTaskType.LOOP,
      };
      const stage: V2beta1PipelineTask = {
        task_id: 'stage',
        parent_task_id: 'missing-root',
        name: 'stage',
        scope_path: 'root.stage',
        type: PipelineTaskTaskType.DAG,
      };
      const body: V2beta1PipelineTask = {
        task_id: 'body',
        parent_task_id: 'stage',
        name: 'body',
        scope_path: 'root.stage.body',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'body',
        scope_path: 'root.stage.body.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [wrongBody, stage, body, nestedTask])).toEqual([
        'root',
        'stage',
        'body',
        'body.2',
      ]);
    });

    it('uses parent IDs to disambiguate repeated contexts across loop iterations', () => {
      const loop: V2beta1PipelineTask = {
        task_id: 'loop',
        parent_task_id: 'missing-root',
        name: 'loop',
        scope_path: 'root.loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const bodyZero: V2beta1PipelineTask = {
        task_id: 'body-0',
        parent_task_id: 'loop',
        name: 'body',
        scope_path: 'root.loop.body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '0' },
      };
      const bodyOne: V2beta1PipelineTask = {
        task_id: 'body-1',
        parent_task_id: 'loop',
        name: 'body',
        scope_path: 'root.loop.body',
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '1' },
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        parent_task_id: 'body-1',
        name: 'train',
        scope_path: 'root.loop.body.train',
        type_attributes: { iteration_index: '1' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [loop, bodyZero, bodyOne, nestedTask])).toEqual([
        'root',
        'loop',
        'loop.1',
        'body',
      ]);
    });

    it('rejects linked runtime ancestry whose scope does not match the static path', () => {
      const linkedWrongBody: V2beta1PipelineTask = {
        task_id: 'wrong-body',
        parent_task_id: 'missing-root',
        name: 'body',
        scope_path: 'root.other.body',
        type: PipelineTaskTaskType.DAG,
      };
      const unlinkedExactBody: V2beta1PipelineTask = {
        task_id: 'exact-body',
        parent_task_id: 'missing-root',
        name: 'body',
        scope_path: 'root.body',
        type: PipelineTaskTaskType.DAG,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        parent_task_id: 'wrong-body',
        name: 'train',
        scope_path: 'root.body.train',
        type_attributes: { iteration_index: '1' },
      };

      expect(
        getTaskRuntimeLayers(nestedTask, [linkedWrongBody, unlinkedExactBody, nestedTask]),
      ).toBeUndefined();
    });

    it('derives each nested loop iteration from its immediate child context', () => {
      const outer: V2beta1PipelineTask = {
        task_id: 'outer',
        parent_task_id: 'missing-root',
        name: 'outer',
        scope_path: 'root.outer',
        type: PipelineTaskTaskType.LOOP,
      };
      const inner: V2beta1PipelineTask = {
        task_id: 'inner',
        parent_task_id: 'outer',
        name: 'inner',
        scope_path: 'root.outer.inner',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_index: '1' },
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'inner',
        scope_path: 'root.outer.inner.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [outer, inner, nestedTask])).toEqual([
        'root',
        'outer',
        'outer.1',
        'inner',
        'inner.2',
      ]);
    });

    it('preserves condition contexts while reconstructing a loop path', () => {
      const branches: V2beta1PipelineTask = {
        task_id: 'branches',
        parent_task_id: 'missing-root',
        name: 'condition-branches-1',
        scope_path: 'root.condition-branches-1',
        type: PipelineTaskTaskType.CONDITION_BRANCH,
      };
      const condition: V2beta1PipelineTask = {
        task_id: 'condition',
        parent_task_id: 'branches',
        name: 'condition-1',
        scope_path: 'root.condition-branches-1.condition-1',
        type: PipelineTaskTaskType.CONDITION,
      };
      const loop: V2beta1PipelineTask = {
        task_id: 'loop',
        parent_task_id: 'condition',
        name: 'loop',
        scope_path: 'root.condition-branches-1.condition-1.loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        name: 'train',
        parent_task_id: 'loop',
        scope_path: 'root.condition-branches-1.condition-1.loop.train',
        type_attributes: { iteration_index: '3' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [branches, condition, loop, nestedTask])).toEqual([
        'root',
        'condition-branches-1',
        'condition-1',
        'loop',
        'loop.3',
      ]);
    });

    it('does not combine exact-scope contexts from different runtime ancestry chains', () => {
      const outer: V2beta1PipelineTask = {
        task_id: 'outer-attempt-1',
        parent_task_id: 'missing-root',
        name: 'outer',
        scope_path: 'root.outer',
        type: PipelineTaskTaskType.LOOP,
      };
      const inner: V2beta1PipelineTask = {
        task_id: 'inner-attempt-2',
        parent_task_id: 'outer-attempt-2',
        name: 'inner',
        scope_path: 'root.outer.inner',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_index: '1' },
      };
      const nestedTask: V2beta1PipelineTask = {
        task_id: 'nested-task',
        parent_task_id: 'inner-attempt-2',
        name: 'train',
        scope_path: 'root.outer.inner.train',
        type_attributes: { iteration_index: '2' },
      };

      expect(getTaskRuntimeLayers(nestedTask, [outer, inner, nestedTask])).toBeUndefined();
    });
  });

  describe('convertSubDagToRuntimeFlowElements', () => {
    it('builds one synthetic sub-DAG node per loop iteration', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const pipelineSpec = PipelineSpec.fromJSON({
        root: {
          dag: {
            tasks: {
              loop: {
                taskInfo: { name: 'loop' },
                componentRef: { name: 'loop-component' },
              },
            },
          },
        },
        components: {
          'loop-component': { dag: { tasks: {} } },
        },
      });

      const elements = convertSubDagToRuntimeFlowElements(
        pipelineSpec,
        ['root', 'loop'],
        [rootTask, loopTask],
      );
      expect(elements.map((element) => element.id)).toEqual(['task.loop.0', 'task.loop.1']);
    });

    it('replaces a pre-task loop body with iteration nodes when tasks arrive', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const pipelineSpec = PipelineSpec.fromJSON({
        root: {
          dag: {
            tasks: {
              loop: {
                taskInfo: { name: 'loop' },
                componentRef: { name: 'loop-component' },
              },
            },
          },
        },
        components: {
          'loop-component': {
            dag: {
              tasks: {
                body: {
                  taskInfo: { name: 'body' },
                  componentRef: { name: 'body-component' },
                },
              },
            },
          },
          'body-component': { executorLabel: 'exec' },
        },
      });
      const layers = ['root', 'loop'];
      const preTaskElements = convertSubDagToRuntimeFlowElements(pipelineSpec, layers, []);

      expect(preTaskElements.map((element) => element.id)).toContain('task.body');
      expect(
        reconcileRuntimeFlowElements(layers, preTaskElements, [rootTask, loopTask]).map(
          (element) => element.id,
        ),
      ).toEqual(['task.loop.0', 'task.loop.1']);
    });

    it('keeps an iteration running until every declarative body task exists', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        state: PipelineTaskTaskState.RUNNING,
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '1' },
      };
      const bodyA: V2beta1PipelineTask = {
        task_id: 'body-a-0',
        parent_task_id: loopTask.task_id,
        name: 'body-a',
        state: PipelineTaskTaskState.SUCCEEDED,
        type: PipelineTaskTaskType.RUNTIME,
        type_attributes: { iteration_index: '0' },
      };
      const bodyB: V2beta1PipelineTask = {
        task_id: 'body-b-0',
        parent_task_id: loopTask.task_id,
        name: 'body-b',
        state: PipelineTaskTaskState.SUCCEEDED,
        type: PipelineTaskTaskType.RUNTIME,
        type_attributes: { iteration_index: '0' },
      };
      const pipelineSpec = PipelineSpec.fromJSON({
        root: {
          dag: {
            tasks: {
              loop: {
                taskInfo: { name: 'loop' },
                componentRef: { name: 'loop-component' },
              },
            },
          },
        },
        components: {
          'loop-component': {
            dag: {
              tasks: {
                'body-a': { taskInfo: { name: 'body-a' } },
                'body-b': { taskInfo: { name: 'body-b' } },
              },
            },
          },
        },
      });

      const partialElements = convertSubDagToRuntimeFlowElements(
        pipelineSpec,
        ['root', 'loop'],
        [rootTask, loopTask, bodyA],
      );
      expect(partialElements[0].data).toMatchObject({
        expectedTaskCount: 2,
        state: PipelineTaskTaskState.RUNNING,
      });

      const terminalPartialElements = updateFlowElementsState(['root', 'loop'], partialElements, [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.SUCCEEDED },
        bodyA,
      ]);
      expect(terminalPartialElements[0].data?.state).toBe(PipelineTaskTaskState.SUCCEEDED);

      const failedPartialElements = updateFlowElementsState(['root', 'loop'], partialElements, [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.FAILED },
        bodyA,
      ]);
      expect(failedPartialElements[0].data?.state).toBe(PipelineTaskTaskState.RUNNING);

      const terminalFailedPartialTasks = [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.FAILED },
        bodyA,
      ];
      const terminalFailedPartialElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        terminalFailedPartialTasks,
        buildRuntimeFlowContext(['root', 'loop'], terminalFailedPartialTasks, true),
      );
      expect(terminalFailedPartialElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const terminalStaleRunningLoopTasks = [rootTask, loopTask, bodyA];
      const terminalStaleRunningLoopElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        terminalStaleRunningLoopTasks,
        buildRuntimeFlowContext(['root', 'loop'], terminalStaleRunningLoopTasks, true),
      );
      expect(terminalStaleRunningLoopElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const staleRunningBody = { ...bodyA, state: PipelineTaskTaskState.RUNNING };
      const failedIncompleteTasks = [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.FAILED },
        staleRunningBody,
      ];
      const activeFailedWithRunningIncompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        failedIncompleteTasks,
      );
      expect(activeFailedWithRunningIncompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNNING,
      );
      const terminalFailedWithStaleIncompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        failedIncompleteTasks,
        buildRuntimeFlowContext(['root', 'loop'], failedIncompleteTasks, true),
      );
      expect(terminalFailedWithStaleIncompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const failedCompleteTasks = [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.FAILED },
        staleRunningBody,
        bodyB,
      ];
      const activeFailedWithRunningCompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        failedCompleteTasks,
      );
      expect(activeFailedWithRunningCompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNNING,
      );
      const terminalFailedWithStaleCompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        failedCompleteTasks,
        buildRuntimeFlowContext(['root', 'loop'], failedCompleteTasks, true),
      );
      expect(terminalFailedWithStaleCompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const succeededWithStaleIncompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        [rootTask, { ...loopTask, state: PipelineTaskTaskState.SUCCEEDED }, staleRunningBody],
      );
      expect(succeededWithStaleIncompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const succeededWithStaleCompleteElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        [
          rootTask,
          { ...loopTask, state: PipelineTaskTaskState.SUCCEEDED },
          staleRunningBody,
          bodyB,
        ],
      );
      expect(succeededWithStaleCompleteElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const succeededWithStaleFailedElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        [
          rootTask,
          { ...loopTask, state: PipelineTaskTaskState.SUCCEEDED },
          { ...bodyA, state: PipelineTaskTaskState.FAILED },
          bodyB,
        ],
      );
      expect(succeededWithStaleFailedElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const failedChildElements = updateFlowElementsState(['root', 'loop'], partialElements, [
        rootTask,
        { ...loopTask, state: PipelineTaskTaskState.FAILED },
        { ...bodyA, state: PipelineTaskTaskState.FAILED },
      ]);
      expect(failedChildElements[0].data?.state).toBe(PipelineTaskTaskState.FAILED);

      const successfulRunWithStaleFailedChildTasks = [
        rootTask,
        loopTask,
        { ...bodyA, state: PipelineTaskTaskState.FAILED },
      ];
      const successfulRunWithStaleFailedChildElements = updateFlowElementsState(
        ['root', 'loop'],
        partialElements,
        successfulRunWithStaleFailedChildTasks,
        buildRuntimeFlowContext(
          ['root', 'loop'],
          successfulRunWithStaleFailedChildTasks,
          true,
          true,
        ),
      );
      expect(successfulRunWithStaleFailedChildElements[0].data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );

      const completeElements = updateFlowElementsState(['root', 'loop'], partialElements, [
        rootTask,
        loopTask,
        bodyA,
        bodyB,
      ]);
      expect(completeElements[0].data?.state).toBe(PipelineTaskTaskState.SUCCEEDED);
    });

    it('does not leave ordinary task nodes running after the enclosing run terminates', () => {
      const runningTask: V2beta1PipelineTask = {
        task_id: 'preprocess-task',
        parent_task_id: rootTask.task_id,
        name: 'preprocess',
        state: PipelineTaskTaskState.RUNNING,
      };
      const graph = convertFlowElements(PipelineSpec.fromJSON(load(v2YamlTemplateString)));
      const tasks = [rootTask, runningTask];

      const activeGraph = updateFlowElementsState(['root'], graph, tasks);
      expect(activeGraph.find((element) => element.id === 'task.preprocess')?.data?.state).toBe(
        PipelineTaskTaskState.RUNNING,
      );

      const terminalGraph = updateFlowElementsState(
        ['root'],
        graph,
        tasks,
        buildRuntimeFlowContext(['root'], tasks, true),
      );
      expect(terminalGraph.find((element) => element.id === 'task.preprocess')?.data?.state).toBe(
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      );
    });

    it('does not show stale failed task or sub-DAG nodes after a successful run', () => {
      const failedExecution: V2beta1PipelineTask = {
        task_id: 'execution-task',
        parent_task_id: rootTask.task_id,
        name: 'execution',
        state: PipelineTaskTaskState.FAILED,
      };
      const failedSubDag: V2beta1PipelineTask = {
        task_id: 'sub-dag-task',
        parent_task_id: rootTask.task_id,
        name: 'sub-dag',
        state: PipelineTaskTaskState.FAILED,
      };
      const elements: Node<FlowElementDataBase>[] = [
        {
          id: 'task.execution',
          data: { label: 'execution' },
          position: { x: 0, y: 0 },
          type: NodeTypeNames.EXECUTION,
        },
        {
          id: 'task.sub-dag',
          data: { label: 'sub-dag' },
          position: { x: 0, y: 0 },
          type: NodeTypeNames.SUB_DAG,
        },
      ];
      const tasks = [rootTask, failedExecution, failedSubDag];

      const graph = updateFlowElementsState(
        ['root'],
        elements,
        tasks,
        buildRuntimeFlowContext(['root'], tasks, true, true),
      );

      expect(graph.map((element) => element.data?.state)).toEqual([
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      ]);
    });

    it('does not show stale failed or running children below a successful nested DAG', () => {
      const successfulDag: V2beta1PipelineTask = {
        task_id: 'successful-dag',
        parent_task_id: rootTask.task_id,
        name: 'successful-dag',
        state: PipelineTaskTaskState.SUCCEEDED,
      };
      const staleFailedChild: V2beta1PipelineTask = {
        task_id: 'stale-failed-child',
        parent_task_id: successfulDag.task_id,
        name: 'child',
        state: PipelineTaskTaskState.FAILED,
      };
      const staleRunningChild: V2beta1PipelineTask = {
        task_id: 'stale-running-child',
        parent_task_id: successfulDag.task_id,
        name: 'running-child',
        state: PipelineTaskTaskState.RUNNING,
      };
      const childElements: Node<FlowElementDataBase>[] = [
        {
          id: 'task.child',
          data: { label: 'child' },
          position: { x: 0, y: 0 },
          type: NodeTypeNames.EXECUTION,
        },
        {
          id: 'task.running-child',
          data: { label: 'running-child' },
          position: { x: 0, y: 0 },
          type: NodeTypeNames.EXECUTION,
        },
      ];
      const tasks = [rootTask, successfulDag, staleFailedChild, staleRunningChild];

      const graph = updateFlowElementsState(
        ['root', 'successful-dag'],
        childElements,
        tasks,
        buildRuntimeFlowContext(['root', 'successful-dag'], tasks, false, false),
      );

      expect(graph.map((element) => element.data?.state)).toEqual([
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
        PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
      ]);
    });

    it('keeps the static loop body when iteration_count is not yet available', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
      };
      const pipelineSpec = PipelineSpec.fromJSON({
        root: {
          dag: {
            tasks: {
              loop: {
                taskInfo: { name: 'loop' },
                componentRef: { name: 'loop-component' },
              },
            },
          },
        },
        components: {
          'loop-component': {
            dag: {
              tasks: {
                body: {
                  taskInfo: { name: 'body' },
                  componentRef: { name: 'body-component' },
                },
              },
            },
          },
          'body-component': { executorLabel: 'exec' },
        },
      });
      const layers = ['root', 'loop'];
      const staticElements = convertSubDagToRuntimeFlowElements(pipelineSpec, layers, []);

      expect(
        reconcileRuntimeFlowElements(layers, staticElements, [rootTask, loopTask]).map(
          (element) => element.id,
        ),
      ).toEqual(staticElements.map((element) => element.id));
    });

    it('does not rerun graph layout when the ParallelFor structure is unchanged', () => {
      const loopTask: V2beta1PipelineTask = {
        task_id: 'loop-task',
        parent_task_id: rootTask.task_id,
        name: 'loop',
        type: PipelineTaskTaskType.LOOP,
        type_attributes: { iteration_count: '2' },
      };
      const pipelineSpec = PipelineSpec.fromJSON({
        root: {
          dag: {
            tasks: {
              loop: {
                taskInfo: { name: 'loop' },
                componentRef: { name: 'loop-component' },
              },
            },
          },
        },
        components: { 'loop-component': { dag: { tasks: {} } } },
      });
      const layers = ['root', 'loop'];
      const elements = convertSubDagToRuntimeFlowElements(pipelineSpec, layers, [
        rootTask,
        loopTask,
      ]);
      const randomSpy = vi.spyOn(Math, 'random');

      reconcileRuntimeFlowElements(layers, elements, [rootTask, loopTask]);

      expect(randomSpy).not.toHaveBeenCalled();
      randomSpy.mockRestore();
    });
  });
});
