package compiler_test

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/v2/compiler"
)

func Test_argo_compiler(t *testing.T) {
	jobPath := "testdata/hello_world.json"
	job := load(t, jobPath)
	wf, err := compiler.Compile(job)
	if err != nil {
		t.Error(err)
	}
	expectedText := `
  apiVersion: argoproj.io/v1alpha1
  kind: Workflow
  metadata:
    creationTimestamp: null
    generateName: hello-world-
  spec:
    arguments: {}
    entrypoint: root
    podMetadata:
      annotations:
        pipelines.kubeflow.org/v2_component: "true"
      labels:
        pipelines.kubeflow.org/v2_component: "true"
    serviceAccountName: pipeline-runner
    templates:
    - container:
        args:
        - --type
        - CONTAINER
        - --pipeline_name
        - hello-world
        - --run_id
        - '{{workflow.uid}}'
        - --dag_context_id
        - '{{inputs.parameters.dag-context-id}}'
        - --dag_execution_id
        - '{{inputs.parameters.dag-execution-id}}'
        - --component
        - '{{inputs.parameters.component}}'
        - --task
        - '{{inputs.parameters.task}}'
        - --execution_id_path
        - '{{outputs.parameters.execution-id.path}}'
        - --executor_input_path
        - '{{outputs.parameters.executor-input.path}}'
        command:
        - /bin/kfp/driver
        image: gcr.io/gongyuan-dev/dev/kfp-driver:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: component
        - name: task
        - name: dag-context-id
        - name: dag-execution-id
      metadata: {}
      name: system-container-driver
      outputs:
        parameters:
        - name: execution-id
          valueFrom:
            path: /tmp/outputs/execution-id
        - name: executor-input
          valueFrom:
            path: /tmp/outputs/executor-input
    - container:
        args:
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
        - --text
        - '{{$.inputs.parameters[''text'']}}'
        command:
        - /kfp-launcher/launch
        - --execution_id
        - '{{inputs.parameters.execution-id}}'
        - --executor_input
        - '{{inputs.parameters.executor-input}}'
        - --namespace
        - $(KFP_NAMESPACE)
        - --pod_name
        - $(KFP_POD_NAME)
        - --pod_uid
        - $(KFP_POD_UID)
        - --mlmd_server_address
        - $(METADATA_GRPC_SERVICE_HOST)
        - --mlmd_server_port
        - $(METADATA_GRPC_SERVICE_PORT)
        - --
        env:
        - name: KFP_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: KFP_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: KFP_POD_UID
          valueFrom:
            fieldRef:
              fieldPath: metadata.uid
        envFrom:
        - configMapRef:
            name: metadata-grpc-configmap
            optional: true
        image: python:3.7
        name: ""
        resources: {}
        volumeMounts:
        - mountPath: /kfp-launcher
          name: kfp-launcher
      initContainers:
      - command:
        - mount_launcher.sh
        image: gcr.io/gongyuan-dev/dev/kfp-launcher-v2:latest
        imagePullPolicy: Always
        name: kfp-launcher
        resources: {}
        volumeMounts:
        - mountPath: /kfp-launcher
          name: kfp-launcher
      inputs:
        parameters:
        - name: executor-input
        - name: execution-id
      metadata: {}
      name: comp-hello-world-container
      outputs: {}
      volumes:
      - emptyDir: {}
        name: kfp-launcher
    - dag:
        tasks:
        - arguments:
            parameters:
            - name: component
              value: '{"inputDefinitions":{"parameters":{"text":{"type":"STRING"}}},"executorLabel":"exec-hello-world"}'
            - name: task
              value: '{{inputs.parameters.task}}'
            - name: dag-context-id
              value: '{{inputs.parameters.dag-context-id}}'
            - name: dag-execution-id
              value: '{{inputs.parameters.dag-execution-id}}'
          name: driver
          template: system-container-driver
        - arguments:
            parameters:
            - name: executor-input
              value: '{{tasks.driver.outputs.parameters.executor-input}}'
            - name: execution-id
              value: '{{tasks.driver.outputs.parameters.execution-id}}'
          dependencies:
          - driver
          name: container
          template: comp-hello-world-container
      inputs:
        parameters:
        - name: task
        - name: dag-context-id
        - name: dag-execution-id
      metadata: {}
      name: comp-hello-world
      outputs: {}
    - dag:
        tasks:
        - arguments:
            parameters:
            - name: dag-context-id
              value: '{{inputs.parameters.dag-context-id}}'
            - name: dag-execution-id
              value: '{{inputs.parameters.dag-execution-id}}'
            - name: task
              value: '{"taskInfo":{"name":"hello-world"},"inputs":{"parameters":{"text":{"componentInputParameter":"text"}}},"cachingOptions":{"enableCache":true},"componentRef":{"name":"comp-hello-world"}}'
          name: hello-world
          template: comp-hello-world
      inputs:
        parameters:
        - name: dag-context-id
        - name: dag-execution-id
      metadata: {}
      name: root-dag
      outputs: {}
    - container:
        args:
        - --type
        - ROOT_DAG
        - --pipeline_name
        - hello-world
        - --run_id
        - '{{workflow.uid}}'
        - --component
        - '{{inputs.parameters.component}}'
        - --runtime_config
        - '{{inputs.parameters.runtime-config}}'
        - --execution_id_path
        - '{{outputs.parameters.execution-id.path}}'
        - --context_id_path
        - '{{outputs.parameters.context-id.path}}'
        command:
        - /bin/kfp/driver
        image: gcr.io/gongyuan-dev/dev/kfp-driver:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: component
        - name: runtime-config
      metadata: {}
      name: system-dag-driver
      outputs:
        parameters:
        - name: execution-id
          valueFrom:
            path: /tmp/outputs/execution-id
        - name: context-id
          valueFrom:
            path: /tmp/outputs/context-id
    - dag:
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
        - arguments:
            parameters:
            - name: dag-execution-id
              value: '{{tasks.driver.outputs.parameters.execution-id}}'
            - name: dag-context-id
              value: '{{tasks.driver.outputs.parameters.context-id}}'
          dependencies:
          - driver
          name: dag
          template: root-dag
      inputs: {}
      metadata: {}
      name: root
      outputs: {}
  status:
    finishedAt: null
    startedAt: null
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
