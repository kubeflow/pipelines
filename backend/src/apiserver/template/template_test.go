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

package template

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
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
    "name": "namespace/n1/pipeline/hello-world"
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

func TestToCrdParameter(t *testing.T) {
	assert.Equal(t,
		toCRDParameter([]*api.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}}),
		[]scheduledworkflow.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}})
}

func TestToCrdCronSchedule(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
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

func TestToCrdCronSchedule_NilCron(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 123},
		EndTime:   &timestamp.Timestamp{Seconds: 456},
	})
	assert.Nil(t, actualCronSchedule)
}

func TestToCrdCronSchedule_NilStartTime(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		Cron:    "123",
		EndTime: &timestamp.Timestamp{Seconds: 456},
	})
	endTime := metav1.NewTime(time.Unix(456, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:    "123",
		EndTime: &endTime,
	})
}

func TestToCrdCronSchedule_NilEndTime(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		Cron:      "123",
		StartTime: &timestamp.Timestamp{Seconds: 123},
	})
	startTime := metav1.NewTime(time.Unix(123, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:      "123",
		StartTime: &startTime,
	})
}

func TestToCrdPeriodicSchedule(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
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

func TestToCrdPeriodicSchedule_NilInterval(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 1},
		EndTime:   &timestamp.Timestamp{Seconds: 2},
	})
	assert.Nil(t, actualPeriodicSchedule)
}

func TestToCrdPeriodicSchedule_NilStartTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &timestamp.Timestamp{Seconds: 2},
	})
	endTime := metav1.NewTime(time.Unix(2, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &endTime,
	})
}

func TestToCrdPeriodicSchedule_NilEndTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &timestamp.Timestamp{Seconds: 1},
	})
	startTime := metav1.NewTime(time.Unix(1, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &startTime,
	})
}
