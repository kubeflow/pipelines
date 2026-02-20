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

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
)

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }

func TestDefaultContainerSecurityContext(t *testing.T) {
	securityContext := defaultContainerSecurityContext()

	assert.NotNil(t, securityContext)
	assert.NotNil(t, securityContext.AllowPrivilegeEscalation)
	assert.False(t, *securityContext.AllowPrivilegeEscalation)
	assert.NotNil(t, securityContext.RunAsNonRoot)
	assert.True(t, *securityContext.RunAsNonRoot)
	assert.NotNil(t, securityContext.Capabilities)
	assert.Equal(t, []k8score.Capability{"ALL"}, securityContext.Capabilities.Drop)
	assert.NotNil(t, securityContext.SeccompProfile)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, securityContext.SeccompProfile.Type)
	// Should not set fields that are left to the user
	assert.Nil(t, securityContext.RunAsUser)
	assert.Nil(t, securityContext.RunAsGroup)
	assert.Nil(t, securityContext.Privileged)
	assert.Nil(t, securityContext.ReadOnlyRootFilesystem)
}

func TestApplySecurityContextToContainer_NilContainer(t *testing.T) {
	// Should not panic
	applySecurityContextToContainer(nil)
}

func TestApplySecurityContextToContainer_NilSecurityContext(t *testing.T) {
	container := &k8score.Container{Name: "test"}
	applySecurityContextToContainer(container)

	assert.NotNil(t, container.SecurityContext)
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *container.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"ALL"}, container.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, container.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToContainer_PartialSecurityContext(t *testing.T) {
	container := &k8score.Container{
		Name: "test",
		SecurityContext: &k8score.SecurityContext{
			RunAsUser: int64Ptr(1000),
		},
	}
	applySecurityContextToContainer(container)

	// Should preserve existing fields
	assert.Equal(t, int64(1000), *container.SecurityContext.RunAsUser)
	// Should fill in missing defaults
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *container.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"ALL"}, container.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, container.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToContainer_FullSecurityContext(t *testing.T) {
	container := &k8score.Container{
		Name: "test",
		SecurityContext: &k8score.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(true),  // user explicitly set
			RunAsNonRoot:             boolPtr(false), // user explicitly set
			Capabilities: &k8score.Capabilities{
				Drop: []k8score.Capability{"NET_RAW"},
				Add:  []k8score.Capability{"SYS_PTRACE"},
			},
			SeccompProfile: &k8score.SeccompProfile{
				Type: k8score.SeccompProfileTypeUnconfined,
			},
		},
	}
	applySecurityContextToContainer(container)

	// Should not overwrite any fields that are already set
	assert.True(t, *container.SecurityContext.AllowPrivilegeEscalation)
	assert.False(t, *container.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"NET_RAW"}, container.SecurityContext.Capabilities.Drop)
	assert.Equal(t, []k8score.Capability{"SYS_PTRACE"}, container.SecurityContext.Capabilities.Add)
	assert.Equal(t, k8score.SeccompProfileTypeUnconfined, container.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToTemplate_NilTemplate(t *testing.T) {
	// Should not panic
	applySecurityContextToTemplate(nil)
}

func TestApplySecurityContextToTemplate_WithContainer(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{
			Name: "main",
		},
	}
	applySecurityContextToTemplate(template)

	// Pod security context: SeccompProfile and RunAsNonRoot
	assert.NotNil(t, template.SecurityContext)
	assert.NotNil(t, template.SecurityContext.RunAsNonRoot)
	assert.True(t, *template.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, template.SecurityContext.SeccompProfile.Type)

	// Container security context should be set
	assert.NotNil(t, template.Container.SecurityContext)
	assert.False(t, *template.Container.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *template.Container.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"ALL"}, template.Container.SecurityContext.Capabilities.Drop)
}

