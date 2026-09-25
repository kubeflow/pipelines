// Copyright 2026 The Kubeflow Authors
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

package objectstore

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
	"k8s.io/client-go/kubernetes"
)

// Oracle Cloud Infrastructure (OCI) Object Storage support.
//
// OCI Object Storage paths use the oci://<bucket>@<namespace>/<prefix> form shared
// by the OCI CLI, ocifs and OCI Data Science. The "@<namespace>" part is
// mandatory: KFP already uses the bare oci://<registry>/<repository>[:tag] form
// for Modelcar container images, and image references never carry an "@" in
// their authority, so an "@" in the first path segment is what distinguishes an
// Object Storage path from a container image reference.
//
// Buckets are opened through the S3-compatible API that every OCI namespace
// exposes at https://<namespace>.compat.objectstorage.<region>.oraclecloud.com.
// Requests are signed with a Customer Secret Key (an S3-style access key /
// secret key pair) read from a Kubernetes Secret or from the environment. Native
// OCI IAM principals (instance/resource principals, API-key request signing) are
// not supported by this driver; they would require a dedicated blob.Bucket
// implementation on top of the OCI Go SDK.

const (
	// OCIScheme is the URI scheme shared by OCI Object Storage paths and
	// Modelcar container image references.
	OCIScheme = "oci://"
	// OCIProvider is the provider identifier used in SessionInfo and in the
	// kfp-launcher "providers" configuration.
	OCIProvider = "oci"
	// OCIRegionEnv is the environment variable consulted for the Object Storage
	// region when neither the providers config nor the pipeline root specify one.
	OCIRegionEnv = "OCI_REGION"

	ociNamespaceSeparator = "@"
)

var ociCompatEndpointPattern = regexp.MustCompile(
	`^(?:https?://)?[a-z0-9]+\.compat\.objectstorage\.([a-z0-9-]+)\.oraclecloud\.com(?::[0-9]+)?/?$`,
)

// OCICompatEndpoint returns the S3-compatible Object Storage endpoint for the
// given namespace and region.
func OCICompatEndpoint(namespace, region string) string {
	return fmt.Sprintf("https://%s.compat.objectstorage.%s.oraclecloud.com", namespace, region)
}

// OCIRegionFromEndpoint extracts the region from an OCI S3-compatible endpoint.
// It returns "" when the endpoint does not follow the
// <namespace>.compat.objectstorage.<region>.oraclecloud.com convention.
func OCIRegionFromEndpoint(endpoint string) string {
	ms := ociCompatEndpointPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(endpoint)))
	if ms == nil {
		return ""
	}
	return ms[1]
}

// IsOCIObjectStorageURI reports whether uri is an OCI Object Storage path of the
// form oci://<bucket>@<namespace>/<key>.
func IsOCIObjectStorageURI(uri string) bool {
	if !strings.HasPrefix(uri, OCIScheme) {
		return false
	}
	authority := uri[len(OCIScheme):]
	if i := strings.IndexAny(authority, "/?#"); i >= 0 {
		authority = authority[:i]
	}
	return strings.Contains(authority, ociNamespaceSeparator)
}

// IsModelcarURI reports whether uri is an oci:// container image reference
// (a Modelcar), as opposed to an OCI Object Storage path.
func IsModelcarURI(uri string) bool {
	return strings.HasPrefix(uri, OCIScheme) && !IsOCIObjectStorageURI(uri)
}

// parseOCIAuthority splits the "<bucket>@<namespace>" authority of an OCI
// Object Storage path into its parts.
func parseOCIAuthority(authority string) (bucket, namespace string, err error) {
	parts := strings.Split(authority, ociNamespaceSeparator)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf(
			"OCI Object Storage paths must be of the form oci://<bucket>@<namespace>/<prefix>, got %q",
			OCIScheme+authority,
		)
	}
	return parts[0], parts[1], nil
}

// resolveOCIEndpoint determines the S3-compatible endpoint and the signing
// region for an OCI Object Storage bucket. The endpoint is derived from the
// namespace and region unless an explicit endpoint is configured, in which
// case the region is taken from the endpoint host when not configured.
func resolveOCIEndpoint(namespace, endpoint, region string) (string, string, error) {
	if region == "" {
		region = os.Getenv(OCIRegionEnv)
	}
	if endpoint == "" {
		if region == "" {
			return "", "", fmt.Errorf(
				"OCI Object Storage requires a region: set providers.oci.default.region in the kfp-launcher config, "+
					"a ?region= query parameter on the pipeline root, or the %s environment variable",
				OCIRegionEnv,
			)
		}
		return OCICompatEndpoint(namespace, region), region, nil
	}
	if region == "" {
		region = OCIRegionFromEndpoint(endpoint)
		if region == "" {
			return "", "", fmt.Errorf(
				"OCI Object Storage endpoint %q is not a <namespace>.compat.objectstorage.<region>.oraclecloud.com host; "+
					"set the region explicitly",
				endpoint,
			)
		}
	}
	return endpoint, region, nil
}

