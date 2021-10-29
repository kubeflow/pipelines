package api_server

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"

	"path"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	pipelinemodel "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PipelineForDefaultTest     = "PIPELINE_ID_10"
	PipelineForClientErrorTest = "PIPELINE_ID_11"
	PipelineValidURL           = "http://www.mydomain.com/foo.yaml"
	PipelineInvalidURL         = "foobar.something"
)

func getDefaultPipeline(id string) *pipelinemodel.APIPipeline {
	return &pipelinemodel.APIPipeline{
		CreatedAt:   strfmt.NewDateTime(),
		Description: "PIPELINE_DESCRIPTION",
		ID:          id,
		Name:        "PIPELINE_NAME",
		Parameters: []*pipelinemodel.APIParameter{&pipelinemodel.APIParameter{
			Name:  "PARAM_NAME",
			Value: "PARAM_VALUE",
		}},
	}
}

func getDefaultWorkflow() *workflowapi.Workflow {
	return &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		}}
}

func getDefaultTemplate() template.Template {
	tmpl, _ := template.NewArgoTemplateFromWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		}})
	return tmpl
}

func getDefaultWorkflowAsString() string {
	tmpl := getDefaultTemplate()
	return string(tmpl.Bytes())
}

type PipelineClientFake struct{}

func NewPipelineClientFake() *PipelineClientFake {
	return &PipelineClientFake{}
}

func (c *PipelineClientFake) Create(params *pipelineparams.CreatePipelineParams) (
	*pipelinemodel.APIPipeline, error) {
	switch params.Body.URL.PipelineURL {
	case PipelineInvalidURL:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultPipeline(path.Base(params.Body.URL.PipelineURL)), nil
	}
}

func (c *PipelineClientFake) Get(params *pipelineparams.GetPipelineParams) (
	*pipelinemodel.APIPipeline, error) {
	switch params.ID {
	case PipelineForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultPipeline(params.ID), nil
	}
}

func (c *PipelineClientFake) Delete(params *pipelineparams.DeletePipelineParams) error {
	switch params.ID {
	case PipelineForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *PipelineClientFake) GetTemplate(params *pipelineparams.GetTemplateParams) (
	template.Template, error) {
	switch params.ID {
	case PipelineForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultTemplate(), nil
	}
}

func (c *PipelineClientFake) List(params *pipelineparams.ListPipelinesParams) (
	[]*pipelinemodel.APIPipeline, int, string, error) {

	const (
		FirstToken  = ""
		SecondToken = "SECOND_TOKEN"
		FinalToken  = ""
	)

	token := ""
	if params.PageToken != nil {
		token = *params.PageToken
	}

	switch token {
	case FirstToken:
		return []*pipelinemodel.APIPipeline{
			getDefaultPipeline("PIPELINE_ID_100"),
			getDefaultPipeline("PIPELINE_ID_101"),
		}, 2, SecondToken, nil
	case SecondToken:
		return []*pipelinemodel.APIPipeline{
			getDefaultPipeline("PIPELINE_ID_102"),
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *PipelineClientFake) ListAll(params *pipelineparams.ListPipelinesParams,
	maxResultSize int) ([]*pipelinemodel.APIPipeline, error) {
	return listAllForPipeline(c, params, maxResultSize)
}

func (c *PipelineClientFake) UpdateDefaultVersion(params *params.UpdatePipelineDefaultVersionParams) error {
	switch params.PipelineID {
	case PipelineForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}
