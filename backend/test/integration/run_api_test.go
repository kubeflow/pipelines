package integration

import (
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	experimentparams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runparams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type RunApiTestSuite struct {
	suite.Suite
	namespace            string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
}

type RunResourceReferenceSorter []*run_model.APIResourceReference

func (r RunResourceReferenceSorter) Len() int           { return len(r) }
func (r RunResourceReferenceSorter) Less(i, j int) bool { return r[i].Name < r[j].Name }
func (r RunResourceReferenceSorter) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// Check the namespace have ML pipeline installed and ready
func (s *RunApiTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	var err error
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
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

	s.cleanUp()
}

func (s *RunApiTestSuite) TestRunApis() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload a pipeline version YAML under helloWorldPipeline ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(helloWorldPipeline.ID),
		})
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "hello world experiment"}
	helloWorldExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world run by specifying pipeline version ID ---------- */
	createRunRequest := &runparams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Name: helloWorldExperiment.Name, Relationship: run_model.APIRelationshipOWNER},
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: helloWorldPipelineVersion.ID},
				Relationship: run_model.APIRelationshipCREATOR},
		},
	}}
	helloWorldRunDetail, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, _, err = s.runClient.Get(&runparams.GetRunParams{RunID: helloWorldRunDetail.Run.ID})
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Create a new argument parameter experiment ---------- */
	createExperimentRequest := &experimentparams.CreateExperimentParams{Body: &experiment_model.APIExperiment{Name: "argument parameter experiment"}}
	argParamsExperiment, err := s.experimentClient.Create(createExperimentRequest)
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter run by uploading workflow manifest ---------- */
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createRunRequest = &runparams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: argParamsExperiment.ID},
				Relationship: run_model.APIRelationshipOWNER},
		},
	}}
	argParamsRunDetail, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkArgParamsRunDetail(t, argParamsRunDetail, argParamsExperiment.ID, argParamsExperiment.Name)

	/* ---------- List all the runs. Both runs should be returned ---------- */
	runs, totalSize, _, err := s.runClient.List(&runparams.ListRunsParams{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, totalSize)

	/* ---------- List the runs, paginated, sorted by creation time ---------- */
	runs, totalSize, nextPageToken, err := s.runClient.List(
		&runparams.ListRunsParams{PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	/* TODO(issues/1762): fix the following flaky assertion. */
	/* assert.Equal(t, "hello world", runs[0].Name) */
	runs, totalSize, _, err = s.runClient.List(&runparams.ListRunsParams{
		PageSize: util.Int32Pointer(1), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	/* TODO(issues/1762): fix the following flaky assertion. */
	/* assert.Equal(t, "argument parameter", runs[0].Name) */

	/* ---------- List the runs, paginated, sort by name ---------- */
	runs, totalSize, nextPageToken, err = s.runClient.List(&runparams.ListRunsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", runs[0].Name)
	runs, totalSize, _, err = s.runClient.List(&runparams.ListRunsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name"), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)

	/* ---------- List the runs, sort by unsupported field ---------- */
	_, _, _, err = s.runClient.List(&runparams.ListRunsParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	assert.NotNil(t, err)

	/* ---------- List runs for hello world experiment. One run should be returned ---------- */
	runs, totalSize, _, err = s.runClient.List(&runparams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)

	/* ---------- Archive a run ------------*/
	err = s.runClient.Archive(&runparams.ArchiveRunParams{
		ID: helloWorldRunDetail.Run.ID,
	})

	/* ---------- List runs for hello world experiment. The same run should still be returned, but should be archived ---------- */
	runs, totalSize, _, err = s.runClient.List(&runparams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)
	assert.Equal(t, string(runs[0].StorageState), api.Run_STORAGESTATE_ARCHIVED.String())

	/* ---------- Upload long-running pipeline YAML ---------- */
	longRunningPipeline, err := s.pipelineUploadClient.UploadFile("../resources/long-running.yaml", uploadParams.NewUploadPipelineParamsWithTimeout(350))
	assert.Nil(t, err)

	/* ---------- Upload a long-running pipeline version YAML under longRunningPipeline ---------- */
	time.Sleep(1 * time.Second)
	longRunningPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/long-running.yaml", &uploadParams.UploadPipelineVersionParams{
		Name:       util.StringPointer("long-running-version"),
		Pipelineid: util.StringPointer(longRunningPipeline.ID),
	})
	assert.Nil(t, err)

	/* ---------- Create a new long-running run by specifying pipeline ID ---------- */
	createLongRunningRunRequest := &runparams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "long running",
		Description: "this pipeline will run long enough for us to manually terminate it before it finishes",
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: run_model.APIRelationshipOWNER},
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: longRunningPipelineVersion.ID},
				Relationship: run_model.APIRelationshipCREATOR},
		},
	}}
	longRunningRunDetail, _, err := s.runClient.Create(createLongRunningRunRequest)
	assert.Nil(t, err)

	/* ---------- Terminate the long-running run ------------*/
	err = s.runClient.Terminate(&runparams.TerminateRunParams{
		RunID: longRunningRunDetail.Run.ID,
	})

	/* ---------- Get long-running run ---------- */
	longRunningRunDetail, _, err = s.runClient.Get(&runparams.GetRunParams{RunID: longRunningRunDetail.Run.ID})
	assert.Nil(t, err)
	s.checkTerminatedRunDetail(t, longRunningRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, longRunningPipelineVersion.ID, longRunningPipelineVersion.Name)
}

func (s *RunApiTestSuite) checkTerminatedRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "wait-awhile")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "wait-awhile")

	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "long running",
		Description:    "this pipeline will run long enough for us to manually terminate it before it finishes",
		Status:         "Terminating",
		ServiceAccount: "pipeline-runner",
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: run_model.APIRelationshipCREATOR},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	// Need to sort resource references before equality check as the order is non-deterministic
	sort.Sort(RunResourceReferenceSorter(runDetail.Run.ResourceReferences))
	sort.Sort(RunResourceReferenceSorter(expectedRun.ResourceReferences))
	assert.Equal(t, expectedRun, runDetail.Run)
}

func (s *RunApiTestSuite) checkHelloWorldRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "whalesay")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "whalesay")

	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "hello world",
		Description:    "this is hello world",
		Status:         runDetail.Run.Status,
		ServiceAccount: "pipeline-runner",
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: run_model.APIRelationshipCREATOR,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	// Need to sort resource references before equality check as the order is non-deterministic
	sort.Sort(RunResourceReferenceSorter(runDetail.Run.ResourceReferences))
	sort.Sort(RunResourceReferenceSorter(expectedRun.ResourceReferences))
	assert.Equal(t, expectedRun, runDetail.Run)
}

func (s *RunApiTestSuite) checkArgParamsRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string) {
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "arguments-parameters-")
	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "argument parameter",
		Description:    "this is argument parameter",
		Status:         runDetail.Run.Status,
		ServiceAccount: "pipeline-runner",
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	assert.Equal(t, expectedRun, runDetail.Run)
}

func TestRunApi(t *testing.T) {
	suite.Run(t, new(RunApiTestSuite))
}

func (s *RunApiTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *RunApiTestSuite) cleanUp() {
	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllRuns(s.runClient, s.T())
}
