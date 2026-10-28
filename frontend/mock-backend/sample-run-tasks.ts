// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { complexRunTasks } from './complex-run-samples';
import { ArtifactArtifactType, V2Artifact } from '../src/apisv2/artifact';
import { PipelineTaskTaskState, V2PipelineTask, V2Run } from '../src/apisv2/run';

const steps = [
  { name: 'prepare-data', label: 'Prepare taxi data', dependencies: [], start: 10, end: 120 },
  {
    name: 'train-model',
    label: 'Train fare model',
    dependencies: ['prepare-data'],
    start: 130,
    end: 540,
  },
  {
    name: 'validate-data',
    label: 'Validate features',
    dependencies: ['prepare-data'],
    start: 130,
    end: 300,
  },
  {
    name: 'evaluate-model',
    label: 'Evaluate model',
    dependencies: ['train-model', 'validate-data'],
    start: 550,
    end: 710,
  },
];

// Artifact ports are shared by the saved definition and hydrated runtime records.
const outputPorts: Record<string, Record<string, string>> = {
  'prepare-data': { dataset: 'Dataset' },
  'train-model': { model: 'Model' },
  'evaluate-model': { metrics: 'Metrics' },
};
const inputPorts: Record<string, Record<string, { task: string; key: string; type: string }>> = {
  'train-model': { dataset: { task: 'prepare-data', key: 'dataset', type: 'Dataset' } },
  'validate-data': { dataset: { task: 'prepare-data', key: 'dataset', type: 'Dataset' } },
  'evaluate-model': {
    dataset: { task: 'prepare-data', key: 'dataset', type: 'Dataset' },
    model: { task: 'train-model', key: 'model', type: 'Model' },
  },
};
const artifactDefinition = (type: string) => ({
  artifactType: { schemaTitle: `system.${type}`, schemaVersion: '0.0.1' },
});

// Use one definition for the graph and its runtime tasks so task links resolve.
export const sampleRunPipelineSpec = {
  pipelineInfo: {
    name: 'taxi-fare-training',
    description: 'Synthetic taxi training workflow with parallel training and validation.',
  },
  schemaVersion: '2.1.0',
  sdkVersion: 'kfp-2.0.0',
  root: {
    dag: {
      tasks: Object.fromEntries(
        steps.map((step) => [
          step.name,
          {
            taskInfo: { name: step.name },
            componentRef: { name: `comp-${step.name}` },
            dependentTasks: step.dependencies,
            inputs: {
              artifacts: Object.fromEntries(
                Object.entries(inputPorts[step.name] || {}).map(([key, source]) => [
                  key,
                  {
                    taskOutputArtifact: {
                      producerTask: source.task,
                      outputArtifactKey: source.key,
                    },
                  },
                ]),
              ),
            },
          },
        ]),
      ),
    },
  },
  components: Object.fromEntries(
    steps.map((step) => [
      `comp-${step.name}`,
      {
        executorLabel: `exec-${step.name}`,
        inputDefinitions: {
          artifacts: Object.fromEntries(
            Object.entries(inputPorts[step.name] || {}).map(([key, source]) => [
              key,
              artifactDefinition(source.type),
            ]),
          ),
        },
        outputDefinitions: {
          artifacts: Object.fromEntries(
            Object.entries(outputPorts[step.name] || {}).map(([key, type]) => [
              key,
              artifactDefinition(type),
            ]),
          ),
        },
      },
    ]),
  ),
  deploymentSpec: {
    executors: Object.fromEntries(
      steps.map((step) => [
        `exec-${step.name}`,
        {
          container: {
            image: 'python:3.11-slim',
            command: ['python', '-c'],
            args: [`print("${step.label}")`],
          },
        },
      ]),
    ),
  },
};

