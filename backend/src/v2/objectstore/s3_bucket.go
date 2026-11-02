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

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// OpenS3Bucket opens bucketName through an already-configured S3 client.
// Callers keep credentials, endpoints and HTTP settings on the client, and may
// pass bucket options such as encryption settings, which are preserved.
//
// The client's RequestChecksumCalculation always overrides the option of the
// same name. s3blob uploads through the S3 transfer manager, which ignores the
// client's setting and defaults to when_supported. Over HTTPS that sends
// aws-chunked trailers, which some S3-compatible stores reject with 501.
func OpenS3Bucket(ctx context.Context, client *s3.Client, bucketName string, options *s3blob.Options) (*blob.Bucket, error) {
	return s3blob.OpenBucketV2(ctx, client, bucketName, s3BucketOptions(client, options))
}

// s3BucketOptions returns a copy of options, so the caller's struct is never
// modified, with the client's checksum policy applied.
func s3BucketOptions(client *s3.Client, options *s3blob.Options) *s3blob.Options {
	var bucketOptions s3blob.Options
	if options != nil {
		bucketOptions = *options
	}
	bucketOptions.RequestChecksumCalculation = client.Options().RequestChecksumCalculation
	return &bucketOptions
}
