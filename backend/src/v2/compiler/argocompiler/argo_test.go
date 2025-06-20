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
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

var update = flag.Bool("update", false, "update golden files")

func Test_argo_compiler(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	tests := []struct {
		jobPath          string // path of input PipelineJob to compile
		platformSpecPath string // path of possible input PlatformSpec to compile
		argoYAMLPath     string // path of expected output argo workflow YAML
		envVars          map[string]string
		compilerOptions  argocompiler.Options
	}{
		{
			jobPath:          "../testdata/hello_world_with_retry.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world_with_retry.yaml",
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world.yaml",
		},
		{
			jobPath:          "../testdata/importer.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/importer.yaml",
		},
		{
			jobPath:          "../testdata/multiple_parallel_loops.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/multiple_parallel_loops.yaml",
		},
		{
			jobPath:          "../testdata/create_mount_delete_dynamic_pvc.json",
			platformSpecPath: "../testdata/create_mount_delete_dynamic_pvc_platform.json",
			argoYAMLPath:     "testdata/create_mount_delete_dynamic_pvc.yaml",
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "../testdata/create_pod_metadata.json",
			argoYAMLPath:     "testdata/create_pod_metadata.yaml",
		},
		{
			jobPath:          "../testdata/exit_handler.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/exit_handler.yaml",
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world_run_as_user.yaml",
			envVars:          map[string]string{"PIPELINE_RUN_AS_USER": "1001"},
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world_log_level.yaml",
			envVars:          map[string]string{"PIPELINE_LOG_LEVEL": "3"},
		},
		{
			jobPath:          "../testdata/hello_world_with_retry_all_args.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world_with_retry_all_args.yaml",
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/hello_world_cache_disabled.yaml",
			compilerOptions:  argocompiler.Options{CacheDisabled: true},
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "../testdata/hello_world_pvc_workspace.json",
			argoYAMLPath:     "testdata/hello_world_pvc_workspace.yaml",
		},
		// retry set at pipeline level only.
		{
			jobPath:          "../testdata/nested_pipeline_pipeline_retry.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/nested_pipeline_pipeline_retry.yaml",
		},
		// retry set at component level only.
		{
			jobPath:          "../testdata/nested_pipeline_sub_component_retry.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/nested_pipeline_sub_component_retry.yaml",
		},
		// retry set at both component and pipeline level.
		{
			jobPath:          "../testdata/nested_pipeline_all_level_retry.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/nested_pipeline_all_level_retry.yaml",
		},
		{
			jobPath:          "../testdata/final_status_state.json",
			platformSpecPath: "",
			argoYAMLPath:     "testdata/final_status_state.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			prevEnvVars := map[string]string{}

			for envVarName, envVarValue := range tt.envVars {
				prevEnvVars[envVarName] = os.Getenv(envVarName)

				os.Setenv(envVarName, envVarValue)
			}

			defer func() {
				for envVarName, envVarValue := range prevEnvVars {
					if envVarValue == "" {
						os.Unsetenv(envVarName)
					} else {
						os.Setenv(envVarName, envVarValue)
					}
				}
			}()

			job, platformSpec := load(t, tt.jobPath, tt.platformSpecPath)
			if *update {
				wf, err := argocompiler.Compile(job, platformSpec, &tt.compilerOptions)
				if err != nil {
					t.Fatal(err)
				}
				got, err := yaml.Marshal(wf)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(tt.argoYAMLPath, got, 0x664)
				if err != nil {
					t.Fatal(err)
				}
			}
			argoYAML, err := os.ReadFile(tt.argoYAMLPath)
			if err != nil {
				t.Fatal(err)
			}
			wf, err := argocompiler.Compile(job, platformSpec, &tt.compilerOptions)
			if err != nil {
				t.Error(err)
			}

			// mask the driver launcher image hash to maintain test stability
			for _, template := range wf.Spec.Templates {
				if template.Container != nil && strings.Contains(template.Container.Image, "kfp-driver") {
					template.Container.Image = "ghcr.io/kubeflow/kfp-driver"
				}
				if template.Container != nil && strings.Contains(template.Container.Image, "kfp-launcher") {
					template.Container.Image = "ghcr.io/kubeflow/kfp-launcher"
				}
				for i := range template.InitContainers {
					if strings.Contains(template.InitContainers[i].Image, "kfp-launcher") {
						template.InitContainers[i].Image = "ghcr.io/kubeflow/kfp-launcher"
					}
				}
			}

			var expected wfapi.Workflow
			err = yaml.Unmarshal(argoYAML, &expected)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(wf, &expected) {
				t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n", tt.jobPath, cmp.Diff(&expected, wf))
			}
		})

	}

}

func load(t *testing.T, path string, platformSpecPath string) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	job := &pipelinespec.PipelineJob{}
	if err := protojson.Unmarshal(content, job); err != nil {
		t.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, string(content))
	}

	platformSpec := &pipelinespec.PlatformSpec{}
	if platformSpecPath != "" {
		content, err = os.ReadFile(platformSpecPath)
		if err != nil {
			t.Error(err)
		}
		if err := protojson.Unmarshal(content, platformSpec); err != nil {
			t.Errorf("Failed to parse platform spec, error: %s, spec: %v", err, string(content))
		}
		return job, platformSpec.Platforms["kubernetes"]
	}
	return job, nil
}
