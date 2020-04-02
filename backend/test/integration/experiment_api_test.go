package integration

import (
	"testing"

	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	jobParams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExperimentApiTest struct {
	suite.Suite
	namespace            string
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
	clientConfig := test.GetClientConfig(*namespace)
	var err error
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
	s.pipelineUploadClient, err = api_server.NewPipelineUploadClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = api_server.NewPipelineClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = api_server.NewRunClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}
	s.jobClient, err = api_server.NewJobClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *ExperimentApiTest) TestExperimentAPI() {
	t := s.T()

	/* ---------- Verify no experiment exist ---------- */
	experiments, totalSize, _, err := s.experimentClient.List(&params.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 0, totalSize)
	assert.True(t, len(experiments) == 0)

	/* ---------- Create a new experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "training", Description: "my first experiment"}
	trainingExperiment, err := s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
	expectedTrainingExperiment := &experiment_model.APIExperiment{
		ID: trainingExperiment.ID, Name: experiment.Name,
		Description: experiment.Description, CreatedAt: trainingExperiment.CreatedAt, StorageState: "STORAGESTATE_AVAILABLE"}
	assert.Equal(t, expectedTrainingExperiment, trainingExperiment)

	/* ---------- Create an experiment with same name. Should fail due to name uniqueness ---------- */
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{Body: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a new name")

	/* ---------- Create a few more new experiment ---------- */
	// 1 second interval. This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "prediction", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "moonshot", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Verify list experiments works ---------- */
	experiments, totalSize, nextPageToken, err := s.experimentClient.List(&params.ListExperimentParams{})
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
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List experiments sort by unsupported field. Should fail. ---------- */
	_, _, _, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	assert.NotNil(t, err)

	/* ---------- List experiments sorted by names descend order ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
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
	createRunRequest := &runParams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experiment.ID},
				Name: experiment.Name, Relationship: run_model.APIRelationshipOWNER},
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersion.ID},
				Relationship: run_model.APIRelationshipCREATOR},
		},
	}}
	run1, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	run2, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobParams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersion.ID},
				Relationship: job_model.APIRelationshipCREATOR},
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
	assert.Equal(t, experiment_model.ExperimentStorageState("STORAGESTATE_ARCHIVED"), experiment.StorageState)
	retrievedRun1, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: run1.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun1.Run.StorageState)
	retrievedRun2, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: run2.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun2.Run.StorageState)
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
	assert.Equal(t, experiment_model.ExperimentStorageState("STORAGESTATE_AVAILABLE"), experiment.StorageState)
	retrievedRun1, _, err = s.runClient.Get(&runParams.GetRunParams{RunID: run1.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun1.Run.StorageState)
	retrievedRun2, _, err = s.runClient.Get(&runParams.GetRunParams{RunID: run2.Run.ID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.RunStorageState("STORAGESTATE_ARCHIVED"), retrievedRun2.Run.StorageState)
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
	test.DeleteAllExperiments(s.experimentClient, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllRuns(s.runClient, s.T())
	test.DeleteAllJobs(s.jobClient, s.T())
}
