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

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseOptionalInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int64
	}{
		{"empty string returns nil", "", nil},
		{"whitespace returns nil", "  ", nil},
		{"valid positive", "1000", int64Ptr(1000)},
		{"zero", "0", int64Ptr(0)},
		{"negative returns nil", "-1", nil},
		{"invalid string returns nil", "abc", nil},
		{"leading/trailing whitespace trimmed", " 42 ", int64Ptr(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptionalInt64(tt.input)
			if tt.expected == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, *tt.expected, *got)
			}
		})
	}
}

func TestParseOptionalBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *bool
	}{
		{"empty string returns nil", "", nil},
		{"whitespace returns nil", "  ", nil},
		{"true", "true", boolPtr(true)},
		{"false", "false", boolPtr(false)},
		{"True (case insensitive)", "True", boolPtr(true)},
		{"FALSE (case insensitive)", "FALSE", boolPtr(false)},
		{"1 is true", "1", boolPtr(true)},
		{"0 is false", "0", boolPtr(false)},
		{"invalid string returns nil", "abc", nil},
		{"leading/trailing whitespace trimmed", " true ", boolPtr(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptionalBool(tt.input)
			if tt.expected == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, *tt.expected, *got)
			}
		})
	}
}

func TestGrpcCustomMatcher(t *testing.T) {
	tests := []struct {
		name          string
		headerKey     string
		expectedKey   string
		expectedMatch bool
	}{
		{
			name:          "matching kubeflow user ID header",
			headerKey:     common.GoogleIAPUserIdentityHeader,
			expectedKey:   common.GoogleIAPUserIdentityHeader,
			expectedMatch: true,
		},
		{
			name:          "matching header case insensitive",
			headerKey:     "X-Goog-Authenticated-User-Email",
			expectedKey:   "x-goog-authenticated-user-email",
			expectedMatch: true,
		},
		{
			name:          "non-matching header returns false",
			headerKey:     "Authorization",
			expectedKey:   "authorization",
			expectedMatch: false,
		},
		{
			name:          "empty header returns false",
			headerKey:     "",
			expectedKey:   "",
			expectedMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, matched := grpcCustomMatcher(tt.headerKey)
			assert.Equal(t, tt.expectedKey, key)
			assert.Equal(t, tt.expectedMatch, matched)
		})
	}
}

func TestApiServerInterceptor(t *testing.T) {
	t.Run("successful handler call", func(t *testing.T) {
		expectedResponse := "success"
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return expectedResponse, nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/api.TestService/TestMethod"}

		response, err := apiServerInterceptor(context.Background(), "request", info, handler)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, response)
	})

	t.Run("handler returns error", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, fmt.Errorf("something went wrong")
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/api.TestService/FailMethod"}

		response, err := apiServerInterceptor(context.Background(), "request", info, handler)
		assert.Nil(t, response)
		assert.Error(t, err)
		// The interceptor converts errors to gRPC status errors
		grpcStatus, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, grpcStatus.Code())
	})
}

