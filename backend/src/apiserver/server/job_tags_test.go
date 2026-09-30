// Copyright 2026 The Kubeflow Authors
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

package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestUpdateRecurringRunTags(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	job, err := clients.JobStore().CreateJob(&model.Job{UUID: "tagged", DisplayName: "nightly", Namespace: experiment.Namespace, ExperimentId: experiment.UUID, Tags: map[string]string{"team": "ml"}})
	require.NoError(t, err)
	server := createJobServer(manager)
	for _, tc := range []struct {
		name      string
		tags      map[string]string
		mask      []string
		want      map[string]string
		wantError string
	}{
		{name: "omitted", want: map[string]string{"team": "ml"}},
		{name: "replace", tags: map[string]string{"env": "prod"}, want: map[string]string{"env": "prod"}},
		{name: "invalid key", tags: map[string]string{"bad.key": "value"}, wantError: "must not contain"},
		{name: "unsupported mask", mask: []string{"display_name"}, wantError: "only tags"},
		{name: "clear over protobuf", tags: map[string]string{}, mask: []string{"tags"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &api.UpdateRecurringRunRequest{RecurringRunId: job.UUID, RecurringRun: &api.RecurringRun{DisplayName: "ignored", Tags: tc.tags}, UpdateMask: &fieldmaskpb.FieldMask{Paths: tc.mask}}
			wire, err := proto.Marshal(request)
			require.NoError(t, err)
			require.NoError(t, proto.Unmarshal(wire, request))
			result, err := server.UpdateRecurringRun(context.Background(), request)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, result.Tags)
			require.Equal(t, "nightly", result.DisplayName)
		})
	}
	_, err = server.UpdateRecurringRun(context.Background(), &api.UpdateRecurringRunRequest{})
	require.ErrorContains(t, err, "ID is required")
	_, err = server.UpdateRecurringRun(context.Background(), &api.UpdateRecurringRunRequest{RecurringRunId: "missing", RecurringRun: &api.RecurringRun{}})
	require.Error(t, err)
}

func TestUpdateRecurringRunTagsUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	original := map[string]string{"team": "ml"}
	_, err := clients.JobStore().CreateJob(&model.Job{UUID: "tagged", DisplayName: "nightly", Namespace: experiment.Namespace, ExperimentId: experiment.UUID, Tags: original})
	require.NoError(t, err)
	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	_, err = createJobServer(manager).UpdateRecurringRun(context.Background(), &api.UpdateRecurringRunRequest{RecurringRunId: "tagged", RecurringRun: &api.RecurringRun{Tags: map[string]string{"env": "prod"}}})
	require.Error(t, err)
	persisted, err := clients.JobStore().GetJob("tagged")
	require.NoError(t, err)
	require.Equal(t, original, persisted.Tags)
}

func TestRecurringRunTagsGateway(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	_, err := clients.JobStore().CreateJob(&model.Job{UUID: "tagged", DisplayName: "nightly", Namespace: experiment.Namespace, ExperimentId: experiment.UUID, Tags: map[string]string{"team": "ml"}})
	require.NoError(t, err)
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	api.RegisterRecurringRunServiceServer(grpcServer, createJobServer(manager))
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	defer conn.Close()
	mux := runtime.NewServeMux()
	require.NoError(t, api.RegisterRecurringRunServiceHandler(context.Background(), mux, conn))
	for _, body := range []string{`{"tags":{"environment":"production"}}`, `{"tags":{}}`} {
		request := httptest.NewRequest(http.MethodPatch, "/apis/v2beta1/recurringruns/tagged?update_mask=tags", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		job, err := clients.JobStore().GetJob("tagged")
		require.NoError(t, err)
		if body == `{"tags":{}}` {
			require.Empty(t, job.Tags)
		} else {
			require.Equal(t, map[string]string{"environment": "production"}, job.Tags)
		}
	}
}

func TestCreateRecurringRunRejectsInvalidTagsBeforeSideEffects(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	_, err := manager.CreateJob(context.Background(), &model.Job{Tags: map[string]string{"": "invalid"}})
	require.ErrorContains(t, err, "tag key cannot be empty")
}

func TestRecurringRunTagConversion(t *testing.T) {
	job := proto.Clone(commonApiRecurringRun).(*api.RecurringRun)
	job.Tags = map[string]string{"team": "ml"}
	converted, err := toModelJob(job)
	require.NoError(t, err)
	require.Equal(t, job.Tags, converted.Tags)
	require.Equal(t, job.Tags, toApiRecurringRun(converted).Tags)
}

func TestListRecurringRunsTags(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	for _, job := range []*model.Job{
		{UUID: "1", DisplayName: "nightly", Namespace: experiment.Namespace, ExperimentId: experiment.UUID, Tags: map[string]string{"team": "ml"}},
		{UUID: "2", DisplayName: "other", Namespace: experiment.Namespace, ExperimentId: experiment.UUID, Tags: map[string]string{"team": "platform"}},
	} {
		_, err := clients.JobStore().CreateJob(job)
		require.NoError(t, err)
	}
	result, err := createJobServer(manager).ListRecurringRuns(context.Background(), &api.ListRecurringRunsRequest{ExperimentId: experiment.UUID, Filter: `{"predicates":[{"key":"tags.team","operation":"EQUALS","string_value":"ml"}]}`})
	require.NoError(t, err)
	require.EqualValues(t, 1, result.TotalSize)
	require.Len(t, result.RecurringRuns, 1)
	require.Equal(t, "1", result.RecurringRuns[0].RecurringRunId)
	require.Equal(t, map[string]string{"team": "ml"}, result.RecurringRuns[0].Tags)
	_, err = createJobServer(manager).ListRecurringRuns(context.Background(), &api.ListRecurringRunsRequest{Filter: `{"predicates":[{"key":"tags.team","operation":"NOT_EQUALS","string_value":"ml"}]}`})
	require.Error(t, err)
}
