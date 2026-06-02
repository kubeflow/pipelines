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
	"sync"
	"time"
)

// MockFileSystem is a mock implementation of FileSystem for testing
type MockFileSystem struct {
	mu sync.Mutex

	// Track calls
	MkdirAllCalls  []MkdirAllCall
	CreateCalls    []string
	ReadFileCalls  []string
	WriteFileCalls []WriteFileCall
	StatCalls      []string

	// In-memory file system
	files map[string][]byte
	dirs  map[string]bool

	// Control behavior
	MkdirAllError  error
	CreateError    error
	ReadFileError  error
	WriteFileError error
	StatError      error
}

type MkdirAllCall struct {
	Path string
	Perm os.FileMode
}

type WriteFileCall struct {
	Name string
	Data []byte
	Perm os.FileMode
}

type mockFileInfo struct {
	name string
	size int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MkdirAllCalls = append(m.MkdirAllCalls, MkdirAllCall{Path: path, Perm: perm})
	if m.MkdirAllError != nil {
		return m.MkdirAllError
	}
	m.dirs[path] = true
	return nil
}

func (m *MockFileSystem) Create(name string) (*os.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls = append(m.CreateCalls, name)
	if m.CreateError != nil {
		return nil, m.CreateError
	}
	// For mock purposes, we don't return a real file
	// Tests should use WriteFile instead
	m.files[name] = []byte{}
	return nil, nil
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReadFileCalls = append(m.ReadFileCalls, name)
	if m.ReadFileError != nil {
		return nil, m.ReadFileError
	}
	data, exists := m.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m *MockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WriteFileCalls = append(m.WriteFileCalls, WriteFileCall{Name: name, Data: data, Perm: perm})
	if m.WriteFileError != nil {
		return m.WriteFileError
	}
	m.files[name] = data
	return nil
}

func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StatCalls = append(m.StatCalls, name)
	if m.StatError != nil {
		return nil, m.StatError
	}
	data, exists := m.files[name]
	if !exists {
		return nil, os.ErrNotExist
	}
	return &mockFileInfo{name: name, size: int64(len(data))}, nil
}

// SetFileContent sets file content for testing.
func (m *MockFileSystem) SetFileContent(name string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[name] = data
}

// GetFileContent gets file content for assertions.
func (m *MockFileSystem) GetFileContent(name string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, exists := m.files[name]
	return data, exists
}

// MockCommandExecutor is a mock implementation of CommandExecutor for testing
type MockCommandExecutor struct {
	mu sync.Mutex

	// Track calls
	RunCalls []CommandCall

	// Control behavior
	RunError error
	// Optional: custom function to execute instead
	RunFunc func(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type CommandCall struct {
	Cmd    string
	Args   []string
	Stdout string
	Stderr string
}

func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{}
}

func (m *MockCommandExecutor) Run(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := CommandCall{
		Cmd:  cmd,
		Args: args,
	}

	// If custom function is provided, use it
	if m.RunFunc != nil {
		err := m.RunFunc(ctx, cmd, args, stdin, stdout, stderr)
		m.RunCalls = append(m.RunCalls, call)
		return err
	}

	// Otherwise use default error or success
	m.RunCalls = append(m.RunCalls, call)
	return m.RunError
}

// CallCount returns the number of times Run was called.
func (m *MockCommandExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.RunCalls)
}

// MockObjectStoreClient is a mock implementation of ObjectStoreClientInterface for testing
type MockObjectStoreClient struct {
	mu sync.Mutex

	// Track calls
	UploadCalls   []ArtifactCall
	DownloadCalls []ArtifactCall

	// In-memory artifact storage (remoteURI -> data)
	artifacts map[string][]byte

	// Control behavior
	UploadError   error
	DownloadError error
}

type ArtifactCall struct {
	LocalPath   string
	RemoteURI   string
	ArtifactKey string
}

func NewMockObjectStoreClient() *MockObjectStoreClient {
	return &MockObjectStoreClient{
		artifacts: make(map[string][]byte),
	}
}

func (m *MockObjectStoreClient) UploadArtifact(ctx context.Context, localPath, remoteURI, artifactKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UploadCalls = append(m.UploadCalls, ArtifactCall{
		LocalPath:   localPath,
		RemoteURI:   remoteURI,
		ArtifactKey: artifactKey,
	})

	if m.UploadError != nil {
		return m.UploadError
	}

	// Simulate upload by storing in memory
	m.artifacts[remoteURI] = []byte(fmt.Sprintf("uploaded from %s", localPath))
	return nil
}

func (m *MockObjectStoreClient) DownloadArtifact(ctx context.Context, remoteURI, localPath, artifactKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DownloadCalls = append(m.DownloadCalls, ArtifactCall{
		LocalPath:   localPath,
		RemoteURI:   remoteURI,
		ArtifactKey: artifactKey,
	})

	if m.DownloadError != nil {
		return m.DownloadError
	}

	// Check if artifact exists
	if _, exists := m.artifacts[remoteURI]; !exists {
		return fmt.Errorf("artifact not found: %s", remoteURI)
	}

	return nil
}

// SetArtifact sets artifact content for testing.
func (m *MockObjectStoreClient) SetArtifact(remoteURI string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.artifacts[remoteURI] = data
}

// WasUploaded is a helper to check if artifact was uploaded
func (m *MockObjectStoreClient) WasUploaded(remoteURI string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.artifacts[remoteURI]
	return exists
}

// GetUploadCallsForKey is a helper to get upload calls for a specific artifact key
func (m *MockObjectStoreClient) GetUploadCallsForKey(artifactKey string) []ArtifactCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	var calls []ArtifactCall
	for _, call := range m.UploadCalls {
		if call.ArtifactKey == artifactKey {
			calls = append(calls, call)
		}
	}
	return calls
}

// GetDownloadCallsForKey is a helper to get download calls for a specific artifact key
func (m *MockObjectStoreClient) GetDownloadCallsForKey(artifactKey string) []ArtifactCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	var calls []ArtifactCall
	for _, call := range m.DownloadCalls {
		if call.ArtifactKey == artifactKey {
			calls = append(calls, call)
		}
	}
	return calls
}
