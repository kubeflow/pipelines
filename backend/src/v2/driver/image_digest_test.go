// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/cachekey"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseImageReference(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		want    parsedImageReference
		wantErr bool
	}{
		{
			name:  "short docker hub image with tag",
			image: "ubuntu:22.04",
			want: parsedImageReference{
				registry:   "docker.io",
				repository: "library/ubuntu",
				tag:        "22.04",
			},
		},
		{
			name:  "docker hub org image",
			image: "bitnami/kubectl:latest",
			want: parsedImageReference{
				registry:   "docker.io",
				repository: "bitnami/kubectl",
				tag:        "latest",
			},
		},
		{
			name:  "gcr image with tag",
			image: "gcr.io/my-project/app:v1",
			want: parsedImageReference{
				registry:   "gcr.io",
				repository: "my-project/app",
				tag:        "v1",
			},
		},
		{
			name:  "already digest-pinned",
			image: "gcr.io/my-project/app@sha256:abc",
			want: parsedImageReference{
				registry:   "gcr.io",
				repository: "my-project/app",
				tag:        "latest",
				digest:     "sha256:abc",
			},
		},
		{
			name:  "tag and digest",
			image: "gcr.io/my-project/app:v1@sha256:abc",
			want: parsedImageReference{
				registry:   "gcr.io",
				repository: "my-project/app",
				tag:        "v1",
				digest:     "sha256:abc",
			},
		},
		{
			name:  "localhost registry with port",
			image: "localhost:5000/foo:bar",
			want: parsedImageReference{
				registry:   "localhost:5000",
				repository: "foo",
				tag:        "bar",
			},
		},
		{
			name:    "empty image",
			image:   "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseImageReference(test.image)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_resolveImageForCache(t *testing.T) {
	t.Run("disabled returns original image", func(t *testing.T) {
		opts := Options{CacheResolveImageDigest: false}
		assert.Equal(t, "my-image:latest", resolveImageForCache(opts, "my-image:latest"))
	})

	t.Run("dummy image is not resolved", func(t *testing.T) {
		called := false
		opts := Options{
			CacheResolveImageDigest: true,
			ImageDigestResolver: func(image string) (string, error) {
				called = true
				return "should-not-be-used", nil
			},
		}
		assert.Equal(t, "argostub/createpvc", resolveImageForCache(opts, "argostub/createpvc"))
		assert.False(t, called)
	})

	t.Run("resolver success uses digest", func(t *testing.T) {
		opts := Options{
			CacheResolveImageDigest: true,
			ImageDigestResolver: func(image string) (string, error) {
				assert.Equal(t, "my-image:latest", image)
				return "docker.io/library/my-image@sha256:deadbeef", nil
			},
		}
		assert.Equal(t, "docker.io/library/my-image@sha256:deadbeef", resolveImageForCache(opts, "my-image:latest"))
	})

	t.Run("resolver failure falls back to original", func(t *testing.T) {
		opts := Options{
			CacheResolveImageDigest: true,
			ImageDigestResolver: func(image string) (string, error) {
				return "", fmt.Errorf("registry unavailable")
			},
		}
		assert.Equal(t, "my-image:latest", resolveImageForCache(opts, "my-image:latest"))
	})
}

func Test_parseBearerChallenge(t *testing.T) {
	challenge, ok := parseBearerChallenge(`Bearer realm="https://auth.example.com/token",service="registry.example.com",scope="repository:foo/bar:pull"`)
	require.True(t, ok)
	assert.Equal(t, "https://auth.example.com/token", challenge.realm)
	assert.Equal(t, "registry.example.com", challenge.service)
	assert.Equal(t, "repository:foo/bar:pull", challenge.scope)
}

func Test_dockerConfigCredentials(t *testing.T) {
	auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	cfg := &dockerConfigFile{Auths: map[string]dockerConfigAuth{
		"registry.example.com": {Auth: auth},
		"https://index.docker.io/v1/": {
			Username: "hubuser",
			Password: "hubpass",
		},
	}}

	username, password, ok := cfg.credentialsFor("registry.example.com", false)
	require.True(t, ok)
	assert.Equal(t, "user", username)
	assert.Equal(t, "pass", password)

	username, password, ok = cfg.credentialsFor(defaultDockerHubHost, false)
	require.True(t, ok)
	assert.Equal(t, "hubuser", username)
	assert.Equal(t, "hubpass", password)

	// Mirror host without its own auth entry should still find Docker Hub keys
	// when includeDockerHubKeys is true.
	username, password, ok = cfg.credentialsFor("mirror.example.com", true)
	require.True(t, ok)
	assert.Equal(t, "hubuser", username)
	assert.Equal(t, "hubpass", password)

	// Without the Docker Hub fallback, a random mirror host must not pick up Hub creds.
	_, _, ok = cfg.credentialsFor("mirror.example.com", false)
	assert.False(t, ok)
}

func Test_resolveImageDigestHTTP_privateRegistryWithToken(t *testing.T) {
	var sawBasicAuth bool
	var sawBearer bool

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/my-project/app/manifests/v1", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="`+schemeHost(r)+`/token",service="registry.example.com",scope="repository:my-project/app:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasPrefix(auth, "Bearer ") {
			sawBearer = true
			w.Header().Set("Docker-Content-Digest", "sha256:privatedigest")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", username)
		assert.Equal(t, "pass", password)
		sawBasicAuth = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"registry-token"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	host := server.Listener.Addr().String()
	configPath := writeDockerConfig(t, map[string]dockerConfigAuth{
		host: {Auth: base64.StdEncoding.EncodeToString([]byte("user:pass"))},
	})

	got, err := resolveImageDigestHTTP(
		host+"/my-project/app:v1",
		server.Client(),
		ImageDigestResolveConfig{
			DockerConfigPath:   configPath,
			InsecureRegistries: host,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, host+"/my-project/app@sha256:privatedigest", got)
	assert.True(t, sawBasicAuth)
	assert.True(t, sawBearer)
}

func Test_resolveImageDigestHTTP_configurableDockerHub(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/library/ubuntu/manifests/22.04", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="`+schemeHost(r)+`/auth/token",service="mirror.service",scope="repository:library/ubuntu:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:mirror")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "mirror.service", r.URL.Query().Get("service"))
		_, _ = w.Write([]byte(`{"token":"mirror-token"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	host := server.Listener.Addr().String()

	got, err := resolveImageDigestHTTP(
		"ubuntu:22.04",
		server.Client(),
		ImageDigestResolveConfig{
			DockerHubRegistryHost: host,
			InsecureRegistries:    host,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "docker.io/library/ubuntu@sha256:mirror", got)
}

func Test_resolveImageDigestHTTP_dockerHubMirrorUsesIndexCredentials(t *testing.T) {
	var sawHubUser bool

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/library/ubuntu/manifests/22.04", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("Www-Authenticate", `Bearer realm="`+schemeHost(r)+`/auth/token",service="mirror.service",scope="repository:library/ubuntu:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:mirror-auth")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "hubuser", username)
		assert.Equal(t, "hubpass", password)
		sawHubUser = true
		_, _ = w.Write([]byte(`{"token":"mirror-token"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	host := server.Listener.Addr().String()

	configPath := writeDockerConfig(t, map[string]dockerConfigAuth{
		"https://index.docker.io/v1/": {
			Username: "hubuser",
			Password: "hubpass",
		},
	})

	got, err := resolveImageDigestHTTP(
		"ubuntu:22.04",
		server.Client(),
		ImageDigestResolveConfig{
			DockerConfigPath:      configPath,
			DockerHubRegistryHost: host,
			InsecureRegistries:    host,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "docker.io/library/ubuntu@sha256:mirror-auth", got)
	assert.True(t, sawHubUser)
}

func Test_newImageDigestHTTPClient_withCACert(t *testing.T) {
	client, err := newImageDigestHTTPClient("")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Nil(t, client.Transport)

	_, err = newImageDigestHTTPClient(filepath.Join(t.TempDir(), "missing.pem"))
	require.Error(t, err)

	certFile := writeTestCACert(t)
	client, err = newImageDigestHTTPClient(certFile)
	require.NoError(t, err)
	require.NotNil(t, client)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	require.NotNil(t, transport.TLSClientConfig.RootCAs)
	// Cloning DefaultTransport must keep proxy support for corporate / CI proxies.
	require.NotNil(t, transport.Proxy)
}

func Test_resolveImageDigestHTTP_headMissingDigestFallsBackToGET(t *testing.T) {
	var headCalls, getCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/my-project/app/manifests/v1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			headCalls++
			// Some registries omit Docker-Content-Digest on HEAD.
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			getCalls++
			w.Header().Set("Docker-Content-Digest", "sha256:fromget")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	host := server.Listener.Addr().String()

	got, err := resolveImageDigestHTTP(
		host+"/my-project/app:v1",
		server.Client(),
		ImageDigestResolveConfig{InsecureRegistries: host},
	)
	require.NoError(t, err)
	assert.Equal(t, host+"/my-project/app@sha256:fromget", got)
	assert.Equal(t, 1, headCalls)
	assert.Equal(t, 1, getCalls)
}

func Test_resolveImageDigestHTTP_headMethodNotAllowedFallsBackToGET(t *testing.T) {
	var getCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/my-project/app/manifests/v1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		getCalls++
		w.Header().Set("Docker-Content-Digest", "sha256:getonly")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	host := server.Listener.Addr().String()

	got, err := resolveImageDigestHTTP(
		host+"/my-project/app:v1",
		server.Client(),
		ImageDigestResolveConfig{InsecureRegistries: host},
	)
	require.NoError(t, err)
	assert.Equal(t, host+"/my-project/app@sha256:getonly", got)
	assert.Equal(t, 1, getCalls)
}

func Test_resolveImageDigestHTTP_alreadyPinned(t *testing.T) {
	got, err := resolveImageDigestHTTP("gcr.io/proj/app@sha256:abc123", http.DefaultClient, ImageDigestResolveConfig{})
	require.NoError(t, err)
	assert.Equal(t, "gcr.io/proj/app@sha256:abc123", got)
}

func Test_getFingerPrint_resolvesImageWhenEnabled(t *testing.T) {
	opts := Options{
		CacheResolveImageDigest: true,
		ImageDigestResolver: func(image string) (string, error) {
			return "docker.io/library/test-image@sha256:resolved", nil
		},
		Component: &pipelinespec.ComponentSpec{},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image: "test-image:latest",
		},
	}
	var gotImage string
	mockClient := &mockCacheClient{
		generateCacheKeyFunc: func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error) {
			gotImage = image
			return &cachekey.CacheKey{}, nil
		},
		generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
			return "fp", nil
		},
	}

	fingerPrint, err := getFingerPrint(opts, &pipelinespec.ExecutorInput{}, mockClient, nil)
	require.NoError(t, err)
	assert.Equal(t, "fp", fingerPrint)
	assert.Equal(t, "docker.io/library/test-image@sha256:resolved", gotImage)
}

func Test_getFingerPrint_keepsImageWhenDisabled(t *testing.T) {
	opts := Options{
		CacheResolveImageDigest: false,
		ImageDigestResolver: func(image string) (string, error) {
			t.Fatal("resolver should not be called when feature is disabled")
			return "", nil
		},
		Component: &pipelinespec.ComponentSpec{},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image: "test-image:latest",
		},
	}
	var gotImage string
	mockClient := &mockCacheClient{
		generateCacheKeyFunc: func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error) {
			gotImage = image
			return &cachekey.CacheKey{}, nil
		},
		generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
			return "fp", nil
		},
	}

	_, err := getFingerPrint(opts, &pipelinespec.ExecutorInput{}, mockClient, nil)
	require.NoError(t, err)
	assert.Equal(t, "test-image:latest", gotImage)
}

func writeDockerConfig(t *testing.T, auths map[string]dockerConfigAuth) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	payload, err := json.Marshal(dockerConfigFile{Auths: auths})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, payload, 0o600))
	return path
}

func writeTestCACert(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"KFP Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}), 0o600))
	return certFile
}

func schemeHost(r *http.Request) string {
	return "http://" + r.Host
}
