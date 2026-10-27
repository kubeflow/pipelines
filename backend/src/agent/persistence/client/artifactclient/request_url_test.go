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

package artifactclient

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type artifactRoundTrip func(*http.Request) (*http.Response, error)

func (f artifactRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReadArtifactRejectsRouteSegmentsBeforeTransport(t *testing.T) {
	for _, segment := range []string{"", ".", "..", "../other", "a/b", `a\b`, "%2e%2e", "%252f", "?other", "#fragment", "control\x01"} {
		for _, field := range []string{"run", "node", "artifact"} {
			t.Run(field+"/"+segment, func(t *testing.T) {
				calls := 0
				transport := artifactRoundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
				})
				c := NewClient("https://api.invalid/prefix", &http.Client{Transport: transport}, newTestTokenRefresher(t))
				request := &ReadArtifactRequest{RunID: "run", NodeID: "node", ArtifactName: "metrics"}
				switch field {
				case "run":
					request.RunID = segment
				case "node":
					request.NodeID = segment
				case "artifact":
					request.ArtifactName = segment
				}
				_, err := c.ReadArtifact(request)
				require.Error(t, err)
				require.Zero(t, calls)
			})
		}
	}
}

func TestReadArtifactDoesNotFollowRedirects(t *testing.T) {
	for _, location := range []string{"/other", "https://other.invalid/path"} {
		t.Run(location, func(t *testing.T) {
			calls, originalPolicy := 0, 0
			transport := artifactRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{location}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			})
			original := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { originalPolicy++; return http.ErrUseLastResponse }}
			c := NewClient("https://api.invalid", original, newTestTokenRefresher(t))
			_, err := c.ReadArtifact(&ReadArtifactRequest{RunID: "run", NodeID: "node", ArtifactName: "metrics"})
			require.ErrorContains(t, err, "redirect rejected")
			var artifactErr *Error
			require.ErrorAs(t, err, &artifactErr)
			require.Equal(t, ErrorCodePermanent, artifactErr.Code)
			require.Equal(t, 1, calls)
			require.Zero(t, originalPolicy)
			require.ErrorIs(t, original.CheckRedirect(nil, nil), http.ErrUseLastResponse)
			require.Equal(t, 1, originalPolicy)
			copied := c.(*client).httpClient
			require.Equal(t, original.Timeout, copied.Timeout)
			require.Equal(t, reflect.ValueOf(original.Transport).Pointer(), reflect.ValueOf(copied.Transport).Pointer())
		})
	}
}

func TestReadArtifactPreservesCanonicalBasePath(t *testing.T) {
	calls := 0
	transport := artifactRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "api.invalid", r.URL.Host)
		require.Equal(t, "/prefix/apis/v1beta1/runs/run-1/nodes/node.id/artifacts/metrics name:read", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	})
	c := NewClient("https://api.invalid/prefix/", &http.Client{Transport: transport}, newTestTokenRefresher(t))
	result, err := c.ReadArtifact(&ReadArtifactRequest{RunID: "run-1", NodeID: "node.id", ArtifactName: "metrics name"})
	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, 1, calls)
}

func TestArtifactURLValidationPrecedesTokenAccess(t *testing.T) {
	c := NewClient("https://api.invalid", http.DefaultClient, nil)
	require.NotPanics(t, func() {
		_, err := c.ReadArtifact(&ReadArtifactRequest{RunID: "run", NodeID: "../node", ArtifactName: "metrics"})
		require.Error(t, err)
	})
}

func TestArtifactRequestURL(t *testing.T) {
	request := &ReadArtifactRequest{RunID: "run@1", NodeID: "node+id", ArtifactName: "metrics name"}
	for _, base := range []string{"https://api.invalid", "http://api.invalid/prefix", "http://api.invalid/prefix/"} {
		u, err := artifactRequestURL(base, request)
		require.NoError(t, err)
		require.Contains(t, u, "metrics%20name:read")
	}
	for _, base := range []string{"ftp://api.invalid", "https://user:pass@api.invalid", "https://api.invalid?q=x", "https://api.invalid?", "https://api.invalid#", "https://api.invalid/a/../b", "https://api.invalid/a//b", "https://api.invalid/a%2f../b"} {
		_, err := artifactRequestURL(base, request)
		require.Error(t, err, base)
	}
	_, err := artifactRequestURL("https://api.invalid", nil)
	require.Error(t, err)
}

func TestReadArtifactStopsDefaultRedirectFollowing(t *testing.T) {
	calls := 0
	transport := artifactRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"/redirected"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	c := NewClient("https://api.invalid", &http.Client{Transport: transport}, newTestTokenRefresher(t))
	_, err := c.ReadArtifact(&ReadArtifactRequest{RunID: "run", NodeID: "node", ArtifactName: "metrics"})
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
