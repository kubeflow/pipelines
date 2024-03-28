// Copyright 2021-2023 The Kubeflow Authors
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

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package config

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"io/ioutil"
	"sigs.k8s.io/yaml"
	"strings"

	"github.com/golang/glog"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapName                = "kfp-launcher"
	defaultPipelineRoot          = "minio://mlpipeline/v2/artifacts"
	configKeyDefaultPipelineRoot = "defaultPipelineRoot"
	configBucketProviders        = "providers"
	minioArtifactSecretName      = "mlpipeline-minio-artifact"
	minioArtifactSecretKeyKey    = "secretkey"
	minioArtifactAccessKeyKey    = "accesskey"
)

type BucketProviders struct {
	Minio *ProviderConfig `json:"minio"`
	S3    *ProviderConfig `json:"s3"`
	GCS   *ProviderConfig `json:"gcs"`
}

type ProviderConfig struct {
	Endpoint                 string     `json:"endpoint"`
	DefaultProviderSecretRef *SecretRef `json:"defaultProviderSecretRef"`
	Region                   string     `json:"region"`
	// optional
	DisableSSL bool `json:"disableSSL"`
	// optional, ordered, the auth config for the first matching prefix is used
	AuthConfigs []AuthConfig `json:"authConfigs"`
}

type AuthConfig struct {
	BucketName string `json:"bucketName"`
	KeyPrefix  string `json:"keyPrefix"`
	*SecretRef `json:"secretRef"`
}

type SecretRef struct {
	SecretName   string `json:"secretName"`
	AccessKeyKey string `json:"accessKeyKey"`
	SecretKeyKey string `json:"secretKeyKey"`
}

// Config is the KFP runtime configuration.
type Config struct {
	data map[string]string
}

// FromConfigMap loads config from a kfp-launcher Kubernetes config map.
func FromConfigMap(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*Config, error) {
	config, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if k8errors.IsNotFound(err) {
			glog.Infof("cannot find launcher configmap: name=%q namespace=%q, will use default config", configMapName, namespace)
			// LauncherConfig is optional, so ignore not found error.
			return nil, nil
		}
		return nil, err
	}
	return &Config{data: config.Data}, nil
}

// DefaultPipelineRoot gets the configured default pipeline root.
func (c *Config) DefaultPipelineRoot() string {
	// The key defaultPipelineRoot is optional in launcher config.
	if c == nil || c.data[configKeyDefaultPipelineRoot] == "" {
		return defaultPipelineRoot
	}
	return c.data[configKeyDefaultPipelineRoot]
}

// InPodNamespace gets current namespace from inside a Kubernetes Pod.
func InPodNamespace() (string, error) {
	// The path is available in Pods.
	// https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("failed to get namespace in Pod: %w", err)
	}
	return string(ns), nil
}

// InPodName gets the pod name from inside a Kubernetes Pod.
func InPodName() (string, error) {
	podName, err := ioutil.ReadFile("/etc/hostname")
	if err != nil {
		return "", fmt.Errorf("failed to get pod name in Pod: %w", err)
	}
	name := string(podName)
	return strings.TrimSuffix(name, "\n"), nil
}

