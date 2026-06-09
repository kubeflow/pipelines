// Copyright 2024 The Kubeflow Authors
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

package main

import (
	"context"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/client"
	util "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeRunServiceClient captures the context passed to CreateRun so tests can
// inspect outgoing gRPC metadata.
type fakeRunServiceClient struct {
	capturedCtx context.Context
}

func (f *fakeRunServiceClient) CreateRun(ctx context.Context, in *api.CreateRunRequest, opts ...grpc.CallOption) (*api.Run, error) {
	f.capturedCtx = ctx
	return &api.Run{DisplayName: "fake-run"}, nil
}

func (f *fakeRunServiceClient) GetRun(ctx context.Context, in *api.GetRunRequest, opts ...grpc.CallOption) (*api.Run, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) ListRuns(ctx context.Context, in *api.ListRunsRequest, opts ...grpc.CallOption) (*api.ListRunsResponse, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) ArchiveRun(ctx context.Context, in *api.ArchiveRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) UnarchiveRun(ctx context.Context, in *api.UnarchiveRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) DeleteRun(ctx context.Context, in *api.DeleteRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) TerminateRun(ctx context.Context, in *api.TerminateRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeRunServiceClient) RetryRun(ctx context.Context, in *api.RetryRunRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// fakeTokenSource implements transport.ResettableTokenSource for testing.
type fakeTokenSource struct {
	token string
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: f.token}, nil
}

func (f *fakeTokenSource) ResetTokenOlderThan(time.Time) {}

// newTestSWFForAPIPath creates a minimal ScheduledWorkflow that triggers the
// API-server code path (PipelineId set, no embedded Workflow.Spec).
func newTestSWFForAPIPath() *util.ScheduledWorkflow {
	return util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
			Kind:       "ScheduledWorkflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-swf",
			Namespace: "kubeflow",
			UID:       "test-uid",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			PipelineId: "pipeline-123",
			Workflow:   &swfapi.WorkflowResource{},
		},
	})
}

// TestUserIdentityHeader_BothSet verifies that when both userIdentityHeader and
// userIdentityValue are configured, the CreateRun call carries the corresponding
// gRPC metadata key/value.
func TestUserIdentityHeader_BothSet(test *testing.T) {
	fakeRun := &fakeRunServiceClient{}
	controller := &Controller{
		workflowClient:     client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
		runClient:          fakeRun,
		userIdentityHeader: "kubeflow-userid",
		userIdentityValue:  "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow",
	}

	swf := newTestSWFForAPIPath()
	submitted, _, err := controller.submitNewWorkflowIfNotAlreadySubmitted(
		context.Background(), swf, 100, 200)

	require.NoError(test, err)
	assert.True(test, submitted)

	md, ok := metadata.FromOutgoingContext(fakeRun.capturedCtx)
	require.True(test, ok, "expected outgoing metadata on context")
	assert.Equal(test, []string{"system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow"},
		md.Get("kubeflow-userid"))
}

// TestUserIdentityHeader_BothEmpty verifies that when both fields are empty
// (default), no user identity header is set on the outgoing context.
func TestUserIdentityHeader_BothEmpty(test *testing.T) {
	fakeRun := &fakeRunServiceClient{}
	controller := &Controller{
		workflowClient:     client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
		runClient:          fakeRun,
		userIdentityHeader: "",
		userIdentityValue:  "",
	}

	swf := newTestSWFForAPIPath()
	submitted, _, err := controller.submitNewWorkflowIfNotAlreadySubmitted(
		context.Background(), swf, 100, 200)

	require.NoError(test, err)
	assert.True(test, submitted)

	md, ok := metadata.FromOutgoingContext(fakeRun.capturedCtx)
	if ok {
		assert.Empty(test, md.Get("kubeflow-userid"),
			"expected no kubeflow-userid header when both fields are empty")
	}
}

