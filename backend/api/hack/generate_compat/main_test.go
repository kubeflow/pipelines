// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectAndRenderAliases(t *testing.T) {
	root := t.TempDir()
	source := `package client

type V2Thing struct{}
const (V2State = iota; V2Other)
var File_backend_api_v2_run_proto string
var private int
func NewV2Thing() *V2Thing { return nil }
func (*V2Thing) Reset() {}
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "client.go"), []byte(source), 0o644))
	// Tests and receiver methods must not become package-level exports.
	require.NoError(t, os.WriteFile(filepath.Join(root, "client_test.go"), []byte("invalid test source"), 0o644))
	packages, err := collectPackages(root)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	pkg := packages["."]
	require.Len(t, pkg.symbols, 5)
	for name, kind := range map[string]string{
		"V2beta1Thing": "type", "V2beta1State": "const", "V2beta1Other": "const",
		"File_backend_api_v2beta1_run_proto": "var", "NewV2beta1Thing": "var",
	} {
		require.Equal(t, kind, pkg.symbols[name].kind, name)
	}
	content, err := renderPackage(pkg, canonicalImport+"go_client")
	require.NoError(t, err)
	require.Contains(t, string(content), "V2beta1Thing = canonical.V2Thing")
	require.Regexp(t, `NewV2beta1Thing\s*= canonical\.NewV2Thing`, string(content))
	require.Contains(t, string(content), "File_backend_api_v2beta1_run_proto = canonical.File_backend_api_v2_run_proto")
	require.NotContains(t, string(content), "Reset")
	require.Contains(t, string(content), `"github.com/kubeflow/pipelines/backend/api/v2beta1"`)
}

func TestGenerateReplacesOnlyClientTreesAndIsDeterministic(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"go_client", "go_http_client/run_model"} {
		canonical := filepath.Join(root, "backend/api/v2", dir)
		legacy := filepath.Join(root, "backend/api/v2beta1", dir)
		require.NoError(t, os.MkdirAll(canonical, 0o755))
		require.NoError(t, os.MkdirAll(legacy, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(canonical, "model.go"), []byte("package client; type V2Run struct{}"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(legacy, "obsolete.go"), []byte("old generated code"), 0o644))
	}
	frozen := filepath.Join(root, "backend/api/v2beta1/legacy_descriptor.pb")
	require.NoError(t, os.WriteFile(frozen, []byte("immutable schema"), 0o644))
	require.NoError(t, generate(root))
	before := make(map[string]string)
	for _, dir := range []string{"go_client", "go_http_client/run_model"} {
		legacy := filepath.Join(root, "backend/api/v2beta1", dir)
		require.NoFileExists(t, filepath.Join(legacy, "obsolete.go"))
		content, err := os.ReadFile(filepath.Join(legacy, "aliases.go"))
		require.NoError(t, err)
		before[dir] = string(content)
	}
	require.NoError(t, generate(root))
	for dir, want := range before {
		content, err := os.ReadFile(filepath.Join(root, "backend/api/v2beta1", dir, "aliases.go"))
		require.NoError(t, err)
		require.Equal(t, want, string(content))
	}
	content, err := os.ReadFile(frozen)
	require.NoError(t, err)
	require.Equal(t, "immutable schema", string(content))
}

func TestGenerateRejectsMissingCanonicalPackagesBeforeDeletingShims(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "backend/api/v2beta1/go_client/aliases.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacy), 0o755))
	require.NoError(t, os.WriteFile(legacy, []byte("preserved"), 0o644))
	require.Error(t, generate(root))
	content, err := os.ReadFile(legacy)
	require.NoError(t, err)
	require.Equal(t, "preserved", string(content))
}
