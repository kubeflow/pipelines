package testutil

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	"github.com/stretchr/testify/require"
)

func mathOutputRun() *run_model.V2Run {
	return &run_model.V2Run{Tasks: []*run_model.V2PipelineTask{
		{Name: "add", Outputs: &run_model.PipelineTaskInputOutputs{Parameters: []*run_model.InputOutputsIOParameter{{ParameterKey: "Output", Value: float64(12)}}}},
		{Name: "add-container", Outputs: &run_model.PipelineTaskInputOutputs{Parameters: []*run_model.InputOutputsIOParameter{{ParameterKey: "sum", Value: json.Number("12")}}}},
		{Name: "ROOT", Outputs: &run_model.PipelineTaskInputOutputs{Parameters: []*run_model.InputOutputsIOParameter{{ParameterKey: "Output", Value: []interface{}{float64(2), json.Number("4"), float64(6)}}}}},
	}}
}

func TestValidateCollectedMathOutputs(t *testing.T) {
	require.NoError(t, ValidateCollectedMathOutputs(mathOutputRun()))
	require.ErrorContains(t, ValidateCollectedMathOutputs(nil), "missing")
	run := mathOutputRun()
	run.Tasks[0].Outputs.Parameters[0].Value = float64(18)
	require.ErrorContains(t, ValidateCollectedMathOutputs(run), "expected 12")
	run = mathOutputRun()
	run.Tasks[2].Outputs.Parameters[0].Value = []interface{}{float64(6), float64(6), float64(6)}
	require.ErrorContains(t, ValidateCollectedMathOutputs(run), "index 0")
	run = mathOutputRun()
	run.Tasks = run.Tasks[:2]
	require.ErrorContains(t, ValidateCollectedMathOutputs(run), "ROOT")
}
