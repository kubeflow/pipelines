package test

import (
	"testing"

	"encoding/json"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/src/cmd/ml/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	defaultPageSize    = int32(10)
	myUploadedPipeline = "my-uploaded-pipeline"
	// If true, run as a unit test using a fake client.
	// If false, run as an integration test calling the service.
	// This is useful to test locally before running an e2e test (which takes a while).
	// IMPORTANT: This should always be set to FALSE in the checked-in code.
	runAsUnitTest = false
)

func GetRealRootCommand() (*cmd.RootCommand, cmd.ClientFactoryInterface) {
	if runAsUnitTest {
		return cmd.GetFakeRootCommand()
	} else {
		clientFactory := cmd.NewClientFactoryWithByteBuffer()
		rootCmd := cmd.NewRootCmd(clientFactory)
		rootCmd = cmd.CreateSubCommands(rootCmd, defaultPageSize)
		return rootCmd, clientFactory
	}
}

type CLIIntegrationTest struct {
	suite.Suite
	namespace string
}

// Check the cluster namespace has Kubeflow pipelines installed and ready.
func (c *CLIIntegrationTest) SetupTest() {
	c.namespace = *namespace

	if !runAsUnitTest {
		// Wait for the system to be ready.
		err := waitForReady(c.namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Cluster namespace '%s' is still not ready after timeout. Error: %s", c.namespace,
				err.Error())
		}
	}
}

func (c *CLIIntegrationTest) TearDownTest() {
	err := c.deletePipelines()
	if err != nil {
		glog.Exitf("Failed to delete pipelines: %v", err)
	}
}

func (c *CLIIntegrationTest) TestPipelineListSuccess() {
	t := c.T()

	// Create pipeline
	_, err := c.createPipeline("TestPipelineListSuccess")
	assert.Nil(t, err)

	// List pipelines
	pipelines, err := c.listPipelines()
	assert.Nil(t, err)
	assert.True(t, len(pipelines) >= 1)
}

func (c *CLIIntegrationTest) TestPipelineListFailureInvalidArgument() {
	t := c.T()
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "list", "EXTRA_ARGUMENT"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
}

func (c *CLIIntegrationTest) TestPipelineUploadSuccess() {
	t := c.T()
	_, err := c.createPipeline("TestPipelineUploadSuccess")
	assert.Nil(t, err)
}

