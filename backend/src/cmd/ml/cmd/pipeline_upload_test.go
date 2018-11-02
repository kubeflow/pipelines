package cmd

import (
	"fmt"
	"strings"
	"testing"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
)

func TestPipelineUpload(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--no-color",
		client.FileForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	fmt.Println(factory.Result())
}

func TestPipelineUploadJson(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--no-color", "-o", "json",
		client.FileForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := `
SUCCESS
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	fmt.Println(factory.Result())
}

func TestPipelineUploadColor(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", client.FileForDefaultTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.Nil(t, err)

	expected := fmt.Sprintf(`
%s[32mSUCCESS%s[0m
`, Escape, Escape)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(factory.Result()))
	fmt.Println(factory.Result())
}

func TestPipelineUploadNoFile(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--no-color"})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing 'FILE' argument")
	fmt.Println(factory.Result())
}

func TestPipelineUploadClientError(t *testing.T) {
	rootCmd, factory := getFakeRootCommand()
	rootCmd.Command().SetArgs([]string{"pipeline", "upload", "--no-color",
		client.FileForClientErrorTest})
	_, err := rootCmd.Command().ExecuteC()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), client.ClientErrorString)
	fmt.Println(factory.Result())
}
