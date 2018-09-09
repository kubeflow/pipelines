package api_server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2/json"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	pipelineUploadFieldName      = "uploadfile"
	pipelineUploadPath           = "pipelines/upload"
	pipelineUploadServerBasePath = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/apis/v1alpha2/%s"
	pipelineUploadContentTypeKey = "Content-Type"
)

type PipelineUploadInterface interface {
	Upload(filePath string) (*api.Pipeline, error)
}

type PipelineUploadClient struct {
	k8Client  *kubernetes.Clientset
	namespace string
}

func NewPipelineUploadClient(clientConfig clientcmd.ClientConfig) (
	*PipelineUploadClient, error) {
	// Creating k8 client
	k8Client, _, namespace, err := util.GetKubernetesClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating K8 client")
	}

	// Creating upload client
	return &PipelineUploadClient{
		k8Client:  k8Client,
		namespace: namespace,
	}, nil
}

func (c *PipelineUploadClient) Upload(filePath string) (*api.Pipeline, error) {
	// Convert file to bytes
	bytes, writer, err := toBytesFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Upload bytes
	responseBytes, err := c.k8Client.RESTClient().Post().Context(ctx).
		AbsPath(fmt.Sprintf(pipelineUploadServerBasePath, c.namespace, pipelineUploadPath)).
		SetHeader(pipelineUploadContentTypeKey, writer.FormDataContentType()).
		Body(bytes).Do().Raw()
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to upload pipeline from file '%s'", filePath))
	}

	// Unmarshal pipeline from response result
	var pkg api.Pipeline
	err = json.Unmarshal(responseBytes, &pkg)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to unmarshal API call result from response bytes"))
	}

	return &pkg, nil
}

func toBytesFromFile(filePath string) (*bytes.Buffer, *multipart.Writer, error) {
	// Opening file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()

	// Creating byte writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(pipelineUploadFieldName, filepath.Base(filePath))
	if err != nil {
		return nil, nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to create byte writer for file '%s'", filePath))
	}

	// Copying file
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to turn file '%s' into bytes", filePath))
	}

	// Closing byte writer
	err = writer.Close()
	if err != nil {
		return nil, nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to close byte writer"))
	}

	return body, writer, nil
}
