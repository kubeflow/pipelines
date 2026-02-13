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
	sc := defaultContainerSecurityContext()

	assert.NotNil(t, sc)
	assert.NotNil(t, sc.AllowPrivilegeEscalation)
	assert.False(t, *sc.AllowPrivilegeEscalation)
	assert.NotNil(t, sc.RunAsNonRoot)
	assert.True(t, *sc.RunAsNonRoot)
	assert.NotNil(t, sc.Capabilities)
	assert.Equal(t, []k8score.Capability{"ALL"}, sc.Capabilities.Drop)
	assert.NotNil(t, sc.SeccompProfile)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, sc.SeccompProfile.Type)
	// Should not set fields that are left to the user
	assert.Nil(t, sc.RunAsUser)
	assert.Nil(t, sc.RunAsGroup)
	assert.Nil(t, sc.Privileged)
	assert.Nil(t, sc.ReadOnlyRootFilesystem)
}

func TestApplySecurityContextToContainer_NilContainer(t *testing.T) {
	// Should not panic
	applySecurityContextToContainer(nil)
}

func TestApplySecurityContextToContainer_NilSecurityContext(t *testing.T) {
	c := &k8score.Container{Name: "test"}
	applySecurityContextToContainer(c)

	assert.NotNil(t, c.SecurityContext)
	assert.False(t, *c.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *c.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"ALL"}, c.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, c.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToContainer_PartialSecurityContext(t *testing.T) {
	c := &k8score.Container{
		Name: "test",
		SecurityContext: &k8score.SecurityContext{
			RunAsUser: int64Ptr(1000),
		},
	}
	applySecurityContextToContainer(c)

	// Should preserve existing fields
	assert.Equal(t, int64(1000), *c.SecurityContext.RunAsUser)
	// Should fill in missing defaults
	assert.False(t, *c.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *c.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"ALL"}, c.SecurityContext.Capabilities.Drop)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, c.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToContainer_FullSecurityContext(t *testing.T) {
	c := &k8score.Container{
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
	applySecurityContextToContainer(c)

	// Should not overwrite any fields that are already set
	assert.True(t, *c.SecurityContext.AllowPrivilegeEscalation)
	assert.False(t, *c.SecurityContext.RunAsNonRoot)
	assert.Equal(t, []k8score.Capability{"NET_RAW"}, c.SecurityContext.Capabilities.Drop)
	assert.Equal(t, []k8score.Capability{"SYS_PTRACE"}, c.SecurityContext.Capabilities.Add)
	assert.Equal(t, k8score.SeccompProfileTypeUnconfined, c.SecurityContext.SeccompProfile.Type)
}

func TestApplySecurityContextToTemplate_NilTemplate(t *testing.T) {
	// Should not panic
	applySecurityContextToTemplate(nil)
}

func TestApplySecurityContextToTemplate_WithContainer(t *testing.T) {
	tmpl := &wfapi.Template{
		Container: &k8score.Container{
			Name: "main",
		},
	}
	applySecurityContextToTemplate(tmpl)

	// Pod security context: SeccompProfile and RunAsNonRoot
	assert.NotNil(t, tmpl.SecurityContext)
	assert.NotNil(t, tmpl.SecurityContext.RunAsNonRoot)
	assert.True(t, *tmpl.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, tmpl.SecurityContext.SeccompProfile.Type)

	// Container security context should be set
	assert.NotNil(t, tmpl.Container.SecurityContext)
	assert.False(t, *tmpl.Container.SecurityContext.AllowPrivilegeEscalation)
	assert.True(t, *tmpl.Container.SecurityContext.RunAsNonRoot)
}

func TestApplySecurityContextToTemplate_WithInitContainersAndSidecars(t *testing.T) {
	tmpl := &wfapi.Template{
		Container: &k8score.Container{Name: "main"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "init-1"}},
			{Container: k8score.Container{Name: "init-2"}},
		},
		Sidecars: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "sidecar-1"}},
		},
	}
	applySecurityContextToTemplate(tmpl)

	// All init containers should have security context
	for _, ic := range tmpl.InitContainers {
		assert.NotNil(t, ic.SecurityContext, "init container %s should have security context", ic.Name)
		assert.False(t, *ic.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *ic.SecurityContext.RunAsNonRoot)
	}

	// All sidecars should have security context
	for _, sc := range tmpl.Sidecars {
		assert.NotNil(t, sc.SecurityContext, "sidecar %s should have security context", sc.Name)
		assert.False(t, *sc.SecurityContext.AllowPrivilegeEscalation)
		assert.True(t, *sc.SecurityContext.RunAsNonRoot)
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
	tmpl := &wfapi.Template{
		Name: "dag-template",
		DAG:  &wfapi.DAGTemplate{},
	}
	applySecurityContextToTemplate(tmpl)

	// Pod security context: SeccompProfile and RunAsNonRoot
	assert.NotNil(t, tmpl.SecurityContext)
	assert.NotNil(t, tmpl.SecurityContext.RunAsNonRoot)
	assert.True(t, *tmpl.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, tmpl.SecurityContext.SeccompProfile.Type)
}

