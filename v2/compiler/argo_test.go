package compiler_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/v2/compiler"
)

var update = flag.Bool("update", false, "update golden files")

func Test_argo_compiler(t *testing.T) {
	tests := []struct {
		jobPath      string // path of input PipelineJob to compile
		argoYAMLPath string // path of expected output argo workflow YAML
	}{
		{
			jobPath:      "testdata/hello_world.json",
			argoYAMLPath: "testdata/hello_world.yaml",
		},
		// TODO(Bobgy): re-enable importer
		// {
		// 	jobPath:      "testdata/importer.json",
		// 	argoYAMLPath: "testdata/importer.yaml",
		// },
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {

			job := load(t, tt.jobPath)
			if *update {
				wf, err := compiler.Compile(job, nil)
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
			wf, err := compiler.Compile(job, nil)
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
