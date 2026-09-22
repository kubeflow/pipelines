// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"strings"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

// Converts pipeline spec in the form of structpb.Struct to a yaml string.
// Handles three possible cases:
// 1) pipeline spec passed directly
// 2) pipeline spec along with platform specific configs, each as a sub struct
// 3) pipeline spec as a sub struct, with platform specific field empty
// Note: this method does not verify the validity of pipeline spec.
func pipelineSpecStructToYamlString(s *structpb.Struct) (string, error) {
	var bytes []byte
	var err error
	if spec, ok := s.GetFields()["pipeline_spec"]; ok {
		bytes, err = yaml.Marshal(spec)
		if err != nil {
			return "", util.Wrap(err, "Failed to convert pipeline protobuf struct to a yaml string")
		}
		if platforms, ok := s.GetFields()["platform_spec"]; ok {
			bytesPlatforms, err := yaml.Marshal(platforms)
			if err != nil {
				return "", util.Wrap(err, "Failed to convert platforms protobuf struct to a yaml string")
			}
			bytes = append(bytes, []byte("\n---\n")...)
			bytes = append(bytes, bytesPlatforms...)
		}
	} else {
		bytes, err = yaml.Marshal(s)
		if err != nil {
			return "", util.Wrap(err, "Failed to convert pipeline spec protobuf struct to a yaml string")
		}
	}
	return string(bytes), nil
}

// Converts pipeline spec in yaml string into structpb.Struct (v2).
// The string may contain multiple yaml documents, where there has to be one doc for pipeline spec,
// and there may be a second doc for platform specific spec.
// If platform spec is empty, then return pipeline spec directly. Else, return a struct with
// pipeline spec and platform spec as subfields.
func YamlStringToPipelineSpecStruct(s string) (*structpb.Struct, error) {
	if s == "" {
		return nil, nil
	}
	var pipelineSpec structpb.Struct
	var platformSpec structpb.Struct
	hasPipelineSpec := false
	hasPlatformSpec := false

	yamlStrings := strings.Split(s, "\n---\n")
	for _, yamlString := range yamlStrings {
		if template.IsPlatformSpecWithKubernetesConfig([]byte(yamlString)) {
			hasPlatformSpec = true
			jsonBytes, err := yaml.YAMLToJSON([]byte(yamlString))
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert platformSpec yaml string into a protobuf struct")
			}
			err = protojson.Unmarshal(jsonBytes, &platformSpec)
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert platformSpec yaml string into a protobuf struct")
			}
		} else {
			hasPipelineSpec = true
			jsonBytes, err := yaml.YAMLToJSON([]byte(yamlString))
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert pipelineSpec yaml string into a protobuf struct")
			}
			err = protojson.Unmarshal(jsonBytes, &pipelineSpec)
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert pipelineSpec yaml string into a protobuf struct")
			}
		}
	}
	if !hasPipelineSpec {
		return nil, util.NewInvalidInputError("No pipeline spec provided in yaml doc")
	} else if !hasPlatformSpec {
		return &pipelineSpec, nil
	} else {
		pipeline := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"pipeline_spec": structpb.NewStructValue(&pipelineSpec),
				"platform_spec": structpb.NewStructValue(&platformSpec),
			},
		}
		return pipeline, nil
	}
}

// Fetches a PipelineRoot from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetPipelineRootFromRunInterface(r interface{}) (string, error) {
	switch r := r.(type) {

	case *apiv2beta1.Run:
		return r.GetRuntimeConfig().GetPipelineRoot(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetPipelineRootFromRunInterface()", r)
	}
}

// Fetches a RuntimeConfig from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetRuntimeConfigFromRunInterface(r interface{}) (map[string]interface{}, error) {
	switch r := r.(type) {

	case *apiv2beta1.Run:
		newRuntimeConfig := map[string]interface{}{
			"Parameters":   r.GetRuntimeConfig().GetParameters(),
			"PipelineRoot": r.GetRuntimeConfig().GetPipelineRoot(),
		}
		return newRuntimeConfig, nil
	default:
		return nil, util.NewUnknownApiVersionError("GetRuntimeConfigFromRunInterface()", r)
	}
}

// Fetches a RunDetails from a Run.
// This is not intended for validation.
// Raises error if an incompatible interface is used.
func GetRunDetailsFromRunInterface(r interface{}) (string, error) {
	switch r := r.(type) {

	case *apiv2beta1.Run:
		return r.GetRunDetails().String(), nil
	default:
		return "", util.NewUnknownApiVersionError("GetRunDetailsFromRunInterface()", r)
	}
}
