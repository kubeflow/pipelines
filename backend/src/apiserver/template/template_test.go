// Copyright 2018-2022 The Kubeflow Authors
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

package template

import (
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFailValidation(t *testing.T) {
	wf := unmarshalWf(emptyName)
	wf.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "dup", Value: v1alpha1.AnyStringPtr("value1")}}
	templateBytes, _ := yaml.Marshal(wf)
	_, err := ValidateWorkflow([]byte(templateBytes))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "name is required")
	}
}

func TestValidateWorkflow_ParametersTooLong(t *testing.T) {
	var params []v1alpha1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, v1alpha1.Parameter{Name: "name1", Value: v1alpha1.AnyStringPtr("value1")})
	}
	template := v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
		Parameters: params}}}
	templateBytes, _ := yaml.Marshal(template)
	_, err := ValidateWorkflow(templateBytes)
	assert.Equal(t, codes.InvalidArgument, err.(*commonutil.UserError).ExternalStatusCode())
}

func TestParseSpecFormat(t *testing.T) {
	tt := []struct {
		template     string
		templateType TemplateType
	}{{
		// standard match
		template: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow`,
		templateType: V1,
	}, { // template contains content too
		template:     template,
		templateType: V1,
	}, {
		// version does not matter
		template: `
apiVersion: argoproj.io/v1alpha2
kind: Workflow`,
		templateType: V1,
	}, {
		template:     "",
		templateType: Unknown,
	}, {
		template:     "{}",
		templateType: Unknown,
	}, {
		// group incorrect
		template: `
apiVersion: pipelines.kubeflow.org/v1alpha1
kind: Workflow`,
		templateType: Unknown,
	}, {
		// kind incorrect
		template: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow`,
		templateType: Unknown,
	}, {
		template:     `{"abc": "def", "b": {"key": 3}}`,
		templateType: Unknown,
	}, {
		template:     v2SpecHelloWorldYAML,
		templateType: V2,
	}}

	for _, test := range tt {
		format := inferTemplateFormat([]byte(test.template))
		if format != test.templateType {
			t.Errorf("InferSpecFormat(%s)=%q, expect %q", test.template, format, test.templateType)
		}
	}
}

