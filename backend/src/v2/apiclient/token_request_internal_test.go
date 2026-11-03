// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiclient

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type fakeServiceAccountTokenClient struct {
	typedcorev1.ServiceAccountInterface
	createToken func(context.Context, string, *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
}

func (client *fakeServiceAccountTokenClient) CreateToken(ctx context.Context, name string, request *authenticationv1.TokenRequest, _ metav1.CreateOptions) (*authenticationv1.TokenRequest, error) {
	return client.createToken(ctx, name, request)
}

func tokenResponse(token string, expiration time.Time) *authenticationv1.TokenRequest {
	return &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{
		Token:               token,
		ExpirationTimestamp: metav1.NewTime(expiration),
	}}
}

func TestNewServiceAccountTokenSourceRequiresBinding(t *testing.T) {
	client := &fakeServiceAccountTokenClient{}
	_, err := NewServiceAccountTokenSource(nil, "runner", "audience", "pod", "uid")
	require.Error(t, err)
	for _, missing := range []string{"service account", "audience", "pod name", "pod UID"} {
		t.Run(missing, func(t *testing.T) {
			values := map[string]string{"service account": "runner", "audience": "audience", "pod name": "pod", "pod UID": "uid"}
			values[missing] = ""
			_, err := NewServiceAccountTokenSource(client, values["service account"], values["audience"], values["pod name"], values["pod UID"])
			require.Error(t, err)
		})
	}
}

func TestServiceAccountTokenSourceBindsAndRefreshesUsingReturnedExpiration(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	client := &fakeServiceAccountTokenClient{createToken: func(received context.Context, name string, request *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		require.Equal(t, ctx, received)
		require.Equal(t, "workflow-runner", name)
		require.Equal(t, []string{"pipelines.kubeflow.org/runs/run-id"}, request.Spec.Audiences)
		require.NotNil(t, request.Spec.ExpirationSeconds)
		require.Equal(t, int64(7200), *request.Spec.ExpirationSeconds)
		require.Equal(t, &authenticationv1.BoundObjectReference{
			APIVersion: "v1", Kind: "Pod", Name: "workflow-agent", UID: types.UID("pod-uid"),
		}, request.Spec.BoundObjectRef)
		calls++
		// The API server shortens the requested two-hour lifetime to ten minutes.
		return tokenResponse(fmt.Sprintf("token-%d", calls), now.Add(10*time.Minute)), nil
	}}
	source, err := NewServiceAccountTokenSource(client, "workflow-runner", "pipelines.kubeflow.org/runs/run-id", "workflow-agent", "pod-uid")
	require.NoError(t, err)
	source.(*serviceAccountTokenSource).now = func() time.Time { return now }

	token, err := source.Token(ctx)
	require.NoError(t, err)
	require.Equal(t, "token-1", token)
	now = now.Add(7*time.Minute + 59*time.Second)
	token, err = source.Token(ctx)
	require.NoError(t, err)
	require.Equal(t, "token-1", token)
	require.Equal(t, 1, calls)
	now = now.Add(time.Second)
	token, err = source.Token(ctx)
	require.NoError(t, err)
	require.Equal(t, "token-2", token)
	require.Equal(t, 2, calls)
}

