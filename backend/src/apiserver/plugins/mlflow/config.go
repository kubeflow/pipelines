// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// DefaultExperimentName is the MLflow experiment name used when the user
	// and admin configuration do not specify one.
	DefaultExperimentName = "KFP-Default"
)

// ApplySettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplySettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := true
		settings.WorkspacesEnabled = &defaultEnabled
	}
	if settings.DefaultExperimentName == "" {
		settings.DefaultExperimentName = DefaultExperimentName
	}
	if settings.ExperimentDescription == nil {
		d := DefaultExperimentDescription
		settings.ExperimentDescription = &d
	}
	return settings
}

// MLflowPluginInput represents the user-facing plugins_input.mlflow schema.
type MLflowPluginInput struct {
	ExperimentName string `json:"experiment_name,omitempty"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
}

// IsEnabled reports whether the global plugins.mlflow configuration is present,
// indicating the API server has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet("plugins.mlflow")
}

// GetGlobalMLflowConfig reads the global plugins.mlflow configuration
func GetGlobalMLflowConfig() (apiserverPlugins.PluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return apiserverPlugins.PluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg apiserverPlugins.PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

// ResolveMLflowPluginConfig builds an MLflowPluginConfig for the given input generic PluginConfig.
func ResolveMLflowPluginConfig(runPluginConfig *apiserverPlugins.PluginConfig, resolvedMLflowSettings *commonmlflow.MLflowPluginSettings) (*commonmlflow.MLflowPluginConfig, error) {
	if runPluginConfig == nil || resolvedMLflowSettings == nil {
		return nil, fmt.Errorf("runPluginConfig and resolvedMLflowSettings must be non-nil")
	}

	resolvedTimeout := runPluginConfig.Timeout
	if resolvedTimeout == "" {
		resolvedTimeout = apiserverPlugins.DefaultTimeout
	}

	resolvedMLflowCfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: runPluginConfig.Endpoint,
		Timeout:  resolvedTimeout,
		TLS:      runPluginConfig.TLS,
		Settings: resolvedMLflowSettings,
	}
	return resolvedMLflowCfg, nil
}

// BuildMLflowRunRequestContext constructs a fully initialized RequestContext by
// performing API-server-specific validation and then delegating to the common
// BuildRequestContext.
func BuildMLflowRunRequestContext(namespace string, requestCfg *commonmlflow.MLflowPluginConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "cannot build MLflow request context without a resolved config")
	}
	if requestCfg.Endpoint == "" {
		return nil, util.NewInvalidInputError("plugins.mlflow endpoint must be set")
	}
	settings := requestCfg.Settings
	if settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "BuildMLflowRequestContext requires resolved settings")
	}
	workspacesEnabled := settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled
	return commonmlflow.BuildMLflowRequestContext(*requestCfg, namespace, workspacesEnabled)
}

// ResolveMLflowPluginInput parses the plugins_input.mlflow JSON from a run model,
// validates it against the MLflowPluginInput schema, and applies defaults.
func ResolveMLflowPluginInput(pluginsInputString *string) (*MLflowPluginInput, error) {
	defaultInput := &MLflowPluginInput{ExperimentName: DefaultExperimentName}

	if pluginsInputString == nil || *pluginsInputString == "" {
		return defaultInput, nil
	}

	var pluginInputs apiserverPlugins.PluginsInputMap
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	mlflowRaw, ok := pluginInputs["mlflow"]
	if !ok || len(mlflowRaw) == 0 {
		return defaultInput, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(mlflowRaw))
	decoder.DisallowUnknownFields()
	input := &MLflowPluginInput{}
	if err := decoder.Decode(input); err != nil {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must follow schema {experiment_name?: string, experiment_id?: string, disabled?: bool}: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must be a single JSON object")
	}

	if input.Disabled {
		return input, nil
	}
	if input.ExperimentName == "" {
		input.ExperimentName = DefaultExperimentName
	}
	return input, nil
}

// SelectMLflowExperiment chooses the selector used for MLflow experiment resolution.
// Priority: user-provided experiment_id > user-provided experiment_name >
// admin-configured defaultExperimentName > hardcoded "KFP-Default".
func SelectMLflowExperiment(input *MLflowPluginInput, settings *commonmlflow.MLflowPluginSettings) (experimentID string, experimentName string) {
	if input != nil {
		if input.ExperimentID != "" {
			return input.ExperimentID, ""
		}
		if input.ExperimentName != "" {
			return "", input.ExperimentName
		}
	}
	if settings != nil && settings.DefaultExperimentName != "" {
		return "", settings.DefaultExperimentName
	}
	return "", DefaultExperimentName
}

// ToMLflowTerminalStatus converts a KFP RuntimeState string to an MLflow
// terminal status.
func ToMLflowTerminalStatus(stateV2 string) string {
	switch stateV2 {
	case "SUCCEEDED":
		return "FINISHED"
	case "CANCELED", "CANCELING":
		return "KILLED"
	default:
		return "FAILED"
	}
}

// ResolvePluginSettings parses and validates MLflow plugin settings from raw map, and applies default values where
// necessary.
func ResolvePluginSettings(rawSettings map[string]interface{}) *commonmlflow.MLflowPluginSettings {
	var settings commonmlflow.MLflowPluginSettings
	for key, value := range rawSettings {
		switch strings.ToLower(key) {
		case "workspacesenabled":
			settings.WorkspacesEnabled = asBoolPointer(key, value)
		case "experimentdescription":
			settings.ExperimentDescription = asStringPointer(key, value)
		case "defaultexperimentname":
			if s, ok := asString(key, value); ok {
				settings.DefaultExperimentName = s
			}
		case "kfpbaseurl":
			if s, ok := asString(key, value); ok {
				settings.KFPBaseURL = s
			}
		case "kfprunurlpathtemplate":
			if s, ok := asString(key, value); ok {
				settings.KFPRunURLPathTemplate = s
			}
		case "mlflowbaseurl":
			if s, ok := asString(key, value); ok {
				settings.MLflowBaseURL = s
			}
		case "mlflowuipathprefix":
			if s, ok := asString(key, value); ok {
				settings.MLflowUIPathPrefix = s
			}
		case "injectuserenvvars":
			settings.InjectUserEnvVars = asBoolPointer(key, value)
		default:
			glog.Warningf("unrecognized MLflow plugin setting: %s", key)
		}
	}
	return ApplySettingsDefaults(&settings)
}

func asBoolPointer(key string, value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case *bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			glog.Errorf("failed to parse %s as bool from MLflow plugin settings: %v", key, err)
			return nil
		}
		return &parsed
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return nil
	}
}

func asStringPointer(key string, value interface{}) *string {
	if s, ok := asString(key, value); ok {
		return &s
	}
	return nil
}

func asString(key string, value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
		return "", false
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return "", false
	}
}
