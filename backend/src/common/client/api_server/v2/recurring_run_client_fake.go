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
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
)

func getDefaultJob(id string, name string) *model.V2beta1RecurringRun {
	return &model.V2beta1RecurringRun{
		CreatedAt:      strfmt.NewDateTime(),
		Description:    "RECURRING_RUN_DESCRIPTION",
		RecurringRunID: id,
		DisplayName:    name,
	}
}

type RecurringRunClientFake struct{}

func NewRecurringRunClientFake() *RecurringRunClientFake {
	return &RecurringRunClientFake{}
}

func (c *RecurringRunClientFake) Create(params *params.RecurringRunServiceCreateRecurringRunParams) (
	*model.V2beta1RecurringRun, error) {
	return getDefaultJob("500", params.Body.DisplayName), nil
}

func (c *RecurringRunClientFake) Get(params *params.RecurringRunServiceGetRecurringRunParams) (
	*model.V2beta1RecurringRun, error) {
	return getDefaultJob(params.RecurringRunID, "RECURRING_RUN_NAME"), nil
}

func (c *RecurringRunClientFake) Delete(params *params.RecurringRunServiceDeleteRecurringRunParams) error {
	return nil
}

func (c *RecurringRunClientFake) Enable(params *params.RecurringRunServiceEnableRecurringRunParams) error {
	return nil
}

func (c *RecurringRunClientFake) Disable(params *params.RecurringRunServiceDisableRecurringRunParams) error {
	return nil
}

func (c *RecurringRunClientFake) List(params *params.RecurringRunServiceListRecurringRunsParams) (
	[]*model.V2beta1RecurringRun, int, string, error) {
	return []*model.V2beta1RecurringRun{
		getDefaultJob("100", "MY_FIRST_RECURRING_RUN"),
		getDefaultJob("101", "MY_SECOND_RECURRING_RUN"),
	}, 2, "", nil
}

func (c *RecurringRunClientFake) ListAll(params *params.RecurringRunServiceListRecurringRunsParams,
	maxResultSize int) ([]*model.V2beta1RecurringRun, error) {
	return listAllForJob(c, params, maxResultSize)
}
