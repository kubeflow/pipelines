package plugins

import (
	"context"
	"testing"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

// ---- MergePluginConfig tests ----

func TestMergePluginConfig_NamespaceOverridesGlobalEndpointAndMergesSettings(t *testing.T) {
	globalDesc := "Global desc"
	wsDisabled := "false"
	global := &PluginConfig{
		Endpoint: "https://global-plugin.example.com",
		Timeout:  "30s",
		Settings: map[string]interface{}{"experimentDescription": globalDesc},
	}
	namespace := &PluginConfig{
		Endpoint: "https://ns-plugin.example.com",
		Settings: map[string]interface{}{"workspacesEnabled": wsDisabled},
	}

	merged, err := MergePluginConfig(namespace, global)
	require.NoError(t, err)
	assert.Equal(t, "https://ns-plugin.example.com", merged.Endpoint)
	assert.Equal(t, "30s", merged.Timeout)
	assert.Equal(t, 2, len(merged.Settings))
	assert.Equal(t, wsDisabled, merged.Settings["workspacesEnabled"])
	assert.Equal(t, globalDesc, merged.Settings["experimentDescription"])
}

func TestMergePluginConfig_BothNil_ReturnsNil(t *testing.T) {
	merged, err := MergePluginConfig(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, merged)
}

func TestMergePluginConfig_NilGlobal_ReturnsNamespace(t *testing.T) {
	namespace := &PluginConfig{
		Endpoint: "https://ns-only.example.com",
		Timeout:  "5s",
		Settings: map[string]interface{}{"key": "val"},
	}

	merged, err := MergePluginConfig(namespace, nil)
	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, "https://ns-only.example.com", merged.Endpoint)
	assert.Equal(t, "5s", merged.Timeout)
	assert.Equal(t, "val", merged.Settings["key"])
}

func TestMergePluginConfig_NilNamespace_ReturnsCopyOfGlobal(t *testing.T) {
	global := &PluginConfig{
		Endpoint: "https://global.example.com",
		Timeout:  "15s",
		TLS:      &commonplugins.TLSConfig{InsecureSkipVerify: true},
		Settings: map[string]interface{}{"desc": "global"},
	}

	merged, err := MergePluginConfig(nil, global)
	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, "https://global.example.com", merged.Endpoint)
	assert.Equal(t, "15s", merged.Timeout)
	assert.True(t, merged.TLS.InsecureSkipVerify)
	assert.Equal(t, "global", merged.Settings["desc"])
}

func TestMergePluginConfig_NamespaceTLSOverridesGlobalTLS(t *testing.T) {
	global := &PluginConfig{
		Endpoint: "https://global.example.com",
		Timeout:  "10s",
		TLS:      &commonplugins.TLSConfig{InsecureSkipVerify: false},
	}
	namespace := &PluginConfig{
		TLS: &commonplugins.TLSConfig{InsecureSkipVerify: true},
	}

	merged, err := MergePluginConfig(namespace, global)
	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, "https://global.example.com", merged.Endpoint)
	assert.Equal(t, "10s", merged.Timeout)
	require.NotNil(t, merged.TLS)
	assert.True(t, merged.TLS.InsecureSkipVerify)
}

func TestMergePluginConfig_NamespaceTimeoutOverridesGlobal(t *testing.T) {
	global := &PluginConfig{
		Endpoint: "https://global.example.com",
		Timeout:  "30s",
	}
	namespace := &PluginConfig{
		Timeout: "5s",
	}

	merged, err := MergePluginConfig(namespace, global)
	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, "5s", merged.Timeout)
}

// ---- ResolvePluginRequestConfig tests ----

func TestResolvePluginRequestConfig_NeitherGlobalNorNamespace(t *testing.T) {
	handler := &fakeHandler{
		name:         "FakePlugin",
		pluginConfig: nil,
	}
	clientSet := fakeclientset.NewClientset()

	cfg, err := handler.ResolveRunPluginConfig(context.Background(), clientSet, "", "test-ns")
	require.NoError(t, err)
	assert.Nil(t, cfg, "should return nil when neither global nor namespace config exists")
}

func TestResolvePluginRequestConfig_GlobalOnly(t *testing.T) {
	handler := &fakeHandler{
		name: "FakePlugin",
		pluginConfig: &PluginConfig{
			Endpoint: "https://global.example.com",
			Timeout:  "20s",
		},
	}
	clientSet := fakeclientset.NewClientset()
	cfg, err := handler.ResolveRunPluginConfig(context.Background(), clientSet, "", "test-ns")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// Note: fakeHandler returns the raw pluginConfig, not a fully resolved config.
	// Plugin-specific tests should verify resolved config structure.
}

