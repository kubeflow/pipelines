// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import fs from 'fs';
import { complexPipelineSpec, complexRunSamples } from './complex-run-samples';
import { sampleRunPipelineSpec } from './sample-run-tasks';
import YAML from 'yaml';
import { V2Experiment } from '../src/apisv2/experiment';
import { V2Pipeline, V2PipelineVersion } from '../src/apisv2/pipeline';
import { V2Run, V2RuntimeState } from '../src/apisv2/run';
import { V2RecurringRun } from '../src/apisv2/recurringrun';

const examples = [
  [
    '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
    'Python two steps',
    'lightweight_python_functions_v2_pipeline.json',
  ],
  [
    '8fbe3bd6-a01f-11e8-98d0-529269fb1461',
    'Loops and conditions',
    'pipeline_with_loops_and_conditions.yaml',
  ],
  ['8fbe3bd6-a01f-11e8-98d0-529269fb1462', 'XGBoost', 'xgboost_sample_pipeline.yaml'],
  [
    '8fbe3bd6-a01f-11e8-98d0-529269fb2222',
    'Various IO types',
    'pipeline_with_various_io_types.yaml',
  ],
];
const pipelines: V2Pipeline[] = examples.map(([id, name]) => ({
  pipeline_id: id,
  name,
  display_name: name,
  created_at: new Date('2026-01-01'),
}));
const versions: V2PipelineVersion[] = examples.map(([id, , file]) => ({
  pipeline_id: id,
  pipeline_version_id: id,
  display_name: 'default version',
  created_at: new Date('2026-01-01'),
  pipeline_spec: YAML.parse(
    fs.readFileSync(new URL(`./data/v2/pipeline/${file}`, import.meta.url), 'utf8'),
  ),
}));
versions.push({
  pipeline_id: examples[0][0],
  pipeline_version_id: '9fbe3bd6-a01f-11e8-98d0-529269fb1460',
  display_name: 'revision',
  created_at: new Date('2026-02-01'),
  pipeline_spec: YAML.parse(
    fs.readFileSync(
      new URL(
        './data/v2/pipeline/lightweight_python_functions_v2_pipeline_rev.yaml',
        import.meta.url,
      ),
      'utf8',
    ),
  ),
});
const taxiPipelineId = 'sample-taxi-pipeline';
pipelines.push({
  pipeline_id: taxiPipelineId,
  name: 'taxi-fare-training',
  display_name: 'Taxi fare training',
  description: 'Taxi training workflow with parallel training and validation.',
  created_at: new Date('2026-01-01'),
});
for (const version of ['v1.0', 'v1.1']) {
  versions.push({
    pipeline_id: taxiPipelineId,
    pipeline_version_id: `sample-taxi-${version}`,
    display_name: version,
    created_at: new Date(version === 'v1.0' ? '2026-01-01' : '2026-02-01'),
    pipeline_spec: {
      ...sampleRunPipelineSpec,
      pipelineInfo: {
        ...sampleRunPipelineSpec.pipelineInfo,
        description: `Taxi training ${version}`,
      },
    },
  });
}

for (const sample of complexRunSamples) {
  pipelines.push({
    pipeline_id: sample.id,
    name: sample.id,
    display_name: sample.name,
    created_at: new Date('2026-01-01'),
  });
  versions.push({
    pipeline_id: sample.id,
    pipeline_version_id: `${sample.id}-v1`,
    display_name: 'v1.0',
    created_at: new Date('2026-01-01'),
    pipeline_spec: complexPipelineSpec(sample),
  });
}

