package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/glog"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	recurringRunParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	apiServer "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata/testutils"
	"github.com/kubeflow/pipelines/backend/test/v2"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	pipelineClient       *apiServer.PipelineClient
	pipelineUploadClient *apiServer.PipelineUploadClient
	runClient            *apiServer.RunClient
	recurringRunClient   *apiServer.RecurringRunClient
	mlmdClient           pb.MetadataStoreServiceClient
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

func (s *CacheTestSuite) SetupSuite() {
	var err error
	s.mlmdClient, err = testutils.NewTestMlmdClient("127.0.0.1", metadata.DefaultConfig().Port)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), s.mlmdClient)
}

func (s *CacheTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace

	var newPipelineUploadClient func() (*apiServer.PipelineUploadClient, error)
	var newPipelineClient func() (*apiServer.PipelineClient, error)
	var newRunClient func() (*apiServer.RunClient, error)
	var newRecurringRunClient func() (*apiServer.RecurringRunClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newPipelineUploadClient = func() (*apiServer.PipelineUploadClient, error) {
			return apiServer.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*apiServer.PipelineClient, error) {
			return apiServer.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*apiServer.RunClient, error) {
			return apiServer.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
		newRecurringRunClient = func() (*apiServer.RecurringRunClient, error) {
			return apiServer.NewKubeflowInClusterRecurringRunClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newPipelineUploadClient = func() (*apiServer.PipelineUploadClient, error) {
			return apiServer.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*apiServer.PipelineClient, error) {
			return apiServer.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*apiServer.RunClient, error) {
			return apiServer.NewRunClient(clientConfig, *isDebugMode)
		}
		newRecurringRunClient = func() (*apiServer.RecurringRunClient, error) {
			return apiServer.NewRecurringRunClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = newPipelineClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = newRunClient()
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}
	s.recurringRunClient, err = newRecurringRunClient()
	if err != nil {
		glog.Exitf("Failed to get recurring run client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *CacheTestSuite) TestCacheRecurringRun() {
	t := s.T()

	pipelineVersion := s.preparePipeline()

	createRecurringRunRequest := &recurringRunParams.RecurringRunServiceCreateRecurringRunParams{RecurringRun: &recurring_run_model.V2beta1RecurringRun{
		DisplayName: "hello world",
		Description: "this is hello world",
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE.Pointer(),
		Trigger: &recurring_run_model.V2beta1Trigger{
			PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{
				IntervalSecond: 60,
			},
		},
		RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"message": "Hello world",
			},
		},
	}}
	helloWorldRecurringRun, err := s.recurringRunClient.Create(createRecurringRunRequest)
	require.NoError(t, err)
	require.NotNil(t, helloWorldRecurringRun)

	var allRuns []*run_model.V2beta1Run
	require.Eventually(s.T(), func() bool {
		allRuns, err = s.runClient.ListAll(runParams.NewRunServiceListRunsParams(), 10)
		if err != nil {
			return false
		}

		if len(allRuns) >= 2 {
			for _, run := range allRuns {
				if run.State != run_model.V2beta1RuntimeStateSUCCEEDED.Pointer() {
					return false
				}
			}
			return true
		}

		return false
	}, 4*time.Minute, 5*time.Second)

	contextsFilterQuery := fmt.Sprintf("name = '%s'", allRuns[1].RunID)

	contexts, err := s.mlmdClient.GetContexts(context.Background(), &pb.GetContextsRequest{
		Options: &pb.ListOperationOptions{
			FilterQuery: &contextsFilterQuery,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, contexts)

	executionsByContext, err := s.mlmdClient.GetExecutionsByContext(context.Background(), &pb.GetExecutionsByContextRequest{
		ContextId: contexts.Contexts[0].Id,
	})

	require.NoError(t, err)
	require.NotNil(t, executionsByContext)
	require.NotEmpty(t, executionsByContext.Executions)

	var containerExecution *pb.Execution
	for _, execution := range executionsByContext.Executions {
		if metadata.ExecutionType(execution.GetType()) == metadata.ContainerExecutionTypeName {
			containerExecution = execution
			break
		}
	}
	require.NotNil(t, containerExecution)

	if *cacheEnabled {
		require.Equal(t, pb.Execution_CACHED.Enum().String(), containerExecution.LastKnownState.String())
	} else {
		require.Equal(t, pb.Execution_COMPLETE.Enum().String(), containerExecution.LastKnownState.String())
	}
}

func (s *CacheTestSuite) TestCacheSingleRun() {
	t := s.T()

	pipelineVersion := s.preparePipeline()

	pipelineRunDetail, err := s.createRun(pipelineVersion)
	require.NoError(t, err)

	// Create the second run
	pipelineRunDetail, err = s.createRun(pipelineVersion)
	require.NoError(t, err)
	require.NotNil(t, pipelineRunDetail)

	contextsFilterQuery := fmt.Sprintf("name = '%s'", pipelineRunDetail.RunID)

	contexts, err := s.mlmdClient.GetContexts(context.Background(), &pb.GetContextsRequest{
		Options: &pb.ListOperationOptions{
			FilterQuery: &contextsFilterQuery,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, contexts)

	executionsByContext, err := s.mlmdClient.GetExecutionsByContext(context.Background(), &pb.GetExecutionsByContextRequest{
		ContextId: contexts.Contexts[0].Id,
	})

	require.NoError(t, err)
	require.NotNil(t, executionsByContext)
	require.NotEmpty(t, executionsByContext.Executions)

	var containerExecution *pb.Execution
	for _, execution := range executionsByContext.Executions {
		if metadata.ExecutionType(execution.GetType()) == metadata.ContainerExecutionTypeName {
			containerExecution = execution
			break
		}
	}
	require.NotNil(t, containerExecution)

	if *cacheEnabled {
		require.Equal(t, pb.Execution_CACHED.Enum().String(), containerExecution.LastKnownState.String())
	} else {
		require.Equal(t, pb.Execution_COMPLETE.Enum().String(), containerExecution.LastKnownState.String())
	}
}

func (s *CacheTestSuite) createRun(pipelineVersion *pipeline_upload_model.V2beta1PipelineVersion) (*run_model.V2beta1Run, error) {
	createRunRequest := &runParams.RunServiceCreateRunParams{Run: &run_model.V2beta1Run{
		DisplayName: "hello-world",
		Description: "this is hello-world",
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"message": "Hello world",
			},
		},
	}}
	pipelineRunDetail, err := s.runClient.Create(createRunRequest)
	require.NoError(s.T(), err)

	expectedState := run_model.V2beta1RuntimeStateSUCCEEDED
	require.Eventually(s.T(), func() bool {
		pipelineRunDetail, err = s.runClient.Get(&runParams.RunServiceGetRunParams{RunID: pipelineRunDetail.RunID})

		s.T().Logf("Pipeline %v state: %v", pipelineRunDetail.RunID, pipelineRunDetail.State)

		return err == nil && *pipelineRunDetail.State == expectedState
	}, 2*time.Minute, 10*time.Second)

	return pipelineRunDetail, err
}

func (s *CacheTestSuite) preparePipeline() *pipeline_upload_model.V2beta1PipelineVersion {
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world-with-returning-component.yaml", uploadParams.NewUploadPipelineParams())
	require.NoError(s.T(), err)

	time.Sleep(1 * time.Second)
	pipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world-with-returning-component.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(pipeline.PipelineID),
		})
	require.NoError(s.T(), err)

	return pipelineVersion
}

func (s *CacheTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *CacheTestSuite) cleanUp() {
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllRecurringRuns(s.recurringRunClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}
