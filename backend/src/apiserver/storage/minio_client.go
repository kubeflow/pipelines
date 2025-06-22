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
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// Create interface for S3 client struct, making it more unit testable.
// Renamed from MinioClientInterface but keeping same method signatures for compatibility
type MinioClientInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (n int64, err error)
	GetObject(bucketName, objectName string) (io.ReadCloser, error)
	DeleteObject(bucketName, objectName string) error
}

type MinioClient struct {
	Client s3iface.S3API
}

func (c *MinioClient) PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (n int64, err error) {
	// Read the content into a buffer to create a ReadSeeker
	buf := new(bytes.Buffer)
	n, err = buf.ReadFrom(reader)
	if err != nil {
		return 0, err
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(objectName),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(n),
	}

	_, err = c.Client.PutObject(input)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (c *MinioClient) GetObject(bucketName, objectName string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}

	result, err := c.Client.GetObject(input)
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

func (c *MinioClient) DeleteObject(bucketName, objectName string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}

	_, err := c.Client.DeleteObject(input)
	return err
}