const experiments: V2Experiment[] = [
  {
    experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
    display_name: 'Default',
    storage_state: 'AVAILABLE',
  },
  {
    experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
    display_name: 'Recurring',
    storage_state: 'AVAILABLE',
  },
  { experiment_id: 'empty-experiment', display_name: 'No runs', storage_state: 'AVAILABLE' },
  { experiment_id: 'archived-experiment', display_name: 'Archived', storage_state: 'ARCHIVED' },
];
const runs: V2Run[] = examples.map(([, name], i) => ({
  run_id: i === 2 ? 'e0115ac1-0479-4194-a22d-01e65e09a32b' : `mock-run-${i}`,
  display_name: i === 2 ? 'v2-xgboost-ilbo' : name,
  pipeline_spec: versions[i].pipeline_spec,
  experiment_id: experiments[0].experiment_id,
  created_at: new Date('2026-01-01T00:00:00Z'),
  finished_at: new Date('2026-01-01T00:03:00Z'),
  state: 'SUCCEEDED',
  storage_state: 'AVAILABLE',
}));
const recurringRuns: V2RecurringRun[] = [0, 1].map((i) => ({
  recurring_run_id: `mock-recurring-${i}`,
  created_at: new Date('2026-01-01T00:00:00Z'),
  display_name: `Scheduled ${examples[i][1]}`,
  experiment_id: experiments[1].experiment_id,
  pipeline_spec: versions[i].pipeline_spec,
  status: 'ENABLED',
  trigger: { periodic_schedule: { interval_second: '3600' } },
}));
// Relative windows keep review examples visibly scheduled/expired over time.
function reviewSchedules(): V2RecurringRun[] {
  const today = new Date();
  today.setUTCHours(0, 0, 0, 0);
  const offset = (days: number) => new Date(today.getTime() + days * 86_400_000);
  const schedules: Pick<V2RecurringRun, 'display_name' | 'status' | 'trigger'>[] = [
    {
      display_name: 'Daily taxi training',
      status: 'ENABLED',
      trigger: { cron_schedule: { cron: '0 0 9 * * *' } },
    },
    {
      display_name: 'Weekday validation',
      status: 'ENABLED',
      trigger: { cron_schedule: { cron: '0 30 8 * * MON-FRI' } },
    },
    {
      display_name: 'Paused data refresh',
      status: 'DISABLED',
      trigger: { periodic_schedule: { interval_second: '900' } },
    },
    {
      display_name: 'Upcoming daily training',
      status: 'ENABLED',
      trigger: {
        periodic_schedule: {
          interval_second: '86400',
          start_time: offset(7),
          end_time: offset(37),
        },
      },
    },
    {
      display_name: 'Completed evaluation schedule',
      status: 'ENABLED',
      trigger: {
        periodic_schedule: {
          interval_second: '7200',
          start_time: offset(-30),
          end_time: offset(-1),
        },
      },
    },
  ];
  return schedules.map((schedule, index) => ({
    ...schedule,
    recurring_run_id: `sample-recurring-${index}`,
    created_at: offset(-30),
    experiment_id: experiments[1].experiment_id,
    pipeline_spec: sampleRunPipelineSpec,
  }));
}

const sampleRuns: {
  id: string;
  name: string;
  state: V2RuntimeState;
  daysAgo: number;
  archived?: boolean;
}[] = [
  { id: 'training-running', name: 'Taxi fare training', state: 'RUNNING', daysAgo: 0 },
  { id: 'validation-pending', name: 'Validation queued', state: 'PENDING', daysAgo: 0 },
  { id: 'daily-success', name: 'Daily taxi data refresh', state: 'SUCCEEDED', daysAgo: 1 },
  { id: 'training-failed', name: 'Training with invalid features', state: 'FAILED', daysAgo: 2 },
  { id: 'sweep-canceled', name: 'Hyperparameter sweep', state: 'CANCELED', daysAgo: 4 },
  { id: 'refresh-canceling', name: 'Superseded data refresh', state: 'CANCELING', daysAgo: 0 },
  { id: 'weekly-success', name: 'Weekly model evaluation', state: 'SUCCEEDED', daysAgo: 6 },
  { id: 'import-failed', name: 'Dataset import retry', state: 'FAILED', daysAgo: 12 },
  { id: 'monthly-success', name: 'Monthly baseline comparison', state: 'SUCCEEDED', daysAgo: 24 },
  { id: 'backfill-canceled', name: 'Historical taxi backfill', state: 'CANCELED', daysAgo: 45 },
  { id: 'evaluation-failed', name: 'Previous model evaluation', state: 'FAILED', daysAgo: 75 },
  { id: 'baseline-success', name: 'Original fare baseline', state: 'SUCCEEDED', daysAgo: 120 },
  {
    id: 'archived-success',
    name: 'Archived weekly training',
    state: 'SUCCEEDED',
    daysAgo: 3,
    archived: true,
  },
  {
    id: 'archived-failed',
    name: 'Archived import failure',
    state: 'FAILED',
    daysAgo: 40,
    archived: true,
  },
];

