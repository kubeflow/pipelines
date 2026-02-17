// Copyright 2018-2025 The Kubeflow Authors
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

package clientmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
)

// getTestSQLite returns an isolated in-memory sqlite DB for each test.
// We use a unique DSN per test to avoid "table already exists" collisions when tests reuse the same shared cache.
func getTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	// sanitize test name to be a valid sqlite file identifier
	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func createOldExperimentSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Ensure a clean slate if a previous test already created it in the shared cache.
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS experiments`).Error)

	stmt := `
CREATE TABLE IF NOT EXISTS experiments (
  UUID TEXT NOT NULL,
  Name TEXT NOT NULL,
  Namespace TEXT NOT NULL,
  CreatedAtInSec INTEGER NOT NULL DEFAULT 0,
  LastRunCreatedAtInSec INTEGER NOT NULL DEFAULT 0,
  StorageState TEXT NOT NULL DEFAULT 'STORAGESTATE_AVAILABLE',
  PRIMARY KEY (UUID)
);`
	require.NoError(t, db.Exec(stmt).Error)
}

// insertTooLongExperimentName inserts one row whose Name exceeds 128 chars.
func insertTooLongExperimentName(t *testing.T, db *gorm.DB) {
	t.Helper()
	longName := strings.Repeat("x", 150)
	err := db.Exec(`
