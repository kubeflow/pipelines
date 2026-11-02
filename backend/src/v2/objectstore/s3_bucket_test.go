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

func TestS3BucketOptions_PropagatesClientChecksumPolicy(t *testing.T) {
	for _, test := range []struct {
		name   string
		policy aws.RequestChecksumCalculation
	}{
		{name: "when_required", policy: aws.RequestChecksumCalculationWhenRequired},
		{name: "when_supported", policy: aws.RequestChecksumCalculationWhenSupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := s3BucketOptions(s3ClientWithChecksumPolicy(test.policy), nil)

			assert.Equal(t, test.policy, options.RequestChecksumCalculation)
		})
	}
}

func TestS3BucketOptions_PreservesCallerOptions(t *testing.T) {
	callerOptions := &s3blob.Options{
		UseLegacyList:              true,
		EncryptionType:             types.ServerSideEncryptionAwsKms,
		KMSEncryptionID:            "kms-key-id",
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenSupported,
	}
	original := *callerOptions

	options := s3BucketOptions(s3ClientWithChecksumPolicy(aws.RequestChecksumCalculationWhenRequired), callerOptions)

	assert.Equal(t, &s3blob.Options{
		UseLegacyList:              true,
		EncryptionType:             types.ServerSideEncryptionAwsKms,
		KMSEncryptionID:            "kms-key-id",
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
	}, options)
	assert.Equal(t, original, *callerOptions, "the caller's options must not be modified")
}

// OpenS3Bucket must apply the client's checksum policy to uploads even when the
// caller's options say otherwise, because s3blob's transfer manager does not
// read the client's setting.
func TestOpenS3Bucket_UploadsUseClientChecksumPolicy(t *testing.T) {
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

			bucket, err := OpenS3Bucket(context.Background(), client, "test-bucket", callerOptions)
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
