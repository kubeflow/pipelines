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
	swfregister "github.com/googleprivate/ml/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/googleprivate/ml/crd/pkg/apis/scheduledworkflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Workflow is a type to help manipulate Workflow objects.
type Workflow struct {
	*workflowapi.Workflow
}

// NewWorkflow creates an Workflow.
func NewWorkflow(workflow *workflowapi.Workflow) *Workflow {
	return &Workflow{
		workflow,
	}
}

// Get converts this object to a workflowapi.Workflow.
func (w *Workflow) Get() *workflowapi.Workflow {
	return w.Workflow
}

// OverrideName sets the name of a Workflow.
func (w *Workflow) OverrideName(name string) {
	w.GenerateName = ""
	w.Name = name
}

// OverrideParameters overrides some of the parameters of a Workflow.
func (w *Workflow) OverrideParameters(desiredMap map[string]string) {
	desiredSlice := make([]workflowapi.Parameter, 0)
	for _, currentParam := range w.Spec.Arguments.Parameters {

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

	w.Spec.Arguments.Parameters = desiredSlice
}

// SetCanonicalLabels sets the labels needed by the ScheduledWorkflow on the Workflow.
func (w *Workflow) SetCanonicalLabels(scheduleName string,
	nextScheduledEpoch int64, index int64) {
	if w.Labels == nil {
		w.Labels = make(map[string]string)
	}
	w.Labels[LabelKeyWorkflowScheduledWorkflowName] = scheduleName
	w.Labels[LabelKeyWorkflowEpoch] = formatInt64ForLabel(
		nextScheduledEpoch)
	w.Labels[LabelKeyWorkflowIndex] = formatInt64ForLabel(index)
	w.Labels[LabelKeyWorkflowIsOwnedByScheduledWorkflow] = "true"
}

// SetOwnerReferences sets owner references on a Workflow.
func (w *Workflow) SetOwnerReferences(schedule *swfapi.ScheduledWorkflow) {
	w.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(schedule, schema.GroupVersionKind{
			Group:   swfapi.SchemeGroupVersion.Group,
			Version: swfapi.SchemeGroupVersion.Version,
			Kind:    swfregister.Kind,
		}),
	}
}