func TestApplySecurityContextToTemplate_WithInitContainersAndSidecars(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "main"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "init-1"}},
			{Container: k8score.Container{Name: "init-2"}},
		},
		Sidecars: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "sidecar-1"}},
		},
	}
	applySecurityContextToTemplate(template)

	// All init containers should have security context
	for _, initContainer := range template.InitContainers {
		assert.NotNil(t, initContainer.SecurityContext, "init container %s should have security context", initContainer.Name)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)
		assert.Equal(t, []k8score.Capability{"ALL"}, initContainer.SecurityContext.Capabilities.Drop)
	}

	// All sidecars should have security context
	for _, sidecar := range template.Sidecars {
		assert.NotNil(t, sidecar.SecurityContext, "sidecar %s should have security context", sidecar.Name)
		assert.False(t, *sidecar.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *sidecar.SecurityContext.RunAsNonRoot)
		assert.Equal(t, []k8score.Capability{"ALL"}, sidecar.SecurityContext.Capabilities.Drop)
	}
}

func TestApplySecurityContextToUserContainers_Empty(t *testing.T) {
	// Should not panic on empty slice
	applySecurityContextToUserContainers([]wfapi.UserContainer{})
	// Should not panic on nil slice
	applySecurityContextToUserContainers(nil)
}

func TestApplySecurityContextToTemplate_NoContainer(t *testing.T) {
	// Template with DAG (no container)
	template := &wfapi.Template{
		Name: "dag-template",
		DAG:  &wfapi.DAGTemplate{},
	}
	applySecurityContextToTemplate(template)

	// Pod security context: SeccompProfile and RunAsNonRoot
	assert.NotNil(t, template.SecurityContext)
	assert.NotNil(t, template.SecurityContext.RunAsNonRoot)
	assert.True(t, *template.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, template.SecurityContext.SeccompProfile.Type)
}

// --- Executor template tests (user containers) ---

func TestDefaultUserContainerSecurityContext(t *testing.T) {
	securityContext := defaultUserContainerSecurityContext()

	assert.NotNil(t, securityContext)
	assert.NotNil(t, securityContext.AllowPrivilegeEscalation)
	assert.False(t, *securityContext.AllowPrivilegeEscalation)
	assert.NotNil(t, securityContext.Capabilities)
	assert.Equal(t, []k8score.Capability{"ALL"}, securityContext.Capabilities.Drop)
	assert.NotNil(t, securityContext.SeccompProfile)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, securityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot - user images may run as root
	assert.Nil(t, securityContext.RunAsNonRoot)
}

func TestApplyUserContainerSecurityContext_NilContainer(t *testing.T) {
	// Should not panic
	applyUserContainerSecurityContext(nil)
}

func TestApplyUserContainerSecurityContext_NilSecurityContext(t *testing.T) {
	container := &k8score.Container{Name: "user"}
	applyUserContainerSecurityContext(container)

	assert.NotNil(t, container.SecurityContext)
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []k8score.Capability{"ALL"}, container.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, container.SecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, container.SecurityContext.RunAsNonRoot)
}

func TestApplyUserContainerSecurityContext_PartialSecurityContext(t *testing.T) {
	container := &k8score.Container{
		Name: "user",
		SecurityContext: &k8score.SecurityContext{
			RunAsUser: int64Ptr(1000),
		},
	}
	applyUserContainerSecurityContext(container)

	assert.Equal(t, int64(1000), *container.SecurityContext.RunAsUser)
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []k8score.Capability{"ALL"}, container.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, container.SecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, container.SecurityContext.RunAsNonRoot)
}

func TestApplyPodSeccompProfileOnly_NilContext(t *testing.T) {
	var podSecurityContext *k8score.PodSecurityContext
	applyPodSeccompProfileOnly(&podSecurityContext)

	assert.NotNil(t, podSecurityContext)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, podSecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, podSecurityContext.RunAsNonRoot)
}