func unmarshalWf(yamlStr string) *v1alpha1.Workflow {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

var template = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: dup
        value: "value1"
    container:
      image: docker/whalesay:latest`

var emptyName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: ""
        value: "value1"
    container:
      image: docker/whalesay:latest`

var v2SpecHelloWorldYAML = `
# this is a comment
components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        text:
          type: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - "--text"
        - "{{$.inputs.parameters['text']}}"
        command:
        - sh
        - "-ec"
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
pipelineInfo:
  name: namespace/n1/pipeline/hello-world
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            text:
              componentInputParameter: text
        taskInfo:
          name: hello-world
  inputDefinitions:
    parameters:
      text:
        type: STRING
schemaVersion: 2.0.0
sdkVersion: kfp-1.6.5
`

var WorkflowSpecV1 = "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"generateName\":\"hello-world-\",\"creationTimestamp\":null,\"annotations\":{\"pipelines.kubeflow.org/components-comp-hello-world\":\"{\\\"executorLabel\\\":\\\"exec-hello-world\\\",\\\"inputDefinitions\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"type\\\":\\\"STRING\\\"}}}}\",\"pipelines.kubeflow.org/components-root\":\"{\\\"dag\\\":{\\\"tasks\\\":{\\\"hello-world\\\":{\\\"cachingOptions\\\":{\\\"enableCache\\\":true},\\\"componentRef\\\":{\\\"name\\\":\\\"comp-hello-world\\\"},\\\"inputs\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"componentInputParameter\\\":\\\"text\\\"}}},\\\"taskInfo\\\":{\\\"name\\\":\\\"hello-world\\\"}}}},\\\"inputDefinitions\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"type\\\":\\\"STRING\\\"}}}}\",\"pipelines.kubeflow.org/implementations-comp-hello-world\":\"{\\\"args\\\":[\\\"--text\\\",\\\"{{$.inputs.parameters['text']}}\\\"],\\\"command\\\":[\\\"sh\\\",\\\"-ec\\\",\\\"program_path=$(mktemp)\\\\nprintf \\\\\\\"%s\\\\\\\" \\\\\\\"$0\\\\\\\" \\\\u003e \\\\\\\"$program_path\\\\\\\"\\\\npython3 -u \\\\\\\"$program_path\\\\\\\" \\\\\\\"$@\\\\\\\"\\\\n\\\",\\\"def hello_world(text):\\\\n    print(text)\\\\n    return text\\\\n\\\\nimport argparse\\\\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\\\\n_parser.add_argument(\\\\\\\"--text\\\\\\\", dest=\\\\\\\"text\\\\\\\", type=str, required=True, default=argparse.SUPPRESS)\\\\n_parsed_args = vars(_parser.parse_args())\\\\n\\\\n_outputs = hello_world(**_parsed_args)\\\\n\\\"],\\\"image\\\":\\\"python:3.7\\\"}\"}},\"spec\":{\"templates\":[{\"name\":\"system-container-driver\",\"inputs\":{\"parameters\":[{\"name\":\"component\"},{\"name\":\"task\"},{\"name\":\"container\"},{\"name\":\"parent-dag-id\"},{\"name\":\"iteration-index\",\"default\":\"-1\"}]},\"outputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"valueFrom\":{\"path\":\"/tmp/outputs/pod-spec-patch\",\"default\":\"\"}},{\"name\":\"cached-decision\",\"default\":\"false\",\"valueFrom\":{\"path\":\"/tmp/outputs/cached-decision\",\"default\":\"false\"}},{\"name\":\"condition\",\"valueFrom\":{\"path\":\"/tmp/outputs/condition\",\"default\":\"true\"}}]},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-driver@sha256:a2efa29022573d9bcc92ee0a843ac37bef20333d3e6b8d9d7fbe97cbd346d84c\",\"command\":[\"driver\"],\"args\":[\"--type\",\"CONTAINER\",\"--pipeline_name\",\"namespace/n1/pipeline/hello-world\",\"--run_id\",\"{{workflow.uid}}\",\"--dag_execution_id\",\"{{inputs.parameters.parent-dag-id}}\",\"--component\",\"{{inputs.parameters.component}}\",\"--task\",\"{{inputs.parameters.task}}\",\"--container\",\"{{inputs.parameters.container}}\",\"--iteration_index\",\"{{inputs.parameters.iteration-index}}\",\"--cached_decision_path\",\"{{outputs.parameters.cached-decision.path}}\",\"--pod_spec_patch_path\",\"{{outputs.parameters.pod-spec-patch.path}}\",\"--condition_path\",\"{{outputs.parameters.condition.path}}\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"512Mi\"},\"requests\":{\"cpu\":\"100m\",\"memory\":\"64Mi\"}}}},{\"name\":\"system-container-executor\",\"inputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\"},{\"name\":\"cached-decision\",\"default\":\"false\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"executor\",\"template\":\"system-container-impl\",\"arguments\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"value\":\"{{inputs.parameters.pod-spec-patch}}\"}]},\"when\":\"{{inputs.parameters.cached-decision}} != true\"}]}},{\"name\":\"system-container-impl\",\"inputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline/should-be-overridden-during-runtime\",\"command\":[\"should-be-overridden-during-runtime\"],\"envFrom\":[{\"configMapRef\":{\"name\":\"metadata-grpc-configmap\",\"optional\":true}}],\"env\":[{\"name\":\"KFP_POD_NAME\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.name\"}}},{\"name\":\"KFP_POD_UID\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.uid\"}}}],\"resources\":{},\"volumeMounts\":[{\"name\":\"kfp-launcher\",\"mountPath\":\"/kfp-launcher\"}]},\"volumes\":[{\"name\":\"kfp-launcher\",\"emptyDir\":{}}],\"initContainers\":[{\"name\":\"kfp-launcher\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-launcher-v2@sha256:4513cf5c10c252d94f383ce51a890514799c200795e3de5e90f91b98b2e2f959\",\"command\":[\"launcher-v2\",\"--copy\",\"/kfp-launcher/launch\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"128Mi\"},\"requests\":{\"cpu\":\"100m\"}},\"volumeMounts\":[{\"name\":\"kfp-launcher\",\"mountPath\":\"/kfp-launcher\"}]}],\"podSpecPatch\":\"{{inputs.parameters.pod-spec-patch}}\"},{\"name\":\"root\",\"inputs\":{\"parameters\":[{\"name\":\"parent-dag-id\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"hello-world-driver\",\"template\":\"system-container-driver\",\"arguments\":{\"parameters\":[{\"name\":\"component\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/components-comp-hello-world}}\"},{\"name\":\"task\",\"value\":\"{\\\"cachingOptions\\\":{\\\"enableCache\\\":true},\\\"componentRef\\\":{\\\"name\\\":\\\"comp-hello-world\\\"},\\\"inputs\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"componentInputParameter\\\":\\\"text\\\"}}},\\\"taskInfo\\\":{\\\"name\\\":\\\"hello-world\\\"}}\"},{\"name\":\"container\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/implementations-comp-hello-world}}\"},{\"name\":\"parent-dag-id\",\"value\":\"{{inputs.parameters.parent-dag-id}}\"}]}},{\"name\":\"hello-world\",\"template\":\"system-container-executor\",\"arguments\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"value\":\"{{tasks.hello-world-driver.outputs.parameters.pod-spec-patch}}\"},{\"name\":\"cached-decision\",\"default\":\"false\",\"value\":\"{{tasks.hello-world-driver.outputs.parameters.cached-decision}}\"}]},\"depends\":\"hello-world-driver.Succeeded\"}]}},{\"name\":\"system-dag-driver\",\"inputs\":{\"parameters\":[{\"name\":\"component\"},{\"name\":\"runtime-config\",\"default\":\"\"},{\"name\":\"task\",\"default\":\"\"},{\"name\":\"parent-dag-id\",\"default\":\"0\"},{\"name\":\"iteration-index\",\"default\":\"-1\"},{\"name\":\"driver-type\",\"default\":\"DAG\"}]},\"outputs\":{\"parameters\":[{\"name\":\"execution-id\",\"valueFrom\":{\"path\":\"/tmp/outputs/execution-id\"}},{\"name\":\"iteration-count\",\"valueFrom\":{\"path\":\"/tmp/outputs/iteration-count\",\"default\":\"0\"}},{\"name\":\"condition\",\"valueFrom\":{\"path\":\"/tmp/outputs/condition\",\"default\":\"true\"}}]},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-driver@sha256:a2efa29022573d9bcc92ee0a843ac37bef20333d3e6b8d9d7fbe97cbd346d84c\",\"command\":[\"driver\"],\"args\":[\"--type\",\"{{inputs.parameters.driver-type}}\",\"--pipeline_name\",\"namespace/n1/pipeline/hello-world\",\"--run_id\",\"{{workflow.uid}}\",\"--dag_execution_id\",\"{{inputs.parameters.parent-dag-id}}\",\"--component\",\"{{inputs.parameters.component}}\",\"--task\",\"{{inputs.parameters.task}}\",\"--runtime_config\",\"{{inputs.parameters.runtime-config}}\",\"--iteration_index\",\"{{inputs.parameters.iteration-index}}\",\"--execution_id_path\",\"{{outputs.parameters.execution-id.path}}\",\"--iteration_count_path\",\"{{outputs.parameters.iteration-count.path}}\",\"--condition_path\",\"{{outputs.parameters.condition.path}}\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"512Mi\"},\"requests\":{\"cpu\":\"100m\",\"memory\":\"64Mi\"}}}},{\"name\":\"entrypoint\",\"inputs\":{},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"root-driver\",\"template\":\"system-dag-driver\",\"arguments\":{\"parameters\":[{\"name\":\"component\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/components-root}}\"},{\"name\":\"runtime-config\",\"value\":\"{}\"},{\"name\":\"driver-type\",\"value\":\"ROOT_DAG\"}]}},{\"name\":\"root\",\"template\":\"root\",\"arguments\":{\"parameters\":[{\"name\":\"parent-dag-id\",\"value\":\"{{tasks.root-driver.outputs.parameters.execution-id}}\"},{\"name\":\"condition\",\"value\":\"\"}]},\"depends\":\"root-driver.Succeeded\"}]}}],\"entrypoint\":\"entrypoint\",\"arguments\":{},\"serviceAccountName\":\"pipeline-runner\",\"podMetadata\":{\"annotations\":{\"pipelines.kubeflow.org/v2_component\":\"true\"},\"labels\":{\"pipelines.kubeflow.org/v2_component\":\"true\"}}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}"
var ExpectedWorkflowSpecV2 = "{\"kind\":\"Workflow\",\"apiVersion\":\"argoproj.io/v1alpha1\",\"metadata\":{\"generateName\":\"hello-world-\",\"creationTimestamp\":null,\"annotations\":{\"pipelines.kubeflow.org/components-comp-hello-world\":\"{\\\"executorLabel\\\":\\\"exec-hello-world\\\",\\\"inputDefinitions\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"type\\\":\\\"STRING\\\"}}}}\",\"pipelines.kubeflow.org/components-root\":\"{\\\"dag\\\":{\\\"tasks\\\":{\\\"hello-world\\\":{\\\"cachingOptions\\\":{\\\"enableCache\\\":true},\\\"componentRef\\\":{\\\"name\\\":\\\"comp-hello-world\\\"},\\\"inputs\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"componentInputParameter\\\":\\\"text\\\"}}},\\\"taskInfo\\\":{\\\"name\\\":\\\"hello-world\\\"}}}},\\\"inputDefinitions\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"type\\\":\\\"STRING\\\"}}}}\",\"pipelines.kubeflow.org/implementations-comp-hello-world\":\"{\\\"args\\\":[\\\"--text\\\",\\\"{{$.inputs.parameters['text']}}\\\"],\\\"command\\\":[\\\"sh\\\",\\\"-ec\\\",\\\"program_path=$(mktemp)\\\\nprintf \\\\\\\"%s\\\\\\\" \\\\\\\"$0\\\\\\\" \\\\u003e \\\\\\\"$program_path\\\\\\\"\\\\npython3 -u \\\\\\\"$program_path\\\\\\\" \\\\\\\"$@\\\\\\\"\\\\n\\\",\\\"def hello_world(text):\\\\n    print(text)\\\\n    return text\\\\n\\\\nimport argparse\\\\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\\\\n_parser.add_argument(\\\\\\\"--text\\\\\\\", dest=\\\\\\\"text\\\\\\\", type=str, required=True, default=argparse.SUPPRESS)\\\\n_parsed_args = vars(_parser.parse_args())\\\\n\\\\n_outputs = hello_world(**_parsed_args)\\\\n\\\"],\\\"image\\\":\\\"python:3.7\\\"}\"}},\"spec\":{\"templates\":[{\"name\":\"system-container-driver\",\"inputs\":{\"parameters\":[{\"name\":\"component\"},{\"name\":\"task\"},{\"name\":\"container\"},{\"name\":\"parent-dag-id\"},{\"name\":\"iteration-index\",\"default\":\"-1\"}]},\"outputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"valueFrom\":{\"path\":\"/tmp/outputs/pod-spec-patch\",\"default\":\"\"}},{\"name\":\"cached-decision\",\"default\":\"false\",\"valueFrom\":{\"path\":\"/tmp/outputs/cached-decision\",\"default\":\"false\"}},{\"name\":\"condition\",\"valueFrom\":{\"path\":\"/tmp/outputs/condition\",\"default\":\"true\"}}]},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-driver@sha256:a2efa29022573d9bcc92ee0a843ac37bef20333d3e6b8d9d7fbe97cbd346d84c\",\"command\":[\"driver\"],\"args\":[\"--type\",\"CONTAINER\",\"--pipeline_name\",\"namespace/n1/pipeline/hello-world\",\"--run_id\",\"{{workflow.uid}}\",\"--dag_execution_id\",\"{{inputs.parameters.parent-dag-id}}\",\"--component\",\"{{inputs.parameters.component}}\",\"--task\",\"{{inputs.parameters.task}}\",\"--container\",\"{{inputs.parameters.container}}\",\"--iteration_index\",\"{{inputs.parameters.iteration-index}}\",\"--cached_decision_path\",\"{{outputs.parameters.cached-decision.path}}\",\"--pod_spec_patch_path\",\"{{outputs.parameters.pod-spec-patch.path}}\",\"--condition_path\",\"{{outputs.parameters.condition.path}}\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"512Mi\"},\"requests\":{\"cpu\":\"100m\",\"memory\":\"64Mi\"}}}},{\"name\":\"system-container-executor\",\"inputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\"},{\"name\":\"cached-decision\",\"default\":\"false\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"executor\",\"template\":\"system-container-impl\",\"arguments\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"value\":\"{{inputs.parameters.pod-spec-patch}}\"}]},\"when\":\"{{inputs.parameters.cached-decision}} != true\"}]}},{\"name\":\"system-container-impl\",\"inputs\":{\"parameters\":[{\"name\":\"pod-spec-patch\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline/should-be-overridden-during-runtime\",\"command\":[\"should-be-overridden-during-runtime\"],\"envFrom\":[{\"configMapRef\":{\"name\":\"metadata-grpc-configmap\",\"optional\":true}}],\"env\":[{\"name\":\"KFP_POD_NAME\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.name\"}}},{\"name\":\"KFP_POD_UID\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.uid\"}}}],\"resources\":{},\"volumeMounts\":[{\"name\":\"kfp-launcher\",\"mountPath\":\"/kfp-launcher\"}]},\"volumes\":[{\"name\":\"kfp-launcher\",\"emptyDir\":{}}],\"initContainers\":[{\"name\":\"kfp-launcher\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-launcher-v2@sha256:4513cf5c10c252d94f383ce51a890514799c200795e3de5e90f91b98b2e2f959\",\"command\":[\"launcher-v2\",\"--copy\",\"/kfp-launcher/launch\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"128Mi\"},\"requests\":{\"cpu\":\"100m\"}},\"volumeMounts\":[{\"name\":\"kfp-launcher\",\"mountPath\":\"/kfp-launcher\"}]}],\"podSpecPatch\":\"{{inputs.parameters.pod-spec-patch}}\"},{\"name\":\"root\",\"inputs\":{\"parameters\":[{\"name\":\"parent-dag-id\"}]},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"hello-world-driver\",\"template\":\"system-container-driver\",\"arguments\":{\"parameters\":[{\"name\":\"component\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/components-comp-hello-world}}\"},{\"name\":\"task\",\"value\":\"{\\\"cachingOptions\\\":{\\\"enableCache\\\":true},\\\"componentRef\\\":{\\\"name\\\":\\\"comp-hello-world\\\"},\\\"inputs\\\":{\\\"parameters\\\":{\\\"text\\\":{\\\"componentInputParameter\\\":\\\"text\\\"}}},\\\"taskInfo\\\":{\\\"name\\\":\\\"hello-world\\\"}}\"},{\"name\":\"container\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/implementations-comp-hello-world}}\"},{\"name\":\"parent-dag-id\",\"value\":\"{{inputs.parameters.parent-dag-id}}\"}]}},{\"name\":\"hello-world\",\"template\":\"system-container-executor\",\"arguments\":{\"parameters\":[{\"name\":\"pod-spec-patch\",\"value\":\"{{tasks.hello-world-driver.outputs.parameters.pod-spec-patch}}\"},{\"name\":\"cached-decision\",\"default\":\"false\",\"value\":\"{{tasks.hello-world-driver.outputs.parameters.cached-decision}}\"}]},\"depends\":\"hello-world-driver.Succeeded\"}]}},{\"name\":\"system-dag-driver\",\"inputs\":{\"parameters\":[{\"name\":\"component\"},{\"name\":\"runtime-config\",\"default\":\"\"},{\"name\":\"task\",\"default\":\"\"},{\"name\":\"parent-dag-id\",\"default\":\"0\"},{\"name\":\"iteration-index\",\"default\":\"-1\"},{\"name\":\"driver-type\",\"default\":\"DAG\"}]},\"outputs\":{\"parameters\":[{\"name\":\"execution-id\",\"valueFrom\":{\"path\":\"/tmp/outputs/execution-id\"}},{\"name\":\"iteration-count\",\"valueFrom\":{\"path\":\"/tmp/outputs/iteration-count\",\"default\":\"0\"}},{\"name\":\"condition\",\"valueFrom\":{\"path\":\"/tmp/outputs/condition\",\"default\":\"true\"}}]},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"container\":{\"name\":\"\",\"image\":\"gcr.io/ml-pipeline-test/dev/kfp-driver@sha256:a2efa29022573d9bcc92ee0a843ac37bef20333d3e6b8d9d7fbe97cbd346d84c\",\"command\":[\"driver\"],\"args\":[\"--type\",\"{{inputs.parameters.driver-type}}\",\"--pipeline_name\",\"namespace/n1/pipeline/hello-world\",\"--run_id\",\"{{workflow.uid}}\",\"--dag_execution_id\",\"{{inputs.parameters.parent-dag-id}}\",\"--component\",\"{{inputs.parameters.component}}\",\"--task\",\"{{inputs.parameters.task}}\",\"--runtime_config\",\"{{inputs.parameters.runtime-config}}\",\"--iteration_index\",\"{{inputs.parameters.iteration-index}}\",\"--execution_id_path\",\"{{outputs.parameters.execution-id.path}}\",\"--iteration_count_path\",\"{{outputs.parameters.iteration-count.path}}\",\"--condition_path\",\"{{outputs.parameters.condition.path}}\"],\"resources\":{\"limits\":{\"cpu\":\"500m\",\"memory\":\"512Mi\"},\"requests\":{\"cpu\":\"100m\",\"memory\":\"64Mi\"}}}},{\"name\":\"entrypoint\",\"inputs\":{},\"outputs\":{},\"metadata\":{\"annotations\":{\"sidecar.istio.io/inject\":\"false\"}},\"dag\":{\"tasks\":[{\"name\":\"root-driver\",\"template\":\"system-dag-driver\",\"arguments\":{\"parameters\":[{\"name\":\"component\",\"value\":\"{{workflow.annotations.pipelines.kubeflow.org/components-root}}\"},{\"name\":\"runtime-config\",\"value\":\"{\\\"parameterValues\\\":{\\\"param2\\\":\\\"world\\\"}}\"},{\"name\":\"driver-type\",\"value\":\"ROOT_DAG\"}]}},{\"name\":\"root\",\"template\":\"root\",\"arguments\":{\"parameters\":[{\"name\":\"parent-dag-id\",\"value\":\"{{tasks.root-driver.outputs.parameters.execution-id}}\"},{\"name\":\"condition\",\"value\":\"\"}]},\"depends\":\"root-driver.Succeeded\"}]}}],\"entrypoint\":\"entrypoint\",\"arguments\":{},\"serviceAccountName\":\"pipeline-runner\",\"podMetadata\":{\"annotations\":{\"pipelines.kubeflow.org/v2_component\":\"true\"},\"labels\":{\"pipelines.kubeflow.org/v2_component\":\"true\"}}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}"

func TestToSwfCRDResourceGeneratedName_SpecialCharsAndSpace(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("! HaVe ä £unky name")
	assert.Nil(t, err)
	assert.Equal(t, name, "haveunkyname")
}

func TestToSwfCRDResourceGeneratedName_TruncateLongName(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("AloooooooooooooooooongName")
	assert.Nil(t, err)
	assert.Equal(t, name, "aloooooooooooooooooongnam")
}

func TestToSwfCRDResourceGeneratedName_EmptyName(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("")
	assert.Nil(t, err)
	assert.Equal(t, name, "job-")
}

func TestToCrdParametersV1(t *testing.T) {
	assert.Equal(t,
		toCRDParametersV1([]*apiv1beta1.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}}),
		[]scheduledworkflow.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}})
}

