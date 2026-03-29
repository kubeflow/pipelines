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
	"fmt"
	"strings"
	"sync"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const tokenRequestExpirationSeconds int64 = 7200

type serviceAccountTokenSource struct {
	serviceAccounts    typedcorev1.ServiceAccountInterface
	serviceAccountName string
	audience           string
	podName            string
	podUID             string
	now                func() time.Time

	mu        sync.Mutex
	token     string
	refreshAt time.Time
	inFlight  *tokenRequestResult
}

type tokenRequestResult struct {
	done  chan struct{}
	token string
	err   error
}

// NewServiceAccountTokenSource requests and refreshes a token for the given
// service account and audience, bound to the Pod identified by name and UID.
// serviceAccounts must address that Pod's namespace; Kubernetes requires the
// requested service account to match the Pod's serviceAccountName.
func NewServiceAccountTokenSource(serviceAccounts typedcorev1.ServiceAccountInterface, serviceAccountName, audience, podName, podUID string) (TokenSource, error) {
	if serviceAccounts == nil {
		return nil, fmt.Errorf("service account token source requires a Kubernetes service account client")
	}
	for _, field := range []struct{ name, value string }{
		{"service account name", serviceAccountName},
		{"audience", audience},
		{"pod name", podName},
		{"pod UID", podUID},
	} {
		if strings.TrimSpace(field.value) == "" {
			return nil, fmt.Errorf("service account token source requires %s", field.name)
		}
	}
	return &serviceAccountTokenSource{
		serviceAccounts:    serviceAccounts,
		serviceAccountName: serviceAccountName,
		audience:           audience,
		podName:            podName,
		podUID:             podUID,
		now:                time.Now,
	}, nil
}

func (s *serviceAccountTokenSource) Token(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	if s.token != "" && s.now().Before(s.refreshAt) {
		token := s.token
		s.mu.Unlock()
		return token, nil
	}
	if pending := s.inFlight; pending != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-pending.done:
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return pending.token, pending.err
		}
	}
	pending := &tokenRequestResult{done: make(chan struct{})}
	s.inFlight = pending
	s.mu.Unlock()

	token, expiration, err := s.requestToken(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.token = token
		// Use the server's actual lifetime; it may differ from the requested TTL.
		// Refresh after 80% so later RPCs never depend on a nearly expired token.
		now := s.now()
		s.refreshAt = now.Add(expiration.Sub(now) * 4 / 5)
	}
	pending.token, pending.err = token, err
	s.inFlight = nil
	close(pending.done)
	return token, err
}

func (s *serviceAccountTokenSource) requestToken(ctx context.Context) (string, time.Time, error) {
	expirationSeconds := tokenRequestExpirationSeconds
	response, err := s.serviceAccounts.CreateToken(ctx, s.serviceAccountName, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{s.audience},
			ExpirationSeconds: &expirationSeconds,
			BoundObjectRef: &authenticationv1.BoundObjectReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       s.podName,
				UID:        types.UID(s.podUID),
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("request service account token: %w", err)
	}
	if response == nil || response.Status.Token == "" {
		return "", time.Time{}, fmt.Errorf("kubernetes returned an empty service account token")
	}
	expiration := response.Status.ExpirationTimestamp.Time
	if !expiration.After(s.now()) {
		return "", time.Time{}, fmt.Errorf("kubernetes returned an expired service account token")
	}
	return response.Status.Token, expiration, nil
}
