package util

import (
	"github.com/argoproj/argo/workflow/common"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// GetRequirementForCompletedWorkflowOrFatal returns a label requirement indicating
// whether a workflow is completed.
func GetRequirementForCompletedWorkflowOrFatal(completed bool) *labels.Requirement {
	operator := selection.NotEquals
	if completed == true {
		operator = selection.Equals
	}
	req, err := labels.NewRequirement(common.LabelKeyCompleted, operator,
		[]string{"true"})
	if err != nil {
		log.Fatalf("Error while creating requirement: %s", err)
	}
	return req
}

// GetRequirementForScheduleNameOrFatal returns a label requirement for a specific
// ScheduledWorkflow name.
func GetRequirementForScheduleNameOrFatal(swf string) *labels.Requirement {
	req, err := labels.NewRequirement(commonutil.LabelKeyWorkflowScheduledWorkflowName, selection.Equals, []string{swf})
	if err != nil {
		log.Fatalf("Error while creating requirement: %s", err)
	}
	return req
}

// GetRequirementForScheduleNameOrFatal returns a label requirement for a minimum
// index of creation of a workflow (to avoid querying the whole list).
func GetRequirementForMinIndexOrFatal(minIndex int64) *labels.Requirement {
	req, err := labels.NewRequirement(commonutil.LabelKeyWorkflowIndex, selection.GreaterThan,
		[]string{commonutil.FormatInt64ForLabel(minIndex)})
	if err != nil {
		log.Fatalf("Error while creating requirement: %s", err)
	}
	return req
}
