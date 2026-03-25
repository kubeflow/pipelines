package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

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
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/v2"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"

	"github.com/golang/glog"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type CacheTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	pipelineClient       *apiServer.PipelineClient
	pipelineUploadClient apiServer.PipelineUploadInterface
	runClient            *apiServer.RunClient
	recurringRunClient   *apiServer.RecurringRunClient
	mlmdClient           pb.MetadataStoreServiceClient
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

func (s *CacheTestSuite) SetupSuite() {
	var err error
	s.mlmdClient, err = testutils.NewTestMlmdClient("127.0.0.1", metadata.GetMetadataConfig().Port, *config.TLSEnabled, *config.CaCertPath)
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
	s.namespace = *config.Namespace

	var newPipelineClient func() (*apiServer.PipelineClient, error)
	var newRunClient func() (*apiServer.RunClient, error)
	var newRecurringRunClient func() (*apiServer.RecurringRunClient, error)

	var tlsCfg *tls.Config
	var err error
	if *config.TLSEnabled {
		tlsCfg, err = test.GetTLSConfig(*config.CaCertPath)
		if err != nil {
			glog.Exitf("Failed to get TLS config. Error: %s", err.Error())
		}
	}

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newPipelineClient = func() (*apiServer.PipelineClient, error) {
			return apiServer.NewKubeflowInClusterPipelineClient(s.namespace, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiServer.RunClient, error) {
			return apiServer.NewKubeflowInClusterRunClient(s.namespace, *config.DebugMode, tlsCfg)
		}
		newRecurringRunClient = func() (*apiServer.RecurringRunClient, error) {
			return apiServer.NewKubeflowInClusterRecurringRunClient(s.namespace, *config.DebugMode, tlsCfg)
		}
	} else {
		clientConfig := test.GetClientConfig(*config.Namespace)

		newPipelineClient = func() (*apiServer.PipelineClient, error) {
			return apiServer.NewPipelineClient(clientConfig, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiServer.RunClient, error) {
			return apiServer.NewRunClient(clientConfig, *config.DebugMode, tlsCfg)
		}
		newRecurringRunClient = func() (*apiServer.RecurringRunClient, error) {
			return apiServer.NewRecurringRunClient(clientConfig, *config.DebugMode, tlsCfg)
		}
	}

	s.pipelineUploadClient, err = test.GetPipelineUploadClient(
		*uploadPipelinesWithKubernetes,
		*isKubeflowMode,
		*config.DebugMode,
		s.namespace,
		test.GetClientConfig(s.namespace),
		tlsCfg,
	)
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
				if *run.State != *run_model.V2beta1RuntimeStateSUCCEEDED.Pointer() {
					return false
				}
			}
			return true
		}

		return false
	}, 4*time.Minute, 5*time.Second)

	state := s.getContainerExecutionState(t, allRuns[1].RunID)
	if *cacheEnabled {
		require.Equal(t, pb.Execution_CACHED, state)
	} else {
		require.Equal(t, pb.Execution_COMPLETE, state)
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

	state := s.getContainerExecutionState(t, pipelineRunDetail.RunID)
	if *cacheEnabled {
		require.Equal(t, pb.Execution_CACHED, state)
	} else {
		require.Equal(t, pb.Execution_COMPLETE, state)
	}
}

