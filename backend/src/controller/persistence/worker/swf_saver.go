package worker

import (
	"github.com/googleprivate/ml/backend/src/controller/persistence/client"
	"github.com/googleprivate/ml/backend/src/util"
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
