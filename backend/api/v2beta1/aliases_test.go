// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package v2beta1_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-openapi/strfmt"
	canonical "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	canonicalModel "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/experiment_model"
	legacy "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	legacyClient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client"
	legacyService "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	legacyModel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestLegacyFileHandlesMatchFrozenRegistry(t *testing.T) {
	for _, file := range []protoreflect.FileDescriptor{
		legacy.File_backend_api_v2beta1_artifact_proto,
		legacy.File_backend_api_v2beta1_auth_proto,
		legacy.File_backend_api_v2beta1_experiment_proto,
		legacy.File_backend_api_v2beta1_filter_proto,
		legacy.File_backend_api_v2beta1_healthz_proto,
		legacy.File_backend_api_v2beta1_pipeline_proto,
		legacy.File_backend_api_v2beta1_recurring_run_proto,
		legacy.File_backend_api_v2beta1_report_proto,
		legacy.File_backend_api_v2beta1_run_proto,
		legacy.File_backend_api_v2beta1_runtime_config_proto,
	} {
		t.Run(file.Path(), func(t *testing.T) {
			require.Equal(t, protoreflect.FullName("kubeflow.pipelines.backend.api.v2beta1"), file.Package())
			registered, err := protoregistry.GlobalFiles.FindFileByPath(file.Path())
			require.NoError(t, err)
			require.Same(t, registered, file)
		})
	}
}

func TestLegacyGoModelsAreCanonicalAliases(t *testing.T) {
	old := &legacy.Experiment{DisplayName: "experiment"}
	require.IsType(t, new(canonical.Experiment), old)
	require.Equal(t, canonical.Experiment_ARCHIVED, legacy.Experiment_ARCHIVED)
	require.Equal(t, canonical.ExperimentService_CreateExperiment_FullMethodName, legacy.ExperimentService_CreateExperiment_FullMethodName)
	// Recompiled Go aliases have canonical protobuf identity; frozen clients
	// still use legacy method names and descriptors at the server boundary.
	require.Equal(t, "kubeflow.pipelines.backend.api.v2.Experiment", string(old.ProtoReflect().Descriptor().FullName()))
}

//nolint:staticcheck // Exercise deprecated legacy constructors for compatibility.
func TestLegacyGoHTTPClientUsesCanonicalRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/apis/v2/experiments/experiment-id", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"experiment_id":"experiment-id","display_name":"shared implementation"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(upstream.Close)
	endpoint, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	config := legacyClient.DefaultTransportConfig().WithHost(endpoint.Host).WithSchemes([]string{endpoint.Scheme})
	client := legacyClient.NewHTTPClientWithConfig(strfmt.Default, config)
	params := legacyService.NewExperimentServiceGetExperimentParams().WithExperimentID("experiment-id")
	response, err := client.ExperimentService.ExperimentServiceGetExperimentContext(t.Context(), params)
	require.NoError(t, err)
	require.IsType(t, new(legacyModel.V2beta1Experiment), response.Payload)
	require.IsType(t, new(canonicalModel.V2Experiment), response.Payload)
	require.Equal(t, "shared implementation", response.Payload.DisplayName)
}

//nolint:staticcheck // Exercise deprecated legacy constructors for compatibility.
func TestLegacyHTTPDefaultSchemesRemainConfigurable(t *testing.T) {
	original := legacyClient.DefaultSchemes
	t.Cleanup(func() { legacyClient.DefaultSchemes = original })
	legacyClient.DefaultSchemes = []string{"https"}
	require.Equal(t, []string{"https"}, legacyClient.DefaultTransportConfig().Schemes)
	require.NotNil(t, legacyClient.NewHTTPClient(nil))
	require.NotNil(t, legacyClient.NewHTTPClientWithConfig(nil, nil))
}
