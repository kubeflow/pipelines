package cacheutils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kubeflow/pipelines/api/kfp_pipeline_spec/go/cachekey"
	"github.com/kubeflow/pipelines/api/kfp_pipeline_spec/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGenerateCacheKey(t *testing.T) {

	tests := []struct {
		name                    string
		executorInputInputs     *pipelinespec.ExecutorInput_Inputs
		executorInputOutputs    *pipelinespec.ExecutorInput_Outputs
		outputParametersTypeMap map[string]string
		cmdArgs                 []string
		image                   string
		want                    *cachekey.CacheKey
		wantErr                 bool
	}{
		{
			name: "Generate CacheKey Correctly",
			executorInputInputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"message":   {Kind: &structpb.Value_StringValue{StringValue: "Some string value"}},
					"num_steps": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
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
							}}}},
			},

			executorInputOutputs: &pipelinespec.ExecutorInput_Outputs{
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
								Uri: "gs://my-bucket/some-prefix/pipeline/task/model",
								Metadata: &structpb.Struct{
									Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
								}}}},
					"metrics": {
						Artifacts: []*pipelinespec.RuntimeArtifact{
							{
								Name: "metrics",
								Type: &pipelinespec.ArtifactTypeSchema{
									Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
								},
								Uri: "gs://my-bucket/some-prefix/pipeline/task/metrics",
								Metadata: &structpb.Struct{
									Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
								}}}},
				},
				OutputFile: "/tmp/kfp_outputs/output_metadata.json",
			},
			outputParametersTypeMap: map[string]string{
				"output_parameter_one": "STRING",
				"output_parameter_two": "INT",
			},
			cmdArgs: []string{"sh", "ec", "test"},
			image:   "python:3.9",
			want: &cachekey.CacheKey{
				InputArtifactNames: map[string]*cachekey.ArtifactNameList{
					"dataset_one": {ArtifactNames: []string{"1"}},
					"dataset_two": {ArtifactNames: []string{"2"}},
				},
				InputParameterValues: map[string]*structpb.Value{
					"message":   {Kind: &structpb.Value_StringValue{StringValue: "Some string value"}},
					"num_steps": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				},
				OutputArtifactsSpec: map[string]*pipelinespec.RuntimeArtifact{
					"model": {
						Name: "model",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
						}},
					"metrics": {
						Name: "metrics",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
						}},
				},
				OutputParametersSpec: map[string]string{
					"output_parameter_one": "STRING",
					"output_parameter_two": "INT",
				},
				ContainerSpec: &cachekey.ContainerSpec{
					CmdArgs: []string{"sh", "ec", "test"},
					Image:   "python:3.9",
				},
			},

			wantErr: false,
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			got, err := GenerateCacheKey(test.executorInputInputs, test.executorInputOutputs, test.outputParametersTypeMap, test.cmdArgs, test.image)
			if (err != nil) != test.wantErr {
				t.Errorf("GenerateCacheKey() error = %v", err)
				return
			}

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("GenerateCacheKey() = %+v, want %+v\nDiff (-want, +got)\n%s", got, test.want, diff)
				s, _ := json.MarshalIndent(test.want, "", "  ")
				fmt.Printf("Want\n%s", s)
			}

		})
	}
}

func TestGenerateFingerPrint(t *testing.T) {
	cacheKey := &cachekey.CacheKey{
		InputArtifactNames: map[string]*cachekey.ArtifactNameList{
			"dataset_one": {ArtifactNames: []string{"1"}},
			"dataset_two": {ArtifactNames: []string{"2"}},
		},
		InputParameterValues: map[string]*structpb.Value{
			"message":   {Kind: &structpb.Value_StringValue{StringValue: "Some string value"}},
			"num_steps": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
		},
		OutputArtifactsSpec: map[string]*pipelinespec.RuntimeArtifact{
			"model": {
				Name: "model",
				Type: &pipelinespec.ArtifactTypeSchema{
					Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
				},
				Metadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
				}},
			"metrics": {
				Name: "metrics",
				Type: &pipelinespec.ArtifactTypeSchema{
					Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
				},
				Metadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
				}},
		},
		OutputParametersSpec: map[string]string{
			"output_parameter_one": "STRING",
			"output_parameter_two": "INT",
		},
		ContainerSpec: &cachekey.ContainerSpec{
			CmdArgs: []string{"sh", "ec", "test"},
			Image:   "python:3.9",
		},
	}
	tests := []struct {
		name        string
		cacheKey    *cachekey.CacheKey
		wantEqual   bool
		fingerPrint string
	}{
		{
			name: "Generated Same FingerPrint",
			cacheKey: &cachekey.CacheKey{
				InputArtifactNames: map[string]*cachekey.ArtifactNameList{
					"dataset_one": {ArtifactNames: []string{"1"}},
					"dataset_two": {ArtifactNames: []string{"2"}},
				},
				InputParameterValues: map[string]*structpb.Value{
					"message":   {Kind: &structpb.Value_StringValue{StringValue: "Some string value"}},
					"num_steps": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				},
				OutputArtifactsSpec: map[string]*pipelinespec.RuntimeArtifact{
					"model": {
						Name: "model",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
						}},
					"metrics": {
						Name: "metrics",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
						}},
				},
				OutputParametersSpec: map[string]string{
					"output_parameter_one": "STRING",
					"output_parameter_two": "INT",
				},
				ContainerSpec: &cachekey.ContainerSpec{
					CmdArgs: []string{"sh", "ec", "test"},
					Image:   "python:3.9",
				},
			},
			wantEqual:   true,
			fingerPrint: "4e8a5d7d70997b0a35429fcd481af8fcd5b9f58ef4391bdb6ad900fd1c63622b",
		}, {
			name: "Generated Different FingerPrint",
			cacheKey: &cachekey.CacheKey{
				InputArtifactNames: map[string]*cachekey.ArtifactNameList{
					"dataset": {ArtifactNames: []string{"10"}},
				},
				OutputParametersSpec: map[string]string{
					"output_parameter": "DOUBLE",
				},
				ContainerSpec: &cachekey.ContainerSpec{
					CmdArgs: []string{"sh", "ec", "run"},
					Image:   "python:3.9",
				},
			},
			wantEqual:   false,
			fingerPrint: "0a4cc1f15cdfad5170e1358518f7128c5278500a670db1b9a3f3d83b93db396e",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fingerPrint, err := GenerateFingerPrint(cacheKey)
			assert.Nil(t, err)
			testFingerPrint, err := GenerateFingerPrint(test.cacheKey)
			assert.Nil(t, err)
			assert.Equal(t, fingerPrint == testFingerPrint, test.wantEqual)
			assert.Equal(t, test.fingerPrint, testFingerPrint)
		})
	}
}
