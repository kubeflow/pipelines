package test

import (
	"testing"

	"encoding/json"

	"github.com/golang/glog"
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

	// Create pipelines
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "create",
		"https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// List pipeline
	rootCmd, factory := GetRealRootCommand()
	args = []string{"pipeline", "list"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Convert and assert result
	pipelines, err := toPipelines(factory.Result())
	assert.Nil(t, err)
	assert.True(t, len(pipelines) >= 1)
}

func (c *CLIIntegrationTest) TestPipelineListFailureInvalidArgument() {
	t := c.T()
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "list", "askjdfskldjf"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
}

func (c *CLIIntegrationTest) TestPipelineUploadSuccess() {
	t := c.T()
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "upload", "--name", myUploadedPipeline,
		"./resources/hello-world.yaml"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)
}

func (c *CLIIntegrationTest) TestPipelineCreateGetDeleteSuccess() {
	t := c.T()
	rootCmd, factory := GetRealRootCommand()

	// Create pipeline
	args := []string{"pipeline", "create",
		"https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Get ID
	pipelineID, err := toPipelineID(factory.Result())
	assert.Nil(t, err)

	// Get pipeline
	args = []string{"pipeline", "get", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Get manifest
	args = []string{"pipeline", "get-manifest", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	// Delete pipeline
	args = []string{"pipeline", "delete", pipelineID}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err = rootCmd.Command().ExecuteC()
	assert.Nil(t, err)
}

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
	args := []string{"pipeline", "delete", pipeline.ID}
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
