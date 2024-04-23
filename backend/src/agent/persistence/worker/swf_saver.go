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
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// ScheduledWorkflowSaver provides a function to persist a workflow to a database.
type ScheduledWorkflowSaver struct {
	client         client.ScheduledWorkflowClientInterface
	pipelineClient client.PipelineClientInterface
}

func NewScheduledWorkflowSaver(
	client client.ScheduledWorkflowClientInterface,
	pipelineClient client.PipelineClientInterface) *ScheduledWorkflowSaver {
	return &ScheduledWorkflowSaver{
		client:         client,
		pipelineClient: pipelineClient,
	}
}

func (c *ScheduledWorkflowSaver) Save(key string, namespace string, name string, nowEpoch int64) error {
	// Get the ScheduledWorkflow with this namespace/name
	swf, err := c.client.Get(namespace, name)
	isNotFound := util.HasCustomCode(err, util.CUSTOM_CODE_NOT_FOUND)
	if err != nil && isNotFound {
		// Permanent failure.
		// The ScheduledWorkflow may no longer exist, we stop processing and do not retry.
		return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"ScheduledWorkflow (%s) in work queue no longer exists: %v", key, err)
	}
	if err != nil && !isNotFound {
		// Transient failure, we will retry.
		return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"ScheduledWorkflow (%s): transient failure: %v", key, err)

	}

	// TODO: wait for officially update to v2beta1
	//       temporally hack this to v2beta1
	swf.APIVersion = "kubeflow.org/v2beta1"
	swf.Kind = "ScheduledWorkflow"
	// Save this Scheduled Workflow to the database.
	err = c.pipelineClient.ReportScheduledWorkflow(swf)
	retry := util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT)

	// Failure
	if err != nil && retry {
		return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Syncing ScheduledWorkflow (%v): transient failure: %v", name, err)
	}

	if err != nil && !retry {
		return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Syncing ScheduledWorkflow (%v): permanent failure: %v", name, err)
	}

	// Success
	log.WithFields(log.Fields{
		"ScheduledWorkflow": name,
	}).Infof("Syncing ScheduledWorkflow (%v): success, processing complete.", name)
	return nil
}
