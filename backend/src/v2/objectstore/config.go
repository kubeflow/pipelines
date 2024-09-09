// Copyright 2024 The Kubeflow Authors
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

// This package contains helper methods for using object stores.
package objectstore

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// The endpoint uses Kubernetes service DNS name with namespace:
// https://kubernetes.io/docs/concepts/services-networking/service/#dns
const defaultMinioEndpointInMultiUserMode = "minio-service.kubeflow:9000"

type Config struct {
	Scheme      string
	BucketName  string
	Prefix      string
	QueryString string
	SessionInfo *SessionInfo
}

type SessionInfo struct {
	Provider string
	Params   map[string]string
}

type GCSParams struct {
	FromEnv    bool
	SecretName string
	TokenKey   string
}

type S3Params struct {
	FromEnv    bool
	SecretName string
	// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
	AccessKeyKey string
	SecretKeyKey string
	Region       string
	Endpoint     string
	DisableSSL   bool
}

func (b *Config) bucketURL() string {
	u := b.Scheme + b.BucketName

	// append prefix=b.prefix to existing queryString
	q := b.QueryString
	if len(b.Prefix) > 0 {
		if len(q) > 0 {
			q = q + "&prefix=" + b.Prefix
		} else {
			q = "?prefix=" + b.Prefix
		}
	}

	u = u + q
	return u
}

func (b *Config) PrefixedBucket() string {
	return b.Scheme + path.Join(b.BucketName, b.Prefix)
}

func (b *Config) KeyFromURI(uri string) (string, error) {
	prefixedBucket := b.PrefixedBucket()
	if !strings.HasPrefix(uri, prefixedBucket) {
		return "", fmt.Errorf("URI %q does not have expected bucket prefix %q", uri, prefixedBucket)
	}

	key := strings.TrimLeft(strings.TrimPrefix(uri, prefixedBucket), "/")
	if len(key) == 0 {
		return "", fmt.Errorf("URI %q has empty key given prefixed bucket %q", uri, prefixedBucket)
	}
	return key, nil
}

func (b *Config) UriFromKey(blobKey string) string {
	return b.Scheme + path.Join(b.BucketName, b.Prefix, blobKey)
}

var bucketPattern = regexp.MustCompile(`(^[a-z][a-z0-9]+:///?)([^/?]+)(/[^?]*)?(\?.+)?$`)

func ParseBucketConfig(path string, sess *SessionInfo) (*Config, error) {
	config, err := ParseBucketPathToConfig(path)
	if err != nil {
		return nil, err
	}
	config.SessionInfo = sess

	return config, nil
}

func ParseBucketPathToConfig(path string) (*Config, error) {
	ms := bucketPattern.FindStringSubmatch(path)
	if ms == nil || len(ms) != 5 {
		return nil, fmt.Errorf("parse bucket config failed: unrecognized pipeline root format: %q", path)
	}

	// TODO: Verify/add support for file:///.
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" && ms[1] != "mem://" {
		return nil, fmt.Errorf("parse bucket config failed: unsupported Cloud bucket: %q", path)
	}

	prefix := strings.TrimPrefix(ms[3], "/")
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return &Config{
		Scheme:      ms[1],
		BucketName:  ms[2],
		Prefix:      prefix,
		QueryString: ms[4],
	}, nil
}

func ParseBucketConfigForArtifactURI(uri string) (*Config, error) {
	ms := bucketPattern.FindStringSubmatch(uri)
	if ms == nil || len(ms) != 5 {
		return nil, fmt.Errorf("parse bucket config failed: unrecognized uri format: %q", uri)
	}

	// TODO: Verify/add support for file:///.
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" && ms[1] != "mem://" {
		return nil, fmt.Errorf("parse bucket config failed: unsupported Cloud bucket: %q", uri)
	}

	return &Config{
		Scheme:     ms[1],
		BucketName: ms[2],
	}, nil
}

// ParseProviderFromPath prases the uri and returns the scheme, which is
// used as the Provider string
func ParseProviderFromPath(uri string) (string, error) {
	bucketConfig, err := ParseBucketPathToConfig(uri)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(bucketConfig.Scheme, "://"), nil
}

func MinioDefaultEndpoint() string {
	// Discover minio-service in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	minioHost := os.Getenv("MINIO_SERVICE_SERVICE_HOST")
	minioPort := os.Getenv("MINIO_SERVICE_SERVICE_PORT")
	if minioHost != "" && minioPort != "" {
		// If there is a minio-service Kubernetes service in the same namespace,
		// MINIO_SERVICE_SERVICE_HOST and MINIO_SERVICE_SERVICE_PORT env vars should
		// exist by default, so we use it as default.
		return minioHost + ":" + minioPort
	}
	// If the env vars do not exist, we guess that we are running in KFP multi user mode, so default minio service should be `minio-service.kubeflow:9000`.
	glog.Infof("Cannot detect minio-service in the same namespace, default to %s as MinIO endpoint.", defaultMinioEndpointInMultiUserMode)
	return defaultMinioEndpointInMultiUserMode
}

func GetSessionInfoFromString(sessionInfoJSON string) (*SessionInfo, error) {
	sessionInfo := &SessionInfo{}
	if sessionInfoJSON == "" {
		return nil, nil
	}
	err := json.Unmarshal([]byte(sessionInfoJSON), sessionInfo)
	if err != nil {
		return nil, fmt.Errorf("Encountered error when attempting to unmarshall bucket session info properties: %w", err)
	}
	return sessionInfo, nil
}

func StructuredS3Params(p map[string]string) (*S3Params, error) {
	sparams := &S3Params{}
	if val, ok := p["fromEnv"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.FromEnv = boolVal
	}
	if val, ok := p["secretName"]; ok {
		sparams.SecretName = val
	}
	// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
	if val, ok := p["accessKeyKey"]; ok {
		sparams.AccessKeyKey = val
	}
	if val, ok := p["secretKeyKey"]; ok {
		sparams.SecretKeyKey = val
	}
	if val, ok := p["region"]; ok {
		sparams.Region = val
	}
	if val, ok := p["endpoint"]; ok {
		sparams.Endpoint = val
	}
	if val, ok := p["disableSSL"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.DisableSSL = boolVal
	}
	return sparams, nil
}

func StructuredGCSParams(p map[string]string) (*GCSParams, error) {
	sparams := &GCSParams{}
	if val, ok := p["fromEnv"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.FromEnv = boolVal
	}
	if val, ok := p["secretName"]; ok {
		sparams.SecretName = val
	}
	if val, ok := p["tokenKey"]; ok {
		sparams.TokenKey = val
	}
	return sparams, nil
}
