// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentialrotation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Client implements the reconciler's restricted namespace operations.
type Client struct {
	Clientset kubernetes.Interface
	Namespace string
}

func (c *Client) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	s, err := c.Clientset.CoreV1().Secrets(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return s, err
}

func (c *Client) PatchSecret(ctx context.Context, name, version string, data map[string][]byte, annotations map[string]string) (*corev1.Secret, error) {
	metadata := map[string]interface{}{"resourceVersion": version}
	if annotations != nil {
		metadata["annotations"] = annotations
	}
	patch := map[string]interface{}{"metadata": metadata}
	if data != nil {
		patch["data"] = data
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	return c.Clientset.CoreV1().Secrets(c.Namespace).Patch(ctx, name, types.MergePatchType, body, metav1.PatchOptions{})
}

func (c *Client) CreateSecret(ctx context.Context, s *corev1.Secret) (*corev1.Secret, error) {
	return c.Clientset.CoreV1().Secrets(c.Namespace).Create(ctx, s, metav1.CreateOptions{})
}

func (c *Client) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	d, err := c.Clientset.AppsV1().Deployments(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return d, err
}

func (c *Client) PatchDeploymentRevision(ctx context.Context, name, version, revision string) error {
	body, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{"resourceVersion": version},
		"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{RevisionAnnotation: revision}}}},
	})
	if err != nil {
		return err
	}
	_, err = c.Clientset.AppsV1().Deployments(c.Namespace).Patch(ctx, name, types.MergePatchType, body, metav1.PatchOptions{})
	return err
}

func (c *Client) ListPods(ctx context.Context, options metav1.ListOptions) (*corev1.PodList, error) {
	return c.Clientset.CoreV1().Pods(c.Namespace).List(ctx, options)
}

// RemoteStorage uses the existing S3 SDK and SigV4 signer. IAM's small Query API
// surface avoids an additional SDK dependency and AWS-only access-key length rules.
type RemoteStorage struct {
	pair        Credentials
	s3, oldS3   *s3.Client
	http        *http.Client
	signer      *v4.Signer
	iamEndpoint string
}

func NewRemoteStorage(pair Credentials, s3Endpoint, iamEndpoint string) (*RemoteStorage, error) {
	for _, endpoint := range []string{s3Endpoint, iamEndpoint} {
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, failure("invalid storage endpoint", nil)
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 4
	h := &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	newS3 := func(p Credentials) *s3.Client {
		cfg := aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider(p.AccessKey, p.SecretKey, ""), HTTPClient: h, RetryMaxAttempts: 2}
		return s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(s3Endpoint); o.UsePathStyle = true })
	}
	return &RemoteStorage{pair: pair, s3: newS3(pair), oldS3: newS3(Credentials{DefaultAccessKey, DefaultSecretKey}), http: h, signer: v4.NewSigner(), iamEndpoint: iamEndpoint}, nil
}

// iamResult deliberately decodes only non-secret fields needed for authentication
// and bounded credential-owner discovery. Never log the raw IAM response.
type iamResult struct {
	Code           string   `xml:"Code"`
	ErrorCode      string   `xml:"Error>Code"`
	Users          []string `xml:"ListUsersResult>Users>member>UserName"`
	UsersTruncated bool     `xml:"ListUsersResult>IsTruncated"`
	Keys           []string `xml:"ListAccessKeysResult>AccessKeyMetadata>member>AccessKeyId"`
	KeysTruncated  bool     `xml:"ListAccessKeysResult>IsTruncated"`
}

func (s *RemoteStorage) iam(ctx context.Context, pair Credentials, action string, fields url.Values) (int, iamResult, error) {
	var result iamResult
	if fields == nil {
		fields = url.Values{}
	}
	fields.Set("Action", action)
	fields.Set("Version", "2010-05-08")
	body := fields.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.iamEndpoint, strings.NewReader(body))
	if err != nil {
		return 0, result, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	hash := sha256.Sum256([]byte(body))
	if err = s.signer.SignHTTP(ctx, aws.Credentials{AccessKeyID: pair.AccessKey, SecretAccessKey: pair.SecretKey}, req, hex.EncodeToString(hash[:]), "iam", "us-east-1", time.Now()); err != nil {
		return 0, result, err
	}
	response, err := s.http.Do(req)
	if err != nil {
		return 0, result, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil {
		return 0, result, failure("cannot read IAM response", nil)
	}
	if len(data) > 64*1024 {
		return 0, result, failure("IAM inventory exceeds bounded response limit; inspect storage identities", nil)
	}
	if err := xml.Unmarshal(data, &result); err != nil {
		return 0, result, failure("cannot decode IAM response", nil)
	}
	return response.StatusCode, result, nil
}

func (s *RemoteStorage) VerifyRegistration(ctx context.Context) error {
	_, err := s.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String("mlpipeline")})
	return err
}

