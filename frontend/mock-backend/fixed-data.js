const coinflipJob = require('./mock-coinflip-job-runtime.json');
const xgboostJob = require('./mock-xgboost-job-runtime.json');
const errorJob = require('./mock-error-runtime.json');
const helloWorldJob = require('./hello-world-runtime.json');
const helloWorldWithStepsJob = require('./hello-world-with-steps-runtime.json');

// The number of simple, dummy Pipelines that will be appended to the list.
const NUM_DUMMY_JOBS = 20;
const NUM_DUMMY_PIPELINES = 20;

const jobs = [
  {
    job: {
      created_at: "2017-03-17T20:58:23.000Z",
      id: '3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7',
      name: 'coinflip-recursive-job-lknlfs3',
      namespace: 'namespace',
      scheduled_at: "2017-03-17T20:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(coinflipJob),
  },
  {
    job: {
      created_at: "2017-04-17T21:00:00.000Z",
      id: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
      name: 'coinflip-error-nklng2',
      namespace: 'namespace',
      scheduled_at: "2017-04-17T21:00:00.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(errorJob),
  },
  {
    job: {
      created_at: "2017-05-17T21:58:23.000Z",
      id: 'fa5d897e-88d3-4dfc-b189-9dea6947c9bc',
      name: 'hello-world-7sm94',
      namespace: 'namespace',
      scheduled_at: "2017-05-17T21:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(helloWorldJob),
  },
  {
    job: {
      created_at: "2017-06-17T22:58:23.000Z",
      id: '21afb688-7597-47e9-b6c3-35d3145fe5e1',
      name: 'hello-world-with-steps-kajnkv4',
      namespace: 'namespace',
      scheduled_at: "2017-06-17T22:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(helloWorldWithStepsJob),
  },
  {
    job: {
      created_at: "2017-07-17T23:58:23.000Z",
      id: 'a8c471b1-a64e-4713-a660-3899815a40e4',
      name: 'xgboost-evaluation-asdlk2',
      namespace: 'namespace',
      scheduled_at: "2017-07-17T23:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostJob),
  },
  {
    job: {
      created_at: "2017-08-18T20:58:23.000Z",
      id: '7fc01714-4a13-4c05-8044-a8a72c14253b',
      name: 'xgboost-job-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-aifk298',
      namespace: 'namespace',
      scheduled_at: "2017-08-18T20:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostJob),
  },
];

jobs.push(...generateNJobs());

const examplePackage = {
  id: 1,
  created_at: "2018-04-01T20:58:23.000Z",
  name: 'Unstructured text',
  description: 'An awesome unstructured text pipeline package.',
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
};

const examplePackage2 = {
  id: 2,
  created_at: "2018-04-02T20:58:23.000Z",
  name: 'Image classification',
  description: 'An awesome image classification pipeline package.',
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
};

const noParamsPackage = {
  id: 3,
  created_at: "2018-04-03T20:58:23.000Z",
  name: 'No parameters',
  description: 'This package has no parameters',
  parameters: []
};

const undefinedParamsPackage = {
  id: 4,
  created_at: "2018-04-04T20:58:23.000Z",
  name: 'Undefined parameters',
  description: 'This package has undefined parameters',
  parameters: undefined
}

const data = {
  jobs,
  packages: [ examplePackage, examplePackage2, noParamsPackage, undefinedParamsPackage ],
  pipelines: [
    {
      id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
      created_at: "2018-03-01T21:58:23.000Z",
      name: 'No Jobs',
      description: 'This pipeline has no jobs',
      package_id: 2,
      status: 'Succeeded',
      trigger: { cron_schedule: { cron: '30 1 * * * ?' } },
      enabled: true,
      updated_at: "2018-03-01T21:58:23.000Z",
      max_concurrency: 10,
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
      jobs: [],
    },
    {
      id: '7fc01714-4a13-4c05-7186-a8a72c14253b',
      created_at: "2018-03-02T22:58:23.000Z",
      name: 'Cannot be deleted - 1',
      description: 'This pipeline cannot be deleted',
      package_id: 1,
      status: 'Succeeded',
      trigger: { cron_schedule: { cron: '0 0 * * * ?' }, },
      enabled: false,
      updated_at: "2018-03-02T22:58:23.000Z",
      max_concurrency: 10,
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
      jobs,
    },
    {
      id: '7fc01714-4a13-4c05-8957-a8a72c14253b',
      created_at: "2018-03-03T23:58:23.000Z",
      name: 'Cannot be deleted - 2',
      description: 'This pipeline cannot be deleted',
      package_id: 2,
      status: 'Succeeded',
      trigger: { periodic_schedule: { interval_second: 439652 } },
      enabled: true,
      updated_at: "2018-03-03T23:58:23.000Z",
      max_concurrency: 10,
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
      jobs,
    },
  ],
};

function generateNJobs() {
  dummyJobs = [];
  for (i = jobs.length + 1; i < NUM_DUMMY_JOBS + jobs.length + 1; i++) {
    dummyJobs.push( {
      job: {
        // TODO: this should be a UUID.
        id: 'Some-job-id-' + i,
        created_at: '2018-07-12T20:' + i.toString().padStart(2, '0') + ':23.000Z',
        name: 'dummy-coinflip-recursive-asdlx' + i,
        namespace: 'namespace',
        scheduled_at: '2018-07-12T20:' + i.toString().padStart(2, '0') + ':23.000Z',
        status: 'Succeeded',
      },
      workflow: JSON.stringify(coinflipJob),
    });
  }
  return dummyJobs;
}

function generateNPipelines() {
  pipelines = [];
  for (i = data.pipelines.length + 1; i < NUM_DUMMY_PIPELINES + data.pipelines.length + 1; i++) {
    pipelines.push( {
      // TODO: this should be a UUID.
      id: 'Some-pipeline-id-' + i,
      created_at: '2018-04-01T20:' + i.toString().padStart(2, '0') + ':23.000Z',
      description: 'Some description',
      name: 'Pipeline#' + i,
      package_id: (i % 6) + 1,
      status: 'Succeeded',
      trigger: null,
      enabled: false,
      updated_at: '2018-04-01T20:' + i.toString().padStart(2, '0') + ':23.000Z',
      max_concurrency: 10,
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
      jobs,
    });
  }
  return pipelines;
}

data.pipelines.push(...generateNPipelines());

module.exports = {
  data,
  namedPackages: {
    examplePackage,
    examplePackage2,
    noParamsPackage,
    undefinedParamsPackage
  }
};

