// Copyright 2024 The Kubeflow Authors
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

// Package resource manages API-server resources.
package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// SetRunIntermediateInputsRequest carries the caller-supplied parameter values
// for a paused (suspend) node in a running workflow.
type SetRunIntermediateInputsRequest struct {
	// RunID is the KFP run identifier.
	RunID string
	// NodeDisplayName is the display name of the suspend node (e.g. "approval-gate").
	NodeDisplayName string
	// Parameters maps output parameter name → human-supplied value.
	Parameters map[string]string
}

// SetRunIntermediateInputs validates and applies human-supplied parameter
// values to a paused Argo suspend node, then resumes the workflow.
//
// Validation rules:
//   - The run must exist and be in a non-terminal state.
//   - A node with matching display name must exist, be of type Suspend, and be
//     in phase Running (i.e. actually paused waiting for input).
//   - Every key in Parameters must correspond to a declared output parameter on
//     the suspend template whose valueFrom is "supplied".
//   - If the template declares an enum for a parameter the supplied value must
//     be one of the listed choices.
//
// On success the node's output parameters are set, its phase is advanced to
// Succeeded, and the workflow is patched so the Argo controller can continue.
func (r *ResourceManager) SetRunIntermediateInputs(ctx context.Context, req SetRunIntermediateInputsRequest) error {
	run, err := r.GetRun(req.RunID)
	if err != nil {
		return util.Wrapf(err, "SetRunIntermediateInputs: failed to fetch run %s", req.RunID)
	}

	namespace, err := r.getNamespaceFromRunId(req.RunID)
	if err != nil {
		return util.Wrapf(err, "SetRunIntermediateInputs: failed to determine namespace for run %s", req.RunID)
	}
	if namespace == "" {
		namespace = getDefaultNamespace()
	}

	wfClient := r.getWorkflowClient(namespace)
	execSpec, err := wfClient.Get(ctx, run.K8SName, metav1.GetOptions{})
	if err != nil {
		return util.NewInternalServerError(err, "SetRunIntermediateInputs: failed to fetch workflow %s", run.K8SName)
	}

	wf, ok := execSpec.(*util.Workflow)
	if !ok {
		return util.NewInternalServerError(fmt.Errorf("unexpected ExecutionSpec type %T", execSpec),
			"SetRunIntermediateInputs: workflow type assertion failed")
	}

	// Find the target node by display name.
	nodeID, nodeStatus, err := findSuspendNodeByDisplayName(wf.Get(), req.NodeDisplayName)
	if err != nil {
		return err
	}

	// Locate the suspend template to validate declared parameters.
	tmpl, err := findTemplateForNode(wf.Get(), nodeStatus)
	if err != nil {
		return err
	}

	// Validate the supplied parameters against the template's declared outputs.
	if err := validateIntermediateParams(tmpl, req.Parameters); err != nil {
		return err
	}

	// Build the status patch: set the output parameter values and mark the node
	// as Succeeded so the workflow controller can proceed.
	patchBytes, err := buildIntermediateInputsPatch(nodeID, nodeStatus, tmpl, req.Parameters)
	if err != nil {
		return util.NewInternalServerError(err, "SetRunIntermediateInputs: failed to build status patch")
	}

	glog.Infof("SetRunIntermediateInputs: patching node %q (id=%s) in workflow %s", req.NodeDisplayName, nodeID, run.K8SName)
	if _, err := wfClient.Patch(ctx, run.K8SName, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status"); err != nil {
		return util.NewInternalServerError(err, "SetRunIntermediateInputs: failed to patch workflow %s", run.K8SName)
	}
	return nil
}

// findSuspendNodeByDisplayName scans the workflow's status nodes for a node
// whose DisplayName matches displayName, is of type Suspend, and is in phase
// Running (i.e. awaiting input).
func findSuspendNodeByDisplayName(wf *wfapi.Workflow, displayName string) (string, wfapi.NodeStatus, error) {
	for nodeID, nodeStatus := range wf.Status.Nodes {
		if nodeStatus.DisplayName != displayName {
			continue
		}
		if nodeStatus.Type != wfapi.NodeTypeSuspend {
			return "", wfapi.NodeStatus{}, util.NewBadRequestError(
				fmt.Errorf("node %q has type %q, not Suspend", displayName, nodeStatus.Type),
				"The specified node is not a suspend/human-input node",
			)
		}
		if nodeStatus.Phase != wfapi.NodeRunning {
			return "", wfapi.NodeStatus{}, util.NewBadRequestError(
				fmt.Errorf("node %q is in phase %q; expected Running", displayName, nodeStatus.Phase),
				"The suspend node is not currently waiting for input (phase must be Running)",
			)
		}
		return nodeID, nodeStatus, nil
	}
	return "", wfapi.NodeStatus{}, util.NewResourceNotFoundError(
		"SuspendNode", displayName,
	)
}

