package integration

import (
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
	jobParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExperimentApiTest struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	jobClient            *api_server.JobClient
}

// Check the namespace have ML job installed and ready
func (s *ExperimentApiTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %v", err)
		}
	}

	s.namespace = *namespace

	var newExperimentClient func() (*api_server.ExperimentClient, error)
	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)
	var newJobClient func() (*api_server.JobClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewKubeflowInClusterExperimentClient(s.namespace, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
		newJobClient = func() (*api_server.JobClient, error) {
			return api_server.NewKubeflowInClusterJobClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewExperimentClient(clientConfig, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewRunClient(clientConfig, *isDebugMode)
		}
		newJobClient = func() (*api_server.JobClient, error) {
			return api_server.NewJobClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.experimentClient, err = newExperimentClient()
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
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
	s.jobClient, err = newJobClient()
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *ExperimentApiTest) TestExperimentAPI() {
	t := s.T()

	/* ---------- Verify no experiment exist ---------- */
	experiments, totalSize, _, err := test.ListAllExperiment(s.experimentClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 0, totalSize)
	assert.True(t, len(experiments) == 0)

	/* ---------- Create a new experiment ---------- */
	experiment := test.GetExperiment("training", "my first experiment", s.resourceNamespace)
	trainingExperiment, err := s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
	expectedTrainingExperiment := test.GetExperiment(experiment.Name, experiment.Description, s.resourceNamespace)
	expectedTrainingExperiment.ID = trainingExperiment.ID
	expectedTrainingExperiment.CreatedAt = trainingExperiment.CreatedAt
	expectedTrainingExperiment.StorageState = "STORAGESTATE_AVAILABLE"
	assert.Equal(t, expectedTrainingExperiment, trainingExperiment)

	/* ---------- Create an experiment with same name. Should fail due to name uniqueness ---------- */
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{Body: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a new name")

	/* ---------- Create a few more new experiment ---------- */
	// 1 second interval. This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = test.GetExperiment("prediction", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	time.Sleep(1 * time.Second)
	experiment = test.GetExperiment("moonshot", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Verify list experiments works ---------- */
	experiments, totalSize, nextPageToken, err := test.ListAllExperiment(s.experimentClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 3, len(experiments))
	for _, e := range experiments {
		// Sampling one of the experiments and verify the result is expected.
		if e.Name == "training" {
			assert.Equal(t, expectedTrainingExperiment, trainingExperiment)
		}
	}

	/* ---------- Verify list experiments sorted by names ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("name")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("name")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("created_at")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("created_at")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List experiments sort by unsupported field. Should fail. ---------- */
	_, _, _, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("unknownfield")},
		s.resourceNamespace)
	assert.NotNil(t, err)

	/* ---------- List experiments sorted by names descend order ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("name desc")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ListExperimentParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("name desc")},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get experiment works ---------- */
	experiment, err = s.experimentClient.Get(&params.GetExperimentParams{ID: trainingExperiment.ID})
	assert.Nil(t, err)
	assert.Equal(t, expectedTrainingExperiment, experiment)

	/* ---------- Create a pipeline version and two runs and two jobs -------------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
	pipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(pipeline.ID),
		})
	assert.Nil(t, err)
	createRunRequest := &runParams.CreateRunParams{Body: &run_model.V1beta1Run{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*run_model.V1beta1ResourceReference{
			{Key: &run_model.V1beta1ResourceKey{Type: run_model.V1beta1ResourceTypeEXPERIMENT, ID: experiment.ID},
				Name: experiment.Name, Relationship: run_model.V1beta1RelationshipOWNER},
			{Key: &run_model.V1beta1ResourceKey{Type: run_model.V1beta1ResourceTypePIPELINEVERSION, ID: pipelineVersion.ID},
				Relationship: run_model.V1beta1RelationshipCREATOR},
		},
	}}
	run1, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	run2, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobParams.CreateJobParams{Body: &job_model.V1beta1Job{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*job_model.V1beta1ResourceReference{
			{Key: &job_model.V1beta1ResourceKey{Type: job_model.V1beta1ResourceTypeEXPERIMENT, ID: experiment.ID},
				Relationship: job_model.V1beta1RelationshipOWNER},
			{Key: &job_model.V1beta1ResourceKey{Type: job_model.V1beta1ResourceTypePIPELINEVERSION, ID: pipelineVersion.ID},
				Relationship: job_model.V1beta1RelationshipCREATOR},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	job1, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	job2, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* ---------- Archive an experiment -----------------*/
	err = s.experimentClient.Archive(&params.ArchiveExperimentParams{ID: trainingExperiment.ID})

	/* ---------- Verify experiment and its runs ------- */
	experiment, err = s.experimentClient.Get(&params.GetExperimentParams{ID: trainingExperiment.ID})
	assert.Nil(t, err)
	assert.Equal(t, experiment_model.V1beta1ExperimentStorageState("STORAGESTATE_ARCHIVED"), experiment.StorageState)
	retrievedRun1, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: run1.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V1beta1RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun1.Run.StorageState)
	retrievedRun2, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: run2.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V1beta1RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun2.Run.StorageState)
	retrievedJob1, err := s.jobClient.Get(&jobParams.GetJobParams{ID: job1.ID})
	assert.Nil(t, err)
	assert.Equal(t, false, retrievedJob1.Enabled)
	retrievedJob2, err := s.jobClient.Get(&jobParams.GetJobParams{ID: job2.ID})
	assert.Nil(t, err)
	assert.Equal(t, false, retrievedJob2.Enabled)

	/* ---------- Unarchive an experiment -----------------*/
	err = s.experimentClient.Unarchive(&params.UnarchiveExperimentParams{ID: trainingExperiment.ID})

	/* ---------- Verify experiment and its runs and jobs --------- */
	experiment, err = s.experimentClient.Get(&params.GetExperimentParams{ID: trainingExperiment.ID})
	assert.Nil(t, err)
	assert.Equal(t, experiment_model.V1beta1ExperimentStorageState("STORAGESTATE_AVAILABLE"), experiment.StorageState)
	retrievedRun1, _, err = s.runClient.Get(&runParams.GetRunParams{RunID: run1.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V1beta1RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun1.Run.StorageState)
	retrievedRun2, _, err = s.runClient.Get(&runParams.GetRunParams{RunID: run2.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V1beta1RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun2.Run.StorageState)
	retrievedJob1, err = s.jobClient.Get(&jobParams.GetJobParams{ID: job1.ID})
	assert.Nil(t, err)
	assert.Equal(t, false, retrievedJob1.Enabled)
	retrievedJob2, err = s.jobClient.Get(&jobParams.GetJobParams{ID: job2.ID})
	assert.Nil(t, err)
	assert.Equal(t, false, retrievedJob2.Enabled)
}

func TestExperimentAPI(t *testing.T) {
	suite.Run(t, new(ExperimentApiTest))
}

func (s *ExperimentApiTest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *ExperimentApiTest) cleanUp() {
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllJobs(s.jobClient, s.resourceNamespace, s.T())
}
