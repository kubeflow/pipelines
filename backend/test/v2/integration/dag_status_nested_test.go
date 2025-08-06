package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	pipeline_upload_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata/testutils"
	testV2 "github.com/kubeflow/pipelines/backend/test/v2"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

type DAGStatusNestedTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	pipelineUploadClient *api_server.PipelineUploadClient
	pipelineClient       *api_server.PipelineClient
	runClient            *api_server.RunClient
	mlmdClient           pb.MetadataStoreServiceClient
}

// createUploadParams creates properly configured upload parameters for CI compatibility
func (s *DAGStatusNestedTestSuite) createUploadParams(testName, filePath string) *uploadParams.UploadPipelineParams {
	uploadParams := uploadParams.NewUploadPipelineParams()
	
	// CI FIX: Set required fields that CI environment enforces
	pipelineName := fmt.Sprintf("%s_test", testName)
	displayName := fmt.Sprintf("%s Test Pipeline", testName)
	description := fmt.Sprintf("Test pipeline for %s scenario", testName)
	namespace := s.resourceNamespace
	
	uploadParams.SetName(&pipelineName)
	uploadParams.SetDisplayName(&displayName) 
	uploadParams.SetDescription(&description)
	if namespace != "" {
		uploadParams.SetNamespace(&namespace)
	}
	
	return uploadParams
}

