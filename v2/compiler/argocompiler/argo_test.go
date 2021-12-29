// Copyright 2021 The Kubeflow Authors
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
	"io/ioutil"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/compiler/argocompiler"
	"google.golang.org/protobuf/encoding/protojson"
)

var update = flag.Bool("update", false, "update golden files")

func Test_argo_compiler(t *testing.T) {
	tests := []struct {
		jobPath      string // path of input PipelineJob to compile
		argoYAMLPath string // path of expected output argo workflow YAML
	}{
		{
			jobPath:      "../testdata/hello_world.json",
			argoYAMLPath: "testdata/hello_world.yaml",
		},
		{
			jobPath:      "../testdata/importer.json",
			argoYAMLPath: "testdata/importer.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {

			job := load(t, tt.jobPath)
			if *update {
				wf, err := argocompiler.Compile(job, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := yaml.Marshal(wf)
				if err != nil {
					t.Fatal(err)
				}
				err = ioutil.WriteFile(tt.argoYAMLPath, got, 0x664)
				if err != nil {
					t.Fatal(err)
				}
			}
			argoYAML, err := ioutil.ReadFile(tt.argoYAMLPath)
			if err != nil {
				t.Fatal(err)
			}
			wf, err := argocompiler.Compile(job, nil)
			if err != nil {
				t.Error(err)
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

func load(t *testing.T, path string) *pipelinespec.PipelineJob {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	job := &pipelinespec.PipelineJob{}
	if err := protojson.Unmarshal(content, job); err != nil {
		t.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, string(content))
	}
	return job
}
