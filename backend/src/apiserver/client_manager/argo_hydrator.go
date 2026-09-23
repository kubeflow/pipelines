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

package clientmanager

import (
	"context"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (c *ClientManager) initWorkflowHydrator(ctx context.Context) {
	kube := c.k8sCoreClient.GetClientSet()
	if kube == nil {
		glog.Warning("Skipping Argo offload hydrator: Kubernetes client is unavailable")
		return
	}
	persist, secretsNamespace, err := loadArgoPersistConfig(ctx, kube)
	if err != nil {
		glog.Warningf("Skipping Argo offload hydrator: %v. Retry of offloaded workflows will fail.", err)
		return
	}
	if persist == nil || !persist.NodeStatusOffload {
		glog.Info("Argo node status offload is disabled; using no-op workflow hydrator")
		return
	}
	if err := util.InitWorkflowHydrator(ctx, kube, persist, secretsNamespace); err != nil {
		glog.Warningf("Failed to initialize Argo offload hydrator: %v. Retry of offloaded workflows will fail.", err)
		return
	}
	glog.Info("Argo offload hydrator initialized")
}

func loadArgoPersistConfig(ctx context.Context, kube kubernetes.Interface) (*argoconfig.PersistConfig, string, error) {
	namespace := common.GetArgoWorkflowControllerNamespace()
	configMapName := common.GetArgoWorkflowControllerConfigMap()
	configMap, err := kube.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	persistenceYAML := []byte(configMap.Data["persistence"])
	persist, err := util.ParseArgoPersistConfig(persistenceYAML)
	if err != nil {
		return nil, "", err
	}
	return persist, namespace, nil
}