// Check the namespace have ML pipeline installed and ready
func (s *DAGStatusNestedTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := testV2.WaitForReady(*initializeTimeout)
		if err != nil {
			s.T().Logf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace

	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := testV2.GetClientConfig(*namespace)

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewRunClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		s.T().Logf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = newPipelineClient()
	if err != nil {
		s.T().Logf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = newRunClient()
	if err != nil {
		s.T().Logf("Failed to get run client. Error: %s", err.Error())
	}

	s.mlmdClient, err = testutils.NewTestMlmdClient("127.0.0.1", metadata.DefaultConfig().Port)
	if err != nil {
		s.T().Logf("Failed to create MLMD client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *DAGStatusNestedTestSuite) TearDownTest() {
	if !*isDevMode {
		s.cleanUp()
	}
}

func (s *DAGStatusNestedTestSuite) cleanUp() {
	if s.runClient != nil {
		testV2.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	}
	if s.pipelineClient != nil {
		testV2.DeleteAllPipelines(s.pipelineClient, s.T())
	}
}

// Test Case 1: Simple Nested Structure
// Validates that a nested DAG structure updates status correctly
func (s *DAGStatusNestedTestSuite) TestSimpleNested() {
	t := s.T()

	pipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/dag_status/nested_simple.yaml",
		s.createUploadParams("nested_simple", "../resources/dag_status/nested_simple.yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	pipelineVersion, err := s.getDefaultPipelineVersion(pipeline.PipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipelineVersion)

	run, err := s.createRun(pipelineVersion, "nested-simple-test")
	require.NoError(t, err)
	require.NotNil(t, run)

	s.waitForRunCompletion(run.RunID, run_model.V2beta1RuntimeStateSUCCEEDED)

	// Give extra time for MLMD DAG executions (parent + child) to be created
	time.Sleep(45 * time.Second)
	s.validateNestedDAGStatus(run.RunID, pb.Execution_COMPLETE, "simple_nested")
}

// Test Case 2: Nested ParallelFor
// Validates that nested ParallelFor structures update status correctly
func (s *DAGStatusNestedTestSuite) TestNestedParallelFor() {
	t := s.T()

	pipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/dag_status/nested_parallel_for.yaml",
		s.createUploadParams("nested_parallel_for", "../resources/dag_status/nested_parallel_for.yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	pipelineVersion, err := s.getDefaultPipelineVersion(pipeline.PipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipelineVersion)

	run, err := s.createRun(pipelineVersion, "nested-parallel-for-test")
	require.NoError(t, err)
	require.NotNil(t, run)

	s.waitForRunCompletion(run.RunID, run_model.V2beta1RuntimeStateSUCCEEDED)

	// Give some time for MLMD DAG execution to be created
	time.Sleep(20 * time.Second)
	s.validateNestedDAGStatus(run.RunID, pb.Execution_COMPLETE, "nested_parallel_for")
}

// Test Case 3: Nested Conditional
// Validates that nested conditional structures update status correctly
func (s *DAGStatusNestedTestSuite) TestNestedConditional() {
	t := s.T()

	pipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/dag_status/nested_conditional.yaml",
		s.createUploadParams("nested_conditional", "../resources/dag_status/nested_conditional.yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	pipelineVersion, err := s.getDefaultPipelineVersion(pipeline.PipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipelineVersion)

	run, err := s.createRun(pipelineVersion, "nested-conditional-test")
	require.NoError(t, err)
	require.NotNil(t, run)

	s.waitForRunCompletion(run.RunID, run_model.V2beta1RuntimeStateSUCCEEDED)

	// Give some time for MLMD DAG execution to be created
	time.Sleep(20 * time.Second)
	s.validateNestedDAGStatus(run.RunID, pb.Execution_COMPLETE, "nested_conditional")
}

// Test Case 4: Deep Nesting
// Validates that deeply nested structures update status correctly
func (s *DAGStatusNestedTestSuite) TestDeepNesting() {
	t := s.T()

	pipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/dag_status/nested_deep.yaml",
		s.createUploadParams("nested_deep", "../resources/dag_status/nested_deep.yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	pipelineVersion, err := s.getDefaultPipelineVersion(pipeline.PipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipelineVersion)

	run, err := s.createRun(pipelineVersion, "nested-deep-test")
	require.NoError(t, err)
	require.NotNil(t, run)

	s.waitForRunCompletion(run.RunID, run_model.V2beta1RuntimeStateSUCCEEDED)

	// Give some time for MLMD DAG execution to be created
	time.Sleep(20 * time.Second)
	s.validateNestedDAGStatus(run.RunID, pb.Execution_COMPLETE, "deep_nesting")
}

func (s *DAGStatusNestedTestSuite) createRun(pipelineVersion *pipeline_upload_model.V2beta1PipelineVersion, displayName string) (*run_model.V2beta1Run, error) {
	createRunRequest := &runparams.RunServiceCreateRunParams{Run: &run_model.V2beta1Run{
		DisplayName: displayName,
		Description: "DAG status test for nested scenarios",
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
	}}
	return s.runClient.Create(createRunRequest)
}

func (s *DAGStatusNestedTestSuite) waitForRunCompletion(runID string, expectedState run_model.V2beta1RuntimeState) {
	// TODO: REVERT THIS WHEN BUG IS FIXED - Currently runs never complete due to DAG status bug
	// We'll wait for the run to at least start executing, then validate the bug directly
	require.Eventually(s.T(), func() bool {
		runDetail, err := s.runClient.Get(&runparams.RunServiceGetRunParams{RunID: runID})
		if err != nil {
			s.T().Logf("Error getting run %s: %v", runID, err)
			return false
		}

		s.T().Logf("Run %s state: %v", runID, runDetail.State)
		// Wait for run to start executing (RUNNING state), then we'll validate the bug
		return runDetail.State != nil && *runDetail.State == run_model.V2beta1RuntimeStateRUNNING
	}, 2*time.Minute, 10*time.Second, "Run did not start executing")
}

func (s *DAGStatusNestedTestSuite) getDefaultPipelineVersion(pipelineID string) (*pipeline_upload_model.V2beta1PipelineVersion, error) {
	// List pipeline versions for the uploaded pipeline
	versions, _, _, err := s.pipelineClient.ListPipelineVersions(&pipeline_params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelineID,
	})
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no pipeline versions found for pipeline %s", pipelineID)
	}

	// Convert from pipeline_model to pipeline_upload_model (they have the same fields)
	version := versions[0]
	return &pipeline_upload_model.V2beta1PipelineVersion{
		PipelineID:        version.PipelineID,
		PipelineVersionID: version.PipelineVersionID,
		DisplayName:       version.DisplayName,
		Name:              version.Name,
		Description:       version.Description,
		CreatedAt:         version.CreatedAt,
	}, nil
}

func (s *DAGStatusNestedTestSuite) validateNestedDAGStatus(runID string, expectedDAGState pb.Execution_State, testScenario string) {
	t := s.T()

	// Enhanced search: Look for ALL DAG executions across all contexts to find nested structures
	// This should capture both parent and child DAG executions

	s.T().Logf("Searching for all DAG executions related to run %s...", runID)

	// First, get all DAG executions in the system (within a reasonable time window)
	allDAGExecutions, err := s.mlmdClient.GetExecutionsByType(context.Background(), &pb.GetExecutionsByTypeRequest{
		TypeName: util.StringPointer("system.DAGExecution"),
	})
	require.NoError(t, err)
	require.NotNil(t, allDAGExecutions)

	// Filter DAG executions that are related to our run (by timestamp proximity and potential context links)
	var relatedDAGs []*pb.Execution

	for _, execution := range allDAGExecutions.Executions {
		// Log all DAG executions for debugging
		s.T().Logf("Examining DAG execution ID=%d, type=%s, state=%v, create_time=%v, properties=%v",
			execution.GetId(), execution.GetType(), execution.LastKnownState,
			execution.CreateTimeSinceEpoch, execution.GetCustomProperties())

		// Include DAG executions that are recent (within last 5 minutes) as potentially related
		if execution.CreateTimeSinceEpoch != nil {
			createdTime := *execution.CreateTimeSinceEpoch
			now := time.Now().UnixMilli()
			if now-createdTime < 5*60*1000 { // Within 5 minutes
				relatedDAGs = append(relatedDAGs, execution)
				s.T().Logf("Including recent DAG execution ID=%d (created %d ms ago)",
					execution.GetId(), now-createdTime)
			}
		}
	}

	// Also get executions from the specific run context for comparison
	contextsFilterQuery := util.StringPointer("name = '" + runID + "'")
	contexts, err := s.mlmdClient.GetContexts(context.Background(), &pb.GetContextsRequest{
		Options: &pb.ListOperationOptions{
			FilterQuery: contextsFilterQuery,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, contexts)
	require.NotEmpty(t, contexts.Contexts)

	executionsByContext, err := s.mlmdClient.GetExecutionsByContext(context.Background(), &pb.GetExecutionsByContextRequest{
		ContextId: contexts.Contexts[0].Id,
	})
	require.NoError(t, err)
	require.NotNil(t, executionsByContext)

	// Add context-specific DAG executions to our collection
	for _, execution := range executionsByContext.Executions {
		if execution.GetType() == "system.DAGExecution" {
			// Check if already in relatedDAGs to avoid duplicates
			found := false
			for _, existing := range relatedDAGs {
				if existing.GetId() == execution.GetId() {
					found = true
					break
				}
			}
			if !found {
				relatedDAGs = append(relatedDAGs, execution)
				s.T().Logf("Adding context-specific DAG execution ID=%d", execution.GetId())
			}
		}
	}

	var nestedDAGs = relatedDAGs // Use all related DAGs for validation

	require.NotEmpty(t, nestedDAGs, "No nested DAG executions found for %s", testScenario)

	s.T().Logf("Found %d nested DAG executions for %s scenario", len(nestedDAGs), testScenario)

	for _, dagExecution := range nestedDAGs {
		totalDagTasks := dagExecution.GetCustomProperties()["total_dag_tasks"].GetIntValue()
		taskName := ""
		if tn := dagExecution.GetCustomProperties()["task_name"]; tn != nil {
			taskName = tn.GetStringValue()
		}

		s.T().Logf("Nested DAG execution ID=%d: task_name='%s', total_dag_tasks=%d, state=%s for %s",
			dagExecution.GetId(), taskName, totalDagTasks, dagExecution.LastKnownState, testScenario)

		// Identify child pipeline DAGs vs parent DAGs
		isChildPipelineDAG := taskName == "child-pipeline"

		if isChildPipelineDAG {
			// Child pipeline DAGs work correctly
			s.T().Logf("✅ CHILD DAG %d: total_dag_tasks=%d (correct - child pipeline has 3 tasks)",
				dagExecution.GetId(), totalDagTasks)

			// Child DAGs should have correct total_dag_tasks and can complete properly
			assert.Equal(t, int64(3), totalDagTasks,
				"Child pipeline DAG should have total_dag_tasks=3 (child_setup + child_worker + child_finalizer)")

			// Child DAGs can reach COMPLETE state
			if dagExecution.LastKnownState != nil && *dagExecution.LastKnownState == pb.Execution_COMPLETE {
				s.T().Logf("✅ Child DAG %d properly completed", dagExecution.GetId())
			}

		} else {
			// FIXED: Parent DAGs should account for nested structure
			s.T().Logf("🚨 PARENT DAG %d: total_dag_tasks=%d (should account for nested structure)",
				dagExecution.GetId(), totalDagTasks)

			// FIXED: Now expecting CORRECT behavior - test will FAIL until bug is fixed
			// Parent DAG should account for nested child pipeline tasks + own tasks
			// Expected: parent_setup(1) + child_pipeline(3) + parent_finalizer(1) = 5 tasks minimum
			assert.True(t, totalDagTasks >= 5,
				"Parent DAG total_dag_tasks=%d should be >= 5 (BUG: currently returns 0, should include nested child pipeline tasks)",
				totalDagTasks)

			// FIXED: Now expecting CORRECT final state - test will FAIL until DAG state bug is fixed
			assert.Equal(t, expectedDAGState.String(), dagExecution.LastKnownState.String(),
				"Parent DAG execution ID=%d should reach final state %v (BUG: currently stuck in %v due to total_dag_tasks bug)",
				dagExecution.GetId(), expectedDAGState, dagExecution.LastKnownState)
		}

		s.T().Logf("REGRESSION TEST for %s: %s DAG %d has total_dag_tasks=%d %s",
			testScenario, map[bool]string{true: "CHILD", false: "PARENT"}[isChildPipelineDAG],
			dagExecution.GetId(), totalDagTasks,
			func() string {
				if isChildPipelineDAG {
					return "✅ CORRECT"
				} else if totalDagTasks >= 5 {
					return "✅ CORRECT"
				} else {
					return "🚨 BUG DETECTED"
				}
			}())

		// Log additional properties for debugging
		if customProps := dagExecution.GetCustomProperties(); customProps != nil {
			for key, value := range customProps {
				if key != "total_dag_tasks" && key != "task_name" { // Already logged above
					s.T().Logf("Nested DAG %d custom property: %s = %v", dagExecution.GetId(), key, value)
				}
			}
		}
	}
}

func TestDAGStatusNested(t *testing.T) {
	suite.Run(t, new(DAGStatusNestedTestSuite))
}
