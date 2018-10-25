package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var validReference = []*api.ResourceReference{
	{
		Key: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT, Id: "123"},
		Relationship: api.Relationship_OWNER,
	},
}

var validRun = &api.Run{
	Name:               "123",
	PipelineSpec:       &api.PipelineSpec{PipelineId: "123"},
	ResourceReferences: validReference,
}

func TestValidateRunResourceReference(t *testing.T) {
	assert.Nil(t, ValidateRunResourceReference(validReference))
}

func TestValidateRunResourceReference_MoreThanOneRef(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "456"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateRunResourceReference(references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "more resource references than expected")
}

func TestValidateRunResourceReference_UnexpectedType(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateRunResourceReference(references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestValidateRunResourceReference_EmptyID(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateRunResourceReference(references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Resource ID is empty")
}

func TestValidateRunResourceReference_UnexpectedRelationship(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
	}
	err := ValidateRunResourceReference(references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected relationship for the experiment")
}

func TestValidateCreateRunRequest(t *testing.T) {
	err := ValidateCreateRunRequest(&api.CreateRunRequest{Run: validRun})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_EmptyName(t *testing.T) {
	var run = &api.Run{
		PipelineSpec:       &api.PipelineSpec{PipelineId: "123"},
		ResourceReferences: validReference,
	}

	err := ValidateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The run name is empty")
}

func TestValidateCreateRunRequest_EmptyPipelineSpec(t *testing.T) {
	var run = &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
	}

	err := ValidateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a pipeline ID or workflow manifest")
}

func TestValidateCreateRunRequest_TooMuchParameters(t *testing.T) {
	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	var run = &api.Run{
		Name:               "123",
		PipelineSpec:       &api.PipelineSpec{PipelineId: "123", Parameters: params},
		ResourceReferences: validReference,
	}

	err := ValidateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestReportRunMetrics_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	runServer := RunServer{resourceManager: resourceManager}
	_, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId: "1",
	})
	AssertUserError(t, err, codes.NotFound)
}

func TestReportRunMetrics_Succeed(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	workflow, _ := json.Marshal(v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "run1",
			Namespace:         "n1",
			UID:               "1",
			CreationTimestamp: v1.Time{Time: time.Unix(1, 0)},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	reportServer := ReportServer{resourceManager: resourceManager}
	_, err := reportServer.ReportWorkflow(context.Background(), &api.ReportWorkflowRequest{
		Workflow: string(workflow),
	})
	assert.Nil(t, err, "Unexpected error: %v", err)
	runServer := RunServer{resourceManager: resourceManager}

	metric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
	}
	response, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId:   "1",
		Metrics: []*api.RunMetric{metric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   metric.Name,
				MetricNodeId: metric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
		},
	}
	assert.Equal(t, expectedResponse, response)

	run, err := runServer.GetRun(context.Background(), &api.GetRunRequest{
		RunId: "1",
	})
	assert.Nil(t, err)
	assert.Equal(t, []*api.RunMetric{metric}, run.GetRun().GetMetrics())
}

func TestReportRunMetrics_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	workflow, _ := json.Marshal(v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "run1",
			Namespace:         "n1",
			UID:               "1",
			CreationTimestamp: v1.Time{Time: time.Unix(1, 0)},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	reportServer := ReportServer{resourceManager: resourceManager}
	_, err := reportServer.ReportWorkflow(context.Background(), &api.ReportWorkflowRequest{
		Workflow: string(workflow),
	})
	assert.Nil(t, err, "Unexpected error: %v", err)
	runServer := RunServer{resourceManager: resourceManager}

	validMetric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
	}
	invalidNameMetric := &api.RunMetric{
		Name:   "$metric-1",
		NodeId: "node-1",
	}
	invalidNodeIDMetric := &api.RunMetric{
		Name: "metric-1",
	}
	response, err := runServer.ReportRunMetrics(context.Background(), &api.ReportRunMetricsRequest{
		RunId:   "1",
		Metrics: []*api.RunMetric{validMetric, invalidNameMetric, invalidNodeIDMetric},
	})
	assert.Nil(t, err)
	expectedResponse := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{
			{
				MetricName:   validMetric.Name,
				MetricNodeId: validMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_OK,
			},
			{
				MetricName:   invalidNameMetric.Name,
				MetricNodeId: invalidNameMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
			{
				MetricName:   invalidNodeIDMetric.Name,
				MetricNodeId: invalidNodeIDMetric.NodeId,
				Status:       api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT,
			},
		},
	}
	// Message fields, which are not reliable, are ignored from the test.
	for _, result := range response.Results {
		result.Message = ""
	}
	assert.Equal(t, expectedResponse, response)
}
