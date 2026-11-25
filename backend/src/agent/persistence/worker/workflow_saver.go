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

package worker

import (
	"context"
	"time"

	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// defaultStuckPodCheckTimeout bounds a single stuck pod check,
// including its termination request, so a stalled API server connection cannot
// hold a persistence worker indefinitely.
const defaultStuckPodCheckTimeout = 30 * time.Second

// WorkflowSaver provides a function to persist a workflow to a database.
type WorkflowSaver struct {
	client                        client.WorkflowClientInterface
	pipelineClient                client.PipelineClientInterface
	ttlSecondsAfterWorkflowFinish int64
	stuckPodChecker               StuckPodChecker
	stuckPodCheckTimeout          time.Duration
}

func NewWorkflowSaver(client client.WorkflowClientInterface,
	pipelineClient client.PipelineClientInterface, ttlSecondsAfterWorkflowFinish int64) *WorkflowSaver {
	return &WorkflowSaver{
		client:                        client,
		pipelineClient:                pipelineClient,
		ttlSecondsAfterWorkflowFinish: ttlSecondsAfterWorkflowFinish,
		stuckPodCheckTimeout:          defaultStuckPodCheckTimeout,
	}
}

// SetStuckPodChecker sets the optional stuck pod checker. When set, running
// workflows are checked for pods that are stuck for good (for example in
// ImagePullBackOff or unschedulable) and terminated once the failure has
// outlasted its grace period.
func (s *WorkflowSaver) SetStuckPodChecker(checker StuckPodChecker) {
	s.stuckPodChecker = checker
}

func (s *WorkflowSaver) Save(key string, namespace string, name string, nowEpoch int64) error {
	// Get the Workflow with this namespace/name
	wf, err := s.client.Get(namespace, name)
	isNotFound := util.HasCustomCode(err, util.CUSTOM_CODE_NOT_FOUND)
	if err != nil && isNotFound {
		// Permanent failure.
		// The Workflow may no longer exist, we stop processing and do not retry.
		if s.stuckPodChecker != nil {
			s.stuckPodChecker.Forget(namespace, name)
		}
		return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Workflow (%s) in work queue no longer exists: %v", key, err)
	}
	if err != nil && !isNotFound {
		// Transient failure, we will retry.
		return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Workflow (%s): transient failure: %v", key, err)

	}
	if _, ok := wf.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowRunId]; !ok {
		log.Infof("Skip syncing Workflow (%v): workflow does not have a Run ID label.", name)
		return nil
	}
	// Drop any stuck pod tracking state as soon as the workflow is
	// seen in a final state. This must happen before the persisted-final
	// shortcut below: the API server may have adopted and persisted the
	// terminal workflow while this agent still saw it running, in which case
	// every later sync takes the shortcut and the state would otherwise
	// survive until a retry of the run inherits the consumed grace period.
	if s.stuckPodChecker != nil && wf.ExecutionStatus().IsInFinalState() {
		s.stuckPodChecker.Forget(namespace, name)
	}
	if wf.PersistedFinalState() && time.Now().Unix()-wf.ExecutionStatus().FinishedAt() < s.ttlSecondsAfterWorkflowFinish {
		// Skip persisting the workflow if the workflow is finished
		// and the workflow hasn't being passing the TTL
		log.Infof("Skip syncing Workflow (%v): workflow marked as persisted.", name)
		return nil
	}

	// Check for stuck pods on workflows that are still running. The
	// check is bounded by a timeout so a stalled termination request cannot
	// block reporting; errors are logged and reporting proceeds regardless.
	if s.stuckPodChecker != nil && !wf.ExecutionStatus().IsInFinalState() {
		s.checkStuckPods(wf)
	}

	// Save this Workflow to the database.
	err = s.pipelineClient.ReportWorkflow(wf)
	retry := util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT)

	// Failure
	if err != nil && retry {
		return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Syncing Workflow (%v): transient failure: %v", name, err)
	}

	if err != nil && !retry {
		return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Syncing Workflow (%v): permanent failure: %v", name, err)
	}

	// Success
	log.WithFields(log.Fields{
		"Workflow": name,
	}).Infof("Syncing Workflow (%v): success, processing complete.", name)
	return nil
}

func (s *WorkflowSaver) checkStuckPods(wf util.ExecutionSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), s.stuckPodCheckTimeout)
	defer cancel()
	if err := s.stuckPodChecker.CheckAndTerminate(ctx, wf.ExecutionObjectMeta()); err != nil {
		log.Warnf("Workflow (%v): error checking for stuck pods: %v", wf.ExecutionName(), err)
	}
}
