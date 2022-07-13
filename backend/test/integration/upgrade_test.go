package integration

import (
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	experimentParams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	jobparams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	pipelineParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	pipelinetemplate "github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
)

// Methods are organized into two types: "prepare" and "verify".
// "prepare" tests setup resources before upgrade
// "verify" tests verifies resources are expected after upgrade
type UpgradeTests struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	jobClient            *api_server.JobClient
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTests))
}

func (s *UpgradeTests) TestPrepare() {
	t := s.T()

	test.DeleteAllJobs(s.jobClient, s.resourceNamespace, t)
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, t)
	test.DeleteAllPipelines(s.pipelineClient, t)
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, t)

	s.PrepareExperiments()
	s.PreparePipelines()
	s.PrepareRuns()
	s.PrepareJobs()
}

func (s *UpgradeTests) TestVerify() {
	s.VerifyExperiments()
	s.VerifyPipelines()
	s.VerifyRuns()
	s.VerifyJobs()
}

// Check the namespace have ML job installed and ready
func (s *UpgradeTests) SetupSuite() {
	// Integration tests also run these tests to first ensure they work, so that
	// when integration tests pass and upgrade tests fail, we know for sure
	// upgrade process went wrong somehow.
	if !(*runIntegrationTests || *runUpgradeTests) {
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
}

func (s *UpgradeTests) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			t := s.T()
			// Clean up after the suite to unblock other tests. (Not needed for upgrade
			// tests because it needs changes in prepare tests to persist and verified
			// later.)
			test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, t)
			test.DeleteAllPipelines(s.pipelineClient, t)
			test.DeleteAllRuns(s.runClient, s.resourceNamespace, t)
			test.DeleteAllJobs(s.jobClient, s.resourceNamespace, t)
		}
	}
}

func (s *UpgradeTests) PrepareExperiments() {
	t := s.T()

	/* ---------- Create a new experiment ---------- */
	experiment := test.GetExperiment("training", "my first experiment", s.resourceNamespace)
	_, err := s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)

	/* ---------- Create a few more new experiment ---------- */
	// This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = test.GetExperiment("prediction", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)

	time.Sleep(1 * time.Second)
	experiment = test.GetExperiment("moonshot", "my third experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyExperiments() {
	t := s.T()

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, _, _, err := test.ListExperiment(
		s.experimentClient,
		&experimentParams.ListExperimentParams{SortBy: util.StringPointer("created_at")},
		s.resourceNamespace)
	require.Nil(t, err)
	// after upgrade, default experiment may be inserted, but the oldest 3
	// experiments should be the ones created in this test
	require.True(t, len(experiments) >= 3)

	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "my first experiment", experiments[0].Description)
	assert.NotEmpty(t, experiments[0].ID)
	assert.NotEmpty(t, experiments[0].CreatedAt)

	assert.Equal(t, "prediction", experiments[1].Name)
	assert.Equal(t, "my second experiment", experiments[1].Description)
	assert.NotEmpty(t, experiments[1].ID)
	assert.NotEmpty(t, experiments[1].CreatedAt)

	assert.Equal(t, "moonshot", experiments[2].Name)
	assert.Equal(t, "my third experiment", experiments[2].Description)
	assert.NotEmpty(t, experiments[2].ID)
	assert.NotEmpty(t, experiments[2].CreatedAt)
}

// TODO(jingzhang36): prepare pipeline versions.
func (s *UpgradeTests) PreparePipelines() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload pipelines YAML ---------- */
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&pipelineParams.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{Name: "sequential", URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}}})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.Name)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&pipelineParams.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip"}}})
	require.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)

	time.Sleep(1 * time.Second)
}

func (s *UpgradeTests) VerifyPipelines() {
	t := s.T()

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	pipelines, _, _, err := s.pipelineClient.List(
		&pipelineParams.ListPipelinesParams{SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	// During upgrade, default pipelines may be installed, so we only verify the
	// 4 oldest pipelines here.
	assert.True(t, len(pipelines) >= 4)
	assert.Equal(t, "arguments-parameters.yaml", pipelines[0].Name)
	assert.Equal(t, "sequential", pipelines[1].Name)
	assert.Equal(t, "zip-arguments-parameters", pipelines[2].Name)
	assert.Equal(t, "arguments.pipeline.zip", pipelines[3].Name)

	verifyPipeline(t, pipelines[0])

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetTemplate(&pipelineParams.GetTemplateParams{ID: pipelines[0].ID})
	require.Nil(t, err)
	bytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected, err := pipelinetemplate.New(bytes)
	require.Nil(t, err)
	assert.Equal(t, expected, template)
}

func (s *UpgradeTests) PrepareRuns() {
	t := s.T()

	helloWorldPipeline := s.getHelloWorldPipeline(true)
	helloWorldExperiment := s.getHelloWorldExperiment(true)
	if helloWorldExperiment == nil {
		helloWorldExperiment = s.createHelloWorldExperiment()
	}

	hello2 := s.getHelloWorldExperiment(true)
	require.Equal(t, hello2, helloWorldExperiment)

	/* ---------- Create a new hello world run by specifying pipeline ID ---------- */
	createRunRequest := &runParams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID: helloWorldPipeline.ID,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Name: helloWorldExperiment.Name, Relationship: run_model.APIRelationshipOWNER},
		},
	}}
	_, _, err := s.runClient.Create(createRunRequest)
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyRuns() {
	t := s.T()

	/* ---------- List the runs, sorted by creation time ---------- */
	runs, _, _, err := test.ListRuns(
		s.runClient,
		&runParams.ListRunsParams{SortBy: util.StringPointer("created_at")},
		s.resourceNamespace)
	glog.Infof("Logs: line 324: %#v", runs)
	require.Nil(t, err)
	require.True(t, len(runs) >= 1)
	require.Equal(t, "hello world", runs[0].Name)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: runs[0].ID})
	require.Nil(t, err)
	checkHelloWorldRunDetail(t, helloWorldRunDetail)
}

