// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package requestprofile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

const testID = "12345678-1234-1234-1234-123456789abc"

func requestContext(ids ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.MD{MetadataKey: ids, "authorization": {"Bearer secret-value"}})
}
func TestExplicitOptInAndBoundedRecord(t *testing.T) {
	for _, tc := range []struct {
		enabled bool
		ids     []string
		want    int
	}{
		{false, []string{testID}, 0}, {true, nil, 0}, {true, []string{testID, testID}, 0},
		{true, []string{"untrusted\nvalue"}, 0}, {true, []string{strings.Repeat("x", 10000)}, 0}, {true, []string{testID}, 1},
	} {
		count := 0
		ctx, finish := start(requestContext(tc.ids...), "CreateTask", tc.enabled, func(data []byte) {
			count++
			if len(data) > 2048 || strings.Contains(string(data), "secret") {
				t.Fatalf("Unexpected record %s", data)
			}
			var got record
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.CorrelationID != testID || got.Method != "CreateTask" || got.PhasesMS[TaskCreate] <= 0 {
				t.Fatalf("Bad record %+v", got)
			}
			if _, ok := got.PhasesMS[Phase("untrusted")]; ok {
				t.Fatal("unknown phase recorded")
			}
		})
		StartPhase(ctx, TaskCreate)()
		StartPhase(ctx, Phase("untrusted"))()
		finish()
		finish()
		if count != tc.want {
			t.Fatalf("got %d records, want %d", count, tc.want)
		}
	}
}
func TestPhasePreservesContextAndAccumulatesNestedCalls(t *testing.T) {
	ctx, finish := start(requestContext(testID), "UpdateTask", true, func(data []byte) {
		var got record
		_ = json.Unmarshal(data, &got)
		if got.PhasesMS[Authorization] < got.PhasesMS[TokenReview] || got.TotalMS < got.PhasesMS[Authorization] {
			t.Fatalf("Invalid inclusive phases %+v", got)
		}
	})
	authorization := StartPhase(ctx, Authorization)
	for range 2 {
		end := StartPhase(ctx, TokenReview)
		time.Sleep(time.Millisecond)
		end()
	}
	authorization()
	finish()
	if values := metadata.ValueFromIncomingContext(ctx, "authorization"); len(values) != 1 || values[0] != "Bearer secret-value" {
		t.Fatal("request metadata changed")
	}
}

type limiter struct {
	flowcontrol.RateLimiter
	calls int
	err   error
}

func (l *limiter) Wait(context.Context) error { l.calls++; return l.err }

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestKubernetesTimingPreservesLimiterTransportAndErrors(t *testing.T) {
	t.Setenv(Environment, "true")
	expectedErr := errors.New("unchanged error")
	underlying := &limiter{RateLimiter: flowcontrol.NewFakeAlwaysRateLimiter(), err: expectedErr}
	wrappedCalls := 0
	config := &rest.Config{QPS: 5, Burst: 10, RateLimiter: underlying, WrapTransport: func(rt http.RoundTripper) http.RoundTripper { wrappedCalls++; return rt }}
	ConfigureKubernetes(config)
	ctx, finish := start(requestContext(testID), "GetTask", true, func(data []byte) {
		var got record
		_ = json.Unmarshal(data, &got)
		if got.PhasesMS[KubernetesThrottle] <= 0 || got.PhasesMS[KubernetesHTTP] <= 0 {
			t.Fatalf("Missing client phases %+v", got)
		}
	})
	if err := config.RateLimiter.Wait(ctx); err != expectedErr || underlying.calls != 1 {
		t.Fatal("limiter behavior changed")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://kubernetes.invalid", nil)
	req.Header.Set("Authorization", "Bearer secret-value")
	rt := config.WrapTransport(transportFunc(func(actual *http.Request) (*http.Response, error) {
		if actual != req || actual.Header.Get("Authorization") != "Bearer secret-value" {
			t.Fatal("request changed")
		}
		return nil, expectedErr
	}))
	if _, err := rt.RoundTrip(req); err != expectedErr || wrappedCalls != 1 {
		t.Fatal("transport behavior changed")
	}
	if config.QPS != 5 || config.Burst != 10 {
		t.Fatal("rate limits changed")
	}
	finish()
}
func TestKubernetesDisabledAndInvalidConfigUnchanged(t *testing.T) {
	t.Setenv(Environment, "")
	config := &rest.Config{QPS: 5, Burst: 10}
	ConfigureKubernetes(config)
	if config.RateLimiter != nil || config.WrapTransport != nil {
		t.Fatal("disabled configuration changed")
	}
	t.Setenv(Environment, "true")
	config = &rest.Config{QPS: 5, Burst: 0}
	ConfigureKubernetes(config)
	if config.RateLimiter != nil {
		t.Fatal("invalid client-go burst silently accepted")
	}
}
