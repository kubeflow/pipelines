// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driverapi

import (
	"encoding/json"
	"fmt"
)

// PluginName is the key under an Argo template's plugin object that carries
// driver arguments. The compiler writes it and the executor plugin reads it.
const PluginName = "driver-plugin"

// PluginConfig is the driver-plugin entry of an Argo template plugin value.
type PluginConfig struct {
	Args map[string]any `json:"args"`
}

// EncodePluginValue builds a template plugin value carrying args under PluginName.
func EncodePluginValue(args map[string]any) ([]byte, error) {
	value, err := json.Marshal(map[string]any{PluginName: PluginConfig{Args: args}})
	if err != nil {
		return nil, fmt.Errorf("cannot encode %s args; pass JSON-serializable values: %w", PluginName, err)
	}
	return value, nil
}

// DecodePluginConfig reads the driver-plugin entry of a template plugin value.
// owned is false when the template belongs to another executor plugin.
func DecodePluginConfig(value []byte) (config *PluginConfig, owned bool, err error) {
	var plugins map[string]json.RawMessage
	if err := json.Unmarshal(value, &plugins); err != nil {
		return nil, false, fmt.Errorf("template plugin value is not a JSON object; send Argo ExecuteTemplateArgs with template.plugin: %w", err)
	}
	raw, owned := plugins[PluginName]
	if !owned {
		return nil, false, nil
	}
	if err := json.Unmarshal(raw, &config); err != nil || config == nil {
		return nil, true, fmt.Errorf("template plugin %q must be an object with an args object; recompile the pipeline with a current API server", PluginName)
	}
	if config.Args == nil {
		return nil, true, fmt.Errorf("template plugin %q has no args; recompile the pipeline with a current API server", PluginName)
	}
	return config, true, nil
}

// ParseDriverPluginArgs decodes and validates driver arguments from a template
// plugin value. owned is false when the template belongs to another plugin.
func ParseDriverPluginArgs(value []byte) (args *DriverPluginArgs, owned bool, err error) {
	config, owned, err := DecodePluginConfig(value)
	if err != nil || !owned {
		return nil, owned, err
	}
	encoded, err := json.Marshal(config.Args)
	if err != nil {
		return nil, true, fmt.Errorf("cannot encode %s args: %w", PluginName, err)
	}
	args = &DriverPluginArgs{}
	if err := json.Unmarshal(encoded, args); err != nil {
		return nil, true, fmt.Errorf("%s args have the wrong shape; recompile the pipeline with a current API server: %w", PluginName, err)
	}
	if err := args.validate(config.Args); err != nil {
		return nil, true, err
	}
	return args, true, nil
}