func TestApplyPodSeccompProfileOnly_ExistingContext(t *testing.T) {
	podSecurityContext := &k8score.PodSecurityContext{
		RunAsUser: int64Ptr(1000),
	}
	applyPodSeccompProfileOnly(&podSecurityContext)

	assert.Equal(t, int64(1000), *podSecurityContext.RunAsUser)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, podSecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, podSecurityContext.RunAsNonRoot)
}

func TestApplySecurityContextToExecutorTemplate_NilTemplate(t *testing.T) {
	// Should not panic
	applySecurityContextToExecutorTemplate(nil, nil, nil, nil)
}

func TestApplySecurityContextToExecutorTemplate_WithContainerAndInitContainers(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "kfp-launcher"}},
		},
		Sidecars: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "sidecar"}},
		},
	}
	applySecurityContextToExecutorTemplate(template, nil, nil, nil)

	// Pod security context: SeccompProfile only, no RunAsNonRoot
	assert.NotNil(t, template.SecurityContext)
	assert.Nil(t, template.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, template.SecurityContext.SeccompProfile.Type)

	// Main container (user): no RunAsNonRoot, but capabilities drop ALL
	assert.NotNil(t, template.Container.SecurityContext)
	assert.Nil(t, template.Container.SecurityContext.RunAsNonRoot)
	assert.False(t, *template.Container.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []k8score.Capability{"ALL"}, template.Container.SecurityContext.Capabilities.Drop)
	// No admin defaults set, so RunAsUser/RunAsGroup should not be set
	assert.Nil(t, template.Container.SecurityContext.RunAsUser)
	assert.Nil(t, template.Container.SecurityContext.RunAsGroup)

	// Init containers (system): full security WITH RunAsNonRoot
	for _, initContainer := range template.InitContainers {
		assert.NotNil(t, initContainer.SecurityContext)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.Equal(t, []k8score.Capability{"ALL"}, initContainer.SecurityContext.Capabilities.Drop)
	}

	// Sidecars: no RunAsNonRoot, but capabilities drop ALL
	for _, sidecar := range template.Sidecars {
		assert.NotNil(t, sidecar.SecurityContext)
		assert.Nil(t, sidecar.SecurityContext.RunAsNonRoot)
		assert.False(t, *sidecar.SecurityContext.AllowPrivilegeEscalation)
		assert.Equal(t, []k8score.Capability{"ALL"}, sidecar.SecurityContext.Capabilities.Drop)
	}
}

func TestApplySecurityContextToExecutorTemplate_WithAdminDefaults(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "kfp-launcher"}},
		},
		Sidecars: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "sidecar"}},
		},
	}
	defaultRunAsUser := int64Ptr(1000)
	defaultRunAsGroup := int64Ptr(100)
	applySecurityContextToExecutorTemplate(template, defaultRunAsUser, defaultRunAsGroup, nil)

	// Main container should have admin defaults applied
	assert.NotNil(t, template.Container.SecurityContext)
	assert.Equal(t, int64(1000), *template.Container.SecurityContext.RunAsUser)
	assert.Equal(t, int64(100), *template.Container.SecurityContext.RunAsGroup)
	// Other security defaults should still be applied
	assert.False(t, *template.Container.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, []k8score.Capability{"ALL"}, template.Container.SecurityContext.Capabilities.Drop)

	// Init containers must NOT have admin defaults
	for _, initContainer := range template.InitContainers {
		assert.Nil(t, initContainer.SecurityContext.RunAsUser)
		assert.Nil(t, initContainer.SecurityContext.RunAsGroup)
		// But should have full system security context
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)
	}

	// Sidecars must NOT have admin defaults
	for _, sidecar := range template.Sidecars {
		assert.Nil(t, sidecar.SecurityContext.RunAsUser)
		assert.Nil(t, sidecar.SecurityContext.RunAsGroup)
	}
}

