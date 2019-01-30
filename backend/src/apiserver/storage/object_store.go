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

// ObjectStoreInterface is an interface for object store.
type ObjectStoreInterface interface {
	AddFile(template []byte, filePath string) error
	DeleteFile(filePath string) error
	GetFile(filePath string) ([]byte, error)
	AddAsYamlFile(o interface{}, filePath string) error
	GetFromYamlFile(o interface{}, filePath string) error
}

// NewMinioObjectStore ...
func NewMinioObjectStore(minioClient MinioClientInterface, bucketName string) *MinioObjectStore {
	return &MinioObjectStore{minioClient: minioClient, bucketName: bucketName}
}

// NewS3ObjectStore ...
func NewS3ObjectStore(client S3Client, bucketName string) *S3ObjectStore {
	return &S3ObjectStore{s3Client: client, bucketName: bucketName}
}

func buildPath(folder, file string) string {
	return folder + "/" + file
}