func (c *Config) GetBucketSessionInfo(path string) (objectstore.SessionInfo, error) {
	bucketConfig, err := objectstore.ParseBucketPathToConfig(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	bucketName := bucketConfig.BucketName
	bucketPrefix := bucketConfig.Prefix
	provider := strings.TrimSuffix(bucketConfig.Scheme, "://")
	bucketProviders, err := c.getBucketProviders()
	if err != nil {
		return objectstore.SessionInfo{}, err
	}

	// Case 1: No "providers" field in kfp-launcher
	if bucketProviders == nil {
		// Use default minio if provider is minio, otherwise we default to executor env
		if provider == "minio" {
			return getDefaultMinioSessionInfo(), nil
		} else {
			// If not using minio, and no other provider config is provided
			// rely on executor env (e.g. IRSA) for authenticating with provider
			return objectstore.SessionInfo{}, nil
		}
	}

	var providerConfig *ProviderConfig
	switch provider {
	case "minio":
		providerConfig = bucketProviders.Minio
		break
	case "s3":
		providerConfig = bucketProviders.S3
		break
	case "gs":
		providerConfig = bucketProviders.GCS
		break
	default:
		return objectstore.SessionInfo{}, fmt.Errorf("Encountered unsupported provider in BucketProviders %s", provider)
	}

	// Case 2: "providers" field is empty {}
	if providerConfig == nil {
		if provider == "minio" {
			return getDefaultMinioSessionInfo(), nil
		} else {
			return objectstore.SessionInfo{}, nil
		}
	}

	// Case 3: a provider is specified
	endpoint := providerConfig.Endpoint
	if endpoint == "" {
		if provider == "minio" {
			endpoint = objectstore.MinioDefaultEndpoint()
		} else {
			return objectstore.SessionInfo{}, fmt.Errorf("Invalid provider config, %s.defaultProviderSecretRef is required for this storage provider", provider)
		}
	}

	// DefaultProviderSecretRef takes precedent over other configs
	secretRef := providerConfig.DefaultProviderSecretRef
	if secretRef == nil {
		if provider == "minio" {
			secretRef = &SecretRef{
				SecretName:   minioArtifactSecretName,
				SecretKeyKey: minioArtifactSecretKeyKey,
				AccessKeyKey: minioArtifactAccessKeyKey,
			}
		} else {
			return objectstore.SessionInfo{}, fmt.Errorf("Invalid provider config, %s.defaultProviderSecretRef is required for this storage provider", provider)
		}
	}

	// if not provided, defaults to false
	disableSSL := providerConfig.DisableSSL

	region := providerConfig.Region
	if region == "" {
		return objectstore.SessionInfo{}, fmt.Errorf("Invalid provider config, missing provider region")
	}

	// if another secret is specified for a given bucket/prefix then that takes
	// higher precedent over DefaultProviderSecretRef
	authConfig := getBucketAuthByPrefix(providerConfig.AuthConfigs, bucketName, bucketPrefix)
	if authConfig != nil {
		if authConfig.SecretRef == nil || authConfig.SecretRef.SecretKeyKey == "" || authConfig.SecretRef.AccessKeyKey == "" || authConfig.SecretRef.SecretName == "" {
			return objectstore.SessionInfo{}, fmt.Errorf("Invalid provider config, %s.AuthConfigs[].secretRef is missing or invalid", provider)
		}
		secretRef = authConfig.SecretRef
	}

	return objectstore.SessionInfo{
		Region:       region,
		Endpoint:     endpoint,
		DisableSSL:   disableSSL,
		SecretName:   secretRef.SecretName,
		AccessKeyKey: secretRef.AccessKeyKey,
		SecretKeyKey: secretRef.SecretKeyKey,
	}, nil
}

func getDefaultMinioSessionInfo() (sessionInfo objectstore.SessionInfo) {
	sess := objectstore.SessionInfo{
		Region:       "minio",
		Endpoint:     objectstore.MinioDefaultEndpoint(),
		DisableSSL:   true,
		SecretName:   minioArtifactSecretName,
		AccessKeyKey: minioArtifactAccessKeyKey,
		SecretKeyKey: minioArtifactSecretKeyKey,
	}
	return sess
}

// getBucketProviders gets the provider configuration
func (c *Config) getBucketProviders() (*BucketProviders, error) {
	if c == nil || c.data[configBucketProviders] == "" {
		return nil, nil
	}
	bucketProviders := &BucketProviders{}
	configAuth := c.data[configBucketProviders]
	err := yaml.Unmarshal([]byte(configAuth), bucketProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall kfp bucket providers, ensure that providers config is well formed: %w", err)
	}
	return bucketProviders, nil
}

// getBucketAuthByPrefix returns first matching bucketname and prefix in authConfigs
func getBucketAuthByPrefix(authConfigs []AuthConfig, bucketName, prefix string) *AuthConfig {
	for _, authConfig := range authConfigs {
		if authConfig.BucketName == bucketName && strings.HasPrefix(prefix, authConfig.KeyPrefix) {
			return &authConfig
		}
	}
	return nil
}
