// Copyright 2021 The Kubeflow Authors
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

const TWO_STEP_PIPELINE = `{
    "pipelineSpec": {
      "components": {
        "comp-preprocess": {
          "executorLabel": "exec-preprocess",
          "inputDefinitions": {
            "parameters": {
              "message": {
                "type": "STRING"
              }
            }
          },
          "outputDefinitions": {
            "artifacts": {
              "output_dataset_one": {
                "artifactType": {
                  "schemaTitle": "system.Dataset"
                }
              },
              "output_dataset_two": {
                "artifactType": {
                  "schemaTitle": "system.Dataset"
                }
              }
            },
            "parameters": {
              "output_bool_parameter": {
                "type": "STRING"
              },
              "output_dict_parameter": {
                "type": "STRING"
              },
              "output_list_parameter": {
                "type": "STRING"
              },
              "output_parameter": {
                "type": "STRING"
              }
            }
          }
        },
        "comp-train": {
          "executorLabel": "exec-train",
          "inputDefinitions": {
            "artifacts": {
              "dataset_one": {
                "artifactType": {
                  "schemaTitle": "system.Dataset"
                }
              },
              "dataset_two": {
                "artifactType": {
                  "schemaTitle": "system.Dataset"
                }
              }
            },
            "parameters": {
              "input_bool": {
                "type": "STRING"
              },
              "input_dict": {
                "type": "STRING"
              },
              "input_list": {
                "type": "STRING"
              },
              "message": {
                "type": "STRING"
              },
              "num_steps": {
                "type": "INT"
              }
            }
          },
          "outputDefinitions": {
            "artifacts": {
              "model": {
                "artifactType": {
                  "schemaTitle": "system.Model"
                }
              }
            }
          }
        }
      },
      "deploymentSpec": {
        "executors": {
          "exec-preprocess": {
            "container": {
              "args": [
                "--executor_input",
                "{{$}}",
                "--function_to_execute",
                "preprocess",
                "--message-output-path",
                "{{$.inputs.parameters['message']}}",
                "--output-dataset-one-output-path",
                "{{$.outputs.artifacts['output_dataset_one'].path}}",
                "--output-dataset-two-output-path",
                "{{$.outputs.artifacts['output_dataset_two'].path}}",
                "--output-parameter-output-path",
                "{{$.outputs.parameters['output_parameter'].output_file}}",
                "--output-bool-parameter-output-path",
                "{{$.outputs.parameters['output_bool_parameter'].output_file}}",
                "--output-dict-parameter-output-path",
                "{{$.outputs.parameters['output_dict_parameter'].output_file}}",
                "--output-list-parameter-output-path",
                "{{$.outputs.parameters['output_list_parameter'].output_file}}"
              ],
              "image": "python:3.7"
            }
          },
          "exec-train": {
            "container": {
              "args": [
                "--executor_input",
                "{{$}}",
                "--function_to_execute",
                "train",
                "--dataset-one-output-path",
                "{{$.inputs.artifacts['dataset_one'].path}}",
                "--dataset-two-output-path",
                "{{$.inputs.artifacts['dataset_two'].path}}",
                "--message-output-path",
                "{{$.inputs.parameters['message']}}",
                "--input-bool-output-path",
                "{{$.inputs.parameters['input_bool']}}",
                "--input-dict-output-path",
                "{{$.inputs.parameters['input_dict']}}",
                "--input-list-output-path",
                "{{$.inputs.parameters['input_list']}}",
                "--num-steps-output-path",
                "{{$.inputs.parameters['num_steps']}}",
                "--model-output-path",
                "{{$.outputs.artifacts['model'].path}}"
              ],
              "image": "python:3.7"
            }
          }
        }
      },
      "pipelineInfo": {
        "name": "my-test-pipeline-beta"
      },
      "root": {
        "dag": {
          "tasks": {
            "preprocess": {
              "cachingOptions": {
                "enableCache": true
              },
              "componentRef": {
                "name": "comp-preprocess"
              },
              "inputs": {
                "parameters": {
                  "message": {
                    "componentInputParameter": "message"
                  }
                }
              },
              "taskInfo": {
                "name": "preprocess"
              }
            },
            "train": {
              "cachingOptions": {
                "enableCache": true
              },
              "componentRef": {
                "name": "comp-train"
              },
              "dependentTasks": [
                "preprocess"
              ],
              "inputs": {
                "artifacts": {
                  "dataset_one": {
                    "taskOutputArtifact": {
                      "outputArtifactKey": "output_dataset_one",
                      "producerTask": "preprocess"
                    }
                  },
                  "dataset_two": {
                    "taskOutputArtifact": {
                      "outputArtifactKey": "output_dataset_two",
                      "producerTask": "preprocess"
                    }
                  }
                },
                "parameters": {
                  "input_bool": {
                    "taskOutputParameter": {
                      "outputParameterKey": "output_bool_parameter",
                      "producerTask": "preprocess"
                    }
                  },
                  "input_dict": {
                    "taskOutputParameter": {
                      "outputParameterKey": "output_dict_parameter",
                      "producerTask": "preprocess"
                    }
                  },
                  "input_list": {
                    "taskOutputParameter": {
                      "outputParameterKey": "output_list_parameter",
                      "producerTask": "preprocess"
                    }
                  },
                  "message": {
                    "taskOutputParameter": {
                      "outputParameterKey": "output_parameter",
                      "producerTask": "preprocess"
                    }
                  },
                  "num_steps": {
                    "runtimeValue": {
                      "constantValue": {
                        "intValue": "100"
                      }
                    }
                  }
                }
              },
              "taskInfo": {
                "name": "train"
              }
            }
          }
        },
        "inputDefinitions": {
          "parameters": {
            "message": {
              "type": "STRING"
            }
          }
        }
      },
      "schemaVersion": "2.0.0",
      "sdkVersion": "kfp-1.6.4"
    },
    "runtimeConfig": {
      "gcsOutputDirectory": "dummy_root"
    }
  }
`;

export { TWO_STEP_PIPELINE };