// TestUserIdentityHeader_OnlyOneSet verifies that when only one of the two
// fields is set, no header is injected (the guard requires both).
func TestUserIdentityHeader_OnlyOneSet(test *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{
			name:   "header set, value empty",
			header: "kubeflow-userid",
			value:  "",
		},
		{
			name:   "header empty, value set",
			header: "",
			value:  "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow",
		},
	}

	for _, testCase := range tests {
		test.Run(testCase.name, func(test *testing.T) {
			fakeRun := &fakeRunServiceClient{}
			controller := &Controller{
				workflowClient:     client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
				runClient:          fakeRun,
				userIdentityHeader: testCase.header,
				userIdentityValue:  testCase.value,
			}

			swf := newTestSWFForAPIPath()
			submitted, _, err := controller.submitNewWorkflowIfNotAlreadySubmitted(
				context.Background(), swf, 100, 200)

			require.NoError(test, err)
			assert.True(test, submitted)

			md, ok := metadata.FromOutgoingContext(fakeRun.capturedCtx)
			if ok {
				assert.Empty(test, md.Get("kubeflow-userid"),
					"expected no kubeflow-userid header when only one field is set")
			}
		})
	}
}

// TestUserIdentityHeader_CoexistsWithBearerToken verifies that both the
// Authorization: Bearer header and the user identity header are present on
// the outgoing context when tokenSrc is non-nil and identity fields are set.
func TestUserIdentityHeader_CoexistsWithBearerToken(test *testing.T) {
	fakeRun := &fakeRunServiceClient{}
	controller := &Controller{
		workflowClient:     client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
		runClient:          fakeRun,
		tokenSrc:           &fakeTokenSource{token: "my-sa-token"},
		userIdentityHeader: "kubeflow-userid",
		userIdentityValue:  "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow",
	}

	swf := newTestSWFForAPIPath()
	submitted, _, err := controller.submitNewWorkflowIfNotAlreadySubmitted(
		context.Background(), swf, 100, 200)

	require.NoError(test, err)
	assert.True(test, submitted)

	md, ok := metadata.FromOutgoingContext(fakeRun.capturedCtx)
	require.True(test, ok, "expected outgoing metadata on context")

	// Verify Authorization header is present
	assert.Equal(test, []string{"Bearer my-sa-token"}, md.Get("authorization"))

	// Verify user identity header is present
	assert.Equal(test, []string{"system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow"},
		md.Get("kubeflow-userid"))
}

// TestNormalizeAndValidateMetadataKey verifies that gRPC metadata keys are
// lowercased and validated, so the controller can fail fast at startup on a
// malformed --userIdentityHeader instead of failing requests at runtime.
func TestNormalizeAndValidateMetadataKey(test *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "already valid", input: "kubeflow-userid", want: "kubeflow-userid"},
		{name: "uppercase normalized to lowercase", input: "Kubeflow-UserID", want: "kubeflow-userid"},
		{name: "digits, dot, underscore allowed", input: "x-user_id.v2", want: "x-user_id.v2"},
		{name: "space is invalid", input: "kubeflow userid", expectErr: true},
		{name: "colon is invalid", input: "kubeflow:userid", expectErr: true},
		{name: "empty is invalid", input: "", expectErr: true},
	}

	for _, testCase := range tests {
		test.Run(testCase.name, func(test *testing.T) {
			got, err := normalizeAndValidateMetadataKey(testCase.input)
			if testCase.expectErr {
				require.Error(test, err)
				assert.Empty(test, got)
				return
			}
			require.NoError(test, err)
			assert.Equal(test, testCase.want, got)
		})
	}
}

// TestNewController_InvalidUserIdentityHeader verifies that NewController fails
// fast when the configured user identity metadata key is malformed. Validation
// happens before any client or informer is used, so nil dependencies are fine.
func TestNewController_InvalidUserIdentityHeader(test *testing.T) {
	_, err := NewController(
		nil, // kubeClientSet
		nil, // swfClientSet
		nil, // workflowClientSet
		nil, // runClient
		nil, // swfInformerFactory
		nil, // executionInformer
		nil, // time
		nil, // location
		nil, // tokenSrc
		"invalid header",
		"some-value",
	)
	require.Error(test, err)
	assert.Contains(test, err.Error(), "invalid userIdentityHeader")
}
