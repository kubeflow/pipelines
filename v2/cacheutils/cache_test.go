package cacheutils

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/testing/protocmp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGenerateCacheKey(t *testing.T) {

	tests := []struct {
		name                    string
		executorInputInputs     *pipeline_spec.ExecutorInput_Inputs
		executorInputOutputs    *pipeline_spec.ExecutorInput_Outputs
		outputParametersTypeMap map[string]string
		cmdArgs                 []string
		image                   string
		want                    *CacheKey
		wantErr                 bool
	}{
		{
			name: "Generate CacheKey Correctly",
			executorInputInputs: &pipeline_spec.ExecutorInput_Inputs{
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
							}}}},
			},

			executorInputOutputs: &pipeline_spec.ExecutorInput_Outputs{
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
				OutputFile: "/tmp/kfp_outputs/output_metadata.json",
			},
			outputParametersTypeMap: map[string]string{
				"output_parameter_one": "STRING",
				"output_parameter_two": "INT",
			},
			cmdArgs: []string{"sh", "ec", "test"},
			image:   "python:3.9",
			want: &CacheKey{
				inputArtifactNames: map[string]artifactNameList{
					"dataset_one": {artifactNames: []string{"1"}},
					"dataset_two": {artifactNames: []string{"2"}},
				},
				inputParameters: map[string]pipeline_spec.Value{
					"message":   {Value: &pipeline_spec.Value_StringValue{StringValue: "Some string value"}},
					"num_steps": {Value: &pipeline_spec.Value_IntValue{IntValue: 5}},
				},
				outputArtifactsSpec: map[string]pipeline_spec.RuntimeArtifact{
					"model": {
						Name: "model",
						Type: &pipeline_spec.ArtifactTypeSchema{
							Kind: &pipeline_spec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: "title: kfp.Model\ntype: object\nproperties:\n  framework:\n    type: string\n  framework_version:\n    type: string\n"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "model"}}},
						}},
					"metrics": {
						Name: "metrics",
						Type: &pipeline_spec.ArtifactTypeSchema{
							Kind: &pipeline_spec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "kfp.Metrics"},
						},
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{"name": {Kind: &structpb.Value_StringValue{StringValue: "metrics"}}},
						}},
				},
				outputParametersSpec: map[string]string{
					"output_parameter_one": "STRING",
					"output_parameter_two": "INT",
				},
				containerSpec: containerSpec{
					cmdArgs: []string{"sh", "ec", "test"},
					image:   "python:3.9",
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

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty(), protocmp.Transform(), cmp.AllowUnexported(CacheKey{}, artifactNameList{}, containerSpec{})); diff != "" {
				t.Errorf("GenerateCacheKey() = %+v, want %+v\nDiff (-want, +got)\n%s", got, test.want, diff)
				s, _ := json.MarshalIndent(test.want, "", "  ")
				fmt.Printf("Want\n%s", s)
			}

		})
	}
}