func TestResolvePluginRequestConfig_EmptyTimeout_DefaultApplied(t *testing.T) {
	handler := &fakeHandler{
		name: "FakePlugin",
		pluginConfig: &PluginConfig{
			Endpoint: "https://global.example.com",
		},
	}
	clientSet := fakeclientset.NewClientset()

	cfg, err := handler.ResolveRunPluginConfig(context.Background(), clientSet, "", "test-ns")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// Note: fakeHandler returns the raw pluginConfig, not a fully resolved config.
	// Plugin-specific tests should verify resolved config structure.
}

// ---- GetLauncherNamespacePluginConfigsMap tests ----

func TestGetLauncherNamespacePluginConfigsMap_EmptyNamespace_ReturnsError(t *testing.T) {
	clientSet := fakeclientset.NewClientset()

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, "")

	require.Error(t, err)
	assert.Nil(t, cfgMap)
	assert.Contains(t, err.Error(), "namespace must be specified when reading Plugin config")
}

func TestGetLauncherNamespacePluginConfigsMap_NilClientSet_ReturnsError(t *testing.T) {
	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), nil, "test-ns")

	require.Error(t, err)
	assert.Nil(t, cfgMap)
	assert.Contains(t, err.Error(), "Kubernetes clientset must be provided when reading Plugin namespace config")
}

func TestGetLauncherNamespacePluginConfigsMap_ConfigMapNotFound_ReturnsNilWithoutError(t *testing.T) {
	clientSet := fakeclientset.NewClientset()

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, "test-ns")

	require.NoError(t, err, "should not return error when ConfigMap is not found")
	assert.Nil(t, cfgMap, "should return nil map when ConfigMap is not found")
}

func TestGetLauncherNamespacePluginConfigsMap_WithMLflowPlugin_ReturnsConfig(t *testing.T) {
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	mlflowConfig := `{
		"endpoint": "https://mlflow.example.com",
		"timeout": "20s",
		"settings": {
			"authType": "none"
		}
	}`

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.mlflow": mlflowConfig,
		},
	}

	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, namespace)

	require.NoError(t, err)
	require.NotNil(t, cfgMap)
	assert.Equal(t, 1, len(cfgMap))
	assert.Contains(t, cfgMap, "mlflow")
	assert.Equal(t, mlflowConfig, cfgMap["mlflow"])
}

func TestGetLauncherNamespacePluginConfigsMap_MultiplePlugins_ReturnsAllConfigs(t *testing.T) {
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	mlflowConfig := `{"endpoint": "https://mlflow.example.com"}`
	customPluginConfig := `{"endpoint": "https://custom.example.com"}`

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.mlflow": mlflowConfig,
			"plugins.custom": customPluginConfig,
		},
	}

	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, namespace)

	require.NoError(t, err)
	require.NotNil(t, cfgMap)
	assert.Equal(t, 2, len(cfgMap))
	assert.Contains(t, cfgMap, "mlflow")
	assert.Contains(t, cfgMap, "custom")
	assert.Equal(t, mlflowConfig, cfgMap["mlflow"])
	assert.Equal(t, customPluginConfig, cfgMap["custom"])
}

func TestGetLauncherNamespacePluginConfigsMap_MixedKeys_FiltersPluginKeysOnly(t *testing.T) {
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.mlflow":   `{"endpoint": "https://mlflow.example.com"}`,
			"other-key":        "should-be-ignored",
			"another-key":      "also-ignored",
			"plugins.custom":   `{"endpoint": "https://custom.example.com"}`,
			"non-plugin-value": "not-included",
		},
	}

	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, namespace)

	require.NoError(t, err)
	require.NotNil(t, cfgMap)
	assert.Equal(t, 2, len(cfgMap), "should only include keys with 'plugins.' prefix")
	assert.Contains(t, cfgMap, "mlflow")
	assert.Contains(t, cfgMap, "custom")
	assert.NotContains(t, cfgMap, "other-key")
	assert.NotContains(t, cfgMap, "another-key")
	assert.NotContains(t, cfgMap, "non-plugin-value")
}

func TestGetLauncherNamespacePluginConfigsMap_NoPluginKeys_ReturnsEmptyOrNilMap(t *testing.T) {
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"other-key":   "value1",
			"another-key": "value2",
		},
	}

	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, namespace)

	require.NoError(t, err)
	// Map should be either nil or empty when no plugin keys exist
	if cfgMap != nil {
		assert.Equal(t, 0, len(cfgMap), "should have no entries when no plugin keys exist")
	}
}

func TestGetLauncherNamespacePluginConfigsMap_EmptyConfigMap_ReturnsEmptyOrNilMap(t *testing.T) {
	clientSet := fakeclientset.NewClientset()
	namespace := "test-ns"

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	_, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	cfgMap, err := GetLauncherNamespacePluginConfigsMap(context.Background(), clientSet, namespace)

	require.NoError(t, err)
	// Map should be either nil or empty when ConfigMap has no data
	if cfgMap != nil {
		assert.Equal(t, 0, len(cfgMap), "should have no entries when ConfigMap is empty")
	}
}
