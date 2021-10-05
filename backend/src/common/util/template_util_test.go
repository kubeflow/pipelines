// Copyright 2018 The Kubeflow Authors
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

package util

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
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
	assert.Equal(t, codes.InvalidArgument, err.(*UserError).ExternalStatusCode())
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
		template:     v2SpecHelloWorld,
		templateType: V2,
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
	}}
	for _, test := range tt {
		format := InferTemplateFormat([]byte(test.template))
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

var v2SpecHelloWorld = `
{
  "components": {
    "comp-hello-world": {
      "executorLabel": "exec-hello-world",
      "inputDefinitions": {
	"parameters": {
	  "text": {
	    "type": "STRING"
	  }
	}
      }
    }
  },
  "deploymentSpec": {
    "executors": {
      "exec-hello-world": {
	"container": {
	  "args": [
	    "--text",
	    "{{$.inputs.parameters['text']}}"
	  ],
	  "command": [
	    "sh",
	    "-ec",
	    "program_path=$(mktemp)\nprintf \"%s\" \"$0\" > \"$program_path\"\npython3 -u \"$program_path\" \"$@\"\n",
	    "def hello_world(text):\n    print(text)\n    return text\n\nimport argparse\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\n_parser.add_argument(\"--text\", dest=\"text\", type=str, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = hello_world(**_parsed_args)\n"
	  ],
	  "image": "python:3.7"
	}
      }
    }
  },
  "pipelineInfo": {
    "name": "hello-world"
  },
  "root": {
    "dag": {
      "tasks": {
	"hello-world": {
	  "cachingOptions": {
	    "enableCache": true
	  },
	  "componentRef": {
	    "name": "comp-hello-world"
	  },
	  "inputs": {
	    "parameters": {
	      "text": {
		"componentInputParameter": "text"
	      }
	    }
	  },
	  "taskInfo": {
	    "name": "hello-world"
	  }
	}
      }
    },
    "inputDefinitions": {
      "parameters": {
	"text": {
	  "type": "STRING"
	}
      }
    }
  },
  "schemaVersion": "2.0.0",
  "sdkVersion": "kfp-1.6.5"
}
`
