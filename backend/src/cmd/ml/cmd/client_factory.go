package cmd

import (
	"bytes"
	"io"
	"os"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientFactoryInterface interface {
	CreatePipelineUploadClient(config clientcmd.ClientConfig, debug bool) (
		client.PipelineUploadInterface, error)
	CreatePipelineClient(config clientcmd.ClientConfig, debug bool) (
		client.PipelineInterface, error)
	CreateJobClient(config clientcmd.ClientConfig, debug bool) (
		client.JobInterface, error)
	CreateRunClient(config clientcmd.ClientConfig, debug bool) (
		client.RunInterface, error)
	Time() util.TimeInterface
	Writer() io.Writer
	Result() string
}

type ClientFactory struct {
	time   util.TimeInterface
	buffer *bytes.Buffer
	writer io.Writer
}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		time:   util.NewRealTime(),
		writer: os.Stdout,
	}
}

func NewClientFactoryWithByteBuffer() *ClientFactory {
	buffer := new(bytes.Buffer)
	return &ClientFactory{
		time:   util.NewRealTime(),
		buffer: buffer,
		writer: buffer,
	}
}

func (f *ClientFactory) CreatePipelineUploadClient(config clientcmd.ClientConfig, debug bool) (
	client.PipelineUploadInterface, error) {
	return client.NewPipelineUploadClient(config, debug)
}

func (f *ClientFactory) CreatePipelineClient(config clientcmd.ClientConfig, debug bool) (
	client.PipelineInterface, error) {
	return client.NewPipelineClient(config, debug)
}

func (f *ClientFactory) CreateJobClient(config clientcmd.ClientConfig, debug bool) (
	client.JobInterface, error) {
	return client.NewJobClient(config, debug)
}

func (f *ClientFactory) CreateRunClient(config clientcmd.ClientConfig, debug bool) (
	client.RunInterface, error) {
	return client.NewRunClient(config, debug)
}

func (f *ClientFactory) Time() util.TimeInterface {
	return f.time
}

func (f *ClientFactory) Writer() io.Writer {
	return f.writer
}

func (f *ClientFactory) Result() string {
	if f.buffer == nil {
		return "The writer is set to 'os.Stdout'. The result is not recorded."
	}
	return (*f.buffer).String()
}
