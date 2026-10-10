// Copyright 2021 The Kubeflow Authors
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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func OpenBucket(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	namespace string,
	config *Config,
	sessionInfo *SessionInfo,
) (bucket *blob.Bucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("Failed to open bucket %q: %w", config.BucketName, err)
		}
	}()
	if sessionInfo != nil {
		switch sessionInfo.Provider {
		case "minio", "s3":
			if config.QueryString == "" {
				s3Client, err1 := createS3BucketSession(ctx, namespace, sessionInfo, k8sClient)
				if err1 != nil {
					return nil, fmt.Errorf("failed to retrieve credentials for bucket %s: %w", config.BucketName, err1)
				}
				if s3Client != nil {
					// Use s3blob.OpenBucketV2 with the configured S3 client to leverage retry logic.
					openedBucket, err2 := s3blob.OpenBucketV2(ctx, s3Client, config.BucketName, nil)
					if err2 != nil {
						return nil, err2
					}
					// Directly calling s3blob.OpenBucketV2 does not allow overriding prefix via bucketConfig.BucketURL().
					// Therefore, we need to explicitly configure the prefixed bucket.
					return blob.PrefixedBucket(openedBucket, config.Prefix), nil
				}
			}
		case "gs":
			client, err1 := getGCSTokenClient(ctx, namespace, sessionInfo, k8sClient)
			if err1 != nil {
				return nil, err1
			}
			if client != nil {
				openedBucket, err2 := gcsblob.OpenBucket(ctx, client, config.BucketName, nil)
				if err2 != nil {
					return openedBucket, err2
				}
				return blob.PrefixedBucket(openedBucket, config.Prefix), nil
			}
		}
	}

	bucketURL := normalizeBucketURLForBlobOpen(config.bucketURL())

	// When no session info is provided for a plain s3:// or minio:// URL,
	// build the S3 client directly so checksum options are applied.
	useExplicitS3Client := strings.HasPrefix(bucketURL, "minio://") ||
		(config.QueryString == "" && strings.HasPrefix(bucketURL, "s3://"))
	if useExplicitS3Client {
		s3Client, err1 := newS3Client(ctx, nil, nil)
		if err1 != nil {
			return nil, err1
		}
		openedBucket, err2 := s3blob.OpenBucketV2(ctx, s3Client, config.BucketName, nil)
		if err2 != nil {
			return nil, err2
		}
		return blob.PrefixedBucket(openedBucket, config.Prefix), nil
	}

	// Keep Go CDK's query parsing, encryption and prefix handling on URL paths.
	if _, overridden := util.HTTPProxyFrom(ctx); overridden && config.Scheme == "gs://" {
		return openGCSURL(ctx, bucketURL)
	}
	bucket, err = blob.OpenBucket(ctx, bucketURL)
	if err == nil {
		if resolver, overridden := util.HTTPProxyFrom(ctx); overridden {
			var s3Client *s3.Client
			if bucket.As(&s3Client) {
				// This URL opener owns a fresh client. Retain its resolved CA trust.
				options := s3Client.Options()
				client, ok := options.HTTPClient.(*awshttp.BuildableClient)
				if !ok {
					_ = bucket.Close()
					return nil, fmt.Errorf("S3 URL client cannot preserve custom CA trust; configure a structured bucket provider")
				}
				options.HTTPClient = client.WithTransportOptions(func(transport *http.Transport) { transport.Proxy = resolver })
				credentialConfig, credentialErr := s3URLCredentialConfig(ctx, bucketURL, options)
				if credentialErr != nil {
					_ = bucket.Close()
					return nil, credentialErr
				}
				options.Credentials = credentialConfig.Credentials
				options.HTTPClient = credentialConfig.HTTPClient
				*s3Client = *s3.New(options)
			}
		}
	}
	return bucket, err
}

// s3URLCredentialConfig rebuilds the credential chain and CA-aware HTTP client
// together; replacing only S3's transport leaves STS/metadata clients behind.
func s3URLCredentialConfig(ctx context.Context, bucketURL string, options s3.Options) (aws.Config, error) {
	parsed, err := url.Parse(bucketURL)
	if err != nil {
		return aws.Config{}, err
	}
	query := parsed.Query()
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithHTTPClient(options.HTTPClient),
		awsconfig.WithRegion(options.Region),
	}
	if profile := query.Get("profile"); profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(profile))
	}
	if anonymous, _ := strconv.ParseBool(query.Get("anonymous")); anonymous {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}))
	}
	if dualstack, _ := strconv.ParseBool(query.Get("dualstack")); dualstack {
		loadOptions = append(loadOptions, awsconfig.WithUseDualStackEndpoint(aws.DualStackEndpointStateEnabled))
	}
	if fips, _ := strconv.ParseBool(query.Get("fips")); fips {
		loadOptions = append(loadOptions, awsconfig.WithUseFIPSEndpoint(aws.FIPSEndpointStateEnabled))
	}
	if options.Retryer != nil {
		loadOptions = append(loadOptions, awsconfig.WithRetryer(func() aws.Retryer { return options.Retryer }))
	}
	config, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("configure S3 URL credentials with invocation proxy: %w", err)
	}
	if role := query.Get("role"); role != "" {
		config.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(config), role))
	}
	return config, nil
}

