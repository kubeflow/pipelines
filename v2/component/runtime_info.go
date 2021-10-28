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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type inputParameter struct {
	// Type should be one of:
	// - STRING
	// - NUMBER_INTEGER
	// - NUMBER_DOUBLE
	// - BOOLEAN
	// - LIST
	// - STRUCT
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
	// Type should be one of:
	// - STRING
	// - NUMBER_INTEGER
	// - NUMBER_DOUBLE
	// - BOOLEAN
	// - LIST
	// - STRUCT
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

func parseRuntimeInfo(jsonEncoded string) (*runtimeInfo, error) {
	r := &runtimeInfo{
		InputParameters:  make(map[string]*inputParameter),
		InputArtifacts:   make(map[string]*inputArtifact),
		OutputParameters: make(map[string]*outputParameter),
		OutputArtifacts:  make(map[string]*outputArtifact),
	}

	if err := json.Unmarshal([]byte(jsonEncoded), r); err != nil {
		// Do not quote jsonEncoded, because JSON format is hard to read if quoted.
		return nil, fmt.Errorf("Invalid runtime info: %w.\n===RuntimeInfo===\n%s\n======", err, jsonEncoded)
	}

	return r, nil
}

// Parse launcher arguments with the following sections:
// 1. parameters in format "key1=value1", "key2=value2", ...
// 2. a separator "--" as end of parameters passed to launcher
// 3. arguments of the original user program command + args
//
// Returns:
// * parameters will be recorded in runtimeInfo
// * (command + args) is the return value
func parseArgs(args []string, rt *runtimeInfo) ([]string, error) {
	argsError := func(err error) error {
		return fmt.Errorf("error parsing input parameters from args %v: %w", args, err)
	}
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
		// parse input parameter argument like key=value
		segs := strings.SplitN(arg, "=", 2)
		if len(segs) != 2 {
			return nil, argsError(fmt.Errorf("invalid arg, expecting format like key=value, got %q", arg))
		}
		name := segs[0]
		value := segs[1]
		param, ok := rt.InputParameters[name]
		if !ok {
			return nil, argsError(fmt.Errorf("unexpected input parameter %q, not found in spec %+v", name, rt.InputParameters))
		}
		param.Value = value
	}
	if separator == -1 {
		return nil, argsError(fmt.Errorf("cannot find separator \"--\""))
	}
	return args[separator+1:], nil
}

func setRuntimeArtifactType(rta *pipelinespec.RuntimeArtifact, instanceSchema, schemaTitle string) error {
	if len(instanceSchema) != 0 && len(schemaTitle) != 0 {
		return fmt.Errorf("only one of instanceSchema or schemaTitle should be specified. Got schemaTitle = %q, instanceSchema = %q", schemaTitle, instanceSchema)
	}

	rta.Type = &pipelinespec.ArtifactTypeSchema{}

	if len(instanceSchema) != 0 {
		rta.Type.Kind = &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: instanceSchema}
	}
	if len(schemaTitle) != 0 {
		rta.Type.Kind = &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: schemaTitle}
	}
	return nil
}

func readArtifact(filePath string) (*pipelinespec.RuntimeArtifact, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact metadata file %q: %w", filePath, err)
	}
	return metadata.UnmarshalRuntimeArtifact(b)
}

type generateOutputURI func(outputName string) string

func (r *runtimeInfo) generateExecutorInput(genOutputURI generateOutputURI, outputMetadataFilepath string) (*pipelinespec.ExecutorInput, error) {

	inputs := &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}

	outputs := &pipelinespec.ExecutorInput_Outputs{
		Parameters: make(map[string]*pipelinespec.ExecutorInput_OutputParameter),
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
		OutputFile: outputMetadataFilepath,
	}

	for name, ip := range r.InputParameters {
		var value *structpb.Value
		switch ip.Type {
		case "STRING":
			value = structpb.NewStringValue(ip.Value)
		case "NUMBER_INTEGER", "NUMBER_DOUBLE":
			f, err := strconv.ParseFloat(ip.Value, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to parse number parameter %q from '%v': %w", name, ip.Value, err)
			}
			value = structpb.NewNumberValue(f)
		case "BOOLEAN":
			b, err := strconv.ParseBool(ip.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse boolean parameter %q from '%v': %w", name, ip.Value, err)
			}
			value = structpb.NewBoolValue(b)
		case "LIST":
			value = &structpb.Value{}
			if err := value.UnmarshalJSON([]byte(ip.Value)); err != nil {
				return nil, fmt.Errorf("failed to parse list parameter %q from '%v': %w", name, ip.Value, err)

			}
		case "STRUCT":
			value = &structpb.Value{}
			if err := value.UnmarshalJSON([]byte(ip.Value)); err != nil {
				return nil, fmt.Errorf("failed to parse struct parameter %q from '%v': %w", name, ip.Value, err)

			}
		default:
			return nil, fmt.Errorf("unknown ParameterType for parameter %q: %q", name, ip.Type)
		}
		inputs.ParameterValues[name] = value
	}

	for name, ia := range r.InputArtifacts {
		if len(ia.MetadataPath) == 0 {
			return nil, fmt.Errorf("missing input artifact metadata file for input %q", name)
		}
		if !filepath.IsAbs(ia.MetadataPath) {
			return nil, fmt.Errorf("unexpected input artifact metadata file %q for input %q: must be absolute local path", ia.MetadataPath, name)
		}

		rta, err := readArtifact(ia.MetadataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read Artifact %q from file %s: %w", name, ia.MetadataPath, err)
		}
		if err := setRuntimeArtifactType(rta, ia.InstanceSchema, ia.SchemaTitle); err != nil {
			return nil, fmt.Errorf("failed to set type to RuntimeArtifact %q: %w", name, err)
		}
		inputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{rta},
		}
	}

	for name, op := range r.OutputParameters {
		outputParameter := &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: op.Path,
		}
		outputs.Parameters[name] = outputParameter
	}

	for name, oa := range r.OutputArtifacts {
		uri := genOutputURI(name)
		rta := &pipelinespec.RuntimeArtifact{
			Name: name,
			Uri:  uri,
			Metadata: &structpb.Struct{
				Fields: make(map[string]*structpb.Value)},
		}
		if strings.HasPrefix(uri, "s3://") {
			s3Region := os.Getenv("AWS_REGION")
			rta.Metadata.Fields["s3_region"] = stringToStructValue(s3Region)
		}

		if err := setRuntimeArtifactType(rta, oa.InstanceSchema, oa.SchemaTitle); err != nil {
			return nil, fmt.Errorf("failed to generate output RuntimeArtifact: %w", err)
		}
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{rta},
		}
	}

	return &pipelinespec.ExecutorInput{
		Inputs:  inputs,
		Outputs: outputs,
	}, nil
}

func stringToStructValue(v string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
}
