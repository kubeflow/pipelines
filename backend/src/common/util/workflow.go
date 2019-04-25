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
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
)

// Workflow is a type to help manipulate Workflow objects.
type Workflow struct {
	*workflowapi.Workflow
}

// NewWorkflow creates a Workflow.
func NewWorkflow(workflow *workflowapi.Workflow) *Workflow {
	return &Workflow{
		workflow,
	}
}

// OverrideParameters overrides some of the parameters of a Workflow.
func (w *Workflow) OverrideParameters(desiredParams map[string]string) {
	desiredSlice := make([]workflowapi.Parameter, 0)
	for _, currentParam := range w.Spec.Arguments.Parameters {
		var desiredValue *string = nil
		if param, ok := desiredParams[currentParam.Name]; ok {
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

func (w *Workflow) VerifyParameters(desiredParams map[string]string) error {
	templateParamsMap := make(map[string]*string)
	for _, param := range w.Spec.Arguments.Parameters {
		templateParamsMap[param.Name] = param.Value
	}
	for k := range desiredParams {
		_, ok := templateParamsMap[k]
		if !ok {
			return NewInvalidInputError("Unrecognized input parameter: %v", k)
		}
	}
	return nil
}

// Get converts this object to a workflowapi.Workflow.
func (w *Workflow) Get() *workflowapi.Workflow {
	return w.Workflow
}

func (w *Workflow) ScheduledWorkflowUUIDAsStringOrEmpty() string {
	if w.OwnerReferences == nil {
		return ""
	}

	for _, reference := range w.OwnerReferences {
		if isScheduledWorkflow(reference) {
			return string(reference.UID)
		}
	}

	return ""
}

func containsScheduledWorkflow(references []metav1.OwnerReference) bool {
	if references == nil {
		return false
	}

	for _, reference := range references {
		if isScheduledWorkflow(reference) {
			return true
		}
	}

	return false
}

func isScheduledWorkflow(reference metav1.OwnerReference) bool {
	gvk := schema.GroupVersionKind{
		Group:   swfapi.SchemeGroupVersion.Group,
		Version: swfapi.SchemeGroupVersion.Version,
		Kind:    swfregister.Kind,
	}

	if reference.APIVersion == gvk.GroupVersion().String() &&
		reference.Kind == gvk.Kind &&
		reference.UID != "" {
		return true
	}
	return false
}

func (w *Workflow) ScheduledAtInSecOr0() int64 {
	if w.Labels == nil {
		return 0
	}

	for key, value := range w.Labels {
		if key == LabelKeyWorkflowEpoch {
			result, err := RetrieveInt64FromLabel(value)
			if err != nil {
				glog.Errorf("Could not retrieve scheduled epoch from label key (%v) and label value (%v).", key, value)
				return 0
			}
			return result
		}
	}

	return 0
}

func (w *Workflow) FinishedAt() int64 {
	if w.Status.FinishedAt.IsZero(){
		// If workflow is not finished
		return 0
	}
	return w.Status.FinishedAt.Unix()
}

func (w *Workflow) Condition() string {
	return string(w.Status.Phase)
}

func (w *Workflow) ToStringForStore() string {

	workflow, err := json.Marshal(w.Workflow)
	if err != nil {
		glog.Errorf("Could not marshal the workflow: %v", w.Workflow)
		return ""
	}
	return string(workflow)
}

func (w *Workflow) HasScheduledWorkflowAsParent() bool {
	return containsScheduledWorkflow(w.Workflow.OwnerReferences)
}

func (w *Workflow) GetSpec() *Workflow {
	spec := w.DeepCopy()
	spec.Status = workflowapi.WorkflowStatus{}
	return NewWorkflow(spec)
}

// OverrideName sets the name of a Workflow.
func (w *Workflow) OverrideName(name string) {
	w.GenerateName = ""
	w.Name = name
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

func (w *Workflow) SetLabels(key string, value string) {
	if w.Labels == nil {
		w.Labels = make(map[string]string)
	}
	w.Labels[key] = value
}

func (w *Workflow) SetCannonicalLabels(name string, nextScheduledEpoch int64, index int64) {
	w.SetLabels(LabelKeyWorkflowScheduledWorkflowName, name)
	w.SetLabels(LabelKeyWorkflowEpoch, FormatInt64ForLabel(nextScheduledEpoch))
	w.SetLabels(LabelKeyWorkflowIndex, FormatInt64ForLabel(index))
	w.SetLabels(LabelKeyWorkflowIsOwnedByScheduledWorkflow, "true")
}

// FindObjectStoreArtifactKeyOrEmpty loops through all node running statuses and look up the first
// S3 artifact with the specified nodeID and artifactName. Returns empty if nothing is found.
func (w *Workflow) FindObjectStoreArtifactKeyOrEmpty(nodeID string, artifactName string) string {
	if w.Status.Nodes == nil {
		return ""
	}
	node, found := w.Status.Nodes[nodeID]
	if !found {
		return ""
	}
	if node.Outputs == nil || node.Outputs.Artifacts == nil {
		return ""
	}
	var s3Key string
	for _, artifact := range node.Outputs.Artifacts {
		if artifact.Name != artifactName || artifact.S3 == nil || artifact.S3.Key == "" {
			continue
		}
		s3Key = artifact.S3.Key
	}
	return s3Key
}
