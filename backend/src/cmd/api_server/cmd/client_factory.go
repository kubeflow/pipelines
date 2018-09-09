package cmd

import (
	"io"
	"os"

	client "github.com/googleprivate/ml/backend/src/common/client/api_server"
	"github.com/googleprivate/ml/backend/src/common/util"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientFactoryInterface interface {
	CreatePipelineUploadClient(config clientcmd.ClientConfig) (
		client.PipelineUploadInterface, error)
	CreatePipelineClient(config clientcmd.ClientConfig) (
		client.PipelineInterface, error)
	CreateJobClient(config clientcmd.ClientConfig) (
		client.JobInterface, error)
	CreateRunClient(config clientcmd.ClientConfig) (
		client.RunInterface, error)
	Time() util.TimeInterface
	Writer() io.Writer
}

type ClientFactory struct {
	time   util.TimeInterface
	writer io.Writer
}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		time:   util.NewRealTime(),
		writer: os.Stdout,
	}
}

func (f *ClientFactory) CreatePipelineUploadClient(config clientcmd.ClientConfig) (
	client.PipelineUploadInterface, error) {
	return client.NewPipelineUploadClient(config)
}

func (f *ClientFactory) CreatePipelineClient(config clientcmd.ClientConfig) (
	client.PipelineInterface, error) {
	return client.NewPipelineClient(config)
}

func (f *ClientFactory) CreateJobClient(config clientcmd.ClientConfig) (
	client.JobInterface, error) {
	return client.NewJobClient(config)
}

func (f *ClientFactory) CreateRunClient(config clientcmd.ClientConfig) (
	client.RunInterface, error) {
	return client.NewRunClient(config)
}

func (f *ClientFactory) Time() util.TimeInterface {
	return f.time
}

func (f *ClientFactory) Writer() io.Writer {
	return f.writer
}
