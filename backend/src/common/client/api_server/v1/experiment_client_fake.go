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
	experimentparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	experimentmodel "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
)

const (
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

func (c *ExperimentClientFake) Create(params *experimentparams.ExperimentServiceCreateExperimentV1Params) (
	*experimentmodel.APIExperiment, error) {
	switch params.Body.Name {
	case ExperimentForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultExperiment("500", params.Body.Name), nil
	}
}

func (c *ExperimentClientFake) Get(params *experimentparams.ExperimentServiceGetExperimentV1Params) (
	*experimentmodel.APIExperiment, error) {
	switch params.ID {
	case ExperimentForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultExperiment(params.ID, "EXPERIMENT_NAME"), nil
	}
}

func (c *ExperimentClientFake) List(params *experimentparams.ExperimentServiceListExperimentsV1Params) (
	[]*experimentmodel.APIExperiment, int, string, error) {
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
		}, 2, SecondToken, nil
	case SecondToken:
		return []*experimentmodel.APIExperiment{
			getDefaultExperiment("102", "MY_THIRD_EXPERIMENT"),
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *ExperimentClientFake) ListAll(params *experimentparams.ExperimentServiceListExperimentsV1Params,
	maxResultSize int) ([]*experimentmodel.APIExperiment, error) {
	return listAllForExperiment(c, params, maxResultSize)
}

func (c *ExperimentClientFake) Archive(params *experimentparams.ExperimentServiceArchiveExperimentV1Params) error {
	return nil
}

func (c *ExperimentClientFake) Unarchive(params *experimentparams.ExperimentServiceUnarchiveExperimentV1Params) error {
	return nil
}
