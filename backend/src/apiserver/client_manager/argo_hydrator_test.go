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
	"testing"

	argoconfig "github.com/argoproj/argo-workflows/v4/config"
	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type stubKubernetesCore struct {
	client kubernetes.Interface
}

func (s stubKubernetesCore) PodClient(namespace string) v1.PodInterface {
	return s.client.CoreV1().Pods(namespace)
}

func (s stubKubernetesCore) GetClientSet() kubernetes.Interface {
	return s.client
}

func TestLoadArgoPersistConfig(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	kube := k8sfake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-controller-configmap",
			Namespace: "kubeflow-pipelines",
		},
		Data: map[string]string{
			"persistence": `archive: true
nodeStatusOffLoad: true
clusterName: default
postgresql:
  host: postgres.example.invalid
  port: 5432
  database: argo
  tableName: argo_workflows
  userNameSecret:
    name: argo-postgres-config
    key: username
  passwordSecret:
    name: argo-postgres-config
    key: password
`,
		},
	})

	persist, namespace, err := loadArgoPersistConfig(context.Background(), kube)
	require.NoError(t, err)
	assert.Equal(t, "kubeflow-pipelines", namespace)
	require.NotNil(t, persist)
	assert.True(t, persist.NodeStatusOffload)
	require.NotNil(t, persist.PostgreSQL)
	assert.Equal(t, "argo_workflows", persist.PostgreSQL.TableName)
	assert.Equal(t, []string{"argo-postgres-config"}, util.ArgoPersistSecretNames(persist))
}

func TestLoadArgoPersistConfig_MissingPersistenceKey(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	kube := k8sfake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-controller-configmap",
			Namespace: "kubeflow-pipelines",
		},
		Data: map[string]string{
			"containerRuntimeExecutor": "emissary",
		},
	})

	persist, namespace, err := loadArgoPersistConfig(context.Background(), kube)
	require.NoError(t, err)
	assert.Equal(t, "kubeflow-pipelines", namespace)
	assert.Nil(t, persist)
}

func TestLoadArgoPersistConfig_FromDataConfig(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	kube := k8sfake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-controller-configmap",
			Namespace: "kubeflow-pipelines",
		},
		Data: map[string]string{
			"config": `persistence:
  archive: true
  nodeStatusOffLoad: true
  clusterName: default
  postgresql:
    host: postgres.example.invalid
    port: 5432
    database: argo
    tableName: argo_workflows
    userNameSecret:
      name: argo-postgres-config
      key: username
    passwordSecret:
      name: argo-postgres-config
      key: password
`,
		},
	})

	persist, namespace, err := loadArgoPersistConfig(context.Background(), kube)
	require.NoError(t, err)
	assert.Equal(t, "kubeflow-pipelines", namespace)
	require.NotNil(t, persist)
	assert.True(t, persist.NodeStatusOffload)
	require.NotNil(t, persist.PostgreSQL)
	assert.Equal(t, "argo_workflows", persist.PostgreSQL.TableName)
}

func TestLoadArgoPersistConfig_MalformedConfigWithExtraKeys(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	kube := k8sfake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-controller-configmap",
			Namespace: "kubeflow-pipelines",
		},
		Data: map[string]string{
			"config":   "persistence: {}",
			"extraKey": "value",
		},
	})

	_, _, err := loadArgoPersistConfig(context.Background(), kube)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must only have one item")
}

func TestLoadArgoPersistConfig_MissingConfigMap(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.AutomaticEnv()

	_, _, err := loadArgoPersistConfig(context.Background(), k8sfake.NewClientset())
	require.Error(t, err)
}

func TestTryInitLazyOffloadHydrator_DisabledWhenPersistenceAbsent(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	kube := k8sfake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-controller-configmap",
			Namespace: "kubeflow-pipelines",
		},
		Data: map[string]string{
			"containerRuntimeExecutor": "emissary",
		},
	})
	lazy := util.NewLazyOffloadHydrator(kube, func(ctx context.Context) (persistCfg *argoconfig.PersistConfig, ns string, err error) {
		return loadArgoPersistConfig(ctx, kube)
	})

	offloadEnabled, err := util.TryInitLazyOffloadHydrator(lazy, context.Background())
	require.NoError(t, err)
	assert.False(t, offloadEnabled)
}

func TestInitWorkflowHydrator_InstallsLazyWhenConfigMapMissing(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set(common.PodNamespace, "kubeflow-pipelines")
	viper.Set(common.ArgoWorkflowControllerConfigMap, "workflow-controller-configmap")
	viper.AutomaticEnv()

	previous := util.CurrentWorkflowHydrator()
	util.SetWorkflowHydrator(hydratorfake.Noop)
	t.Cleanup(func() {
		util.SetWorkflowHydrator(previous)
	})

	cm := &ClientManager{
		k8sCoreClient: stubKubernetesCore{client: k8sfake.NewClientset()},
	}
	cm.initWorkflowHydrator(context.Background())

	h := util.CurrentWorkflowHydrator()
	require.NotSame(t, hydratorfake.Noop, h)

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{
				"ok": {ID: "ok", Name: "my-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
			},
		},
	}
	require.NoError(t, h.Hydrate(context.Background(), wf))

	offloaded := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "wf-uid"},
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "offload-hash",
		},
	}
	err := h.Hydrate(context.Background(), offloaded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}
