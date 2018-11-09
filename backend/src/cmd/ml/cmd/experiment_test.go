package cmd

import (
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestCreateExperiment(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "create", "--name",
		client.ExperimentForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
created_at: "1970-01-01T00:00:00.000Z"
description: EXPERIMENT_DESCRIPTION
id: "500"
name: EXPERIMENT_DEFAULT
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestCreateExperimentClientError(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "create", "--name",
		client.ExperimentForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestCreateExperimentInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "create", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestGetExperiment(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "get", "--id",
		client.ExperimentForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
created_at: "1970-01-01T00:00:00.000Z"
description: EXPERIMENT_DESCRIPTION
id: EXPERIMENT_DEFAULT
name: EXPERIMENT_NAME
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestGetExperimentClientError(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "get", "--id",
		client.ExperimentForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetExperimentInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "get", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestListExperiment(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "list"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
- created_at: "1970-01-01T00:00:00.000Z"
  description: EXPERIMENT_DESCRIPTION
  id: "100"
  name: MY_FIRST_EXPERIMENT
- created_at: "1970-01-01T00:00:00.000Z"
  description: EXPERIMENT_DESCRIPTION
  id: "101"
  name: MY_SECOND_EXPERIMENT
- created_at: "1970-01-01T00:00:00.000Z"
  description: EXPERIMENT_DESCRIPTION
  id: "102"
  name: MY_THIRD_EXPERIMENT
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListExperimentMaxItems(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "list",
		"--max-items", "1"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
- created_at: "1970-01-01T00:00:00.000Z"
  description: EXPERIMENT_DESCRIPTION
  id: "100"
  name: MY_FIRST_EXPERIMENT
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListExperimentInvalidMaxItems(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "list",
		"--max-items", "INVALID_MAX_ITEMS"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid argument \"INVALID_MAX_ITEMS\"")
}

func TestListExperimentInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"experiment", "list", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}