// findTemplateForNode locates the Argo template used by the given node status,
// searching first in StoredTemplates (the canonical source at runtime) and
// then falling back to the workflow spec's Templates list.
func findTemplateForNode(wf *wfapi.Workflow, node wfapi.NodeStatus) (*wfapi.Template, error) {
	tmplName := node.TemplateName
	if tmplName == "" {
		return nil, util.NewBadRequestError(
			fmt.Errorf("node %q has no TemplateName", node.DisplayName),
			"Cannot find template for the suspend node",
		)
	}

	// Prefer StoredTemplates (available during execution).
	if st, ok := wf.Status.StoredTemplates[tmplName]; ok {
		tmpl := st
		return &tmpl, nil
	}
	// Fall back to the workflow spec.
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == tmplName {
			tmpl := wf.Spec.Templates[i]
			return &tmpl, nil
		}
	}
	return nil, util.NewInternalServerError(
		fmt.Errorf("template %q not found in workflow", tmplName),
		"Cannot locate suspend template %q",
		tmplName,
	)
}

// validateIntermediateParams checks the caller-supplied parameters against the
// suspend template's declared output parameters:
//   - All supplied keys must correspond to a declared output with valueFrom.supplied.
//   - If the parameter has an enum constraint the value must be one of the choices.
func validateIntermediateParams(tmpl *wfapi.Template, params map[string]string) error {
	declaredOutputs := make(map[string]wfapi.Parameter, len(tmpl.Outputs.Parameters))
	for _, p := range tmpl.Outputs.Parameters {
		if p.ValueFrom != nil && p.ValueFrom.Supplied != nil {
			declaredOutputs[p.Name] = p
		}
	}
	inputMetadata := make(map[string]wfapi.Parameter, len(tmpl.Inputs.Parameters))
	for _, p := range tmpl.Inputs.Parameters {
		inputMetadata[p.Name] = p
	}

	for key, value := range params {
		_, ok := declaredOutputs[key]
		if !ok {
			return util.NewBadRequestError(
				fmt.Errorf("parameter %q is not a declared supplied output of template %q", key, tmpl.Name),
				"Supplied parameter key %q is not declared on this suspend node", key,
			)
		}
		if input, ok := inputMetadata[key]; ok && len(input.Enum) > 0 {
			valid := false
			for _, choice := range input.Enum {
				if choice.String() == value {
					valid = true
					break
				}
			}
			if !valid {
				allowed := make([]string, len(input.Enum))
				for i, e := range input.Enum {
					allowed[i] = e.String()
				}
				return util.NewBadRequestError(
					fmt.Errorf("value %q for parameter %q is not in the allowed enum %v", value, key, allowed),
					"Value %q is not one of the allowed choices for parameter %q", value, key,
				)
			}
		}
	}
	return nil
}

// buildIntermediateInputsPatch constructs the JSON merge-patch document that:
//   - Sets output parameter values on the suspend node.
//   - Advances the node phase to Succeeded and records the current time as
//     finishedAt so the Argo workflow controller can proceed.
func buildIntermediateInputsPatch(
	nodeID string,
	existing wfapi.NodeStatus,
	tmpl *wfapi.Template,
	params map[string]string,
) ([]byte, error) {
	// Build the updated parameters list by merging the template-declared
	// outputs with the caller-supplied values.
	updatedParams := make([]wfapi.Parameter, 0, len(tmpl.Outputs.Parameters))
	for _, p := range tmpl.Outputs.Parameters {
		if p.ValueFrom == nil || p.ValueFrom.Supplied == nil {
			updatedParams = append(updatedParams, p)
			continue
		}
		if v, ok := params[p.Name]; ok {
			p.Value = wfapi.AnyStringPtr(v)
		}
		updatedParams = append(updatedParams, p)
	}

	// Preserve any existing artifacts/result on the outputs.
	updatedOutputs := existing.Outputs
	if updatedOutputs == nil {
		updatedOutputs = &wfapi.Outputs{}
	}
	updatedOutputs.Parameters = updatedParams

	// The node patch uses a partial node structure so only the fields we
	// intend to change are included in the merge-patch document.
	type nodePatch struct {
		Phase      wfapi.NodePhase `json:"phase"`
		FinishedAt metav1.Time     `json:"finishedAt"`
		Message    string          `json:"message"`
		Outputs    *wfapi.Outputs  `json:"outputs,omitempty"`
	}

	now := metav1.NewTime(time.Now().UTC())
	patch := struct {
		Status struct {
			Nodes map[string]nodePatch `json:"nodes"`
		} `json:"status"`
	}{}
	patch.Status.Nodes = map[string]nodePatch{
		nodeID: {
			Phase:      wfapi.NodeSucceeded,
			FinishedAt: now,
			Message:    "human input provided",
			Outputs:    updatedOutputs,
		},
	}

	return json.Marshal(patch)
}