// invocationTransport clones rather than changes the process-wide HTTP transport.
func invocationTransport(ctx context.Context) *http.Transport {
	resolver, overridden := util.HTTPProxyFrom(ctx)
	if !overridden {
		return nil
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = resolver
	return transport
}

func openGCSURL(ctx context.Context, bucketURL string) (*blob.Bucket, error) {
	parsed, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}
	transport := invocationTransport(ctx)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: transport})
	query := parsed.Query()
	anonymous := query.Get("access_id") == "-" || os.Getenv("STORAGE_EMULATOR_HOST") != ""
	if value := query.Get("anonymous"); value != "" {
		flag, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		anonymous = anonymous || flag
	}
	var client *gcp.HTTPClient
	var options gcsblob.Options
	universeDomain := query.Get("universe_domain")
	if anonymous {
		client = gcp.NewAnonymousHTTPClient(transport)
	} else {
		creds, err := gcp.DefaultCredentialsWithParams(ctx, google.CredentialsParams{UniverseDomain: query.Get("universe_domain")})
		if err != nil {
			return nil, err
		}
		client, err = gcp.NewHTTPClient(transport, gcp.CredentialsTokenSource(creds))
		if err != nil {
			return nil, err
		}
		universeDomain, err = creds.GetUniverseDomain()
		if err != nil {
			return nil, fmt.Errorf("resolve GCS credential universe domain: %w", err)
		}
		var signing struct {
			Email string `json:"client_email"`
			Key   string `json:"private_key"`
		}
		if err := json.Unmarshal(creds.JSON, &signing); err == nil {
			options.GoogleAccessID, options.PrivateKey = signing.Email, []byte(signing.Key)
		}
	}
	if universeDomain != "" {
		options.ClientOptions = append(options.ClientOptions, option.WithUniverseDomain(universeDomain))
	}
	// The opener may replace Client for anonymous=true. Its final client option
	// preserves our invocation transport in that case as well.
	options.ClientOptions = append(options.ClientOptions, option.WithHTTPClient(&client.Client))
	mux := new(blob.URLMux)
	mux.RegisterBucket("gs", &gcsblob.URLOpener{Client: client, Options: options})
	return mux.OpenBucket(ctx, bucketURL)
}

func normalizeBucketURLForBlobOpen(bucketURL string) string {
	// Go CDK uses the S3 driver for MinIO-compatible bucket URLs in the fallback
	// blob.OpenBucket path, so normalize minio:// URLs before opening them.
	if strings.HasPrefix(bucketURL, "minio://") {
		return strings.Replace(bucketURL, "minio://", "s3://", 1)
	}
	return bucketURL
}

func UploadBlob(ctx context.Context, bucket *blob.Bucket, localPath, blobPath string) error {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("unable to stat local filepath %q: %w", localPath, err)
	}

	if !fileInfo.IsDir() {
		return uploadFile(ctx, bucket, localPath, blobPath)
	}

	// localPath is a directory.
	files, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("unable to list local directory %q: %w", localPath, err)
	}

	for _, f := range files {
		if f.IsDir() {
			err = UploadBlob(ctx, bucket, filepath.Join(localPath, f.Name()), blobPath+"/"+f.Name())
			if err != nil {
				return err
			}
		} else {
			blobFilePath := filepath.Join(blobPath, filepath.Base(f.Name()))
			localFilePath := filepath.Join(localPath, f.Name())
			if err := uploadFile(ctx, bucket, localFilePath, blobFilePath); err != nil {
				return err
			}
		}

	}

	return nil
}