INSERT INTO experiments (UUID, Name, Namespace, CreatedAtInSec, LastRunCreatedAtInSec, StorageState)
VALUES (?, ?, 'ns', 0, 0, 'STORAGESTATE_AVAILABLE')`, "uuid-1", longName).Error
	require.NoError(t, err)
}

func TestRunPreflightLengthChecks_FailOnTooLong(t *testing.T) {
	db := getTestSQLite(t)

	createOldExperimentSchema(t, db)
	insertTooLongExperimentName(t, db)

	specs := []validation.ColLenSpec{
		{Model: &model.Experiment{}, Field: "Name", Max: 128},
	}

	dialect := GetDialect("sqlite")
	err := runPreflightLengthChecks(db, dialect, specs)
	t.Logf("FULL ERR:\n%+v", err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Preflight")
}

func TestRunPreflightLengthChecks_PassWhenOK(t *testing.T) {
	db := getTestSQLite(t)

	createOldExperimentSchema(t, db)
	// no long rows

	dialect := GetDialect("sqlite")
	err := runPreflightLengthChecks(db, dialect, []validation.ColLenSpec{
		{Model: &model.Experiment{}, Field: "Name", Max: 128},
	})
	require.NoError(t, err)
}

func TestFieldMeta_TaskRunId(t *testing.T) {
	// FieldMeta only inspects schema; sqlite driver is sufficient.
	db := getTestSQLite(t)
	table, dbCol, err := FieldMeta(db, &model.Task{}, "RunID")
	require.NoError(t, err)
	assert.Equal(t, "tasks", table)
	assert.Equal(t, "RunUUID", dbCol)
}

func TestValidateRequiredConfig(t *testing.T) {
	tests := []struct {
		name        string
		bucketName  string
		host        string
		accessKey   string
		secretKey   string
		wantErr     bool
		errContains string
	}{
		{
			name:       "all fields provided",
			bucketName: "my-bucket",
			host:       "minio-service",
			accessKey:  "access",
			secretKey:  "secret",
			wantErr:    false,
		},
		{
			name:        "empty bucket name",
			bucketName:  "",
			host:        "minio-service",
			accessKey:   "access",
			secretKey:   "secret",
			wantErr:     true,
			errContains: "BucketName is required",
		},
		{
			name:        "empty host",
			bucketName:  "my-bucket",
			host:        "",
			accessKey:   "access",
			secretKey:   "secret",
			wantErr:     true,
			errContains: "Host is required",
		},
		{
			name:       "empty credentials for IRSA - should pass",
			bucketName: "my-bucket",
			host:       "s3.amazonaws.com",
			accessKey:  "",
			secretKey:  "",
			wantErr:    false,
		},
		{
			name:        "only accessKey provided - should fail",
			bucketName:  "my-bucket",
			host:        "s3.amazonaws.com",
			accessKey:   "access",
			secretKey:   "",
			wantErr:     true,
			errContains: "must both be set or both be empty",
		},
		{
			name:        "only secretKey provided - should fail",
			bucketName:  "my-bucket",
			host:        "s3.amazonaws.com",
			accessKey:   "",
			secretKey:   "secret",
			wantErr:     true,
			errContains: "must both be set or both be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequiredConfig(tt.bucketName, tt.host, tt.accessKey, tt.secretKey)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type fakeS3Client struct {
	headErr         error
	createErr       error
	headCalls       int
	createCalls     int
	lastCreateInput *s3.CreateBucketInput
}

func (f *fakeS3Client) HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	f.headCalls++
	if f.headErr != nil {
		return nil, f.headErr
	}
	return &s3.HeadBucketOutput{}, nil
}

func (f *fakeS3Client) CreateBucket(_ context.Context, params *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	f.createCalls++
	f.lastCreateInput = params
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &s3.CreateBucketOutput{}, nil
}

func TestEnsureBucketExistsWithClient_BucketAlreadyExists(t *testing.T) {
	client := &fakeS3Client{}
	cfg := &blobStorageConfig{bucketName: "existing", region: "us-east-1"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 0, client.createCalls)
}

func TestEnsureBucketExistsWithClient_CreatesBucket(t *testing.T) {
	client := &fakeS3Client{headErr: &s3types.NotFound{}}
	cfg := &blobStorageConfig{bucketName: "missing", region: "us-west-2"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 1, client.createCalls)
	require.NotNil(t, client.lastCreateInput.CreateBucketConfiguration)
	assert.Equal(t, s3types.BucketLocationConstraint("us-west-2"), client.lastCreateInput.CreateBucketConfiguration.LocationConstraint)
}

func TestEnsureBucketExistsWithClient_CreateBucketAlreadyOwned(t *testing.T) {
	client := &fakeS3Client{
		headErr:   &s3types.NotFound{},
		createErr: &s3types.BucketAlreadyOwnedByYou{},
	}
	cfg := &blobStorageConfig{bucketName: "missing", region: "us-east-1"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 1, client.createCalls)
}

func TestEnsureBucketExistsWithClient_CreateBucketAlreadyExistsForeignOwner(t *testing.T) {
	client := &fakeS3Client{
		headErr:   &s3types.NotFound{},
		createErr: &s3types.BucketAlreadyExists{},
	}
	cfg := &blobStorageConfig{bucketName: "foreign-bucket", region: "us-east-1"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create object store bucket")
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 1, client.createCalls)
}

func TestEnsureBucketExistsWithClient_CreateBucketError(t *testing.T) {
	client := &fakeS3Client{
		headErr:   &s3types.NotFound{},
		createErr: errors.New("access denied"),
	}
	cfg := &blobStorageConfig{bucketName: "missing", region: "us-east-1"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create object store bucket")
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 1, client.createCalls)
}

func TestEnsureBucketExistsWithClient_HeadBucketPermissionDenied(t *testing.T) {
	client := &fakeS3Client{
		headErr: errors.New("access denied"),
	}
	cfg := &blobStorageConfig{bucketName: "forbidden", region: "us-east-1"}

	err := ensureBucketExistsWithClient(context.Background(), client, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if bucket")
	assert.Equal(t, 1, client.headCalls)
	assert.Equal(t, 0, client.createCalls)
}

func TestBuildCreateBucketInput_DefaultRegion(t *testing.T) {
	input := buildCreateBucketInput(&blobStorageConfig{
		bucketName: "test",
		region:     "us-east-1",
	})

	assert.Nil(t, input.CreateBucketConfiguration)
}

type fakeS3HTTPServer struct {
	t       *testing.T
	server  *httptest.Server
	buckets map[string]bool
	log     []string
}

func newFakeS3HTTPServer(t *testing.T) *fakeS3HTTPServer {
	f := &fakeS3HTTPServer{
		t:       t,
		buckets: make(map[string]bool),
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeS3HTTPServer) endpoint() string {
	return strings.TrimPrefix(f.server.URL, "http://")
}

func (f *fakeS3HTTPServer) handle(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing bucket", http.StatusBadRequest)
		return
	}
	bucket := parts[0]

	switch r.Method {
	case http.MethodHead:
		f.log = append(f.log, "HEAD:"+bucket)
		if f.buckets[bucket] {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	case http.MethodPut:
		f.log = append(f.log, "PUT:"+bucket)
		if f.buckets[bucket] {
			// Simulate already existing bucket.
			http.Error(w, "BucketAlreadyOwnedByYou", http.StatusConflict)
			return
		}
		f.buckets[bucket] = true
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unsupported", http.StatusMethodNotAllowed)
	}
}

func (f *fakeS3HTTPServer) setBucket(name string) {
	f.buckets[name] = true
}

func (f *fakeS3HTTPServer) requestLog() []string {
	return f.log
}

func TestEnsureBucketExists_IntegrationCreatesBucket(t *testing.T) {
	server := newFakeS3HTTPServer(t)
	cfg := &blobStorageConfig{
		bucketName: "integration-bucket",
		endpoint:   server.endpoint(),
		region:     "us-west-2",
		secure:     false,
		accessKey:  "key",
		secretKey:  "secret",
	}

	err := ensureBucketExists(context.Background(), cfg)
	require.NoError(t, err)

	log := server.requestLog()
	require.Len(t, log, 2)
	assert.Equal(t, "HEAD:integration-bucket", log[0])
	assert.Equal(t, "PUT:integration-bucket", log[1])
}

func TestEnsureBucketExists_IntegrationBucketAlreadyExists(t *testing.T) {
	server := newFakeS3HTTPServer(t)
	server.setBucket("existing-bucket")

	cfg := &blobStorageConfig{
		bucketName: "existing-bucket",
		endpoint:   server.endpoint(),
		region:     "us-east-1",
		secure:     false,
		accessKey:  "key",
		secretKey:  "secret",
	}

	err := ensureBucketExists(context.Background(), cfg)
	require.NoError(t, err)

	log := server.requestLog()
	require.Len(t, log, 1)
	assert.Equal(t, "HEAD:existing-bucket", log[0])
}

func TestLoadAWSConfig_EmptyCredentials(t *testing.T) {
	cfg := &blobStorageConfig{
		region:    "us-west-2",
		accessKey: "",
		secretKey: "",
	}

	awsCfg, err := loadAWSConfig(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", awsCfg.Region)
}

func TestLoadAWSConfig_WithCredentials(t *testing.T) {
	cfg := &blobStorageConfig{
		region:    "us-east-1",
		accessKey: "test-key",
		secretKey: "test-secret",
	}

	awsCfg, err := loadAWSConfig(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", awsCfg.Region)

	creds, err := awsCfg.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-key", creds.AccessKeyID)
	assert.Equal(t, "test-secret", creds.SecretAccessKey)
}
