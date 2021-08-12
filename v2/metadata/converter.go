package metadata

import (
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func mlmdValueToPipelineSpecValue(v *pb.Value) (*pipelinespec.Value, error) {
	switch t := v.Value.(type) {
	case *pb.Value_StringValue:
		return StringValue(t.StringValue), nil
	case *pb.Value_DoubleValue:
		return DoubleValue(t.DoubleValue), nil
	case *pb.Value_IntValue:
		return IntValue(t.IntValue), nil
	default:
		return nil, fmt.Errorf("unknown value type %T", t)
	}
}

func StringValue(v string) *pipelinespec.Value {
	return &pipelinespec.Value{
		Value: &pipelinespec.Value_StringValue{StringValue: v},
	}
}

func DoubleValue(v float64) *pipelinespec.Value {
	return &pipelinespec.Value{
		Value: &pipelinespec.Value_DoubleValue{DoubleValue: v},
	}
}

func IntValue(v int64) *pipelinespec.Value {
	return &pipelinespec.Value{
		Value: &pipelinespec.Value_IntValue{IntValue: v},
	}
}

func pipelineSpecValueToMLMDValue(v *pipelinespec.Value) (*pb.Value, error) {
	switch t := v.Value.(type) {
	case *pipelinespec.Value_StringValue:
		return stringMLMDValue(v.GetStringValue()), nil
	case *pipelinespec.Value_DoubleValue:
		return doubleMLMDValue(v.GetDoubleValue()), nil
	case *pipelinespec.Value_IntValue:
		return intMLMDValue(v.GetIntValue()), nil
	default:
		return nil, fmt.Errorf("unknown value type %T", t)
	}
}

func stringMLMDValue(v string) *pb.Value {
	return &pb.Value{Value: &pb.Value_StringValue{StringValue: v}}
}

func doubleMLMDValue(v float64) *pb.Value {
	return &pb.Value{Value: &pb.Value_DoubleValue{DoubleValue: v}}
}

func intMLMDValue(v int64) *pb.Value {
	return &pb.Value{Value: &pb.Value_IntValue{IntValue: v}}
}

func toMLMDArtifact(runtimeArtifact *pipelinespec.RuntimeArtifact) (*pb.Artifact, error) {
	errorF := func(err error) error {
		return fmt.Errorf("failed to convert RuntimeArtifact to MLMD artifact: %w", err)
	}
	artifact := &pb.Artifact{
		Uri:              &runtimeArtifact.Uri,
		Properties:       make(map[string]*pb.Value),
		CustomProperties: make(map[string]*pb.Value),
	}

	if runtimeArtifact.Metadata != nil {
		for k, v := range runtimeArtifact.Metadata.Fields {
			value, err := structValueToMLMDValue(v)
			if err != nil {
				return nil, errorF(err)
			}
			artifact.CustomProperties[k] = value
		}
	}

	return artifact, nil
}

func structValueToMLMDValue(v *structpb.Value) (*pb.Value, error) {
	boolToInt := func(b bool) int64 {
		if b {
			return 1
		}
		return 0
	}

	switch t := v.Kind.(type) {
	case *structpb.Value_StringValue:
		return stringMLMDValue(v.GetStringValue()), nil
	case *structpb.Value_NumberValue:
		return doubleMLMDValue(v.GetNumberValue()), nil
	case *structpb.Value_BoolValue:
		return intMLMDValue(boolToInt(v.GetBoolValue())), nil
	case *structpb.Value_ListValue:
		return &pb.Value{
			Value: &pb.Value_StructValue{
				StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"list": {Kind: &structpb.Value_ListValue{ListValue: v.GetListValue()}}}}},
		}, nil
	case *structpb.Value_StructValue:
		return &pb.Value{
			Value: &pb.Value_StructValue{
				StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"struct": {Kind: &structpb.Value_StructValue{StructValue: v.GetStructValue()}}}}},
		}, nil
	// TODO: support null
	default:
		return nil, fmt.Errorf("unknown/unsupported value type %T", t)
	}
}

func UnmarshalRuntimeArtifact(bytes []byte) (*pipelinespec.RuntimeArtifact, error) {
	a := &pb.Artifact{}
	if err := protojson.Unmarshal(bytes, a); err != nil {
		return nil, fmt.Errorf("failed to unmarshall runtime artifact metadata: %w", err)
	}
	return toRuntimeArtifact(a)
}

func toRuntimeArtifact(artifact *pb.Artifact) (*pipelinespec.RuntimeArtifact, error) {
	errorF := func(err error) (*pipelinespec.RuntimeArtifact, error) {
		return nil, fmt.Errorf("failed to convert MLMD artifact to RuntimeArtifact: %w", err)
	}

	rta := &pipelinespec.RuntimeArtifact{
		Name: strconv.FormatInt(artifact.GetId(), 10),
		Uri:  artifact.GetUri(),
		Metadata: &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		},
	}

	propertiesToMetadata := func(properties map[string]*pb.Value) error {
		for k, p := range properties {
			value := &structpb.Value{}
			switch t := p.Value.(type) {
			case *pb.Value_StringValue:
				value.Kind = &structpb.Value_StringValue{StringValue: p.GetStringValue()}
			case *pb.Value_DoubleValue:
				value.Kind = &structpb.Value_NumberValue{NumberValue: p.GetDoubleValue()}
			case *pb.Value_IntValue:
				value.Kind = &structpb.Value_NumberValue{NumberValue: float64(p.GetIntValue())}
			case *pb.Value_StructValue:
				value.Kind = &structpb.Value_StructValue{StructValue: p.GetStructValue()}
			default:
				return fmt.Errorf("unknown property type in MLMD artifact: %T", t)
			}
			rta.Metadata.Fields[k] = value
		}
		return nil
	}
	if err := propertiesToMetadata(artifact.Properties); err != nil {
		return errorF(err)
	}
	if err := propertiesToMetadata(artifact.CustomProperties); err != nil {
		return errorF(err)
	}

	return rta, nil
}
