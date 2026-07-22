// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestIntermediateInputsEndpointsRequireAuthorization(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	modelRun, err := toModelRun(&apiv1beta1.Run{
		Name: "run",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
			Relationship: apiv1beta1.Relationship_OWNER,
		}},
	})
	assert.NoError(t, err)
	modelRun.Namespace = experiment.Namespace
	run, err := manager.CreateRun(context.Background(), modelRun)
	assert.NoError(t, err)

	server := NewRunIntermediateInputsServer(manager)
	contextWithIdentity := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com"),
	)
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		body    string
	}{
		{"get", server.GetIntermediateInputsStatus, http.MethodGet, ""},
		{"set", server.SetIntermediateInputs, http.MethodPost, `{"parameters":{"decision":"YES"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/", bytes.NewBufferString(test.body)).WithContext(contextWithIdentity)
			request = mux.SetURLVars(request, map[string]string{RunKey: run.UUID, NodeKey: "approval"})
			response := httptest.NewRecorder()

			test.handler(response, request)

			assert.Equal(t, http.StatusForbidden, response.Code)
		})
	}
}
