// Copyright 2026 The Kubeflow Authors
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
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
)

// OCIProviderConfig configures Oracle Cloud Infrastructure (OCI) Object Storage
// as an artifact store. Pipeline roots use the oci://<bucket>@<namespace>/<prefix>
// form; the namespace is taken from the path, so the provider only needs the
// region (to derive the S3-compatible endpoint) and a Customer Secret Key.
type OCIProviderConfig struct {
	Default *OCIProviderDefault `json:"default"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []OCIOverride `json:"Overrides"`
}

type OCIProviderDefault struct {
	// Region of the Object Storage namespace, e.g. us-ashburn-1. Required unless
	// Endpoint is set; falls back to the OCI_REGION environment variable of the
	// launcher at runtime.
	Region string `json:"region"`
	// optional, overrides the derived
	// https://<namespace>.compat.objectstorage.<region>.oraclecloud.com endpoint
	Endpoint string `json:"endpoint"`
	// Customer Secret Key (access key / secret key pair) used to sign requests.
	Credentials *S3Credentials `json:"credentials"`
	// optional, defaults to true; the S3-compatible API is path-style only
	ForcePathStyle *bool `json:"forcePathStyle"`
	// optional
	MaxRetries *int `json:"maxRetries"`
}

type OCIOverride struct {
	BucketName string `json:"bucketName"`
	// optional, restricts the override to buckets in this Object Storage namespace
	Namespace string `json:"namespace"`
	KeyPrefix string `json:"keyPrefix"`
	// optional
	Region string `json:"region"`
	// optional
	Endpoint string `json:"endpoint"`
	// required
	Credentials *S3Credentials `json:"credentials"`
	// optional
	ForcePathStyle *bool `json:"forcePathStyle"`
	// optional
	MaxRetries *int `json:"maxRetries"`
}

func (p OCIProviderConfig) ProvideSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}

	invalidConfigErr := func(err error) error {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	params := map[string]string{}

	// 1. If provider config did not have a matching configuration for the provider inferred from pipelineroot OR
	// 2. If a user has provided query parameters
	// then credentials come from the environment and the endpoint is derived from the path/query.
	if (p.Default == nil && p.Overrides == nil) || bucketConfig.QueryString != "" {
		params["fromEnv"] = strconv.FormatBool(true)
		return objectstore.SessionInfo{
			Provider: objectstore.OCIProvider,
			Params:   params,
		}, nil
	}

	if p.Default == nil || p.Default.Credentials == nil {
		return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing default credentials"))
	}

	if p.Default.Region != "" {
		params["region"] = p.Default.Region
	}
	if p.Default.Endpoint != "" {
		params["endpoint"] = p.Default.Endpoint
	}
	if p.Default.ForcePathStyle == nil {
		params["forcePathStyle"] = strconv.FormatBool(true)
	} else {
		params["forcePathStyle"] = strconv.FormatBool(*p.Default.ForcePathStyle)
	}
	if p.Default.MaxRetries == nil {
		params["maxRetries"] = strconv.FormatInt(5, 10)
	} else {
		params["maxRetries"] = strconv.FormatInt(int64(*p.Default.MaxRetries), 10)
	}
	if err := applyOCICredentials(params, p.Default.Credentials, "default"); err != nil {
		return objectstore.SessionInfo{}, invalidConfigErr(err)
	}

	sessionInfo := objectstore.SessionInfo{
		Provider: objectstore.OCIProvider,
		Params:   params,
	}

	// If there's a matching override, then override defaults with provided configs
	override := p.getOverrideByPrefix(bucketConfig.BucketName, bucketConfig.Namespace, bucketConfig.Prefix)
	if override != nil {
		if override.Region != "" {
			params["region"] = override.Region
		}
		if override.Endpoint != "" {
			params["endpoint"] = override.Endpoint
		}
		if override.ForcePathStyle != nil {
			params["forcePathStyle"] = strconv.FormatBool(*override.ForcePathStyle)
		}
		if override.MaxRetries != nil {
			params["maxRetries"] = strconv.FormatInt(int64(*override.MaxRetries), 10)
		}
		if override.Credentials == nil {
			return objectstore.SessionInfo{}, invalidConfigErr(fmt.Errorf("missing override credentials"))
		}
		if err := applyOCICredentials(params, override.Credentials, "override"); err != nil {
			return objectstore.SessionInfo{}, invalidConfigErr(err)
		}
	}
	return sessionInfo, nil
}

// applyOCICredentials writes the credential source of creds into params,
// removing any secret reference left over from a previous level when the
// credentials come from the environment.
func applyOCICredentials(params map[string]string, creds *S3Credentials, level string) error {
	params["fromEnv"] = strconv.FormatBool(creds.FromEnv)
	if creds.FromEnv {
		delete(params, "secretName")
		delete(params, "accessKeyKey")
		delete(params, "secretKeyKey")
		return nil
	}
	if creds.SecretRef == nil {
		return fmt.Errorf("missing %s secretref", level)
	}
	params["secretName"] = creds.SecretRef.SecretName
	params["accessKeyKey"] = creds.SecretRef.AccessKeyKey
	params["secretKeyKey"] = creds.SecretRef.SecretKeyKey
	return nil
}

func (p OCIProviderConfig) HasExplicitOverride(path string) (bool, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return false, err
	}
	return p.getOverrideByPrefix(bucketConfig.BucketName, bucketConfig.Namespace, bucketConfig.Prefix) != nil, nil
}

// getOverrideByPrefix returns the first override matching the bucket name,
// the namespace (when the override pins one) and the key prefix.
func (p OCIProviderConfig) getOverrideByPrefix(bucketName, namespace, prefix string) *OCIOverride {
	for _, override := range p.Overrides {
		if override.BucketName != bucketName {
			continue
		}
		if override.Namespace != "" && override.Namespace != namespace {
			continue
		}
		if prefixMatchesOverride(prefix, override.KeyPrefix) {
			return &override
		}
	}
	return nil
}
