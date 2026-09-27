// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import fs from 'fs';
import YAML from 'yaml';
import { V2beta1Experiment } from '../src/apisv2beta1/experiment';
import { V2beta1Pipeline, V2beta1PipelineVersion } from '../src/apisv2beta1/pipeline';
import { V2beta1Run } from '../src/apisv2beta1/run';
import { V2beta1RecurringRun } from '../src/apisv2beta1/recurringrun';

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
const pipelines: V2beta1Pipeline[] = examples.map(([id, name]) => ({
  pipeline_id: id,
  name,
  display_name: name,
  created_at: new Date('2026-01-01'),
}));
const versions: V2beta1PipelineVersion[] = examples.map(([id, , file]) => ({
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
const experiments: V2beta1Experiment[] = [
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
const runs: V2beta1Run[] = examples.map(([, name], i) => ({
  run_id: i === 2 ? 'e0115ac1-0479-4194-a22d-01e65e09a32b' : `mock-run-${i}`,
  display_name: i === 2 ? 'v2-xgboost-ilbo' : name,
  pipeline_spec: versions[i].pipeline_spec,
  experiment_id: experiments[0].experiment_id,
  created_at: new Date('2026-01-01T00:00:00Z'),
  finished_at: new Date('2026-01-01T00:03:00Z'),
  state: 'SUCCEEDED',
  storage_state: 'AVAILABLE',
}));
const recurringRuns: V2beta1RecurringRun[] = [0, 1].map((i) => ({
  recurring_run_id: `mock-recurring-${i}`,
  display_name: `Scheduled ${examples[i][1]}`,
  experiment_id: experiments[1].experiment_id,
  pipeline_spec: versions[i].pipeline_spec,
  status: 'ENABLED',
  trigger: { periodic_schedule: { interval_second: '3600' } },
}));
export const data = { experiments, pipelines, versions, runs, recurringRuns };
