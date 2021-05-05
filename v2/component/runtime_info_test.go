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
	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
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
			name: "Parses InputParameters Correctly",
			jsonEncoded: `{
				"inputParameters": {
					"my_param": {
						"type": "INT",
						"value": "123"
					}
				}
			}`,
			want: &runtimeInfo{
				InputParameters: map[string]*inputParameter{
					"my_param": {
						Type:  "INT",
						Value: "123",
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
		want        *pipeline_spec.ExecutorInput
		wantErr     bool
	}{
		{
			name: "Parses InputParameters Correctly",
			jsonEncoded: fmt.Sprintf(`{
				"inputParameters": {
					"message": {
						"type": "STRING",
						"value": "Some string value"
					},
					"num_steps": {
						"type": "INT",
						"value": "5"
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
						"type": "INT",
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
			want: &pipeline_spec.ExecutorInput{
				Inputs: &pipeline_spec.ExecutorInput_Inputs{
					Parameters: map[string]*pipeline_spec.Value{
						"message":   {Value: &pipeline_spec.Value_StringValue{StringValue: "Some string value"}},
						"num_steps": {Value: &pipeline_spec.Value_IntValue{IntValue: 5}},
					},
					Artifacts: map[string]*pipeline_spec.ArtifactList{
						"dataset_one": {
							Artifacts: []*pipeline_spec.RuntimeArtifact{
								{
									Name: "1",
									Type: &pipeline_spec.ArtifactTypeSchema{
										Kind: &pipeline_spec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Dataset\ntype: object\nproperties:\n  payload_format:\n    type: string\n  container_format:\n    type: string\n"},
									},
									Uri:      "gs://some-bucket/dataset-one",
									Metadata: &structpb.Struct{},
								}}},
						"dataset_two": {
							Artifacts: []*pipeline_spec.RuntimeArtifact{
								{
									Name: "2",
									Type: &pipeline_spec.ArtifactTypeSchema{
										Kind: &pipeline_spec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Model"},
									},
									Uri:      "gs://some-bucket/dataset-two",
									Metadata: &structpb.Struct{},
								}}},
					},
				},
				Outputs: &pipeline_spec.ExecutorInput_Outputs{
					Parameters: map[string]*pipeline_spec.ExecutorInput_OutputParameter{
						"output_parameter_one": {OutputFile: "/tmp/outputs/output_parameter_one/data"},
						"output_parameter_two": {OutputFile: "/tmp/outputs/output_parameter_two/data"},
					},
					Artifacts: map[string]*pipeline_spec.ArtifactList{
						"model": {
							Artifacts: []*pipeline_spec.RuntimeArtifact{
								{
									Name: "model",
									Type: &pipeline_spec.ArtifactTypeSchema{
										Kind: &pipeline_spec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
									},
									Uri: "gs://my-bucket/some-prefix/pipeline/task/model",
									Metadata: &structpb.Struct{
										Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
									}}}},
						"metrics": {
							Artifacts: []*pipeline_spec.RuntimeArtifact{
								{
									Name: "metrics",
									Type: &pipeline_spec.ArtifactTypeSchema{
										Kind: &pipeline_spec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
									},
									Uri: "gs://my-bucket/some-prefix/pipeline/task/metrics",
									Metadata: &structpb.Struct{
										Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
									}}}},
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

			got, err := rti.generateExecutorInput(generateOutputUri, outputMetadataFilepath)
			if (err != nil) != test.wantErr {
				t.Errorf("generateExecutorInput() error = %v", err)
				return
			}

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("generateExecutorInput() = %+v, want %+v\nDiff (-want, +got)\n%s", got, test.want, diff)
				s, _ := json.MarshalIndent(test.want, "", "  ")
				fmt.Printf("Want\n%s", s)
			}

		})
	}
}
