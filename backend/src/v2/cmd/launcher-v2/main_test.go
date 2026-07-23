package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func allProvided(flags []string) map[string]bool {
	provided := make(map[string]bool, len(flags))
	for _, name := range flags {
		provided[name] = true
	}
	return provided
}

func TestRequiredLauncherFlags(t *testing.T) {
	common := []string{
		"executor_type", "pipeline_name", "run_id", "component_spec",
		"pod_name", "pod_uid", "mlmd_server_address", "mlmd_server_port",
	}
	withCommon := func(extra ...string) []string {
		return append(append([]string{}, common...), extra...)
	}
	tests := []struct {
		executorType string
		want         []string
	}{
		{executorType: "container", want: withCommon("execution_id", "executor_input", "ml_pipeline_server_address", "ml_pipeline_server_port")},
		{executorType: "importer", want: withCommon("task_spec", "importer_spec", "parent_dag_id")},
	}
	for _, tc := range tests {
		t.Run(tc.executorType, func(t *testing.T) {
			got, err := requiredLauncherFlags(tc.executorType)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.want, got)
		})
	}

	_, err := requiredLauncherFlags("unknown")
	assert.Error(t, err)
}

func TestValidateLauncherFlags(t *testing.T) {
	tests := []struct {
		name         string
		executorType string
		omit         []string
		wantErr      bool
	}{
		{
			name:         "container with all required flags",
			executorType: "container",
		},
		{
			name:         "importer with all required flags",
			executorType: "importer",
		},
		{
			name:         "container missing execution_id",
			executorType: "container",
			omit:         []string{"execution_id"},
			wantErr:      true,
		},
		{
			name:         "container missing common flag pod_name",
			executorType: "container",
			omit:         []string{"pod_name"},
			wantErr:      true,
		},
		{
			name:         "importer missing importer_spec",
			executorType: "importer",
			omit:         []string{"importer_spec"},
			wantErr:      true,
		},
		{
			name:         "importer missing executor_type",
			executorType: "importer",
			omit:         []string{"executor_type"},
			wantErr:      true,
		},
		{
			name:         "unsupported executor type",
			executorType: "unknown",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			required, err := requiredLauncherFlags(tc.executorType)
			if err != nil {
				assert.True(t, tc.wantErr)
				assert.Error(t, validateLauncherFlags(map[string]bool{}, tc.executorType))
				return
			}
			provided := allProvided(required)
			for _, name := range tc.omit {
				delete(provided, name)
			}
			err = validateLauncherFlags(provided, tc.executorType)
			assert.Equal(t, tc.wantErr, err != nil, "unexpected error state: %v", err)
		})
	}
}
