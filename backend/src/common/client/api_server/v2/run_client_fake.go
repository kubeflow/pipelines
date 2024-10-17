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
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

func getDefaultRun(id string, name string) *model.V2beta1Run {
	return &model.V2beta1Run{
		CreatedAt:   strfmt.NewDateTime(),
		RunID:       id,
		DisplayName: name,
	}
}

type RunClientFake struct{}

func NewRunClientFake() *RunClientFake {
	return &RunClientFake{}
}

func (c *RunClientFake) Create(params *params.RunServiceCreateRunParams) (*model.V2beta1Run, error) {
	return getDefaultRun("100", "RUN_NAME"), nil
}

func (c *RunClientFake) Get(params *params.RunServiceGetRunParams) (*model.V2beta1Run, error) {
	return getDefaultRun(params.RunID, "RUN_NAME"), nil
}

func (c *RunClientFake) List(params *params.RunServiceListRunsParams) (
	[]*model.V2beta1Run, int, string, error) {
	return []*model.V2beta1Run{
		getDefaultRun("100", "MY_FIRST_RUN"),
		getDefaultRun("101", "MY_SECOND_RUN"),
	}, 2, "", nil
}

func (c *RunClientFake) ListAll(params *params.RunServiceListRunsParams, maxResultSize int) (
	[]*model.V2beta1Run, error) {
	return listAllForRun(c, params, maxResultSize)
}

func (c *RunClientFake) Archive(params *params.RunServiceArchiveRunParams) error {
	return nil
}

func (c *RunClientFake) Unarchive(params *params.RunServiceUnarchiveRunParams) error {
	return nil
}

func (c *RunClientFake) Terminate(params *params.RunServiceTerminateRunParams) error {
	return fmt.Errorf(InvalidFakeRequest, params.RunID)

}
