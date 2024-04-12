// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
)

type MinioProviderConfig S3ProviderConfig

// ProvideSessionInfo provides the SessionInfo for minio provider.
// this is the same as s3ProviderConfig.ProvideSessionInfo except
// the provider is set to minio
func (p MinioProviderConfig) ProvideSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	queryString := bucketConfig.QueryString

	// When using minio root, with no query strings, if no matching provider in kfp-launcher exists
	// we use the default minio configurations
	if (p.Default == nil && p.Overrides == nil) && queryString == "" {
		sess, sessErr := getDefaultMinioSessionInfo()
		if sessErr != nil {
			return objectstore.SessionInfo{}, nil
		}
		return sess, nil
	}

	s3ProviderConfig := S3ProviderConfig(p)

	info, err := s3ProviderConfig.ProvideSessionInfo(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	info.Provider = "minio"
	return info, nil
}
