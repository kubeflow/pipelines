package plugins

import (
	"context"
	"encoding/json"
	"testing"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	cfg, err := ResolvePluginRequestConfig(context.Background(), clientSet, handler, "test-ns")
	require.NoError(t, err)
	assert.Nil(t, cfg, "should return nil when neither global nor namespace config exists")
}

func TestResolvePluginRequestConfig_NamespaceOnlyWithoutGlobal(t *testing.T) {
	handler := &fakeHandler{
		name:         "FakePlugin",
		pluginConfig: nil,
	}

	namespaceCfg := PluginConfig{
		Endpoint: "https://ns-endpoint.example.com",
		Settings: map[string]interface{}{"key": "ns-val"},
	}
	namespaceCfgJSON, err := json.Marshal(namespaceCfg)
	require.NoError(t, err)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      LauncherConfigMapName,
			Namespace: "test-ns",
		},
		Data: map[string]string{
			LauncherConfigKeyPrefix + "FakePlugin": string(namespaceCfgJSON),
		},
	}
	clientSet := fakeclientset.NewClientset(configMap)

	cfg, err := ResolvePluginRequestConfig(context.Background(), clientSet, handler, "test-ns")
	require.NoError(t, err)
	require.NotNil(t, cfg, "namespace config should be returned even without global config")
	assert.Equal(t, "https://ns-endpoint.example.com", cfg.Endpoint)
	assert.Equal(t, DefaultTimeout, cfg.Timeout, "empty timeout should be filled with default")
	assert.Equal(t, "ns-val", cfg.Settings["key"])
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

	cfg, err := ResolvePluginRequestConfig(context.Background(), clientSet, handler, "test-ns")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "https://global.example.com", cfg.Endpoint)
	assert.Equal(t, "20s", cfg.Timeout)
}

func TestResolvePluginRequestConfig_EmptyTimeout_DefaultApplied(t *testing.T) {
	handler := &fakeHandler{
		name: "FakePlugin",
		pluginConfig: &PluginConfig{
			Endpoint: "https://global.example.com",
		},
	}
	clientSet := fakeclientset.NewClientset()

	cfg, err := ResolvePluginRequestConfig(context.Background(), clientSet, handler, "test-ns")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, DefaultTimeout, cfg.Timeout)
}

func TestResolvePluginRequestConfig_NilHandler_ReturnsError(t *testing.T) {
	clientSet := fakeclientset.NewClientset()

	cfg, err := ResolvePluginRequestConfig(context.Background(), clientSet, nil, "test-ns")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "handler is nil")
}
