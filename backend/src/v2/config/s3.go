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
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"strconv"
	"strings"
)

type S3ProviderConfig struct {
	Default *S3ProviderDefault `json:"default"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []S3Override `json:"Overrides"`
}

type S3ProviderDefault struct {
	Endpoint    string         `json:"endpoint"`
	Credentials *S3Credentials `json:"credentials"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL *bool `json:"disableSSL"`
}

type S3Credentials struct {
	// optional
	FromEnv bool `json:"fromEnv"`
	// if FromEnv is False then SecretRef is required
	SecretRef *S3SecretRef `json:"secretRef"`
}
type S3Override struct {
	Endpoint string `json:"endpoint"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL *bool  `json:"disableSSL"`
	BucketName string `json:"bucketName"`
	KeyPrefix  string `json:"keyPrefix"`
	// required
	Credentials *S3Credentials `json:"credentials"`
}
type S3SecretRef struct {
	SecretName string `json:"secretName"`
	// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
	AccessKeyKey string `json:"accessKeyKey"`
	SecretKeyKey string `json:"secretKeyKey"`
}

func (p S3ProviderConfig) ProvideSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	bucketName := bucketConfig.BucketName
	bucketPrefix := bucketConfig.Prefix
	queryString := bucketConfig.QueryString

	invalidConfigErr := func(err error) error {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	params := map[string]string{}

	// 1. If provider config did not have a matching configuration for the provider inferred from pipelineroot OR
	// 2. If a user has provided query parameters
	// then we use blob.OpenBucket(ctx, config.bucketURL()) by setting "FromEnv = True"
	if (p.Default == nil && p.Overrides == nil) || queryString != "" {
		params["fromEnv"] = strconv.FormatBool(true)
		return objectstore.SessionInfo{
			Provider: "s3",
			Params:   params,
		}, nil
	}

	if p.Default == nil || p.Default.Credentials == nil {
		return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing default credentials"))
	}

	params["endpoint"] = p.Default.Endpoint
	params["region"] = p.Default.Region

	if p.Default.DisableSSL == nil {
		params["disableSSL"] = strconv.FormatBool(false)
	} else {
		params["disableSSL"] = strconv.FormatBool(*p.Default.DisableSSL)
	}

	params["fromEnv"] = strconv.FormatBool(p.Default.Credentials.FromEnv)
	if !p.Default.Credentials.FromEnv {
		params["secretName"] = p.Default.Credentials.SecretRef.SecretName
		params["accessKeyKey"] = p.Default.Credentials.SecretRef.AccessKeyKey
		params["secretKeyKey"] = p.Default.Credentials.SecretRef.SecretKeyKey
	}

	// Set defaults
	sessionInfo := objectstore.SessionInfo{
		Provider: "s3",
		Params:   params,
	}

	// If there's a matching override, then override defaults with provided configs
	override := p.getOverrideByPrefix(bucketName, bucketPrefix)
	if override != nil {
		if override.Endpoint != "" {
			sessionInfo.Params["endpoint"] = override.Endpoint
		}
		if override.Region != "" {
			sessionInfo.Params["region"] = override.Region
		}
		if override.DisableSSL != nil {
			sessionInfo.Params["disableSSL"] = strconv.FormatBool(*override.DisableSSL)
		}
		if override.Credentials == nil {
			return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing override credentials"))
		}
		params["fromEnv"] = strconv.FormatBool(override.Credentials.FromEnv)
		if !override.Credentials.FromEnv {
			if override.Credentials.SecretRef == nil {
				return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing override secretref"))
			}
			params["secretName"] = override.Credentials.SecretRef.SecretName
			params["accessKeyKey"] = override.Credentials.SecretRef.AccessKeyKey
			params["secretKeyKey"] = override.Credentials.SecretRef.SecretKeyKey
		} else {
			// Don't need a secret if pulling from Env
			delete(params, "secretName")
			delete(params, "accessKeyKey")
			delete(params, "secretKeyKey")
		}
	}
	return sessionInfo, nil
}

// getOverrideByPrefix returns first matching bucketname and prefix in overrides
func (p S3ProviderConfig) getOverrideByPrefix(bucketName, prefix string) *S3Override {
	for _, override := range p.Overrides {
		if override.BucketName == bucketName && strings.HasPrefix(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
