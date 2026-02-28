// Package matchers defines custom matchers for compiled workflows
/*
Copyright 2018-2023 The Kubeflow Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package matchers

import (
	"reflect"
	"sort"

	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/v2/api/matcher"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

// CompareWorkflows - compare 2 workflows
func CompareWorkflows(actual *v1alpha1.Workflow, expected *v1alpha1.Workflow) {
	logger.Log("Compared Compiled Workflow with expected input YAML file")
	gomega.Expect(actual.Namespace).To(gomega.Equal(expected.Namespace), "Namespace is not same")
	gomega.Expect(actual.Finalizers).To(gomega.Equal(expected.Finalizers), "Finalizers are not same")
	gomega.Expect(actual.Name).To(gomega.Equal(expected.Name), "Name is not same")
	gomega.Expect(actual.Kind).To(gomega.Equal(expected.Kind), "Kind is not same")
	gomega.Expect(actual.GenerateName).To(gomega.Equal(expected.GenerateName), "Generate Name is not same")
	matcher.MatchMaps(actual.Labels, expected.Labels, "Labels")
	matcher.MatchMaps(actual.Annotations, expected.Annotations, "Annotations")
	// Match Specs
	sort.Slice(expected.Spec.Arguments.Parameters, func(i, j int) bool {
		return expected.Spec.Arguments.Parameters[i].Name < expected.Spec.Arguments.Parameters[j].Name
	})
	sort.Slice(actual.Spec.Arguments.Parameters, func(i, j int) bool {
		return actual.Spec.Arguments.Parameters[i].Name < actual.Spec.Arguments.Parameters[j].Name
	})
	for paramIndex, param := range expected.Spec.Arguments.Parameters {
		gomega.Expect(actual.Spec.Arguments.Parameters[paramIndex].Name).To(gomega.Equal(param.Name), "Parameter Name is not same")
		gomega.Expect(actual.Spec.Arguments.Parameters[paramIndex].Description).To(gomega.Equal(param.Description), "Parameter Description is not same")
		gomega.Expect(actual.Spec.Arguments.Parameters[paramIndex].Default).To(gomega.Equal(param.Default), "Parameter Default is not same")
		gomega.Expect(AreStringsSameWithoutOrder(actual.Spec.Arguments.Parameters[paramIndex].Value.String(), param.Value.String())).To(gomega.BeTrue(), "Parameter Value is not same")
		gomega.Expect(actual.Spec.Arguments.Parameters[paramIndex].Enum).To(gomega.Equal(param.Enum), "Parameter Enum is not same")
		gomega.Expect(actual.Spec.Arguments.Parameters[paramIndex].ValueFrom).To(gomega.Equal(param.ValueFrom), "Parameter ValueFrom is not same")
	}
	if expected.Spec.Affinity != nil {
		gomega.Expect(actual.Spec.Affinity.NodeAffinity).To(gomega.Equal(expected.Spec.Affinity.NodeAffinity), "Node Affinity not same")
		gomega.Expect(actual.Spec.Affinity.PodAffinity).To(gomega.Equal(expected.Spec.Affinity.PodAffinity), "Pod Affinity not same")
		gomega.Expect(actual.Spec.Affinity.PodAntiAffinity).To(gomega.Equal(expected.Spec.Affinity.PodAntiAffinity), "Pod Anti Affinity not same")
	} else {
		gomega.Expect(actual.Spec.Affinity).To(gomega.BeNil(), "Affinity is not nil")
	}
	gomega.Expect(actual.Spec.ActiveDeadlineSeconds).To(gomega.Equal(expected.Spec.ActiveDeadlineSeconds), "ActiveDeadlineSeconds is not same")
	gomega.Expect(actual.Spec.ArchiveLogs).To(gomega.Equal(expected.Spec.ArchiveLogs), "ArchiveLogs is not same")
	gomega.Expect(actual.Spec.ArtifactGC).To(gomega.Equal(expected.Spec.ArtifactGC), "ArtifactGC is not same")
	gomega.Expect(actual.Spec.ArtifactRepositoryRef).To(gomega.Equal(expected.Spec.ArtifactRepositoryRef), "ArtifactRepositoryRef is not same")
	gomega.Expect(actual.Spec.AutomountServiceAccountToken).To(gomega.Equal(expected.Spec.AutomountServiceAccountToken), "AutomountServiceAccountToken is not same")
	gomega.Expect(actual.Spec.DNSConfig).To(gomega.Equal(expected.Spec.DNSConfig), "DNSConfig is not same")
	gomega.Expect(actual.Spec.DNSPolicy).To(gomega.Equal(expected.Spec.DNSPolicy), "DNSPolicy is not same")
	gomega.Expect(actual.Spec.Entrypoint).To(gomega.Equal(expected.Spec.Entrypoint), "Entrypoint is not same")
	gomega.Expect(actual.Spec.Executor).To(gomega.Equal(expected.Spec.Executor), "Executor is not same")
	gomega.Expect(actual.Spec.Hooks).To(gomega.Equal(expected.Spec.Hooks), "Hooks is not same")
	gomega.Expect(actual.Spec.HostAliases).To(gomega.Equal(expected.Spec.HostAliases), "HostAliases is not same")
	gomega.Expect(actual.Spec.HostNetwork).To(gomega.Equal(expected.Spec.HostNetwork), "HostNetwork is not same")
	gomega.Expect(actual.Spec.ImagePullSecrets).To(gomega.Equal(expected.Spec.ImagePullSecrets), "ImagePullSecrets are not same")
	gomega.Expect(actual.Spec.Metrics).To(gomega.Equal(expected.Spec.Metrics), "Metrics are not same")
	gomega.Expect(actual.Spec.NodeSelector).To(gomega.Equal(expected.Spec.NodeSelector), "NodeSelector is not same")
	gomega.Expect(actual.Spec.OnExit).To(gomega.Equal(expected.Spec.OnExit), "OnExit is not same")
	gomega.Expect(actual.Spec.Parallelism).To(gomega.Equal(expected.Spec.Parallelism), "Parallelism is not same")
	gomega.Expect(actual.Spec.PodDisruptionBudget).To(gomega.Equal(expected.Spec.PodDisruptionBudget), "PodDisruptionBudget is not same")
	gomega.Expect(actual.Spec.PodGC).To(gomega.Equal(expected.Spec.PodGC), "PodGC is not same")
	gomega.Expect(actual.Spec.PodMetadata).To(gomega.Equal(expected.Spec.PodMetadata), "PodMetadata is not same")
	gomega.Expect(actual.Spec.PodPriorityClassName).To(gomega.Equal(expected.Spec.PodPriorityClassName), "PodPriorityClassName is not same")
	gomega.Expect(actual.Spec.PodSpecPatch).To(gomega.Equal(expected.Spec.PodSpecPatch), "PodSpecPatch is not same")
	gomega.Expect(actual.Spec.Priority).To(gomega.Equal(expected.Spec.Priority), "Priority is not same")
	gomega.Expect(actual.Spec.RetryStrategy).To(gomega.Equal(expected.Spec.RetryStrategy), "RetryStrategy is not same")
	gomega.Expect(actual.Spec.SchedulerName).To(gomega.Equal(expected.Spec.SchedulerName), "SchedulerName is not same")
	matchPodSecurityContext(actual.Spec.SecurityContext, expected.Spec.SecurityContext, "WorkflowSpec SecurityContext is not same")
	gomega.Expect(actual.Spec.Shutdown).To(gomega.Equal(expected.Spec.Shutdown), "Shutdown is not same")
	gomega.Expect(actual.Spec.ServiceAccountName).To(gomega.Equal(expected.Spec.ServiceAccountName), "ServiceAccountName is not same")
	gomega.Expect(actual.Spec.Suspend).To(gomega.Equal(expected.Spec.Suspend), "Suspend is not same")
	gomega.Expect(actual.Spec.Synchronization).To(gomega.Equal(expected.Spec.Synchronization), "Synchronization is not same")
	gomega.Expect(actual.Spec.Tolerations).To(gomega.Equal(expected.Spec.Tolerations), "Tolerations is not same")
	gomega.Expect(actual.Spec.TTLStrategy).To(gomega.Equal(expected.Spec.TTLStrategy), "TTLStrategy is not same")
	for index, template := range expected.Spec.Templates {
		gomega.Expect(actual.Spec.Templates[index].Inputs).To(gomega.Equal(template.Inputs), "Template Inputs is not same")
		gomega.Expect(actual.Spec.Templates[index].Outputs).To(gomega.Equal(template.Outputs), "Template Outputs is not same")
		gomega.Expect(actual.Spec.Templates[index].Tolerations).To(gomega.Equal(template.Tolerations), "Tolerations is not same")
		gomega.Expect(actual.Spec.Templates[index].PodSpecPatch).To(gomega.Equal(template.PodSpecPatch), "PodSpecPatch is not same")
		gomega.Expect(actual.Spec.Templates[index].Synchronization).To(gomega.Equal(template.Synchronization), "Synchronization is not same")
		gomega.Expect(actual.Spec.Templates[index].Volumes).To(gomega.Equal(template.Volumes), "Volumes is not same")
		gomega.Expect(actual.Spec.Templates[index].Suspend).To(gomega.Equal(template.Suspend), "Suspend is not same")
		matchPodSecurityContext(actual.Spec.Templates[index].SecurityContext, template.SecurityContext, "Template SecurityContext is not same")
		gomega.Expect(actual.Spec.Templates[index].SchedulerName).To(gomega.Equal(template.SchedulerName), "SchedulerName is not same")
		gomega.Expect(actual.Spec.Templates[index].RetryStrategy).To(gomega.Equal(template.RetryStrategy), "RetryStrategy is not same")
		gomega.Expect(actual.Spec.Templates[index].Parallelism).To(gomega.Equal(template.Parallelism), "Parallelism is not same")
		gomega.Expect(actual.Spec.Templates[index].NodeSelector).To(gomega.Equal(template.NodeSelector), "NodeSelector is not same")
		gomega.Expect(actual.Spec.Templates[index].Metrics).To(gomega.Equal(template.Metrics), "Metrics is not same")
		gomega.Expect(actual.Spec.Templates[index].HostAliases).To(gomega.Equal(template.HostAliases), "HostAliases is not same")
		gomega.Expect(actual.Spec.Templates[index].Executor).To(gomega.Equal(template.Executor), "Executor is not same")
		gomega.Expect(actual.Spec.Templates[index].ServiceAccountName).To(gomega.Equal(template.ServiceAccountName), "ServiceAccountName is not same")
		gomega.Expect(actual.Spec.Templates[index].AutomountServiceAccountToken).To(gomega.Equal(template.AutomountServiceAccountToken), "AutomountServiceAccountToken is not same")
		gomega.Expect(actual.Spec.Templates[index].ActiveDeadlineSeconds).To(gomega.Equal(template.ActiveDeadlineSeconds), "ActiveDeadlineSeconds is not same")
		gomega.Expect(actual.Spec.Templates[index].Affinity).To(gomega.Equal(template.Affinity), "Affinity is not same")
		gomega.Expect(actual.Spec.Templates[index].Name).To(gomega.Equal(template.Name), "Name is not same")
		// Compare Container
		MatchContainer(actual.Spec.Templates[index].Container, template.Container)
		for containerIndex, userContainer := range template.InitContainers {
			MatchUserContainer(&actual.Spec.Templates[index].InitContainers[containerIndex], &userContainer)
		}

		gomega.Expect(actual.Spec.Templates[index].FailFast).To(gomega.Equal(template.FailFast), "FailFast is not same")
		gomega.Expect(actual.Spec.Templates[index].ArchiveLocation).To(gomega.Equal(template.ArchiveLocation), "ArchiveLocation is not same")
		gomega.Expect(actual.Spec.Templates[index].ContainerSet).To(gomega.Equal(template.ContainerSet), "ContainerSet is not same")
		gomega.Expect(actual.Spec.Templates[index].Daemon).To(gomega.Equal(template.Daemon), "Daemon is not same")
		gomega.Expect(actual.Spec.Templates[index].Data).To(gomega.Equal(template.Data), "Data is not same")
		MatchDAG(actual.Spec.Templates[index].DAG, template.DAG)
		gomega.Expect(actual.Spec.Templates[index].HTTP).To(gomega.Equal(template.HTTP), "HTTP is not same")
		gomega.Expect(actual.Spec.Templates[index].Memoize).To(gomega.Equal(template.Memoize), "Memoize is not same")
		gomega.Expect(actual.Spec.Templates[index].Metadata).To(gomega.Equal(template.Metadata), "Metadata is not same")
		gomega.Expect(actual.Spec.Templates[index].Plugin).To(gomega.Equal(template.Plugin), "Plugin is not same")
		gomega.Expect(actual.Spec.Templates[index].PriorityClassName).To(gomega.Equal(template.PriorityClassName), "PriorityClassName is not same")
		gomega.Expect(actual.Spec.Templates[index].Resource).To(gomega.Equal(template.Resource), "Resource is not same")
		gomega.Expect(actual.Spec.Templates[index].Script).To(gomega.Equal(template.Script), "Script is not same")
		gomega.Expect(actual.Spec.Templates[index].Sidecars).To(gomega.Equal(template.Sidecars), "Sidecars is not same")
		gomega.Expect(actual.Spec.Templates[index].Steps).To(gomega.Equal(template.Steps), "Steps is not same")
		gomega.Expect(actual.Spec.Templates[index].Timeout).To(gomega.Equal(template.Timeout), "Timeout is not same")
	}
	gomega.Expect(actual.Spec.TemplateDefaults).To(gomega.Equal(expected.Spec.TemplateDefaults), "TemplateDefaults are not same")
	gomega.Expect(actual.Spec.VolumeClaimGC).To(gomega.Equal(expected.Spec.VolumeClaimGC), "VolumeClaimGC is not same")
	gomega.Expect(actual.Spec.Volumes).To(gomega.Equal(expected.Spec.Volumes), "Volumes is not same")
	gomega.Expect(actual.Spec.VolumeClaimTemplates).To(gomega.Equal(expected.Spec.VolumeClaimTemplates), "VolumeClaimTemplates are not same")
	gomega.Expect(actual.Spec.WorkflowMetadata).To(gomega.Equal(expected.Spec.WorkflowMetadata), "WorkflowMetadata is not same")
	gomega.Expect(actual.Spec.WorkflowTemplateRef).To(gomega.Equal(expected.Spec.WorkflowTemplateRef), "WorkflowTemplateRef is not same")
}

// MatchContainerResourceLimits - Compare resource limits of a container
func MatchContainerResourceLimits(actual v1.ResourceList, expected v1.ResourceList) {
	gomega.Expect(actual.Pods()).To(gomega.Equal(expected.Pods()), "Container Resources Limits Pods is not same")
	// convert the CPU and memory limits to the same approximation units
	actualCPU := actual.Cpu().AsApproximateFloat64()
	expectedCPU := expected.Cpu().AsApproximateFloat64()
	actualMemory := actual.Memory().AsApproximateFloat64()
	expectedMemory := expected.Memory().AsApproximateFloat64()
	gomega.Expect(actualCPU).To(gomega.Equal(expectedCPU), "Container Resources Limits Cpu is not same")
	gomega.Expect(actualMemory).To(gomega.Equal(expectedMemory), "Container Resources Limits Memory is not same")
	gomega.Expect(actual.Storage()).To(gomega.Equal(expected.Storage()), "Container Resources Limits Storage is not same")
	gomega.Expect(actual.StorageEphemeral()).To(gomega.Equal(expected.StorageEphemeral()), "Container Resources Limits StorageEphemeral is not same")
}

// MatchContainer - Compare 2 containers
func MatchContainer(actual *v1.Container, expected *v1.Container) {
	if expected != nil {
		gomega.Expect(actual.Name).To(gomega.Equal(expected.Name), "Container Name is not same")
		gomega.Expect(actual.Args).To(gomega.ConsistOf(expected.Args), "Container Args is not same")
		matchContainerSecurityContext(actual.SecurityContext, expected.SecurityContext, "Container SecurityContext is not same")
		gomega.Expect(actual.Env).To(gomega.Equal(expected.Env), "Container Env is not same")
		gomega.Expect(actual.EnvFrom).To(gomega.Equal(expected.EnvFrom), "Container EnvFrom is not same")
		gomega.Expect(actual.Command).To(gomega.Equal(expected.Command), "Container Command is not same")
		gomega.Expect(actual.ImagePullPolicy).To(gomega.Equal(expected.ImagePullPolicy), "Container ImagePullPolicy is not same")
		gomega.Expect(actual.Image).To(gomega.Equal(expected.Image), "Container Image is not same")
		gomega.Expect(actual.Lifecycle).To(gomega.Equal(expected.Lifecycle), "Container Lifecycle is not same")
		gomega.Expect(actual.LivenessProbe).To(gomega.Equal(expected.LivenessProbe), "Container LivenessProbe is not same")
		gomega.Expect(actual.Ports).To(gomega.Equal(expected.Ports), "Container Ports is not same")
		gomega.Expect(actual.ReadinessProbe).To(gomega.Equal(expected.ReadinessProbe), "Container ReadinessProbe is not same")
		gomega.Expect(actual.ResizePolicy).To(gomega.Equal(expected.ResizePolicy), "Container ResizePolicy is not same")
		gomega.Expect(actual.StartupProbe).To(gomega.Equal(expected.StartupProbe), "Container StartupProbe is not same")
		gomega.Expect(actual.Stdin).To(gomega.Equal(expected.Stdin), "Container Stdin is not same")
		gomega.Expect(actual.StdinOnce).To(gomega.Equal(expected.StdinOnce), "Container StdinOnce is not same")
		gomega.Expect(actual.RestartPolicy).To(gomega.Equal(expected.RestartPolicy), "Container RestartPolicy is not same")
		gomega.Expect(actual.TerminationMessagePath).To(gomega.Equal(expected.TerminationMessagePath), "Container TerminationMessagePath is not same")
		gomega.Expect(actual.TTY).To(gomega.Equal(expected.TTY), "Container TTY is not same")
		gomega.Expect(actual.TerminationMessagePolicy).To(gomega.Equal(expected.TerminationMessagePolicy), "Container TerminationMessagePolicy is not same")
		gomega.Expect(actual.VolumeDevices).To(gomega.Equal(expected.VolumeDevices), "Container VolumeDevices is not same")
		gomega.Expect(actual.VolumeMounts).To(gomega.Equal(expected.VolumeMounts), "Container VolumeMounts is not same")
		gomega.Expect(actual.WorkingDir).To(gomega.Equal(expected.WorkingDir), "Container WorkingDir is not same")
		gomega.Expect(actual.Resources.Claims).To(gomega.Equal(expected.Resources.Claims), "Container Resources Claims is not same")
		MatchContainerResourceLimits(actual.Resources.Limits, expected.Resources.Limits)
		MatchContainerResourceLimits(actual.Resources.Requests, expected.Resources.Requests)
	} else {
		gomega.Expect(actual).To(gomega.BeNil(), "Container is expected to be nil")
	}
}

// MatchUserContainer - Compare 2 user containers
func MatchUserContainer(actual *v1alpha1.UserContainer, expected *v1alpha1.UserContainer) {
	if expected != nil {
		gomega.Expect(actual.Name).To(gomega.Equal(expected.Name), "User Container Name is not same")
		gomega.Expect(actual.Args).To(gomega.ConsistOf(expected.Args), "User Container Args is not same")
		matchContainerSecurityContext(actual.SecurityContext, expected.SecurityContext, "User Container SecurityContext is not same")
		gomega.Expect(actual.Env).To(gomega.Equal(expected.Env), "User Container Env is not same")
		gomega.Expect(actual.EnvFrom).To(gomega.Equal(expected.EnvFrom), "User Container EnvFrom is not same")
		gomega.Expect(actual.Command).To(gomega.Equal(expected.Command), "User Container Command is not same")
		gomega.Expect(actual.ImagePullPolicy).To(gomega.Equal(expected.ImagePullPolicy), "User Container ImagePullPolicy is not same")
		gomega.Expect(actual.Image).To(gomega.Equal(expected.Image), "User Container Image is not same")
		gomega.Expect(actual.Lifecycle).To(gomega.Equal(expected.Lifecycle), "User Container Lifecycle is not same")
		gomega.Expect(actual.LivenessProbe).To(gomega.Equal(expected.LivenessProbe), "User Container LivenessProbe is not same")
		gomega.Expect(actual.Ports).To(gomega.Equal(expected.Ports), "User Container Ports is not same")
		gomega.Expect(actual.ReadinessProbe).To(gomega.Equal(expected.ReadinessProbe), "User Container ReadinessProbe is not same")
		gomega.Expect(actual.ResizePolicy).To(gomega.Equal(expected.ResizePolicy), "User Container ResizePolicy is not same")
		gomega.Expect(actual.StartupProbe).To(gomega.Equal(expected.StartupProbe), "User Container StartupProbe is not same")
		gomega.Expect(actual.Stdin).To(gomega.Equal(expected.Stdin), "User Container Stdin is not same")
		gomega.Expect(actual.StdinOnce).To(gomega.Equal(expected.StdinOnce), "User Container StdinOnce is not same")
		gomega.Expect(actual.RestartPolicy).To(gomega.Equal(expected.RestartPolicy), "User Container RestartPolicy is not same")
		gomega.Expect(actual.TerminationMessagePath).To(gomega.Equal(expected.TerminationMessagePath), "User Container TerminationMessagePath is not same")
		gomega.Expect(actual.TTY).To(gomega.Equal(expected.TTY), "User Container TTY is not same")
		gomega.Expect(actual.TerminationMessagePolicy).To(gomega.Equal(expected.TerminationMessagePolicy), "User Container TerminationMessagePolicy is not same")
		gomega.Expect(actual.VolumeDevices).To(gomega.Equal(expected.VolumeDevices), "User Container VolumeDevices is not same")
		gomega.Expect(actual.VolumeMounts).To(gomega.Equal(expected.VolumeMounts), "User Container VolumeMounts is not same")
		gomega.Expect(actual.WorkingDir).To(gomega.Equal(expected.WorkingDir), "User Container WorkingDir is not same")
		gomega.Expect(actual.Resources.Claims).To(gomega.Equal(expected.Resources.Claims), "User Container Resources Claims is not same")
		MatchContainerResourceLimits(actual.Resources.Limits, expected.Resources.Limits)
		MatchContainerResourceLimits(actual.Resources.Requests, expected.Resources.Requests)
	} else {
		gomega.Expect(actual).To(gomega.BeNil(), "User Container is expected to be nil")
	}
}

// MatchDAG - Match 2 DAG templates
func MatchDAG(actual *v1alpha1.DAGTemplate, expected *v1alpha1.DAGTemplate) {
	if expected != nil {
		gomega.Expect(actual.FailFast).To(gomega.Equal(expected.FailFast), "DAGTemplate FailFast is not same")
		gomega.Expect(actual.Target).To(gomega.Equal(expected.Target), "DAGTemplate Target is not same")
		for index, task := range actual.Tasks {
			gomega.Expect(actual.Tasks[index].Name).To(gomega.Equal(task.Name), "DAGTemplate Task Name is not same")
			gomega.Expect(actual.Tasks[index].Hooks).To(gomega.Equal(task.Hooks), "DAGTemplate Task Hooks is not same")
			gomega.Expect(actual.Tasks[index].Template).To(gomega.Equal(task.Template), "DAGTemplate Task Template is not same")
			gomega.Expect(actual.Tasks[index].Arguments).To(gomega.Equal(task.Arguments), "DAGTemplate Task Arguments is not same")
			gomega.Expect(actual.Tasks[index].When).To(gomega.Equal(task.When), "DAGTemplate Task When is not same")
			gomega.Expect(actual.Tasks[index].ContinueOn).To(gomega.Equal(task.ContinueOn), "DAGTemplate Task ContinueOn is not same")
			gomega.Expect(actual.Tasks[index].Dependencies).To(gomega.Equal(task.Dependencies), "DAGTemplate Task Dependencies is not same")
			gomega.Expect(actual.Tasks[index].Depends).To(gomega.Equal(task.Depends), "DAGTemplate Task Depends is not same")
			gomega.Expect(actual.Tasks[index].Inline).To(gomega.Equal(task.Inline), "DAGTemplate Task Inline is not same")
			gomega.Expect(actual.Tasks[index].TemplateRef).To(gomega.Equal(task.TemplateRef), "DAGTemplate Task TemplateRef is not same")
			gomega.Expect(actual.Tasks[index].WithItems).To(gomega.Equal(task.WithItems), "DAGTemplate Task WithItems is not same")
			gomega.Expect(actual.Tasks[index].WithParam).To(gomega.Equal(task.WithParam), "DAGTemplate Task WithParam is not same")
			gomega.Expect(actual.Tasks[index].WithSequence).To(gomega.Equal(task.WithSequence), "DAGTemplate Task WithItems is not same")
			gomega.Expect(actual.Tasks[index].OnExit).To(gomega.Equal(task.OnExit), "DAGTemplate Task WithItems is not same")
		}
	} else {
		gomega.Expect(actual).To(gomega.BeNil(), "DAGTemplate is expected to be nil")
	}

}

// AreStringsSameWithoutOrder - checks if two strings contain the same characters, regardless of order.
func AreStringsSameWithoutOrder(s1, s2 string) bool {
	// Convert strings to rune slices
	r1 := []rune(s1)
	r2 := []rune(s2)

	// Sort the rune slices
	sort.Slice(r1, func(i, j int) bool { return r1[i] < r1[j] })
	sort.Slice(r2, func(i, j int) bool { return r2[i] < r2[j] })

	// Compare the sorted slices
	return reflect.DeepEqual(r1, r2)
}

func matchPodSecurityContext(actual *v1.PodSecurityContext, expected *v1.PodSecurityContext, msg string) {
	if expected == nil {
		return
	}
	gomega.Expect(actual).NotTo(gomega.BeNil(), msg)
	if expected.RunAsUser != nil {
		gomega.Expect(actual.RunAsUser).To(gomega.Equal(expected.RunAsUser), msg)
	}
	if expected.RunAsGroup != nil {
		gomega.Expect(actual.RunAsGroup).To(gomega.Equal(expected.RunAsGroup), msg)
	}
	if expected.FSGroup != nil {
		gomega.Expect(actual.FSGroup).To(gomega.Equal(expected.FSGroup), msg)
	}
	if expected.RunAsNonRoot != nil {
		gomega.Expect(actual.RunAsNonRoot).To(gomega.Equal(expected.RunAsNonRoot), msg)
	}
	if expected.SeccompProfile != nil {
		gomega.Expect(actual.SeccompProfile).NotTo(gomega.BeNil(), msg)
		gomega.Expect(actual.SeccompProfile.Type).To(gomega.Equal(expected.SeccompProfile.Type), msg)
		gomega.Expect(actual.SeccompProfile.LocalhostProfile).To(gomega.Equal(expected.SeccompProfile.LocalhostProfile), msg)
	}
}

func matchContainerSecurityContext(actual *v1.SecurityContext, expected *v1.SecurityContext, msg string) {
	if expected == nil {
		return
	}
	gomega.Expect(actual).NotTo(gomega.BeNil(), msg)
	if expected.AllowPrivilegeEscalation != nil {
		gomega.Expect(actual.AllowPrivilegeEscalation).To(gomega.Equal(expected.AllowPrivilegeEscalation), msg)
	}
	if expected.Privileged != nil {
		gomega.Expect(actual.Privileged).To(gomega.Equal(expected.Privileged), msg)
	}
	if expected.ReadOnlyRootFilesystem != nil {
		gomega.Expect(actual.ReadOnlyRootFilesystem).To(gomega.Equal(expected.ReadOnlyRootFilesystem), msg)
	}
	if expected.RunAsNonRoot != nil {
		gomega.Expect(actual.RunAsNonRoot).To(gomega.Equal(expected.RunAsNonRoot), msg)
	}
	if expected.RunAsUser != nil {
		gomega.Expect(actual.RunAsUser).To(gomega.Equal(expected.RunAsUser), msg)
	}
	if expected.RunAsGroup != nil {
		gomega.Expect(actual.RunAsGroup).To(gomega.Equal(expected.RunAsGroup), msg)
	}
	if expected.Capabilities != nil {
		gomega.Expect(actual.Capabilities).NotTo(gomega.BeNil(), msg)
		gomega.Expect(actual.Capabilities.Drop).To(gomega.Equal(expected.Capabilities.Drop), msg)
		gomega.Expect(actual.Capabilities.Add).To(gomega.Equal(expected.Capabilities.Add), msg)
	}
	if expected.SeccompProfile != nil {
		gomega.Expect(actual.SeccompProfile).NotTo(gomega.BeNil(), msg)
		gomega.Expect(actual.SeccompProfile.Type).To(gomega.Equal(expected.SeccompProfile.Type), msg)
		gomega.Expect(actual.SeccompProfile.LocalhostProfile).To(gomega.Equal(expected.SeccompProfile.LocalhostProfile), msg)
	}
}