func TestToCrdCronScheduleV1(t *testing.T) {
	actualCronSchedule := toCRDCronScheduleV1(&apiv1beta1.CronSchedule{
		Cron:      "123",
		StartTime: &timestamp.Timestamp{Seconds: 123},
		EndTime:   &timestamp.Timestamp{Seconds: 456},
	})
	startTime := metav1.NewTime(time.Unix(123, 0))
	endTime := metav1.NewTime(time.Unix(456, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:      "123",
		StartTime: &startTime,
		EndTime:   &endTime,
	})
}

func TestToCrdCronScheduleV1_NilCron(t *testing.T) {
	actualCronSchedule := toCRDCronScheduleV1(&apiv1beta1.CronSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 123},
		EndTime:   &timestamp.Timestamp{Seconds: 456},
	})
	assert.Nil(t, actualCronSchedule)
}

func TestToCrdCronScheduleV1_NilStartTime(t *testing.T) {
	actualCronSchedule := toCRDCronScheduleV1(&apiv1beta1.CronSchedule{
		Cron:    "123",
		EndTime: &timestamp.Timestamp{Seconds: 456},
	})
	endTime := metav1.NewTime(time.Unix(456, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:    "123",
		EndTime: &endTime,
	})
}

func TestToCrdCronScheduleV1_NilEndTime(t *testing.T) {
	actualCronSchedule := toCRDCronScheduleV1(&apiv1beta1.CronSchedule{
		Cron:      "123",
		StartTime: &timestamp.Timestamp{Seconds: 123},
	})
	startTime := metav1.NewTime(time.Unix(123, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:      "123",
		StartTime: &startTime,
	})
}

