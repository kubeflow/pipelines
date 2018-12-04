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
import { ApiExperiment } from '../src/apis/experiment';
import { ApiJob } from '../src/apis/job';
import { ApiPipeline } from '../src/apis/pipeline';
import { ApiRunDetail, ApiResourceType, ApiRelationship, RunMetricFormat } from '../src/apis/run';

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

const pipelines: ApiPipeline[] = [
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

const jobs: ApiJob[] = [
  {
    created_at: new Date('2018-03-01T21:58:23.000Z'),
    description: 'This job has no runs',
    enabled: true,
    id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
    max_concurrency: '10',
    name: 'No Runs',
    pipeline_spec: {
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
    },
    resource_references: [{
      key: {
        id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
        type: ApiResourceType.EXPERIMENT,
      },
      relationship: ApiRelationship.OWNER,
    }],
    status: 'Failed:Succeeded',
    trigger: {
      cron_schedule: {
        cron: '30 1 * * * ?',
        end_time: new Date('2018-04-01T21:58:23.000Z'),
        start_time: new Date('2018-03-01T21:58:23.000Z'),
      }
    },
    updated_at: new Date('2018-03-01T21:58:23.000Z'),
  },
  {
    created_at: new Date('2018-03-02T22:58:23.000Z'),
    description: 'This job cannot be deleted',
    enabled: false,
    id: '7fc01714-4a13-4c05-7186-a8a72c14253b',
    max_concurrency: '10',
    name: 'Cannot be deleted - 1',
    pipeline_spec: {
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
    },
    resource_references: [{
      key: {
        id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
        type: ApiResourceType.EXPERIMENT,
      },
      relationship: ApiRelationship.OWNER,
    }],
    status: 'Succeeded',
    trigger: {
      cron_schedule: {
        cron: '0 0 * * * ?',
        start_time: new Date('2018-03-01T21:58:23.000Z'),
      },
    },
    updated_at: new Date('2018-03-02T22:58:23.000Z'),
  },
  {
    created_at: new Date('2018-03-03T23:58:23.000Z'),
    description: 'This job cannot be deleted',
    enabled: true,
    id: '7fc01714-4a13-4c05-8957-a8a72c14253b',
    max_concurrency: '10',
    name: 'Cannot be deleted - 2',
    pipeline_spec: {
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
    },
    resource_references: [{
      key: {
        id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
        type: ApiResourceType.EXPERIMENT,
      },
      relationship: ApiRelationship.OWNER,
    }],
    status: 'Succeeded',
    trigger: {
      periodic_schedule: {
        end_time: new Date('2018-03-03T23:58:23.000Z'),
        interval_second: '439652',
      }
    },
    updated_at: new Date('2018-03-03T23:58:23.000Z'),
  },
];

jobs.push(...generateNJobs());

const experiments: ApiExperiment[] = [
  {
    description: 'This experiment has no runs',
    id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
    name: 'No Runs',
  },
  {
    description: 'A Pipeline experiment with runs',
    id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
    name: 'Kubeflow Pipelines Experiment',
  },
  {
    description: 'A different Pipeline experiment used to group runs. ' +
      'This experiment also has a very long description, which should overflow the container card.',
    id: 'a4d4f8c6-ce9c-4200-a92e-c48ec759b733',
    name: 'Experiment Number 2',
  },
];

const runs: ApiRunDetail[] = [
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(coinflipRun),
    },
    run: {
      created_at: new Date('2018-03-17T20:58:23.000Z'),
      description: 'A recursive coinflip run',
      id: '3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7',
      metrics: [
        {
          format: RunMetricFormat.PERCENTAGE,
          name: 'accuracy',
          number_value: 0.6762,
        },
        {
          format: RunMetricFormat.RAW,
          name: 'log_loss',
          number_value: -0.573,
        }
      ],
      name: 'coinflip-recursive-run-lknlfs3',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
      },
      resource_references: [{
        key: {
          id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-03-17T20:58:23.000Z'),
      status: 'Failed:Succeeded',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(errorRun),
    },
    run: {
      error: 'Mock error retrieving this run. All fields will be empty besides ID and this',
      id: 'f9486999-e853-40ee-993d-a0199b2cb7bd',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(errorRun),
    },
    run: {
      created_at: new Date('2018-04-17T21:00:00.000Z'),
      description: 'A coinflip run with an error. No metrics',
      id: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
      metrics: [
        {
          format: RunMetricFormat.PERCENTAGE,
          name: 'accuracy',
          number_value: 0.5512,
        },
        {
          format: RunMetricFormat.RAW,
          name: 'log_loss',
          number_value: -0.78,
        }
      ],
      name: 'coinflip-error-nklng2',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
      },
      resource_references: [{
        key: {
          id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-04-17T21:00:00.000Z'),
      status: 'Succeeded',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(helloWorldRun),
    },
    run: {
      created_at: new Date('2018-05-17T21:58:23.000Z'),
      description: 'A simple hello world run',
      id: 'fa5d897e-88d3-4dfc-b189-9dea6947c9bc',
      metrics: [{
        format: RunMetricFormat.PERCENTAGE,
        name: 'accuracy',
        number_value: 0.5423,
      }],
      name: 'hello-world-7sm94',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe41b2-a01f-11e8-98d0-529269fb1459',
      },
      resource_references: [{
        key: {
          id: 'a4d4f8c6-ce9c-4200-a92e-c48ec759b733',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-05-17T21:58:23.000Z'),
      status: 'Running',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(helloWorldWithStepsRun),
    },
    run: {
      created_at: new Date('2018-06-17T22:58:23.000Z'),
      description: 'A simple hello world run, but with steps. Not attached to any experiment',
      id: '21afb688-7597-47e9-b6c3-35d3145fe5e1',
      metrics: [{
        format: RunMetricFormat.PERCENTAGE,
        name: 'accuracy',
        number_value: 0.43,
      }],
      name: 'hello-world-with-steps-kajnkv4',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe42f2-a01f-11e8-98d0-529269fb1459',
      },
      scheduled_at: new Date('2018-06-17T22:58:23.000Z'),
      status: 'Failed',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(xgboostRun),
    },
    run: {
      created_at: new Date('2018-07-17T23:58:23.000Z'),
      description: 'An xgboost evaluation run',
      id: 'a8c471b1-a64e-4713-a660-3899815a40e4',
      metrics: [
        {
          format: RunMetricFormat.RAW,
          name: 'numeric_metric',
          node_id: 'some node ID',
          number_value: 24,
        },
        {
          format: RunMetricFormat.PERCENTAGE,
          name: 'accuracy',
          number_value: 0.95675,
        },
        {
          format: RunMetricFormat.UNSPECIFIED,
          name: 'unspecified format metric',
          node_id: 'one more node ID',
          number_value: 1.34,
        },
      ],
      name: 'xgboost-evaluation-asdlk2',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe3f78-a01f-11e8-98d0-529269fb1459',
      },
      resource_references: [{
        key: {
          id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-07-17T23:58:23.000Z'),
      status: 'Pending',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(xgboostRun),
    },
    run: {
      created_at: new Date('2018-08-18T20:58:23.000Z'),
      description: 'An xgboost evaluation run with a very long description that includes:'
        + ' Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent fermentum commodo'
        + ' libero, a imperdiet ipsum cursus id. Nullam odio sem, ornare id sollicitudin ac,'
        + ' rutrum in dolor. Integer interdum lacus in ex rutrum elementum. Mauris gravida feugiat'
        + ' enim, ac dapibus augue rhoncus in. Integer vel tempus nulla. Cras sed ultrices dolor.'
        + ' Ut nec dapibus eros, vitae iaculis nunc. In aliquet accumsan rhoncus. Donec vitae'
        + ' ipsum a tellus fermentum pharetra in in neque. Pellentesque consequat felis non est'
        + ' vulputate pellentesque. Aliquam eget cursus enim.',
      id: '7fc01714-4a13-4c05-8044-a8a72c14253b',
      metrics: [
        {
          format: RunMetricFormat.PERCENTAGE,
          name: 'accuracy',
          number_value: 0.8999,
        },
        {
          format: RunMetricFormat.RAW,
          name: 'log_loss',
          number_value: -0.123,
        }
      ],
      name: 'xgboost-run-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-' +
        'loooooooooooooooooooooooooooong-name-aifk298',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        pipeline_id: '8fbe3f78-a01f-11e8-98d0-529269fb1459',
      },
      resource_references: [{
        key: {
          id: 'a4d4f8c6-ce9c-4200-a92e-c48ec759b733',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-08-18T20:58:23.000Z'),
      status: 'Succeeded',
    },
  },
  {
    pipeline_runtime: {
      workflow_manifest: JSON.stringify(helloWorldRun),
    },
    run: {
      created_at: new Date('2018-08-18T20:58:23.000Z'),
      description: 'simple run with pipeline spec embedded in it.',
      id: '7fc01715-4a93-4c00-8044-a8a72c14253b',
      metrics: [
        {
          format: RunMetricFormat.PERCENTAGE,
          name: 'accuracy',
          number_value: 0.5999,
        },
        {
          format: RunMetricFormat.RAW,
          name: 'log_loss',
          number_value: -0.223,
        }
      ],
      name: 'hello-world-with-pipeline',
      namespace: 'namespace',
      pipeline_spec: {
        parameters: [
          { name: 'paramName1', value: 'paramVal1' },
          { name: 'paramName2', value: 'paramVal2' },
        ],
        workflow_manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  serviceAccountName: pipeline-runner
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`,
      },
      resource_references: [{
        key: {
          id: 'a4d4f8c6-ce9c-4200-a92e-c48ec759b733',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      scheduled_at: new Date('2018-08-18T20:58:23.000Z'),
      status: 'Succeeded',
    },
  },
];

runs.push(...generateNRuns());

function generateNPipelines(): ApiPipeline[] {
  const dummyPipelines: ApiPipeline[] = [];
  for (let i = pipelines.length + 1; i < NUM_DUMMY_PIPELINES + pipelines.length + 1; i++) {
    dummyPipelines.push({
      created_at: new Date('2018-02-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
      description: `A dummy pipeline (${i})`,
      id: 'Some-pipeline-id-' + i,
      name: 'Kubeflow Pipeline number ' + i,
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

function generateNRuns(): ApiRunDetail[] {
  const dummyRuns: ApiRunDetail[] = [];
  for (let i = runs.length + 1; i < NUM_DUMMY_RUNS + runs.length + 1; i++) {
    dummyRuns.push({
      pipeline_runtime: {
        workflow_manifest: JSON.stringify(coinflipRun),
      },
      run: {
        created_at: new Date('2018-02-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
        description: 'The description of a dummy run',
        id: 'Some-run-id-' + i,
        metrics: [
          {
            format: RunMetricFormat.RAW,
            name: 'numeric_metric',
            node_id: 'some node ID',
            number_value: i,
          },
          {
            format: RunMetricFormat.PERCENTAGE,
            name: 'accuracy',
            node_id: 'another node ID',
            number_value: ((i + 50) % 100) / 100.0,
          },
          {
            format: RunMetricFormat.UNSPECIFIED,
            name: 'unspecified format metric',
            node_id: 'one more node ID',
            number_value: i + 0.43,
          },
        ],
        name: 'dummy-coinflip-recursive-asdlx' + i,
        namespace: 'namespace',
        pipeline_spec: {
          parameters: [
            { name: 'paramName1', value: 'paramVal1' },
            { name: 'paramName2', value: 'paramVal2' },
          ],
          pipeline_id: 'Some-pipeline-id-' + i,
        },
        resource_references: [{
          key: {
            id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        }],
        scheduled_at: new Date('2018-02-12T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
        status: 'Succeeded',
      },
    });
  }
  return dummyRuns;
}

function generateNJobs(): ApiJob[] {
  const dummyJobs: ApiJob[] = [];
  for (let i = jobs.length + 1; i < NUM_DUMMY_JOBS + jobs.length + 1; i++) {
    dummyJobs.push({
      created_at: new Date('2018-02-01T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
      description: 'Some description',
      enabled: false,
      id: 'Some-job-id-' + i,
      max_concurrency: '10',
      name: 'Job#' + i,
      pipeline_spec: {
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
      },
      resource_references: [{
        key: {
          id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      }],
      status: 'Succeeded',
      trigger: undefined,
      updated_at: new Date('2018-02-01T20:' + padStartTwoZeroes(i.toString()) + ':23.000Z'),
    });
  }
  return dummyJobs;
}

export const data = {
  experiments,
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
