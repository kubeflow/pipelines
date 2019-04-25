package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	jobparams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	jobmodel "github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
)

const (
	JobForDefaultTest     = "JOB_DEFAULT"
	JobForClientErrorTest = "JOB_CLIENT_ERROR"
)

func getDefaultJob(id string, name string) *jobmodel.APIJob {
	return &jobmodel.APIJob{
		CreatedAt:   strfmt.NewDateTime(),
		Description: "JOB_DESCRIPTION",
		ID:          id,
		Name:        name,
	}
}

type JobClientFake struct{}

func NewJobClientFake() *JobClientFake {
	return &JobClientFake{}
}

func (c *JobClientFake) Create(params *jobparams.CreateJobParams) (
	*jobmodel.APIJob, error) {
	switch params.Body.Name {
	case JobForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultJob("500", params.Body.Name), nil
	}
}

func (c *JobClientFake) Get(params *jobparams.GetJobParams) (
	*jobmodel.APIJob, error) {
	switch params.ID {
	case JobForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultJob(params.ID, "JOB_NAME"), nil
	}
}

func (c *JobClientFake) Delete(params *jobparams.DeleteJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) Enable(params *jobparams.EnableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) Disable(params *jobparams.DisableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) List(params *jobparams.ListJobsParams) (
	[]*jobmodel.APIJob, int, string, error) {
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
		return []*jobmodel.APIJob{
			getDefaultJob("100", "MY_FIRST_JOB"),
			getDefaultJob("101", "MY_SECOND_JOB"),
		}, 2, SecondToken, nil
	case SecondToken:
		return []*jobmodel.APIJob{
			getDefaultJob("102", "MY_THIRD_JOB"),
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *JobClientFake) ListAll(params *jobparams.ListJobsParams,
	maxResultSize int) ([]*jobmodel.APIJob, error) {
	return listAllForJob(c, params, maxResultSize)
}
