// Package all imports all plugin packages to trigger factory registration via init().
// Binaries that use plugins.GetPluginDispatcher should blank-import this package.
package all

import (
	_ "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/mlflow"
)
