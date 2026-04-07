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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	structpb "google.golang.org/protobuf/types/known/structpb"
	goyaml "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
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
		template:     awfTemplate,
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

var awfTemplate = `
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

var defaultPVC = &corev1.PersistentVolumeClaimSpec{
	AccessModes: []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany,
	},
	StorageClassName: util.StringPointer("my-storage"),
}

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
	proxy.InitializeConfigWithEmptyForTests()

	v2SpecHelloWorldYAML := loadYaml(t, "testdata/hello_world.yaml")
	v2Template, _ := New([]byte(v2SpecHelloWorldYAML), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})

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
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorldYAML),
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
			PipelineId:     "1",
			PipelineName:   "pipeline name",
			NoCatchup:      util.BoolPointer(true),
			ServiceAccount: "pipeline-runner",
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
	res, err := os.ReadFile(path)
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
		spec:            &expectedSpec,
		templateOptions: TemplateOptions{DefaultWorkspace: defaultPVC},
	}
	templateV2Spec, err := New([]byte(template), TemplateOptions{DefaultWorkspace: defaultPVC})
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
		spec:            &expectedPipelineSpec,
		platformSpec:    &expectedPlatformSpec,
		templateOptions: TemplateOptions{DefaultWorkspace: defaultPVC},
	}
	templateV2Spec, err := New([]byte(template), TemplateOptions{DefaultWorkspace: defaultPVC})
	assert.Nil(t, err)
	assert.Equal(t, expectedTemplate, templateV2Spec)
}

func TestNewTemplate_V2_InvalidSchemaVersion(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world_schema_2_0_0.yaml")
	_, err := New([]byte(template), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "KFP only supports schema version 2.1.0")
}

// Verify that the V2Spec object created from Bytes() method is the same as the original object.
// The byte slice may be slightly different from the original input during the conversion,
// so we verify the parsed object.
func TestBytes_V2_WithExecutorConfig(t *testing.T) {
	template := loadYaml(t, "testdata/pipeline_with_volume.yaml")
	templateV2Spec, _ := New([]byte(template), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	templateBytes := templateV2Spec.Bytes()
	newTemplateV2Spec, err := New(templateBytes, TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	assert.Nil(t, err)
	assert.Equal(t, templateV2Spec, newTemplateV2Spec)
}

// Verify that the V2Spec object created from Bytes() method is the same as the original object.
// The byte slice may be slightly different from the original input during the conversion,
// so we verify the parsed object.
func TestBytes_V2(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	templateV2Spec, _ := New([]byte(template), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	templateBytes := templateV2Spec.Bytes()
	newTemplateV2Spec, err := New(templateBytes, TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	assert.Nil(t, err)
	assert.Equal(t, templateV2Spec, newTemplateV2Spec)
}

// --- modelToPipelineJobRuntimeConfig tests ---

func TestModelToPipelineJobRuntimeConfig_Nil(t *testing.T) {
	result, err := modelToPipelineJobRuntimeConfig(nil)
	assert.Nil(t, err)
	assert.Nil(t, result)
}

func TestModelToPipelineJobRuntimeConfig_EmptyParams(t *testing.T) {
	rc := &model.RuntimeConfig{Parameters: "", PipelineRoot: "gs://bucket/root"}
	result, err := modelToPipelineJobRuntimeConfig(rc)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "gs://bucket/root", result.GcsOutputDirectory)
	assert.Empty(t, result.ParameterValues)
}

func TestModelToPipelineJobRuntimeConfig_WithParams(t *testing.T) {
	rc := &model.RuntimeConfig{
		Parameters:   `{"param1":"hello","param2":"world"}`,
		PipelineRoot: "gs://my-bucket",
	}
	result, err := modelToPipelineJobRuntimeConfig(rc)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "gs://my-bucket", result.GcsOutputDirectory)
	assert.Equal(t, "hello", result.ParameterValues["param1"].GetStringValue())
	assert.Equal(t, "world", result.ParameterValues["param2"].GetStringValue())
}

func TestModelToPipelineJobRuntimeConfig_InvalidJSON(t *testing.T) {
	rc := &model.RuntimeConfig{Parameters: "not json"}
	_, err := modelToPipelineJobRuntimeConfig(rc)
	assert.NotNil(t, err)
}

// --- StringMapToCRDParameters tests ---

func TestStringMapToCRDParameters_Empty(t *testing.T) {
	result, err := StringMapToCRDParameters("")
	assert.Nil(t, err)
	assert.Empty(t, result)
}

func TestStringMapToCRDParameters_ValidParams(t *testing.T) {
	result, err := StringMapToCRDParameters(`{"x":"hello","y":"world"}`)
	assert.Nil(t, err)
	assert.Len(t, result, 2)
	paramMap := make(map[string]string)
	for _, p := range result {
		paramMap[p.Name] = p.Value
	}
	assert.Equal(t, `"hello"`, paramMap["x"])
	assert.Equal(t, `"world"`, paramMap["y"])
}

func TestStringMapToCRDParameters_InvalidJSON(t *testing.T) {
	_, err := StringMapToCRDParameters("bad json")
	assert.NotNil(t, err)
}

// --- stringArrayToCRDParameters tests ---

func TestStringArrayToCRDParameters_Empty(t *testing.T) {
	result, err := stringArrayToCRDParameters("")
	assert.Nil(t, err)
	assert.Empty(t, result)
}

func TestStringArrayToCRDParameters_ValidParams(t *testing.T) {
	input := `[{"name":"param1","value":"val1"},{"name":"param2","value":"val2"}]`
	result, err := stringArrayToCRDParameters(input)
	assert.Nil(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "param1", result[0].Name)
	assert.Equal(t, "val1", result[0].Value)
	assert.Equal(t, "param2", result[1].Name)
	assert.Equal(t, "val2", result[1].Value)
}

func TestStringArrayToCRDParameters_InvalidJSON(t *testing.T) {
	_, err := stringArrayToCRDParameters("not json")
	assert.NotNil(t, err)
}

// --- modelToParametersMap tests ---

func TestModelToParametersMap_Empty(t *testing.T) {
	result, err := modelToParametersMap("")
	assert.Nil(t, err)
	assert.Empty(t, result)
}

func TestModelToParametersMap_ValidParams(t *testing.T) {
	input := `[{"name":"param1","value":"val1"},{"name":"param2","value":"val2"}]`
	result, err := modelToParametersMap(input)
	assert.Nil(t, err)
	assert.Equal(t, "val1", result["param1"])
	assert.Equal(t, "val2", result["param2"])
}

func TestModelToParametersMap_InvalidJSON(t *testing.T) {
	_, err := modelToParametersMap("{bad")
	assert.NotNil(t, err)
}

// --- modelToCRDTrigger periodic schedule tests ---

func TestModelToCRDTrigger_Periodic(t *testing.T) {
	trigger := model.Trigger{
		PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: util.Int64Pointer(100),
			PeriodicScheduleEndTimeInSec:   util.Int64Pointer(200),
			IntervalSecond:                 util.Int64Pointer(60),
		},
	}
	result, err := modelToCRDTrigger(trigger)
	assert.Nil(t, err)
	assert.Nil(t, result.CronSchedule)
	assert.NotNil(t, result.PeriodicSchedule)
	assert.Equal(t, int64(60), result.PeriodicSchedule.IntervalSecond)
	assert.Equal(t, time.Unix(100, 0), result.PeriodicSchedule.StartTime.Time)
	assert.Equal(t, time.Unix(200, 0), result.PeriodicSchedule.EndTime.Time)
}

func TestModelToCRDTrigger_Empty(t *testing.T) {
	trigger := model.Trigger{}
	result, err := modelToCRDTrigger(trigger)
	assert.Nil(t, err)
	assert.Nil(t, result.CronSchedule)
	assert.Nil(t, result.PeriodicSchedule)
}

// --- NewArgoTemplate / NewArgoTemplateFromWorkflow tests ---

func TestNewArgoTemplate_Valid(t *testing.T) {
	tmpl, err := NewArgoTemplate([]byte(awfTemplate))
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, V1, tmpl.GetTemplateType())
	assert.False(t, tmpl.IsCacheDisabled())
}

func TestNewArgoTemplate_InvalidYAML(t *testing.T) {
	_, err := NewArgoTemplate([]byte("not yaml: ["))
	assert.NotNil(t, err)
}

func TestNewArgoTemplateFromWorkflow(t *testing.T) {
	wf := unmarshalWf(awfTemplate)
	tmpl, err := NewArgoTemplateFromWorkflow(wf)
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, V1, tmpl.GetTemplateType())
}

// --- Argo simple method tests ---

func TestArgo_Bytes(t *testing.T) {
	tmpl, _ := NewArgoTemplate([]byte(awfTemplate))
	result := tmpl.Bytes()
	assert.NotEmpty(t, result)
	assert.Contains(t, string(result), "whalesay")
}

func TestArgo_Bytes_Nil(t *testing.T) {
	var tmpl *Argo
	assert.Nil(t, tmpl.Bytes())
}

func TestArgo_IsV2(t *testing.T) {
	tmpl, _ := NewArgoTemplate([]byte(awfTemplate))
	assert.False(t, tmpl.IsV2())
}

func TestArgo_IsV2_Nil(t *testing.T) {
	var tmpl *Argo
	assert.False(t, tmpl.IsV2())
}

func TestArgo_V2PipelineName_Empty(t *testing.T) {
	tmpl, _ := NewArgoTemplate([]byte(awfTemplate))
	assert.Empty(t, tmpl.V2PipelineName())
}

func TestArgo_V2PipelineName_Nil(t *testing.T) {
	var tmpl *Argo
	assert.Empty(t, tmpl.V2PipelineName())
}

func TestArgo_ParametersJSON(t *testing.T) {
	tmpl, _ := NewArgoTemplate([]byte(awfTemplate))
	result, err := tmpl.ParametersJSON()
	assert.Nil(t, err)
	assert.NotEmpty(t, result)
}

func TestArgo_ParametersJSON_Nil(t *testing.T) {
	var tmpl *Argo
	result, err := tmpl.ParametersJSON()
	assert.Nil(t, err)
	assert.Empty(t, result)
}

func TestArgo_OverrideV2PipelineName_NonV2(t *testing.T) {
	tmpl, _ := NewArgoTemplate([]byte(awfTemplate))
	// Should be a no-op for non-V2 workflows.
	tmpl.OverrideV2PipelineName("my-pipeline", "my-ns")
	assert.Empty(t, tmpl.V2PipelineName())
}

func TestArgo_OverrideV2PipelineName_Nil(t *testing.T) {
	var tmpl *Argo
	tmpl.OverrideV2PipelineName("my-pipeline", "my-ns") // should not panic
}

// --- V2Spec simple method tests ---

func TestV2Spec_GetTemplateType(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, err := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.Nil(t, err)
	assert.Equal(t, V2, tmpl.GetTemplateType())
}

func TestV2Spec_IsV2(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.True(t, tmpl.IsV2())
}

func TestV2Spec_V2PipelineName(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.Equal(t, "namespace/n1/pipeline/hello-world", tmpl.V2PipelineName())
}

func TestV2Spec_V2PipelineName_Nil(t *testing.T) {
	var tmpl *V2Spec
	assert.Empty(t, tmpl.V2PipelineName())
}

func TestV2Spec_OverrideV2PipelineName_WithNamespace(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	tmpl.OverrideV2PipelineName("my-pipeline", "my-ns")
	assert.Equal(t, "namespace/my-ns/pipeline/my-pipeline", tmpl.V2PipelineName())
}

func TestV2Spec_OverrideV2PipelineName_WithoutNamespace(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	tmpl.OverrideV2PipelineName("my-pipeline", "")
	assert.Equal(t, "pipeline/my-pipeline", tmpl.V2PipelineName())
}

func TestV2Spec_OverrideV2PipelineName_Nil(t *testing.T) {
	var tmpl *V2Spec
	tmpl.OverrideV2PipelineName("my-pipeline", "ns") // should not panic
}

func TestV2Spec_ParametersJSON(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	result, err := tmpl.ParametersJSON()
	assert.Nil(t, err)
	assert.Equal(t, "[]", result)
}

func TestV2Spec_IsCacheDisabled(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmplEnabled, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{CacheDisabled: false})
	assert.False(t, tmplEnabled.IsCacheDisabled())

	tmplDisabled, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{CacheDisabled: true})
	assert.True(t, tmplDisabled.IsCacheDisabled())
}

func TestV2Spec_PipelineSpec(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.NotNil(t, tmpl.PipelineSpec())
	assert.Equal(t, "namespace/n1/pipeline/hello-world", tmpl.PipelineSpec().GetPipelineInfo().GetName())
}

func TestV2Spec_PlatformSpec_Nil(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.Nil(t, tmpl.PlatformSpec())
}

func TestV2Spec_PlatformSpec_Present(t *testing.T) {
	template := loadYaml(t, "testdata/pipeline_with_volume.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	assert.NotNil(t, tmpl.PlatformSpec())
	_, hasKubernetes := tmpl.PlatformSpec().Platforms["kubernetes"]
	assert.True(t, hasKubernetes)
}

func TestV2Spec_Bytes_Nil(t *testing.T) {
	var tmpl *V2Spec
	assert.Nil(t, tmpl.Bytes())
}

// --- NewV2SpecTemplate validation tests ---

func TestNewV2SpecTemplate_EmptyInput(t *testing.T) {
	_, err := NewV2SpecTemplate([]byte(""), TemplateOptions{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no pipeline spec is provided")
}

func TestNewV2SpecTemplate_InvalidYAML(t *testing.T) {
	_, err := NewV2SpecTemplate([]byte("not: [valid yaml"), TemplateOptions{})
	assert.NotNil(t, err)
}

// --- AddRuntimeMetadata tests ---

func TestAddRuntimeMetadata(t *testing.T) {
	wf := unmarshalWf(awfTemplate)
	AddRuntimeMetadata(wf)
	template := wf.Spec.Templates[0]
	assert.Equal(t, "false", template.Metadata.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "true", template.Metadata.Labels["pipelines.kubeflow.org/cache_enabled"])
	assert.NotEmpty(t, template.Metadata.Labels[util.LabelKeyWorkflowRunId])
}

// --- New factory error test ---

func TestNew_UnknownFormat(t *testing.T) {
	_, err := New([]byte("not a valid template"), TemplateOptions{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown template format")
}

// --- setDefaultServiceAccount tests ---

func TestSetDefaultServiceAccount_ExplicitAccount(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	wf := util.NewWorkflow(unmarshalWf(awfTemplate))
	setDefaultServiceAccount(wf, "my-sa")
	assert.Equal(t, "my-sa", wf.ServiceAccount())
}

func TestSetDefaultServiceAccount_EmptyFallsToConfig(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	wf := util.NewWorkflow(unmarshalWf(awfTemplate))
	setDefaultServiceAccount(wf, "")
	// When no SA is set on workflow and empty is passed, it falls back to config default.
	assert.Equal(t, common.DefaultPipelineRunnerServiceAccount, wf.ServiceAccount())
}

// --- validatePipelineJobInputs tests ---

func TestValidatePipelineJobInputs_RequiredParamMissing(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	job := &pipelinespec.PipelineJob{}
	// hello_world has root input "y" (STRING), and it is required because it has
	// neither an optional flag nor a default value. With nil runtime config, this should error.
	err := tmpl.validatePipelineJobInputs(job)
	assert.NotNil(t, err)
}

func TestValidatePipelineJobInputs_CorrectType(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"y": structpb.NewStringValue("hello"),
			},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.Nil(t, err)
}

func TestValidatePipelineJobInputs_WrongType(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"y": structpb.NewNumberValue(42),
			},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "requires type string")
}

func TestValidatePipelineJobInputs_ExtraParams(t *testing.T) {
	template := loadYaml(t, "testdata/hello_world.yaml")
	tmpl, _ := NewV2SpecTemplate([]byte(template), TemplateOptions{})
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"y":     structpb.NewStringValue("hello"),
				"extra": structpb.NewStringValue("not expected"),
			},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not required by pipeline")
}

func TestValidatePipelineJobInputs_AllParameterTypes(t *testing.T) {
	// Build a V2Spec with multiple parameter types to cover all type-check branches.
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:  &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		SchemaVersion: SCHEMA_VERSION_2_1_0,
		Root: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"num":    {ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE},
					"flag":   {ParameterType: pipelinespec.ParameterType_BOOLEAN},
					"items":  {ParameterType: pipelinespec.ParameterType_LIST},
					"config": {ParameterType: pipelinespec.ParameterType_STRUCT},
				},
			},
		},
	}
	tmpl := &V2Spec{spec: spec}

	t.Run("all correct types", func(t *testing.T) {
		job := &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"num":    structpb.NewNumberValue(3.14),
					"flag":   structpb.NewBoolValue(true),
					"items":  {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{}}},
					"config": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{}}},
				},
			},
		}
		assert.Nil(t, tmpl.validatePipelineJobInputs(job))
	})

	t.Run("number param gets string", func(t *testing.T) {
		job := &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"num":    structpb.NewStringValue("not a number"),
					"flag":   structpb.NewBoolValue(true),
					"items":  {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{}}},
					"config": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{}}},
				},
			},
		}
		err := tmpl.validatePipelineJobInputs(job)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "requires type double or integer")
	})

	t.Run("bool param gets number", func(t *testing.T) {
		job := &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"num":    structpb.NewNumberValue(1),
					"flag":   structpb.NewNumberValue(1),
					"items":  {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{}}},
					"config": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{}}},
				},
			},
		}
		err := tmpl.validatePipelineJobInputs(job)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "requires type bool")
	})

	t.Run("list param gets string", func(t *testing.T) {
		job := &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"num":    structpb.NewNumberValue(1),
					"flag":   structpb.NewBoolValue(true),
					"items":  structpb.NewStringValue("not a list"),
					"config": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{}}},
				},
			},
		}
		err := tmpl.validatePipelineJobInputs(job)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "requires type list")
	})

	t.Run("struct param gets string", func(t *testing.T) {
		job := &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"num":    structpb.NewNumberValue(1),
					"flag":   structpb.NewBoolValue(true),
					"items":  {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{}}},
					"config": structpb.NewStringValue("not a struct"),
				},
			},
		}
		err := tmpl.validatePipelineJobInputs(job)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "requires type struct")
	})
}

func TestValidatePipelineJobInputs_UnspecifiedType(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:  &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		SchemaVersion: SCHEMA_VERSION_2_1_0,
		Root: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"bad": {ParameterType: pipelinespec.ParameterType_PARAMETER_TYPE_ENUM_UNSPECIFIED},
				},
			},
		},
	}
	tmpl := &V2Spec{spec: spec}
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"bad": structpb.NewStringValue("anything"),
			},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has unspecified type")
}

func TestValidatePipelineJobInputs_TaskFinalStatus(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:  &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		SchemaVersion: SCHEMA_VERSION_2_1_0,
		Root: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"status": {ParameterType: pipelinespec.ParameterType_TASK_FINAL_STATUS},
				},
			},
		},
	}
	tmpl := &V2Spec{spec: spec}
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"status": structpb.NewStringValue("anything"),
			},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TASK_FINAL_STATUS")
}

func TestValidatePipelineJobInputs_OptionalParamNotProvided(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:  &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		SchemaVersion: SCHEMA_VERSION_2_1_0,
		Root: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"opt": {ParameterType: pipelinespec.ParameterType_STRING, IsOptional: true},
				},
			},
		},
	}
	tmpl := &V2Spec{spec: spec}
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{},
		},
	}
	err := tmpl.validatePipelineJobInputs(job)
	assert.Nil(t, err)
}

func TestValidatePipelineJobInputs_NoInputsNoParams(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:  &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		SchemaVersion: SCHEMA_VERSION_2_1_0,
		Root:          &pipelinespec.ComponentSpec{},
	}
	tmpl := &V2Spec{spec: spec}
	job := &pipelinespec.PipelineJob{}
	assert.Nil(t, tmpl.validatePipelineJobInputs(job))
}

// --- NewGenericScheduledWorkflow tests ---

func TestNewGenericScheduledWorkflow(t *testing.T) {
	modelJob := &model.Job{
		K8SName:        "my-job",
		Enabled:        true,
		MaxConcurrency: 3,
		NoCatchup:      false,
		ExperimentId:   "exp-1",
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "pipe-1",
			PipelineName: "My Pipeline",
		},
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(120),
			},
		},
	}
	swf, err := NewGenericScheduledWorkflow(modelJob)
	assert.Nil(t, err)
	assert.Equal(t, "kubeflow.org/v2beta1", swf.APIVersion)
	assert.Equal(t, "ScheduledWorkflow", swf.Kind)
	assert.Equal(t, "my-job", swf.GenerateName)
	assert.True(t, swf.Spec.Enabled)
	assert.Equal(t, int64(3), *swf.Spec.MaxConcurrency)
	assert.Equal(t, "exp-1", swf.Spec.ExperimentId)
	assert.Equal(t, "pipe-1", swf.Spec.PipelineId)
	assert.Equal(t, "My Pipeline", swf.Spec.PipelineName)
	assert.NotNil(t, swf.Spec.PeriodicSchedule)
	assert.Equal(t, int64(120), swf.Spec.PeriodicSchedule.IntervalSecond)
}