func TestInitCerts(t *testing.T) {
	originalCertPath := *tlsCertPath
	originalKeyPath := *tlsCertKeyPath
	t.Cleanup(func() {
		*tlsCertPath = originalCertPath
		*tlsCertKeyPath = originalKeyPath
	})

	t.Run("no certs provided returns nil", func(t *testing.T) {
		*tlsCertPath = ""
		*tlsCertKeyPath = ""

		tlsCfg, err := initCerts()
		assert.NoError(t, err)
		assert.Nil(t, tlsCfg)
	})

	t.Run("cert path without key path returns error", func(t *testing.T) {
		*tlsCertPath = "/some/cert.pem"
		*tlsCertKeyPath = ""

		tlsCfg, err := initCerts()
		assert.Error(t, err)
		assert.Nil(t, tlsCfg)
		assert.Contains(t, err.Error(), "missing tlsCertKeyPath")
	})

	t.Run("key path without cert path returns error", func(t *testing.T) {
		*tlsCertPath = ""
		*tlsCertKeyPath = "/some/key.pem"

		tlsCfg, err := initCerts()
		assert.Error(t, err)
		assert.Nil(t, tlsCfg)
		assert.Contains(t, err.Error(), "missing tlsCertPath")
	})

	t.Run("invalid cert files returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		certFile := filepath.Join(tempDir, "cert.pem")
		keyFile := filepath.Join(tempDir, "key.pem")
		require.NoError(t, os.WriteFile(certFile, []byte("not a cert"), 0600))
		require.NoError(t, os.WriteFile(keyFile, []byte("not a key"), 0600))

		*tlsCertPath = certFile
		*tlsCertKeyPath = keyFile

		tlsCfg, err := initCerts()
		assert.Error(t, err)
		assert.Nil(t, tlsCfg)
	})

	t.Run("valid cert files returns tls config", func(t *testing.T) {
		tempDir := t.TempDir()
		certFile := filepath.Join(tempDir, "cert.pem")
		keyFile := filepath.Join(tempDir, "key.pem")
		generateSelfSignedCert(t, certFile, keyFile)

		*tlsCertPath = certFile
		*tlsCertKeyPath = keyFile

		tlsCfg, err := initCerts()
		assert.NoError(t, err)
		assert.NotNil(t, tlsCfg)
		assert.Len(t, tlsCfg.Certificates, 1)
	})
}

func TestGetPVCSpec(t *testing.T) {
	t.Run("no workspace config returns nil", func(t *testing.T) {
		viper.Reset()

		pvcSpec, err := getPVCSpec()
		assert.NoError(t, err)
		assert.Nil(t, pvcSpec)
	})

	t.Run("valid workspace config returns PVC spec", func(t *testing.T) {
		viper.Reset()
		storageName := "standard"
		viper.Set("workspace.volumeclaimtemplatespec.accessmodes", []string{"ReadWriteOnce"})
		viper.Set("workspace.volumeclaimtemplatespec.storageclassname", storageName)

		pvcSpec, err := getPVCSpec()
		require.NoError(t, err)
		require.NotNil(t, pvcSpec)
		require.NotNil(t, pvcSpec.StorageClassName)
		assert.Equal(t, storageName, *pvcSpec.StorageClassName)
	})

	t.Run("missing access modes returns error", func(t *testing.T) {
		viper.Reset()
		storageName := "standard"
		viper.Set("workspace.volumeclaimtemplatespec.storageclassname", storageName)

		pvcSpec, err := getPVCSpec()
		assert.Error(t, err)
		assert.Nil(t, pvcSpec)
		assert.Contains(t, err.Error(), "must specify accessModes and storageClassName")
	})

	t.Run("missing storage class name returns error", func(t *testing.T) {
		viper.Reset()
		viper.Set("workspace.volumeclaimtemplatespec.accessmodes", []string{"ReadWriteOnce"})

		pvcSpec, err := getPVCSpec()
		assert.Error(t, err)
		assert.Nil(t, pvcSpec)
		assert.Contains(t, err.Error(), "must specify accessModes and storageClassName")
	})

	t.Run("empty storage class name returns error", func(t *testing.T) {
		viper.Reset()
		viper.Set("workspace.volumeclaimtemplatespec.accessmodes", []string{"ReadWriteOnce"})
		viper.Set("workspace.volumeclaimtemplatespec.storageclassname", "")

		pvcSpec, err := getPVCSpec()
		assert.Error(t, err)
		assert.Nil(t, pvcSpec)
	})

	t.Cleanup(func() {
		viper.Reset()
	})
}

// generateSelfSignedCert creates a self-signed TLS certificate and key for testing.
func generateSelfSignedCert(t *testing.T, certPath, keyPath string) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	err = os.WriteFile(certPath, certPEM, 0600)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	err = os.WriteFile(keyPath, keyPEM, 0600)
	require.NoError(t, err)
}