// IntermediateInputsStatus describes the current state of a suspend node's
// human-input parameters, returned by GetRunIntermediateInputsStatus.
type IntermediateInputsStatus struct {
	// Suspended is true when the node is a Suspend node currently in the
	// Running phase (i.e. waiting for human input).
	Suspended bool `json:"suspended"`
	// Parameters contains per-parameter metadata and the current supplied value
	// (nil when not yet provided).
	Parameters map[string]*IntermediateParamStatus `json:"parameters,omitempty"`
}

// IntermediateParamStatus holds metadata and the current value for one
// human-input parameter.
type IntermediateParamStatus struct {
	Description  string   `json:"description,omitempty"`
	Default      string   `json:"default,omitempty"`
	Enum         []string `json:"enum,omitempty"`
	CurrentValue *string  `json:"current_value,omitempty"`
}

// GetRunIntermediateInputsStatus returns the current state of a suspend node's
// human-input parameters without modifying the workflow.  It is safe to call
// regardless of the node's current phase; the Suspended field in the response
// indicates whether the caller may now call SetRunIntermediateInputs.
func (r *ResourceManager) GetRunIntermediateInputsStatus(ctx context.Context, runID, nodeDisplayName string) (*IntermediateInputsStatus, error) {
	run, err := r.GetRun(runID)
	if err != nil {
		return nil, util.Wrapf(err, "GetRunIntermediateInputsStatus: failed to fetch run %s", runID)
	}

	namespace, err := r.getNamespaceFromRunId(runID)
	if err != nil {
		return nil, util.Wrapf(err, "GetRunIntermediateInputsStatus: failed to determine namespace for run %s", runID)
	}
	if namespace == "" {
		namespace = getDefaultNamespace()
	}

	wfClient := r.getWorkflowClient(namespace)
	execSpec, err := wfClient.Get(ctx, run.K8SName, metav1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "GetRunIntermediateInputsStatus: failed to fetch workflow %s", run.K8SName)
	}

	wf, ok := execSpec.(*util.Workflow)
	if !ok {
		return nil, util.NewInternalServerError(fmt.Errorf("unexpected ExecutionSpec type %T", execSpec),
			"GetRunIntermediateInputsStatus: workflow type assertion failed")
	}

	// Scan for the node by display name; any phase is acceptable here.
	var (
		foundNode   *wfapi.NodeStatus
		foundNodeID string
	)
	for nodeID, ns := range wf.Get().Status.Nodes {
		if ns.DisplayName == nodeDisplayName {
			ns := ns
			foundNode = &ns
			foundNodeID = nodeID
			break
		}
	}
	_ = foundNodeID // used only for future logging

	// If the node doesn't exist yet, return a non-suspended status so the UI
	// knows the task hasn't reached this point in the workflow.
	if foundNode == nil {
		return &IntermediateInputsStatus{Suspended: false}, nil
	}

	isSuspended := foundNode.Type == wfapi.NodeTypeSuspend && foundNode.Phase == wfapi.NodeRunning
	status := &IntermediateInputsStatus{Suspended: isSuspended}

	// Try to retrieve template metadata so the UI can render the form fields.
	tmpl, err := findTemplateForNode(wf.Get(), *foundNode)
	if err != nil {
		// Template not found is non-fatal for a status query.
		return status, nil
	}

	paramsMeta := make(map[string]*IntermediateParamStatus, len(tmpl.Outputs.Parameters))

	// Build a map of already-supplied values from the node's current outputs.
	suppliedValues := make(map[string]string)
	if foundNode.Outputs != nil {
		for _, p := range foundNode.Outputs.Parameters {
			if p.Value != nil {
				suppliedValues[p.Name] = p.Value.String()
			}
		}
	}

	// Merge template input metadata (description, default, enum) with template
	// output declarations (which carry the valueFrom.supplied marker).
	inputMeta := make(map[string]*wfapi.Parameter)
	for i := range tmpl.Inputs.Parameters {
		p := tmpl.Inputs.Parameters[i]
		inputMeta[p.Name] = &p
	}

	for _, p := range tmpl.Outputs.Parameters {
		if p.ValueFrom == nil || p.ValueFrom.Supplied == nil {
			continue
		}
		ps := &IntermediateParamStatus{}

		// Pull description and enum from the matching input parameter if present.
		if meta, ok := inputMeta[p.Name]; ok {
			if meta.Description != nil {
				ps.Description = meta.Description.String()
			}
			if meta.Default != nil {
				ps.Default = meta.Default.String()
			}
			for _, e := range meta.Enum {
				ps.Enum = append(ps.Enum, e.String())
			}
		}

		if v, ok := suppliedValues[p.Name]; ok {
			ps.CurrentValue = &v
		}
		paramsMeta[p.Name] = ps
	}
	if len(paramsMeta) > 0 {
		status.Parameters = paramsMeta
	}
	return status, nil
}

// getDefaultNamespace returns the pod namespace as the default KFP namespace.
func getDefaultNamespace() string {
	return common.GetPodNamespace()
}
