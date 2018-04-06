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
          metadata: {
            name: 'xgboost-evaluation-asdlk2',
            generateName: 'xgboost-evaluation-',
            namespace: 'default',
            uid: '0ba1d914-2d57-11e8-acba-42010a8a01d3',
            resourceVersion: '2224268',
            creationTimestamp: '2018-03-08T01:55:12Z',
          },
          spec: {
            templates: [
              {
                name: 'xgboost-evaluation',
                inputs: {
                  parameters: [
                    {
                      name: 'project'
                    },
                  ]
                },
                outputs: {},
                metadata: {},
                steps: [
                  [
                    {
                      name: 'transform',
                      template: 'transform',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          },
                          {
                            name: 'region',
                            value: '{{inputs.parameters.region}}'
                          },
                          {
                            name: 'cluster',
                            value: '{{inputs.parameters.cluster}}'
                          },
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/transform'
                          },
                          {
                            name: 'eval',
                            value: '{{inputs.parameters.eval}}'
                          },
                          {
                            name: 'target',
                            value: '{{inputs.parameters.target}}'
                          },
                          {
                            name: 'analysis',
                            value: '{{inputs.parameters.analysis}}'
                          }
                        ]
                      }
                    }
                  ],
                  [
                    {
                      name: 'batchpredict',
                      template: 'batchpredict',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          },
                          {
                            name: 'region',
                            value: '{{inputs.parameters.region}}'
                          },
                          {
                            name: 'cluster',
                            value: '{{inputs.parameters.cluster}}'
                          },
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict'
                          },
                          {
                            name: 'eval',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/transform/eval/part-*'
                          },
                          {
                            name: 'target',
                            value: '{{inputs.parameters.target}}'
                          },
                          {
                            name: 'analysis',
                            value: '{{inputs.parameters.analysis}}'
                          },
                          {
                            name: 'package',
                            value: '{{inputs.parameters.package}}'
                          },
                          {
                            name: 'model',
                            value: '{{inputs.parameters.model}}'
                          }
                        ]
                      }
                    }
                  ],
                  [
                    {
                      name: 'roc',
                      template: 'roc',
                      arguments: {
                        parameters: [
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/roc'
                          },
                          {
                            name: 'predictions',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict/part-*'
                          },
                          {
                            name: 'trueclass',
                            value: '{{inputs.parameters.trueclass}}'
                          }
                        ]
                      }
                    },
                    {
                      name: 'confusionmatrix',
                      template: 'confusionmatrix',
                      arguments: {
                        parameters: [
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/confusionmatrix'
                          },
                          {
                            name: 'predictions',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict/part-*.csv'
                          },
                          {
                            name: 'analysis',
                            value: '{{inputs.parameters.analysis}}'
                          },
                          {
                            name: 'target',
                            value: '{{inputs.parameters.target}}'
                          }
                        ]
                      }
                    }
                  ]
                ]
              },
            ],
            entrypoint: 'xgboost-evaluation',
            arguments: {
              parameters: [
                {
                  name: 'project',
                  value: 'some-project1'
                },
                {
                  name: 'region',
                  value: 'us-central1'
                },
                {
                  name: 'cluster',
                  value: ''
                },
                {
                  name: 'output',
                  value: 'gs://some-project1/tmp'
                },
              ]
            }
          },
          status: {
            phase: 'Succeeded',
            startedAt: '2018-03-21T22:27:32Z',
            finishedAt: '2018-03-21T22:27:34Z',
          }
        },
        {
          metadata: {
            name: 'test-job-lknlfs3',
            generateName: 'test-job-',
            namespace: 'default',
            uid: '737b170d-ef74-4bbb-81e3-76b696f003e7',
            resourceVersion: '1879452',
            creationTimestamp: '2018-03-24T04:55:18Z',
          },
          spec: {
            templates: [
              {
                name: 'test-job',
                inputs: {
                  parameters: [
                    {
                      name: 'project'
                    }
                  ]
                },
                outputs: {},
                metadata: {},
                steps: [
                  [
                    {
                      name: 'transform',
                      template: 'transform',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          }
                        ]
                      },
                    }
                  ]
                ]
              },
              {
                name: 'transform',
                inputs: {
                  parameters: [
                    {
                      'name': 'project'
                    }
                  ]
                },
                outputs: {},
                metadata: {},
                container: {
                  name: '',
                  image: 'gcr.io/some/path/to/an/image',
                  command: [
                    'sh',
                    '-c'
                  ],
                  args: [
                    'python /ml/transform.py --project {{inputs.parameters.project}} --region {{inputs.parameters.region}} --cluster {{inputs.parameters.cluster}} --output {{inputs.parameters.output}} --eval {{inputs.parameters.eval}} --target {{inputs.parameters.target}} --analysis {{inputs.parameters.analysis}}'
                  ],
                  resources: {}
                }
              },
            ],
            entrypoint: 'test-job',
          },
          status: {
            phase: 'Running',
            startedAt: '2018-03-25T22:27:32Z',
            finishedAt: '2018-03-25T22:27:34Z',
          }
        },
        {
          metadata: {
            name: 'a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-of-test-job-aifk298',
            generateName: 'a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-of-test-job-',
            namespace: 'default',
            uid: '65eb315e-1e51-47c6-82d5-9ae091a8c85a',
            resourceVersion: '2085832',
            creationTimestamp: '2018-03-23T14:55:18Z',
          },
          spec: {
            templates: [
              {
                name: 'a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-of-test-job',
                inputs: {
                  parameters: [
                    {
                      name: 'project'
                    }
                  ]
                },
                outputs: {},
                metadata: {},
                steps: [
                  [
                    {
                      name: 'transform',
                      template: 'transform',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          }
                        ]
                      },
                    }
                  ]
                ]
              },
              {
                name: 'transform',
                inputs: {
                  parameters: [
                    {
                      'name': 'project'
                    }
                  ]
                },
                outputs: {},
                metadata: {},
                container: {
                  name: '',
                  image: 'gcr.io/some/path/to/an/image',
                  command: [
                    'sh',
                    '-c'
                  ],
                  args: [
                    'python /ml/transform.py --project {{inputs.parameters.project}} --region {{inputs.parameters.region}} --cluster {{inputs.parameters.cluster}} --output {{inputs.parameters.output}} --eval {{inputs.parameters.eval}} --target {{inputs.parameters.target}} --analysis {{inputs.parameters.analysis}}'
                  ],
                  resources: {}
                }
              },
            ],
            entrypoint: 'a-veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeery-loooooooooooooooooooooooooooong-name-of-test-job',
          },
          status: {
            phase: 'Error',
            startedAt: '2018-03-24T12:27:32Z',
            finishedAt: '2018-03-24T14:27:40Z',
          }
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
          metadata: {
            name: 'unstructured-text-experiment-lknlfs3',
            generateName: 'unstructured-text-experiment-',
            namespace: 'default',
            uid: 'ba0c08d3-af89-22a5-caab-198329039123',
            resourceVersion: '1112222',
            creationTimestamp: '2018-03-23T04:55:18Z',
          },
          spec: {
            templates: [
              {
                name: 'unstructured-text-experiment',
                inputs: {
                  parameters: [
                    {
                      name: 'project'
                    },
                  ]
                },
                outputs: {},
                metadata: {},
                steps: [
                  [
                    {
                      name: 'transform',
                      template: 'transform',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          },
                        ]
                      }
                    }
                  ],
                  [
                    {
                      name: 'batchpredict',
                      template: 'batchpredict',
                      arguments: {
                        parameters: [
                          {
                            name: 'project',
                            value: '{{inputs.parameters.project}}'
                          },
                        ]
                      }
                    }
                  ],
                  [
                    {
                      name: 'roc',
                      template: 'roc',
                      arguments: {
                        parameters: [
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/roc'
                          },
                          {
                            name: 'predictions',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict/part-*'
                          },
                          {
                            name: 'trueclass',
                            value: '{{inputs.parameters.trueclass}}'
                          }
                        ]
                      }
                    },
                    {
                      name: 'tensorboard',
                      template: 'tensorboard',
                      arguments: {
                        parameters: [
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/tensorboard'
                          }
                        ]
                      }
                    },
                    {
                      name: 'confusionmatrix',
                      template: 'confusionmatrix',
                      arguments: {
                        parameters: [
                          {
                            name: 'output',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/confusionmatrix'
                          },
                          {
                            name: 'predictions',
                            value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict/part-*.csv'
                          },
                          {
                            name: 'analysis',
                            value: '{{inputs.parameters.analysis}}'
                          },
                          {
                            name: 'target',
                            value: '{{inputs.parameters.target}}'
                          }
                        ]
                      }
                    }
                  ]
                ]
              },
            ],
            entrypoint: 'unstructured-text-experiment',
            arguments: {
              parameters: [
                {
                  name: 'project',
                  value: 'some-project2'
                },
                {
                  name: 'region',
                  value: 'us-central1'
                },
                {
                  name: 'cluster',
                  value: ''
                },
                {
                  name: 'output',
                  value: 'gs://some-project2/tmp'
                },
              ]
            }
          },
          status: {
            phase: 'Error',
            startedAt: '2018-03-23T22:27:32Z',
            finishedAt: '2018-03-23T22:27:34Z',
          }
        },
      ],
    },
  ]
};
