const coinflipRun = require('./mock-coinflip-runtime.json');
const xgboostRun = require('./mock-xgboost-runtime.json');
const errorRun = require('./mock-error-runtime.json');
const helloWorldRun = require('./hello-world-runtime.json');
const helloWorldWithStepsRun = require('./hello-world-with-steps-runtime.json');

// The number of simple, dummy Runs and Jobs that will be appended to the list.
const NUM_DUMMY_RUNS = 20;
const NUM_DUMMY_JOBS = 20;

const runs = [
  {
    run: {
      created_at: "2017-03-17T20:58:23.000Z",
      id: '3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7',
      name: 'coinflip-recursive-run-lknlfs3',
      namespace: 'namespace',
      scheduled_at: "2017-03-17T20:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(coinflipRun),
  },
  {
    run: {
      created_at: "2017-04-17T21:00:00.000Z",
      id: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
      name: 'coinflip-error-nklng2',
      namespace: 'namespace',
      scheduled_at: "2017-04-17T21:00:00.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(errorRun),
  },
  {
    run: {
      created_at: "2017-05-17T21:58:23.000Z",
      id: 'fa5d897e-88d3-4dfc-b189-9dea6947c9bc',
      name: 'hello-world-7sm94',
      namespace: 'namespace',
      scheduled_at: "2017-05-17T21:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(helloWorldRun),
  },
  {
    run: {
      created_at: "2017-06-17T22:58:23.000Z",
      id: '21afb688-7597-47e9-b6c3-35d3145fe5e1',
      name: 'hello-world-with-steps-kajnkv4',
      namespace: 'namespace',
      scheduled_at: "2017-06-17T22:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(helloWorldWithStepsRun),
  },
  {
    run: {
      created_at: "2017-07-17T23:58:23.000Z",
      id: 'a8c471b1-a64e-4713-a660-3899815a40e4',
      name: 'xgboost-evaluation-asdlk2',
      namespace: 'namespace',
      scheduled_at: "2017-07-17T23:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostRun),
  },
  {
    run: {
      created_at: "2017-08-18T20:58:23.000Z",
      id: '7fc01714-4a13-4c05-8044-a8a72c14253b',
      name: 'xgboost-run-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-aifk298',
      namespace: 'namespace',
      scheduled_at: "2017-08-18T20:58:23.000Z",
      status: 'Succeeded',
    },
    workflow: JSON.stringify(xgboostRun),
  },
];

runs.push(...generateNRuns());

const examplePipeline = {
  id: 1,
  created_at: "2018-04-01T20:58:23.000Z",
  name: 'Unstructured text',
  description: 'An awesome unstructured text pipeline.',
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

const examplePipeline2 = {
  id: 2,
  created_at: "2018-04-02T20:58:23.000Z",
  name: 'Image classification',
  description: 'An awesome image classification pipeline.',
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

const noParamsPipeline = {
  id: 3,
  created_at: "2018-04-03T20:58:23.000Z",
  name: 'No parameters',
  description: 'This pipeline has no parameters',
  parameters: []
};

const undefinedParamsPipeline = {
  id: 4,
  created_at: "2018-04-04T20:58:23.000Z",
  name: 'Undefined parameters',
  description: 'This pipeline has undefined parameters',
  parameters: undefined
}

const data = {
  runs,
  pipelines: [ examplePipeline, examplePipeline2, noParamsPipeline, undefinedParamsPipeline ],
  jobs: [
    {
      id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
      created_at: "2018-03-01T21:58:23.000Z",
      name: 'No Runs',
      description: 'This job has no runs',
      pipeline_id: 2,
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
      runs: [],
    },
    {
      id: '7fc01714-4a13-4c05-7186-a8a72c14253b',
      created_at: "2018-03-02T22:58:23.000Z",
      name: 'Cannot be deleted - 1',
      description: 'This job cannot be deleted',
      pipeline_id: 1,
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
      runs,
    },
    {
      id: '7fc01714-4a13-4c05-8957-a8a72c14253b',
      created_at: "2018-03-03T23:58:23.000Z",
      name: 'Cannot be deleted - 2',
      description: 'This job cannot be deleted',
      pipeline_id: 2,
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
      runs,
    },
  ],
};

function generateNRuns() {
  const dummyRuns = [];
  for (i = runs.length + 1; i < NUM_DUMMY_RUNS + runs.length + 1; i++) {
    dummyRuns.push( {
      run: {
        id: 'Some-run-id-' + i,
        created_at: '2018-07-12T20:' + i.toString().padStart(2, '0') + ':23.000Z',
        name: 'dummy-coinflip-recursive-asdlx' + i,
        namespace: 'namespace',
        scheduled_at: '2018-07-12T20:' + i.toString().padStart(2, '0') + ':23.000Z',
        status: 'Succeeded',
      },
      workflow: JSON.stringify(coinflipRun),
    });
  }
  return dummyRuns;
}

function generateNJobs() {
  jobs = [];
  for (i = data.jobs.length + 1; i < NUM_DUMMY_JOBS + data.jobs.length + 1; i++) {
    jobs.push( {
      id: 'Some-job-id-' + i,
      created_at: '2018-04-01T20:' + i.toString().padStart(2, '0') + ':23.000Z',
      description: 'Some description',
      name: 'Job#' + i,
      pipeline_id: (i % 6) + 1,
      status: 'Succeeded',
      trigger: undefined,
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
      runs,
    });
  }
  return jobs;
}

data.jobs.push(...generateNJobs());

module.exports = {
  data,
  namedPipelines: {
    examplePipeline,
    examplePipeline2,
    noParamsPipeline,
    undefinedParamsPipeline
  }
};