func TestServiceAccountTokenSourceConcurrentRequestsShareRefresh(t *testing.T) {
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	client := &fakeServiceAccountTokenClient{createToken: func(context.Context, string, *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return tokenResponse("shared-token", time.Now().Add(time.Hour)), nil
	}}
	source, err := NewServiceAccountTokenSource(client, "runner", "run-audience", "agent", "uid")
	require.NoError(t, err)

	const workers = 24
	errors := make(chan error, workers)
	var workersDone sync.WaitGroup
	workersDone.Add(workers)
	for range workers {
		go func() {
			defer workersDone.Done()
			token, err := source.Token(context.Background())
			if err == nil && token != "shared-token" {
				err = fmt.Errorf("unexpected token")
			}
			errors <- err
		}()
	}
	<-started
	close(release)
	workersDone.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestServiceAccountTokenSourceWaitingCallCanCancel(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	client := &fakeServiceAccountTokenClient{createToken: func(context.Context, string, *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		close(started)
		<-release
		return tokenResponse("token", time.Now().Add(time.Hour)), nil
	}}
	source, err := NewServiceAccountTokenSource(client, "runner", "audience", "agent", "uid")
	require.NoError(t, err)
	firstDone := make(chan error, 1)
	go func() {
		_, err := source.Token(context.Background())
		firstDone <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	secondDone := make(chan error, 1)
	go func() {
		_, err := source.Token(ctx)
		secondDone <- err
	}()
	select {
	case err := <-secondDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected canceled waiting request, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("canceled caller waited for another caller's token request")
	}
	close(release)
	require.NoError(t, <-firstDone)
}

func TestServiceAccountTokenSourcePropagatesCancellationToKubernetes(t *testing.T) {
	started := make(chan struct{})
	client := &fakeServiceAccountTokenClient{createToken: func(ctx context.Context, _ string, _ *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	source, err := NewServiceAccountTokenSource(client, "runner", "audience", "agent", "uid")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := source.Token(ctx)
		done <- err
	}()
	<-started
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestServiceAccountTokenSourceFailuresDoNotUseStaleCredentials(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	requestErr := errors.New("token request forbidden")
	calls := 0
	client := &fakeServiceAccountTokenClient{createToken: func(context.Context, string, *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		calls++
		switch calls {
		case 1:
			return tokenResponse("initial-token", now.Add(time.Hour)), nil
		case 2:
			return nil, requestErr
		default:
			return tokenResponse("refreshed-token", now.Add(time.Hour)), nil
		}
	}}
	source, err := NewServiceAccountTokenSource(client, "runner", "audience", "agent", "uid")
	require.NoError(t, err)
	source.(*serviceAccountTokenSource).now = func() time.Time { return now }
	_, err = source.Token(context.Background())
	require.NoError(t, err)
	now = now.Add(time.Hour)
	token, err := source.Token(context.Background())
	require.ErrorIs(t, err, requestErr)
	require.Empty(t, token)
	token, err = source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "refreshed-token", token)
	require.Equal(t, 3, calls)
}

func TestServiceAccountTokenSourceRejectsInvalidResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		response *authenticationv1.TokenRequest
	}{
		{name: "nil response"},
		{name: "empty token", response: tokenResponse("", time.Now().Add(time.Hour))},
		{name: "missing expiration", response: tokenResponse("token", time.Time{})},
		{name: "expired token", response: tokenResponse("token", time.Now().Add(-time.Second))},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeServiceAccountTokenClient{createToken: func(context.Context, string, *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
				return test.response, nil
			}}
			source, err := NewServiceAccountTokenSource(client, "runner", "audience", "agent", "uid")
			require.NoError(t, err)
			token, err := source.Token(context.Background())
			require.Error(t, err)
			require.Empty(t, token)
		})
	}
}

func TestServiceAccountTokenSourcesDoNotShareTokens(t *testing.T) {
	var requestedAudiences []string
	client := &fakeServiceAccountTokenClient{createToken: func(_ context.Context, _ string, request *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		audience := request.Spec.Audiences[0]
		requestedAudiences = append(requestedAudiences, audience)
		return tokenResponse(audience+"-token", time.Now().Add(time.Hour)), nil
	}}
	first, err := NewServiceAccountTokenSource(client, "runner", "first-run", "first-agent", "first-uid")
	require.NoError(t, err)
	second, err := NewServiceAccountTokenSource(client, "runner", "second-run", "second-agent", "second-uid")
	require.NoError(t, err)
	for range 2 {
		token, err := first.Token(context.Background())
		require.NoError(t, err)
		require.Equal(t, "first-run-token", token)
		token, err = second.Token(context.Background())
		require.NoError(t, err)
		require.Equal(t, "second-run-token", token)
	}
	require.Equal(t, []string{"first-run", "second-run"}, requestedAudiences)
}
