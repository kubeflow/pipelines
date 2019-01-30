package storage

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
)

// S3Client ...
type S3Client struct {
	sess *session.Session
	svc  *s3.S3
}

// Init ...
func (t *S3Client) Init(region string) {

	glog.Infof("init s3 client, default region: %s \n", region)
	// Use default authentication schema: 1. ENV vars; 2. credential file (.aws/credentials); 3. IAM EC2 roles
	conf := aws.Config{
		Region: aws.String(region),
	}

	t.sess, _ = session.NewSession(&conf)
	t.svc = s3.New(t.sess)
}

// PutObject ...
func (t *S3Client) PutObject(bucketName, objectName string, reader io.ReadSeeker, contentType string) (n int64, err error) {

	glog.Infof("put object, bucket: %s, key: %s \n", bucketName, objectName)
	req := s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &objectName,
		Body:        reader,
		ContentType: &contentType,
	}
	_, err = t.svc.PutObject(&req)
	return 0, err
}

// GetObject ...
func (t *S3Client) GetObject(bucketName, objectName string) (io.Reader, error) {

	glog.Infof("get object, bucket: %s, key: %s \n", bucketName, objectName)
	req := s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &objectName,
	}
	res, err := t.svc.GetObject(&req)
	return res.Body, err
}

// DeleteObject ...
func (t *S3Client) DeleteObject(bucketName, objectName string) error {

	glog.Infof("delete object, bucket: %s, key: %s \n", bucketName, objectName)
	req := s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &objectName,
	}
	_, err := t.svc.DeleteObject(&req)
	return err
}