func (s *RemoteStorage) Verify(ctx context.Context) error {
	if err := s.VerifyRegistration(ctx); err != nil {
		return err
	}
	status, _, err := s.iam(ctx, s.pair, "GetUser", url.Values{"UserName": {"kubeflow-admin"}})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("IAM verification returned HTTP %d", status)
	}
	return nil
}

func authenticationRejected(code string) bool {
	return code == "InvalidAccessKeyId" || code == "InvalidAccessKeyID" || code == "SignatureDoesNotMatch" || code == "InvalidClientTokenId"
}

func (s *RemoteStorage) DefaultAccepted(ctx context.Context) (bool, error) {
	// HEAD discards the error body and cannot distinguish denied permissions from
	// a rejected signature. This read returns the explicit S3 authentication code.
	_, err := s.oldS3.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: aws.String("mlpipeline")})
	if err == nil {
		return true, nil
	}
	var apiError smithy.APIError
	s3Rejected := errors.As(err, &apiError) && authenticationRejected(apiError.ErrorCode())
	status, result, iamErr := s.iam(ctx, Credentials{DefaultAccessKey, DefaultSecretKey}, "ListUsers", nil)
	if iamErr != nil {
		return false, iamErr
	}
	if status == http.StatusOK {
		return true, nil
	}
	if s3Rejected && (status == http.StatusUnauthorized || status == http.StatusForbidden) && (authenticationRejected(result.Code) || authenticationRejected(result.ErrorCode)) {
		return false, nil
	}
	return false, failure("cannot prove default credential rejection; inspect ambiguous S3 or IAM authorization failure", nil)
}

func (s *RemoteStorage) defaultOwner(ctx context.Context) (string, error) {
	status, result, err := s.iam(ctx, s.pair, "ListUsers", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK || result.UsersTruncated || len(result.Users) == 0 || len(result.Users) > 1000 {
		return "", failure("cannot verify complete bounded IAM identity inventory", nil)
	}
	owner := ""
	seen := map[string]bool{}
	for _, user := range result.Users {
		if user == "" || seen[user] {
			return "", failure("invalid IAM identity inventory", nil)
		}
		seen[user] = true
		status, keys, err := s.iam(ctx, s.pair, "ListAccessKeys", url.Values{"UserName": {user}})
		if err != nil {
			return "", err
		}
		if status != http.StatusOK || keys.KeysTruncated {
			return "", failure("cannot verify complete IAM access-key inventory", nil)
		}
		for _, key := range keys.Keys {
			if key != DefaultAccessKey {
				continue
			}
			if owner != "" {
				// IAM does not expose secrets. Duplicate IDs may include an
				// operator-customized pair: never delete those by ID alone.
				return "", failure("multiple identities own the shipped access-key ID; remove only the exact shipped pair using operator credential inventory, then retry", nil)
			}
			owner = user
		}
	}
	if owner == "" {
		return "", failure("accepted default credential has no unique IAM owner; inspect storage identity caches", nil)
	}
	return owner, nil
}

func (s *RemoteStorage) RevokeDefault(ctx context.Context) error {
	accepted, err := s.DefaultAccepted(ctx)
	if err != nil || !accepted {
		return err
	}
	if s.pair.AccessKey == DefaultAccessKey {
		return failure("default remains accepted under an operator-owned access-key ID", nil)
	}
	// A stale S3 cache alone must not authorize deletion from IAM's newer
	// inventory. Require the shipped pair to authenticate to IAM as well.
	status, proof, err := s.iam(ctx, Credentials{DefaultAccessKey, DefaultSecretKey}, "ListUsers", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK || proof.Code != "" || proof.ErrorCode != "" {
		return failure("S3 and IAM disagree about the shipped credential; reconcile storage identity caches before retrying", nil)
	}
	owner, err := s.defaultOwner(ctx)
	if err != nil {
		return err
	}
	status, _, err = s.iam(ctx, s.pair, "DeleteAccessKey", url.Values{"UserName": {owner}, "AccessKeyId": {DefaultAccessKey}})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("IAM deletion returned HTTP %d", status)
	}
	accepted, err = s.DefaultAccepted(ctx)
	if err != nil {
		return err
	}
	if accepted {
		return &Waiting{Message: "Waiting for S3 and IAM to reject the default credential"}
	}
	return nil
}

func (s *RemoteStorage) Close() { s.http.CloseIdleConnections() }
