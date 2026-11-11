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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
			return nil, errors.New("invalid storage endpoint")
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

func (s *RemoteStorage) iam(ctx context.Context, pair Credentials, action string, fields url.Values) (int, error) {
	if fields == nil {
		fields = url.Values{}
	}
	fields.Set("Action", action)
	fields.Set("Version", "2010-05-08")
	body := fields.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.iamEndpoint, strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	hash := sha256.Sum256([]byte(body))
	if err = s.signer.SignHTTP(ctx, aws.Credentials{AccessKeyID: pair.AccessKey, SecretAccessKey: pair.SecretKey}, req, hex.EncodeToString(hash[:]), "iam", "us-east-1", time.Now()); err != nil {
		return 0, err
	}
	response, err := s.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	// Do not retain or log IAM response bodies, which can contain credentials.
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))
	return response.StatusCode, nil
}

func (s *RemoteStorage) VerifyRegistration(ctx context.Context) error {
	_, err := s.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String("mlpipeline")})
	return err
}

func (s *RemoteStorage) Verify(ctx context.Context) error {
	if err := s.VerifyRegistration(ctx); err != nil {
		return err
	}
	status, err := s.iam(ctx, s.pair, "GetUser", url.Values{"UserName": {"kubeflow-admin"}})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("IAM verification returned HTTP %d", status)
	}
	return nil
}

func rejected(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func (s *RemoteStorage) DefaultAccepted(ctx context.Context) (bool, error) {
	_, err := s.oldS3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String("mlpipeline")})
	if err == nil {
		return true, nil
	}
	var response interface{ HTTPStatusCode() int }
	if !errors.As(err, &response) || !rejected(response.HTTPStatusCode()) {
		return false, err
	}
	status, err := s.iam(ctx, Credentials{DefaultAccessKey, DefaultSecretKey}, "ListUsers", nil)
	if err != nil {
		return false, err
	}
	if rejected(status) {
		return false, nil
	}
	if status == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("IAM negative probe returned HTTP %d", status)
}

func (s *RemoteStorage) RevokeDefault(ctx context.Context) error {
	accepted, err := s.DefaultAccepted(ctx)
	if err != nil || !accepted {
		return err
	}
	if s.pair.AccessKey == DefaultAccessKey {
		return errors.New("default remains accepted under an operator-owned access-key ID")
	}
	status, err := s.iam(ctx, s.pair, "DeleteAccessKey", url.Values{"UserName": {"kubeflow-admin"}, "AccessKeyId": {DefaultAccessKey}})
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
