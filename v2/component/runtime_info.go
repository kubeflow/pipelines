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
	"os"
	"regexp"
	"strconv"
	"strings"

	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"github.com/kubeflow/pipelines/v2/third_party/pipeline_spec"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type inputParameter struct {
	// Type should be one of "INT", "STRING" or "DOUBLE".
	Type string
	// File used to read input parameters.
	Value string
}

type inputArtifact struct {
	// Schema description of the input artifact.
	InstanceSchema string
	SchemaTitle    string

	// Where to read the input MLMD artifact metadata. This file is passed using
	// Argo artifacts.
	MetadataPath string
}

type outputParameter struct {
	// Type should be one of "INT", "STRING" or "DOUBLE".
	Type string
	// File used to write output parameters to.
	Path string
}

type outputArtifact struct {
	// Schema description of the output artifact.
	InstanceSchema string
	SchemaTitle    string

	// Where to write the output MLMD artifact metadata. This file is passed using
	// Argo artifacts.
	MetadataPath string
}

// runtimeInfo represents JSON object present in all ML components compiled
// under the V2-compatible flag in KFP.
type runtimeInfo struct {
	InputParameters  map[string]*inputParameter
	InputArtifacts   map[string]*inputArtifact
	OutputParameters map[string]*outputParameter
	OutputArtifacts  map[string]*outputArtifact
}

// paramDelimiterRE is used to find all instances of parameter string that need
// to be escaped. We use .+? to do a non-greedy match here.
var paramDelimiterRE = regexp.MustCompile(`"BEGIN-KFP-PARAM[[].+?[]]END-KFP-PARAM"`)

func escapeParameters(unescaped string) (string, error) {
	var jsonEncodeErr error
	escapeFunc := func(value string) string {
		value = strings.TrimPrefix(value, `"BEGIN-KFP-PARAM[`)
		value = strings.TrimSuffix(value, `]END-KFP-PARAM"`)
		b, err := json.Marshal(value)
		if err != nil {
			jsonEncodeErr = err
		}
		return string(b)
	}

	escaped := paramDelimiterRE.ReplaceAllStringFunc(unescaped, escapeFunc)

	if jsonEncodeErr != nil {
		return "", fmt.Errorf("failed to JSON-encode parameter %q: %w", unescaped, jsonEncodeErr)
	}

	return escaped, nil
}

func parseRuntimeInfo(jsonEncoded string) (*runtimeInfo, error) {
	r := &runtimeInfo{
		InputParameters:  make(map[string]*inputParameter),
		InputArtifacts:   make(map[string]*inputArtifact),
		OutputParameters: make(map[string]*outputParameter),
		OutputArtifacts:  make(map[string]*outputArtifact),
	}

	escaped, err := escapeParameters(jsonEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to escape parameters from RuntimeInfo: %w", err)
	}

	if err := json.Unmarshal([]byte(escaped), r); err != nil {
		// Do not quote jsonEncoded, because JSON format is hard to read if quoted.
		return nil, fmt.Errorf("Invalid runtime info: %w.\n===RuntimeInfo===\n%s\n======", err, escaped)
	}

	return r, nil
}