// Subtests must not run in parallel: they mutate package-level flags.
func TestInitConfig(t *testing.T) {
	t.Run("success with valid config file", func(t *testing.T) {
		viper.Reset()
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "config.json")
		require.NoError(t, os.WriteFile(configFile, []byte(`{}`), 0600))

		originalConfigPath := *configPath
		*configPath = tempDir
		t.Cleanup(func() {
			*configPath = originalConfigPath
			viper.Reset()
		})

		err := initConfig()
		assert.NoError(t, err)
	})

	t.Run("error with missing config file", func(t *testing.T) {
		viper.Reset()
		originalConfigPath := *configPath
		*configPath = "/nonexistent/path"
		t.Cleanup(func() {
			*configPath = originalConfigPath
			viper.Reset()
		})

		err := initConfig()
		assert.Error(t, err)
		var configNotFound viper.ConfigFileNotFoundError
		assert.ErrorAs(t, err, &configNotFound)
	})
}

// Subtests must not run in parallel: they mutate package-level flags.
func TestRegisterHTTPHandlerFromEndpoint(t *testing.T) {
	t.Run("success without TLS", func(t *testing.T) {
		handler := func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			return nil
		}
		serveMux := runtime.NewServeMux()

		err := registerHTTPHandlerFromEndpoint(context.Background(), handler, "TestService", serveMux, nil)
		assert.NoError(t, err)
	})

	t.Run("success with TLS", func(t *testing.T) {
		handler := func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			return nil
		}
		serveMux := runtime.NewServeMux()
		tlsCfg := &tls.Config{}

		err := registerHTTPHandlerFromEndpoint(context.Background(), handler, "TestService", serveMux, tlsCfg)
		assert.NoError(t, err)
	})

	t.Run("handler returns error", func(t *testing.T) {
		handler := func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			return fmt.Errorf("registration failed")
		}
		serveMux := runtime.NewServeMux()

		err := registerHTTPHandlerFromEndpoint(context.Background(), handler, "FailService", serveMux, nil)
		assert.ErrorContains(t, err, "failed to register FailService handler")
		assert.ErrorContains(t, err, "registration failed")
	})

	t.Run("uses configured RPC port in endpoint", func(t *testing.T) {
		originalPort := *rpcPortFlag
		*rpcPortFlag = ":9999"
		t.Cleanup(func() { *rpcPortFlag = originalPort })

		var capturedEndpoint string
		handler := func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			capturedEndpoint = endpoint
			return nil
		}
		serveMux := runtime.NewServeMux()

		err := registerHTTPHandlerFromEndpoint(context.Background(), handler, "TestService", serveMux, nil)
		assert.NoError(t, err)
		assert.Equal(t, "localhost:9999", capturedEndpoint)
	})
}