// openOCIBucket opens an OCI Object Storage bucket through the S3-compatible
// API. The S3 client is always built explicitly (never through gocloud's URL
// opener) so that the OCI-specific client adjustments apply. Client settings
// come from, in order of precedence: the query string on the pipeline root
// (environment credentials), the session info resolved from the kfp-launcher
// providers config, or environment credentials with the OCI_REGION fallback.
func openOCIBucket(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	namespace string,
	config *Config,
	sessionInfo *SessionInfo,
) (*blob.Bucket, error) {
	var params *S3Params
	var err error
	switch {
	case config.QueryString != "":
		params, err = ociParamsFromQuery(config)
	case sessionInfo != nil:
		params, err = StructuredS3Params(sessionInfo.Params)
	default:
		params = &S3Params{FromEnv: true, ForcePathStyle: true}
	}
	if err != nil {
		return nil, err
	}
	s3Client, err := createOCIBucketSession(ctx, namespace, config, params, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI Object Storage session for bucket %s: %w", config.BucketName, err)
	}
	openedBucket, err := s3blob.OpenBucketV2(ctx, s3Client, config.BucketName, nil)
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(openedBucket, config.Prefix), nil
}

func createOCIBucketSession(
	ctx context.Context,
	namespace string,
	config *Config,
	params *S3Params,
	client kubernetes.Interface,
) (*s3.Client, error) {
	endpoint, region, err := resolveOCIEndpoint(config.Namespace, params.Endpoint, params.Region)
	if err != nil {
		return nil, err
	}
	params.Endpoint = endpoint
	params.Region = region

	var creds *credentials.StaticCredentialsProvider
	if !params.FromEnv {
		creds, err = getS3BucketCredential(ctx, client, namespace, params.SecretName, params.SecretKeyKey, params.AccessKeyKey)
		if err != nil {
			return nil, err
		}
	}
	return newS3Client(ctx, params, creds, withOCICompatibility)
}

// withOCICompatibility adjusts an S3 client for OCI's S3-compatible API. Both
// adjustments were established against a live OCI endpoint; other S3 clients
// (for example botocore) behave this way out of the box.
func withOCICompatibility(o *s3.Options) {
	o.APIOptions = append(o.APIOptions, removeSDKTelemetryHeaders, disableChecksumTrailers)
}

// removeSDKTelemetryHeaders drops the Amz-Sdk-Invocation-Id and Amz-Sdk-Request
// headers. aws-sdk-go-v2 adds and signs them by default, and OCI fails SigV4
// verification (SignatureDoesNotMatch) whenever the signature covers them.
func removeSDKTelemetryHeaders(stack *middleware.Stack) error {
	// Both middlewares only add telemetry headers; tolerate SDK versions that
	// no longer register them.
	_, _ = stack.Build.Remove("ClientRequestID")
	_, _ = stack.Finalize.Remove("RetryMetricsHeader")
	return nil
}

// disableChecksumTrailers clears the request checksum algorithm on uploads.
//
// As soon as a checksum algorithm is set on an HTTPS upload, the SDK switches
// to a trailing checksum with aws-chunked payload encoding, which OCI rejects
// with "501 NotImplemented: AWS chunked encoding not supported". gocloud's
// writer always requests CRC32, so the algorithm is cleared here; together with
// RequestChecksumCalculationWhenRequired on the client this leaves the payload
// unsigned (UNSIGNED-PAYLOAD) over TLS, exactly like botocore does.
func disableChecksumTrailers(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc(
		"OCIDisableChecksumTrailers",
		func(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
			middleware.InitializeOutput, middleware.Metadata, error,
		) {
			switch input := in.Parameters.(type) {
			case *s3.PutObjectInput:
				input.ChecksumAlgorithm = ""
			case *s3.CreateMultipartUploadInput:
				input.ChecksumAlgorithm = ""
				input.ChecksumType = ""
			case *s3.UploadPartInput:
				input.ChecksumAlgorithm = ""
			}
			return next.HandleInitialize(ctx, in)
		},
	), middleware.Before)
}

// ociParamsFromQuery builds the client parameters for an OCI Object Storage
// pipeline root that carries its settings as query parameters, for example
// oci://<bucket>@<namespace>/<prefix>?region=us-ashburn-1. The supported
// parameters mirror gocloud's s3:// URL opener: region, endpoint and
// use_path_style (or its legacy spelling s3ForcePathStyle). Credentials always
// come from the environment in this mode.
func ociParamsFromQuery(config *Config) (*S3Params, error) {
	q, err := url.ParseQuery(strings.TrimPrefix(config.QueryString, "?"))
	if err != nil {
		return nil, fmt.Errorf("invalid query string on OCI Object Storage path: %w", err)
	}
	params := &S3Params{FromEnv: true, ForcePathStyle: true}
	for key, values := range q {
		value := values[0]
		switch key {
		case "region":
			params.Region = value
		case "endpoint":
			params.Endpoint = value
		case "use_path_style", "s3ForcePathStyle":
			pathStyle, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("invalid value for %q on OCI Object Storage path: %w", key, err)
			}
			params.ForcePathStyle = pathStyle
		default:
			return nil, fmt.Errorf(
				"unsupported query parameter %q on OCI Object Storage path; supported parameters are region, endpoint and use_path_style",
				key,
			)
		}
	}
	return params, nil
}
