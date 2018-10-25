package server

import (
	"encoding/json"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestReportWorkflow(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	reportServer := NewReportServer(resourceManager)
	gvk := schema.GroupVersionKind{
		Group:   swfapi.SchemeGroupVersion.Group,
		Version: swfapi.SchemeGroupVersion.Version,
		Kind:    "ScheduledWorkflow",
	}

	workflow := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-name",
			Namespace: "default",
			UID:       "123",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					UID:        "123",
				},
			},
		},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		}}
	workflowBytes, _ := json.Marshal(workflow)

	_, err := reportServer.ReportWorkflow(nil, &api.ReportWorkflowRequest{
		Workflow: string(workflowBytes),
	})
	assert.Nil(t, err)
	run, err := resourceManager.GetRun("123")
	assert.Nil(t, err)
	assert.NotNil(t, run)
}

func TestReportWorkflow_ValidationFailed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	reportServer := NewReportServer(resourceManager)

	workflow := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			UID:       "123",
		},
	}
	workflowBytes, _ := json.Marshal(workflow)

	_, err := reportServer.ReportWorkflow(nil, &api.ReportWorkflowRequest{
		Workflow: string(workflowBytes),
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

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
	workflow := util.NewWorkflow(&workflowapi.Workflow{
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
	})
	_, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	workflow = util.NewWorkflow(&workflowapi.Workflow{
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
	})

	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	workflow = util.NewWorkflow(&workflowapi.Workflow{
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
	})

	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestValidateReportScheduledWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: "WRONG_SCHEDULED_WORKFLOW"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportScheduledWorkflowRequest_MissingField(t *testing.T) {
	// Name
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})

	_, err := ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
		},
	})

	_, err = ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
		},
	})

	_, err = ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}
