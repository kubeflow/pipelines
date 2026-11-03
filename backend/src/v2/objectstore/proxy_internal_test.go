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

package objectstore

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestS3UploadsUseInvocationProxy(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("HTTP_PROXY", "http://unused-process-proxy.invalid")
	var requests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Host != "storage.invalid" {
			t.Errorf("unexpected host %s", r.URL.Host)
		}
		if r.URL.Path != "/bucket/prefix/log" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "driver logs" {
			t.Errorf("unexpected body %q", body)
		}
		w.Header().Set("ETag", `"test-etag"`)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx := util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL))
	cfg, err := ParseBucketPathToConfig("s3://bucket/prefix")
	require.NoError(t, err)
	session := &SessionInfo{Provider: "s3", Params: map[string]string{
		S3ParamFromEnv: "true", S3ParamEndpoint: "storage.invalid", S3ParamDisableSSL: "true", S3ParamForcePathStyle: "true", S3ParamRegion: "us-east-1",
	}}
	for _, scheme := range []string{"structured", "s3", "minio"} {
		t.Run(scheme, func(t *testing.T) {
			if scheme != "structured" {
				cfg, err = ParseBucketPathToConfig(scheme + "://bucket/prefix?endpoint=http://storage.invalid&region=us-east-1&s3ForcePathStyle=true&request_checksum_calculation=when_required")
				require.NoError(t, err)
			}
			requests.Store(0)
			bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, session)
			require.NoError(t, err)
			defer bucket.Close()
			path := filepath.Join(t.TempDir(), "log")
			require.NoError(t, os.WriteFile(path, []byte("driver logs"), 0600))
			require.NoError(t, UploadBlob(ctx, bucket, path, "log"))
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

func TestS3InvocationProxyPreservesCustomCA(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, caSource := range []string{"environment", "profile"} {
		for _, proxied := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/proxy=%t", caSource, proxied), func(t *testing.T) {
				var uploads, tunnels atomic.Int32
				origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					uploads.Add(1)
					_, _ = io.Copy(io.Discard, r.Body)
					w.Header().Set("ETag", `"test-etag"`)
				}))
				t.Cleanup(origin.Close)
				ca := filepath.Join(t.TempDir(), "ca.pem")
				require.NoError(t, os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: origin.Certificate().Raw}), 0600))
				if caSource == "environment" {
					t.Setenv("AWS_CA_BUNDLE", ca)
				} else {
					config := filepath.Join(t.TempDir(), "config")
					require.NoError(t, os.WriteFile(config, []byte("[profile custom-ca]\nregion=us-east-1\naws_access_key_id=test-access\naws_secret_access_key=test-secret\nca_bundle="+ca+"\n"), 0600))
					t.Setenv("AWS_CONFIG_FILE", config)
					t.Setenv("AWS_PROFILE", "custom-ca")
					t.Setenv("AWS_CA_BUNDLE", "")
				}
				ctx := util.WithHTTPProxy(context.Background(), nil)
				if proxied {
					proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodConnect {
							http.Error(w, "CONNECT required", 400)
							return
						}
						upstream, err := net.Dial("tcp", r.Host)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadGateway)
							return
						}
						client, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							upstream.Close()
							return
						}
						t.Cleanup(func() { client.Close(); upstream.Close() })
						tunnels.Add(1)
						fmt.Fprint(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
						go func() { _, _ = io.Copy(upstream, client) }()
						_, _ = io.Copy(client, upstream)
					}))
					t.Cleanup(proxy.Close)
					proxyURL, err := url.Parse(proxy.URL)
					require.NoError(t, err)
					ctx = util.WithHTTPProxy(ctx, http.ProxyURL(proxyURL))
				}
				cfg, err := ParseBucketPathToConfig("s3://bucket/prefix")
				require.NoError(t, err)
				session := &SessionInfo{Provider: "s3", Params: map[string]string{"fromEnv": "true", "endpoint": origin.URL, "region": "us-east-1", "forcePathStyle": "true"}}
				for _, configuredByURL := range []bool{false, true} {
					t.Run(fmt.Sprintf("url=%t", configuredByURL), func(t *testing.T) {
						if configuredByURL {
							cfg, err = ParseBucketPathToConfig("s3://bucket/prefix?endpoint=" + url.QueryEscape(origin.URL) + "&region=us-east-1&use_path_style=true")
							require.NoError(t, err)
						}
						uploads.Store(0)
						bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, session)
						require.NoError(t, err)
						defer bucket.Close()
						require.NoError(t, bucket.WriteAll(ctx, "log", []byte("logs"), nil))
						require.EqualValues(t, 1, uploads.Load())
						require.Equal(t, proxied, tunnels.Load() > 0)
					})
				}
			})
		}
	}
}

