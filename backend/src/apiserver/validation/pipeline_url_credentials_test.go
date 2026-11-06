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

package validation

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var artifactoryBearerEnviron = []string{
	"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_domain=artifactory.example.com",
	"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_type=bearer",
	"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_token=test-token",
}

func allowAllDomains(string) bool { return true }

type recordingRoundTripper struct {
	requests []*http.Request
	respond  func(*http.Request) *http.Response
}

func (r *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	r.requests = append(r.requests, request)
	if r.respond != nil {
		return r.respond(request), nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody, Request: request}, nil
}

func mustParseCredentials(t *testing.T, environ []string) *pipelineURLCredentials {
	t.Helper()
	credentials, problems := parsePipelineURLCredentials(environ, allowAllDomains)
	require.Empty(t, problems)
	return credentials
}

func setTestCredentialEnv(t *testing.T, domain string) {
	t.Helper()
	t.Setenv("PIPELINE_URL_CREDENTIAL_TEST_domain", domain)
	t.Setenv("PIPELINE_URL_CREDENTIAL_TEST_type", "bearer")
	t.Setenv("PIPELINE_URL_CREDENTIAL_TEST_token", "test-token")
}

func TestParsePipelineURLCredentials_IgnoresOtherVariables(t *testing.T) {
	credentials, problems := parsePipelineURLCredentials([]string{
		"PATH=/usr/bin",
		"PIPELINE_URL_ALLOWED_DOMAINS=artifactory.example.com",
		"PIPELINE_URL_TIMEOUT=30",
	}, allowAllDomains)
	assert.Empty(t, problems)
	assert.Empty(t, credentials.domains())
}

func TestParsePipelineURLCredentials_GroupsFieldsByName(t *testing.T) {
	credentials := mustParseCredentials(t, []string{
		"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_domain=Artifactory.Example.COM.",
		"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_type=Bearer",
		"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_token=dGVzdA==\n",
		"PIPELINE_URL_CREDENTIAL_GIT_MIRROR_DOMAIN=github.example.com",
		"PIPELINE_URL_CREDENTIAL_GIT_MIRROR_TYPE=basic",
		"PIPELINE_URL_CREDENTIAL_GIT_MIRROR_USERNAME=svc-kfp",
		"PIPELINE_URL_CREDENTIAL_GIT_MIRROR_PASSWORD=test-password",
		"PIPELINE_URL_CREDENTIAL_LEGACY_domain=legacy.example.com",
		"PIPELINE_URL_CREDENTIAL_LEGACY_type=header",
		"PIPELINE_URL_CREDENTIAL_LEGACY_headerName=X-JFrog-Art-Api",
		"PIPELINE_URL_CREDENTIAL_LEGACY_headervalue=test-key",
	})

	assert.Equal(t, []string{"artifactory.example.com", "github.example.com", "legacy.example.com"}, credentials.domains())
	artifactory := credentials.byDomain["artifactory.example.com"]
	assert.Equal(t, "ARTIFACTORY", artifactory.name)
	assert.Equal(t, credentialTypeBearer, artifactory.authType)
	assert.Equal(t, "dGVzdA==", artifactory.token, "keeps '=' in values and drops a trailing newline")
	assert.Equal(t, "GIT_MIRROR", credentials.byDomain["github.example.com"].name)
	assert.Equal(t, "svc-kfp", credentials.byDomain["github.example.com"].username)
	assert.Equal(t, "X-JFrog-Art-Api", credentials.byDomain["legacy.example.com"].headerName)
	assert.Equal(t, "test-key", credentials.byDomain["legacy.example.com"].headerValue)
}

