package metadata

import (
	"encoding/json"
	"fmt"
	"ml_metadata/metadata_store/mlmetadata"
	mlpb "ml_metadata/proto/metadata_store_go_proto"
	"testing"

	argo "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
)

func (as artifactStructs) String() string {
	var s string
	for _, a := range as {
		s += fmt.Sprintf("%+v\n", a)
	}
	return s
}

func TestParseValidTFXMetadata(t *testing.T) {
	tests := []struct {
		input string
		want  artifactStructs
	}{
		{
			`[{
				"artifact_type": {
					"name": "Artifact",
					"properties": {"state": "STRING", "span": "INT" } },
			 	"artifact": {
					"uri": "/location",
					 "properties": {
							 "state": {"stringValue": "complete"},
							 "span": {"intValue": 10} }
					}
				}]`,
			[]*artifactStruct{
				&artifactStruct{
					ArtifactType: &mlpb.ArtifactType{
						Name: proto.String("Artifact"),
						Properties: map[string]mlpb.PropertyType{
							"state": mlpb.PropertyType_STRING,
							"span":  mlpb.PropertyType_INT,
						},
					},
					Artifact: &mlpb.Artifact{
						Uri: proto.String("/location"),
						Properties: map[string]*mlpb.Value{
							"state": &mlpb.Value{Value: &mlpb.Value_StringValue{"complete"}},
							"span":  &mlpb.Value{Value: &mlpb.Value_IntValue{10}},
						},
					},
				},
			},
		},
		{
			`[{
				"artifact_type": {
					"name": "Artifact 1",
					"properties": {"state": "STRING", "span": "INT" } },
			 	"artifact": {
					"uri": "/location 1",
					 "properties": {
							 "state": {"stringValue": "complete"},
							 "span": {"intValue": 10} }
					}
				},
				{
					"artifact_type": {
						"name": "Artifact 2",
						"properties": {"state": "STRING", "span": "INT" } },
			 	"artifact": {
					"uri": "/location 2",
					 "properties": {
							 "state": {"stringValue": "complete"},
							 "span": {"intValue": 10} }
					}
				}]`,
			[]*artifactStruct{
				&artifactStruct{
					ArtifactType: &mlpb.ArtifactType{
						Name: proto.String("Artifact 1"),
						Properties: map[string]mlpb.PropertyType{
							"state": mlpb.PropertyType_STRING,
							"span":  mlpb.PropertyType_INT,
						},
					},
					Artifact: &mlpb.Artifact{
						Uri: proto.String("/location 1"),
						Properties: map[string]*mlpb.Value{
							"state": &mlpb.Value{Value: &mlpb.Value_StringValue{"complete"}},
							"span":  &mlpb.Value{Value: &mlpb.Value_IntValue{10}},
						},
					},
				},
				&artifactStruct{
					ArtifactType: &mlpb.ArtifactType{
						Name: proto.String("Artifact 2"),
						Properties: map[string]mlpb.PropertyType{
							"state": mlpb.PropertyType_STRING,
							"span":  mlpb.PropertyType_INT,
						},
					},
					Artifact: &mlpb.Artifact{
						Uri: proto.String("/location 2"),
						Properties: map[string]*mlpb.Value{
							"state": &mlpb.Value{Value: &mlpb.Value_StringValue{"complete"}},
							"span":  &mlpb.Value{Value: &mlpb.Value_IntValue{10}},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		got, err := parseTFXMetadata(test.input)
		if err != nil || !cmp.Equal(got, test.want) {
			t.Errorf("parseTFXMetadata(%q)\nGot:\n<%+v, %+v>\nWant:\n%+v, nil error\nDiff:\n%s", test.input, got, err, test.want, cmp.Diff(test.want, got))
		}
	}
}

func TestParseInvalidTFXMetadata(t *testing.T) {
	tests := []struct {
		desc  string
		input string
	}{
		{
			"no artifact type",
			`[{
			 	"artifact": {
					"uri": "/location",
					 "properties": {
							 "state": {"stringValue": "complete"},
							 "span": {"intValue": 10} }
					}
				}]`,
		},
		{
			"no artifact",
			`[{
				"artifact_type": {
					"name": "Artifact",
					"properties": {"state": "STRING", "span": "INT" } },
				}]`,
		},
		{
			"empty string",
			"",
		},
	}

	for _, test := range tests {
		_, err := parseTFXMetadata(test.input)
		if err == nil {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("parseTFXMetadata(%q)\nGot non-nil error. Want error.", test.input)
		}
	}
}

func fakeMLMDStore(t *testing.T) *mlmetadata.Store {
	cfg := &mlpb.ConnectionConfig{
		Config: &mlpb.ConnectionConfig_FakeDatabase{&mlpb.FakeDatabaseConfig{}},
	}

	mlmdStore, err := mlmetadata.NewStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create ML Metadata Store: %v", err)
	}
	return mlmdStore
}

func TestRecordOutputArtifacts(t *testing.T) {

	tests := []struct {
		desc          string
		stored        *argo.Workflow
		current       *argo.Workflow
		wantTypes     []*mlpb.ArtifactType
		wantArtifacts []*mlpb.Artifact
	}{
		{
			desc: "nothing is recorded when node is still running",
			stored: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeRunning,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									ValueFrom: &argo.ValueFrom{Path: "/output/ml_metadata/output1"},
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/location 2",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			current: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeRunning,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									ValueFrom: &argo.ValueFrom{Path: "/output/ml_metadata/output1"},
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/step_1_location",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			wantTypes:     nil,
			wantArtifacts: nil,
		},
		{
			desc: "artifacts are recorded when node transitions to Complete",
			stored: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeRunning,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									ValueFrom: &argo.ValueFrom{Path: "/output/ml_metadata/output1"},
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/location 2",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			current: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeSucceeded,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									ValueFrom: &argo.ValueFrom{Path: "/output/ml_metadata/output1"},
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/step_1_location",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			wantTypes: []*mlpb.ArtifactType{
				&mlpb.ArtifactType{
					Id:         proto.Int64(1),
					Name:       proto.String("step_1_artifact_type"),
					Properties: map[string]mlpb.PropertyType{"state": mlpb.PropertyType_STRING},
				}},
			wantArtifacts: []*mlpb.Artifact{
				&mlpb.Artifact{
					Id:     proto.Int64(1),
					TypeId: proto.Int64(1),
					Uri:    proto.String("/step_1_location"),
					Properties: map[string]*mlpb.Value{
						"state": &mlpb.Value{Value: &mlpb.Value_StringValue{"complete"}}},
				}},
		},
		{
			desc: "Records artifacts only from Node with output parameter specified in ValueFrom Path",
			stored: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeRunning,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/location 2",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			current: &argo.Workflow{Status: argo.WorkflowStatus{
				Nodes: map[string]argo.NodeStatus{
					"step_1": argo.NodeStatus{
						ID:    "step_1_node_id",
						Phase: argo.NodeSucceeded,
						Outputs: &argo.Outputs{
							Parameters: []argo.Parameter{
								argo.Parameter{
									Value: proto.String(`[{
										"artifact_type": { "name": "step_1_artifact_type",
																			 "properties": {"state": "STRING" } },
										"artifact": { "uri": "/step_1_location",
										  						"properties": {
																		 "state": {"stringValue": "complete"} } }
										}]`),
								}}}}}}},
			wantTypes:     nil,
			wantArtifacts: nil,
		},
	}

	for _, test := range tests {
		mlmdStore := fakeMLMDStore(t)
		store := Store{mlmdStore: mlmdStore}

		current, err := json.Marshal(test.current)
		if err != nil {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("json.Marshal(%v) = %v", test.current, err)
			continue
		}

		stored, err := json.Marshal(test.stored)
		if err != nil {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("json.Marshal(%v) = %v", test.stored, err)
			continue
		}

		if err := store.RecordOutputArtifacts("", string(stored), string(current)); err != nil {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("store.RecordOutputArtifacts(%q, %q) = %v\nWant non-nil error\n", current, stored, err)
			continue
		}

		gotTypes, err := mlmdStore.GetArtifactTypesByID([]mlmetadata.ArtifactTypeID{1})
		if !cmp.Equal(gotTypes, test.wantTypes) {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("mlmdStore.GetArtifactTypes()\n")
			t.Errorf("Got artifact types:\n%v\nWant\n%v\n", gotTypes, test.wantTypes)
			t.Errorf("Got error:\n%v\nWant nil error\n", err)
		}

		gotArtifacts, err := mlmdStore.GetArtifacts()
		if !cmp.Equal(gotArtifacts, test.wantArtifacts) {
			t.Errorf("Test: %q", test.desc)
			t.Errorf("mlmdStore.GetArtifacts()\n")
			t.Errorf("Got artifacts:\n%v\nWant\n%v\n", gotArtifacts, test.wantArtifacts)
			t.Errorf("Got error:\n%v\nWant nil error\n", err)
		}
	}

}
