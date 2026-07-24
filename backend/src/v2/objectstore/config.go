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
	"path"
	"regexp"
	"strconv"
	"strings"
)

// The endpoint uses Kubernetes service DNS name with namespace:
// https://kubernetes.io/docs/concepts/services-networking/service/#dns
const DefaultEndpointInMultiUserMode = "seaweedfs.kubeflow:9000"

// S3 session param keys shared by object store config parsing.
//
// If S3 session fields are added or renamed in the launcher/provider config
// flow, update these constants and review the S3-related helpers in this file,
// including StructuredS3Params and HasStructuredS3Settings.
const (
	S3ParamFromEnv        = "fromEnv"
	S3ParamSecretName     = "secretName"
	S3ParamAccessKeyKey   = "accessKeyKey"
	S3ParamSecretKeyKey   = "secretKeyKey"
	S3ParamRegion         = "region"
	S3ParamEndpoint       = "endpoint"
	S3ParamDisableSSL     = "disableSSL"
	S3ParamForcePathStyle = "forcePathStyle"
	S3ParamMaxRetries     = "maxRetries"
)

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
	AccessKeyKey   string
	SecretKeyKey   string
	Region         string
	Endpoint       string
	DisableSSL     bool
	ForcePathStyle bool
	MaxRetries     int
}

type HuggingFaceParams struct {
	FromEnv    bool
	SecretName string
	TokenKey   string
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
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" && ms[1] != "mem://" && ms[1] != "huggingface://" {
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
	if ms[1] != "gs://" && ms[1] != "s3://" && ms[1] != "minio://" && ms[1] != "mem://" && ms[1] != "huggingface://" {
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
	if val, ok := p[S3ParamFromEnv]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.FromEnv = boolVal
	}
	if val, ok := p[S3ParamSecretName]; ok {
		sparams.SecretName = val
	}
	// The k8s secret "Key" for "Artifact SecretKey" and "Artifact AccessKey"
	if val, ok := p[S3ParamAccessKeyKey]; ok {
		sparams.AccessKeyKey = val
	}
	if val, ok := p[S3ParamSecretKeyKey]; ok {
		sparams.SecretKeyKey = val
	}
	if val, ok := p[S3ParamRegion]; ok {
		sparams.Region = val
	}
	if val, ok := p[S3ParamEndpoint]; ok {
		sparams.Endpoint = val
	}
	if val, ok := p[S3ParamDisableSSL]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.DisableSSL = boolVal
	}
	if val, ok := p[S3ParamForcePathStyle]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.ForcePathStyle = boolVal
	} else {
		sparams.ForcePathStyle = true
	}
	if val, ok := p[S3ParamMaxRetries]; ok {
		intVal, err := strconv.ParseInt(val, 10, 0)
		if err != nil {
			return nil, err
		}
		sparams.MaxRetries = int(intVal)
	}
	return sparams, nil
}

// HasStructuredS3Settings reports whether the session params include explicit
// S3 client settings beyond credential-source metadata.
func HasStructuredS3Settings(p map[string]string) bool {
	for _, key := range []string{
		S3ParamRegion,
		S3ParamEndpoint,
		S3ParamDisableSSL,
		S3ParamForcePathStyle,
		S3ParamMaxRetries,
	} {
		if _, ok := p[key]; ok {
			return true
		}
	}
	return false
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

func StructuredHuggingFaceParams(p map[string]string) (*HuggingFaceParams, error) {
	hfparams := &HuggingFaceParams{}
	if val, ok := p["fromEnv"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		hfparams.FromEnv = boolVal
	}
	if val, ok := p["secretName"]; ok {
		hfparams.SecretName = val
	}
	if val, ok := p["tokenKey"]; ok {
		hfparams.TokenKey = val
	}
	return hfparams, nil
}
