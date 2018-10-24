package cmd

import (
	"fmt"
	"strings"
	"testing"

	client "github.com/googleprivate/ml/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestGetPipeline(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--no-color",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--no-color", "-o", "json",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--no-color",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetPipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestListPipeline(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list", "--no-color",
		"--max-items", "1"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list", "--no-color",
		"--max-items", "INVALID_MAX_ITEMS"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid argument \"INVALID_MAX_ITEMS\"")
}

func TestListPipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "list", "--no-color", "INVALID_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestDeletePipeline(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "--no-color",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestDeletePipelineClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "--no-color",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestDeletePipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "delete", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestGetTemplate(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "--no-color",
		fmt.Sprintf("%v", client.PipelineForDefaultTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "--no-color",
		fmt.Sprintf("%v", client.PipelineForClientErrorTest)})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetTemplateInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "get-manifest", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestCreatePipeline(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "--no-color",
		client.PipelineValidURL})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
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
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "--no-color",
		client.PipelineInvalidURL})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid URL format")
}

func TestCreatePipelineInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "create", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'url' argument")
}
