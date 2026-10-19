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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	gcaws "gocloud.dev/aws"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// openBucketURL uses a private mux so concurrent invocations never share proxy
// settings or the default GCS opener's cached credentials and HTTP client.
func openBucketURL(ctx context.Context, bucketURL string) (*blob.Bucket, error) {
	if _, overridden := util.HTTPProxyFrom(ctx); !overridden {
		return blob.OpenBucket(ctx, bucketURL)
	}
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}
	mux := new(blob.URLMux)
	switch u.Scheme {
	case s3blob.Scheme:
		mux.RegisterBucket(s3blob.Scheme, invocationS3URLOpener{})
	case gcsblob.Scheme:
		mux.RegisterBucket(gcsblob.Scheme, invocationGCSURLOpener{})
	default:
		return blob.OpenBucket(ctx, bucketURL)
	}
	return mux.OpenBucket(ctx, bucketURL)
}

type invocationS3URLOpener struct{}

func (invocationS3URLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	query := u.Query()
	var bucketOptions s3blob.Options
	if encryption := query.Get("ssetype"); encryption != "" {
		for _, candidate := range types.ServerSideEncryptionAes256.Values() {
			if strings.EqualFold(encryption, string(candidate)) {
				bucketOptions.EncryptionType = candidate
			}
		}
		if bucketOptions.EncryptionType == "" {
			return nil, fmt.Errorf("invalid ssetype %q: use AES256, aws:kms, or aws:kms:dsse", encryption)
		}
	}
	query.Del("ssetype")
	bucketOptions.KMSEncryptionID = query.Get("kmskeyid")
	query.Del("kmskeyid")
	var clientOptions s3.Options
	for _, setting := range []struct {
		name  string
		value *bool
	}{
		{"accelerate", &clientOptions.UseAccelerate},
		{"disable_https", &clientOptions.EndpointOptions.DisableHTTPS},
		{"use_path_style", &clientOptions.UsePathStyle},
		{"s3ForcePathStyle", &clientOptions.UsePathStyle},
	} {
		if value := query.Get(setting.name); value != "" {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: use a boolean value: %w", setting.name, err)
			}
			*setting.value = parsed
		}
		query.Del(setting.name)
	}

	// Construct STS only after installing the invocation HTTP client.
	role := query.Get("role")
	query.Del("role")
	if role != "" && !strings.HasPrefix(role, "arn:") {
		return nil, fmt.Errorf("invalid role %q: specify an IAM role ARN", role)
	}
	config, err := gcaws.V2ConfigFromURLParams(ctx, query)
	if err != nil {
		return nil, err
	}
	resolver, _ := util.HTTPProxyFrom(ctx)
	client := awshttp.NewBuildableClient().WithTransportOptions(func(t *http.Transport) { t.Proxy = resolver })
	// Reapply the URL loader options before resolving credentials, so token
	// providers and custom CA bundles also use the invocation HTTP client.
	var loadOptions *awsconfig.LoadOptions
	for _, source := range config.ConfigSources {
		if options, ok := source.(awsconfig.LoadOptions); ok {
			loadOptions = &options
			break
		}
	}
	if loadOptions == nil {
		return nil, fmt.Errorf("AWS URL configuration is missing loader options; verify Go CDK and AWS SDK compatibility")
	}
	config, err = awsconfig.LoadDefaultConfig(ctx, func(options *awsconfig.LoadOptions) error {
		*options = *loadOptions
		options.HTTPClient = client
		return nil
	})
	if err != nil {
		return nil, err
	}
	if role != "" {
		// The URL's endpoint= names the storage service, so STS must resolve
		// its own endpoint as the Go CDK opener does, while still using the
		// invocation HTTP client.
		stsConfig, err := awsconfig.LoadDefaultConfig(ctx, func(options *awsconfig.LoadOptions) error {
			*options = *loadOptions
			options.HTTPClient = client
			options.EndpointResolverWithOptions = nil
			return nil
		})
		if err != nil {
			return nil, err
		}
		config.Credentials = stscreds.NewAssumeRoleProvider(sts.NewFromConfig(stsConfig), role)
	}
	bucketOptions.RequestChecksumCalculation = config.RequestChecksumCalculation
	clientS3 := s3.NewFromConfig(config, func(options *s3.Options) {
		options.UseAccelerate = clientOptions.UseAccelerate
		options.UsePathStyle = clientOptions.UsePathStyle
		options.EndpointOptions.DisableHTTPS = clientOptions.EndpointOptions.DisableHTTPS
	})
	return s3blob.OpenBucketV2(ctx, clientS3, u.Host, &bucketOptions)
}

type invocationGCSURLOpener struct{}

func (invocationGCSURLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	transport := invocationTransport(ctx)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: transport})
	query := u.Query()
	anonymous, err := strconv.ParseBool(query.Get("anonymous"))
	if query.Get("anonymous") != "" && err != nil {
		return nil, fmt.Errorf("invalid anonymous: use a boolean value: %w", err)
	}
	var client *gcp.HTTPClient
	var options gcsblob.Options
	if anonymous || query.Get("access_id") == "-" || os.Getenv("STORAGE_EMULATOR_HOST") != "" {
		client = gcp.NewAnonymousHTTPClient(transport)
	} else {
		credentials, err := gcp.DefaultCredentialsWithParams(ctx, google.CredentialsParams{
			Scopes:         []string{"https://www.googleapis.com/auth/devstorage.read_write"},
			UniverseDomain: query.Get("universe_domain"),
		})
		if err != nil {
			return nil, err
		}
		client, err = gcp.NewHTTPClient(transport, credentials.TokenSource)
		if err != nil {
			return nil, err
		}
		universe, err := credentials.GetUniverseDomain()
		if err != nil {
			return nil, err
		}
		if universe != "" {
			options.ClientOptions = append(options.ClientOptions, option.WithUniverseDomain(universe))
		}
	}
	// Override even the anonymous/emulator clients created by the URL opener.
	options.ClientOptions = append(options.ClientOptions, option.WithHTTPClient(&client.Client))
	opener := gcsblob.URLOpener{Client: client, Options: options}
	return opener.OpenBucketURL(ctx, u)
}
