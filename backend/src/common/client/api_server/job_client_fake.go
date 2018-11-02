package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	jobparams "github.com/googleprivate/ml/backend/api/go_http_client/job_client/job_service"
	jobmodel "github.com/googleprivate/ml/backend/api/go_http_client/job_model"
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
	case JobForDefaultTest:
		return getDefaultJob("500", params.Body.Name), nil
	default:
		return nil, fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) Get(params *jobparams.GetJobParams) (
	*jobmodel.APIJob, error) {
	switch params.ID {
	case JobForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	case JobForDefaultTest:
		return getDefaultJob(params.ID, "JOB_NAME"), nil
	default:
		return nil, fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) Delete(params *jobparams.DeleteJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case JobForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) Enable(params *jobparams.EnableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case JobForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) Disable(params *jobparams.DisableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case JobForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) List(params *jobparams.ListJobsParams) (
	[]*jobmodel.APIJob, string, error) {
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
		}, SecondToken, nil
	case SecondToken:
		return []*jobmodel.APIJob{
			getDefaultJob("102", "MY_THIRD_JOB"),
		}, FinalToken, nil
	default:
		return nil, "", fmt.Errorf(InvalidFakeRequest)
	}
}

func (c *JobClientFake) ListAll(params *jobparams.ListJobsParams,
	maxResultSize int) ([]*jobmodel.APIJob, error) {
	return listAllForJob(c, params, maxResultSize)
}
