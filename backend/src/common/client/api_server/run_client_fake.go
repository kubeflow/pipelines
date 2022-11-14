package api_server

import (
	"fmt"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	runparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	runmodel "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
)

const (
	RunForDefaultTest     = "RUN_DEFAULT"
	RunForClientErrorTest = "RUN_CLIENT_ERROR"
)

func getDefaultRun(id string, name string) *runmodel.V1beta1RunDetail {
	return &runmodel.V1beta1RunDetail{
		PipelineRuntime: &runmodel.V1beta1PipelineRuntime{WorkflowManifest: getDefaultWorkflowAsString()},
		Run: &runmodel.V1beta1Run{
			CreatedAt: strfmt.NewDateTime(),
			ID:        id,
			Name:      name,
			Metrics:   []*runmodel.V1beta1RunMetric{},
		},
	}
}

type RunClientFake struct{}

func NewRunClientFake() *RunClientFake {
	return &RunClientFake{}
}

func (c *RunClientFake) Get(params *runparams.GetRunV1Params) (*runmodel.V1beta1RunDetail,
	*workflowapi.Workflow, error) {
	switch params.RunID {
	case RunForClientErrorTest:
		return nil, nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultRun(params.RunID, "RUN_NAME"), getDefaultWorkflow(), nil
	}
}

func (c *RunClientFake) List(params *runparams.ListRunsV1Params) (
	[]*runmodel.V1beta1Run, int, string, error) {
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
		return []*runmodel.V1beta1Run{
			getDefaultRun("100", "MY_FIRST_RUN").Run,
			getDefaultRun("101", "MY_SECOND_RUN").Run,
		}, 2, SecondToken, nil
	case SecondToken:
		return []*runmodel.V1beta1Run{
			getDefaultRun("102", "MY_THIRD_RUN").Run,
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *RunClientFake) ListAll(params *runparams.ListRunsV1Params, maxResultSize int) (
	[]*runmodel.V1beta1Run, error) {
	return listAllForRun(c, params, maxResultSize)
}

func (c *RunClientFake) Archive(params *runparams.ArchiveRunV1Params) error {
	return nil
}

func (c *RunClientFake) Unarchive(params *runparams.UnarchiveRunV1Params) error {
	return nil
}

func (c *RunClientFake) Terminate(params *runparams.TerminateRunV1Params) error {
	switch params.RunID {
	case RunForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case RunForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest, params.RunID)
	}
}
