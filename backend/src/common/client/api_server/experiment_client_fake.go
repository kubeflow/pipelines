package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	experimentparams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	experimentmodel "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
)

const (
	ExperimentForDefaultTest     = "EXPERIMENT_DEFAULT"
	ExperimentForClientErrorTest = "EXPERIMENT_CLIENT_ERROR"
)

func getDefaultExperiment(id string, name string) *experimentmodel.APIExperiment {
	return &experimentmodel.APIExperiment{
		CreatedAt:   strfmt.NewDateTime(),
		Description: "EXPERIMENT_DESCRIPTION",
		ID:          id,
		Name:        name,
	}
}

type ExperimentClientFake struct{}

func NewExperimentClientFake() *ExperimentClientFake {
	return &ExperimentClientFake{}
}

func (c *ExperimentClientFake) Create(params *experimentparams.CreateExperimentParams) (
	*experimentmodel.APIExperiment, error) {
	switch params.Body.Name {
	case ExperimentForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultExperiment("500", params.Body.Name), nil
	}
}

func (c *ExperimentClientFake) Get(params *experimentparams.GetExperimentParams) (
	*experimentmodel.APIExperiment, error) {
	switch params.ID {
	case ExperimentForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultExperiment(params.ID, "EXPERIMENT_NAME"), nil
	}
}

func (c *ExperimentClientFake) List(params *experimentparams.ListExperimentParams) (
	[]*experimentmodel.APIExperiment, string, error) {
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
		return []*experimentmodel.APIExperiment{
			getDefaultExperiment("100", "MY_FIRST_EXPERIMENT"),
			getDefaultExperiment("101", "MY_SECOND_EXPERIMENT"),
		}, SecondToken, nil
	case SecondToken:
		return []*experimentmodel.APIExperiment{
			getDefaultExperiment("102", "MY_THIRD_EXPERIMENT"),
		}, FinalToken, nil
	default:
		return nil, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *ExperimentClientFake) ListAll(params *experimentparams.ListExperimentParams,
	maxResultSize int) ([]*experimentmodel.APIExperiment, error) {
	return listAllForExperiment(c, params, maxResultSize)
}
