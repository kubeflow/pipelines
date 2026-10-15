// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"fmt"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"k8s.io/apimachinery/pkg/types"
)

// attemptResolver identifies the active taskset attempt a request belongs to,
// as "<workflow UID>/<taskset node ID>".
type attemptResolver func(context.Context, driverapi.DriverPluginArgs) (string, error)

func tasksetAttemptResolver(agent agentLoader, taskset tasksetSource) attemptResolver {
	return func(ctx context.Context, args driverapi.DriverPluginArgs) (string, error) {
		clients, err := agent(ctx)
		if err != nil {
			return "", err
		}
		if err := clients.validateRequest(args); err != nil {
			return "", err
		}
		owner := clients.workflow
		current, err := taskset(ctx, clients)
		if err != nil {
			return "", err
		}
		if current.Namespace != clients.pod.Namespace || current.Name != owner.Name {
			return "", fmt.Errorf("WorkflowTaskSet %s/%s does not belong to agent Workflow %s/%s; report this driver bug", current.Namespace, current.Name, clients.pod.Namespace, owner.Name)
		}
		if !ownedByWorkflow(current, owner.UID) {
			return "", fmt.Errorf("WorkflowTaskSet %s/%s is not owned by agent Workflow UID %s; wait for Argo to recreate it", current.Namespace, current.Name, owner.UID)
		}
		match, err := activeAttempt(current, args)
		if err != nil {
			return "", err
		}
		return string(owner.UID) + "/" + match, nil
	}
}

func ownedByWorkflow(taskset *workflowapi.WorkflowTaskSet, workflowUID types.UID) bool {
	for _, ref := range taskset.OwnerReferences {
		if ref.Kind == "Workflow" && ref.UID == workflowUID {
			return true
		}
	}
	return false
}

// activeAttempt finds the single unfinished taskset node whose driver-plugin
// args match the request. A retried node has a new ID while the old one stays
// in a terminal phase.
func activeAttempt(taskset *workflowapi.WorkflowTaskSet, args driverapi.DriverPluginArgs) (string, error) {
	want := argsDigest(args)
	var match string
	for id, tmpl := range taskset.Spec.Tasks {
		if tmpl.Plugin == nil {
			continue
		}
		candidate, owned, err := driverapi.ParseDriverPluginArgs(tmpl.Plugin.Value)
		if err != nil {
			return "", fmt.Errorf("WorkflowTaskSet node %q has an invalid driver template; recompile the pipeline with a current API server: %w", id, err)
		}
		if !owned || argsDigest(*candidate) != want {
			continue
		}
		switch taskset.Status.Nodes[id].Phase {
		case workflowapi.NodeSucceeded, workflowapi.NodeFailed, workflowapi.NodeError, workflowapi.NodeSkipped, workflowapi.NodeOmitted:
			continue
		}
		if match != "" || id == "" {
			return "", fmt.Errorf("WorkflowTaskSet %s/%s has more than one active node for task %q; wait for Argo to reconcile", taskset.Namespace, taskset.Name, args.TaskName)
		}
		match = id
	}
	if match == "" {
		return "", fmt.Errorf("WorkflowTaskSet %s/%s has no active node for task %q; retry after Argo adds the task", taskset.Namespace, taskset.Name, args.TaskName)
	}
	return match, nil
}
