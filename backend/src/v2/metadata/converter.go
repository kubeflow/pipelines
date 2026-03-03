package metadata

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func PbValueToText(v *structpb.Value) (string, error) {
	wrap := func(err error) error {
		return fmt.Errorf("failed to convert protobuf.Value to text: %w", err)
	}
	if v == nil {
		return "", nil
	}
	var text string
	switch t := v.Kind.(type) {
	case *structpb.Value_NullValue:
		text = ""
	case *structpb.Value_StringValue:
		text = v.GetStringValue()
	case *structpb.Value_NumberValue:
		text = strconv.FormatFloat(v.GetNumberValue(), 'f', -1, 64)
	case *structpb.Value_BoolValue:
		text = strconv.FormatBool(v.GetBoolValue())
	case *structpb.Value_ListValue:
		b, err := json.Marshal(v.GetListValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a list: %w", err))
		}
		text = string(b)
	case *structpb.Value_StructValue:
		b, err := json.Marshal(v.GetStructValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a struct: %w", err))
		}
		text = string(b)
	default:
		return "", wrap(fmt.Errorf("unknown type %T", t))
	}
	return text, nil
}

func TextToPbValue(text string, t pipelinespec.ParameterType_ParameterTypeEnum) (*structpb.Value, error) {
	msg := func(err error) error {
		return fmt.Errorf("TextToPbValue(text=%q, type=%q) failed: %w", text, t, err)
	}
	switch t {
	case pipelinespec.ParameterType_STRING:
		return structpb.NewStringValue(text), nil
	case pipelinespec.ParameterType_NUMBER_INTEGER:
		i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 0)
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewNumberValue(float64(i)), nil
	case pipelinespec.ParameterType_NUMBER_DOUBLE:
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 0)
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewNumberValue(f), nil
	case pipelinespec.ParameterType_BOOLEAN:
		v, err := strconv.ParseBool(strings.TrimSpace(text))
		if err != nil {
			return nil, msg(err)
		}
		return structpb.NewBoolValue(v), nil
	case pipelinespec.ParameterType_LIST:
		v := &structpb.Value{}
		if err := v.UnmarshalJSON([]byte(text)); err != nil {
			return nil, msg(err)
		}
		if _, ok := v.GetKind().(*structpb.Value_ListValue); !ok {
			return nil, msg(fmt.Errorf("unexpected type"))
		}
		return v, nil
	case pipelinespec.ParameterType_STRUCT:
		v := &structpb.Value{}
		if err := v.UnmarshalJSON([]byte(text)); err != nil {
			return nil, msg(err)
		}
		if _, ok := v.GetKind().(*structpb.Value_StructValue); !ok {
			return nil, msg(fmt.Errorf("unexpected type"))
		}
		return v, nil
	default:
		return nil, msg(fmt.Errorf("unknown type. Expected STRING, NUMBER_INTEGER, NUMBER_DOUBLE, BOOLEAN, LIST or STRUCT"))
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
			// _kfp_workspace flag is needed only at runtime for the executor to know
			//  the correct path to the artifact; do not persist to MLMD
			if k == "_kfp_workspace" {
				continue
			}
			value, err := StructValueToMLMDValue(v)
			if err != nil {
				return nil, errorF(err)
			}
			artifact.CustomProperties[k] = value
		}
	}

	return artifact, nil
}

func StructValueToMLMDValue(v *structpb.Value) (*pb.Value, error) {
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
