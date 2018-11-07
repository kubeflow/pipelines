package test

import (
	"testing"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/cmd/ml/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	defaultPageSize = int32(10)
)

func GetRealRootCommand() (*cmd.RootCommand, *cmd.ClientFactory) {
	clientFactory := cmd.NewClientFactory()
	rootCmd := cmd.NewRootCmd(clientFactory)
	rootCmd = cmd.CreateSubCommands(rootCmd, defaultPageSize)
	return rootCmd, clientFactory
}

type CLIIntegrationTest struct {
	suite.Suite
	namespace string
}

// Check the cluster namespace has Kubeflow pipelines installed and ready.
func (c *CLIIntegrationTest) SetupTest() {
	c.namespace = *namespace

	// Wait for the system to be ready.
	err := waitForReady(c.namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Cluster namespace '%s' is still not ready after timeout. Error: %s", c.namespace,
			err.Error())
	}
}

func (c *CLIIntegrationTest) TearDownTest() {
	// Nothing to do.
}

func (c *CLIIntegrationTest) TestPipelineListSuccess() {
	t := c.T()
	rootCmd, _ := GetRealRootCommand()
	args := []string{"pipeline", "list"}
	args = addCommonArgs(args, c.namespace)
	rootCmd.Command().SetArgs(args)
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)
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

func TestPipelineAPI(t *testing.T) {
	suite.Run(t, new(CLIIntegrationTest))
}

func addCommonArgs(args []string, namespace string) []string {
	args = append(args, "--debug", "--namespace", namespace)
	return args
}
