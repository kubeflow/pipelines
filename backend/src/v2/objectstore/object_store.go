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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/golang/glog"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"golang.org/x/oauth2/google"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func OpenBucket(ctx context.Context, k8sClient kubernetes.Interface, namespace string, config *Config) (bucket *blob.Bucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("Failed to open bucket %q: %w", config.BucketName, err)
		}
	}()
	if config.SessionInfo != nil {
		if config.SessionInfo.Provider == "minio" || config.SessionInfo.Provider == "s3" {
			sess, err1 := createS3BucketSession(ctx, namespace, config.SessionInfo, k8sClient)
			if err1 != nil {
				return nil, fmt.Errorf("Failed to retrieve credentials for bucket %s: %w", config.BucketName, err1)
			}
			if sess != nil {
				openedBucket, err2 := s3blob.OpenBucket(ctx, sess, config.BucketName, nil)
				if err2 != nil {
					return nil, err2
				}
				// Directly calling s3blob.OpenBucket does not allow overriding prefix via bucketConfig.BucketURL().
				// Therefore, we need to explicitly configure the prefixed bucket.
				return blob.PrefixedBucket(openedBucket, config.Prefix), nil
			}
		} else if config.SessionInfo.Provider == "gs" {
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
	files, err := ioutil.ReadDir(localPath)
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

func createS3BucketSession(ctx context.Context, namespace string, sessionInfo *SessionInfo, client kubernetes.Interface) (*session.Session, error) {
	if sessionInfo == nil {
		return nil, nil
	}
	config := &aws.Config{}
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
	config.Credentials = creds
	config.Region = aws.String(params.Region)
	config.DisableSSL = aws.Bool(params.DisableSSL)
	config.S3ForcePathStyle = aws.Bool(params.ForcePathStyle)

	// AWS Specific:
	// Path-style S3 endpoints, which are commonly used, may fall into either of two subdomains:
	// 1) [https://]s3.amazonaws.com
	// 2) s3.<AWS Region>.amazonaws.com
	// for (1) the endpoint is not required, thus we skip it, otherwise the writer will fail to close due to region mismatch.
	// https://aws.amazon.com/blogs/infrastructure-and-automation/best-practices-for-using-amazon-s3-endpoints-in-aws-cloudformation-templates/
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
	awsEndpoint, _ := regexp.MatchString(`^(https://)?s3.amazonaws.com`, strings.ToLower(params.Endpoint))
	if !awsEndpoint {
		config.Endpoint = aws.String(params.Endpoint)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create object store session, %v", err)
	}
	return sess, nil
}

func getS3BucketCredential(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	secretName string,
	bucketSecretKeyKey string,
	bucketAccessKeyKey string,
) (cred *credentials.Credentials, err error) {
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
		cred = credentials.NewStaticCredentials(accessKey, secretKey, "")
		return cred, err
	}
	return nil, fmt.Errorf("could not find specified keys '%s' or '%s'", bucketAccessKeyKey, bucketSecretKeyKey)
}
