package config

import (
	"encoding/json"

	"github.com/golang/glog"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/spf13/viper"
)

const (
	mlflowRunID     = "MLFLOW_RUN_ID"
	kfpMLflowConfig = "KFP_MLFLOW_CONFIG"
)

func GetStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Infof("config %s not set", configName)
	}
	return viper.GetString(configName)
}

func GetMLflowRunID() string {
	return GetStringConfig(mlflowRunID)
}

func GetKfpMLflowRuntimeConfig() string {
	return GetStringConfig(kfpMLflowConfig)
}

func FormatKfpMLflowRuntimeConfig() (commonmlflow.MLflowRuntimeConfig, error) {
	var cfg commonmlflow.MLflowRuntimeConfig
	runtimeCfg := GetKfpMLflowRuntimeConfig()
	if err := json.Unmarshal([]byte(runtimeCfg), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
