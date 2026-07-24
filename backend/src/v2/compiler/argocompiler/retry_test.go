// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler_test

import (
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

func TestRetryInParallelFor(t *testing.T) {
	// Initialize proxy config to avoid panic
	proxy.InitializeConfigWithEmptyForTests()

	// Read the yaml file
	yamlBytes, err := os.ReadFile("testdata/parallel_for_retry.yaml")
	if err != nil {
		t.Fatalf("Failed to read testdata/parallel_for_retry.yaml: %v", err)
	}

	// Unmarshal to a generic map first to get the PipelineSpec part
	var genericMap map[string]interface{}
	err = yaml.Unmarshal(yamlBytes, &genericMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal yaml: %v", err)
	}

	// Assuming the file content is directly mapped to PipelineSpec, but usually keys match json fields
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to convert yaml to json: %v", err)
	}

	spec := &pipelinespec.PipelineSpec{}
	err = protojson.Unmarshal(jsonBytes, spec)
	if err != nil {
		t.Fatalf("Failed to unmarshal pipeline spec: %v", err)
	}

	// Construct PipelineJob
	job := &pipelinespec.PipelineJob{
		PipelineSpec: &structpb.Struct{}, // We need to put spec into this struct?
		// Wait, PipelineJob definition
	}
	// The Compile function expects PipelineJob.
	// But PipelineJob has "pipeline_spec" as a Map (Struct).
	// We need to convert spec back to Struct.

	specStruct := &structpb.Struct{}
	specJson, _ := protojson.Marshal(spec)
	err = protojson.Unmarshal(specJson, specStruct)
	if err != nil {
		t.Fatalf("Failed to convert spec to struct: %v", err)
	}
	
	job = &pipelinespec.PipelineJob{
		PipelineSpec: specStruct,
	}

	options := &argocompiler.Options{
		// No PipelineName
	}

	// Compile
	wf, err := argocompiler.Compile(job, nil, options)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Check if retry strategy is present on the implementation template
	foundImplWithRetry := false
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.RetryStrategy != nil {
			if tmpl.RetryStrategy.Limit != nil && tmpl.RetryStrategy.Limit.StrVal == "{{inputs.parameters.retry-max-count}}" {
				foundImplWithRetry = true
				break
			}
		}
	}
	if !foundImplWithRetry {
		t.Errorf("Generic retry implementation template not found")
	}

	// Check if the loop body DAG passes the correct retry count
	foundArg := false
	// The loop component name is comp-for-loop-1
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "comp-for-loop-1" {
			if tmpl.DAG == nil {
				continue
			}
			for _, task := range tmpl.DAG.Tasks {
				if task.Name == "random-failure-op" {
					for _, p := range task.Arguments.Parameters {
						if p.Name == "retry-max-count" && p.Value != nil && *p.Value == "20" {
							foundArg = true
						}
					}
				}
			}
		}
	}

	// Print the workflow yaml for manual inspection
	wfBytes, _ := yaml.Marshal(wf)
	t.Logf("Workflow YAML:\n%s", string(wfBytes))

	if !foundArg {
		t.Errorf("Argument retry-max-count=20 not found in comp-for-loop-1 DAG")
	}

	// Check if retry strategy is present on the iteration template (structural fix for Argo issue)
	foundIterationRetry := false
	for _, tmpl := range wf.Spec.Templates {
		// Look for the iteration template (e.g. comp-for-loop-1-iteration)
		if strings.HasSuffix(tmpl.Name, "-iteration") {
			if tmpl.RetryStrategy != nil && tmpl.RetryStrategy.Limit != nil && tmpl.RetryStrategy.Limit.StrVal == "20" {
				foundIterationRetry = true
			}
		}
	}
	if !foundIterationRetry {
		t.Errorf("Retry strategy not found on iteration template (expected limit 20)")
	}
}