func DownloadBlob(ctx context.Context, bucket *blob.Bucket, localDir, blobDir string) error {
	iter := bucket.List(&blob.ListOptions{Prefix: blobDir})
	normalizedBlobDir := strings.TrimSuffix(blobDir, "/")
	var exactPrefixObject *blob.ListObject
	hasNestedObjects := false

	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to list objects in remote storage %q: %w", blobDir, err)
		}
		if obj.IsDir {
			// Object stores list all files with the same prefix,
			// there is no need to recursively list each folder.
			continue
		}

		normalizedKey := strings.TrimSuffix(obj.Key, "/")
		if normalizedKey != normalizedBlobDir && !strings.HasPrefix(normalizedKey, normalizedBlobDir+"/") {
			continue
		}
		// Hold the exact-prefix object until listing finishes. Nested objects
		// prove it is only a directory marker and should be discarded; otherwise
		// download it as a single-file artifact.
		if normalizedKey == normalizedBlobDir {
			exactPrefixObject = obj
			continue
		}

		hasNestedObjects = true
		if err := downloadListedObject(ctx, bucket, localDir, normalizedBlobDir, obj); err != nil {
			return err
		}
	}

	if exactPrefixObject != nil && !hasNestedObjects {
		if err := downloadListedObject(ctx, bucket, localDir, normalizedBlobDir, exactPrefixObject); err != nil {
			return err
		}
	}
	return nil
}