func TestParsePipelineURLCredentials_SkipsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name        string
		fields      map[string]string
		errContains string
	}{
		{"missing domain", map[string]string{"type": "bearer", "token": "secret-value"}, "domain is required"},
		{"domain with scheme", map[string]string{"domain": "https://broken.example.com", "type": "bearer", "token": "secret-value"}, "bare hostname"},
		{"domain with port", map[string]string{"domain": "broken.example.com:8443", "type": "bearer", "token": "secret-value"}, "bare hostname"},
		{"domain with path", map[string]string{"domain": "broken.example.com/artifactory", "type": "bearer", "token": "secret-value"}, "bare hostname"},
		{"wildcard domain", map[string]string{"domain": "*.example.com", "type": "bearer", "token": "secret-value"}, "bare hostname"},
		{"non-ASCII domain", map[string]string{"domain": "gİthub.com", "type": "bearer", "token": "secret-value"}, "must use ASCII characters"},
		{"missing type", map[string]string{"domain": "broken.example.com", "token": "secret-value"}, "type is required"},
		{"unsupported type", map[string]string{"domain": "broken.example.com", "type": "digest", "token": "secret-value"}, `type "digest" is not supported`},
		{"bearer without token", map[string]string{"domain": "broken.example.com", "type": "bearer", "password": "secret-value"}, "requires a token"},
		{"bearer token with control character", map[string]string{"domain": "broken.example.com", "type": "bearer", "token": "secret-value\x7f"}, "without control characters"},
		{"basic without password", map[string]string{"domain": "broken.example.com", "type": "basic", "username": "secret-value"}, "requires username and password"},
		{"basic username with colon", map[string]string{"domain": "broken.example.com", "type": "basic", "username": "svc:kfp", "password": "secret-value"}, "must not contain a colon"},
		{"header with invalid name", map[string]string{"domain": "broken.example.com", "type": "header", "headerName": "X Api Key", "headerValue": "secret-value"}, "valid headerName"},
		{"header without value", map[string]string{"domain": "broken.example.com", "type": "header", "headerName": "X-Api-Key"}, "requires a headerValue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environ := slices.Clone(artifactoryBearerEnviron)
			for field, value := range tt.fields {
				environ = append(environ, "PIPELINE_URL_CREDENTIAL_BROKEN_"+field+"="+value)
			}

			credentials, problems := parsePipelineURLCredentials(environ, allowAllDomains)

			require.Len(t, problems, 1)
			assert.Contains(t, problems[0].Error(), "ignoring credential BROKEN")
			assert.Contains(t, problems[0].Error(), tt.errContains)
			assert.NotContains(t, problems[0].Error(), "secret-value", "problems must not reveal credential values")
			assert.Equal(t, []string{"artifactory.example.com"}, credentials.domains(), "valid credentials still load")
		})
	}
}

func TestParsePipelineURLCredentials_SkipsDomainOutsideAllowlist(t *testing.T) {
	credentials, problems := parsePipelineURLCredentials(artifactoryBearerEnviron, func(string) bool { return false })

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Error(), `ignoring credential ARTIFACTORY: domain "artifactory.example.com" is not allowed; add it to PIPELINE_URL_ALLOWED_DOMAINS`)
	assert.Empty(t, credentials.domains())
}

func TestParsePipelineURLCredentials_KeepsCredentialWithUnknownField(t *testing.T) {
	environ := append(slices.Clone(artifactoryBearerEnviron), "PIPELINE_URL_CREDENTIAL_ARTIFACTORY_description=Artifactory")

	credentials, problems := parsePipelineURLCredentials(environ, allowAllDomains)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Error(), `credential ARTIFACTORY: ignoring unknown field "description"`)
	assert.Equal(t, []string{"artifactory.example.com"}, credentials.domains())
}

func TestParsePipelineURLCredentials_SkipsCredentialsSharingADomain(t *testing.T) {
	environ := append(slices.Clone(artifactoryBearerEnviron),
		"PIPELINE_URL_CREDENTIAL_MIRROR_domain=ARTIFACTORY.example.com",
		"PIPELINE_URL_CREDENTIAL_MIRROR_type=bearer",
		"PIPELINE_URL_CREDENTIAL_MIRROR_token=other-token",
	)

	credentials, problems := parsePipelineURLCredentials(environ, allowAllDomains)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Error(), `ignoring credentials ARTIFACTORY, MIRROR: they share domain "artifactory.example.com"`)
	assert.Empty(t, credentials.domains())
}

func TestParsePipelineURLCredentials_SkipsFieldSetTwice(t *testing.T) {
	environ := append(slices.Clone(artifactoryBearerEnviron), "PIPELINE_URL_CREDENTIAL_ARTIFACTORY_TOKEN=other-token")

	credentials, problems := parsePipelineURLCredentials(environ, allowAllDomains)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Error(), "ignoring credential ARTIFACTORY: field TOKEN is set more than once")
	assert.NotContains(t, problems[0].Error(), "other-token")
	assert.Empty(t, credentials.domains())
}

func TestParsePipelineURLCredentials_ReportsMalformedVariableNames(t *testing.T) {
	credentials, problems := parsePipelineURLCredentials([]string{
		"PIPELINE_URL_CREDENTIAL_token=secret-value",
		"PIPELINE_URL_CREDENTIAL__token=secret-value",
		"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_=secret-value",
		"PIPELINE_URL_CREDENTIAL_ARTIFACTORYtoken=secret-value",
	}, allowAllDomains)

	require.Len(t, problems, 4)
	for _, problem := range problems {
		assert.Contains(t, problem.Error(), "name it PIPELINE_URL_CREDENTIAL_<NAME>_<field>")
		assert.NotContains(t, problem.Error(), "secret-value")
	}
	assert.Empty(t, credentials.domains())
}

