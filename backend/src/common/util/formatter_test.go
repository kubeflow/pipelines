// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultUUID = "123e4567-e89b-12d3-a456-426655440000"
)

func getDefaultCreatedAtSec() int64 {
	return time.Date(2018, 8, 7, 6, 5, 4, 0, time.UTC).Unix()
}

func getDefaultScheduledAtSec() int64 {
	return time.Date(2017, 7, 6, 5, 4, 3, 0, time.UTC).Unix()
}

func TestCreateSubstitute(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	// Note: The time format constants for GO are described here:
	// https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format/20234207#20234207

	result, err := formatter.createSubtitute("[[uuid]]")
	assert.Nil(t, err)
	assert.Equal(t, defaultUUID, result)

	result, err = formatter.createSubtitute("[[schedule]]")
	assert.Nil(t, err)
	assert.Equal(t, "20170706050403", result)

	result, err = formatter.createSubtitute("[[now]]")
	assert.Nil(t, err)
	assert.Equal(t, "20180807060504", result)

	result, err = formatter.createSubtitute("[[now.2006-01-02T15-04-05]]")
	assert.Nil(t, err)
	assert.Equal(t, "2018-08-07T06-05-04", result)

	result, err = formatter.createSubtitute("[[schedule.2006-01-02T15-04-05]]")
	assert.Nil(t, err)
	assert.Equal(t, "2017-07-06T05-04-03", result)

	result, err = formatter.createSubtitute("[[something]]")
	assert.Nil(t, err)
	assert.Equal(t, "[[something]]", result)

}

func TestCreateSubstituteError(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, errors.New("UUID generation failed"))
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	result, err := formatter.createSubtitute("[[uuid]]")
	assert.Contains(t, err.Error(), "UUID generation failed")
	assert.Equal(t, "", result)
}

func TestFormatString(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	result, err := formatter.formatString("something")
	assert.Nil(t, err)
	assert.Equal(t, "something", result)

	result, err = formatter.formatString("something [[uuid]] something")
	assert.Nil(t, err)
	assert.Equal(t, "something "+defaultUUID+" something", result)

	result, err = formatter.formatString("a [[schedule]] b [[now.2006]] c [[schedule.01]] d")
	assert.Nil(t, err)
	assert.Equal(t, "a 20170706050403 b 2018 c 07 d", result)
}

func TestFormatStringError(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, errors.New("UUID generation failed"))
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	result, err := formatter.formatString("something [[uuid]] something")
	assert.Contains(t, err.Error(), "UUID generation failed")
	assert.Equal(t, "", result)
}

func TestFormatParameter(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	param := v1alpha1.Parameter{
		Name:  "PARAM_NAME",
		Value: StringPointer("PARAM_PREFIX_[[uuid]]_SUFFIX"),
	}

	expected := v1alpha1.Parameter{
		Name:  "PARAM_NAME",
		Value: StringPointer("PARAM_PREFIX_" + defaultUUID + "_SUFFIX"),
	}

	result, err := formatter.formatParameter(param)
	assert.Nil(t, err)
	assert.Equal(t, expected, *result)
}

func TestFormatParameterError(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, errors.New("UUID generation failed"))
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	param := v1alpha1.Parameter{
		Name:  "PARAM_NAME",
		Value: StringPointer("PARAM_PREFIX_[[uuid]]_SUFFIX"),
	}

	result, err := formatter.formatParameter(param)
	assert.Contains(t, err.Error(), "UUID generation failed")
	assert.Nil(t, result)
}

func TestFormatNothingToDoExceptAddUUID(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1")},
					{Name: "param2", Value: StringPointer("value2")},
				},
			}}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-name-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1")},
					{Name: "param2", Value: StringPointer("value2")},
				},
			}}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatEverytingToChange(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-[[schedule]]-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1-[[schedule]]")},
					{Name: "param2", Value: StringPointer("value2-[[now]]-suffix")},
				},
			}}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-20170706050403-name-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1-20170706050403")},
					{Name: "param2", Value: StringPointer("value2-20180807060504-suffix")},
				},
			}}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatOnlyWorkflowName(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-[[schedule]]-name"}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-20170706050403-name-"}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatOnlyWorkflowGeneratedName(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-[[schedule]]-name-"}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-20170706050403-name-"}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatNoWorkflowNames(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-"}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormat2WorkflowNames(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:         "workflow-[[schedule]]-name",
			GenerateName: "workflow-[[schedule]]-generated-name-"}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-20170706050403-generated-name-"}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatOnlyWorkflowParameters(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1-[[schedule]]")},
					{Name: "param2", Value: StringPointer("value2-[[now]]-suffix")},
				},
			}}}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1-20170706050403")},
					{Name: "param2", Value: StringPointer("value2-20180807060504-suffix")},
				},
			}}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatEmptyWorkflow(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, nil)
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{}

	expected := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-"}}

	err := formatter.Format(workflow)
	assert.Nil(t, err)
	assert.Equal(t, expected, workflow)
}

func TestFormatError(t *testing.T) {
	uuid := NewFakeUUIDGeneratorOrFatal(defaultUUID, errors.New("UUID generation failed"))
	formatter := NewWorkflowFormatter(uuid,
		getDefaultScheduledAtSec(),
		getDefaultCreatedAtSec())

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-[[schedule]]-name-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: StringPointer("value1-[[schedule]]-[[uuid]]")},
					{Name: "param2", Value: StringPointer("value2-[[now]]-suffix")},
				},
			}}}

	err := formatter.Format(workflow)
	assert.Contains(t, err.Error(), "UUID generation failed")
}
