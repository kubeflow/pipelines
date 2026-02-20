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

func applySecurityContextToTemplate(template *wfapi.Template) {
	if template == nil {
		return
	}
	// Apply SeccompProfile and RunAsNonRoot at the pod level.
	// Argo Workflows is designed to run rootless:
	// https://argo-workflows.readthedocs.io/en/latest/workflow-pod-security-context/
	applyPodSecurityDefaults(&template.SecurityContext)
	applySecurityContextToContainer(template.Container)
	applySecurityContextToUserContainers(template.InitContainers)
	applySecurityContextToUserContainers(template.Sidecars)
}

func applySecurityContextToContainer(container *k8score.Container) {
	if container == nil {
		return
	}
	if container.SecurityContext == nil {
		container.SecurityContext = defaultContainerSecurityContext()
		return
	}
	if container.SecurityContext.AllowPrivilegeEscalation == nil {
		allowPrivilegeEscalation := false
		container.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
	}
	if container.SecurityContext.RunAsNonRoot == nil {
		runAsNonRoot := true
		container.SecurityContext.RunAsNonRoot = &runAsNonRoot
	}
	if container.SecurityContext.Capabilities == nil {
		container.SecurityContext.Capabilities = &k8score.Capabilities{
			Drop: []k8score.Capability{"ALL"},
		}
	}
	if container.SecurityContext.SeccompProfile == nil {
		container.SecurityContext.SeccompProfile = &k8score.SeccompProfile{
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
//
// When defaultRunAsUser, defaultRunAsGroup, or defaultRunAsNonRoot are non-nil,
// the admin-configured values are applied to the main user container only.
// These override any SDK-specified values (the driver also enforces this at runtime).
func applySecurityContextToExecutorTemplate(template *wfapi.Template, defaultRunAsUser, defaultRunAsGroup *int64, defaultRunAsNonRoot *bool) {
	if template == nil {
		return
	}
	applyPodSeccompProfileOnly(&template.SecurityContext)
	applyUserContainerSecurityContext(template.Container)
	// Apply admin defaults to user container only (not init containers or sidecars).
	if template.Container != nil && (defaultRunAsUser != nil || defaultRunAsGroup != nil || defaultRunAsNonRoot != nil) {
		if template.Container.SecurityContext == nil {
			template.Container.SecurityContext = &k8score.SecurityContext{}
		}
		if defaultRunAsUser != nil {
			v := *defaultRunAsUser
			template.Container.SecurityContext.RunAsUser = &v
		}
		if defaultRunAsGroup != nil {
			v := *defaultRunAsGroup
			template.Container.SecurityContext.RunAsGroup = &v
		}
		if defaultRunAsNonRoot != nil {
			v := *defaultRunAsNonRoot
			template.Container.SecurityContext.RunAsNonRoot = &v
		}
	}
	// Init containers use KFP system images (e.g. launcher) that support non-root.
	applySecurityContextToUserContainers(template.InitContainers)
	for i := range template.Sidecars {
		applyUserContainerSecurityContext(&template.Sidecars[i].Container)
	}
}

func applyPodSecurityDefaults(podSecurityContext **k8score.PodSecurityContext) {
	runAsNonRoot := true
	if *podSecurityContext == nil {
		*podSecurityContext = &k8score.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			SeccompProfile: &k8score.SeccompProfile{
				Type: k8score.SeccompProfileTypeRuntimeDefault,
			},
		}
		return
	}
	if (*podSecurityContext).RunAsNonRoot == nil {
		(*podSecurityContext).RunAsNonRoot = &runAsNonRoot
	}
	if (*podSecurityContext).SeccompProfile == nil {
		(*podSecurityContext).SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}

// applyPodSeccompProfileOnly sets only the SeccompProfile at the pod level,
// without RunAsNonRoot. Used for executor templates where user-specified
// images may run as root.
func applyPodSeccompProfileOnly(podSecurityContext **k8score.PodSecurityContext) {
	if *podSecurityContext == nil {
		*podSecurityContext = &k8score.PodSecurityContext{
			SeccompProfile: &k8score.SeccompProfile{
				Type: k8score.SeccompProfileTypeRuntimeDefault,
			},
		}
		return
	}
	if (*podSecurityContext).SeccompProfile == nil {
		(*podSecurityContext).SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}

func defaultUserContainerSecurityContext() *k8score.SecurityContext {
	allowPrivilegeEscalation := false
	return &k8score.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Capabilities: &k8score.Capabilities{
			Drop: []k8score.Capability{"ALL"},
		},
		SeccompProfile: &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func applyUserContainerSecurityContext(container *k8score.Container) {
	if container == nil {
		return
	}
	if container.SecurityContext == nil {
		container.SecurityContext = defaultUserContainerSecurityContext()
		return
	}
	if container.SecurityContext.AllowPrivilegeEscalation == nil {
		allowPrivilegeEscalation := false
		container.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
	}
	if container.SecurityContext.Capabilities == nil {
		container.SecurityContext.Capabilities = &k8score.Capabilities{
			Drop: []k8score.Capability{"ALL"},
		}
	}
	if container.SecurityContext.SeccompProfile == nil {
		container.SecurityContext.SeccompProfile = &k8score.SeccompProfile{
			Type: k8score.SeccompProfileTypeRuntimeDefault,
		}
	}
}
