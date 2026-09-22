// Copyright 2025 The Kubeflow Authors
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

package util

import (
	"bytes"
	"fmt"
	"os"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/encoding/protojson"
	yaml3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

func LoadPipelineAndPlatformSpec(path string) (*pipelinespec.PipelineSpec, *pipelinespec.PlatformSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	dec := yaml3.NewDecoder(bytes.NewReader(data))
	um := protojson.UnmarshalOptions{}

	var first pipelinespec.PipelineSpec
	var second pipelinespec.PlatformSpec

	for i := 0; i < 2; i++ {
		var doc any
		if err := dec.Decode(&doc); err != nil {
			break // io.EOF means fewer than 2 docs
		}

		// Convert YAML -> JSON
		jsonBytes, err := yamlV3ToJSON(doc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}

		switch i {
		case 0:
			if err := um.Unmarshal(jsonBytes, &first); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal first spec: %w", err)
			}
		case 1:
			if err := um.Unmarshal(jsonBytes, &second); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal second spec: %w", err)
			}
		}
	}

	return &first, &second, nil
}

// helper: re-encode the generic YAML doc and convert it to JSON
func yamlV3ToJSON(v any) ([]byte, error) {
	y, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	return yaml.YAMLToJSON(y)
}

func LoadKubernetesExecutorConfig(
	componentSpec *pipelinespec.ComponentSpec,
	platformSpec *pipelinespec.PlatformSpec,
) (*kubernetesplatform.KubernetesExecutorConfig, error) {

	var kubernetesPlatformSpec kubernetesplatform.KubernetesExecutorConfig
	if componentSpec.GetExecutorLabel() == "" {
		return nil, fmt.Errorf("executor label not found")
	}

	executorLabel := componentSpec.GetExecutorLabel()
	if platformSpec.GetPlatforms() != nil {
		if singlePlatformSpecRaw, ok := platformSpec.GetPlatforms()["kubernetes"]; ok {

			var singlePlatformSpec pipelinespec.SinglePlatformSpec
			jsonBytes, err := protojson.Marshal(singlePlatformSpecRaw)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal single platform spec: %w", err)
			}
			if err := protojson.Unmarshal(jsonBytes, &singlePlatformSpec); err != nil {
				return nil, fmt.Errorf("failed to unmarshal single platform spec: %w", err)
			}

			if singlePlatformSpec.GetDeploymentSpec() == nil {
				return nil, fmt.Errorf("deployment spec not found")
			}
			if singlePlatformSpec.GetDeploymentSpec().GetExecutors() == nil {
				return nil, fmt.Errorf("executors not found")
			}

			executor := singlePlatformSpec.GetDeploymentSpec().GetExecutors()[executorLabel]
			if executor != nil {
				jsonBytes, err := protojson.Marshal(executor)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal executor config: %w", err)
				}
				if err := protojson.Unmarshal(jsonBytes, &kubernetesPlatformSpec); err != nil {
					return nil, fmt.Errorf("failed to unmarshal executor config: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("kubernetes platform config not found")
		}
	}
	return &kubernetesPlatformSpec, nil
}
