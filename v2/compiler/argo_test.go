package compiler_test

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/v2/compiler"
)

func Test_argo_compiler(t *testing.T) {
	tests := []struct {
		jobPath      string
		expectedText string
	}{
		{
			jobPath: "testdata/hello_world.json",
			expectedText: `
        apiVersion: argoproj.io/v1alpha1
        kind: Workflow
        metadata:
          annotations:
            pipelines.kubeflow.org/v2_pipeline: "true"
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
              - namespace/n1/pipeline/hello-world
              - --run_id
              - '{{workflow.uid}}'
              - --dag_execution_id
              - '{{inputs.parameters.dag-execution-id}}'
              - --component
              - '{{inputs.parameters.component}}'
              - --task
              - '{{inputs.parameters.task}}'
              - --container
              - '{{inputs.parameters.container}}'
              - --execution_id_path
              - '{{outputs.parameters.execution-id.path}}'
              - --executor_input_path
              - '{{outputs.parameters.executor-input.path}}'
              - --cached_decision_path
              - '{{outputs.parameters.cached-decision.path}}'
              command:
              - driver
              image: gcr.io/ml-pipeline/kfp-driver:latest
              name: ""
              resources: {}
            inputs:
              parameters:
              - name: component
              - name: task
              - name: container
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
              - default: "false"
                name: cached-decision
                valueFrom:
                  default: "false"
                  path: /tmp/outputs/cached-decision
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
              - --pipeline_name
              - namespace/n1/pipeline/hello-world
              - --run_id
              - '{{workflow.uid}}'
              - --execution_id
              - '{{inputs.parameters.execution-id}}'
              - --executor_input
              - '{{inputs.parameters.executor-input}}'
              - --component_spec
              - '{{inputs.parameters.component}}'
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
              - launcher-v2
              - --copy
              - /kfp-launcher/launch
              image: gcr.io/ml-pipeline/kfp-launcher-v2:latest
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
              - name: component
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
                    value: '{{inputs.parameters.component}}'
                  - name: task
                    value: '{{inputs.parameters.task}}'
                  - name: container
                    value: '{"image":"python:3.7","command":["sh","-ec","program_path=$(mktemp)\nprintf
                      \"%s\" \"$0\" \u003e \"$program_path\"\npython3 -u \"$program_path\"
                      \"$@\"\n","def hello_world(text):\n    print(text)\n    return text\n\nimport
                      argparse\n_parser = argparse.ArgumentParser(prog=''Hello world'', description='''')\n_parser.add_argument(\"--text\",
                      dest=\"text\", type=str, required=True, default=argparse.SUPPRESS)\n_parsed_args
                      = vars(_parser.parse_args())\n\n_outputs = hello_world(**_parsed_args)\n"],"args":["--text","{{$.inputs.parameters[''text'']}}"]}'
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
                  - name: component
                    value: '{{inputs.parameters.component}}'
                dependencies:
                - driver
                name: container
                template: comp-hello-world-container
                when: '{{tasks.driver.outputs.parameters.cached-decision}} != true'
            inputs:
              parameters:
              - name: task
              - name: dag-execution-id
              - default: '{"inputDefinitions":{"parameters":{"text":{"type":"STRING"}}},"executorLabel":"exec-hello-world"}'
                name: component
            metadata: {}
            name: comp-hello-world
            outputs: {}
          - dag:
              tasks:
              - arguments:
                  parameters:
                  - name: dag-execution-id
                    value: '{{inputs.parameters.dag-execution-id}}'
                  - name: task
                    value: '{"taskInfo":{"name":"hello-world"},"inputs":{"parameters":{"text":{"componentInputParameter":"text"}}},"cachingOptions":{"enableCache":true},"componentRef":{"name":"comp-hello-world"}}'
                name: hello-world
                template: comp-hello-world
            inputs:
              parameters:
              - name: dag-execution-id
            metadata: {}
            name: root-dag
            outputs: {}
          - container:
              args:
              - --type
              - ROOT_DAG
              - --pipeline_name
              - namespace/n1/pipeline/hello-world
              - --run_id
              - '{{workflow.uid}}'
              - --component
              - '{{inputs.parameters.component}}'
              - --runtime_config
              - '{{inputs.parameters.runtime-config}}'
              - --execution_id_path
              - '{{outputs.parameters.execution-id.path}}'
              command:
              - driver
              image: gcr.io/ml-pipeline/kfp-driver:latest
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
    `,
		},
		{
			jobPath: "testdata/importer.json",
			expectedText: `
        apiVersion: argoproj.io/v1alpha1
        kind: Workflow
        metadata:
          annotations:
            pipelines.kubeflow.org/v2_pipeline: "true"
          creationTimestamp: null
          generateName: pipeline-with-importer-
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
              - --executor_type
              - importer
              - --task_spec
              - '{{inputs.parameters.task}}'
              - --component_spec
              - '{{inputs.parameters.component}}'
              - --importer_spec
              - '{{inputs.parameters.importer}}'
              - --pipeline_name
              - pipeline-with-importer
              - --run_id
              - '{{workflow.uid}}'
              - --pod_name
              - $(KFP_POD_NAME)
              - --pod_uid
              - $(KFP_POD_UID)
              - --mlmd_server_address
              - $(METADATA_GRPC_SERVICE_HOST)
              - --mlmd_server_port
              - $(METADATA_GRPC_SERVICE_PORT)
              command:
              - launcher-v2
              env:
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
              image: gcr.io/ml-pipeline/kfp-launcher-v2:latest
              name: ""
              resources: {}
            inputs:
              parameters:
              - name: task
              - default: '{"inputDefinitions":{"parameters":{"uri":{"type":"STRING"}}},"outputDefinitions":{"artifacts":{"artifact":{"artifactType":{"schemaTitle":"system.Dataset"}}}},"executorLabel":"exec-importer"}'
                name: component
              - default: '{"artifactUri":{"constantValue":{"stringValue":"gs://ml-pipeline-playground/shakespeare1.txt"}},"typeSchema":{"schemaTitle":"system.Dataset"}}'
                name: importer
            metadata: {}
            name: comp-importer
            outputs: {}
          - dag:
              tasks:
              - arguments:
                  parameters:
                  - name: dag-execution-id
                    value: '{{inputs.parameters.dag-execution-id}}'
                  - name: task
                    value: '{"taskInfo":{"name":"importer"},"inputs":{"parameters":{"uri":{"runtimeValue":{"constantValue":{"stringValue":"gs://ml-pipeline-playground/shakespeare1.txt"}}}}},"cachingOptions":{"enableCache":true},"componentRef":{"name":"comp-importer"}}'
                name: importer
                template: comp-importer
            inputs:
              parameters:
              - name: dag-execution-id
            metadata: {}
            name: root-dag
            outputs: {}
          - container:
              args:
              - --type
              - ROOT_DAG
              - --pipeline_name
              - pipeline-with-importer
              - --run_id
              - '{{workflow.uid}}'
              - --component
              - '{{inputs.parameters.component}}'
              - --runtime_config
              - '{{inputs.parameters.runtime-config}}'
              - --execution_id_path
              - '{{outputs.parameters.execution-id.path}}'
              command:
              - driver
              image: gcr.io/ml-pipeline/kfp-driver:latest
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
          - dag:
              tasks:
              - arguments:
                  parameters:
                  - name: component
                    value: '{"inputDefinitions":{"parameters":{"dataset2":{"type":"STRING"}}},"dag":{"tasks":{"importer":{"taskInfo":{"name":"importer"},"inputs":{"parameters":{"uri":{"runtimeValue":{"constantValue":{"stringValue":"gs://ml-pipeline-playground/shakespeare1.txt"}}}}},"cachingOptions":{"enableCache":true},"componentRef":{"name":"comp-importer"}}}}}'
                  - name: task
                    value: '{}'
                  - name: runtime-config
                    value: '{}'
                name: driver
                template: system-dag-driver
              - arguments:
                  parameters:
                  - name: dag-execution-id
                    value: '{{tasks.driver.outputs.parameters.execution-id}}'
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
          `,
		},
	}
	for _, tt := range tests {
		job := load(t, tt.jobPath)
		wf, err := compiler.Compile(job, nil)
		if err != nil {
			t.Error(err)
		}
		var expected wfapi.Workflow
		err = yaml.Unmarshal([]byte(tt.expectedText), &expected)
		if err != nil {
			t.Fatal(err)
		}
		if !cmp.Equal(wf, &expected) {
			got, err := yaml.Marshal(wf)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n got:\n%s\n", tt.jobPath, cmp.Diff(&expected, wf), string(got))
		}

	}

}
