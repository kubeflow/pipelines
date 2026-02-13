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
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	k8score "k8s.io/api/core/v1"
)

func defaultContainerSecurityContext() *k8score.SecurityContext {
	allowPrivilegeEscalation := false
	runAsNonRoot := true
	return &k8score.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		RunAsNonRoot:             &runAsNonRoot,
		Capabilities: &k8score.Capabilities{
			Drop: []k8score.Capability{"ALL"},
		},
		SeccompProfile: &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func applySecurityContextToTemplate(t *wfapi.Template) {
	if t == nil {
		return
	}
	// Apply SeccompProfile and RunAsNonRoot at the pod level.
	// Argo Workflows is designed to run rootless:
	// https://argo-workflows.readthedocs.io/en/latest/workflow-pod-security-context/
	applyPodSecurityDefaults(&t.SecurityContext)
	applySecurityContextToContainer(t.Container)
	applySecurityContextToUserContainers(t.InitContainers)
	applySecurityContextToUserContainers(t.Sidecars)
}

func applySecurityContextToContainer(c *k8score.Container) {
	if c == nil {
		return
	}
	if c.SecurityContext == nil {
		c.SecurityContext = defaultContainerSecurityContext()
		return
	}
	if c.SecurityContext.AllowPrivilegeEscalation == nil {
		allowPrivilegeEscalation := false
		c.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
	}
	if c.SecurityContext.RunAsNonRoot == nil {
		runAsNonRoot := true
		c.SecurityContext.RunAsNonRoot = &runAsNonRoot
	}
	if c.SecurityContext.Capabilities == nil {
		c.SecurityContext.Capabilities = &k8score.Capabilities{
			Drop: []k8score.Capability{"ALL"},
		}
	}
	if c.SecurityContext.SeccompProfile == nil {
		c.SecurityContext.SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}

func applySecurityContextToUserContainers(containers []wfapi.UserContainer) {
	for i := range containers {
		applySecurityContextToContainer(&containers[i].Container)
	}
}

// applySecurityContextToExecutorTemplate applies security defaults to executor
// templates without enforcing runAsNonRoot on the pod or main container, since
// user-specified images (e.g. python:3.12) may run as root. Init containers
// (KFP system images like the launcher) still get the full security context.
func applySecurityContextToExecutorTemplate(t *wfapi.Template) {
	if t == nil {
		return
	}
	applyPodSeccompProfileOnly(&t.SecurityContext)
	applyUserContainerSecurityContext(t.Container)
	// Init containers use KFP system images (e.g. launcher) that support non-root.
	applySecurityContextToUserContainers(t.InitContainers)
	for i := range t.Sidecars {
		applyUserContainerSecurityContext(&t.Sidecars[i].Container)
	}
}

func applyPodSecurityDefaults(sc **k8score.PodSecurityContext) {
	runAsNonRoot := true
	if *sc == nil {
		*sc = &k8score.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			SeccompProfile: &k8score.SeccompProfile{
				Type: k8score.SeccompProfileTypeRuntimeDefault,
			},
		}
		return
	}
	if (*sc).RunAsNonRoot == nil {
		(*sc).RunAsNonRoot = &runAsNonRoot
	}
	if (*sc).SeccompProfile == nil {
		(*sc).SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}

// applyPodSeccompProfileOnly sets only the SeccompProfile at the pod level,
// without RunAsNonRoot. Used for executor templates where user-specified
// images may run as root.
func applyPodSeccompProfileOnly(sc **k8score.PodSecurityContext) {
	if *sc == nil {
		*sc = &k8score.PodSecurityContext{
			SeccompProfile: &k8score.SeccompProfile{
				Type: k8score.SeccompProfileTypeRuntimeDefault,
			},
		}
		return
	}
	if (*sc).SeccompProfile == nil {
		(*sc).SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}

func defaultUserContainerSecurityContext() *k8score.SecurityContext {
	allowPrivilegeEscalation := false
	return &k8score.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		SeccompProfile: &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func applyUserContainerSecurityContext(c *k8score.Container) {
	if c == nil {
		return
	}
	if c.SecurityContext == nil {
		c.SecurityContext = defaultUserContainerSecurityContext()
		return
	}
	if c.SecurityContext.AllowPrivilegeEscalation == nil {
		allowPrivilegeEscalation := false
		c.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
	}
	if c.SecurityContext.SeccompProfile == nil {
		c.SecurityContext.SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}