export function sampleRunTasks(run: V2Run): V2PipelineTask[] {
  const complex = complexRunTasks(run);
  if (complex) return complex;
  if (!run.run_id?.startsWith('sample-run-') || run.state === 'PENDING') return [];
  const origin = run.created_at!.getTime();
  const at = (seconds: number) => new Date(origin + seconds * 1000);
  const active = run.state === 'RUNNING' || run.state === 'CANCELING';
  const interrupted = run.state === 'FAILED' || run.state === 'CANCELED';
  const rootId = `${run.run_id}-root`;
  const rootState = active ? 'RUNNING' : interrupted ? 'FAILED' : 'SUCCEEDED';
  const tasks: V2PipelineTask[] = [
    {
      run_id: run.run_id,
      task_id: rootId,
      name: 'root',
      display_name: 'Taxi fare training',
      scope_path: 'root',
      type: 'ROOT',
      state: rootState,
      create_time: run.created_at,
      end_time: run.finished_at,
    },
  ];
  for (const [index, step] of steps.entries()) {
    // Tasks have no CANCELED enum: an interrupted executor reports FAILED with
    // a cancellation error, while components that never ran report SKIPPED.
    const state: PipelineTaskTaskState =
      index === 1
        ? active
          ? 'RUNNING'
          : interrupted
            ? 'FAILED'
            : 'SUCCEEDED'
        : index === 3 && (active || interrupted)
          ? 'SKIPPED'
          : 'SUCCEEDED';
    // A downstream task is not created until its dependencies settle.
    if (index === 3 && active) continue;
    const skipped = state === 'SKIPPED';
    const created = skipped ? at(540) : at(step.start);
    const ended = state === 'RUNNING' ? undefined : skipped ? created : at(step.end);
    const taskId = `${run.run_id}-${step.name}`;
    tasks.push({
      run_id: run.run_id,
      task_id: taskId,
      parent_task_id: rootId,
      name: step.name,
      display_name: step.label,
      scope_path: `root.${step.name}`,
      type: 'RUNTIME',
      state,
      create_time: created,
      end_time: ended,
      ...(!skipped
        ? {
            status_metadata: {
              custom_properties: {
                // OpenAPI models protobuf Value as object; its JSON representation is scalar.
                driver_logs_uri:
                  `s3://mlpipeline/review-driver-logs/${taskId}` as unknown as object,
              },
            },
          }
        : {}),
      state_history: [
        ...(!skipped ? [{ state: 'RUNNING' as const, update_time: created }] : []),
        ...(ended
          ? [
              {
                state,
                update_time: ended,
                ...(state === 'FAILED'
                  ? {
                      error: {
                        code: run.state === 'CANCELED' ? 1 : 3,
                        message:
                          run.state === 'CANCELED'
                            ? 'Execution stopped after the run was canceled.'
                            : 'Training data contains invalid feature values.',
                      },
                    }
                  : {}),
              },
            ]
          : []),
      ],
      pods: skipped ? [] : [{ name: `${taskId}-executor`, uid: `${taskId}-pod`, type: 'EXECUTOR' }],
    });
  }
  const artifacts = new Map<string, V2Artifact>();
  for (const task of tasks) {
    if (task.state !== 'SUCCEEDED') continue;
    task.outputs = {
      artifacts: Object.entries(outputPorts[task.name!] || {}).map(([key, type]) => {
        const artifact: V2Artifact = {
          artifact_id: `${task.task_id}-${key}`,
          name:
            key === 'dataset'
              ? 'Prepared taxi features'
              : key === 'model'
                ? 'Taxi fare regressor'
                : 'Evaluation metrics',
          description: `Synthetic ${type.toLowerCase()} produced by ${task.display_name}.`,
          type: type === 'Metrics' ? ArtifactArtifactType.Metric : (type as ArtifactArtifactType),
          ...(type === 'Metrics' ? { number_value: 2.38 } : {}),
          uri: `s3://mlpipeline/review-artifacts/${run.run_id}/${task.name}/${key}`,
          namespace: 'kubeflow-user-example-com',
          created_at: task.end_time,
        };
        artifacts.set(`${task.name}/${key}`, artifact);
        return { artifact_key: key, type: 'OUTPUT', artifacts: [artifact] };
      }),
    };
  }
  for (const task of tasks) {
    if (task.state === 'SKIPPED') continue;
    task.inputs = {
      artifacts: Object.entries(inputPorts[task.name!] || {}).flatMap(([key, source]) => {
        const artifact = artifacts.get(`${source.task}/${source.key}`);
        return artifact
          ? [{ artifact_key: key, type: 'TASK_OUTPUT_INPUT' as const, artifacts: [artifact] }]
          : [];
      }),
    };
  }
  // Automatic retries keep earlier executor pods and distinct log artifacts on the task.
  const retried =
    run.run_id === 'sample-run-import-failed'
      ? tasks.find((task) => task.name === 'train-model')
      : undefined;
  if (retried) {
    retried.pods!.unshift({
      name: `${retried.task_id}-executor-attempt-0`,
      uid: `${retried.task_id}-pod-attempt-0`,
      type: 'EXECUTOR',
    });
    retried.outputs = {
      artifacts: [
        {
          artifact_key: 'executor-logs',
          type: 'OUTPUT',
          artifacts: [0, 1].map((attempt) => ({
            artifact_id: `${retried.task_id}-logs-${attempt}`,
            name: 'executor-logs',
            type: 'Artifact',
            uri: `s3://mlpipeline/review-executor-logs/${retried.task_id}/executor-logs-${attempt}`,
            created_at: at(attempt === 0 ? 200 : 540),
          })),
        },
      ],
    };
  }
  return tasks;
}

// Verbatim INFO message templates from backend/src/v2/driver/container.go.
// These fixtures model cache-disabled driver setup, not component execution:
// a component may fail later even though its driver completed successfully.
export function sampleDriverLogs(task: V2PipelineTask, namespace: string): string {
  const time = task.create_time!.toISOString();
  return [
    `time="${time}" level=info msg="Cache disabled globally at the server level."`,
    `time="${time}" level=info msg="Creating task ${task.name} in pod ${namespace}"`,
    '',
  ].join('\n');
}