// Roll review samples forward daily so the relative-date filters remain useful.
// The original fixed runs above retain their IDs, dates, and task relationships.
function reviewRuns(): V2Run[] {
  const today = new Date();
  today.setUTCHours(0, 0, 0, 0);
  return sampleRuns.map((sample) => {
    const createdAt = new Date(today.getTime() - sample.daysAgo * 86_400_000 - 3_600_000);
    const terminal = ['SUCCEEDED', 'FAILED', 'CANCELED'].includes(sample.state);
    return {
      run_id: `sample-run-${sample.id}`,
      display_name: sample.name,
      experiment_id: experiments[0].experiment_id,
      // Keep both submission styles represented; referenced runs also carry their resolved IR.
      ...(['sweep-canceled', 'import-failed'].includes(sample.id)
        ? { pipeline_spec: sampleRunPipelineSpec }
        : {
            pipeline_version_reference: {
              pipeline_id: taxiPipelineId,
              pipeline_version_id: `sample-taxi-${sample.daysAgo >= 24 ? 'v1.0' : 'v1.1'}`,
            },
            pipeline_spec: versions.find(
              (version) =>
                version.pipeline_version_id ===
                `sample-taxi-${sample.daysAgo >= 24 ? 'v1.0' : 'v1.1'}`,
            )!.pipeline_spec,
          }),
      created_at: createdAt,
      finished_at: terminal ? new Date(createdAt.getTime() + 12 * 60_000) : undefined,
      state_history:
        sample.state === 'PENDING'
          ? []
          : [
              { state: 'RUNNING', update_time: new Date(createdAt.getTime() + 10_000) },
              ...(terminal
                ? [
                    {
                      state: sample.state,
                      update_time: new Date(createdAt.getTime() + 12 * 60_000),
                    },
                  ]
                : []),
            ],
      state: sample.state,
      storage_state: sample.archived ? 'ARCHIVED' : 'AVAILABLE',
    };
  });
}

export const data = {
  experiments,
  pipelines,
  versions,
  get runs() {
    return [
      ...runs,
      ...reviewRuns(),
      ...complexRunSamples.map((sample): V2Run => {
        const created = new Date();
        created.setUTCHours(0, 0, 0, 0);
        const finished = new Date(created.getTime() + 12 * 60_000);
        return {
          run_id: `sample-run-${sample.id}`,
          display_name: sample.name,
          experiment_id: experiments[0].experiment_id,
          pipeline_version_reference: {
            pipeline_id: sample.id,
            pipeline_version_id: `${sample.id}-v1`,
          },
          pipeline_spec: complexPipelineSpec(sample),
          created_at: created,
          finished_at: finished,
          state: 'SUCCEEDED',
          storage_state: 'AVAILABLE',
          state_history: [
            { state: 'RUNNING', update_time: new Date(created.getTime() + 10_000) },
            { state: 'SUCCEEDED', update_time: finished },
          ],
        };
      }),
    ];
  },
  get recurringRuns() {
    return [...recurringRuns, ...reviewSchedules()];
  },
};
