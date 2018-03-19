// Copyright 2018 Google LLC
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
	"ml/src/util"

	"github.com/minio/minio-go"
)

// Manager managing actual package file.
type PackageManagerInterface interface {
	// Create the package file
	CreatePackageFile(template []byte, fileName string) error

	// Get the template for a given package.
	GetTemplate(pkgName string) ([]byte, error)
}

// Managing package using Minio
type MinioPackageManager struct {
	minioClient MinioClientInterface
	bucketName  string
}

func (m *MinioPackageManager) CreatePackageFile(template []byte, fileName string) error {
	_, err := m.minioClient.PutObject(m.bucketName, fileName, bytes.NewReader(template), -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalError("Failed to store a new package.", err.Error())
	}
	return nil
}

func (m *MinioPackageManager) GetTemplate(pkgName string) ([]byte, error) {
	reader, err := m.minioClient.GetObject(m.bucketName, pkgName, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalError("Failed to store a new package.", err.Error())
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.Bytes(), nil
}

func NewMinioPackageManager(minioClient MinioClientInterface, bucketName string) *MinioPackageManager {
	return &MinioPackageManager{minioClient: minioClient, bucketName: bucketName}
}
