package api_server

import (
	"context"
	"fmt"
	"os"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/googleprivate/ml/backend/api/go_http_client/pipeline_upload_client"
	params "github.com/googleprivate/ml/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/googleprivate/ml/backend/api/go_http_client/pipeline_upload_model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	pipelineUploadFieldName      = "uploadfile"
	pipelineUploadPath           = "pipelines/upload"
	pipelineUploadServerBasePath = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/apis/v1alpha2/%s"
	pipelineUploadContentTypeKey = "Content-Type"
)

type PipelineUploadInterface interface {
	UploadFile(filePath string, parameters *params.UploadPipelineParams) (*model.APIPipeline, error)
}

type PipelineUploadClient struct {
	apiClient *apiclient.PipelineUpload
}

func NewPipelineUploadClient(clientConfig clientcmd.ClientConfig) (
	*PipelineUploadClient, error) {
	// Creating k8 client
	k8Client, config, namespace, err := util.GetKubernetesClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating K8 client")
	}

	// Create API client
	httpClient := k8Client.RESTClient().(*rest.RESTClient).Client
	masterIPAndPort := util.ExtractMasterIPAndPort(config)
	apiClient := apiclient.New(httptransport.NewWithClient(masterIPAndPort,
		fmt.Sprintf(apiServerBasePath, namespace), nil, httpClient), strfmt.Default)

	// Creating upload client
	return &PipelineUploadClient{
		apiClient: apiClient,
	}, nil
}

func (c *PipelineUploadClient) UploadFile(filePath string, parameters *params.UploadPipelineParams) (
	*model.APIPipeline, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()

	parameters.Uploadfile = runtime.NamedReader(filePath, file)
	return c.Upload(parameters)
}

func (c *PipelineUploadClient) Upload(parameters *params.UploadPipelineParams) (*model.APIPipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	response, err := c.apiClient.PipelineUploadService.UploadPipeline(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.UploadPipelineDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline. Params: %v", parameters),
			fmt.Sprintf("Failed to upload pipeline"))
	}

	return response.Payload, nil
}