func downloadListedObject(
	ctx context.Context,
	bucket *blob.Bucket,
	localDir, normalizedBlobDir string,
	obj *blob.ListObject,
) error {
	normalizedKey := strings.TrimSuffix(obj.Key, "/")
	relativePath, err := filepath.Rel(normalizedBlobDir, normalizedKey)
	if err != nil {
		return fmt.Errorf("unexpected object key %q when listing %q: %w", obj.Key, normalizedBlobDir, err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, "../") || filepath.IsAbs(relativePath) {
		return fmt.Errorf("unexpected object key %q when listing %q", obj.Key, normalizedBlobDir)
	}
	return downloadFile(ctx, bucket, obj.Key, filepath.Join(localDir, relativePath))
}

func uploadFile(ctx context.Context, bucket *blob.Bucket, localFilePath, blobFilePath string) error {
	errorF := func(err error) error {
		return fmt.Errorf("uploadFile(): unable to complete copying %q to remote storage %q: %w", localFilePath, blobFilePath, err)
	}

	w, err := bucket.NewWriter(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open writer for bucket: %w", err))
	}

	r, err := os.Open(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for reading: %w", localFilePath, err))
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	if err = w.Close(); err != nil {
		return errorF(fmt.Errorf("failed to close Writer for bucket: %w", err))
	}

	glog.Infof("uploadFile(localFilePath=%q, blobFilePath=%q)", localFilePath, blobFilePath)
	return nil
}

func downloadFile(ctx context.Context, bucket *blob.Bucket, blobFilePath, localFilePath string) (err error) {
	errorF := func(err error) error {
		return fmt.Errorf("downloadFile(): unable to complete copying %q to local storage %q: %w", blobFilePath, localFilePath, err)
	}

	r, err := bucket.NewReader(ctx, blobFilePath, nil)
	if err != nil {
		return errorF(fmt.Errorf("unable to open reader for bucket: %w", err))
	}
	defer r.Close()

	localDir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return errorF(fmt.Errorf("failed to create local directory %q: %w", localDir, err))
	}

	w, err := os.Create(localFilePath)
	if err != nil {
		return errorF(fmt.Errorf("unable to open local file %q for writing: %w", localFilePath, err))
	}
	defer func() {
		errClose := w.Close()
		if err == nil && errClose != nil {
			// override named return value "err" when there's a close error
			err = errorF(errClose)
		}
	}()

	if _, err = io.Copy(w, r); err != nil {
		return errorF(fmt.Errorf("unable to complete copying: %w", err))
	}

	return nil
}

func getGCSTokenClient(ctx context.Context, namespace string, sessionInfo *SessionInfo, clientSet kubernetes.Interface) (client *gcp.HTTPClient, err error) {
	params, err := StructuredGCSParams(sessionInfo.Params)
	if err != nil {
		return nil, err
	}
	transport := invocationTransport(ctx)
	if transport != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: transport})
	}
	if params.FromEnv {
		// Use the URL opener so default credentials and query options stay aligned.
		return nil, nil
	}
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, params.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	tokenJSON, ok := secret.Data[params.TokenKey]
	if !ok || len(tokenJSON) == 0 {
		return nil, fmt.Errorf("key '%s' not found or is empty", params.TokenKey)
	}
	creds, err := google.CredentialsFromJSON(ctx, tokenJSON, "https://www.googleapis.com/auth/devstorage.read_write") //nolint:staticcheck // SA1019: CredentialsFromJSON still required for secret-mounted service account JSON tokens
	if err != nil {
		return nil, err
	}
	var base http.RoundTripper = gcp.DefaultTransport()
	if transport != nil {
		base = transport
	}
	client, err = gcp.NewHTTPClient(base, gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func createS3BucketSession(ctx context.Context, namespace string, sessionInfo *SessionInfo, client kubernetes.Interface) (*s3.Client, error) {
	if sessionInfo == nil {
		return nil, nil
	}
	params, err := StructuredS3Params(sessionInfo.Params)
	if err != nil {
		return nil, err
	}
	if params.FromEnv && !HasStructuredS3Settings(sessionInfo.Params) {
		return nil, nil
	}
	var creds *credentials.StaticCredentialsProvider
	if !params.FromEnv {
		creds, err = getS3BucketCredential(ctx, client, namespace, params.SecretName, params.SecretKeyKey, params.AccessKeyKey)
		if err != nil {
			return nil, err
		}
	}
	return newS3Client(ctx, params, creds)
}

func newS3Client(ctx context.Context, params *S3Params, creds *credentials.StaticCredentialsProvider) (*s3.Client, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	}
	if resolver, overridden := util.HTTPProxyFrom(ctx); overridden {
		client := awshttp.NewBuildableClient().WithTransportOptions(func(transport *http.Transport) { transport.Proxy = resolver })
		loadOptions = append(loadOptions, awsconfig.WithHTTPClient(client))
	}
	if params != nil {
		if params.MaxRetries > 0 {
			loadOptions = append(loadOptions, awsconfig.WithRetryer(func() aws.Retryer {
				// Use standard retry logic with exponential backoff for transient S3 connection failures.
				// The standard retryer implements exponential backoff with jitter, starting with a base delay
				// and doubling the wait time between retries up to a maximum, helping to avoid thundering herd problems.
				return retry.AddWithMaxAttempts(retry.NewStandard(), params.MaxRetries)
			}))
		}
		if params.Region != "" {
			loadOptions = append(loadOptions, awsconfig.WithRegion(params.Region))
		}
	}
	if creds != nil {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(*creds))
	}
	s3Config, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}
	// AWS Specific:
	// Path-style S3 endpoints, which are commonly used, may fall into either of two subdomains:
	// 1) [https://]s3.amazonaws.com
	// 2) s3.<AWS Region>.amazonaws.com
	// for (1) the endpoint is not required, thus we skip it, otherwise the writer will fail to close due to region mismatch.
	// https://aws.amazon.com/blogs/infrastructure-and-automation/best-practices-for-using-amazon-s3-endpoints-in-aws-cloudformation-templates/
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
	s3Options := func(o *s3.Options) {
		if params == nil {
			return
		}
		awsEndpoint, _ := regexp.MatchString(
			`^(https://)?s3[.]amazonaws[.]com(?::[0-9]+)?(?:/|$)`,
			strings.ToLower(params.Endpoint),
		)
		o.UsePathStyle = *aws.Bool(params.ForcePathStyle)
		o.EndpointOptions.DisableHTTPS = *aws.Bool(params.DisableSSL)
		if !awsEndpoint && params.Endpoint != "" {
			// AWS SDK v2 requires BaseEndpoint to be a valid URI with scheme
			endpoint := params.Endpoint
			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				if params.DisableSSL {
					endpoint = "http://" + endpoint
				} else {
					endpoint = "https://" + endpoint
				}
			}
			o.BaseEndpoint = aws.String(endpoint)
		}
	}
	s3Client := s3.NewFromConfig(s3Config, s3Options)
	if s3Client == nil {
		return nil, fmt.Errorf("failed to create object store session, %v", err)
	}
	return s3Client, nil
}

func getS3BucketCredential(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	secretName string,
	bucketSecretKeyKey string,
	bucketAccessKeyKey string,
) (cred *credentials.StaticCredentialsProvider, err error) {
	defer func() {
		if err != nil {
			// wrap error before returning
			err = fmt.Errorf("failed to get Bucket credentials from secret name=%q namespace=%q: %w", secretName, namespace, err)
		}
	}()
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(
		ctx,
		secretName,
		metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// The k8s secret "Key" for "SecretKey" and "AccessKey"
	accessKey := string(secret.Data[bucketAccessKeyKey])
	secretKey := string(secret.Data[bucketSecretKeyKey])

	if accessKey != "" && secretKey != "" {
		s3Creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		return &s3Creds, err
	}
	// Name which credential role is missing without echoing the configured
	// secret data key names.
	missingValues := make([]string, 0, 2)
	if accessKey == "" {
		missingValues = append(missingValues, "access key")
	}
	if secretKey == "" {
		missingValues = append(missingValues, "secret key")
	}
	return nil, fmt.Errorf("bucket credential secret has no value for: %s", strings.Join(missingValues, ", "))
}
