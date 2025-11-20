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

// Package objectstore contains helper methods for using object stores.
package objectstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// DefaultMinioEndpointInMultiUserMode uses Kubernetes service DNS name with namespace:
// https://kubernetes.io/docs/concepts/services-networking/service/#dns
const DefaultMinioEndpointInMultiUserMode = "minio-service.kubeflow:9000"

type Config struct {
	Scheme      string
	BucketName  string
	Prefix      string
	QueryString string
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

	u += q
	return u
}

func (b *Config) PrefixedBucket() string {
	return b.Scheme + path.Join(b.BucketName, b.Prefix)
}

func (b *Config) Hash() string {
	h := sha256.New()
	h.Write([]byte(b.Scheme))
	h.Write([]byte(b.BucketName))
	h.Write([]byte(b.Prefix))
	h.Write([]byte(b.QueryString))
	return hex.EncodeToString(h.Sum(nil))
}

var bucketPattern = regexp.MustCompile(`(^[a-z][a-z0-9]+:///?)([^/?]+)(/[^?]*)?(\?.+)?$`)

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
		prefix += "/"
	}

	return &Config{
		Scheme:      ms[1],
		BucketName:  ms[2],
		Prefix:      prefix,
		QueryString: ms[4],
	}, nil
}

func SplitObjectURI(uri string) (prefix, base string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", fmt.Errorf("invalid URI: %w", err)
	}

	// Trim trailing slash (if any)
	cleanPath := strings.TrimSuffix(u.Path, "/")

	// Get base name and dir prefix
	base = path.Base(cleanPath)
	dir := path.Dir(cleanPath)

	// Reconstruct prefix (scheme + host + dir)
	prefix = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if dir != "." && dir != "/" {
		prefix += dir
	}
	return prefix, base, nil
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
	if val, ok := p["forcePathStyle"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		sparams.ForcePathStyle = boolVal
	} else {
		sparams.ForcePathStyle = true
	}
	if val, ok := p["maxRetries"]; ok {
		intVal, err := strconv.ParseInt(val, 10, 0)
		if err != nil {
			return nil, err
		}
		sparams.MaxRetries = int(intVal)
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