func (s *UpgradeTests) PrepareJobs() {
	t := s.T()

	pipeline := s.getHelloWorldPipeline(true)
	experiment := s.getHelloWorldExperiment(true)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID: pipeline.ID,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		NoCatchup:      true,
	}}
	_, err := s.jobClient.Create(createJobRequest)
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyJobs() {
	t := s.T()

	pipeline := s.getHelloWorldPipeline(false)
	experiment := s.getHelloWorldExperiment(false)

	/* ---------- Get hello world job ---------- */
	jobs, _, _, err := test.ListAllJobs(s.jobClient, s.resourceNamespace)
	require.Nil(t, err)
	require.Len(t, jobs, 1)
	job := jobs[0]

	// Check workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "whalesay")
	expectedJob := &job_model.APIJob{
		ID:          job.ID,
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID:       pipeline.ID,
			PipelineName:     "hello-world.yaml",
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experiment.ID},
				Name: experiment.Name, Relationship: job_model.APIRelationshipOWNER,
			},
			{Key: &job_model.APIResourceKey{ID: pipeline.ID, Type: job_model.APIResourceTypePIPELINEVERSION},
				Name: "hello-world.yaml", Relationship: job_model.APIRelationshipCREATOR,
			},
		},
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		MaxConcurrency: 10,
		NoCatchup:      true,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
		Trigger:        &job_model.APITrigger{},
	}

	sort.Sort(JobResourceReferenceSorter(job.ResourceReferences))
	sort.Sort(JobResourceReferenceSorter(expectedJob.ResourceReferences))
	assert.Equal(t, expectedJob, job)
}

func checkHelloWorldRunDetail(t *testing.T, runDetail *run_model.APIRunDetail) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "whalesay")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "whalesay")

	expectedExperimentID := test.GetExperimentIDFromAPIResourceReferences(runDetail.Run.ResourceReferences)
	require.NotEmpty(t, expectedExperimentID)

	expectedRun := &run_model.APIRun{
		ID:          runDetail.Run.ID,
		Name:        "hello world",
		Description: "this is hello world",
		Status:      runDetail.Run.Status,
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID:       runDetail.Run.PipelineSpec.PipelineID,
			PipelineName:     "hello-world.yaml",
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: expectedExperimentID},
				Name: "hello world experiment", Relationship: run_model.APIRelationshipOWNER,
			},
			{Key: &run_model.APIResourceKey{ID: runDetail.Run.PipelineSpec.PipelineID, Type: run_model.APIResourceTypePIPELINEVERSION},
				Name: "hello-world.yaml", Relationship: run_model.APIRelationshipCREATOR,
			},
		},
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		CreatedAt:      runDetail.Run.CreatedAt,
		ScheduledAt:    runDetail.Run.ScheduledAt,
		FinishedAt:     runDetail.Run.FinishedAt,
	}
	sort.Sort(RunResourceReferenceSorter(expectedRun.ResourceReferences))
	sort.Sort(RunResourceReferenceSorter(runDetail.Run.ResourceReferences))
	assert.Equal(t, expectedRun, runDetail.Run)
}

func (s *UpgradeTests) createHelloWorldExperiment() *experiment_model.APIExperiment {
	t := s.T()

	experiment := test.GetExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experimentParams.CreateExperimentParams{Body: experiment})
	require.Nil(t, err)

	return helloWorldExperiment
}

func (s *UpgradeTests) getHelloWorldExperiment(createIfNotExist bool) *experiment_model.APIExperiment {
	t := s.T()

	experiments, _, _, err := test.ListExperiment(
		s.experimentClient,
		&experimentParams.ListExperimentParams{
			PageSize: util.Int32Pointer(1000),
		},
		s.resourceNamespace)
	require.Nil(t, err)
	var helloWorldExperiment *experiment_model.APIExperiment
	for _, experiment := range experiments {
		if experiment.Name == "hello world experiment" {
			helloWorldExperiment = experiment
		}
	}

	if helloWorldExperiment == nil && createIfNotExist {
		return s.createHelloWorldExperiment()
	}

	return helloWorldExperiment
}

func (s *UpgradeTests) getHelloWorldPipeline(createIfNotExist bool) *pipeline_model.APIPipeline {
	t := s.T()

	pipelines, err := s.pipelineClient.ListAll(&pipelineParams.ListPipelinesParams{}, 1000)
	require.Nil(t, err)
	var helloWorldPipeline *pipeline_model.APIPipeline
	for _, pipeline := range pipelines {
		if pipeline.Name == "hello-world.yaml" {
			helloWorldPipeline = pipeline
		}
	}

	if helloWorldPipeline == nil && createIfNotExist {
		return s.createHelloWorldPipeline()
	}

	return helloWorldPipeline
}

func (s *UpgradeTests) createHelloWorldPipeline() *pipeline_model.APIPipeline {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	uploadedPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)

	helloWorldPipeline, err := s.pipelineClient.Get(&pipelineParams.GetPipelineParams{ID: uploadedPipeline.ID})
	require.Nil(t, err)

	return helloWorldPipeline
}