func TestToCrdPeriodicScheduleV1(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicScheduleV1(&apiv1beta1.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &timestamp.Timestamp{Seconds: 1},
		EndTime:        &timestamp.Timestamp{Seconds: 2},
	})
	startTime := metav1.NewTime(time.Unix(1, 0))
	endTime := metav1.NewTime(time.Unix(2, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &startTime,
		EndTime:        &endTime,
	})
}

func TestToCrdPeriodicScheduleV1_NilInterval(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicScheduleV1(&apiv1beta1.PeriodicSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 1},
		EndTime:   &timestamp.Timestamp{Seconds: 2},
	})
	assert.Nil(t, actualPeriodicSchedule)
}

func TestToCrdPeriodicScheduleV1_NilStartTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicScheduleV1(&apiv1beta1.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &timestamp.Timestamp{Seconds: 2},
	})
	endTime := metav1.NewTime(time.Unix(2, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &endTime,
	})
}

func TestToCrdPeriodicScheduleV1_NilEndTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicScheduleV1(&apiv1beta1.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &timestamp.Timestamp{Seconds: 1},
	})
	startTime := metav1.NewTime(time.Unix(1, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &startTime,
	})
}

func TestScheduledWorkflow_V1(t *testing.T) {
	v1Template, _ := New([]byte(v2SpecHelloWorldYAML))

	apiJob := &apiv1beta1.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		NoCatchup:      true,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"}, Relationship: apiv1beta1.Relationship_OWNER},
		},
		PipelineSpec: &apiv1beta1.PipelineSpec{PipelineId: "pipeline1", Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}}},
	}

	expectedScheduledWorkflow := scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "name1"},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: &apiJob.MaxConcurrency,
			Trigger: scheduledworkflow.Trigger{
				CronSchedule: &scheduledworkflow.CronSchedule{
					Cron:      "1 * * * *",
					StartTime: &metav1.Time{Time: time.Unix(1, 0)},
				},
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: []scheduledworkflow.Parameter{{Name: "param2", Value: "world"}},
				Spec:       WorkflowSpecV1,
			},
			NoCatchup: &[]bool{true}[0], // returns a pointer to true
		}}

	actualScheduledWorkflow, _ := v1Template.ScheduledWorkflow(apiJob)

	assert.Equal(t, &expectedScheduledWorkflow, actualScheduledWorkflow)
}

func TestScheduledWorkflow(t *testing.T) {
	v2Template, _ := New([]byte(v2SpecHelloWorldYAML))

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "name1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		NoCatchup:      true,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ExperimentId:   "exp1",
		PipelineSource: &apiv2beta1.RecurringRun_PipelineId{"pipeline1"},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters: map[string]*structpb.Value{"param2": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}}},
		},
	}

	expectedScheduledWorkflow := scheduledworkflow.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "name1"},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: &apiRecurringRun.MaxConcurrency,
			Trigger: scheduledworkflow.Trigger{
				CronSchedule: &scheduledworkflow.CronSchedule{
					Cron:      "1 * * * *",
					StartTime: &metav1.Time{Time: time.Unix(1, 0)},
				},
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: []scheduledworkflow.Parameter{{Name: "param2", Value: "world"}},
				Spec:       ExpectedWorkflowSpecV2,
			},
			NoCatchup: &[]bool{true}[0], // returns a pointer to true
		}}

	actualScheduledWorkflow, _ := v2Template.ScheduledWorkflow(apiRecurringRun)

	assert.Equal(t, &expectedScheduledWorkflow, actualScheduledWorkflow)
}
