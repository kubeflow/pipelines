package argocompiler

import (
	"encoding/json"
	"fmt"
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func driverPlugin(params map[string]interface{}) (*wfapi.Plugin, error) {
	pluginConfig := map[string]interface{}{
		"driver-plugin": map[string]interface{}{
			"args": params,
		},
	}
	jsonConfig, err := json.Marshal(pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("driver plugin creation error: marshaling plugin config to JSON failed: %w", err)
	}
	return &wfapi.Plugin{Object: wfapi.Object{
		Value: jsonConfig,
	}}, nil
}
