module.exports = {
  packages: [
    {
      id: 0,
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
      description: 'Try 10 for x',
      id: 1,
      name: 'Unstructured text experiment 1',
      packageId: 1,
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
      createdAt: 1517087010898,
      recurring: false,
      jobs: [
        {
          name: 'unstructured-text-experiment-asdlk2',
          createdAt: '2018-03-08T01:55:12Z',
          startedAt: '0001-01-01T00:00:00Z',
          finishedAt: '0001-01-01T00:00:00Z',
          status: 'Succeeded',
          _pipelineId: 1,
        },
        {
          name: 'test-job-asd32',
          createdAt: '2018-03-08T01:55:12Z',
          startedAt: '0001-01-01T00:00:00Z',
          finishedAt: '0001-01-01T00:00:00Z',
          _pipelineId: 1,
        },
        {
          name: 'a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-of-test-job-3',
          createdAt: '2018-03-08T01:55:12Z',
          startedAt: '0001-01-01T00:00:00Z',
          finishedAt: '0001-01-01T00:00:00Z',
          status: 'Running',
          _pipelineId: 1,
        },
      ],
    },
    {
      description: 'Try 10 and 20 for parameters',
      id: 2,
      name: 'Unstructured text experiment 2',
      packageId: 1,
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
      createdAt: 1517087010898,
      recurring: false,
      jobs: [
        {
          name: 'unstructured-text-experiment-4',
          createdAt: '2018-03-08T01:55:12Z',
          startedAt: '0001-01-01T00:00:00Z',
          finishedAt: '0001-01-01T00:00:00Z',
          status: 'Error',
          _pipelineId: 2,
        },
      ],
    },
  ]
};
