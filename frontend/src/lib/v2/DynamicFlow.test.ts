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

    it('resolves a synthetic ParallelFor iteration node to its iteration DAG task', () => {
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
        type: PipelineTaskTaskType.DAG,
        type_attributes: { iteration_index: '0' },
      };
      const iterationOne: V2beta1PipelineTask = {
        task_id: 'body-1',
        parent_task_id: loopTask.task_id,
        name: 'body',
        type: PipelineTaskTaskType.DAG,
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
          [rootTask, loopTask, iterationZero, iterationOne],
          ['root', 'loop'],
        ),
      ).toEqual({ task: iterationOne });
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
