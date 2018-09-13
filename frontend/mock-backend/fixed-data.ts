// Copyright 2018 Google LLC
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

import helloWorldRun from './hello-world-runtime';
import helloWorldWithStepsRun from './hello-world-with-steps-runtime';
import coinflipRun from './mock-coinflip-runtime';
import errorRun from './mock-error-runtime';
import xgboostRun from './mock-xgboost-runtime';

import { apiJob } from '../src/api/job';
import { apiPipeline } from '../src/api/pipeline';
import { apiRunDetail } from '../src/api/run';

function padStartTwoZeroes(str: string): string {
  let padded = str || '';
  while (padded.length < 2) {
    padded = '0' + padded;
  }
  return padded;
}

// The number of simple, dummy Pipelines, Jobs, and Runs that will be appended to the list.
const NUM_DUMMY_PIPELINES = 30;
const NUM_DUMMY_JOBS = 20;
const NUM_DUMMY_RUNS = 20;

const runs: apiRunDetail[] = [
  {
    run: {
      created_at: new Date('2017-03-17T20:58:23.000Z'),
      id: '3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7',
      name: 'coinflip-recursive-run-lknlfs3',
      namespace: 'namespace',
      scheduled_at: new Date('2017-03-17T20:58:23.000Z'),
      status: 'Failed:Succeeded',
    },
    workflow: JSON.stringify(coinflipRun),
  },
  {
    run: {
      created_at: new Date('2017-04-17T21:00:00.000Z'),
      id: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
      name: 'coinflip-error-nklng2',
      namespace: 'namespace',
      scheduled_at: new Date('2017-04-17T21:00:00.000Z'),
      status: 'Succeeded',
    },
    workflow: JSON.stringify(errorRun),
  },
  {
    run: {
      created_at: new Date('2017-05-17T21:58:23.000Z'),
      id: 'fa5d897e-88d3-4dfc-b189-9dea6947c9bc',
      name: 'hello-world-7sm94',
      namespace: 'namespace',
      scheduled_at: new Date('2017-05-17T21:58:23.000Z'),
      status: 'Running',
    },
    workflow: JSON.stringify(helloWorldRun),
  },
  {
    run: {
      created_at: new Date('2017-06-17T22:58:23.000Z'),
      id: '21afb688-7597-47e9-b6c3-35d3145fe5e1',
      name: 'hello-world-with-steps-kajnkv4',
      namespace: 'namespace',
      scheduled_at: new Date('2017-06-17T22:58:23.000Z'),
      status: 'Failed',
    },
    workflow: JSON.stringify(helloWorldWithStepsRun),
  },
  {
    run: {
      created_at: new Date('2017-07-17T23:58:23.000Z'),
      id: 'a8c471b1-a64e-4713-a660-3899815a40e4',
      name: 'xgboost-evaluation-asdlk2',
      namespace: 'namespace',
      scheduled_at: new Date('2017-07-17T23:58:23.000Z'),
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostRun),
  },
  {
    run: {
      created_at: new Date('2017-08-18T20:58:23.000Z'),
      id: '7fc01714-4a13-4c05-8044-a8a72c14253b',
      name: 'xgboost-run-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-' +
          'loooooooooooooooooooooooooooong-name-aifk298',
      namespace: 'namespace',
      scheduled_at: new Date('2017-08-18T20:58:23.000Z'),
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostRun),
  },
];

runs.push(...generateNRuns());

const pipelines: apiPipeline [] = [
  {
    created_at: new Date('2018-04-01T20:58:23.000Z'),
    description: 'An awesome unstructured text pipeline.',
    id: '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
    name: 'Unstructured text',
    parameters: [
      {
        name: 'x',
      },
      {
        name: 'y',
      },
      {
        name: 'output',
      }
    ]
  },
  {
    created_at: new Date('2018-04-02T20:59:29.000Z'),
    description: 'An awesome image classification pipeline.',
    id: '8fbe3f78-a01f-11e8-98d0-529269fb1459',
    name: 'Image classification',
    parameters: [
      {
        name: 'project',
      },
      {
        name: 'workers',
      },
      {
        name: 'rounds',
      },
      {
        name: 'output',
      }
    ]
  },
  {
    created_at: new Date('2018-04-03T20:58:23.000Z'),
    description: 'This pipeline has no parameters',
    id: '8fbe41b2-a01f-11e8-98d0-529269fb1459',
    name: 'No parameters',
    parameters: [],
  },
  {
    created_at: new Date('2018-04-04T20:58:23.000Z'),
    description: 'This pipeline has undefined parameters',
    id: '8fbe42f2-a01f-11e8-98d0-529269fb1459',
    name: 'Undefined parameters',
    parameters: undefined as any,
  },
  {
    created_at: new Date('2018-04-01T20:59:23.000Z'),
    description: 'Trying to delete this Pipeline will result in an error.',
    id: '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
    name: 'Cannot be deleted',
    parameters: [
      {
        name: 'xx',
      },
      {
        name: 'yy',
      },
      {
        name: 'output',
      }
    ]
  },
];

