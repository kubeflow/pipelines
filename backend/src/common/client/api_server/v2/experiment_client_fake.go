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
	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
)

func getDefaultExperiment(id string, name string) *model.V2beta1Experiment {
	return &model.V2beta1Experiment{
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

func (c *ExperimentClientFake) Create(parameters *params.ExperimentServiceCreateExperimentParams) (
	*model.V2beta1Experiment, error) {
	return getDefaultExperiment("500", parameters.Body.DisplayName), nil
}

func (c *ExperimentClientFake) Get(parameters *params.ExperimentServiceGetExperimentParams) (
	*model.V2beta1Experiment, error) {
	return getDefaultExperiment(parameters.ExperimentID, "EXPERIMENT_NAME"), nil
}

func (c *ExperimentClientFake) List(params *params.ExperimentServiceListExperimentsParams) (
	[]*model.V2beta1Experiment, int, string, error) {
	return []*model.V2beta1Experiment{
		getDefaultExperiment("100", "MY_FIRST_EXPERIMENT"),
		getDefaultExperiment("101", "MY_SECOND_EXPERIMENT"),
	}, 2, "SECOND_TOKEN", nil
}

func (c *ExperimentClientFake) ListAll(params *params.ExperimentServiceListExperimentsParams,
	maxResultSize int) ([]*model.V2beta1Experiment, error) {
	return listAllForExperiment(c, params, 1)
}

func (c *ExperimentClientFake) Archive(parameters *params.ExperimentServiceArchiveExperimentParams) error {
	return nil
}

func (c *ExperimentClientFake) Unarchive(parameters *params.ExperimentServiceUnarchiveExperimentParams) error {
	return nil
}