// --- Executor template tests (user containers) ---

func TestDefaultUserContainerSecurityContext(t *testing.T) {
	sc := defaultUserContainerSecurityContext()

	assert.NotNil(t, sc)
	assert.NotNil(t, sc.AllowPrivilegeEscalation)
	assert.False(t, *sc.AllowPrivilegeEscalation)
	assert.NotNil(t, sc.SeccompProfile)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, sc.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot - user images may run as root
	assert.Nil(t, sc.RunAsNonRoot)
	// Must NOT drop capabilities - user images may need them (e.g. pip install as root)
	assert.Nil(t, sc.Capabilities)
}

func TestApplyUserContainerSecurityContext_NilContainer(t *testing.T) {
	// Should not panic
	applyUserContainerSecurityContext(nil)
}

func TestApplyUserContainerSecurityContext_NilSecurityContext(t *testing.T) {
	c := &k8score.Container{Name: "user"}
	applyUserContainerSecurityContext(c)

	assert.NotNil(t, c.SecurityContext)
	assert.False(t, *c.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, c.SecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, c.SecurityContext.RunAsNonRoot)
	// Must NOT drop capabilities - user images may need them
	assert.Nil(t, c.SecurityContext.Capabilities)
}

func TestApplyUserContainerSecurityContext_PartialSecurityContext(t *testing.T) {
	c := &k8score.Container{
		Name: "user",
		SecurityContext: &k8score.SecurityContext{
			RunAsUser: int64Ptr(1000),
		},
	}
	applyUserContainerSecurityContext(c)

	assert.Equal(t, int64(1000), *c.SecurityContext.RunAsUser)
	assert.False(t, *c.SecurityContext.AllowPrivilegeEscalation)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, c.SecurityContext.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, c.SecurityContext.RunAsNonRoot)
	// Must NOT drop capabilities - user images may need them
	assert.Nil(t, c.SecurityContext.Capabilities)
}

func TestApplyPodSeccompProfileOnly_NilContext(t *testing.T) {
	var sc *k8score.PodSecurityContext
	applyPodSeccompProfileOnly(&sc)

	assert.NotNil(t, sc)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, sc.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, sc.RunAsNonRoot)
}

func TestApplyPodSeccompProfileOnly_ExistingContext(t *testing.T) {
	sc := &k8score.PodSecurityContext{
		RunAsUser: int64Ptr(1000),
	}
	applyPodSeccompProfileOnly(&sc)

	assert.Equal(t, int64(1000), *sc.RunAsUser)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, sc.SeccompProfile.Type)
	// Must NOT set RunAsNonRoot
	assert.Nil(t, sc.RunAsNonRoot)
}

func TestApplySecurityContextToExecutorTemplate_NilTemplate(t *testing.T) {
	// Should not panic
	applySecurityContextToExecutorTemplate(nil)
}

func TestApplySecurityContextToExecutorTemplate_WithContainerAndInitContainers(t *testing.T) {
	tmpl := &wfapi.Template{
		Container: &k8score.Container{Name: "user-container"},
		InitContainers: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "kfp-launcher"}},
		},
		Sidecars: []wfapi.UserContainer{
			{Container: k8score.Container{Name: "sidecar"}},
		},
	}
	applySecurityContextToExecutorTemplate(tmpl)

	// Pod security context: SeccompProfile only, no RunAsNonRoot
	assert.NotNil(t, tmpl.SecurityContext)
	assert.Nil(t, tmpl.SecurityContext.RunAsNonRoot)
	assert.Equal(t, k8score.SeccompProfileTypeRuntimeDefault, tmpl.SecurityContext.SeccompProfile.Type)

	// Main container (user): no RunAsNonRoot, no capabilities drop
	assert.NotNil(t, tmpl.Container.SecurityContext)
	assert.Nil(t, tmpl.Container.SecurityContext.RunAsNonRoot)
	assert.False(t, *tmpl.Container.SecurityContext.AllowPrivilegeEscalation)
	assert.Nil(t, tmpl.Container.SecurityContext.Capabilities)

	// Init containers (system): full security WITH RunAsNonRoot
	for _, ic := range tmpl.InitContainers {
		assert.NotNil(t, ic.SecurityContext)
		assert.True(t, *ic.SecurityContext.RunAsNonRoot)
		assert.False(t, *ic.SecurityContext.AllowPrivilegeEscalation)
	}

	// Sidecars: no RunAsNonRoot, no capabilities drop (could be user-specified)
	for _, sc := range tmpl.Sidecars {
		assert.NotNil(t, sc.SecurityContext)
		assert.Nil(t, sc.SecurityContext.RunAsNonRoot)
		assert.False(t, *sc.SecurityContext.AllowPrivilegeEscalation)
		assert.Nil(t, sc.SecurityContext.Capabilities)
	}
}
