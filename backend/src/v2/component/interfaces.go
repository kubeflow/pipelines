// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"gocloud.dev/blob"
)

// FileSystem provides an interface for file system operations.
// This abstraction allows for easy mocking in tests.
type FileSystem interface {
	// MkdirAll creates a directory path and all parents if needed
	MkdirAll(path string, perm os.FileMode) error

	// Create creates or truncates the named file
	Create(name string) (*os.File, error)

	// ReadFile reads the entire file
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to a file
	WriteFile(name string, data []byte, perm os.FileMode) error

	// Stat returns file info
	Stat(name string) (fs.FileInfo, error)
}

// CommandExecutor provides an interface for executing system commands.
// This abstraction allows for easy mocking in tests.
type CommandExecutor interface {
	// Run executes a command with the given arguments and I/O streams
	Run(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// ObjectStoreClientInterface provides an interface for object store operations.
// This abstraction allows for easy mocking in tests.
type ObjectStoreClientInterface interface {
	// UploadArtifact uploads an artifact from local path to remote URI
	UploadArtifact(ctx context.Context, localPath, remoteURI, artifactKey string) error

	// DownloadArtifact downloads an artifact from remote URI to local path
	DownloadArtifact(ctx context.Context, remoteURI, localPath, artifactKey string) error
}

// OSFileSystem is the production implementation of FileSystem using real os calls
type OSFileSystem struct{}

func (f *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f *OSFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (f *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (f *OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (f *OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// RealCommandExecutor is the production implementation of CommandExecutor
type RealCommandExecutor struct{}

func (e *RealCommandExecutor) Run(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	command := exec.Command(cmd, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

// ObjectStoreClient is the production implementation using the actual objectstore package
type ObjectStoreClient struct {
	launcher *LauncherV2
}

func NewObjectStoreClient(launcher *LauncherV2) *ObjectStoreClient {
	return &ObjectStoreClient{launcher: launcher}
}

func (c *ObjectStoreClient) UploadArtifact(ctx context.Context, localPath, remoteURI, artifactKey string) error {
	openedBucket, blobKey, err := c.getBucket(ctx, artifactKey, remoteURI, c.launcher.launcherConfig)
	if err != nil {
		return fmt.Errorf("failed to get opened bucket for output artifact %q: %w", artifactKey, err)
	}
	uploadErr := objectstore.UploadBlob(ctx, openedBucket, localPath, blobKey)
	if uploadErr != nil {
		return fmt.Errorf("failed to upload output artifact %q: %w", artifactKey, uploadErr)
	}
	return nil
}

func (c *ObjectStoreClient) DownloadArtifact(ctx context.Context, remoteURI, localPath, artifactKey string) error {
	openedBucket, blobKey, err := c.getBucket(ctx, artifactKey, remoteURI, c.launcher.launcherConfig)
	if err != nil {
		return fmt.Errorf("failed to get opened bucket for input artifact %q: %w", artifactKey, err)
	}
	if err = objectstore.DownloadBlob(ctx, openedBucket, localPath, blobKey); err != nil {
		return fmt.Errorf("failed to download input artifact %q from remote storage URI %q: %w", artifactKey, remoteURI, err)
	}
	return err
}

func (c *ObjectStoreClient) getBucket(
	ctx context.Context,
	artifactKey,
	artifactURI string,
	launcherConfig *config.Config,
) (*blob.Bucket, string, error) {
	prefix, base, err := objectstore.SplitObjectURI(artifactURI)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get base URI path for input artifact %q: %w", artifactKey, err)
	}
	bucketConfig, err := objectstore.ParseBucketPathToConfig(prefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get base URI path for input artifact %q: %w", artifactKey, err)
	}

	key := bucketConfig.Hash()
	var openedBucket *blob.Bucket
	if cachedBucket, exists := c.launcher.openedBucketCache[key]; exists {
		openedBucket = cachedBucket
	} else {
		// Create new opened bucket and store in cache
		storeSessionInfo, err := launcherConfig.GetStoreSessionInfo(bucketConfig.PrefixedBucket())
		if err != nil {
			return nil, "", fmt.Errorf("failed to get store session info for bucket %q: %w", bucketConfig.PrefixedBucket(), err)
		}
		newOpenBucket, err := objectstore.OpenBucket(ctx, c.launcher.clientManager.K8sClient(), c.launcher.options.Namespace, bucketConfig, &storeSessionInfo)
		if err != nil {
			return nil, "", fmt.Errorf("failed to open bucket %q: %w", bucketConfig.PrefixedBucket(), err)
		}
		c.launcher.openedBucketCache[bucketConfig.Hash()] = newOpenBucket
		openedBucket = newOpenBucket
	}

	return openedBucket, base, nil
}
