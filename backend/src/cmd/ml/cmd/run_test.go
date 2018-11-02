package cmd

import (
	"fmt"
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestGetRun(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "get", "--no-color",
		fmt.Sprintf("%v", client.RunForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
pipeline_runtime: {}
run:
  created_at: "1970-01-01T00:00:00.000Z"
  id: RUN_DEFAULT
  metrics: []
  name: RUN_NAME
  resource_references: null
  scheduled_at: "0001-01-01T00:00:00.000Z"

workflow:
  metadata:
    creationTimestamp: null
    name: MY_NAME
    namespace: MY_NAMESPACE
  spec:
    arguments: {}
    entrypoint: ""
    templates: null
  status:
    finishedAt: null
    startedAt: null
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestGetRunClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "get", "--no-color",
		fmt.Sprintf("%v", client.RunForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetRunInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "get", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestListRun(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "list", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
- created_at: "1970-01-01T00:00:00.000Z"
  id: "100"
  metrics: []
  name: MY_FIRST_RUN
  resource_references: null
  scheduled_at: "0001-01-01T00:00:00.000Z"
- created_at: "1970-01-01T00:00:00.000Z"
  id: "101"
  metrics: []
  name: MY_SECOND_RUN
  resource_references: null
  scheduled_at: "0001-01-01T00:00:00.000Z"
- created_at: "1970-01-01T00:00:00.000Z"
  id: "102"
  metrics: []
  name: MY_THIRD_RUN
  resource_references: null
  scheduled_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListRunMaxItems(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "list", "--no-color",
		"--max-items", "1"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
- created_at: "1970-01-01T00:00:00.000Z"
  id: "100"
  metrics: []
  name: MY_FIRST_RUN
  resource_references: null
  scheduled_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListRunInvalidMaxItems(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "list", "--no-color",
		"--max-items", "INVALID_MAX_ITEMS"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid argument \"INVALID_MAX_ITEMS\"")
}

func TestListRunInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"run", "list", "--no-color", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}
