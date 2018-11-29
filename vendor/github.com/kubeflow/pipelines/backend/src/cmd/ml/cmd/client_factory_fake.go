package cmd

import (
	"bytes"
	"io"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientFactoryFake struct {
	buffer *bytes.Buffer
	writer io.Writer
	time   util.TimeInterface
}

func NewClientFactoryFake() *ClientFactoryFake {
	buffer := new(bytes.Buffer)
	return &ClientFactoryFake{
		buffer: buffer,
		writer: buffer,
		time:   util.NewFakeTimeForEpoch(),
	}
}

func (f *ClientFactoryFake) CreatePipelineUploadClient(config clientcmd.ClientConfig, debug bool) (
	client.PipelineUploadInterface, error) {
	return client.NewPipelineUploadClientFake(), nil
}

func (f *ClientFactoryFake) CreatePipelineClient(config clientcmd.ClientConfig, debug bool) (
	client.PipelineInterface, error) {
	return client.NewPipelineClientFake(), nil
}

func (f *ClientFactoryFake) CreateJobClient(config clientcmd.ClientConfig, debug bool) (
	client.JobInterface, error) {
	return client.NewJobClientFake(), nil
}

func (f *ClientFactoryFake) CreateRunClient(config clientcmd.ClientConfig, debug bool) (
	client.RunInterface, error) {
	return client.NewRunClientFake(), nil
}

func (f *ClientFactoryFake) Time() util.TimeInterface {
	return f.time
}

func (f *ClientFactoryFake) Writer() io.Writer {
	return f.writer
}

func (f *ClientFactoryFake) Result() string {
	return (*f.buffer).String()
}

func getFakeRootCommand() (*RootCommand, *ClientFactoryFake) {
	factory := NewClientFactoryFake()
	rootCmd := NewRootCmd(factory)
	rootCmd = CreateSubCommands(rootCmd, 10)
	return rootCmd, factory
}
