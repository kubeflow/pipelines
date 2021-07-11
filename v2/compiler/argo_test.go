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
    annotations:
      pipelines.kubeflow.org/v2_pipeline: "true"
    generateName: hello-world-
  spec:
    entrypoint: root
    podMetadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
    serviceAccountName: pipeline-runner
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
      name: root-dag
    - container:
        args:
        - --pipeline_name
        - hello-world
        - --run_id
        - '{{workflow.uid}}'
        - --component
        - '{{inputs.parameters.component}}'
        - --task
        - '{{inputs.parameters.task}}'
        - --runtime_config
        - '{{inputs.parameters.runtime-config}}'
        - --execution_id_path
        - '{{outputs.parameters.execution-id.path}}'
        - --context_id_path
        - '{{outputs.parameters.context-id.path}}'
        command:
        - /bin/kfp/driver
        image: gcr.io/gongyuan-dev/dev/kfp-driver:latest
      inputs:
        parameters:
        - name: component
        - name: task
        - name: runtime-config
      name: system-dag-driver
      outputs:
        parameters:
        - name: execution-id
          valueFrom:
            path: /tmp/outputs/execution-id
        - name: context-id
          valueFrom:
            path: /tmp/outputs/context-id
    - name: root
      dag:
        tasks:
        - arguments:
            parameters:
            - name: component
              value: '{"inputDefinitions":{"parameters":{"text":{"type":"STRING"}}},"dag":{"tasks":{"hello-world":{"taskInfo":{"name":"hello-world"},"inputs":{"parameters":{"text":{"componentInputParameter":"text"}}},"cachingOptions":{"enableCache":true},"componentRef":{"name":"comp-hello-world"}}}}}'
            - name: task
              value: '{}'
            - name: runtime-config
              value: '{"parameters":{"text":{"stringValue":"hi there"}}}'
          name: driver
          template: system-dag-driver
        - dependencies:
          - driver
          name: dag
          template: root-dag
    `
	var expected wfapi.Workflow
	err = yaml.Unmarshal([]byte(expectedText), &expected)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(wf, &expected) {
		got, err := yaml.Marshal(wf)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n got:\n%s\n", jobPath, cmp.Diff(&expected, wf), string(got))
	}
}
