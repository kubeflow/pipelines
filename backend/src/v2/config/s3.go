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
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
)

type S3ProviderConfig struct {
	Default *S3ProviderDefault `json:"default"`
	// optional, ordered, the auth config for the first matching prefix is used
	Overrides []S3Override `json:"Overrides"`
	// optional; controls whether an artifact URI's own query string (e.g.
	// ?endpoint=...) is honored when no Default/Overrides configuration
	// exists for this provider at all. Defaults to true for upgrade
	// compatibility with existing importer/runtime-URI artifacts; a future
	// release will default this to false. Has no effect once any
	// Default/Overrides configuration exists for this provider -- that
	// configuration is always authoritative and a query can never bypass it.
	AllowUnmanagedProviderQueries *bool `json:"allowUnmanagedProviderQueries"`
}

type S3ProviderDefault struct {
	Endpoint    string         `json:"endpoint"`
	Credentials *S3Credentials `json:"credentials"`
	// optional for any non aws s3 provider
	Region string `json:"region"`
	// optional
	DisableSSL *bool `json:"disableSSL"`
	// optional
	ForcePathStyle *bool `json:"forcePathStyle"`
	// optional
	MaxRetries *int `json:"maxRetries"`
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
	// optional
	ForcePathStyle *bool `json:"forcePathStyle"`
	// optional
	MaxRetries *int `json:"maxRetries"`
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

	// No Default/Overrides configuration exists for this provider at all, so
	// there is no admin policy to enforce for this bucket. Defer to
	// blob.OpenBucket(ctx, config.bucketURL()) by setting "FromEnv = True" --
	// which lets an artifact URI's own query string (if any) supply
	// endpoint/region/disableSSL, gated by AllowUnmanagedProviderQueries.
	//
	// Once any Default/Overrides configuration exists for this provider, it
	// is always authoritative below: a query string can never bypass it.
	if p.Default == nil && p.Overrides == nil {
		if queryString != "" {
			if !p.allowUnmanagedProviderQueries() {
				return objectstore.SessionInfo{}, fmt.Errorf(
					"artifact URI provider query for bucket %q rejected: no S3 provider configuration exists and allowUnmanagedProviderQueries is disabled", bucketName)
			}
			if err := validateUnmanagedProviderQuery(queryString); err != nil {
				return objectstore.SessionInfo{}, fmt.Errorf("artifact URI provider query for bucket %q rejected: %w", bucketName, err)
			}
			glog.Warningf("DEPRECATED: honoring an artifact URI's own provider query for bucket %q with no admin S3 provider configuration; a future release will require allowUnmanagedProviderQueries=true for this", bucketName)
		}
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
		if override.ForcePathStyle != nil {
			sessionInfo.Params["forcePathStyle"] = strconv.FormatBool(*override.ForcePathStyle)
		}
		if override.MaxRetries != nil {
			sessionInfo.Params["maxRetries"] = strconv.FormatInt(int64(*override.MaxRetries), 10)
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

// allowUnmanagedProviderQueries reports whether an artifact URI's own query
// string may be honored when no Default/Overrides configuration exists for
// this provider. Defaults to true when unset for upgrade compatibility.
func (p S3ProviderConfig) allowUnmanagedProviderQueries() bool {
	if p.AllowUnmanagedProviderQueries == nil {
		return true
	}
	return *p.AllowUnmanagedProviderQueries
}

// validateUnmanagedProviderQuery applies SSRF/TLS-downgrade guardrails to an
// artifact URI's own provider query, for the unmanaged case only -- no
// Default/Overrides configuration exists for this bucket, so there is no
// admin override to grant permission for disableSSL, and no allowlist entry
// to trust an endpoint against. Once any admin configuration exists for a
// bucket, this function is never called: the admin's settings are used
// as-is and this validation does not apply.
//
// This only rejects a literal loopback/link-local/private IP address in the
// endpoint. It deliberately does not resolve hostnames: a DNS lookup here
// would still not defend against DNS rebinding (the resolved address can
// legitimately differ between this check and the connection actually made
// later), would add a real network call to what is otherwise pure config
// parsing, and would give false confidence about a guarantee this function
// cannot make. Closing the hostname/DNS-rebinding gap requires resolve-time
// IP pinning and/or NetworkPolicy egress control at the cluster level.
func validateUnmanagedProviderQuery(queryString string) error {
	values, err := url.ParseQuery(strings.TrimPrefix(queryString, "?"))
	if err != nil {
		return fmt.Errorf("invalid provider query: %w", err)
	}

	if disableSSL := values.Get(objectstore.S3ParamDisableSSL); disableSSL != "" {
		if b, err := strconv.ParseBool(disableSSL); err == nil && b {
			return fmt.Errorf("%s is only permitted via an admin-configured provider override", objectstore.S3ParamDisableSSL)
		}
	}

	endpoint := values.Get(objectstore.S3ParamEndpoint)
	if endpoint == "" {
		return nil
	}
	host := endpoint
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		host = u.Host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")

	ip := net.ParseIP(host)
	if ip == nil {
		// Not a literal IP -- see the caveat above on why this function does
		// not resolve hostnames.
		return nil
	}
	return rejectUnsafeIP(host, ip)
}

func rejectUnsafeIP(host string, ip net.IP) error {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
		return fmt.Errorf("endpoint %q resolves to a loopback/link-local/private address (%s), which is not permitted", host, ip.String())
	}
	return nil
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
