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

package component

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_parseRuntimeInfo(t *testing.T) {
	tests := []struct {
		name        string
		jsonEncoded string
		want        *runtimeInfo
		wantErr     bool
	}{
		{
			name: "Parse input ints",
			jsonEncoded: `{
				"inputParameters": {
					"my_param": {
						"type": "INT"
					}
				}
			}`,
			want: &runtimeInfo{
				InputParameters: map[string]*inputParameter{
					"my_param": {
						Type: "INT",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parse input strings with quotes",
			jsonEncoded: `{
				"inputParameters": {
					"my_param": {
						"type": "STRING"
					}
				}
			}`,
			want: &runtimeInfo{
				InputParameters: map[string]*inputParameter{
					"my_param": {
						Type: "STRING",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parse serialized dictionaries",
			jsonEncoded: `{
				"inputParameters": {
					"my_param": {
						"type": "STRING"
					}
				}
			}`,
			want: &runtimeInfo{
				InputParameters: map[string]*inputParameter{
					"my_param": {
						Type: "STRING",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parses OutputParameters Correctly",
			jsonEncoded: `{
				"outputParameters": {
					"my_param": {
						"type": "INT",
						"path": "/tmp/outputs/my_param/data"
					}
				}
			}`,
			want: &runtimeInfo{
				OutputParameters: map[string]*outputParameter{
					"my_param": {
						Type: "INT",
						Path: "/tmp/outputs/my_param/data",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parses OutputArtifacts Correctly",
			jsonEncoded: `{
				"outputArtifacts": {
					"my_artifact": {
						"instanceSchema": "properties:\ntitle: kfp.Dataset\ntype: object\n",
					  "metadataPath": "/tmp/outputs/my_artifact/data"
					}
				}
			}`,
			want: &runtimeInfo{
				OutputArtifacts: map[string]*outputArtifact{
					"my_artifact": {
						InstanceSchema: "properties:\ntitle: kfp.Dataset\ntype: object\n",
						MetadataPath:   "/tmp/outputs/my_artifact/data",
					},
				},
			},
			wantErr: false,
		},
		// TODO add tests for input params, artifacts.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRuntimeInfo(tt.jsonEncoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRuntimeInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("parseRuntimeInfo() = %+v, want %+v\nDiff (-want, +got)\n%s", got, tt.want, diff)
				s, _ := json.MarshalIndent(tt.want, "", "  ")
				fmt.Printf("Want %s", s)
			}

		})
	}
}

func TestExecutorInputGeneration(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kfp-launcher-test")
	if err != nil {
		t.Fatal(err)
	}

	dataset_one_path := path.Join(tempDir, "dataset_one")
	dataset_one_contents := `{"id":"1", "typeId":"46", "uri":"gs://some-bucket/dataset-one"}`
	if err := ioutil.WriteFile(dataset_one_path, []byte(dataset_one_contents), 0644); err != nil {
		t.Fatal(err)
	}

	dataset_two_path := path.Join(tempDir, "dataset_two")
	dataset_two_contents := `{"id":"2", "typeId":"46", "uri":"gs://some-bucket/dataset-two"}`
	if err := ioutil.WriteFile(dataset_two_path, []byte(dataset_two_contents), 0644); err != nil {
		t.Fatal(err)
	}

	outputMetadataFilepath := "/tmp/kfp_outputs/output_metadata.json"
	generateOutputUri := func(output string) string {
		return "gs://my-bucket/some-prefix/pipeline/task/" + output
	}

	tests := []struct {
		name        string
		jsonEncoded string
		args        []string
		want        *pipelinespec.ExecutorInput
		wantErr     bool
	}{
		{
			name: "Parses InputParameters Correctly",
			args: []string{
				"message=Some string value with { \"special\": \"chars\" }",
				"num_steps=5",
				"--",
				"sh",
				"-c",
				"user cmd args",
			},
			jsonEncoded: fmt.Sprintf(`{
				"inputParameters": {
					"message": {
						"type": "STRING"
					},
					"num_steps": {
						"type": "NUMBER_INTEGER"
					},
					"list_parameter": {
						"type": "LIST",
						"value": "[1, 2, 3]"
					},
					"dict_parameter": {
						"type": "STRUCT",
						"value": "{\"key_1\": \"value_1\", \"key_2\": 2}"
					}
				},
				"inputArtifacts": {
					"dataset_one": {
						"metadataPath": "%s",
						"schemaTitle": "",
						"instanceSchema": "title: kfp.Dataset\ntype: object\nproperties:\n  payload_format:\n    type: string\n  container_format:\n    type: string\n"
					},
					"dataset_two": {
						"metadataPath": "%s",
						"schemaTitle": "kfp.Model",
						"instanceSchema": ""
					}
				},
				"outputParameters": {
					"output_parameter_one": {
						"type": "STRING",
						"path": "/tmp/outputs/output_parameter_one/data"
					},
					"output_parameter_two": {
						"type": "NUMBER_INTEGER",
						"path": "/tmp/outputs/output_parameter_two/data"
					}
				},
				"outputArtifacts": {
					"model": {
						"schemaTitle": "",
						"instanceSchema": "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n",
						"metadataPath": "/tmp/outputs/model/data"
					},
					"metrics": {
						"schemaTitle": "kfp.Metrics",
						"instanceSchema": "",
						"metadataPath": "/tmp/outputs/metrics/data"
					}
				}
			}`, dataset_one_path, dataset_two_path),
			want: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"message":   structpb.NewStringValue("Some string value with { \"special\": \"chars\" }"),
						"num_steps": structpb.NewNumberValue(5),
						"list_parameter": structpb.NewListValue(&structpb.ListValue{
							Values: []*structpb.Value{
								structpb.NewNumberValue(1),
								structpb.NewNumberValue(2),
								structpb.NewNumberValue(3),
							},
						}),
						"dict_parameter": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key_1": structpb.NewStringValue("value_1"),
								"key_2": structpb.NewNumberValue(2),
							},
						}),
					},
					Artifacts: map[string]*pipelinespec.ArtifactList{
						"dataset_one": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Name: "1",
									Type: &pipelinespec.ArtifactTypeSchema{
										Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Dataset\ntype: object\nproperties:\n  payload_format:\n    type: string\n  container_format:\n    type: string\n"},
									},
									Uri:      "gs://some-bucket/dataset-one",
									Metadata: &structpb.Struct{},
								}}},
						"dataset_two": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Name: "2",
									Type: &pipelinespec.ArtifactTypeSchema{
										Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Model"},
									},
									Uri:      "gs://some-bucket/dataset-two",
									Metadata: &structpb.Struct{},
								}}},
					},
				},
				Outputs: &pipelinespec.ExecutorInput_Outputs{
					Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
						"output_parameter_one": {OutputFile: "/tmp/outputs/output_parameter_one/data"},
						"output_parameter_two": {OutputFile: "/tmp/outputs/output_parameter_two/data"},
					},
					Artifacts: map[string]*pipelinespec.ArtifactList{
						"model": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Name: "model",
									Type: &pipelinespec.ArtifactTypeSchema{
										Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
									},
									Uri:      "gs://my-bucket/some-prefix/pipeline/task/model",
									Metadata: &structpb.Struct{}}}},
						"metrics": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Name: "metrics",
									Type: &pipelinespec.ArtifactTypeSchema{
										Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
									},
									Uri:      "gs://my-bucket/some-prefix/pipeline/task/metrics",
									Metadata: &structpb.Struct{}}}},
					},
					OutputFile: outputMetadataFilepath,
				},
			},
			wantErr: false,
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			rti, err := parseRuntimeInfo(test.jsonEncoded)
			if (err != nil) != test.wantErr {
				t.Errorf("parseRuntimeInfo() error = %v", err)
				return
			}
			_, err = parseArgs(test.args, rti)
			if (err != nil) != test.wantErr {
				t.Errorf("parseArgs() error = %v", err)
			}

			got, err := rti.generateExecutorInput(generateOutputUri, outputMetadataFilepath)
			if (err != nil) != test.wantErr {
				t.Errorf("generateExecutorInput() error = %v", err)
				return
			}

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("generateExecutorInput() =\n%+v\nWant:\n%+v\nDiff (-want, +got)\n%s", got, test.want, diff)
				s, _ := json.MarshalIndent(test.want, "", "  ")
				fmt.Printf("Want\n%s", s)
			}

		})
	}
}
