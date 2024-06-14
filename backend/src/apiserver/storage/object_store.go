// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"bytes"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	minio "github.com/minio/minio-go/v6"
	v1 "k8s.io/api/core/v1"
	"net/url"
	"path"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

const (
	multipartDefaultSize = -1
)

// Interface for object store.
type ObjectStoreInterface interface {
	AddFile(template []byte, filePath string) error
	DeleteFile(filePath string) error
	GetFile(filePath string) ([]byte, error)
	AddAsYamlFile(o interface{}, filePath string) error
	GetFromYamlFile(o interface{}, filePath string) error
	GetPipelineKey(pipelineId string) string
	GetSignedUrl(bucketConfig *objectstore.Config, secret *v1.Secret, expirySeconds time.Duration, artifactURI string) (string, error)
	GetObjectSize(bucketConfig *objectstore.Config, secret *v1.Secret, artifactURI string) (int64, error)
}

// Managing pipeline using Minio.
type MinioObjectStore struct {
	minioClient      MinioClientInterface
	bucketName       string
	baseFolder       string
	disableMultipart bool
}

// GetPipelineKey adds the configured base folder to pipeline id.
func (m *MinioObjectStore) GetPipelineKey(pipelineID string) string {
	return path.Join(m.baseFolder, pipelineID)
}

func (m *MinioObjectStore) AddFile(file []byte, filePath string) error {
	var parts int64

	if m.disableMultipart {
		parts = int64(len(file))
	} else {
		parts = multipartDefaultSize
	}

	_, err := m.minioClient.PutObject(
		m.bucketName, filePath, bytes.NewReader(file),
		parts, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to store file %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) DeleteFile(filePath string) error {
	err := m.minioClient.DeleteObject(m.bucketName, filePath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete file %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) GetFile(filePath string) ([]byte, error) {
	reader, err := m.minioClient.GetObject(m.bucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get file %v", filePath)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	bytes := buf.Bytes()

	// Remove single part signature if exists
	if m.disableMultipart {
		re := regexp.MustCompile(`\w+;chunk-signature=\w+`)
		bytes = []byte(re.ReplaceAllString(string(bytes), ""))
	}

	return bytes, nil
}

func (m *MinioObjectStore) AddAsYamlFile(o interface{}, filePath string) error {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal file %v: %v", filePath, err.Error())
	}
	err = m.AddFile(bytes, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file")
	}
	return nil
}

func (m *MinioObjectStore) GetFromYamlFile(o interface{}, filePath string) error {
	bytes, err := m.GetFile(filePath)
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file")
	}
	err = yaml.Unmarshal(bytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal file %v: %v", filePath, err.Error())
	}
	return nil
}

// GetSignedUrl generates a signed url for the artifact identified by artifactURI and bucketConfig.
// The URL expires after expirySeconds. The secret contains the credentials for accessing the object
// store for this artifact. Signed URLs are built using the "GET" method, and are only intended for
// Artifact downloads.
// TODO: Add support for irsa and gcs app credentials pulled from environment
func (m *MinioObjectStore) GetSignedUrl(bucketConfig *objectstore.Config, secret *v1.Secret, expirySeconds time.Duration, artifactURI string) (string, error) {
	s3Client, err := buildClientFromConfig(bucketConfig, secret)
	if err != nil {
		return "", err
	}

	prefixSplit := strings.Split(bucketConfig.Prefix, "/")
	filename := prefixSplit[len(prefixSplit)-1]
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	key, err := objectstore.ArtifactKeyFromURI(artifactURI)
	if err != nil {
		return "", err
	}
	signedUrl, err := s3Client.Presign("GET", bucketConfig.BucketName, key, expirySeconds, reqParams)
	if err != nil {
		return "", util.Wrap(err, "Failed to generate signed url")
	}

	return signedUrl.String(), nil
}

// GetObjectSize Retrieves the Size of the object in bytes.
// Return zero with no error if artifact URI does not exist.
func (m *MinioObjectStore) GetObjectSize(bucketConfig *objectstore.Config, secret *v1.Secret, artifactURI string) (int64, error) {
	s3Client, err := buildClientFromConfig(bucketConfig, secret)
	if err != nil {
		return 0, err
	}
	key, err := objectstore.ArtifactKeyFromURI(artifactURI)
	if err != nil {
		return 0, err
	}
	objectInfo, err := s3Client.StatObject(bucketConfig.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return 0, nil
		}
		return 0, err
	}
	return objectInfo.Size, nil
}

// buildClientFromConfig returns a minio s3 client constructed via the bucket identified by bucketConfig.
func buildClientFromConfig(bucketConfig *objectstore.Config, secret *v1.Secret) (*minio.Client, error) {
	sessionInfo := bucketConfig.Session

	accessKey := string(secret.Data[sessionInfo.AccessKeyKey])
	secretKey := string(secret.Data[sessionInfo.SecretKeyKey])
	parsedUrl, err := url.Parse(sessionInfo.Endpoint)
	if err != nil {
		return nil, util.Wrap(err, "Failed to parse object store endpoint.")
	}

	var secure bool
	switch parsedUrl.Scheme {
	case "http":
		secure = false
	case "https":
		secure = !sessionInfo.DisableSSL
	}
	s3Client, err := minio.New(
		parsedUrl.Host,
		accessKey,
		secretKey,
		secure)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create s3 client.")
	}
	return s3Client, nil
}

func NewMinioObjectStore(minioClient MinioClientInterface, bucketName string, baseFolder string, disableMultipart bool) *MinioObjectStore {
	return &MinioObjectStore{minioClient: minioClient, bucketName: bucketName, baseFolder: baseFolder, disableMultipart: disableMultipart}
}
