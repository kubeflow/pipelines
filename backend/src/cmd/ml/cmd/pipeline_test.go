package cmd

import (
	"fmt"
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestGetPipeline(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--id",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
created_at: "1970-01-01T00:00:00.000Z"
description: PIPELINE_DESCRIPTION
id: PIPELINE_ID_10
name: PIPELINE_NAME
parameters:
- name: PARAM_NAME
  value: PARAM_VALUE
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestGetPipelineJson(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "-o", "json", "--id",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
{
  "created_at": "1970-01-01T00:00:00.000Z",
  "description": "PIPELINE_DESCRIPTION",
  "id": "PIPELINE_ID_10",
  "name": "PIPELINE_NAME",
  "parameters": [
    {
      "name": "PARAM_NAME",
      "value": "PARAM_VALUE"
    }
  ]
}
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestGetPipelineClientError(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--id",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetPipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestListPipeline(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
- created_at: "1970-01-01T00:00:00.000Z"
  description: PIPELINE_DESCRIPTION
  id: PIPELINE_ID_100
  name: PIPELINE_NAME
  parameters:
  - name: PARAM_NAME
    value: PARAM_VALUE
- created_at: "1970-01-01T00:00:00.000Z"
  description: PIPELINE_DESCRIPTION
  id: PIPELINE_ID_101
  name: PIPELINE_NAME
  parameters:
  - name: PARAM_NAME
    value: PARAM_VALUE
- created_at: "1970-01-01T00:00:00.000Z"
  description: PIPELINE_DESCRIPTION
  id: PIPELINE_ID_102
  name: PIPELINE_NAME
  parameters:
  - name: PARAM_NAME
    value: PARAM_VALUE
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListPipelineMaxItems(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list",
		"--max-items", "1"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
- created_at: "1970-01-01T00:00:00.000Z"
  description: PIPELINE_DESCRIPTION
  id: PIPELINE_ID_100
  name: PIPELINE_NAME
  parameters:
  - name: PARAM_NAME
    value: PARAM_VALUE
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListPipelineInvalidMaxItems(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list",
		"--max-items", "INVALID_MAX_ITEMS"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid argument \"INVALID_MAX_ITEMS\"")
}

func TestListPipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list", "INVALID_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestDeletePipeline(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "--id",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := ""
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestDeletePipelineClientError(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "--id",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestDeletePipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestGetTemplate(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "--id",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
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

func TestGetTemplateClientError(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "--id",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetTemplateInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestCreatePipeline(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "--url",
		client.PipelineValidURL})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
created_at: "1970-01-01T00:00:00.000Z"
description: PIPELINE_DESCRIPTION
id: foo.yaml
name: PIPELINE_NAME
parameters:
- name: PARAM_NAME
  value: PARAM_VALUE
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
}

func TestCreatePipelineInvalidUrlFormat(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "--url",
		client.PipelineInvalidURL})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid URL format")
}

func TestCreatePipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}
