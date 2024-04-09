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
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"google.golang.org/protobuf/types/known/structpb"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getDefaultPipeline(id string) *model.V2beta1Pipeline {
	return &model.V2beta1Pipeline{
		CreatedAt:   strfmt.NewDateTime(),
		Description: "PIPELINE_DESCRIPTION",
		PipelineID:  id,
		DisplayName: "PIPELINE_NAME",
	}
}

func getDefaultPipelineSpec() *structpb.Struct {
	m := make(map[string]interface{})
	m["pipelineInfo"] = map[string]interface{}{"name": "MY_NAME"}
	spec, _ := structpb.NewStruct(m)
	return spec
}

func getDefaultTemplate() template.Template {
	tmpl, _ := template.NewArgoTemplateFromWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		}})
	return tmpl
}

func getDefaultWorkflowAsString() string {
	tmpl := getDefaultTemplate()
	return string(tmpl.Bytes())
}

type PipelineClientFake struct{}

func NewPipelineClientFake() *PipelineClientFake {
	return &PipelineClientFake{}
}

func (c *PipelineClientFake) Create(params *params.PipelineServiceCreatePipelineParams) (
	*model.V2beta1Pipeline, error) {
	return getDefaultPipeline(params.Body.PipelineID), nil
}

func (c *PipelineClientFake) CreatePipelineAndVersion(params *params.PipelineServiceCreatePipelineAndVersionParams) (*model.V2beta1Pipeline, error) {
	return getDefaultPipeline(params.Body.Pipeline.PipelineID), nil
}

func (c *PipelineClientFake) Get(params *params.PipelineServiceGetPipelineParams) (
	*model.V2beta1Pipeline, error) {
	return getDefaultPipeline(params.PipelineID), nil

}

func (c *PipelineClientFake) Delete(params *params.PipelineServiceDeletePipelineParams) error {
	return nil
}

func (c *PipelineClientFake) List(params *params.PipelineServiceListPipelinesParams) (
	[]*model.V2beta1Pipeline, int, string, error) {
	return []*model.V2beta1Pipeline{
		getDefaultPipeline("PIPELINE_ID_100"),
		getDefaultPipeline("PIPELINE_ID_101"),
	}, 2, "", nil
}

func (c *PipelineClientFake) ListAll(params *params.PipelineServiceListPipelinesParams,
	maxResultSize int) ([]*model.V2beta1Pipeline, error) {
	return listAllForPipeline(c, params, maxResultSize)
}
