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

package worker

import (
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"time"
)

// WorkflowSaver provides a function to persist a workflow to a database.
type WorkflowSaver struct {
	client                        client.WorkflowClientInterface
	pipelineClient                client.PipelineClientInterface
	metricsReporter               *MetricsReporter
	ttlSecondsAfterWorkflowFinish int64
}

func NewWorkflowSaver(client client.WorkflowClientInterface,
		pipelineClient client.PipelineClientInterface, ttlSecondsAfterWorkflowFinish int64) *WorkflowSaver {
	return &WorkflowSaver{
		client:                        client,
		pipelineClient:                pipelineClient,
		metricsReporter:               NewMetricsReporter(pipelineClient),
		ttlSecondsAfterWorkflowFinish: ttlSecondsAfterWorkflowFinish,
	}
}

func (s *WorkflowSaver) Save(key string, namespace string, name string, nowEpoch int64) error {
	// Get the Workflow with this namespace/name
	wf, err := s.client.Get(namespace, name)
	isNotFound := util.HasCustomCode(err, util.CUSTOM_CODE_NOT_FOUND)
	if err != nil && isNotFound {
		// Permanent failure.
		// The Workflow may no longer exist, we stop processing and do not retry.
		return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Workflow (%s) in work queue no longer exists: %v", key, err)
	}
	if err != nil && !isNotFound {
		// Transient failure, we will retry.
		return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Workflow (%s): transient failure: %v", key, err)

	}
	if _, ok := wf.ObjectMeta.Labels[util.LabelKeyWorkflowRunId]; !ok {
		log.Infof("Skip syncing Workflow (%v): workflow does not have a Run ID label.", name)
		return nil
	}
	if wf.PersistedFinalState() && time.Now().Unix()-wf.FinishedAt() < s.ttlSecondsAfterWorkflowFinish {
		// Skip persisting the workflow if the workflow is finished
		// and the workflow hasn't being passing the TTL
		log.Infof("Skip syncing Workflow (%v): workflow marked as persisted.", name)
		return nil
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
	return s.metricsReporter.ReportMetrics(wf)
}
