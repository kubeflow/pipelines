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

type GCSProviderConfig struct {
	Default *GCSProviderDefault `json:"default"`

	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []GCSOverride `json:"Overrides"`
}

type GCSProviderDefault struct {
	// required
	Credentials *GCSCredentials `json:"credentials"`
}

type GCSOverride struct {
	BucketName  string          `json:"bucketName"`
	KeyPrefix   string          `json:"keyPrefix"`
	Credentials *GCSCredentials `json:"credentials"`
}
type GCSCredentials struct {
	// optional
	FromEnv bool `json:"fromEnv"`
	// if FromEnv is False then SecretRef is required
	SecretRef *GCSSecretRef `json:"secretRef"`
}
type GCSSecretRef struct {
	SecretName string `json:"secretName"`
	TokenKey   string `json:"tokenKey"`
}

func (p GCSProviderConfig) ProvideSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	bucketName := bucketConfig.BucketName
	bucketPrefix := bucketConfig.Prefix

	invalidConfigErr := func(err error) error {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	params := map[string]string{}

	// 1. If provider config did not have a matching configuration for the provider inferred from pipelineroot OR
	// 2. If a user has provided query parameters
	// then we use blob.OpenBucket(ctx, config.bucketURL()) by setting "FromEnv = True"
	if p.Default == nil && p.Overrides == nil {
		params["fromEnv"] = strconv.FormatBool(true)
		return objectstore.SessionInfo{
			Provider: "gs",
			Params:   params,
		}, nil
	}

	if p.Default == nil || p.Default.Credentials == nil {
		return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing default credentials"))
	}

	params["fromEnv"] = strconv.FormatBool(p.Default.Credentials.FromEnv)
	if !p.Default.Credentials.FromEnv {
		params["secretName"] = p.Default.Credentials.SecretRef.SecretName
		params["tokenKey"] = p.Default.Credentials.SecretRef.TokenKey
	}

	// Set defaults
	sessionInfo := objectstore.SessionInfo{
		Provider: "gs",
		Params:   params,
	}

	// If there's a matching override, then override defaults with provided configs
	override := p.getOverrideByPrefix(bucketName, bucketPrefix)
	if override != nil {
		if override.Credentials == nil {
			return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing override secretref"))
		}
		params["fromEnv"] = strconv.FormatBool(override.Credentials.FromEnv)
		if !override.Credentials.FromEnv {
			if override.Credentials.SecretRef == nil {
				return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing override secretref"))
			}
			params["secretName"] = override.Credentials.SecretRef.SecretName
			params["tokenKey"] = override.Credentials.SecretRef.TokenKey
		} else {
			// Don't need a secret if pulling from Env
			delete(params, "secretName")
			delete(params, "tokenKey")
		}
	}
	return sessionInfo, nil
}

// getOverrideByPrefix returns first matching bucketname and prefix in overrides
func (p GCSProviderConfig) getOverrideByPrefix(bucketName, prefix string) *GCSOverride {
	for _, override := range p.Overrides {
		if override.BucketName == bucketName && strings.HasPrefix(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
