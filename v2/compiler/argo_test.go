package compiler_test

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/v2/compiler"
)

func Test_argo_compiler(t *testing.T) {
	job := load(t, "testdata/hello_world.json")
	wf, err := compiler.Compile(job)
	if err != nil {
		t.Error(err)
	}
	expectedText := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  serviceAccountName: pipeline-runner
  entrypoint: root
  templates:
  - container:
      args:
      - --text
      - '{{$.inputs.parameters[''text'']}}'
      command:
      - sh
      - -ec
      - |
        program_path=$(mktemp)
        printf "%s" "$0" > "$program_path"
        python3 -u "$program_path" "$@"
      - |
        def hello_world(text):
            print(text)
            return text

        import argparse
        _parser = argparse.ArgumentParser(prog='Hello world', description='')
        _parser.add_argument("--text", dest="text", type=str, required=True, default=argparse.SUPPRESS)
        _parsed_args = vars(_parser.parse_args())

        _outputs = hello_world(**_parsed_args)
      image: python:3.7
    name: comp-hello-world
  - dag:
      tasks:
      - name: hello-world
        template: comp-hello-world
    name: root`
	var expected wfapi.Workflow
	err = yaml.Unmarshal([]byte(expectedText), &expected)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wf, &expected) {
		t.Errorf("Incorrect compiled workflow, diff:%s", cmp.Diff(&expected, wf))
	}
}
