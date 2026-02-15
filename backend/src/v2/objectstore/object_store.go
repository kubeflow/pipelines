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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/glog"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"golang.org/x/oauth2/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ErrHuggingFaceNoBucket is returned when the HuggingFace provider is used
// but it does not implement the blob.Bucket interface. Callers of OpenBucket
// should check for this sentinel error and handle it explicitly.
var ErrHuggingFaceNoBucket = errors.New("huggingface provider does not use the bucket interface")

func OpenBucket(ctx context.Context, k8sClient kubernetes.Interface, namespace string, config *Config) (bucket *blob.Bucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("Failed to open bucket %q: %w", config.BucketName, err)
		}
	}()
	if config.SessionInfo != nil {
		switch config.SessionInfo.Provider {
		case "minio", "s3":
			s3Client, err1 := createS3BucketSession(ctx, namespace, config.SessionInfo, k8sClient)
			if err1 != nil {
				return nil, fmt.Errorf("Failed to retrieve credentials for bucket %s: %w", config.BucketName, err1)
			}
			if s3Client != nil {
				// Use s3blob.OpenBucketV2 with the configured S3 client to leverage retry logic
				openedBucket, err2 := s3blob.OpenBucketV2(ctx, s3Client, config.BucketName, nil)
				if err2 != nil {
					return nil, err2
				}
				// Directly calling s3blob.OpenBucketV2 does not allow overriding prefix via bucketConfig.BucketURL().
				// Therefore, we need to explicitly configure the prefixed bucket.
				return blob.PrefixedBucket(openedBucket, config.Prefix), nil
			}
		case "gs":
			client, err1 := getGCSTokenClient(ctx, namespace, config.SessionInfo, k8sClient)
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
		case "huggingface":
			token, err1 := getHuggingFaceToken(ctx, namespace, config.SessionInfo, k8sClient)
			if err1 != nil {
				return nil, err1
			}
			if token == "" {
				glog.Warning("Hugging Face provider selected but received an empty token from secret; downloads may fail for private repos")
			} else {
				glog.V(1).Info("Hugging Face token retrieved; caller must propagate it explicitly into the downloader's process environment (e.g., set HF_TOKEN in the command's Env)")
			}
			return nil, ErrHuggingFaceNoBucket
		}
	}

	bucketURL := config.bucketURL()
	// Since query parameters are only supported for s3:// paths
	// if we detect minio scheme in pipeline root, replace it with s3:// scheme
	// ref: https://gocloud.dev/howto/blob/#s3-compatible
	if len(config.QueryString) > 0 && strings.HasPrefix(bucketURL, "minio://") {
		bucketURL = strings.Replace(bucketURL, "minio://", "s3://", 1)
	}

	// When no provider config is provided, or "FromEnv" is specified, use default credentials from the environment
	return blob.OpenBucket(ctx, bucketURL)
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
	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to list objects in remote storage %q: %w", blobDir, err)
		}
		if obj.IsDir {
			// TODO: is this branch possible?

			// Object stores list all files with the same prefix,
			// there is no need to recursively list each folder.
			continue
		} else {
			relativePath, err := filepath.Rel(blobDir, obj.Key)
			if err != nil {
				return fmt.Errorf("unexpected object key %q when listing %q: %w", obj.Key, blobDir, err)
			}
			if err := downloadFile(ctx, bucket, obj.Key, filepath.Join(localDir, relativePath)); err != nil {
				return err
			}
		}
	}
	return nil
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
	if params.FromEnv {
		return nil, nil
	}
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, params.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	tokenJson, ok := secret.Data[params.TokenKey]
	if !ok || len(tokenJson) == 0 {
		return nil, fmt.Errorf("key '%s' not found or is empty", params.TokenKey)
	}
	creds, err := google.CredentialsFromJSON(ctx, tokenJson, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return nil, err
	}
	client, err = gcp.NewHTTPClient(gcp.DefaultTransport(), gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getHuggingFaceToken(ctx context.Context, namespace string, sessionInfo *SessionInfo, clientSet kubernetes.Interface) (token string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get HuggingFace token from secret: %w", err)
		}
	}()
	params, err := StructuredHuggingFaceParams(sessionInfo.Params)
	if err != nil {
		return "", err
	}
	if params.FromEnv {
		return "", nil
	}
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, params.SecretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	tokenBytes, ok := secret.Data[params.TokenKey]
	if !ok || len(tokenBytes) == 0 {
		return "", fmt.Errorf("key '%s' not found or is empty in secret", params.TokenKey)
	}
	return string(tokenBytes), nil
}

// GetHuggingFaceTokenFromK8sSecret retrieves the Hugging Face token from a Kubernetes Secret
// as specified by the provided SessionInfo. It is a thin wrapper around the internal
// getHuggingFaceToken helper and requires a Kubernetes client and namespace because
// it reads Secret data from the cluster.
func GetHuggingFaceTokenFromK8sSecret(ctx context.Context, k8sClient kubernetes.Interface, namespace string, sessionInfo *SessionInfo) (string, error) {
	if sessionInfo == nil {
		return "", nil
	}
	return getHuggingFaceToken(ctx, namespace, sessionInfo, k8sClient)
}

func createS3BucketSession(ctx context.Context, namespace string, sessionInfo *SessionInfo, client kubernetes.Interface) (*s3.Client, error) {
	if sessionInfo == nil {
		return nil, nil
	}
	params, err := StructuredS3Params(sessionInfo.Params)
	if err != nil {
		return nil, err
	}
	if params.FromEnv {
		return nil, nil
	}
	creds, err := getS3BucketCredential(ctx, client, namespace, params.SecretName, params.SecretKeyKey, params.AccessKeyKey)
	if err != nil {
		return nil, err
	}
	s3Config, err := config.LoadDefaultConfig(ctx,
		config.WithRetryer(func() aws.Retryer {
			// Use standard retry logic with exponential backoff for transient S3 connection failures.
			// The standard retryer implements exponential backoff with jitter, starting with a base delay
			// and doubling the wait time between retries up to a maximum, helping to avoid thundering herd problems.
			return retry.AddWithMaxAttempts(retry.NewStandard(), params.MaxRetries)
		}),
		config.WithCredentialsProvider(*creds),
		config.WithRegion(*aws.String(params.Region)),
	)
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
	awsEndpoint, _ := regexp.MatchString(`^(https://)?s3.amazonaws.com`, strings.ToLower(params.Endpoint))
	s3Options := func(o *s3.Options) {
		o.UsePathStyle = *aws.Bool(params.ForcePathStyle)
		o.EndpointOptions.DisableHTTPS = *aws.Bool(params.DisableSSL)
		if !awsEndpoint {
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
		return nil, fmt.Errorf("Failed to create object store session, %v", err)
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
			err = fmt.Errorf("Failed to get Bucket credentials from secret name=%q namespace=%q: %w", secretName, namespace, err)
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
	return nil, fmt.Errorf("could not find specified keys '%s' or '%s'", bucketAccessKeyKey, bucketSecretKeyKey)
}
