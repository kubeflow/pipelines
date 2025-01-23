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
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	goyaml "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
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
		Parameters: params,
	}}}
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
		template:     loadYaml(t, "testdata/hello_world.yaml"),
		templateType: V2,
	}}

	for _, test := range tt {
		format := inferTemplateFormat([]byte(test.template))
		if format != test.templateType {
			t.Errorf("InferTemplateFormat(%s)=%q, expect %q", test.template, format, test.templateType)
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

func TestScheduledWorkflow(t *testing.T) {
	v2SpecHelloWorldYAML := loadYaml(t, "testdata/hello_world.yaml")
	v2Template, _ := New([]byte(v2SpecHelloWorldYAML))

	modelJob := &model.Job{
		K8SName:        "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		NoCatchup:      true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				CronScheduleEndTimeInSec:   util.Int64Pointer(10),
				Cron:                       util.StringPointer("1 * * * *"),
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineId:           "1",
			PipelineName:         "pipeline name",
			PipelineSpecManifest: v2SpecHelloWorldYAML,
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{\"y\":\"world\"}",
			},
		},
	}

	expectedScheduledWorkflow := scheduledworkflow.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    "name1",
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: scheduledworkflow.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: util.Int64Pointer(1),
			Trigger: scheduledworkflow.Trigger{
				CronSchedule: &scheduledworkflow.CronSchedule{
					Cron:      "1 * * * *",
					StartTime: &metav1.Time{Time: time.Unix(1, 0)},
					EndTime:   &metav1.Time{Time: time.Unix(10, 0)},
				},
			},
			Workflow: &scheduledworkflow.WorkflowResource{
				Parameters: []scheduledworkflow.Parameter{{Name: "y", Value: "\"world\""}},
				Spec:       "",
			},
			PipelineId:   "1",
			PipelineName: "pipeline name",
			NoCatchup:    util.BoolPointer(true),
		},
	}

	actualScheduledWorkflow, err := v2Template.ScheduledWorkflow(modelJob, []metav1.OwnerReference{})
	assert.Nil(t, err)

	// We don't compare this field because it changes with every driver/launcher image release.
	// It is also tested in compiler.
	actualScheduledWorkflow.Spec.Workflow.Spec = ""
	assert.Equal(t, &expectedScheduledWorkflow, actualScheduledWorkflow)
}

func TestModelToCRDTrigger_Cron(t *testing.T) {
	inputModelTrigger := model.Trigger{
		CronSchedule: model.CronSchedule{
			CronScheduleStartTimeInSec: util.Int64Pointer(1),
			CronScheduleEndTimeInSec:   util.Int64Pointer(10),
			Cron:                       util.StringPointer("1 * * * *"),
		},
	}
	expectedCRDTrigger := scheduledworkflow.Trigger{
		CronSchedule: &scheduledworkflow.CronSchedule{
			Cron:      "1 * * * *",
			StartTime: &metav1.Time{Time: time.Unix(1, 0)},
			EndTime:   &metav1.Time{Time: time.Unix(10, 0)},
		},
	}

	actualCRDTrigger, err := modelToCRDTrigger(inputModelTrigger)
	assert.Nil(t, err)
	assert.Equal(t, expectedCRDTrigger, actualCRDTrigger)
}

func loadYaml(t *testing.T, path string) string {
	res, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	return string(res)
}

func TestIsPlatformSpecWithKubernetesConfig(t *testing.T) {
	template := loadYaml(t, "testdata/pipeline_with_volume.yaml")
	splitTemplate := strings.Split(template, "\n---\n")
	assert.True(t, IsPlatformSpecWithKubernetesConfig([]byte(splitTemplate[1])))
	assert.False(t, IsPlatformSpecWithKubernetesConfig([]byte(splitTemplate[0])))
	assert.False(t, IsPlatformSpecWithKubernetesConfig([]byte(" ")))
}

func TestNewTemplate_V2(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	var yamlData map[string]interface{}
	err := goyaml.Unmarshal([]byte(template), &yamlData)
	assert.Nil(t, err)
	jsonData, err := json.Marshal(yamlData)
	assert.Nil(t, err)
	var expectedSpec pipelinespec.PipelineSpec
	err = protojson.Unmarshal(jsonData, &expectedSpec)
	assert.Nil(t, err)
	expectedTemplate := &V2Spec{
		spec: &expectedSpec,
	}
	templateV2Spec, err := New([]byte(template))
	assert.Nil(t, err)
	assert.Equal(t, expectedTemplate, templateV2Spec)
}

func TestNewTemplate_WithPlatformSpec(t *testing.T) {
	template := loadYaml(t, "testdata/pipeline_with_volume.yaml")
	var expectedPipelineSpec pipelinespec.PipelineSpec
	var expectedPlatformSpec pipelinespec.PlatformSpec

	splitTemplate := strings.Split(template, "\n---\n")
	var pipelineSpecData map[string]interface{}
	err := goyaml.Unmarshal([]byte(splitTemplate[0]), &pipelineSpecData)
	assert.Nil(t, err)
	jsonData, err := json.Marshal(pipelineSpecData)
	assert.Nil(t, err)
	protojson.Unmarshal(jsonData, &expectedPipelineSpec)

	var platformSpecData map[string]interface{}
	err = goyaml.Unmarshal([]byte(splitTemplate[1]), &platformSpecData)
	assert.Nil(t, err)
	jsonData, err = json.Marshal(platformSpecData)
	assert.Nil(t, err)
	protojson.Unmarshal(jsonData, &expectedPlatformSpec)

	expectedTemplate := &V2Spec{
		spec:         &expectedPipelineSpec,
		platformSpec: &expectedPlatformSpec,
	}
	templateV2Spec, err := New([]byte(template))
	assert.Nil(t, err)
	assert.Equal(t, expectedTemplate, templateV2Spec)
}

func TestNewTemplate_V2_InvalidSchemaVersion(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world_schema_2_0_0.yaml")
	_, err := New([]byte(template))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "KFP only supports schema version 2.1.0")
}

// Verify that the V2Spec object created from Bytes() method is the same as the original object.
// The byte slice may be slightly different from the original input during the conversion,
// so we verify the parsed object.
func TestBytes_V2_WithExecutorConfig(t *testing.T) {
	template := loadYaml(t, "testdata/pipeline_with_volume.yaml")
	templateV2Spec, _ := New([]byte(template))
	templateBytes := templateV2Spec.Bytes()
	newTemplateV2Spec, err := New(templateBytes)
	assert.Nil(t, err)
	assert.Equal(t, templateV2Spec, newTemplateV2Spec)
}

// Verify that the V2Spec object created from Bytes() method is the same as the original object.
// The byte slice may be slightly different from the original input during the conversion,
// so we verify the parsed object.
func TestBytes_V2(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	templateV2Spec, _ := New([]byte(template))
	templateBytes := templateV2Spec.Bytes()
	newTemplateV2Spec, err := New(templateBytes)
	assert.Nil(t, err)
	assert.Equal(t, templateV2Spec, newTemplateV2Spec)
}