func TestApplySecurityContextToExecutorTemplate_AdminDefaultsOnlyUser(t *testing.T) {
	// Test that only runAsUser is set when only that admin default is configured
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
	}
	defaultRunAsUser := int64Ptr(1000)
	applySecurityContextToExecutorTemplate(template, defaultRunAsUser, nil, nil)

	assert.Equal(t, int64(1000), *template.Container.SecurityContext.RunAsUser)
	assert.Nil(t, template.Container.SecurityContext.RunAsGroup)
}

func TestApplySecurityContextToExecutorTemplate_AdminDefaultsOnlyGroup(t *testing.T) {
	// Test that only runAsGroup is set when only that admin default is configured
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
	}
	defaultRunAsGroup := int64Ptr(100)
	applySecurityContextToExecutorTemplate(template, nil, defaultRunAsGroup, nil)

	assert.Nil(t, template.Container.SecurityContext.RunAsUser)
	assert.Equal(t, int64(100), *template.Container.SecurityContext.RunAsGroup)
}

func TestApplySecurityContextToExecutorTemplate_AdminDefaultsOverrideSDK(t *testing.T) {
	// When admin defaults are set, they override any SDK-specified values
	template := &wfapi.Template{
		Container: &k8score.Container{
			Name: "user-container",
			SecurityContext: &k8score.SecurityContext{
				RunAsUser:  int64Ptr(2000),
				RunAsGroup: int64Ptr(200),
			},
		},
	}
	defaultRunAsUser := int64Ptr(1000)
	defaultRunAsGroup := int64Ptr(100)
	applySecurityContextToExecutorTemplate(template, defaultRunAsUser, defaultRunAsGroup, nil)

	// Admin defaults should override SDK-specified values
	assert.Equal(t, int64(1000), *template.Container.SecurityContext.RunAsUser)
	assert.Equal(t, int64(100), *template.Container.SecurityContext.RunAsGroup)
}

func TestApplySecurityContextToExecutorTemplate_AdminDefaultRunAsNonRoot(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "kfp-launcher"}},
		},
	}
	defaultRunAsNonRoot := boolPtr(true)
	applySecurityContextToExecutorTemplate(template, nil, nil, defaultRunAsNonRoot)

	// Main container should have RunAsNonRoot set by admin default
	assert.NotNil(t, template.Container.SecurityContext)
	assert.NotNil(t, template.Container.SecurityContext.RunAsNonRoot)
	assert.True(t, *template.Container.SecurityContext.RunAsNonRoot)
	// RunAsUser/RunAsGroup should not be set since no admin defaults were given
	assert.Nil(t, template.Container.SecurityContext.RunAsUser)
	assert.Nil(t, template.Container.SecurityContext.RunAsGroup)

	// Init containers should NOT have admin RunAsNonRoot (they get system defaults)
	for _, initContainer := range template.InitContainers {
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot, "init container gets system RunAsNonRoot=true")
	}
}

func TestApplySecurityContextToExecutorTemplate_AdminDefaultRunAsNonRootFalse(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
	}
	defaultRunAsNonRoot := boolPtr(false)
	applySecurityContextToExecutorTemplate(template, nil, nil, defaultRunAsNonRoot)

	// Admin explicitly set RunAsNonRoot=false
	assert.NotNil(t, template.Container.SecurityContext.RunAsNonRoot)
	assert.False(t, *template.Container.SecurityContext.RunAsNonRoot)
}

func TestApplySecurityContextToExecutorTemplate_AllAdminDefaults(t *testing.T) {
	template := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
	}
	defaultRunAsUser := int64Ptr(1000)
	defaultRunAsGroup := int64Ptr(100)
	defaultRunAsNonRoot := boolPtr(true)
	applySecurityContextToExecutorTemplate(template, defaultRunAsUser, defaultRunAsGroup, defaultRunAsNonRoot)

	assert.Equal(t, int64(1000), *template.Container.SecurityContext.RunAsUser)
	assert.Equal(t, int64(100), *template.Container.SecurityContext.RunAsGroup)
	assert.True(t, *template.Container.SecurityContext.RunAsNonRoot)
}
