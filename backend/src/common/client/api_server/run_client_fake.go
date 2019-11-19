package api_server

import (
	"fmt"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	runparams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	runmodel "github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
)

const (
	RunForDefaultTest     = "RUN_DEFAULT"
	RunForClientErrorTest = "RUN_CLIENT_ERROR"
)

func getDefaultRun(id string, name string) *runmodel.APIRunDetail {
	return &runmodel.APIRunDetail{
		PipelineRuntime: &runmodel.APIPipelineRuntime{WorkflowManifest: getDefaultWorkflowAsString()},
		Run: &runmodel.APIRun{
			CreatedAt: strfmt.NewDateTime(),
			ID:        id,
			Name:      name,
			Metrics:   []*runmodel.APIRunMetric{},
		},
	}
}

type RunClientFake struct{}

func NewRunClientFake() *RunClientFake {
	return &RunClientFake{}
}

func (c *RunClientFake) Get(params *runparams.GetRunParams) (*runmodel.APIRunDetail,
	*workflowapi.Workflow, error) {
	switch params.RunID {
	case RunForClientErrorTest:
		return nil, nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultRun(params.RunID, "RUN_NAME"), getDefaultWorkflow(), nil
	}
}

func (c *RunClientFake) List(params *runparams.ListRunsParams) (
	[]*runmodel.APIRun, int, string, error) {
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
		return []*runmodel.APIRun{
			getDefaultRun("100", "MY_FIRST_RUN").Run,
			getDefaultRun("101", "MY_SECOND_RUN").Run,
		}, 2, SecondToken, nil
	case SecondToken:
		return []*runmodel.APIRun{
			getDefaultRun("102", "MY_THIRD_RUN").Run,
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *RunClientFake) ListAll(params *runparams.ListRunsParams, maxResultSize int) (
	[]*runmodel.APIRun, error) {
	return listAllForRun(c, params, maxResultSize)
}

func (c *RunClientFake) Archive(params *runparams.ArchiveRunParams) error {
	return nil
}

func (c *RunClientFake) Unarchive(params *runparams.UnarchiveRunParams) error {
	return nil
}

func (c *RunClientFake) Terminate(params *runparams.TerminateRunParams) error {
	switch params.RunID {
	case RunForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case RunForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest, params.RunID)
	}
}
