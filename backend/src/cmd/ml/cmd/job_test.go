package cmd

import (
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestCreateJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
created_at: "1970-01-01T00:00:00.000Z"
description: JOB_DESCRIPTION
id: "500"
name: JOB_DEFAULT
resource_references: null
updated_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestCreateJobClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForClientErrorTest, "--pipeline-id", "5"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestCreateJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "--no-color", "--pipeline-id", "5", "create"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'NAME' argument")
}

func TestCreateJobInvalidMaxConcurrency(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--max-concurrency", "0"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Flag 'max-concurrency' must be at least 1")
}

func TestCreateJobInvalidStartTime(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--start-time", "INVALID_TIME"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"Value 'INVALID_TIME' (flag 'start-time') is not a valid DateTime")
}

func TestCreateJobInvalidEndTime(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--end-time", "INVALID_TIME"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"Value 'INVALID_TIME' (flag 'end-time') is not a valid DateTime")
}

func TestCreateJobInvalidCron(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--cron", "INVALID_CRON"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"Value 'INVALID_CRON' (flag 'cron') is not a valid cron")
}

func TestCreateJobInvalidInterval(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--period", "INVALID_PERIOD"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"invalid argument \"INVALID_PERIOD\" for \"--period\" flag")
}

func TestCreateJobInvalidParameter(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "-p", "WRONG_PARAMETER"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"Parameter format is not valid. Expected: 'NAME=VALUE'")
}

func TestCreateJobBothCronAndInterval(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "create", "--no-color",
		client.JobForDefaultTest, "--pipeline-id", "5", "--cron", "0 30 * * * *",
		"--period", "10s"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(),
		"Only one of period|cron can be specified")
}

func TestGetJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "get", "--no-color",
		client.JobForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
created_at: "1970-01-01T00:00:00.000Z"
description: JOB_DESCRIPTION
id: JOB_DEFAULT
name: JOB_NAME
resource_references: null
updated_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestGetJobClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "get", "--no-color",
		client.JobForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestGetJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "get", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestListJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "list", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
- created_at: "1970-01-01T00:00:00.000Z"
  description: JOB_DESCRIPTION
  id: "100"
  name: MY_FIRST_JOB
  resource_references: null
  updated_at: "0001-01-01T00:00:00.000Z"
- created_at: "1970-01-01T00:00:00.000Z"
  description: JOB_DESCRIPTION
  id: "101"
  name: MY_SECOND_JOB
  resource_references: null
  updated_at: "0001-01-01T00:00:00.000Z"
- created_at: "1970-01-01T00:00:00.000Z"
  description: JOB_DESCRIPTION
  id: "102"
  name: MY_THIRD_JOB
  resource_references: null
  updated_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListJobMaxItems(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "list", "--no-color",
		"--max-items", "1"})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
- created_at: "1970-01-01T00:00:00.000Z"
  description: JOB_DESCRIPTION
  id: "100"
  name: MY_FIRST_JOB
  resource_references: null
  updated_at: "0001-01-01T00:00:00.000Z"
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestListJobInvalidMaxItems(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "list", "--no-color",
		"--max-items", "INVALID_MAX_ITEMS"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid argument \"INVALID_MAX_ITEMS\"")
}

func TestListJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "list", "--no-color", "EXTRA_ARGUMENT"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expected 0 arguments")
}

func TestEnableJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "enable", "--no-color",
		client.JobForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestEnableJobClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "enable", "--no-color",
		client.JobForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestEnableJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "enable", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestDisableJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "disable", "--no-color",
		client.JobForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestDisableJobClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "disable", "--no-color",
		client.JobForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestDisableJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "disable", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}

func TestDeleteJob(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "delete", "--no-color",
		client.JobForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	//To print the actual output, use: fmt.Println(factory.Result())
}

func TestDeleteJobClientError(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "delete", "--no-color",
		client.JobForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
}

func TestDeleteJobInvalidArgumentCount(t *testing.T) {
	rootCmd, _ := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"job", "delete", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'ID' argument")
}