func TestPipelineURLCredential_StringOmitsSecrets(t *testing.T) {
	credential := pipelineURLCredential{name: "GIT", domain: "a.example.com", authType: credentialTypeBasic, username: "svc-kfp", password: "secret-value"}
	for _, formatted := range []string{fmt.Sprint(credential), fmt.Sprintf("%+v", credential)} {
		assert.Contains(t, formatted, "a.example.com")
		assert.NotContains(t, formatted, "secret-value")
	}
}

func TestCredentialTransport_AddsCredentialForConfiguredDomain(t *testing.T) {
	tests := []struct {
		name       string
		environ    []string
		headerName string
		want       string
	}{
		{"bearer", artifactoryBearerEnviron, "Authorization", "Bearer test-token"},
		{
			"basic",
			[]string{
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_domain=artifactory.example.com",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_type=basic",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_username=svc-kfp",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_password=test-password",
			},
			"Authorization",
			"Basic c3ZjLWtmcDp0ZXN0LXBhc3N3b3Jk",
		},
		{
			"header",
			[]string{
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_domain=artifactory.example.com",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_type=header",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_headerName=X-JFrog-Art-Api",
				"PIPELINE_URL_CREDENTIAL_ARTIFACTORY_headerValue=test-key",
			},
			"X-JFrog-Art-Api",
			"test-key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &recordingRoundTripper{}
			transport := &credentialTransport{base: base, credentials: mustParseCredentials(t, tt.environ)}
			request, err := http.NewRequest(http.MethodGet, "https://Artifactory.Example.com./artifactory/ml/pipeline.yaml", nil)
			require.NoError(t, err)

			_, err = transport.RoundTrip(request)
			require.NoError(t, err)

			require.Len(t, base.requests, 1)
			assert.Equal(t, tt.want, base.requests[0].Header.Get(tt.headerName))
			assert.Empty(t, request.Header.Get(tt.headerName), "the caller's request must not be modified")
		})
	}
}

func TestCredentialTransport_SkipsOtherHostsAndPlainHTTP(t *testing.T) {
	credentials := mustParseCredentials(t, artifactoryBearerEnviron)
	for _, target := range []string{
		"https://github.com/org/repo/pipeline.yaml",
		"https://artifactory.example.com.attacker.example/pipeline.yaml",
		"https://mirror.artifactory.example.com/pipeline.yaml",
		"http://artifactory.example.com/pipeline.yaml",
		"https://artifactory.example.com:8443/pipeline.yaml",
	} {
		base := &recordingRoundTripper{}
		transport := &credentialTransport{base: base, credentials: credentials}
		request, err := http.NewRequest(http.MethodGet, target, nil)
		require.NoError(t, err)

		_, err = transport.RoundTrip(request)
		require.NoError(t, err)

		require.Len(t, base.requests, 1)
		assert.Empty(t, base.requests[0].Header.Get("Authorization"), "no credential expected for %s", target)
	}
}

func TestCredentialTransport_DropsCredentialOnCrossHostRedirect(t *testing.T) {
	const presignedURL = "https://bucket.s3.amazonaws.com/pipeline.yaml?X-Amz-Signature=abc"
	base := &recordingRoundTripper{respond: func(request *http.Request) *http.Response {
		if request.URL.Hostname() == "artifactory.example.com" {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {presignedURL}}, Body: http.NoBody, Request: request}
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody, Request: request}
	}}
	client := &http.Client{Transport: &credentialTransport{base: base, credentials: mustParseCredentials(t, artifactoryBearerEnviron)}}

	response, err := client.Get("https://artifactory.example.com/artifactory/ml/pipeline.yaml")
	require.NoError(t, err)
	defer response.Body.Close()

	require.Len(t, base.requests, 2)
	assert.Equal(t, "Bearer test-token", base.requests[0].Header.Get("Authorization"))
	assert.Equal(t, "bucket.s3.amazonaws.com", base.requests[1].URL.Hostname())
	assert.Empty(t, base.requests[1].Header.Get("Authorization"), "the credential must not follow a redirect to another host")
}

