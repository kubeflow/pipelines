// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"testing"

	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	"github.com/stretchr/testify/require"
)

func TestPipelineVisibilityWaitsForCacheAndTraversesPages(t *testing.T) {
	calls := 0
	list := func(p *params.PipelineServiceListPipelinesParams) ([]*model.V2beta1Pipeline, int, string, error) {
		calls++
		require.Nil(t, p.Filter, "visibility must not depend on the filter under test")
		require.NotNil(t, p.Namespace)
		if calls == 1 {
			require.Nil(t, p.PageToken)
			return nil, 0, "", nil
		}
		if calls == 2 {
			require.Nil(t, p.PageToken)
			return []*model.V2beta1Pipeline{{PipelineID: "unrelated"}}, 2, "second", nil
		}
		require.Equal(t, "second", *p.PageToken)
		return []*model.V2beta1Pipeline{{PipelineID: "created"}}, 2, "", nil
	}
	visible, err := pipelineVisibleInList(list, "created")
	require.NoError(t, err)
	require.False(t, visible)
	visible, err = pipelineVisibleInList(list, "created")
	require.NoError(t, err)
	require.True(t, visible)
	require.Equal(t, 3, calls)
}

func TestPipelineVisibilityPreservesListError(t *testing.T) {
	failure := errors.New("list unavailable")
	visible, err := pipelineVisibleInList(func(*params.PipelineServiceListPipelinesParams) ([]*model.V2beta1Pipeline, int, string, error) {
		return nil, 0, "", failure
	}, "created")
	require.False(t, visible)
	require.ErrorIs(t, err, failure)
}

func TestPipelineVisibilityBoundsPagination(t *testing.T) {
	calls := 0
	visible, err := pipelineVisibleInList(func(*params.PipelineServiceListPipelinesParams) ([]*model.V2beta1Pipeline, int, string, error) {
		calls++
		return nil, 0, "repeated", nil
	}, "created")
	require.False(t, visible)
	require.ErrorContains(t, err, "exceeded 100 pages")
	require.Equal(t, 100, calls)
}