func TestS3URLInvocationOptions(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// The URL's endpoint= applies to storage only; STS keeps its own endpoint.
	t.Setenv("AWS_ENDPOINT_URL_STS", "http://sts.invalid")
	var assumedRoles atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPost && strings.Contains(string(body), "Action=AssumeRole") {
			assumedRoles.Add(1)
			if r.URL.Host != "sts.invalid" {
				t.Errorf("AssumeRole sent to %s, want the STS endpoint", r.URL.Host)
				http.Error(w, "InvalidAction", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprint(w, `<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><AssumeRoleResult><Credentials><AccessKeyId>assumed-key</AccessKeyId><SecretAccessKey>assumed-secret</SecretAccessKey><SessionToken>assumed-token</SessionToken><Expiration>2099-01-01T00:00:00Z</Expiration></Credentials></AssumeRoleResult></AssumeRoleResponse>`)
			return
		}
		if r.URL.Host != "storage.invalid" {
			t.Errorf("object request sent to %s, want the storage endpoint", r.URL.Host)
		}
		if r.URL.Path != "/bucket/prefix/log" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Amz-Server-Side-Encryption") != "aws:kms" || r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id") != "test-key" {
			t.Error("URL encryption options were lost")
		}
		if auth := r.Header.Get("Authorization"); auth != "" && !strings.Contains(auth, "assumed-key") {
			t.Errorf("expected assumed-role credentials, got %s", auth)
		}
		w.Header().Set("ETag", `"test-etag"`)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx := util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL))
	for _, credentials := range []string{"anonymous=true", "role=arn:aws:iam::123456789012:role/test"} {
		config, err := ParseBucketPathToConfig("s3://bucket/prefix?region=us-east-1&endpoint=http://storage.invalid&use_path_style=true&ssetype=aws:kms&kmskeyid=test-key&" + credentials)
		require.NoError(t, err)
		bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", config, nil)
		require.NoError(t, err)
		require.NoError(t, bucket.WriteAll(ctx, "log", []byte("logs"), nil))
		require.NoError(t, bucket.Close())
	}
	require.EqualValues(t, 1, assumedRoles.Load())
	for _, query := range []string{"unknown=true", "use_path_style=invalid", "ssetype=invalid", "role=invalid"} {
		config, err := ParseBucketPathToConfig("s3://bucket/prefix?" + query)
		require.NoError(t, err)
		_, err = OpenBucket(ctx, fake.NewClientset(), "ns", config, nil)
		require.Error(t, err, query)
	}
}

