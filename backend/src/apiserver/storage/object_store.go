// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	v1 "k8s.io/api/core/v1"
)

// ObjectStore is the interface for object store operations.
type ObjectStore interface {
	AddFile(ctx context.Context, template []byte, filePath string) error
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	// GetFileReader returns a streaming reader for the file content.
	// Use this method instead of GetFile for streaming access to large files.
	GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error)
	AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error
	GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error
	GetPipelineKey(pipelineId string) string
	GetSignedUrl(ctx context.Context, bucketConfig *objectstore.Config, secret *v1.Secret, expirySeconds time.Duration, artifactURI string, queryParams url.Values) (string, error)
	GetObjectSize(ctx context.Context, bucketConfig *objectstore.Config, secret *v1.Secret, artifactURI string) (int64, error)
}

// buildClientFromConfig returns a minio s3 client constructed via the bucket identified by bucketConfig.
func buildClientFromConfig(bucketConfig *objectstore.Config, secret *v1.Secret) (*minio.Client, error) {
	params, err := objectstore.StructuredS3Params(bucketConfig.SessionInfo.Params)
	if err != nil {
		return nil, err
	}

	accessKey := string(secret.Data[params.AccessKeyKey])
	secretKey := string(secret.Data[params.SecretKeyKey])
	parsedUrl, err := url.Parse(params.Endpoint)
	if err != nil {
		return nil, util.Wrap(err, "Failed to parse object store endpoint.")
	}

	var secure bool
	switch parsedUrl.Scheme {
	case "http":
		secure = false
	case "https":
		secure = !params.DisableSSL
	}
	s3Client, err := minio.New(
		parsedUrl.Host, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: secure,
		})
	if err != nil {
		return nil, util.Wrap(err, "Failed to create s3 client.")
	}
	return s3Client, nil
}
