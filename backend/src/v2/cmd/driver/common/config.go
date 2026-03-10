package common

import (
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	// MLflow endpoint for Go REST calls. Set on driver, launcher.
	kfpMLflowTrackingURI = "KFP_MLFLOW_TRACKING_URI"

	// MLflow workspace (namespace) for Go REST calls. Set on driver, launcher.
	//todo: where is this included in the rest request?
	kfpMLflowWorkspace = "KFP_MLFLOW_WORKSPACE"

	// Parent MLflow run ID. Set on driver, launcher.
	kfpMLflowParentRunID = "KFP_MLFLOW_PARENT_RUN_ID"

	// Auth mode. Can be one of: kubernetes, bearer, basic-auth. Set on driver. WHY NOT ON LAUNCHER?
	kfpMLflowAuthType = "KFP_MLFLOW_AUTH_TYPE"

	// Bearer token (secret-based bearer mode). Set on all: driver, launcher, user container.
	MLflowTrackingToken = "MLFLOW_TRACKING_TOKEN"

	// Username (secret-based basic-auth mode). Set on all: driver, launcher, user container.
	MLflowTrackingUsername = "MLFLOW_TRACKING_USERNAME"

	// Password (secret-based basic-auth mode). Set on all: driver, launcher, user container.
	MLFlowTrackingPassword = "MLFLOW_TRACKING_PASSWORD"

	// Standard MLflow SDK env var. The driver sets this onto the launcher (-> user container) thru podSpecPatch.
	MLflowTrackingURI = "MLFLOW_TRACKING_URI"

	// Standard MLflwo env env var (3.10+). The driver sets this onto the launcher (-> user container) thru podSpecPatch.
	MLflowWorkspace = "MLFLOW_WORKSPACE"

	// Active nested run for MLflow SDK (3.10+).
	MlflowRunID = "MLFLOW_RUN_ID"

	// Auth provider (kubernetes, when available). The driver sets this onto the launcher (-> user container) thru podSpecPatch.
	MLflowTrackingAuthType = "MLFLOW_TRACKING_AUTH"
)

func GetStringConfigWithDefault(configName, value string) string {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetString(configName)
}

func GetStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		//todo: copied the below err message from backend/src/apiserver/common/config.go. But should potentially say "env var" instead of "flag".
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}

func GetKfpMLflowTrackingURI() string {
	return GetStringConfigWithDefault(kfpMLflowTrackingURI, "")
}

func GetKfpMLflowWorkspace() string {
	return GetStringConfigWithDefault(kfpMLflowWorkspace, "")
}

func GetKfpMLflowParentRunID() string {
	return GetStringConfigWithDefault(kfpMLflowParentRunID, "")
}

func GetKfpMLflowAuthType() string {
	return GetStringConfigWithDefault(kfpMLflowAuthType, DefaultKfpAuthType)
}

func GetMLflowTrackingToken() string {
	//todo: we do not want to error out if no mlflow tracking token is set. or we can error out as long is it doesn't shut system down.
	return GetStringConfigWithDefault(MLflowTrackingToken, "")
}

func GetMLflowTrackingUsername() string {
	return GetStringConfigWithDefault(MLflowTrackingUsername, "")
}

func GetMLflowTrackingPassword() string {
	return GetStringConfigWithDefault(MLFlowTrackingPassword, "")
}

func GetMLflowTrackingURI() string {
	return GetStringConfigWithDefault(MLflowTrackingURI, DefaultMLflowTrackingURI)
}

func GetMLflowWorkspace() string {
	return GetStringConfigWithDefault(MLflowWorkspace, DefaultMLflowWorkspace)
}

func GetMLflowParentRunId() string {
	return GetStringConfigWithDefault(MlflowRunID, "")
}

func SetMLflowParentRunId(runId string) {
	viper.Set(MlflowRunID, runId)
}

func GetMLflowTrackingAuthType() string {
	return GetStringConfigWithDefault(MLflowTrackingAuthType, "")
}
