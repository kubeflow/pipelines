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

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	runparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	runmodel "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
)

const (
	RunForDefaultTest     = "RUN_DEFAULT"
	RunForClientErrorTest = "RUN_CLIENT_ERROR"
)

func getDefaultRun(id string, name string) *runmodel.APIRunDetail {
	return &runmodel.APIRunDetail{
		PipelineRuntime: &runmodel.APIPipelineRuntime{WorkflowManifest: getDefaultWorkflowAsString()},
		Run: &runmodel.APIRun{
			CreatedAt: strfmt.NewDateTime(),
			ID:        id,
			Name:      name,
			Metrics:   []*runmodel.APIRunMetric{},
		},
	}
}

type RunClientFake struct{}

func NewRunClientFake() *RunClientFake {
	return &RunClientFake{}
}

func (c *RunClientFake) Get(params *runparams.RunServiceGetRunV1Params) (*runmodel.APIRunDetail,
	*workflowapi.Workflow, error) {
	switch params.RunID {
	case RunForClientErrorTest:
		return nil, nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultRun(params.RunID, "RUN_NAME"), getDefaultWorkflow(), nil
	}
}

func (c *RunClientFake) List(params *runparams.RunServiceListRunsV1Params) (
	[]*runmodel.APIRun, int, string, error) {
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
		return []*runmodel.APIRun{
			getDefaultRun("100", "MY_FIRST_RUN").Run,
			getDefaultRun("101", "MY_SECOND_RUN").Run,
		}, 2, SecondToken, nil
	case SecondToken:
		return []*runmodel.APIRun{
			getDefaultRun("102", "MY_THIRD_RUN").Run,
		}, 1, FinalToken, nil
	default:
		return nil, 0, "", fmt.Errorf(InvalidFakeRequest, token)
	}
}

func (c *RunClientFake) ListAll(params *runparams.RunServiceListRunsV1Params, maxResultSize int) (
	[]*runmodel.APIRun, error) {
	return listAllForRun(c, params, maxResultSize)
}

func (c *RunClientFake) Archive(params *runparams.RunServiceArchiveRunV1Params) error {
	return nil
}

func (c *RunClientFake) Unarchive(params *runparams.RunServiceUnarchiveRunV1Params) error {
	return nil
}

func (c *RunClientFake) Terminate(params *runparams.RunServiceTerminateRunV1Params) error {
	switch params.RunID {
	case RunForClientErrorTest:
		return fmt.Errorf(ClientErrorString)
	case RunForDefaultTest:
		return nil
	default:
		return fmt.Errorf(InvalidFakeRequest, params.RunID)
	}
}