func pipelineSpecValueToMLMDValue(v *pipeline_spec.Value) (*pb.Value, error) {
	switch t := v.Value.(type) {
	case *pipeline_spec.Value_StringValue:
		return &pb.Value{Value: &pb.Value_StringValue{StringValue: v.GetStringValue()}}, nil
	case *pipeline_spec.Value_DoubleValue:
		return &pb.Value{Value: &pb.Value_DoubleValue{DoubleValue: v.GetDoubleValue()}}, nil
	case *pipeline_spec.Value_IntValue:
		return &pb.Value{Value: &pb.Value_IntValue{IntValue: v.GetIntValue()}}, nil
	default:
		return nil, fmt.Errorf("unknown value type %T", t)
	}
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
		return &pb.Value{Value: &pb.Value_StringValue{StringValue: v.GetStringValue()}}, nil
	case *structpb.Value_NumberValue:
		return &pb.Value{Value: &pb.Value_DoubleValue{DoubleValue: v.GetNumberValue()}}, nil
	case *structpb.Value_BoolValue:
		return &pb.Value{Value: &pb.Value_IntValue{IntValue: boolToInt(v.GetBoolValue())}}, nil
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

func toMLMDArtifact(runtimeArtifact *pipeline_spec.RuntimeArtifact) (*pb.Artifact, error) {
	errorF := func(err error) error {
		return fmt.Errorf("failed to convert RuntimeArtifact to MLMD artifact: %w", err)
	}
	artifact := &pb.Artifact{
		Uri:              &runtimeArtifact.Uri,
		Properties:       make(map[string]*pb.Value),
		CustomProperties: make(map[string]*pb.Value),
	}

	for k, v := range runtimeArtifact.Properties {
		value, err := pipelineSpecValueToMLMDValue(v)
		if err != nil {
			return nil, errorF(err)
		}
		artifact.Properties[k] = value
	}

	for k, v := range runtimeArtifact.CustomProperties {
		value, err := pipelineSpecValueToMLMDValue(v)
		if err != nil {
			return nil, errorF(err)
		}
		artifact.CustomProperties[k] = value
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

func setRuntimeArtifactType(rta *pipeline_spec.RuntimeArtifact, instanceSchema, schemaTitle string) error {
	if len(instanceSchema) != 0 && len(schemaTitle) != 0 {
		return fmt.Errorf("only one of instanceSchema or schemaTitle should be specified. Got schemaTitle = %q, instanceSchema = %q", schemaTitle, instanceSchema)
	}

	rta.Type = &pipeline_spec.ArtifactTypeSchema{}

	if len(instanceSchema) != 0 {
		rta.Type.Kind = &pipeline_spec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: instanceSchema}
	}
	if len(schemaTitle) != 0 {
		rta.Type.Kind = &pipeline_spec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: schemaTitle}
	}
	return nil
}

func toRuntimeArtifact(artifact *pb.Artifact, instanceSchema, schemaTitle string) (*pipeline_spec.RuntimeArtifact, error) {
	errorF := func(err error) (*pipeline_spec.RuntimeArtifact, error) {
		return nil, fmt.Errorf("failed to convert MLMD artifact to RuntimeArtifact: %w", err)
	}

	rta := &pipeline_spec.RuntimeArtifact{
		Name: strconv.FormatInt(artifact.GetId(), 10),
		Uri:  artifact.GetUri(),
		Metadata: &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		},
	}

	if err := setRuntimeArtifactType(rta, instanceSchema, schemaTitle); err != nil {
		return errorF(err)
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

func readArtifact(filePath string) (*pb.Artifact, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact metadata file %q: %w", filePath, err)
	}

	a := &pb.Artifact{}
	if err := protojson.Unmarshal(b, a); err != nil {
		return nil, fmt.Errorf("failed to unmarshall artifact metadata in file %q: %w", filePath, err)
	}
	return a, nil
}

type generateOutputURI func(outputName string) string

func (r *runtimeInfo) generateExecutorInput(genOutputURI generateOutputURI, outputMetadataFilepath string) (*pipeline_spec.ExecutorInput, error) {

	inputs := &pipeline_spec.ExecutorInput_Inputs{
		Parameters: make(map[string]*pipeline_spec.Value),
		Artifacts:  make(map[string]*pipeline_spec.ArtifactList),
	}

	outputs := &pipeline_spec.ExecutorInput_Outputs{
		Parameters: make(map[string]*pipeline_spec.ExecutorInput_OutputParameter),
		Artifacts:  make(map[string]*pipeline_spec.ArtifactList),
		OutputFile: outputMetadataFilepath,
	}

	for name, ip := range r.InputParameters {
		value := &pipeline_spec.Value{}
		switch ip.Type {
		case "STRING":
			value.Value = &pipeline_spec.Value_StringValue{StringValue: ip.Value}
		case "INT":
			i, err := strconv.ParseInt(ip.Value, 10, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int parameter %q from '%v': %w", name, i, err)
			}
			value.Value = &pipeline_spec.Value_IntValue{IntValue: i}
		case "DOUBLE":
			f, err := strconv.ParseFloat(ip.Value, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to parse double parameter %q from '%v': %w", name, f, err)
			}
			value.Value = &pipeline_spec.Value_DoubleValue{DoubleValue: f}
		default:
			return nil, fmt.Errorf("unknown ParameterType for parameter %q: %q", name, ip.Type)
		}
		inputs.Parameters[name] = value
	}

	for name, ia := range r.InputArtifacts {
		if len(ia.MetadataPath) == 0 {
			return nil, fmt.Errorf("missing input artifact metadata file for input %q", name)
		}

		artifact, err := readArtifact(ia.MetadataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read Artifact %q from file %s: %w", name, ia.MetadataPath, err)
		}

		rta, err := toRuntimeArtifact(artifact, ia.InstanceSchema, ia.SchemaTitle)
		if err != nil {
			return nil, fmt.Errorf("failed to convert artifact %q to RuntimeArtifact: %w", name, err)
		}
		inputs.Artifacts[name] = &pipeline_spec.ArtifactList{
			Artifacts: []*pipeline_spec.RuntimeArtifact{rta},
		}
	}

	for name, op := range r.OutputParameters {
		outputParameter := &pipeline_spec.ExecutorInput_OutputParameter{
			OutputFile: op.Path,
		}
		outputs.Parameters[name] = outputParameter
	}

	for name, oa := range r.OutputArtifacts {
		uri := genOutputURI(name)
		rta := &pipeline_spec.RuntimeArtifact{
			Name: name,
			Uri:  uri,
			Metadata: &structpb.Struct{
				Fields: make(map[string]*structpb.Value)},
		}
		if strings.HasPrefix(uri, "s3://") {
			s3Region := os.Getenv("AWS_REGION")
			rta.Metadata.Fields["s3_region"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: s3Region}}
		}
		rta.Metadata.Fields["name"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: name}}

		if err := setRuntimeArtifactType(rta, oa.InstanceSchema, oa.SchemaTitle); err != nil {
			return nil, fmt.Errorf("failed to generate output RuntimeArtifact: %w", err)
		}
		outputs.Artifacts[name] = &pipeline_spec.ArtifactList{
			Artifacts: []*pipeline_spec.RuntimeArtifact{rta},
		}
	}

	return &pipeline_spec.ExecutorInput{
		Inputs:  inputs,
		Outputs: outputs,
	}, nil
}
