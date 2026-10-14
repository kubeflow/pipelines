// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ComponentSpec, PipelineSpec, PipelineTaskSpec } from 'src/generated/pipeline_spec';
import { PipelineTaskSpec_TriggerPolicy_TriggerStrategy as TriggerStrategy } from 'src/generated/pipeline_spec/pipeline_spec';
import {
  PipelineTaskTaskState as State,
  PipelineTaskTaskType as Type,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';

const task = (name: string, component: string, dependencies: string[] = []) =>
  PipelineTaskSpec.fromPartial({
    taskInfo: { name },
    componentRef: { name: component },
    dependentTasks: dependencies,
  });

/** Small, representative IR fixtures shared by graph tests and interactive stories. */
export const nestedArtifactSpec = PipelineSpec.fromPartial({
  pipelineInfo: { name: 'nested-training' },
  deploymentSpec: {
    executors: {
      'exec-train': { container: { image: 'python:3.11' } },
      'exec-evaluate': { container: { image: 'python:3.11' } },
      'exec-prepare': { container: { image: 'python:3.11' } },
    },
  },
  components: {
    train: { executorLabel: 'exec-train', outputDefinitions: { artifacts: { model: {} } } },
    evaluate: { executorLabel: 'exec-evaluate', inputDefinitions: { artifacts: { model: {} } } },
    prepare: { executorLabel: 'exec-prepare' },
    body: {
      dag: {
        tasks: {
          train: task('Train model', 'train'),
          evaluate: {
            ...task('Evaluate model', 'evaluate'),
            inputs: {
              artifacts: {
                model: {
                  taskOutputArtifact: { producerTask: 'train', outputArtifactKey: 'model' },
                },
              },
            },
          },
        },
        outputs: {
          artifacts: {
            model: {
              artifactSelectors: [{ producerSubtask: 'train', outputArtifactKey: 'model' }],
            },
          },
        },
      },
      outputDefinitions: { artifacts: { model: {} } },
    },
    workflow: {
      dag: {
        tasks: {
          prepare: task('Prepare data', 'prepare'),
          fit: task('Training and evaluation', 'body', ['prepare']),
        },
        outputs: {
          artifacts: {
            model: { artifactSelectors: [{ producerSubtask: 'fit', outputArtifactKey: 'model' }] },
          },
        },
      },
      outputDefinitions: { artifacts: { model: {} } },
    },
  },
  root: {
    dag: {
      tasks: {
        workflow: task('Training pipeline', 'workflow'),
        deploy: {
          ...task('Deploy model', 'evaluate'),
          inputs: {
            artifacts: {
              model: {
                taskOutputArtifact: { producerTask: 'workflow', outputArtifactKey: 'model' },
              },
            },
          },
        },
      },
    },
  },
});

export const conditionSpec = PipelineSpec.fromPartial({
  ...nestedArtifactSpec,
  pipelineInfo: { name: 'conditional-training' },
  root: {
    dag: {
      tasks: {
        choose: task('Choose training strategy', 'prepare'),
        accurate: {
          ...task('If accuracy is sufficient', 'body', ['choose']),
          triggerPolicy: { condition: "inputs.parameter_values['accuracy'] > 0.9" },
        },
        otherwise: {
          ...task('Otherwise retrain', 'body', ['choose']),
          triggerPolicy: { condition: "inputs.parameter_values['accuracy'] <= 0.9" },
        },
      },
    },
  },
});

export const loopSpec = PipelineSpec.fromPartial({
  ...nestedArtifactSpec,
  pipelineInfo: { name: 'parallel-training' },
  root: {
    dag: {
      tasks: {
        sweep: {
          ...task('Hyperparameter sweep', 'body'),
          parameterIterator: { itemInput: 'learning-rate', items: { raw: '[0.01, 0.1]' } },
        },
        summarize: task('Summarize results', 'prepare', ['sweep']),
      },
    },
  },
});

export const exitHandlerSpec = PipelineSpec.fromPartial({
  pipelineInfo: { name: 'exit-handler-notification' },
  deploymentSpec: { executors: { 'exec-op': { container: { image: 'python:3.11' } } } },
  components: {
    op: { executorLabel: 'exec-op' },
    guarded: {
      dag: {
        tasks: {
          'fail-op': task('fail-op', 'op'),
          'print-op': task('print-op', 'op'),
        },
      },
    },
    condition: { dag: { tasks: { 'print-op': task('print-op', 'op') } } },
    notification: {
      dag: {
        tasks: {
          'get-run-state': task('get-run-state', 'op'),
          'condition-1': {
            ...task('condition-1', 'condition', ['get-run-state']),
            triggerPolicy: { condition: "inputs.parameter_values['state'] == 'FAILED'" },
          },
        },
      },
    },
  },
  root: {
    dag: {
      tasks: {
        'exit-handler-1': task('my-pipeline', 'guarded'),
        'conditional-notification': {
          ...task('conditional-notification', 'notification', ['exit-handler-1']),
          inputs: {
            parameters: { status: { taskFinalStatus: { producerTask: 'exit-handler-1' } } },
          },
          triggerPolicy: { strategy: TriggerStrategy.ALL_UPSTREAM_TASKS_COMPLETED },
        },
      },
    },
  },
});

/** Reverse declaration order verifies expansion follows dependencies, not object order. */
export function manySubDagsSpec(count = 25, parallelBranches = 1) {
  return PipelineSpec.fromPartial({
    components: {
      step: { executorLabel: 'exec-step' },
      body: { dag: { tasks: { step: task('Step', 'step') } } },
    },
    root: {
      dag: {
        tasks: Object.fromEntries(
          Array.from({ length: count }, (_, index) => [
            `group-${index}`,
            task(
              `Group ${index}`,
              'body',
              index >= parallelBranches ? [`group-${index - parallelBranches}`] : [],
            ),
          ]).reverse(),
        ),
      },
    },
  });
}

/** Two executable components per level, enclosed by a single 25-level DAG chain. */
export function deeplyNestedSpec() {
  const spec = PipelineSpec.fromPartial({
    pipelineInfo: { name: 'nested-50-components-25-levels' },
    deploymentSpec: {
      executors: { 'exec-noop': { container: { image: 'python:3.11', command: ['true'] } } },
    },
    root: { dag: { tasks: { 'level-1': task('Level 01', 'level-1') } } },
  });
  for (let depth = 1; depth <= 25; depth++) {
    const tasks: Record<string, PipelineTaskSpec> = {};
    for (let index = 1; index <= 2; index++) {
      const number = (depth - 1) * 2 + index;
      const key = `component-${number}`;
      tasks[key] = task(`Component ${String(number).padStart(3, '0')}`, key);
      spec.components[key] = ComponentSpec.fromPartial({ executorLabel: 'exec-noop' });
    }
    if (depth < 25) {
      const nextLevel = `level-${depth + 1}`;
      tasks[nextLevel] = task(
        `Level ${String(depth + 1).padStart(2, '0')}`,
        nextLevel,
        Object.keys(tasks),
      );
    }
    spec.components[`level-${depth}`] = ComponentSpec.fromPartial({ dag: { tasks } });
  }
  return spec;
}

export const loopTasks: V2beta1PipelineTask[] = [
  { task_id: 'root', name: 'root', type: Type.ROOT, state: State.RUNNING },
  {
    task_id: 'sweep',
    parent_task_id: 'root',
    name: 'sweep',
    display_name: 'Hyperparameter sweep',
    type: Type.LOOP,
    state: State.RUNNING,
    type_attributes: { iteration_count: '2' },
  },
  ...[0, 1].flatMap((iteration) => [
    {
      task_id: `train-${iteration}`,
      parent_task_id: 'sweep',
      name: 'train',
      type: Type.RUNTIME,
      state: iteration === 0 ? State.CACHED : State.SUCCEEDED,
      type_attributes: { iteration_index: String(iteration) },
      outputs: {
        artifacts: [
          {
            artifact_key: 'model',
            artifacts: [{ artifact_id: String(iteration + 1), uri: `s3://models/${iteration}` }],
          },
        ],
      },
    },
    {
      task_id: `evaluate-${iteration}`,
      parent_task_id: 'sweep',
      name: 'evaluate',
      type: Type.RUNTIME,
      state: iteration === 0 ? State.SUCCEEDED : State.RUNNING,
      type_attributes: { iteration_index: String(iteration) },
    },
  ]),
];
