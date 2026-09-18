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

package objectstore

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsOCIObjectStorageURIAndIsModelcarURI(t *testing.T) {
	tests := []struct {
		uri           string
		objectStorage bool
		modelcar      bool
	}{
		{uri: "oci://my-bucket@mynamespace/v2/artifacts", objectStorage: true},
		{uri: "oci://my-bucket@mynamespace", objectStorage: true},
		{uri: "oci://my-bucket@mynamespace/", objectStorage: true},
		{uri: "oci://my-bucket@mynamespace?region=us-ashburn-1", objectStorage: true},
		{uri: "oci://my_bucket.v2@mynamespace/run-1/model", objectStorage: true},
		{uri: "oci://registry.domain.local/my-model:latest", modelcar: true},
		{uri: "oci://quay.io/org/repo@sha256:0123abcd", modelcar: true},
		{uri: "oci://localhost:5000/repo:tag", modelcar: true},
		{uri: "oci://", modelcar: true},
		{uri: "s3://my-bucket@mynamespace/path"},
		{uri: "s3://my-bucket/path"},
		{uri: ""},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			assert.Equal(t, tt.objectStorage, IsOCIObjectStorageURI(tt.uri), "IsOCIObjectStorageURI")
			assert.Equal(t, tt.modelcar, IsModelcarURI(tt.uri), "IsModelcarURI")
		})
	}
}

func TestParseBucketPathToConfig_OCI(t *testing.T) {
	cfg, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/v2/artifacts")
	require.NoError(t, err)
	assert.Equal(t, &Config{
		Scheme:     OCIScheme,
		BucketName: "kfp-artifacts",
		Namespace:  "mynamespace",
		Prefix:     "v2/artifacts/",
	}, cfg)
	assert.Equal(t, "oci://kfp-artifacts@mynamespace/v2/artifacts", cfg.PrefixedBucket())
	assert.Equal(t, "oci://kfp-artifacts@mynamespace?prefix=v2/artifacts/", cfg.BucketURL())
	assert.Equal(t, "oci://kfp-artifacts@mynamespace/v2/artifacts", cfg.SessionInfoPath())

	provider, err := ParseProviderFromPath("oci://kfp-artifacts@mynamespace/v2/artifacts")
	require.NoError(t, err)
	assert.Equal(t, OCIProvider, provider)

	cfg, err = ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace")
	require.NoError(t, err)
	assert.Equal(t, "kfp-artifacts", cfg.BucketName)
	assert.Equal(t, "mynamespace", cfg.Namespace)
	assert.Equal(t, "", cfg.Prefix)
	assert.Equal(t, "oci://kfp-artifacts@mynamespace", cfg.PrefixedBucket())

	cfg, err = ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1")
	require.NoError(t, err)
	assert.Equal(t, "root/", cfg.Prefix)
	assert.Equal(t, "?region=us-ashburn-1", cfg.QueryString)
	assert.Equal(t, "oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1", cfg.SessionInfoPath())
	assert.Equal(t, "oci://kfp-artifacts@mynamespace?region=us-ashburn-1&prefix=root/", cfg.BucketURL())
}

func TestParseBucketPathToConfig_OCIRequiresNamespace(t *testing.T) {
	for _, uri := range []string{
		"oci://kfp-artifacts/v2/artifacts",
		"oci://registry.domain.local/my-model:latest",
		"oci://quay.io/org/repo@sha256:0123abcd",
		"oci://@mynamespace/v2/artifacts",
		"oci://kfp-artifacts@/v2/artifacts",
		"oci://kfp-artifacts@mynamespace@extra/v2/artifacts",
	} {
		t.Run(uri, func(t *testing.T) {
			_, err := ParseBucketPathToConfig(uri)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "oci://<bucket>@<namespace>/<prefix>")
		})
	}
}

func TestConfigHash_DistinguishesOCINamespaces(t *testing.T) {
	configA := &Config{Scheme: OCIScheme, BucketName: "kfp-artifacts", Namespace: "namespace-a", Prefix: "root/"}
	configB := &Config{Scheme: OCIScheme, BucketName: "kfp-artifacts", Namespace: "namespace-b", Prefix: "root/"}
	require.NotEqual(t, configA.Hash(), configB.Hash())
}

