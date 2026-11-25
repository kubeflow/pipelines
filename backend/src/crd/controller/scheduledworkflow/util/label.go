package util

import (
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	wraperror "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// GetRequirementForCompletedWorkflowOrFatal returns a label requirement indicating
// whether a workflow is completed.
func GetRequirementForCompletedWorkflowOrFatal(completed bool) *labels.Requirement {
	operator := selection.NotEquals
	if completed {
		operator = selection.Equals
	}
	req, err := labels.NewRequirement(common.LabelKeyCompleted, operator,
		[]string{"true"})
	if err != nil {
		log.Fatalf("Error while creating requirement: %s", err)
	}
	return req
}

// GetRequirementForScheduleName returns a label requirement for a specific
// ScheduledWorkflow name. The name comes from a user-created object and may not
// be a valid label value (e.g. longer than 63 characters), so an error is
// returned instead of terminating the controller.
func GetRequirementForScheduleName(swf string) (*labels.Requirement, error) {
	req, err := labels.NewRequirement(commonutil.LabelKeyWorkflowScheduledWorkflowName, selection.Equals, []string{swf})
	if err != nil {
		return nil, wraperror.Wrap(err, "Error while creating requirement")
	}
	return req, nil
}

// GetRequirementForMinIndexOrFatal returns a label requirement for a minimum
// index of creation of a workflow (to avoid querying the whole list).
func GetRequirementForMinIndexOrFatal(minIndex int64) *labels.Requirement {
	req, err := labels.NewRequirement(commonutil.LabelKeyWorkflowIndex, selection.GreaterThan,
		[]string{commonutil.FormatInt64ForLabel(minIndex)})
	if err != nil {
		log.Fatalf("Error while creating requirement: %s", err)
	}
	return req
}
