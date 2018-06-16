const coinflipJob = require('./mock-coinflip-job-runtime.json');
const xgboostJob = require('./mock-xgboost-job-runtime.json');
const errorJob = require('./mock-error-runtime.json');

// The number of simple, dummy Pipelines that will be appended to the list.
const NUM_DUMMY_JOBS = 20;
const NUM_DUMMY_PIPELINES = 20;

const jobs = [
  {
    job: {
      created_at: new Date(1523998703000).toISOString(),
      name: 'coinflip-recursive-job-lknlfs3',
      scheduled_at: new Date(1523998703000).toISOString(),
    },
    workflow: JSON.stringify(coinflipJob),
  },
  {
    job: {
      created_at: new Date(1523998703000).toISOString(),
      name: 'coinflip-error-nklng2',
      scheduled_at: new Date(1523998703000).toISOString(),
    },
    workflow: JSON.stringify(errorJob),
  },
  {
    job: {
      created_at: new Date(1523921868000).toISOString(),
      name: 'xgboost-evaluation-asdlk2',
      scheduled_at: new Date(1523921868000).toISOString(),
    },
    workflow: JSON.stringify(xgboostJob),
  },
  {
    job: {
      created_at: new Date(1523921868000).toISOString(),
      name: 'xgboost-job-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-aifk298',
      scheduled_at: new Date(1523921868000).toISOString(),
    },
    workflow: JSON.stringify(xgboostJob),
  },
];

jobs.push(...generateNJobs());

const examplePackage = {
  id: 1,
  created_at: new Date(1526334129000).toISOString(),
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
  created_at: new Date(1526334129000).toISOString(),
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
  created_at: new Date(1526354129000).toISOString(),
  name: 'No parameters',
  description: 'This package has no parameters',
  parameters: []
};

const undefinedParamsPackage = {
  id: 4,
  created_at: new Date(1526356129000).toISOString(),
  name: 'Undefined parameters',
  description: 'This package has undefined parameters',
  parameters: undefined
}

const data = {
  jobs,
  packages: [ examplePackage, examplePackage2, noParamsPackage, undefinedParamsPackage ],
  pipelines: [
    {
      id: 1,
      created_at: new Date(1526335129000).toISOString(),
      name: 'No Jobs',
      description: 'This pipeline has no jobs',
      package_id: 2,
      schedule: '30 1 * * * ?',
      enabled: true,
      enabled_at: new Date(1483257600000).toISOString(),
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
      id: 2,
      created_at: new Date(1526339129000).toISOString(),
      name: 'Cannot be deleted - 1',
      description: 'This pipeline cannot be deleted',
      package_id: 1,
      schedule: '0 0 * * * ?',
      enabled: false,
      enabled_at: new Date(1483257600000).toISOString(),
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
      id: 3,
      created_at: new Date(1526334129000).toISOString(),
      name: 'Cannot be deleted - 2',
      description: 'This pipeline cannot be deleted',
      package_id: 2,
      schedule: '0 0 0 * * ?',
      enabled: true,
      enabled_at: new Date(1483257600000).toISOString(),
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
        created_at: new Date(1526359129000).toISOString(),
        name: 'coinflip-recursive-asdlx' + i,
        scheduled_at: new Date(1526359129000).toISOString(),
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
      id: i,
      created_at: new Date(1526349129000).toISOString(),
      description: 'Some description',
      name: 'Pipeline#' + i,
      package_id: (i % 6) + 1,
      schedule: '',
      enabled: false,
      enabled_at: -1,
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

