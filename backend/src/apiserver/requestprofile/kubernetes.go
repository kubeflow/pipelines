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
	"net/http"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

type timedLimiter struct{ flowcontrol.RateLimiter }

func (l timedLimiter) Wait(ctx context.Context) error {
	defer StartPhase(ctx, KubernetesThrottle)()
	return l.RateLimiter.Wait(ctx)
}

type timedTransport struct{ underlying http.RoundTripper }

func (t timedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// RoundTrip includes connection/HTTP response-header latency, not body decode.
	defer StartPhase(r.Context(), KubernetesHTTP)()
	return t.underlying.RoundTrip(r)
}

// ConfigureKubernetes preserves client-go's limiter semantics, including its
// invalid-burst validation. Existing limiters/transports are wrapped, not reset.
func ConfigureKubernetes(config *rest.Config) {
	if !Enabled() {
		return
	}
	if config.RateLimiter == nil && config.QPS > 0 && config.Burst > 0 {
		config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(config.QPS, config.Burst)
	}
	if config.RateLimiter != nil {
		config.RateLimiter = timedLimiter{config.RateLimiter}
	}
	config.Wrap(func(transport http.RoundTripper) http.RoundTripper { return timedTransport{transport} })
}