func TestResolveWebhookTLSPaths(t *testing.T) {
	allFilesExist := func(string) bool { return true }
	noFilesExist := func(string) bool { return false }

	t.Run("webhook-specific paths used when both are set and files exist", func(t *testing.T) {
		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			"/webhook/cert.pem", "/webhook/key.pem",
			"/server/cert.pem", "/server/key.pem",
			allFilesExist,
		)
		assert.NoError(t, err)
		assert.True(t, useTLS)
		assert.Equal(t, "/webhook/cert.pem", certPath)
		assert.Equal(t, "/webhook/key.pem", keyPath)
	})

	t.Run("webhook paths set but files missing returns error", func(t *testing.T) {
		_, _, useTLS, err := resolveWebhookTLSPaths(
			"/webhook/cert.pem", "/webhook/key.pem",
			"/server/cert.pem", "/server/key.pem",
			noFilesExist,
		)
		assert.Error(t, err)
		assert.False(t, useTLS)
		assert.Contains(t, err.Error(), "webhook TLS certificate/key paths are set but files do not exist")
	})

	t.Run("falls back to server paths when webhook paths are empty", func(t *testing.T) {
		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			"", "",
			"/server/cert.pem", "/server/key.pem",
			allFilesExist,
		)
		assert.NoError(t, err)
		assert.True(t, useTLS)
		assert.Equal(t, "/server/cert.pem", certPath)
		assert.Equal(t, "/server/key.pem", keyPath)
	})

	t.Run("server paths set but files missing returns error", func(t *testing.T) {
		_, _, useTLS, err := resolveWebhookTLSPaths(
			"", "",
			"/server/cert.pem", "/server/key.pem",
			noFilesExist,
		)
		assert.Error(t, err)
		assert.False(t, useTLS)
		assert.Contains(t, err.Error(), "API server TLS certificate/key paths are set but files do not exist")
	})

	t.Run("no TLS paths configured returns useTLS false", func(t *testing.T) {
		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			"", "",
			"", "",
			allFilesExist,
		)
		assert.NoError(t, err)
		assert.False(t, useTLS)
		assert.Empty(t, certPath)
		assert.Empty(t, keyPath)
	})

	t.Run("webhook cert missing but key exists checks both", func(t *testing.T) {
		fileExistsOnlyKey := func(path string) bool {
			return path == "/webhook/key.pem"
		}
		_, _, useTLS, err := resolveWebhookTLSPaths(
			"/webhook/cert.pem", "/webhook/key.pem",
			"", "",
			fileExistsOnlyKey,
		)
		assert.Error(t, err)
		assert.False(t, useTLS)
	})

	t.Run("server cert exists but key missing returns error", func(t *testing.T) {
		fileExistsOnlyCert := func(path string) bool {
			return path == "/server/cert.pem"
		}
		_, _, useTLS, err := resolveWebhookTLSPaths(
			"", "",
			"/server/cert.pem", "/server/key.pem",
			fileExistsOnlyCert,
		)
		assert.Error(t, err)
		assert.False(t, useTLS)
	})

	t.Run("only webhook cert path set (no key) falls through to server paths", func(t *testing.T) {
		// When only one of the webhook paths is set, it doesn't match the
		// "both set" condition, so it falls through to server path logic.
		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			"/webhook/cert.pem", "",
			"/server/cert.pem", "/server/key.pem",
			allFilesExist,
		)
		assert.NoError(t, err)
		assert.True(t, useTLS)
		assert.Equal(t, "/server/cert.pem", certPath)
		assert.Equal(t, "/server/key.pem", keyPath)
	})

	t.Run("only server cert path set (no key) returns no TLS", func(t *testing.T) {
		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			"", "",
			"/server/cert.pem", "",
			allFilesExist,
		)
		assert.NoError(t, err)
		assert.False(t, useTLS)
		assert.Empty(t, certPath)
		assert.Empty(t, keyPath)
	})

	t.Run("uses real temp files to verify file existence check", func(t *testing.T) {
		tempDir := t.TempDir()
		certFile := filepath.Join(tempDir, "cert.pem")
		keyFile := filepath.Join(tempDir, "key.pem")
		require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0600))
		require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0600))

		certPath, keyPath, useTLS, err := resolveWebhookTLSPaths(
			certFile, keyFile,
			"", "",
			common.FileExists,
		)
		assert.NoError(t, err)
		assert.True(t, useTLS)
		assert.Equal(t, certFile, certPath)
		assert.Equal(t, keyFile, keyPath)
	})

	t.Run("real temp files only cert exists returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		certFile := filepath.Join(tempDir, "cert.pem")
		keyFile := filepath.Join(tempDir, "key.pem")
		require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0600))
		// keyFile intentionally not created

		_, _, useTLS, err := resolveWebhookTLSPaths(
			certFile, keyFile,
			"", "",
			common.FileExists,
		)
		assert.Error(t, err)
		assert.False(t, useTLS)
	})
}

func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }

func TestClearTagsMiddleware(t *testing.T) {
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Got-Clear-Tags", r.Header.Get(common.ClearTagsMetadataKey))
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Body", string(body))
		w.WriteHeader(http.StatusOK)
	})
	handler := clearTagsMiddleware(downstream)

	tests := []struct {
		name       string
		method     string
		body       string
		wantHeader string
	}{
		{
			name:       "PUT with empty tags sets header",
			method:     http.MethodPut,
			body:       `{"tags":{}}`,
			wantHeader: "true",
		},
		{
			name:       "PUT with non-empty tags does not set header",
			method:     http.MethodPut,
			body:       `{"tags":{"k":"v"}}`,
			wantHeader: "",
		},
		{
			name:       "PUT without tags does not set header",
			method:     http.MethodPut,
			body:       `{"display_name":"foo"}`,
			wantHeader: "",
		},
		{
			name:       "GET request is ignored",
			method:     http.MethodGet,
			body:       "",
			wantHeader: "",
		},
		{
			name:       "POST request is ignored",
			method:     http.MethodPost,
			body:       `{"tags":{}}`,
			wantHeader: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, "/apis/v2beta1/pipelines/some-id", bodyReader)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantHeader, rr.Header().Get("X-Got-Clear-Tags"))
			if tt.body != "" {
				assert.Equal(t, tt.body, rr.Header().Get("X-Body"))
			}
		})
	}
}

