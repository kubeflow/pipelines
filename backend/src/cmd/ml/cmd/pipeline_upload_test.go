package cmd

import (
	"fmt"
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestPipelineUpload(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--file",
		client.FileForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
created_at: "1970-01-01T00:00:00.000Z"
description: PIPELINE_DESCRIPTION
id: "500"
name: PIPELINE_NAME
parameters:
- name: PARAM_NAME
  value: PARAM_VALUE
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	fmt.Println(factory.Result())
}

func TestPipelineUploadJson(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "-o", "json", "--file",
		client.FileForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
{
  "created_at": "1970-01-01T00:00:00.000Z",
  "description": "PIPELINE_DESCRIPTION",
  "id": "500",
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
	fmt.Println(factory.Result())
}

func TestPipelineUploadNoFile(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
	fmt.Println(factory.Result())
}

func TestPipelineUploadClientError(t *testing.T) {
	rootCmd, factory := GetFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--file",
		client.FileForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
	fmt.Println(factory.Result())
}
