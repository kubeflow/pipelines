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
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	scheduleregister "github.com/kubeflow/pipelines/pkg/apis/schedule"
	scheduleapi "github.com/kubeflow/pipelines/pkg/apis/schedule/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type WorkflowWrap struct {
	workflow *workflowapi.Workflow
}

func NewWorkflowWrap(workflow *workflowapi.Workflow) *WorkflowWrap {
	return &WorkflowWrap{
		workflow: workflow,
	}
}

func (w *WorkflowWrap) OverrideName(name string) {
	w.workflow.GenerateName = ""
	w.workflow.Name = name
}

func (w *WorkflowWrap) OverrideParameters(desiredMap map[string]string) {
	desiredSlice := make([]workflowapi.Parameter, 0)
	for _, currentParam := range w.workflow.Spec.Arguments.Parameters {

		var desiredValue *string = nil
		if param, ok := desiredMap[currentParam.Name]; ok {
			desiredValue = &param
		} else {
			desiredValue = currentParam.Value
		}
		desiredSlice = append(desiredSlice, workflowapi.Parameter{
			Name:  currentParam.Name,
			Value: desiredValue,
		})
	}

	w.workflow.Spec.Arguments.Parameters = desiredSlice
}

func (w *WorkflowWrap) SetCanonicalLabels(scheduleName string,
	nextScheduledEpoch int64, index int64) {
	if w.workflow.Labels == nil {
		w.workflow.Labels = make(map[string]string)
	}
	w.workflow.Labels[LabelKeyWorkflowName] = scheduleName
	w.workflow.Labels[LabelKeyWorkflowScheduledEpoch] = FormatInt64ForLabel(
		nextScheduledEpoch)
	w.workflow.Labels[LabelKeyWorkflowIndex] = FormatInt64ForLabel(index)
	w.workflow.Labels[LabelKeyWorkflowIsOwnedBySchedule] = "true"
}

func (w *WorkflowWrap) SetOwnerReferences(schedule *scheduleapi.Schedule) {
	w.workflow.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(schedule, schema.GroupVersionKind{
			Group:   scheduleapi.SchemeGroupVersion.Group,
			Version: scheduleapi.SchemeGroupVersion.Version,
			Kind:    scheduleregister.Kind,
		}),
	}
}

func (w *WorkflowWrap) Workflow() *workflowapi.Workflow {
	return w.workflow
}