func TestGrpcCustomMatcher_ClearTags(t *testing.T) {
	key, ok := grpcCustomMatcher(common.ClearTagsMetadataKey)
	assert.True(t, ok)
	assert.Equal(t, common.ClearTagsMetadataKey, key)

	key, ok = grpcCustomMatcher("X-CLEAR-TAGS")
	assert.True(t, ok)
	assert.Equal(t, common.ClearTagsMetadataKey, key)
}

func TestClearTagsMiddleware_BodyPreserved(t *testing.T) {
	original := `{"tags":{},"display_name":"test"}`
	var capturedBody []byte
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	handler := clearTagsMiddleware(downstream)

	req := httptest.NewRequest(http.MethodPut, "/test", bytes.NewBufferString(original))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, original, string(capturedBody))
}

func noOpHandler(w http.ResponseWriter, r *http.Request) {}

func newNoOpHTTPRouterDeps() HTTPRouterDeps {
	return HTTPRouterDeps{
		UploadPipelineV1:        noOpHandler,
		UploadPipelineVersionV1: noOpHandler,
		UploadPipeline:          noOpHandler,
		UploadPipelineVersion:   noOpHandler,
		ReadRunLogV1:            noOpHandler,
		ReadArtifactV1:          noOpHandler,
		ReadArtifact:            noOpHandler,
	}
}

// collectRegisteredRoutes walks a mux.Router and returns a map of
// path template to HTTP methods (nil means all methods accepted).
func collectRegisteredRoutes(t *testing.T, router *mux.Router) map[string][]string {
	t.Helper()
	registeredRoutes := make(map[string][]string)
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil // skip routes without path templates
		}
		methods, _ := route.GetMethods()
		registeredRoutes[pathTemplate] = methods
		return nil
	})
	require.NoError(t, err)
	return registeredRoutes
}

func TestBuildHTTPRouter_AllRoutesRegistered(t *testing.T) {
	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(noOpHandler), "database", true)
	registeredRoutes := collectRegisteredRoutes(t, router)

	expectedRoutes := []struct {
		path            string
		expectedMethods []string
	}{
		{"/apis/v1beta1/pipelines/upload", nil},
		{"/apis/v1beta1/pipelines/upload_version", nil},
		{"/apis/v1beta1/healthz", nil},
		{"/apis/v2beta1/pipelines/upload", nil},
		{"/apis/v2beta1/pipelines/upload_version", nil},
		{"/apis/v2beta1/healthz", nil},
		{"/apis/v1alpha1/runs/{run_id}/nodes/{node_id}/log", nil},
		{"/apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read", []string{"GET"}},
		{"/apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read", []string{"GET"}},
		{"/metrics", nil},
	}

	for _, expectedRoute := range expectedRoutes {
		t.Run(expectedRoute.path, func(t *testing.T) {
			methods, exists := registeredRoutes[expectedRoute.path]
			assert.True(t, exists, "route %s should be registered", expectedRoute.path)
			if expectedRoute.expectedMethods != nil {
				assert.Equal(t, expectedRoute.expectedMethods, methods)
			}
		})
	}
}

func TestBuildHTTPRouter_MetricsConditional(t *testing.T) {
	routerWithMetrics := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(noOpHandler), "database", true)
	routesWithMetrics := collectRegisteredRoutes(t, routerWithMetrics)
	_, hasMetrics := routesWithMetrics["/metrics"]
	assert.True(t, hasMetrics, "/metrics should be registered when collectMetrics is true")

	routerWithoutMetrics := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(noOpHandler), "database", false)
	routesWithoutMetrics := collectRegisteredRoutes(t, routerWithoutMetrics)
	_, hasMetrics = routesWithoutMetrics["/metrics"]
	assert.False(t, hasMetrics, "/metrics should not be registered when collectMetrics is false")
}

