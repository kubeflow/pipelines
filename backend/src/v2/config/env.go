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

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"sigs.k8s.io/yaml"

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
	// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
	minioArtifactSecretKeyKey = "secretkey"
	minioArtifactAccessKeyKey = "accesskey"
)

const (
	mlPipelineGrpcServicePort = "8887"
)

type BucketProviders struct {
	Minio *MinioProviderConfig `json:"minio"`
	S3    *S3ProviderConfig    `json:"s3"`
	GCS   *GCSProviderConfig   `json:"gs"`
}

type SessionInfoProvider interface {
	ProvideSessionInfo(path string) (objectstore.SessionInfo, error)
}

// Config is the KFP runtime configuration.
type Config struct {
	data map[string]string
}

type ServerConfig struct {
	Address string
	Port    string
}

// DefaultPipelineRoot gets the configured default pipeline root.
func (c *Config) DefaultPipelineRoot() string {
	// The key defaultPipelineRoot is optional in launcher config.
	if c == nil || c.data == nil {
		return defaultPipelineRoot
	}
	// Check if key exists and has non-empty value
	if val, exists := c.data[configKeyDefaultPipelineRoot]; !exists || val == "" {
		return defaultPipelineRoot
	}
	return c.data[configKeyDefaultPipelineRoot]
}

func (c *Config) GetStoreSessionInfo(path string) (objectstore.SessionInfo, error) {
	provider, err := objectstore.ParseProviderFromPath(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	bucketProviders, err := c.getBucketProviders()
	if err != nil {
		return objectstore.SessionInfo{}, err
	}

	var sessProvider SessionInfoProvider

	switch provider {
	case "minio":
		if bucketProviders == nil || bucketProviders.Minio == nil {
			sessProvider = &MinioProviderConfig{}
		} else {
			sessProvider = bucketProviders.Minio
		}
		break
	case "s3":
		if bucketProviders == nil || bucketProviders.S3 == nil {
			sessProvider = &S3ProviderConfig{}
		} else {
			sessProvider = bucketProviders.S3
		}
		break
	case "gs":
		if bucketProviders == nil || bucketProviders.GCS == nil {
			sessProvider = &GCSProviderConfig{}
		} else {
			sessProvider = bucketProviders.GCS
		}
		break
	default:
		return objectstore.SessionInfo{}, fmt.Errorf("Encountered unsupported provider in provider config %s", provider)
	}

	sess, err := sessProvider.ProvideSessionInfo(path)
	if err != nil {
		return objectstore.SessionInfo{}, err
	}
	return sess, nil
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

// FetchLauncherConfigMap loads config from a kfp-launcher Kubernetes config map.
func FetchLauncherConfigMap(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*Config, error) {
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

// GetPipelineRootWithPipelineRunContext gets the pipeline root for a run.
// The returned Pipeline Root appends the pipeline name and run id.
func GetPipelineRootWithPipelineRunContext(
	ctx context.Context,
	pipelineName, namespace string,
	k8sClient kubernetes.Interface,
	run *go_client.Run) (string, error) {
	var pipelineRoot string
	if run.RuntimeConfig != nil && run.RuntimeConfig.PipelineRoot != "" {
		pipelineRoot = run.RuntimeConfig.PipelineRoot
		glog.Infof("PipelineRoot=%q from runtime config will be used.", pipelineRoot)
	} else {
		cfg, err := FetchLauncherConfigMap(ctx, k8sClient, namespace)
		if err != nil {
			return "", fmt.Errorf("failed to fetch launcher configmap: %w", err)
		}
		pipelineRoot = cfg.DefaultPipelineRoot()
		glog.Infof("PipelineRoot=%q from default config", pipelineRoot)
	}

	pipelineRootAppended := util.GenerateOutputURI(pipelineRoot, []string{pipelineName, run.RunId}, true)
	return pipelineRootAppended, nil
}

func getDefaultMinioSessionInfo() (objectstore.SessionInfo, error) {
	sess := objectstore.SessionInfo{
		Provider: "minio",
		Params: map[string]string{
			"region":     "minio",
			"endpoint":   getDefaultMinioHost(),
			"disableSSL": strconv.FormatBool(true),
			"fromEnv":    strconv.FormatBool(false),
			"maxRetries": strconv.FormatInt(int64(5), 10),
			"secretName": minioArtifactSecretName,
			// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
			"accessKeyKey": minioArtifactAccessKeyKey,
			"secretKeyKey": minioArtifactSecretKeyKey,
		},
	}
	return sess, nil
}

func getDefaultMinioHost() string {
	endpoint := objectstore.DefaultMinioEndpointInMultiUserMode
	var host, port string
	if os.Getenv("OBJECT_STORE_HOST") != "" {
		host = os.Getenv("OBJECT_STORE_HOST")
	}
	if os.Getenv("OBJECT_STORE_PORT") != "" {
		port = os.Getenv("OBJECT_STORE_PORT")
	}
	if host != "" && port != "" {
		return fmt.Sprintf("%s:%s", host, port)
	} else {
		return endpoint
	}
}

func GetMLPipelineServerConfig() *ServerConfig {
	return &ServerConfig{
		Address: common.GetMLPipelineServiceName() + "." + common.GetPodNamespace(),
		Port:    mlPipelineGrpcServicePort,
	}
}
