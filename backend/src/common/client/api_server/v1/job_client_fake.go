// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	jobparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	jobmodel "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
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

func (c *JobClientFake) Create(params *jobparams.JobServiceCreateJobParams) (
	*jobmodel.APIJob, error) {
	switch params.Body.Name {
	case JobForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultJob("500", params.Body.Name), nil
	}
}

func (c *JobClientFake) Get(params *jobparams.JobServiceGetJobParams) (
	*jobmodel.APIJob, error) {
	switch params.ID {
	case JobForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultJob(params.ID, "JOB_NAME"), nil
	}
}

func (c *JobClientFake) Delete(params *jobparams.JobServiceDeleteJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) Enable(params *jobparams.JobServiceEnableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) Disable(params *jobparams.JobServiceDisableJobParams) error {
	switch params.ID {
	case JobForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	default:
		return nil
	}
}

func (c *JobClientFake) List(params *jobparams.JobServiceListJobsParams) (
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

func (c *JobClientFake) ListAll(params *jobparams.JobServiceListJobsParams,
	maxResultSize int) ([]*jobmodel.APIJob, error) {
	return listAllForJob(c, params, maxResultSize)
}