func (c *CLIIntegrationTest) TestPipelineCreateGetDeleteSuccess() {
	t := c.T()

	// Create pipeline
	pipelineID, err := c.createPipeline("TestPipelineCreateGetDeleteSuccess")
	assert.Nil(t, err)

	// Get pipeline
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "get", "--id", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Get manifest
	args = []string{"pipeline", "get-manifest", "--id", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Delete pipeline
	args = []string{"pipeline", "delete", "--id", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)
}

func (c *CLIIntegrationTest) TestExperimentListSuccess() {
	t := c.T()

	// Create experiment
	_, err := c.createExperiment("TestExperimentCreateGetSuccess")
	assert.Nil(t, err)

	// List experiments
	pipelines, err := c.listExperiments()
	assert.Nil(t, err)
	assert.True(t, len(pipelines) >= 1)
}

func (c *CLIIntegrationTest) TestExperimentListFailureInvalidArgument() {
	t := c.T()
	rootCmd, _ := GetRealRootCommand()
	args := []string{"experiment", "list", "EXTRA_ARGUMENT"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
}

func (c *CLIIntegrationTest) TestExperimentCreateGetSuccess() {
	t := c.T()

	// Create experiment
	experimentID, err := c.createExperiment("TestExperimentCreateGetSuccess")
	assert.Nil(t, err)

	// Get pipeline
	rootCmd, _ := GetRealRootCommand()
	args := []string{"experiment", "get", "--id", experimentID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)
}

func (c *CLIIntegrationTest) TestJobCreateGetEnableDisableDeleteSuccess() {
	t := c.T()

	// Create experiment
	experimentID, err := c.createExperiment("TestJobCreateGetEnableDisableDeleteSuccess")
	assert.Nil(t, err)

	// Create pipeline
	pipelineID, err := c.createPipeline("TestJobCreateGetEnableDisableDeleteSuccess")
	assert.Nil(t, err)

	// Create job
	jobID, err := c.createJob("TestJobCreateGetEnableDisableDeleteSuccess", pipelineID, experimentID)
	assert.Nil(t, err)

	rootCmd, _ := GetRealRootCommand()

	// Get job
	args := []string{"job", "get", "--id", jobID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Disable job
	args = []string{"job", "disable", "--id", jobID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Enable job
	args = []string{"job", "enable", "--id", jobID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Delete job
	args = []string{"job", "delete", "--id", jobID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

}

func (c *CLIIntegrationTest) TestJobListSuccess() {
	t := c.T()

	// Create experiment
	experimentID, err := c.createExperiment("TestJobListSuccess")
	assert.Nil(t, err)

	// Create pipeline
	_, err = c.createPipeline("TestJobListSuccess")
	assert.Nil(t, err)

	// Create job
	// TODO: add a job once issue is resolved: https://github.com/kubeflow/pipelines/issues/171
	// _, err = c.createJob("TestJobListSuccess", pipelineID, experimentID)
	// assert.Nil(t, err)

	// List jobs
	jobs, err := c.listJobsInExperiment(experimentID)
	assert.Nil(t, err)
	// TODO: uncomment this check once issue is resolve: https://github.com/kubeflow/pipelines/issues/171
	assert.True(t, len(jobs) >= 1)
}

// TODO: add tests for runs, including the creation of a run without a job.

func TestPipelineAPI(t *testing.T) {
	suite.Run(t, new(CLIIntegrationTest))
}

func addCommonArgs(args []string, namespace string) []string {
	args = append(args, "--debug", "--namespace", namespace, "-o", "json")
	return args
}

func toPipelineID(jsonPipeline string) (string, error) {
	var pipeline pipeline_model.APIPipeline
	err := json.Unmarshal([]byte(jsonPipeline), &pipeline)
	if err != nil {
		return "", err
	}
	return pipeline.ID, nil
}

func toPipelines(jsonPipelines string) ([]pipeline_model.APIPipeline, error) {
	var pipelines []pipeline_model.APIPipeline
	err := json.Unmarshal([]byte(jsonPipelines), &pipelines)
	if err != nil {
		return nil, err
	}
	return pipelines, nil
}

func toExperimentID(jsonExperiment string) (string, error) {
	var experiment experiment_model.APIExperiment
	err := json.Unmarshal([]byte(jsonExperiment), &experiment)
	if err != nil {
		return "", err
	}
	return experiment.ID, nil
}

func toJobID(jsonJob string) (string, error) {
	var job job_model.APIJob
	err := json.Unmarshal([]byte(jsonJob), &job)
	if err != nil {
		return "", err
	}
	return job.ID, nil
}

func toJobs(jsonJobs string) ([]job_model.APIJob, error) {
	var jobs []job_model.APIJob
	err := json.Unmarshal([]byte(jsonJobs), &jobs)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func toExperiments(jsonExperiments string) ([]experiment_model.APIExperiment, error) {
	var experiments []experiment_model.APIExperiment
	err := json.Unmarshal([]byte(jsonExperiments), &experiments)
	if err != nil {
		return nil, err
	}
	return experiments, nil
}

func (c *CLIIntegrationTest) deletePipelines() error {
	pipelines, err := c.listPipelines()
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		err = c.deletePipeline(pipeline)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CLIIntegrationTest) deletePipeline(pipeline pipeline_model.APIPipeline) error {
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "delete", "--id", pipeline.ID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	return err
}

func (c *CLIIntegrationTest) listPipelines() ([]pipeline_model.APIPipeline, error) {
	rootCmd, factory := GetRealRootCommand()
	args := []string{"pipeline", "list"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return nil, err
	}
	return toPipelines(factory.Result())
}

func (c *CLIIntegrationTest) listExperiments() ([]experiment_model.APIExperiment, error) {
	rootCmd, factory := GetRealRootCommand()
	args := []string{"experiment", "list"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return nil, err
	}
	return toExperiments(factory.Result())
}

func (c *CLIIntegrationTest) listJobsInExperiment(experimentID string) ([]job_model.APIJob, error) {
	rootCmd, factory := GetRealRootCommand()
	args := []string{"job", "list", "--experiment-id", experimentID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return nil, err
	}
	return toJobs(factory.Result())
}

func (c *CLIIntegrationTest) createPipeline(name string) (string, error) {
	rootCmd, factory := GetRealRootCommand()
	args := []string{"pipeline", "upload", "--name", name, "--file",
		"./resources/hello-world.yaml"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return "", err
	}
	return toPipelineID(factory.Result())
}

func (c *CLIIntegrationTest) createExperiment(name string) (string, error) {
	rootCmd, factory := GetRealRootCommand()

	// Create experiment
	args := []string{"experiment", "create", "--name",
		name}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return "", err
	}

	// Get ID
	return toExperimentID(factory.Result())
}

func (c *CLIIntegrationTest) createJob(name string, pipelineID string, experimentID string) (string, error) {
	rootCmd, factory := GetRealRootCommand()
	args := []string{"job", "create", "--name", name, "--pipeline-id", pipelineID, "--experiment-id",
		experimentID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	if err != nil {
		return "", err
	}
	return toJobID(factory.Result())
}