pipelines.push(...generateNPipelines());

const jobs: apiJob[] = [
  {
    created_at: new Date('2018-03-01T21:58:23.000Z'),
    description: 'This job has no runs',
    enabled: true,
    id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
    max_concurrency: '10',
    name: 'No Runs',
    parameters: [
      {
        name: 'project',
        value: 'my-cloud-project',
      },
      {
        name: 'workers',
        value: '6',
      },
      {
        name: 'rounds',
        value: '25',
      },
      {
        name: 'output',
        value: 'gs://path-to-my-project',
      }
    ],
    pipeline_id: pipelines[0].id,
    status: 'Failed:Succeeded',
    trigger: { cron_schedule: { cron: '30 1 * * * ?' } },
    updated_at: new Date('2018-03-01T21:58:23.000Z'),
  },
  {
    created_at: new Date('2018-03-02T22:58:23.000Z'),
    description: 'This job cannot be deleted',
    enabled: false,
    id: '7fc01714-4a13-4c05-7186-a8a72c14253b',
    max_concurrency: '10',
    name: 'Cannot be deleted - 1',
    parameters: [
      {
        name: 'x',
        value: '10',
      },
      {
        name: 'y',
        value: '20',
      },
      {
        name: 'output',
        value: 'some-output-path',
      }
    ],
    pipeline_id: pipelines[1].id,
    status: 'Succeeded',
    trigger: { cron_schedule: { cron: '0 0 * * * ?' }, },
    updated_at: new Date('2018-03-02T22:58:23.000Z'),
  },
  {
    created_at: new Date('2018-03-03T23:58:23.000Z'),
    description: 'This job cannot be deleted',
    enabled: true,
    id: '7fc01714-4a13-4c05-8957-a8a72c14253b',
    max_concurrency: '10',
    name: 'Cannot be deleted - 2',
    parameters: [
      {
        name: 'project',
        value: 'my-other-cloud-project',
      },
      {
        name: 'workers',
        value: '12',
      },
      {
        name: 'rounds',
        value: '50',
      },
      {
        name: 'output',
        value: 'gs://path-to-my-other-project',
      }
    ],
    pipeline_id: pipelines[2].id,
    status: 'Succeeded',
    trigger: { periodic_schedule: { interval_second: '439652' } },
    updated_at: new Date('2018-03-03T23:58:23.000Z'),
  },
];

jobs.push(...generateNJobs());

function generateNPipelines(): apiPipeline[] {
  const dummyPipelines: apiPipeline[] = [];
  for (let i = pipelines.length + 1; i < NUM_DUMMY_PIPELINES + pipelines.length + 1; i++) {
    dummyPipelines.push({
      created_at: new Date('2018-07-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
      description: `A dummy pipeline (${i})`,
      id: 'Some-pipeline-id-' + i,
      name: 'ML Pipeline number ' + i,
      parameters: [
        {
          name: 'project',
          value: 'my-cloud-project',
        },
      ],
    });
  }
  return dummyPipelines;
}

function generateNRuns(): apiRunDetail[] {
  const dummyRuns: apiRunDetail[] = [];
  for (let i = runs.length + 1; i < NUM_DUMMY_RUNS + runs.length + 1; i++) {
    dummyRuns.push({
      run: {
        created_at: new Date('2018-07-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
        id: 'Some-run-id-' + i,
        name: 'dummy-coinflip-recursive-asdlx' + i,
        namespace: 'namespace',
        scheduled_at: new Date('2018-07-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
        status: 'Succeeded',
      },
      workflow: JSON.stringify(coinflipRun),
    });
  }
  return dummyRuns;
}

function generateNJobs(): apiJob[] {
  const dummyJobs: apiJob[] = [];
  for (let i = jobs.length + 1; i < NUM_DUMMY_JOBS + jobs.length + 1; i++) {
    dummyJobs.push({
      created_at: new Date('2018-04-01T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
      description: 'Some description',
      enabled: false,
      id: 'Some-job-id-' + i,
      max_concurrency: '10',
      name: 'Job#' + i,
      parameters: [
        {
          name: 'project',
          value: 'my-cloud-project',
        },
        {
          name: 'workers',
          value: '6',
        },
        {
          name: 'rounds',
          value: '25',
        },
        {
          name: 'output',
          value: 'gs://path-to-my-project',
        }
      ],
      pipeline_id: pipelines[i % pipelines.length].id,
      status: 'Succeeded',
      trigger: undefined,
      updated_at: new Date('2018-04-01T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
    });
  }
  return dummyJobs;
}

export const data = {
  jobs,
  pipelines,
  runs,
};

export const namedPipelines = {
  examplePipeline: pipelines[0],
  examplePipeline2: pipelines[1],
  noParamsPipeline: pipelines[2],
  undefinedParamsPipeline: pipelines[3],
};