func TestIsWithinBucketRoot_OCINamespace(t *testing.T) {
	root, err := ParseBucketPathToConfig("oci://kfp-artifacts@namespace-a/v2/artifacts")
	require.NoError(t, err)

	sameNamespace, err := ParseBucketPathToConfig("oci://kfp-artifacts@namespace-a/v2/artifacts/run-1/model")
	require.NoError(t, err)
	assert.True(t, IsWithinBucketRoot(root, sameNamespace))

	otherNamespace, err := ParseBucketPathToConfig("oci://kfp-artifacts@namespace-b/v2/artifacts/run-1/model")
	require.NoError(t, err)
	assert.False(t, IsWithinBucketRoot(root, otherNamespace))
}

func TestSplitObjectURI_OCIKeepsBucketAndNamespace(t *testing.T) {
	prefix, base, err := SplitObjectURI("oci://kfp-artifacts@mynamespace/v2/artifacts/run-1/model")
	require.NoError(t, err)
	assert.Equal(t, "oci://kfp-artifacts@mynamespace/v2/artifacts/run-1", prefix)
	assert.Equal(t, "model", base)

	prefix, base, err = SplitObjectURI("oci://kfp-artifacts@mynamespace/model")
	require.NoError(t, err)
	assert.Equal(t, "oci://kfp-artifacts@mynamespace", prefix)
	assert.Equal(t, "model", base)

	// The prefix returned by SplitObjectURI must round-trip through the bucket parser.
	cfg, err := ParseBucketPathToConfig(prefix)
	require.NoError(t, err)
	assert.Equal(t, "kfp-artifacts", cfg.BucketName)
	assert.Equal(t, "mynamespace", cfg.Namespace)
}

func TestOCICompatEndpointAndRegionFromEndpoint(t *testing.T) {
	endpoint := OCICompatEndpoint("mynamespace", "us-ashburn-1")
	assert.Equal(t, "https://mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com", endpoint)

	tests := []struct {
		endpoint string
		region   string
	}{
		{endpoint: endpoint, region: "us-ashburn-1"},
		{endpoint: "mynamespace.compat.objectstorage.eu-frankfurt-1.oraclecloud.com", region: "eu-frankfurt-1"},
		{endpoint: "https://MyNamespace.compat.objectstorage.uk-london-1.oraclecloud.com:443/", region: "uk-london-1"},
		{endpoint: "https://objectstorage.us-ashburn-1.oraclecloud.com", region: ""},
		{endpoint: "https://proxy.internal.example:8443", region: ""},
		{endpoint: "s3.amazonaws.com", region: ""},
		{endpoint: "", region: ""},
	}
	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			assert.Equal(t, tt.region, OCIRegionFromEndpoint(tt.endpoint))
		})
	}
}

func TestResolveOCIEndpoint(t *testing.T) {
	t.Setenv(OCIRegionEnv, "")

	tests := []struct {
		name             string
		endpoint         string
		region           string
		envRegion        string
		expectedEndpoint string
		expectedRegion   string
		errorMsg         string
	}{
		{
			name:             "endpoint derived from namespace and region",
			region:           "us-ashburn-1",
			expectedEndpoint: "https://mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com",
			expectedRegion:   "us-ashburn-1",
		},
		{
			name:             "region inferred from a compat endpoint",
			endpoint:         "https://mynamespace.compat.objectstorage.eu-frankfurt-1.oraclecloud.com",
			expectedEndpoint: "https://mynamespace.compat.objectstorage.eu-frankfurt-1.oraclecloud.com",
			expectedRegion:   "eu-frankfurt-1",
		},
		{
			name:             "custom endpoint with explicit region",
			endpoint:         "https://objectstorage.internal.example:8443",
			region:           "us-phoenix-1",
			expectedEndpoint: "https://objectstorage.internal.example:8443",
			expectedRegion:   "us-phoenix-1",
		},
		{
			name:             "region falls back to OCI_REGION",
			envRegion:        "ca-toronto-1",
			expectedEndpoint: "https://mynamespace.compat.objectstorage.ca-toronto-1.oraclecloud.com",
			expectedRegion:   "ca-toronto-1",
		},
		{
			name:     "custom endpoint without region",
			endpoint: "https://objectstorage.internal.example:8443",
			errorMsg: "set the region explicitly",
		},
		{
			name:     "no endpoint and no region",
			errorMsg: OCIRegionEnv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(OCIRegionEnv, tt.envRegion)
			endpoint, region, err := resolveOCIEndpoint("mynamespace", tt.endpoint, tt.region)
			if tt.errorMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedEndpoint, endpoint)
			assert.Equal(t, tt.expectedRegion, region)
		})
	}
}

