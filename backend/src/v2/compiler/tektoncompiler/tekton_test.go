// Copyright 2023 The Kubeflow Authors
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

package tektoncompiler_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/tektoncompiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"google.golang.org/protobuf/encoding/protojson"
)

var update = flag.Bool("update", false, "update golden files")

func Test_tekton_compiler(t *testing.T) {
	tests := []struct {
		jobPath          string // path of input PipelineJob to compile
		platformSpecPath string // path of platform spec
		tektonYAMLPath   string // path of expected output argo workflow YAML
	}{
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/hello_world.yaml",
		},
		{
			jobPath:          "../testdata/importer.json",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/importer.yaml",
		},
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "../testdata/create_pod_metadata.json",
			tektonYAMLPath:   "testdata/pod_metadata.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {

			job, platformSpec := load(t, tt.jobPath, tt.platformSpecPath, "json")
			if *update {
				pr, err := tektoncompiler.Compile(job, platformSpec, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := yaml.Marshal(pr)
				if err != nil {
					t.Fatal(err)
				}
				err = ioutil.WriteFile(tt.tektonYAMLPath, got, 0664)
				if err != nil {
					t.Fatal(err)
				}
			}
			tektonYAML, err := ioutil.ReadFile(tt.tektonYAMLPath)
			if err != nil {
				t.Fatal(err)
			}
			pr, err := tektoncompiler.Compile(job, platformSpec, nil)
			if err != nil {
				t.Error(err)
			}
			var expected pipelineapi.PipelineRun
			err = yaml.Unmarshal(tektonYAML, &expected)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(pr, &expected, compareRawExtension(), cmpopts.EquateEmpty()) {
				t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n", tt.jobPath, cmp.Diff(&expected, pr))
			}
		})

	}

}

type testInputs struct {
	yamlPath         string
	platformSpecPath string
	tektonYAMLPath   string
}

func TestMnist(t *testing.T) {

	testCompile(t, testInputs{
		yamlPath:         "testdata/mnist_pipeline_ir.yaml",
		platformSpecPath: "",
		tektonYAMLPath:   "testdata/mnist_pipeline.yaml",
	})
}

func TestExitHandler(t *testing.T) {

	testCompile(t, testInputs{
		yamlPath:         "testdata/exit_handler_ir.yaml",
		platformSpecPath: "",
		tektonYAMLPath:   "testdata/exit_handler.yaml",
	})
}

func TestLoopStatic(t *testing.T) {
	testCompile(t, testInputs{
		yamlPath:         "testdata/loop_static_ir.yaml",
		platformSpecPath: "",
		tektonYAMLPath:   "testdata/loop_static.yaml",
	})
}

func TestNestedLoop(t *testing.T) {
	testCompile(t, testInputs{
		yamlPath:         "testdata/nestedloop_ir.yaml",
		platformSpecPath: "",
		tektonYAMLPath:   "testdata/nestedloop.yaml",
	})
}

func compareRawExtension() cmp.Option {
	return cmp.Comparer(func(a, b runtime.RawExtension) bool {
		var src, target interface{}
		err := yaml.Unmarshal([]byte(a.Raw), &src)
		if err != nil {
			return false
		}
		err = yaml.Unmarshal([]byte(b.Raw), &target)
		if err != nil {
			return false
		}
		rev := cmp.Equal(src, target, sortedRunAfter(), cmpopts.EquateEmpty())
		if !rev {
			fmt.Printf("RawExtension: %s\n", cmp.Diff(src, target))
		}
		return rev
	})
}

func comparePipelineTask() cmp.Option {
	return cmp.Comparer(func(a, b pipelineapi.PipelineTask) bool {
		sort.Strings(a.RunAfter)
		sort.Strings(b.RunAfter)
		return cmp.Equal(a, b, compareRawExtension(), cmpopts.EquateEmpty())
	})
}

func sortedRunAfter() cmp.Option {
	return cmp.Transformer("Sort", func(in map[string]any) map[string]any {
		v, ok := in["runAfter"]
		if ok {
			runAfter, ok := v.([]any)
			if len(runAfter) == 0 || !ok {
				return in
			}
			sorted := make([]string, 0, len(runAfter))
			for _, i := range runAfter {
				sorted = append(sorted, i.(string))
			}
			sort.Strings(sorted)
			in["runAfter"] = sorted
		}
		return in
	})
}

func testCompile(t *testing.T, test testInputs) {
	t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
		job, platformSpec := load(t, test.yamlPath, test.platformSpecPath, "yaml")
		if *update {
			pr, err := tektoncompiler.Compile(job, platformSpec, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Marshal(pr)
			if err != nil {
				t.Fatal(err)
			}
			err = ioutil.WriteFile(test.tektonYAMLPath, got, 0644)
			if err != nil {
				t.Fatal(err)
			}
		}
		tektonYAML, err := ioutil.ReadFile(test.tektonYAMLPath)
		if err != nil {
			t.Fatal(err)
		}
		pr, err := tektoncompiler.Compile(job, platformSpec, nil)
		if err != nil {
			t.Error(err)
		}
		var expected pipelineapi.PipelineRun
		err = yaml.Unmarshal(tektonYAML, &expected)
		if err != nil {
			t.Fatal(err)
		}
		if !cmp.Equal(pr, &expected, comparePipelineTask(), cmpopts.EquateEmpty()) {
			t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n", test.yamlPath, cmp.Diff(pr, &expected))
		}
	})

}

func load(t *testing.T, path string, platformSpecPath string, fileType string) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	if fileType == "yaml" {
		content, err = yaml.YAMLToJSON(content)
		if err != nil {
			t.Error(err)
		}
	}
	job := &pipelinespec.PipelineJob{}
	if err := protojson.Unmarshal(content, job); err != nil {
		t.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, string(content))
	}

	platformSpec := &pipelinespec.PlatformSpec{}
	if platformSpecPath != "" {
		content, err = ioutil.ReadFile(platformSpecPath)
		if err != nil {
			t.Error(err)
		}
		if fileType == "yaml" {
			content, err = yaml.YAMLToJSON(content)
			if err != nil {
				t.Error(err)
			}
		}
		if err := protojson.Unmarshal(content, platformSpec); err != nil {
			t.Errorf("Failed to parse platform spec, error: %s, spec: %v", err, string(content))
		}
		return job, platformSpec.Platforms["kubernetes"]
	}
	return job, nil
}
