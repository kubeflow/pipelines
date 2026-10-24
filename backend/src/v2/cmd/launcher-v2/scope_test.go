package main

import (
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestLauncherUsesLogicalScopeForMaterializedIteration(t *testing.T) {
	spec := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"root":{"dag":{"tasks":{"loop":{"componentRef":{"name":"leaf"}}}}},
		"components":{"leaf":{"executorLabel":"executor"}}
	}`), spec))
	parent := &apiv2.PipelineTask{ScopePath: "root.loop"}
	task := &apiv2.PipelineTask{ScopePath: "root.loop"}
	scope, err := launcherTaskScope(spec, parent, task, "loop", "kubernetes")
	require.NoError(t, err)
	require.Equal(t, "executor", scope.GetLast().GetComponentSpec().GetExecutorLabel())
}