func TestURLConfiguredStoresIsolateInvocationProxies(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, provider := range []string{"s3", "gs", "gs-anonymous"} {
		t.Run(provider, func(t *testing.T) {
			var directRequests atomic.Int32
			respond := func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("ETag", `"test-etag"`)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"bucket":"bucket","name":"prefix/log","size":"4"}`)
			}
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				directRequests.Add(1)
				respond(w, r)
			}))
			defer origin.Close()
			var proxyRequests [2]atomic.Int32
			contexts := []context.Context{}
			for i := range proxyRequests {
				proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					proxyRequests[i].Add(1)
					respond(w, r)
				}))
				defer proxy.Close()
				proxyURL, err := url.Parse(proxy.URL)
				require.NoError(t, err)
				contexts = append(contexts, util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL)))
			}
			contexts = append(contexts, util.WithHTTPProxy(context.Background(), nil))
			root := "s3://bucket/prefix?region=us-east-1&use_path_style=true&endpoint=" + url.QueryEscape(origin.URL)
			var session *SessionInfo
			if provider != "s3" {
				t.Setenv("STORAGE_EMULATOR_HOST", origin.Listener.Addr().String())
				root = "gs://bucket/prefix"
				if provider == "gs-anonymous" {
					root += "?anonymous=true"
				}
				session = &SessionInfo{Provider: "gs", Params: map[string]string{"fromEnv": "true"}}
			}
			config, err := ParseBucketPathToConfig(root)
			require.NoError(t, err)
			// Open all clients before using any to catch shared-client mutation.
			buckets := make([]*blob.Bucket, len(contexts))
			for i, ctx := range contexts {
				buckets[i], err = OpenBucket(ctx, fake.NewClientset(), "ns", config, session)
				require.NoError(t, err)
				defer buckets[i].Close()
			}
			for i, ctx := range contexts {
				require.NoError(t, buckets[i].WriteAll(ctx, "log", []byte("logs"), nil))
			}
			require.EqualValues(t, 1, directRequests.Load())
			for i := range proxyRequests {
				require.EqualValues(t, 1, proxyRequests[i].Load())
			}
		})
	}
}

func TestGCSFromEnvCredentialsUseInvocationProxy(t *testing.T) {
	t.Setenv("STORAGE_EMULATOR_HOST", "")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "test@example.invalid",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})),
		"token_uri":   "http://tokens.invalid/token",
	})
	require.NoError(t, err)
	credentials := filepath.Join(t.TempDir(), "credentials.json")
	require.NoError(t, os.WriteFile(credentials, raw, 0600))
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentials)
	var tokens, tunnels atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "tokens.invalid" {
			tokens.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		if r.Method == http.MethodConnect && r.Host == "storage.googleapis.com:443" {
			tunnels.Add(1)
		} else {
			t.Errorf("unexpected proxy request: %s %s", r.Method, r.Host)
		}
		// Stop at the proxy rather than contacting the real GCS service.
		http.Error(w, "test proxy stops TLS tunnel", http.StatusForbidden)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL)), time.Second)
	defer cancel()
	config, err := ParseBucketPathToConfig("gs://bucket/prefix")
	require.NoError(t, err)
	bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", config, &SessionInfo{Provider: "gs", Params: map[string]string{"fromEnv": "true"}})
	require.NoError(t, err)
	defer bucket.Close()
	require.Error(t, bucket.WriteAll(ctx, "log", []byte("logs"), nil))
	require.EqualValues(t, 1, tokens.Load())
	require.Positive(t, tunnels.Load())
}

func TestGCSCredentialsAndDataUseInvocationProxy(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "test@example.invalid",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})),
		"token_uri":   "http://tokens.invalid/token",
	})
	require.NoError(t, err)
	var tokens, data atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Host == "tokens.invalid" {
			tokens.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
		} else {
			data.Add(1)
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("missing token")
			}
			fmt.Fprint(w, "data")
		}
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx := util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL))
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "gcs-creds", Namespace: "ns"}, Data: map[string][]byte{"token": raw}}
	client, err := getGCSTokenClient(ctx, "ns", &SessionInfo{Provider: "gs", Params: map[string]string{"secretName": "gcs-creds", "tokenKey": "token"}}, fake.NewClientset(secret))
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://storage.invalid/object", nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.EqualValues(t, 1, tokens.Load())
	require.EqualValues(t, 1, data.Load())
	require.Nil(t, invocationTransport(context.Background()))
	direct := invocationTransport(util.WithHTTPProxy(context.Background(), nil))
	require.NotNil(t, direct)
	require.Nil(t, direct.Proxy)
	require.NotNil(t, http.DefaultTransport.(*http.Transport).Proxy)
}
