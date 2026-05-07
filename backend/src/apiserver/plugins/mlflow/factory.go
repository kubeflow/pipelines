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
	"github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
)

func init() {
	plugins.RegisterHandlerFactory(&mlflowHandlerFactory{})
}

type mlflowHandlerFactory struct{}

func (f *mlflowHandlerFactory) Name() string {
	return "MLflow"
}

func (f *mlflowHandlerFactory) IsEnabled() bool {
	return IsEnabled()
}

func (f *mlflowHandlerFactory) Create() (plugins.RunPluginHandler, error) {
	_, ok, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return NewMLflowRunHandler(), nil
}
