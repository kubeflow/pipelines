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
	for _, mode := range []string{"structured", "query"} {
		t.Run(mode, func(t *testing.T) {
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
			if mode == "query" {
				cfg, err = ParseBucketPathToConfig("s3://bucket/prefix?endpoint=http%3A%2F%2Fstorage.invalid&region=us-east-1&s3ForcePathStyle=true&disable_https=true&request_checksum_calculation=when_required")
				require.NoError(t, err)
				session = nil
			}
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
		for _, mode := range []string{"structured", "query"} {
			for _, proxied := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/proxy=%t", caSource, mode, proxied), func(t *testing.T) {
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
								http.Error(w, err.Error(), 502)
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
					if mode == "query" {
						root := "s3://bucket/prefix?endpoint=" + url.QueryEscape(origin.URL) + "&region=us-east-1&s3ForcePathStyle=true&request_checksum_calculation=when_required"
						if caSource == "profile" {
							root += "&profile=custom-ca"
						}
						cfg, err = ParseBucketPathToConfig(root)
						require.NoError(t, err)
						session = nil
					}
					bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, session)
					require.NoError(t, err)
					defer bucket.Close()
					require.NoError(t, bucket.WriteAll(ctx, "log", []byte("logs"), nil))
					require.EqualValues(t, 1, uploads.Load())
					require.Equal(t, proxied, tunnels.Load() > 0)
				})
			}
		}
	}
}

func TestGCSEmulatorDoesNotRequireADC(t *testing.T) {
	var requests atomic.Int32
	emulator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("emulator must not receive credentials")
		}
		fmt.Fprint(w, "emulated logs")
	}))
	defer emulator.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", strings.TrimPrefix(emulator.URL, "http://"))
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-credentials"))
	ctx := util.WithHTTPProxy(context.Background(), nil)
	cfg, err := ParseBucketPathToConfig("gs://bucket")
	require.NoError(t, err)
	bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, &SessionInfo{Provider: "gs", Params: map[string]string{"fromEnv": "true"}})
	require.NoError(t, err)
	defer bucket.Close()
	data, err := bucket.ReadAll(ctx, "log")
	require.NoError(t, err)
	require.Equal(t, "emulated logs", string(data))
	require.EqualValues(t, 1, requests.Load())
}

func TestS3QueryAssumeRoleUsesInvocationProxy(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "base-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "base-secret")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ENDPOINT_URL_STS", "http://sts.invalid")
	var tokens, uploads atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Host == "sts.invalid" {
			tokens.Add(1)
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "Action=AssumeRole") {
				t.Errorf("unexpected STS body %q", body)
			}
			fmt.Fprintf(w, `<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><AssumeRoleResult><Credentials><AccessKeyId>assumed-access</AccessKeyId><SecretAccessKey>assumed-secret</SecretAccessKey><SessionToken>assumed-token</SessionToken><Expiration>%s</Expiration></Credentials><AssumedRoleUser><Arn>arn:aws:sts::123456789012:assumed-role/test/session</Arn><AssumedRoleId>test:session</AssumedRoleId></AssumedRoleUser></AssumeRoleResult></AssumeRoleResponse>`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
			return
		}
		uploads.Add(1)
		if r.URL.Host != "storage.invalid" || !strings.Contains(r.Header.Get("Authorization"), "assumed-access") {
			t.Errorf("unexpected S3 request: host=%s auth=%s", r.URL.Host, r.Header.Get("Authorization"))
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("ETag", `"test-etag"`)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL)), 5*time.Second)
	defer cancel()
	cfg, err := ParseBucketPathToConfig("s3://bucket/prefix?endpoint=http%3A%2F%2Fstorage.invalid&region=us-east-1&s3ForcePathStyle=true&disable_https=true&request_checksum_calculation=when_required&role=arn:aws:iam::123456789012:role/test")
	require.NoError(t, err)
	bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, nil)
	require.NoError(t, err)
	defer bucket.Close()
	require.NoError(t, bucket.WriteAll(ctx, "log", []byte("driver logs"), nil))
	require.EqualValues(t, 1, tokens.Load())
	require.EqualValues(t, 1, uploads.Load())
}

func TestGCSURLOpenerPreservesUniverseDomain(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	for _, source := range []string{"query", "credentials", "anonymous"} {
		t.Run(source, func(t *testing.T) {
			const domain = "alternate.example"
			credentials := map[string]string{"type": "service_account", "client_email": "test@example.invalid", "private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), "token_uri": "http://tokens.invalid/token"}
			root := "gs://bucket"
			if source == "credentials" {
				credentials["universe_domain"] = domain
			} else {
				root += "?universe_domain=" + domain
			}
			if source == "anonymous" {
				root += "&anonymous=true"
			}
			encoded, err := json.Marshal(credentials)
			require.NoError(t, err)
			file := filepath.Join(t.TempDir(), "credentials.json")
			require.NoError(t, os.WriteFile(file, encoded, 0600))
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", file)
			t.Setenv("GOOGLE_CLOUD_UNIVERSE_DOMAIN", "")
			var storageHost atomic.Value
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodConnect {
					storageHost.Store(r.Host)
					http.Error(w, "intentional stop before external TLS connection", http.StatusBadGateway)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
			}))
			defer proxy.Close()
			proxyURL, err := url.Parse(proxy.URL)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(util.WithHTTPProxy(context.Background(), http.ProxyURL(proxyURL)), time.Second)
			defer cancel()
			cfg, err := ParseBucketPathToConfig(root)
			require.NoError(t, err)
			bucket, err := OpenBucket(ctx, fake.NewClientset(), "ns", cfg, nil)
			require.NoError(t, err)
			defer bucket.Close()
			_, err = bucket.ReadAll(ctx, "log")
			require.Error(t, err, "proxy deliberately rejects CONNECT rather than contacting external storage")
			require.Equal(t, "storage."+domain+":443", storageHost.Load())
		})
	}
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