func TestCredentialTransport_SendsNoCredentialToUnicodeLookalikeHost(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Authorization"))
	}))
	defer server.Close()

	// Every host reaches the test server, as it would for someone who owns the
	// lookalike host and holds a trusted certificate for it.
	dialed := make(chan string, 2)
	base := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed <- address
			return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: &credentialTransport{base: base, credentials: mustParseCredentials(t, []string{
		"PIPELINE_URL_CREDENTIAL_GITHUB_domain=github.com",
		"PIPELINE_URL_CREDENTIAL_GITHUB_type=bearer",
		"PIPELINE_URL_CREDENTIAL_GITHUB_token=test-token",
	})}}

	for _, tt := range []struct{ target, wantDialed, wantAuthorization string }{
		{"https://github.com/pipeline.yaml", "github.com:443", "Bearer test-token"},
		// U+0130 lowercases to "github.com" but net/http dials its IDNA form.
		{"https://gİthub.com/pipeline.yaml", "xn--github-qyd.com:443", ""},
	} {
		response, err := client.Get(tt.target)
		require.NoError(t, err)
		received, err := io.ReadAll(response.Body)
		response.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, tt.wantDialed, <-dialed)
		assert.Equal(t, tt.wantAuthorization, string(received), "credential received by %s", tt.wantDialed)
	}
}

func TestNewPipelineHTTPClient_SendsCredentialOnlyToDefaultHTTPSPort(t *testing.T) {
	viper.Reset()
	// Disabled validation no longer rejects ports other than 443.
	viper.Set(common.PipelineURLValidationEnabled, "false")
	resetURLConfig()
	setTestCredentialEnv(t, "127.0.0.1")

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Authorization"))
	}))
	defer server.Close()
	// Every address reaches the test server, so port 443 can be requested.
	base := server.Client().Transport.(*http.Transport).Clone()
	base.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client := NewPipelineHTTPClient(base)

	for _, tt := range []struct{ target, wantAuthorization string }{
		{"https://127.0.0.1/pipeline.yaml", "Bearer test-token"},
		{"https://127.0.0.1:443/pipeline.yaml", "Bearer test-token"},
		{server.URL + "/pipeline.yaml", ""},
	} {
		response, err := client.Get(tt.target)
		require.NoError(t, err)
		received, err := io.ReadAll(response.Body)
		response.Body.Close()
		require.NoError(t, err)

		assert.Equal(t, tt.wantAuthorization, string(received), "credential received from %s", tt.target)
	}
}

func TestSafePipelineHTTPClient_CredentialDomainMustBeAllowed(t *testing.T) {
	viper.Reset()
	resetURLConfig()
	setTestCredentialEnv(t, "artifactory.example.com")

	_, plain := SafePipelineHTTPClient().Transport.(*http.Transport)
	assert.True(t, plain, "a credential for a domain outside the allowlist is ignored")

	viper.Set(common.PipelineURLAllowedDomains, "artifactory.example.com")
	resetURLConfig()
	_, wrapped := SafePipelineHTTPClient().Transport.(*credentialTransport)
	assert.True(t, wrapped, "the credential is used once its domain is allowed")
}

func TestNewPipelineHTTPClient_CredentialKeepsProxyAndIPGuards(t *testing.T) {
	viper.Reset()
	viper.Set(common.PipelineURLAllowedDomains, "127.0.0.1")
	resetURLConfig()
	setTestCredentialEnv(t, "127.0.0.1")

	base := http.DefaultTransport.(*http.Transport).Clone()
	proxyUsed := false
	base.Proxy = func(*http.Request) (*url.URL, error) {
		proxyUsed = true
		return url.Parse("http://proxy.invalid:3128")
	}
	client := NewPipelineHTTPClient(base)
	transport, wrapped := client.Transport.(*credentialTransport)
	require.True(t, wrapped, "the allowed credential is configured")
	target, err := url.Parse("https://127.0.0.1/pipeline.yaml")
	require.NoError(t, err)
	_, found := transport.credentials.forURL(target)
	require.True(t, found, "the request carries the credential")

	_, err = client.Get(target.String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection to blocked IP range denied")
	assert.False(t, proxyUsed, "the base transport's proxy must not be used")
}

func TestSafePipelineHTTPClient_InvalidCredentialSendsNone(t *testing.T) {
	viper.Reset()
	resetURLConfig()
	t.Setenv("PIPELINE_URL_CREDENTIAL_TEST_domain", "storage.googleapis.com")
	t.Setenv("PIPELINE_URL_CREDENTIAL_TEST_type", "bearer")

	_, plain := SafePipelineHTTPClient().Transport.(*http.Transport)
	assert.True(t, plain, "an invalid credential leaves the transport without credentials")
}
