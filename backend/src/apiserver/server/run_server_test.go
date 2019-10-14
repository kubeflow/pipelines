package server

import (
	"context"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestCreateRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	runDetail, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRuntimeWorkflow := testWorkflow.DeepCopy()
	expectedRuntimeWorkflow.Spec.Arguments.Parameters = []v1alpha1.Parameter{
		{Name: "param1", Value: util.StringPointer("world")}}
	expectedRuntimeWorkflow.Labels = map[string]string{util.LabelKeyWorkflowRunId: "123e4567-e89b-12d3-a456-426655440000"}
	expectedRuntimeWorkflow.Spec.ServiceAccountName = "pipeline-runner"
	expectedRunDetail := api.RunDetail{
		Run: &api.Run{
			Id:           "123e4567-e89b-12d3-a456-426655440000",
			Name:         "123",
			StorageState: api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:  &timestamp.Timestamp{},
			FinishedAt:   &timestamp.Timestamp{},
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: testWorkflow.ToStringForStore(),
				Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
			},
			ResourceReferences: []*api.ResourceReference{
				{
					Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
					Name: "123", Relationship: api.Relationship_OWNER,
				},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: util.NewWorkflow(expectedRuntimeWorkflow).ToStringForStore(),
		},
	}
	assert.Equal(t, expectedRunDetail, *runDetail)
}

func TestListRun(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	_, err := server.CreateRun(nil, &api.CreateRunRequest{Run: run})
	assert.Nil(t, err)

	expectedRun := &api.Run{
		Id:           "123e4567-e89b-12d3-a456-426655440000",
		Name:         "123",
		StorageState: api.Run_STORAGESTATE_AVAILABLE,
		CreatedAt:    &timestamp.Timestamp{Seconds: 2},
		ScheduledAt:  &timestamp.Timestamp{},
		FinishedAt:   &timestamp.Timestamp{},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Name: "123", Relationship: api.Relationship_OWNER,
			},
		},
	}
	listRunsResponse, err := server.ListRuns(nil, &api.ListRunsRequest{})
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	assert.Equal(t, expectedRun, listRunsResponse.Runs[0])
}

func TestValidateCreateRunRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_WithPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_EmptyName(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The run name is empty")
}

func TestValidateCreateRunRequest_NoExperiment(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: nil,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.Nil(t, err)
}

func TestValidateCreateRunRequest_EmptyPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
	}
	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(
		t, err.Error(), "Neither pipeline spec nor pipeline version is valid")
}

func TestValidateCreateRunRequest_TooMuchParameters(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunServer(manager)

	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	run := &api.Run{
		Name:               "123",
		ResourceReferences: validReference,
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       params,
		},
	}

	err := server.validateCreateRunRequest(&api.CreateRunRequest{Run: run})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestReportRunMetrics_RunNotFound(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, _ := initWithOneTimeRun(t)
	defer clientManager.Close()
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

	clientManager, resourceManager, runDetails := initWithOneTimeRun(t)
	defer clientManager.Close()
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
		RunId:   runDetails.UUID,
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
		RunId: runDetails.UUID,
	})
	assert.Nil(t, err)
	assert.Equal(t, []*api.RunMetric{metric}, run.GetRun().GetMetrics())
}

func TestReportRunMetrics_PartialFailures(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager, resourceManager, runDetail := initWithOneTimeRun(t)
	defer clientManager.Close()
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
		RunId:   runDetail.UUID,
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