// Synthetic component stdout for UI review; these are not KFP driver messages.
export function sampleTaskLogs(task: V2PipelineTask, run: V2Run): string {
  const name = task.name || 'component';
  const start = task.create_time!.getTime();
  const end = task.end_time?.getTime();
  const timestamp = (fraction: number) =>
    new Date(start + (end === undefined ? 30000 : end - start) * fraction).toISOString();
  const operation = /prepare|load|extract/.test(name)
    ? [
        'Reading taxi trip records from input dataset',
        'Removed 128 rows with missing pickup coordinates',
        'Prepared 12480 rows with 8 feature columns',
      ]
    : /normalize|transform/.test(name)
      ? [
          'Loading numeric feature columns',
          'Scaling trip distance and passenger count',
          'Wrote transformed feature dataset',
        ]
      : /encode/.test(name)
        ? [
            'Loading categorical feature columns',
            'Encoding pickup and dropoff zones',
            'Wrote encoded feature dataset',
          ]
        : /validate/.test(name)
          ? [
              'Checking input schema and required columns',
              'Checking feature ranges and missing values',
              'Validation passed for 12480 records',
            ]
          : /aggregate/.test(name)
            ? [
                'Reading metrics from 3 validation folds',
                'Calculating mean absolute error across folds',
                'Mean absolute error: 2.41',
              ]
            : /score|evaluate/.test(name)
              ? [
                  'Loading trained model and evaluation dataset',
                  'Scoring 2496 held-out trips',
                  'Mean absolute error: 2.38; root mean squared error: 3.62',
                ]
              : /train/.test(name)
                ? [
                    'Loading training features and fare_amount target',
                    'Training regressor: 100 estimators, max_depth=6',
                    'Saved trained model artifact',
                  ]
                : [
                    `Starting ${task.display_name || name}`,
                    'Processing component inputs',
                    'Wrote component outputs',
                  ];
  const lines = [
    `${timestamp(0)} INFO Synthetic preview output for ${name}`,
    `${timestamp(0.05)} INFO ${operation[0]}`,
  ];
  const dataset = task.inputs?.parameters?.find(
    (parameter) => parameter.parameter_key === 'dataset_uri',
  );
  if (dataset) lines.push(`${timestamp(0.1)} INFO Input dataset: ${String(dataset.value)}`);
  const rowCount = task.outputs?.parameters?.find(
    (parameter) => parameter.parameter_key === 'row_count',
  );
  if (task.state === 'FAILED') {
    const error =
      [...(task.state_history || [])].reverse().find((entry) => entry.error)?.error?.message ||
      'Component execution failed.';
    if (run.state === 'CANCELED') {
      lines.push(`${timestamp(0.5)} INFO ${operation[1]}`, `${timestamp(1)} WARNING ${error}`);
    } else {
      lines.push(
        `${timestamp(0.5)} ERROR ${error}`,
        'Traceback (most recent call last):',
        '  File "/app/train.py", line 42, in train_model',
        '    validate_training_features(features)',
        '  File "/app/train.py", line 18, in validate_training_features',
        '    raise ValueError("Training data contains invalid feature values.")',
        'ValueError: Training data contains invalid feature values.',
        `${timestamp(1)} ERROR Component exited with code 1; no model artifact produced`,
      );
    }
  } else if (task.state === 'RUNNING') {
    lines.push(
      `${timestamp(0.5)} INFO ${operation[1]}`,
      `${timestamp(1)} INFO ${run.state === 'CANCELING' ? 'Cancellation requested; stopping at the next safe checkpoint' : 'Progress: 40 of 100 estimators fitted'}`,
    );
  } else {
    lines.push(
      `${timestamp(0.5)} INFO ${operation[1]}`,
      `${timestamp(0.9)} INFO ${operation[2]}`,
      ...(rowCount ? [`${timestamp(0.95)} INFO Processed ${String(rowCount.value)} rows`] : []),
      `${timestamp(1)} INFO Component completed successfully`,
    );
  }
  return lines.join('\n') + '\n';
}

// Earlier automatic attempt fails before training; the latest attempt reaches feature validation.
export function sampleRetryTaskLogs(task: V2PipelineTask, run: V2Run, attempt: number): string {
  if (attempt !== 0) return sampleTaskLogs(task, run);
  return [
    `${task.create_time!.toISOString()} INFO Synthetic preview output for ${task.name}`,
    'INFO Loading training dataset from object storage',
    'ERROR Dataset download failed after the read timeout',
    'Traceback (most recent call last):',
    '  File "/app/train.py", line 36, in train_model',
    '    features = download_training_dataset(dataset_uri)',
    'TimeoutError: Timed out reading the training dataset from object storage',
    'ERROR Component exited with code 1; no model artifact produced',
    '',
  ].join('\n');
}
