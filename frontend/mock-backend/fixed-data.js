const coinflipJob = require('./mock-coinflip-job-runtime.json');
const xgboostJob = require('./mock-xgboost-job-runtime.json');

// The number of simple, dummy Pipelines that will be appended to the list.
const NUM_DUMMY_PIPELINES = 15;

const jobs = [
  {
    metadata: {
      id: 1,
      createdAt: 1523998703,
      name: 'coinflip-recursive-job-lknlfs3',
      scheduledAt: 1523998703,
    },
    jobDetail: coinflipJob,
  },
  {
    metadata: {
      id: 2,
      createdAt: 1523921868,
      name: 'xgboost-evaluation-asdlk2',
      scheduledAt: 1523921868,
    },
    jobDetail: xgboostJob,
  },
  {
    metadata: {
      id: 3,
      createdAt: 1523921868,
      name: 'xgboost-job-with-a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-aifk298',
      scheduledAt: 1523921868,
    },
    jobDetail: xgboostJob,
  },
];

const examplePackage = {
  id: 1,
  createdAt: 1526334129,
  name: 'Unstructured text',
  description: 'An awesome unstructured text pipeline package.',
  parameters: [
    {
      name: 'x',
      description: 'The x parameter description.'
    },
    {
      name: 'y',
      description: 'The y parameter description. This can be very long!'
    },
    {
      name: 'output',
      description: 'The base output path',
    }
  ]
};

const examplePackage2 = {
  id: 2,
  createdAt: 1526334129,
  name: 'Image classification',
  description: 'An awesome image classification pipeline package.',
  parameters: [
    {
      name: 'project',
      description: 'The name of the GCP project'
    },
    {
      name: 'workers',
      description: 'The number of workers'
    },
    {
      name: 'rounds',
      description: 'The number of rounds'
    },
    {
      name: 'output',
      description: 'The base output path',
    }
  ]
};

const noParamsPackage = {
  id: 3,
  createdAt: 1526354129,
  name: 'No parameters',
  description: 'This package has no parameters',
  parameters: []
};

const undefinedParamsPackage = {
  id: 4,
  createdAt: 1526356129,
  name: 'Undefined parameters',
  description: 'This package has undefined parameters',
  parameters: undefined
}

const data = {
  packages: [ examplePackage, examplePackage2, noParamsPackage, undefinedParamsPackage ],
  pipelines: [
    {
      id: 1,
      createdAt: 1526335129,
      name: 'No Jobs',
      description: 'This pipeline has no jobs',
      packageId: 2,
      schedule: '30 1 * * * ?',
      enabled: true,
      enabledAt: 1483257600,
      parameters: [
        {
          name: 'project',
          value: 'my-cloud-project'
        },
        {
          name: 'workers',
          value: 6
        },
        {
          name: 'rounds',
          value: 25
        },
        {
          name: 'output',
          value: 'gs://path-to-my-project',
        }
      ],
      jobs,
    },
    {
      id: 2,
      createdAt: 1526339129,
      name: 'Cannot be deleted - 1',
      description: 'This pipeline cannot be deleted',
      packageId: 1,
      schedule: '0 0 * * * ?',
      enabled: false,
      enabledAt: 1483257600,
      parameters: [
        {
          name: 'x',
          value: 10
        },
        {
          name: 'y',
          value: 20
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
      createdAt: 1526334129,
      name: 'Cannot be deleted - 2',
      description: 'This pipeline cannot be deleted',
      packageId: 2,
      schedule: '0 0 0 * * ?',
      enabled: true,
      enabledAt: 1483257600,
      parameters: [
        {
          name: 'project',
          value: 'my-other-cloud-project'
        },
        {
          name: 'workers',
          value: 12
        },
        {
          name: 'rounds',
          value: 50
        },
        {
          name: 'output',
          value: 'gs://path-to-my-other-project',
        }
      ],
      jobs: [],
    },
  ],
};

function generateNPipelines() {
  pipelines = [];
  for (i = data.pipelines.length; i < NUM_DUMMY_PIPELINES + data.pipelines.length; i++) {
    pipelines.push( {
      id: i,
      createdAt: 1526349129,
      description: 'Some description',
      name: 'Pipeline#' + i,
      packageId: (i % 6) + 1,
      schedule: '',
      enabled: false,
      enabledAt: -1,
      parameters: [
        {
          name: 'project',
          value: 'my-cloud-project'
        },
        {
          name: 'workers',
          value: 6
        },
        {
          name: 'rounds',
          value: 25
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

