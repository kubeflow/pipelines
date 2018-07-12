package worker

import (
	"github.com/googleprivate/ml/backend/src/agent/persistence/client"
	"github.com/googleprivate/ml/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// WorkflowSaver provides a function to persist a workflow to a database.
type WorkflowSaver struct {
	client         client.WorkflowClientInterface
	pipelineClient client.PipelineClientInterface
}

func NewWorkflowSaver(client client.WorkflowClientInterface,
	pipelineClient client.PipelineClientInterface) *WorkflowSaver {
	return &WorkflowSaver{
		client:         client,
		pipelineClient: pipelineClient,
	}
}

func (s *WorkflowSaver) Save(key string, namespace string, name string, nowEpoch int64) error {
	// Get the Workflow with this namespace/name
	swf, err := s.client.Get(namespace, name)
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

	// Save this Scheduled Workflow to the database.
	err = s.pipelineClient.ReportWorkflow(swf)
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
