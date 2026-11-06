// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2PipelineTask, V2Run } from '../src/apisv2/run';

type Step = {
  name: string;
  after?: string[];
  children?: Step[];
  iterations?: number;
  condition?: boolean;
};
const nested: Step[] = [
  {
    name: 'prepare-features',
    children: [
      { name: 'load-data' },
      {
        name: 'transform-features',
        after: ['load-data'],
        children: [{ name: 'normalize' }, { name: 'encode', after: ['normalize'] }],
      },
    ],
  },
  {
    name: 'train-approved-model',
    after: ['prepare-features'],
    condition: true,
    children: [{ name: 'train' }, { name: 'evaluate', after: ['train'] }],
  },
  {
    name: 'reject-invalid-data',
    after: ['prepare-features'],
    condition: false,
    children: [{ name: 'report-validation-failure' }],
  },
];
const loop: Step[] = [
  { name: 'prepare-training-data' },
  {
    name: 'cross-validation',
    after: ['prepare-training-data'],
    iterations: 3,
    children: [{ name: 'train-fold' }, { name: 'score-fold', after: ['train-fold'] }],
  },
  { name: 'aggregate-scores', after: ['cross-validation'] },
];
// Four stages with eight parallel components each make density/fit/navigation
// review meaningful without hiding the structure inside a single collapsed DAG.
const large: Step[] = Array.from({ length: 4 }, (_, stage) =>
  Array.from({ length: 8 }, (_, shard) => ({
    name: `${['extract', 'transform', 'train', 'evaluate'][stage]}-region-${shard + 1}`,
    after:
      stage === 0 ? [] : [`${['extract', 'transform', 'train'][stage - 1]}-region-${shard + 1}`],
  })),
).flat();

export const complexRunSamples = [
  { id: 'nested-conditions', name: 'Nested feature preparation and conditions', steps: nested },
  { id: 'parallel-folds', name: 'Three-fold parallel validation', steps: loop },
  { id: 'regional-training', name: 'Regional training — 32 components', steps: large },
];

// Synthetic, executable print components using the IR layout of the repository's
// SDK-compiled fixtures. The same tree supplies the graph and runtime ancestry.
export function complexPipelineSpec(sample: (typeof complexRunSamples)[number]) {
  const components: Record<string, object> = {};
  const executors: Record<string, object> = {};
  function dag(steps: Step[]): object {
    return {
      tasks: Object.fromEntries(
        steps.map((step) => {
          const component = `comp-${step.name}`;
          components[component] = step.children
            ? {
                dag: dag(
                  step.iterations
                    ? [{ name: `${step.name}-body`, children: step.children }]
                    : step.children,
                ),
                ...(step.iterations
                  ? {
                      inputDefinitions: {
                        parameters: { fold: { parameterType: 'NUMBER_INTEGER' } },
                      },
                    }
                  : {}),
              }
            : {
                executorLabel: `exec-${step.name}`,
                inputDefinitions: { parameters: { dataset_uri: { parameterType: 'STRING' } } },
                outputDefinitions: {
                  parameters: { row_count: { parameterType: 'NUMBER_INTEGER' } },
                },
              };
          if (!step.children) {
            executors[`exec-${step.name}`] = {
              container: {
                image: 'python:3.11-slim',
                command: ['python', '-c'],
                args: [
                  "import pathlib, sys; print('Processing dataset: ' + sys.argv[1]); output = pathlib.Path(sys.argv[2]); output.parent.mkdir(parents=True, exist_ok=True); output.write_text('12480'); print('Processed 12480 rows')",
                  "{{$.inputs.parameters['dataset_uri']}}",
                  "{{$.outputs.parameters['row_count'].output_file}}",
                ],
              },
            };
          }
          return [
            step.name,
            {
              taskInfo: { name: step.name },
              componentRef: { name: component },
              dependentTasks: step.after || [],
              ...(!step.children
                ? {
                    inputs: {
                      parameters: {
                        dataset_uri: {
                          runtimeValue: {
                            constant: `s3://mlpipeline/review-datasets/${step.name}.parquet`,
                          },
                        },
                      },
                    },
                  }
                : {}),
              ...(step.condition !== undefined
                ? { triggerPolicy: { condition: String(step.condition) } }
                : {}),
              ...(step.iterations
                ? {
                    parameterIterator: {
                      itemInput: 'fold',
                      items: {
                        raw: JSON.stringify(Array.from({ length: step.iterations }, (_, i) => i)),
                      },
                    },
                  }
                : {}),
            },
          ];
        }),
      ),
    };
  }
  const root = { dag: dag(sample.steps) };
  return {
    pipelineInfo: { name: sample.id, description: sample.name },
    schemaVersion: '2.1.0',
    sdkVersion: 'kfp-2.0.0',
    root,
    components,
    deploymentSpec: { executors },
  };
}