func TestBuildHTTPRouter_HealthzResponses(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		pipelineStore string
		commitSHA     string
		tagName       string
		multiUser     bool
		wantV2Store   bool
	}{
		{
			name:      "v1beta1 healthz",
			path:      "/apis/v1beta1/healthz",
			commitSHA: `sha-"v1"`,
			tagName:   "tag-v1\nline",
			multiUser: true,
		},
		{
			name:          "v2beta1 healthz with database store",
			path:          "/apis/v2beta1/healthz",
			pipelineStore: "database",
			commitSHA:     `sha-"db"`,
			tagName:       `tag-"db"`,
			multiUser:     false,
			wantV2Store:   true,
		},
		{
			name:          "v2beta1 healthz with kubernetes store",
			path:          "/apis/v2beta1/healthz",
			pipelineStore: "kubernetes",
			commitSHA:     `sha-"k8s"`,
			tagName:       `tag-"k8s"`,
			multiUser:     true,
			wantV2Store:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("COMMIT_SHA", tt.commitSHA)
			viper.Set("TAG_NAME", tt.tagName)
			viper.Set(common.MultiUserMode, fmt.Sprintf("%t", tt.multiUser))

			pipelineStore := tt.pipelineStore
			if pipelineStore == "" {
				pipelineStore = "database"
			}
			router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(noOpHandler), pipelineStore, true)

			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
			var responseBody map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
			assert.Equal(t, tt.commitSHA, responseBody["commit_sha"])
			assert.Equal(t, tt.tagName, responseBody["tag_name"])
			assert.Equal(t, tt.multiUser, responseBody["multi_user"])
			pipelineStoreValue, hasPipelineStore := responseBody["pipeline_store"]
			assert.Equal(t, tt.wantV2Store, hasPipelineStore)
			if tt.wantV2Store {
				assert.Equal(t, tt.pipelineStore, pipelineStoreValue)
			}
		})
	}
}

func TestBuildHTTPRouter_HandlersAreCalled(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		setHandler func(deps *HTTPRouterDeps, handler http.HandlerFunc)
	}{
		{"v1beta1 upload pipeline", http.MethodPost, "/apis/v1beta1/pipelines/upload",
			func(deps *HTTPRouterDeps, handler http.HandlerFunc) { deps.UploadPipelineV1 = handler }},
		{"v2beta1 upload pipeline version", http.MethodPost, "/apis/v2beta1/pipelines/upload_version",
			func(deps *HTTPRouterDeps, handler http.HandlerFunc) { deps.UploadPipelineVersion = handler }},
		{"v1alpha1 run log", http.MethodGet, "/apis/v1alpha1/runs/run-123/nodes/node-456/log",
			func(deps *HTTPRouterDeps, handler http.HandlerFunc) { deps.ReadRunLogV1 = handler }},
		{"v2beta1 artifact read", http.MethodGet, "/apis/v2beta1/runs/run-123/nodes/node-456/artifacts/my-artifact:read",
			func(deps *HTTPRouterDeps, handler http.HandlerFunc) { deps.ReadArtifact = handler }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			instrumentedHandler := func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			}

			handlerDeps := newNoOpHTTPRouterDeps()
			tt.setHandler(&handlerDeps, instrumentedHandler)

			router := buildHTTPRouter(handlerDeps, http.HandlerFunc(noOpHandler), "database", true)

			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assert.True(t, handlerCalled, "handler for %s should have been called", tt.path)
		})
	}
}

func TestBuildHTTPRouter_UnmatchedAPIsGoToGateway(t *testing.T) {
	gatewayHandlerCalled := false
	gatewayHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), gatewayHandler, "database", true)

	request := httptest.NewRequest(http.MethodGet, "/apis/v2beta1/experiments", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.True(t, gatewayHandlerCalled, "requests to /apis/ paths not matching explicit routes should reach the gRPC gateway handler")
}
