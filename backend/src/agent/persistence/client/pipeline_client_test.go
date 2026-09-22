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

package client

import (
	"context"
	"net"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client/tokenrefresher"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type reportRecorder struct {
	api.UnimplementedReportServiceServer
	workflow, schedule, authorization string
	err                               error
}

func (s *reportRecorder) ReportWorkflow(ctx context.Context, req *api.ReportWorkflowRequest) (*emptypb.Empty, error) {
	s.workflow = req.Workflow
	md, _ := metadata.FromIncomingContext(ctx)
	s.authorization = md.Get("authorization")[0]
	return &emptypb.Empty{}, s.err
}
func (s *reportRecorder) ReportScheduledWorkflow(ctx context.Context, req *api.ReportScheduledWorkflowRequest) (*emptypb.Empty, error) {
	s.schedule = req.ScheduledWorkflow
	return &emptypb.Empty{}, s.err
}

type tokenFile struct{}

func (tokenFile) ReadFile(string) ([]byte, error) { return []byte("agent-token"), nil }

func TestPipelineClient_ReportsThroughV2Service(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	rpc := grpc.NewServer()
	recorder := &reportRecorder{}
	api.RegisterReportServiceServer(rpc, recorder)
	go rpc.Serve(listener)
	t.Cleanup(rpc.Stop)
	conn, err := grpc.NewClient("passthrough:///agent", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	tokens := tokenrefresher.NewTokenRefresher(time.Minute, tokenFile{})
	require.NoError(t, tokens.RefreshToken())
	client := &PipelineClient{reportServiceClient: api.NewReportServiceClient(conn), tokenRefresher: tokens}
	workflow := util.NewWorkflow(&workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "ir-run", Namespace: "ns1"}})
	require.NoError(t, client.ReportWorkflow(workflow))
	require.Equal(t, workflow.ToStringForStore(), recorder.workflow)
	require.Equal(t, "Bearer agent-token", recorder.authorization)
	swf := util.NewScheduledWorkflow(&swapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "ir-schedule", Namespace: "ns1"}})
	require.NoError(t, client.ReportScheduledWorkflow(swf))
	require.Equal(t, swf.ToStringForStore(), recorder.schedule)
	recorder.err = status.Error(codes.InvalidArgument, "invalid workflow")
	require.True(t, util.HasCustomCode(client.ReportWorkflow(workflow), util.CUSTOM_CODE_PERMANENT))
	recorder.err = status.Error(codes.Unavailable, "retry report")
	require.True(t, util.HasCustomCode(client.ReportWorkflow(workflow), util.CUSTOM_CODE_TRANSIENT))
}
