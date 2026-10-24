package testutil

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
)

// ValidateCollectedMathOutputs checks numeric values from parameters_simple.yaml.
func ValidateCollectedMathOutputs(run *run_model.V2Run) error {
	if run == nil {
		return fmt.Errorf("math pipeline run is missing")
	}
	expected := map[string]string{"add": "Output", "add-container": "sum", "ROOT": "Output"}
	found := map[string]bool{}
	for _, task := range run.Tasks {
		if task == nil {
			return fmt.Errorf("math pipeline contains a missing task record")
		}
		key, wanted := expected[task.Name]
		if !wanted {
			continue
		}
		if task.Outputs == nil {
			return fmt.Errorf("task %s has no output parameters", task.Name)
		}
		for _, parameter := range task.Outputs.Parameters {
			if parameter == nil {
				return fmt.Errorf("task %s contains a missing output parameter", task.Name)
			}
			if parameter.ParameterKey != key {
				continue
			}
			if task.Name == "ROOT" {
				values, ok := parameter.Value.([]interface{})
				if !ok || len(values) != 3 {
					return fmt.Errorf("root collected output must be a three-element JSON array, got %v", parameter.Value)
				}
				for index, expected := range []float64{2, 4, 6} {
					if err := validateMathOutputNumber(values[index], expected); err != nil {
						return fmt.Errorf("root collected output index %d: %w", index, err)
					}
				}
			} else if err := validateMathOutputNumber(parameter.Value, 12); err != nil {
				return fmt.Errorf("task %s output %s: %w", task.Name, key, err)
			}
			found[task.Name] = true
		}
	}
	for task := range expected {
		if !found[task] {
			return fmt.Errorf("math pipeline is missing the expected output for task %s", task)
		}
	}
	return nil
}

func validateMathOutputNumber(value interface{}, expected float64) error {
	var actual float64
	switch number := value.(type) {
	case float64:
		actual = number
	case json.Number:
		var err error
		actual, err = number.Float64()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("expected a JSON number %v, got %T (%v)", expected, value, value)
	}
	if actual != expected {
		return fmt.Errorf("expected %v, got %v", expected, actual)
	}
	return nil
}
