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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

func loadYaml(t *testing.T, path string) string {
	res, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	return string(res)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
func TestPipelineSpecStructToYamlString_DirectSpec(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipeline structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))
	err := protojson.Unmarshal(pipelineSpecJson, &pipeline)
	assert.Nil(t, err)

	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshaled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipeline.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
func TestPipelineSpecStructToYamlString_WithPlatform(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipelineSpec structpb.Struct
	var platformSpec structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))

	err := protojson.Unmarshal(pipelineSpecJson, &pipelineSpec)
	assert.Nil(t, err)

	platformSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[1]))
	err = protojson.Unmarshal(platformSpecJson, &platformSpec)
	assert.Nil(t, err)

	pipelineSpecValue := structpb.NewStructValue(&pipelineSpec)
	platformSpecValue := structpb.NewStructValue(&platformSpec)

	pipeline := structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pipeline_spec": pipelineSpecValue,
			"platform_spec": platformSpecValue,
		},
	}
	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshaled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipeline.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
// In this case although the received pipeline spec is nested, because platform spec is empty,
// we return the pipeline spec directly.
func TestPipelineSpecStructToYamlString_NestedPipelineSpec(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipelineSpec structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))
	err := protojson.Unmarshal(pipelineSpecJson, &pipelineSpec)
	assert.Nil(t, err)

	pipelineSpecValue := structpb.NewStructValue(&pipelineSpec)

	pipeline := structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pipeline_spec": pipelineSpecValue,
		},
	}
	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshaled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipelineSpec.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}
