package metadata

import (
	"fmt"
	mlpb "ml_metadata/proto/metadata_store_go_proto"
	"testing"

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