func TestOCIParamsFromQuery(t *testing.T) {
	t.Run("defaults to environment credentials and path style", func(t *testing.T) {
		cfg, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1")
		require.NoError(t, err)
		params, err := ociParamsFromQuery(cfg)
		require.NoError(t, err)
		assert.Equal(t, &S3Params{FromEnv: true, ForcePathStyle: true, Region: "us-ashburn-1"}, params)
	})

	t.Run("accepts endpoint and legacy path style spelling", func(t *testing.T) {
		cfg, err := ParseBucketPathToConfig(
			"oci://kfp-artifacts@mynamespace?endpoint=https://objectstorage.internal.example&region=us-phoenix-1&s3ForcePathStyle=false",
		)
		require.NoError(t, err)
		params, err := ociParamsFromQuery(cfg)
		require.NoError(t, err)
		assert.Equal(t, &S3Params{
			FromEnv:        true,
			ForcePathStyle: false,
			Region:         "us-phoenix-1",
			Endpoint:       "https://objectstorage.internal.example",
		}, params)
	})

	t.Run("rejects unsupported parameters", func(t *testing.T) {
		cfg, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1&disable_https=true")
		require.NoError(t, err)
		_, err = ociParamsFromQuery(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported query parameter "disable_https"`)
	})

	t.Run("rejects malformed booleans", func(t *testing.T) {
		cfg, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1&use_path_style=maybe")
		require.NoError(t, err)
		_, err = ociParamsFromQuery(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid value for "use_path_style"`)
	})
}

func TestOpenBucket_OCI(t *testing.T) {
	t.Setenv(OCIRegionEnv, "")
	const ns = "testnamespace"
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oci-customer-secret-key", Namespace: ns},
		Data: map[string][]byte{
			"accessKey": []byte("customer-secret-key-id"),
			"secretKey": []byte("customer-secret-key-value"),
		},
	}
	k8sClient := fake.NewSimpleClientset(secret)

	bucketConfig, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/v2/artifacts")
	require.NoError(t, err)

	tests := []struct {
		name              string
		sessionInfo       *SessionInfo
		envRegion         string
		expectedEndpoint  string
		expectedRegion    string
		expectedPathStyle bool
		expectedRetries   int
		errorMsg          string
	}{
		{
			name: "secret credentials with derived endpoint",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"forcePathStyle": "true",
					"maxRetries":     "5",
					"fromEnv":        "false",
					"secretName":     "oci-customer-secret-key",
					"accessKeyKey":   "accessKey",
					"secretKeyKey":   "secretKey",
				},
			},
			expectedEndpoint:  "https://mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com",
			expectedRegion:    "us-ashburn-1",
			expectedPathStyle: true,
			expectedRetries:   5,
		},
		{
			name: "explicit endpoint override",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params: map[string]string{
					"endpoint": "https://objectstorage.internal.example:8443",
					"region":   "us-phoenix-1",
					"fromEnv":  "true",
				},
			},
			expectedEndpoint:  "https://objectstorage.internal.example:8443",
			expectedRegion:    "us-phoenix-1",
			expectedPathStyle: true,
		},
		{
			name: "region inferred from compat endpoint",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params: map[string]string{
					"endpoint": "mynamespace.compat.objectstorage.eu-frankfurt-1.oraclecloud.com",
					"fromEnv":  "true",
				},
			},
			expectedEndpoint:  "https://mynamespace.compat.objectstorage.eu-frankfurt-1.oraclecloud.com",
			expectedRegion:    "eu-frankfurt-1",
			expectedPathStyle: true,
		},
		{
			name: "virtual host style when explicitly requested",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params: map[string]string{
					"region":         "us-ashburn-1",
					"forcePathStyle": "false",
					"fromEnv":        "true",
				},
			},
			expectedEndpoint:  "https://mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com",
			expectedRegion:    "us-ashburn-1",
			expectedPathStyle: false,
		},
		{
			name:              "nil session info uses environment credentials and OCI_REGION",
			sessionInfo:       nil,
			envRegion:         "uk-london-1",
			expectedEndpoint:  "https://mynamespace.compat.objectstorage.uk-london-1.oraclecloud.com",
			expectedRegion:    "uk-london-1",
			expectedPathStyle: true,
		},
		{
			name: "missing region",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params:   map[string]string{"fromEnv": "true"},
			},
			errorMsg: "requires a region",
		},
		{
			name: "missing secret",
			sessionInfo: &SessionInfo{
				Provider: OCIProvider,
				Params: map[string]string{
					"region":       "us-ashburn-1",
					"fromEnv":      "false",
					"secretName":   "does-not-exist",
					"accessKeyKey": "accessKey",
					"secretKeyKey": "secretKey",
				},
			},
			errorMsg: "secrets \"does-not-exist\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(OCIRegionEnv, tt.envRegion)
			bucket, err := OpenBucket(ctx, k8sClient, ns, bucketConfig, tt.sessionInfo)
			if tt.errorMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, bucket.Close())
			})

			var client *s3.Client
			require.True(t, bucket.As(&client))
			require.NotNil(t, client)
			require.NotNil(t, client.Options().BaseEndpoint)
			assert.Equal(t, tt.expectedEndpoint, *client.Options().BaseEndpoint)
			assert.Equal(t, tt.expectedRegion, client.Options().Region)
			assert.Equal(t, tt.expectedPathStyle, client.Options().UsePathStyle)
			assert.False(t, client.Options().EndpointOptions.DisableHTTPS)
			assert.Equal(t, aws.RequestChecksumCalculationWhenRequired, client.Options().RequestChecksumCalculation)
			assert.Equal(t, aws.ResponseChecksumValidationWhenRequired, client.Options().ResponseChecksumValidation)
			if tt.expectedRetries > 0 {
				assert.Equal(t, tt.expectedRetries, client.Options().Retryer.MaxAttempts())
			}
		})
	}
}

