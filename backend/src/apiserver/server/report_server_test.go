package server

import (
	"encoding/json"
	"testing"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestValidateReportWorkflowRequest(t *testing.T) {
	// Name
	workflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}
	marshalledWorkflow, _ := json.Marshal(workflow)
	generatedWorkflow, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.Nil(t, err)
	assert.Equal(t, *util.NewWorkflow(workflow), *generatedWorkflow)
}

func TestValidateReportWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: "WRONG WORKFLOW"})
	assert.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportWorkflowRequest_MissingField(t *testing.T) {
	// Name
	workflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}
	marshalledWorkflow, _ := json.Marshal(workflow)
	_, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}

	marshalledWorkflow, _ = json.Marshal(workflow)
	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Owner
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	}

	marshalledWorkflow, _ = json.Marshal(workflow)
	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a valid owner")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}

	marshalledWorkflow, _ = json.Marshal(workflow)
	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}