export function complexRunTasks(run: V2Run): V2PipelineTask[] | undefined {
  const sample = complexRunSamples.find((entry) => run.run_id === `sample-run-${entry.id}`);
  if (!sample) return undefined;
  const tasks: V2PipelineTask[] = [];
  const at = (seconds: number) => new Date(run.created_at!.getTime() + seconds * 1000);
  function add(
    name: string,
    parent: V2PipelineTask | undefined,
    type: V2PipelineTask['type'],
    start: number,
    iteration?: number,
  ) {
    const scope = parent ? `${parent.scope_path}.${name}` : name;
    const task: V2PipelineTask = {
      run_id: run.run_id,
      task_id: `${run.run_id}-${scope}${iteration === undefined ? '' : `-${iteration}`}`,
      parent_task_id: parent?.task_id,
      name,
      display_name: name.replace(/-/g, ' '),
      scope_path: scope,
      type,
      state: 'SUCCEEDED',
      create_time: at(start),
      ...(iteration === undefined
        ? {}
        : { type_attributes: { iteration_index: String(iteration) } }),
    };
    // Iteration identity must also be reflected in descendant IDs/scopes.
    if (iteration !== undefined) task.scope_path = `${scope}.${iteration}`;
    tasks.push(task);
    return task;
  }
  function visit(steps: Step[], parent: V2PipelineTask, origin: number): number {
    const ends = new Map<string, number>();
    for (const step of steps) {
      const start = Math.max(origin, ...(step.after || []).map((name) => ends.get(name)! + 5));
      const task = add(
        step.name,
        parent,
        step.iterations ? 'LOOP' : step.children ? 'DAG' : 'RUNTIME',
        start,
      );
      let end = start + 30;
      if (step.condition === false) {
        task.state = 'SKIPPED';
        end = start;
      } else if (step.iterations && step.children) {
        task.type_attributes = { iteration_count: String(step.iterations) };
        end = Math.max(
          ...Array.from({ length: step.iterations }, (_, iteration) => {
            const body = add(`${step.name}-body`, task, 'DAG', start + iteration * 3, iteration);
            const finish = visit(step.children!, body, start + iteration * 3 + 1);
            body.end_time = at(finish);
            return finish;
          }),
        );
      } else if (step.children) {
        end = visit(step.children, task, start + 1);
      } else {
        task.inputs = {
          parameters: [
            {
              parameter_key: 'dataset_uri',
              value: `s3://mlpipeline/review-datasets/${step.name}.parquet` as unknown as object,
            },
          ],
        };
        task.outputs = {
          parameters: [{ parameter_key: 'row_count', value: 12480 as unknown as object }],
        };
        task.pods = [
          { name: `${task.task_id}-executor`, uid: `${task.task_id}-pod`, type: 'EXECUTOR' },
        ];
        task.status_metadata = {
          custom_properties: {
            driver_logs_uri:
              `s3://mlpipeline/review-driver-logs/${task.task_id}` as unknown as object,
          },
        };
      }
      task.end_time = at(end);
      task.state_history = [
        ...(task.state === 'SKIPPED'
          ? []
          : [{ state: 'RUNNING' as const, update_time: task.create_time }]),
        { state: task.state, update_time: task.end_time },
      ];
      ends.set(step.name, end);
    }
    return Math.max(origin, ...ends.values());
  }
  const root = add('root', undefined, 'ROOT', 10);
  root.end_time = at(visit(sample.steps, root, 10));
  return tasks;
}