func TestOpenBucket_OCIQueryString(t *testing.T) {
	t.Setenv(OCIRegionEnv, "")
	envSession := &SessionInfo{Provider: OCIProvider, Params: map[string]string{"fromEnv": "true"}}

	t.Run("derives endpoint from region", func(t *testing.T) {
		bucketConfig, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1")
		require.NoError(t, err)
		bucket, err := OpenBucket(context.Background(), nil, "", bucketConfig, envSession)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, bucket.Close())
		})

		var client *s3.Client
		require.True(t, bucket.As(&client))
		assert.Equal(t, "us-ashburn-1", client.Options().Region)
		assert.True(t, client.Options().UsePathStyle)
		require.NotNil(t, client.Options().BaseEndpoint)
		assert.Equal(t, "https://mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com", *client.Options().BaseEndpoint)
	})

	t.Run("honours endpoint and path style parameters", func(t *testing.T) {
		bucketConfig, err := ParseBucketPathToConfig(
			"oci://kfp-artifacts@mynamespace/root?endpoint=https://objectstorage.internal.example:8443&region=us-phoenix-1&use_path_style=false",
		)
		require.NoError(t, err)
		bucket, err := OpenBucket(context.Background(), nil, "", bucketConfig, envSession)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, bucket.Close())
		})

		var client *s3.Client
		require.True(t, bucket.As(&client))
		assert.Equal(t, "us-phoenix-1", client.Options().Region)
		assert.False(t, client.Options().UsePathStyle)
		require.NotNil(t, client.Options().BaseEndpoint)
		assert.Equal(t, "https://objectstorage.internal.example:8443", *client.Options().BaseEndpoint)
	})

	t.Run("rejects unsupported parameters", func(t *testing.T) {
		bucketConfig, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1&profile=dev")
		require.NoError(t, err)
		_, err = OpenBucket(context.Background(), nil, "", bucketConfig, envSession)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported query parameter "profile"`)
	})
}

// captureHTTPClient records the last request and answers with an empty ListBucketResult.
type captureHTTPClient struct {
	request *http.Request
}

func (c *captureHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.request = req
	if req.Body != nil {
		// Drain the body like a real server would; trailing checksums are only
		// available once the request stream has been fully read.
		if _, err := io.Copy(io.Discard, req.Body); err != nil {
			return nil, err
		}
	}
	body := `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>kfp-artifacts</Name><KeyCount>0</KeyCount><IsTruncated>false</IsTruncated></ListBucketResult>`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/xml"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Request:    req,
	}, nil
}

func TestOpenBucket_OCIDoesNotSignSDKTelemetryHeaders(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "customer-secret-key-id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "customer-secret-key-value")
	ctx := context.Background()

	bucketConfig, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1")
	require.NoError(t, err)
	bucket, err := OpenBucket(ctx, nil, "", bucketConfig, &SessionInfo{Provider: OCIProvider, Params: map[string]string{"fromEnv": "true"}})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, bucket.Close())
	})

	var ociClient *s3.Client
	require.True(t, bucket.As(&ociClient))
	capture := &captureHTTPClient{}
	probe := s3.New(ociClient.Options(), func(o *s3.Options) {
		o.HTTPClient = capture
	})
	_, err = probe.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String("kfp-artifacts")})
	require.NoError(t, err)
	require.NotNil(t, capture.request)

	// OCI's S3-compatible API rejects signatures that cover these AWS SDK telemetry headers.
	assert.Empty(t, capture.request.Header.Get("Amz-Sdk-Invocation-Id"))
	assert.Empty(t, capture.request.Header.Get("Amz-Sdk-Request"))
	authorization := capture.request.Header.Get("Authorization")
	assert.Contains(t, authorization, "AWS4-HMAC-SHA256 Credential=customer-secret-key-id/")
	assert.Contains(t, authorization, "/us-ashburn-1/s3/aws4_request")
	assert.NotContains(t, authorization, "amz-sdk-invocation-id")
	assert.NotContains(t, authorization, "amz-sdk-request")
	assert.Equal(t, "mynamespace.compat.objectstorage.us-ashburn-1.oraclecloud.com", capture.request.URL.Host)
	assert.Equal(t, "/kfp-artifacts", capture.request.URL.Path)

	// The adjustment is scoped to OCI: a regular S3 client keeps the SDK headers.
	s3Client, err := newS3Client(ctx, &S3Params{FromEnv: true, Region: "us-east-1", Endpoint: "s3.amazonaws.com", ForcePathStyle: true}, nil)
	require.NoError(t, err)
	controlCapture := &captureHTTPClient{}
	control := s3.New(s3Client.Options(), func(o *s3.Options) {
		o.HTTPClient = controlCapture
	})
	_, err = control.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String("kfp-artifacts")})
	require.NoError(t, err)
	assert.NotEmpty(t, controlCapture.request.Header.Get("Amz-Sdk-Invocation-Id"))
	assert.NotEmpty(t, controlCapture.request.Header.Get("Amz-Sdk-Request"))
}

func TestOpenBucket_OCIDisablesChecksumTrailers(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "customer-secret-key-id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "customer-secret-key-value")
	ctx := context.Background()

	bucketConfig, err := ParseBucketPathToConfig("oci://kfp-artifacts@mynamespace/root?region=us-ashburn-1")
	require.NoError(t, err)
	bucket, err := OpenBucket(ctx, nil, "", bucketConfig, &SessionInfo{Provider: OCIProvider, Params: map[string]string{"fromEnv": "true"}})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, bucket.Close())
	})

	var ociClient *s3.Client
	require.True(t, bucket.As(&ociClient))
	capture := &captureHTTPClient{}
	probe := s3.New(ociClient.Options(), func(o *s3.Options) {
		o.HTTPClient = capture
	})
	// gocloud's transfer manager always asks for a CRC32 checksum; mimic that.
	_, err = probe.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String("kfp-artifacts"),
		Key:               aws.String("root/model"),
		Body:              strings.NewReader("model bytes"),
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc32,
	})
	require.NoError(t, err)
	require.NotNil(t, capture.request)

	// OCI does not implement aws-chunked encoding, so no trailing checksum may be used.
	assert.Equal(t, "UNSIGNED-PAYLOAD", capture.request.Header.Get("X-Amz-Content-Sha256"))
	assert.Empty(t, capture.request.Header.Get("X-Amz-Trailer"))
	assert.Empty(t, capture.request.Header.Get("X-Amz-Sdk-Checksum-Algorithm"))
	assert.NotContains(t, capture.request.Header.Get("Content-Encoding"), "aws-chunked")
	assert.Empty(t, capture.request.Header.Get("X-Amz-Decoded-Content-Length"))

	// The adjustment is scoped to OCI: a regular S3 client keeps the SDK behaviour.
	s3Client, err := newS3Client(ctx, &S3Params{FromEnv: true, Region: "us-east-1", Endpoint: "s3.amazonaws.com", ForcePathStyle: true}, nil)
	require.NoError(t, err)
	controlCapture := &captureHTTPClient{}
	control := s3.New(s3Client.Options(), func(o *s3.Options) {
		o.HTTPClient = controlCapture
	})
	_, err = control.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String("kfp-artifacts"),
		Key:               aws.String("root/model"),
		Body:              strings.NewReader("model bytes"),
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc32,
	})
	require.NoError(t, err)
	assert.Equal(t, "STREAMING-UNSIGNED-PAYLOAD-TRAILER", controlCapture.request.Header.Get("X-Amz-Content-Sha256"))
	assert.Equal(t, "x-amz-checksum-crc32", controlCapture.request.Header.Get("X-Amz-Trailer"))
}
