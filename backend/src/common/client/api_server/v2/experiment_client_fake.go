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

package api_server_v2

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	experimentparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	experimentmodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
)

const (
	v2ExperimentForClientErrorTest = "EXPERIMENT_CLIENT_ERROR"
)

func getDefaultExperiment(id string, name string) *experimentmodel.V2beta1Experiment {
	return &experimentmodel.V2beta1Experiment{
		CreatedAt:    strfmt.NewDateTime(),
		Description:  "EXPERIMENT_DESCRIPTION",
		ExperimentID: id,
		DisplayName:  name,
	}
}

type ExperimentClientFake struct{}

func NewExperimentClientFake() *ExperimentClientFake {
	return &ExperimentClientFake{}
}

func (c *ExperimentClientFake) Create(params *experimentparams.CreateExperimentParams) (
	*experimentmodel.V2beta1Experiment, error) {
	switch params.Body.DisplayName {
	case v2ExperimentForClientErrorTest:
		return nil, fmt.Errorf(api_server.ClientErrorString)
	default:
		return getDefaultExperiment("500", params.Body.DisplayName), nil
	}
}

func (c *ExperimentClientFake) Get(params *experimentparams.GetExperimentParams) (
	*experimentmodel.V2beta1Experiment, error) {
	switch params.ExperimentID {
	case v2ExperimentForClientErrorTest:
		return nil, fmt.Errorf(api_server.ClientErrorString)
	default:
		return getDefaultExperiment(params.ExperimentID, "EXPERIMENT_NAME"), nil
	}
}

func (c *ExperimentClientFake) List(params *experimentparams.ListExperimentsParams) (
	[]*experimentmodel.V2beta1Experiment, int, string, error) {
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
		return []*experimentmodel.V2beta1Experiment{
			getDefaultExperiment("100", "MY_FIRST_EXPERIMENT"),
			getDefaultExperiment("101", "MY_SECOND_EXPERIMENT"),
		}, 2, SecondToken, nil
	case SecondToken:
		return []*experimentmodel.V2beta1Experiment{
			getDefaultExperiment("102", "MY_THIRD_EXPERIMENT"),
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(api_serer.InvalidFakeRequest, token)
	}
}

func (c *ExperimentClientFake) ListAll(params *experimentparams.ListExperimentsParams,
	maxResultSize int) ([]*experimentmodel.V2beta1Experiment, error) {
	return listAllForExperiment(c, params, maxResultSize)
}

func (c *ExperimentClientFake) Archive(params *experimentparams.ArchiveExperimentParams) error {
	return nil
}

func (c *ExperimentClientFake) Unarchive(params *experimentparams.UnarchiveExperimentParams) error {
	return nil
}
