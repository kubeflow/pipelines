const fs = require('fs');

// The number of simple, dummy Pipelines that will be appended to the list.
const NUM_DUMMY_PIPELINES = 15;

const coinflipJob =
    JSON.parse(fs.readFileSync(`./mock-backend/mock-coinflip-job-runtime.json`, 'utf-8'));

const xgboostJob =
    JSON.parse(fs.readFileSync(`./mock-backend/mock-xgboost-job-runtime.json`, 'utf-8'));

jobs = [
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

fixedData = {
  packages: [
    {
      id: 0,
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
    },
    {
      id: 1,
      createdAt: 1526334129,
      name: 'Image classification',
      description: 'An awesome image classification pipeline package.',
      parameters: [
        {
          name: 'x',
          description: 'The x parameter'
        },
        {
          name: 'y',
          description: 'The y parameter'
        },
        {
          name: 'output',
          description: 'The base output path',
        }
      ]
    }
  ],
  pipelines: [
    {
      id: 1,
      createdAt: 1526335129,
      name: 'No Jobs',
      description: 'This pipeline has no jobs',
      packageId: 1,
      schedule: '30 1 * * * ?',
      enabled: true,
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
      packageId: 1,
      schedule: '0 0 0 * * ?',
      enabled: true,
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
      jobs: [],
    },
  ],
};

function generateNPipelines() {
  pipelines = [];
  for (i = fixedData.pipelines.length; i < NUM_DUMMY_PIPELINES + fixedData.pipelines.length; i++) {
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
          name: 'x',
          value: 10
        },
        {
          name: 'output',
          value: 'some-output-path',
        }
      ],
      jobs,
    });
  }
  return pipelines;
}

fixedData.pipelines.push(...generateNPipelines());

module.exports = fixedData;

