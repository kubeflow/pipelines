// Copyright 2018 The Kubeflow Authors
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

package resource

import (
	"context"
	"errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func formulateRetryWorkflow(wf *util.Workflow) (*util.Workflow, []string, error) {
	switch wf.Status.Phase {
	case wfv1.WorkflowFailed, wfv1.WorkflowError:
		break
	default:
		return nil, nil, util.NewBadRequestError(errors.New("workflow cannot be retried"), "Workflow must be Failed/Error to retry")
	}

	newWF := wf.DeepCopy()
	// Delete/reset fields which indicate workflow completed
	delete(newWF.Labels, common.LabelKeyCompleted)
	// Delete/reset fields which indicate workflow is finished being persisted to the database
	delete(newWF.Labels, util.LabelKeyWorkflowPersistedFinalState)
	newWF.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeRunning)
	newWF.Status.Phase = wfv1.WorkflowRunning
	newWF.Status.Message = ""
	newWF.Status.FinishedAt = metav1.Time{}
	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	// Iterate the previous nodes. If it was successful Pod carry it forward
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	var podsToDelete []string
	for _, node := range wf.Status.Nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			if !strings.HasPrefix(node.Name, onExitNodeName) {
				newWF.Status.Nodes[node.ID] = node
				continue
			}
		case wfv1.NodeError, wfv1.NodeFailed, wfv1.NodeOmitted:
			if !strings.HasPrefix(node.Name, onExitNodeName) && node.Type == wfv1.NodeTypeDAG {
				newNode := node.DeepCopy()
				newNode.Phase = wfv1.NodeRunning
				newNode.Message = ""
				newNode.FinishedAt = metav1.Time{}
				newWF.Status.Nodes[newNode.ID] = *newNode
				continue
			}
			// do not add this status to the node. pretend as if this node never existed.
		default:
			// Do not allow retry of workflows with pods in Running/Pending phase
			return nil, nil, util.NewInternalServerError(
				errors.New("workflow cannot be retried"),
				"Workflow cannot be retried with node %s in %s phase", node.ID, node.Phase)
		}
		if node.Type == wfv1.NodeTypePod {
			podsToDelete = append(podsToDelete, node.ID)
		}
	}
	return util.NewWorkflow(newWF), podsToDelete, nil
}

func deletePods(ctx context.Context, k8sCoreClient client.KubernetesCoreInterface, podsToDelete []string, namespace string) error {
	for _, podId := range podsToDelete {
		err := k8sCoreClient.PodClient(namespace).Delete(ctx, podId, metav1.DeleteOptions{})
		if err != nil && !apierr.IsNotFound(err) {
			return util.NewInternalServerError(err, "Failed to delete pods.")
		}
	}
	return nil
}

// Convert PipelineId in PipelineSpec to the pipeline's default pipeline version.
// This is for legacy usage of pipeline id to create run. The standard way to
// create run is by specifying the pipeline version.
func convertPipelineIdToDefaultPipelineVersion(pipelineSpec *api.PipelineSpec, resourceReferences *[]*api.ResourceReference, r *ResourceManager) error {
	if pipelineSpec == nil || pipelineSpec.GetPipelineId() == "" {
		return nil
	}
	// If there is already a pipeline version in resource references, don't convert pipeline id.
	for _, reference := range *resourceReferences {
		if reference.Key.Type == api.ResourceType_PIPELINE_VERSION && reference.Relationship == api.Relationship_CREATOR {
			return nil
		}
	}
	pipeline, err := r.pipelineStore.GetPipelineWithStatus(pipelineSpec.GetPipelineId(), model.PipelineReady)
	if err != nil {
		return util.Wrap(err, "Failed to find the specified pipeline")
	}
	// Add default pipeline version to resource references
	*resourceReferences = append(*resourceReferences, &api.ResourceReference{
		Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: pipeline.DefaultVersionId},
		Relationship: api.Relationship_CREATOR,
	})
	return nil
}
