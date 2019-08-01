// Copyright 2018 Google LLC
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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

type RunServer struct {
	resourceManager *resource.ResourceManager
}

func (s *RunServer) CreateRun(ctx context.Context, request *api.CreateRunRequest) (*api.RunDetail, error) {
	err := s.validateCreateRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}
	run, err := s.resourceManager.CreateRun(request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) ResubmitRun(ctx context.Context, request *api.ResubmitRunRequest) (*api.RunDetail, error) {
	oldRunId := request.Id
	runDetails, err := s.resourceManager.GetRun(oldRunId)
	if err != nil {
		return nil, err
	}

	// Verify valid workflow template
	var workflow util.Workflow
	if err := json.Unmarshal([]byte( runDetails.WorkflowRuntimeManifest), &workflow); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to retrieve the runtime pipeline spec from the old run")
	}
	newWorkflow, err := FormulateResubmitWorkflow(workflow.Workflow)
	workflowManifest, err := json.Marshal(newWorkflow)

	newRun := &api.Run{Name: "todo", Description: runDetails.Description, ResourceReferences: toApiResourceReferences(runDetails.ResourceReferences),
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string(workflowManifest)}}

	run, err := s.resourceManager.CreateRun(newRun)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}
	return ToApiRunDetail(run), nil
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// convertNodeID converts an old nodeID to a new nodeID
func convertNodeID(newWf *wfv1.Workflow, regex *regexp.Regexp, oldNodeID string, oldNodes map[string]wfv1.NodeStatus) string {
	node := oldNodes[oldNodeID]
	newNodeName := regex.ReplaceAllString(node.Name, newWf.ObjectMeta.Name)
	return newWf.NodeID(newNodeName)
}

// FormulateResubmitWorkflow formulate a new workflow from a previous workflow optionally re-using successful nodes
func FormulateResubmitWorkflow(wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	newWF := wfv1.Workflow{}
	newWF.TypeMeta = wf.TypeMeta

	// Resubmitted workflow will use generated names
	if wf.ObjectMeta.GenerateName != "" {
		newWF.ObjectMeta.GenerateName = wf.ObjectMeta.GenerateName
	} else {
		newWF.ObjectMeta.GenerateName = wf.ObjectMeta.Name + "-"
	}
	// When resubmitting workflow with memoized nodes, we need to use a predetermined workflow name
	// in order to formulate the node statuses. Which means we cannot reuse metadata.generateName
	// The following simulates the behavior of generateName
	newWF.ObjectMeta.Name = newWF.ObjectMeta.GenerateName + randString(5)

	// carry over the unmodified spec
	newWF.Spec = wf.Spec

	// carry over user labels and annotations from previous workflow.
	// skip any argoproj.io labels except for the controller instanceID label.
	for key, val := range wf.ObjectMeta.Labels {
		if strings.HasPrefix(key, workflow.FullName+"/") && key != workflow.FullName+"/controller-instanceid" {
			continue
		}
		if newWF.ObjectMeta.Labels == nil {
			newWF.ObjectMeta.Labels = make(map[string]string)
		}
		newWF.ObjectMeta.Labels[key] = val
	}
	for key, val := range wf.ObjectMeta.Annotations {
		if newWF.ObjectMeta.Annotations == nil {
			newWF.ObjectMeta.Annotations = make(map[string]string)
		}
		newWF.ObjectMeta.Annotations[key] = val
	}

	// Iterate the previous nodes. If it was successful Pod carry it forward
	replaceRegexp := regexp.MustCompile("^" + wf.ObjectMeta.Name)
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	for _, node := range wf.Status.Nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			originalID := node.ID
			node.Name = replaceRegexp.ReplaceAllString(node.Name, newWF.ObjectMeta.Name)
			node.ID = newWF.NodeID(node.Name)
			node.BoundaryID = convertNodeID(&newWF, replaceRegexp, node.BoundaryID, wf.Status.Nodes)
			node.StartedAt = metav1.Time{Time: time.Now().UTC()}
			node.FinishedAt = node.StartedAt
			newChildren := make([]string, len(node.Children))
			for i, childID := range node.Children {
				newChildren[i] = convertNodeID(&newWF, replaceRegexp, childID, wf.Status.Nodes)
			}
			node.Children = newChildren
			newOutboundNodes := make([]string, len(node.OutboundNodes))
			for i, outboundID := range node.OutboundNodes {
				newOutboundNodes[i] = convertNodeID(&newWF, replaceRegexp, outboundID, wf.Status.Nodes)
			}
			node.OutboundNodes = newOutboundNodes
			if node.Type == wfv1.NodeTypePod {
				node.Phase = wfv1.NodeSkipped
				node.Type = wfv1.NodeTypeSkipped
				node.Message = fmt.Sprintf("original pod: %s", originalID)
			}
			newWF.Status.Nodes[node.ID] = node
		case wfv1.NodeError, wfv1.NodeFailed, wfv1.NodeRunning:
			// do not add this status to the node. pretend as if this node never existed.
			// NOTE: NodeRunning shouldn't really happen except in weird scenarios where controller
			// mismanages state (e.g. panic when operating on a workflow)
		default:
			return nil, errors.InternalErrorf("Workflow cannot be resubmitted with nodes in %s phase", node, node.Phase)
		}
	}
	return &newWF, nil
}

func (s *RunServer) GetRun(ctx context.Context, request *api.GetRunRequest) (*api.RunDetail, error) {
	run, err := s.resourceManager.GetRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) ListRuns(ctx context.Context, request *api.ListRunsRequest) (*api.ListRunsResponse, error) {
	opts, err := validatedListOptions(&model.Run{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}
	runs, total_size, nextPageToken, err := s.resourceManager.ListRuns(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs.")
	}
	return &api.ListRunsResponse{Runs: ToApiRuns(runs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *RunServer) ArchiveRun(ctx context.Context, request *api.ArchiveRunRequest) (*empty.Empty, error) {
	err := s.resourceManager.ArchiveRun(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) UnarchiveRun(ctx context.Context, request *api.UnarchiveRunRequest) (*empty.Empty, error) {
	err := s.resourceManager.UnarchiveRun(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) DeleteRun(ctx context.Context, request *api.DeleteRunRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeleteRun(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) ReportRunMetrics(ctx context.Context, request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	// Makes sure run exists
	_, err := s.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := ValidateRunMetric(metric)
		if err == nil {
			err = s.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			NewReportRunMetricResult(metric.GetName(), metric.GetNodeId(), err))
	}
	return response, nil
}

func (s *RunServer) ReadArtifact(ctx context.Context, request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	content, err := s.resourceManager.ReadArtifact(
		request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		return nil, util.Wrapf(err, "failed to read artifact '%+v'.", request)
	}
	return &api.ReadArtifactResponse{
		Data: content,
	}, nil
}

func (s *RunServer) validateCreateRunRequest(request *api.CreateRunRequest) error {
	run := request.Run
	if run.Name == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}

	if err := ValidatePipelineSpec(s.resourceManager, run.PipelineSpec); err != nil {
		return util.Wrap(err, "The pipeline spec is invalid.")
	}
	return nil
}

func (s *RunServer) TerminateRun(ctx context.Context, request *api.TerminateRunRequest) (*empty.Empty, error) {
	err := s.resourceManager.TerminateRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func NewRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{resourceManager: resourceManager}
}
