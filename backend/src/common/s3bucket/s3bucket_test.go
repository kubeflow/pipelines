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

package s3bucket

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/s3blob"
)

func s3ClientWithChecksumPolicy(policy aws.RequestChecksumCalculation) *s3.Client {
	return s3.New(s3.Options{Region: "us-east-1", RequestChecksumCalculation: policy})
}

// newNoChunkedEncodingS3Server starts an HTTPS server that behaves like an
// S3-compatible store without aws-chunked support, such as OCI Object Storage.
func newNoChunkedEncodingS3Server(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") ||
			strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NotImplemented</Code><Message>AWS chunked encoding not supported.</Message></Error>`)
			return
		}
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestBucketOptions_PropagatesClientChecksumPolicy(t *testing.T) {
	for _, test := range []struct {
		name   string
		policy aws.RequestChecksumCalculation
	}{
		{name: "when_required", policy: aws.RequestChecksumCalculationWhenRequired},
		{name: "when_supported", policy: aws.RequestChecksumCalculationWhenSupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := bucketOptions(s3ClientWithChecksumPolicy(test.policy), nil)

			assert.Equal(t, test.policy, options.RequestChecksumCalculation)
		})
	}
}

func TestBucketOptions_PreservesCallerOptions(t *testing.T) {
	callerOptions := &s3blob.Options{
		UseLegacyList:              true,
		EncryptionType:             types.ServerSideEncryptionAwsKms,
		KMSEncryptionID:            "kms-key-id",
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenSupported,
	}
	original := *callerOptions

	options := bucketOptions(s3ClientWithChecksumPolicy(aws.RequestChecksumCalculationWhenRequired), callerOptions)

	assert.Equal(t, &s3blob.Options{
		UseLegacyList:              true,
		EncryptionType:             types.ServerSideEncryptionAwsKms,
		KMSEncryptionID:            "kms-key-id",
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
	}, options)
	assert.Equal(t, original, *callerOptions, "the caller's options must not be modified")
}

// Open must apply the client's checksum policy to uploads even when the
// caller's options say otherwise, because s3blob's transfer manager does not
// read the client's setting.
func TestOpen_UploadsUseClientChecksumPolicy(t *testing.T) {
	for _, test := range []struct {
		name      string
		policy    aws.RequestChecksumCalculation
		expect501 bool
	}{
		{name: "when_required", policy: aws.RequestChecksumCalculationWhenRequired},
		{name: "when_supported", policy: aws.RequestChecksumCalculationWhenSupported, expect501: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := newNoChunkedEncodingS3Server(t)
			client := s3.New(s3.Options{
				Region:                     "us-east-1",
				BaseEndpoint:               aws.String(server.URL),
				UsePathStyle:               true,
				HTTPClient:                 server.Client(),
				Credentials:                credentials.NewStaticCredentialsProvider("test-access-key", "test-secret-key", ""),
				RequestChecksumCalculation: test.policy,
			})
			callerOptions := &s3blob.Options{RequestChecksumCalculation: aws.RequestChecksumCalculationWhenSupported}
			if test.policy == aws.RequestChecksumCalculationWhenSupported {
				callerOptions.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			}

			bucket, err := Open(context.Background(), client, "test-bucket", callerOptions)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, bucket.Close()) })

			err = bucket.WriteAll(context.Background(), "executor-logs", []byte("hello"), nil)
			if test.expect501 {
				require.ErrorContains(t, err, "AWS chunked encoding not supported")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