// Test that a pipeline using a PVC with the same name across runs hits the cache on the second run.
func (s *CacheTestSuite) TestCacheSingleRunWithPVC_SameName_Caches() {
	t := s.T()

	if !*cacheEnabled {
		t.Skip("Skipping PVC cache test: cache is disabled")

		return
	}

	pvcPipelinePath := "../resources/pvc-mount.yaml"

	// Create a small PVC up-front so the pipeline can mount it by name.
	restCfg, err := util.GetKubernetesConfig()
	require.NoError(t, err)
	clientset, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err)

	pvcName := fmt.Sprintf("test-cache-pvc-%d", time.Now().UnixNano())
	storageClass := "standard"
	qty := k8sres.MustParse("5Mi")
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: qty},
			},
			StorageClassName: &storageClass,
		},
	}
	_, err = clientset.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	require.NoError(t, err)
	defer func() {
		_ = clientset.CoreV1().PersistentVolumeClaims(s.namespace).Delete(context.Background(), pvcName, metav1.DeleteOptions{})
	}()

	// Upload pipeline and create a version
	pipeline, err := s.pipelineUploadClient.UploadFile(pvcPipelinePath, uploadParams.NewUploadPipelineParams())
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	time.Sleep(1 * time.Second)
	pipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		pvcPipelinePath,
		&uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("pvc-cache-version"),
			Pipelineid: util.StringPointer(pipeline.PipelineID),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, pipelineVersion)

	// First run with fixed PVC name
	run1, err := s.createRunWithParams(pipelineVersion, map[string]interface{}{"pvc_name": pvcName})
	require.NoError(t, err)
	require.NotNil(t, run1)

	// Second run with the same PVC name should hit cache when enabled
	run2, err := s.createRunWithParams(pipelineVersion, map[string]interface{}{"pvc_name": pvcName})
	require.NoError(t, err)
	require.NotNil(t, run2)

	state := s.getContainerExecutionState(t, run2.RunID)
	require.Equal(t, pb.Execution_CACHED, state)

	// Third run with a different PVC name should not hit cache.
	otherPVCName := fmt.Sprintf("%s-alt", pvcName)
	// Create the alternate PVC so the pipeline can mount it
	pvcAlt := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: otherPVCName},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources:        v1.VolumeResourceRequirements{Requests: v1.ResourceList{v1.ResourceStorage: qty}},
			StorageClassName: &storageClass,
		},
	}
	_, err = clientset.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.Background(), pvcAlt, metav1.CreateOptions{})
	require.NoError(t, err)
	defer func() {
		_ = clientset.CoreV1().PersistentVolumeClaims(s.namespace).Delete(context.Background(), otherPVCName, metav1.DeleteOptions{})
	}()

	run3, err := s.createRunWithParams(pipelineVersion, map[string]interface{}{"pvc_name": otherPVCName})
	require.NoError(t, err)
	require.NotNil(t, run3)

	state = s.getContainerExecutionState(t, run3.RunID)
	// With a different PVC, do not expect cache hit
	require.Equal(t, pb.Execution_COMPLETE, state)
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

		if err == nil {
			s.T().Logf("Pipeline %v state: %v", pipelineRunDetail.RunID, *pipelineRunDetail.State)
		} else {
			s.T().Logf("Pipeline %v state: %v", pipelineRunDetail.RunID, err.Error())
		}

		return err == nil && *pipelineRunDetail.State == expectedState
	}, 4*time.Minute, 10*time.Second)

	return pipelineRunDetail, err
}

func (s *CacheTestSuite) createRunWithParams(pipelineVersion *pipeline_upload_model.V2beta1PipelineVersion, params map[string]interface{}) (*run_model.V2beta1Run, error) {
	createRunRequest := &runParams.RunServiceCreateRunParams{Run: &run_model.V2beta1Run{
		DisplayName: "pvc-cache",
		Description: "pvc cache test",
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{Parameters: params},
	}}
	pipelineRunDetail, err := s.runClient.Create(createRunRequest)
	require.NoError(s.T(), err)

	expectedState := run_model.V2beta1RuntimeStateSUCCEEDED
	require.Eventually(s.T(), func() bool {
		pipelineRunDetail, err = s.runClient.Get(&runParams.RunServiceGetRunParams{RunID: pipelineRunDetail.RunID})
		if err == nil {
			s.T().Logf("PVC pipeline %v state: %v", pipelineRunDetail.RunID, *pipelineRunDetail.State)
		} else {
			s.T().Logf("PVC pipeline %v state: %v", pipelineRunDetail.RunID, err.Error())
		}
		return err == nil && *pipelineRunDetail.State == expectedState
	}, 4*time.Minute, 10*time.Second)

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

// getContainerExecutionState fetches the container execution state for a given run ID.
func (s *CacheTestSuite) getContainerExecutionState(t *testing.T, runID string) pb.Execution_State {
	contextsFilterQuery := fmt.Sprintf("name = '%s'", runID)

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

	for _, execution := range executionsByContext.Executions {
		if metadata.ExecutionType(execution.GetType()) == metadata.ContainerExecutionTypeName {
			return execution.GetLastKnownState()
		}
	}
	t.Fatalf("no container execution found for run %s", runID)
	return pb.Execution_UNKNOWN
}
