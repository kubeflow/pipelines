// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Inventory Deployments, not registered Pods: legacy writers and zero-replica
// peers must remain visible. Names are not identities; the existing app roles and
// container names are. Conflicting evidence must not hide a possible old writer.
func discoverScheduleWriters(ctx context.Context, client kubernetes.Interface, namespace string) (scheduleWriterConfiguration, map[string]*appsv1.Deployment, error) {
	list, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("discover schedule writer Deployments; apply the complete handoff RBAC: %w", err)
	}
	config := scheduleWriterConfiguration{}
	deployments := map[string]*appsv1.Deployment{}
	roles := map[string]bool{}
	for i := range list.Items {
		deployment := &list.Items[i]
		var candidates []scheduleWriterIdentity
		for _, standard := range defaultScheduleWriters() {
			hinted := deployment.Name == standard.Deployment || deployment.Labels["app"] == standard.App || deployment.Spec.Template.Labels["app"] == standard.App
			for _, container := range deployment.Spec.Template.Spec.Containers {
				hinted = hinted || container.Name == standard.Container
			}
			if deployment.Spec.Selector != nil {
				hinted = hinted || deployment.Spec.Selector.MatchLabels["app"] == standard.App
				for _, requirement := range deployment.Spec.Selector.MatchExpressions {
					if requirement.Key == "app" && requirement.Operator == metav1.LabelSelectorOpIn {
						for _, value := range requirement.Values {
							hinted = hinted || value == standard.App
						}
					}
				}
			}
			if hinted {
				candidates = append(candidates, standard)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		invalid := func() (scheduleWriterConfiguration, map[string]*appsv1.Deployment, error) {
			return nil, nil, apierrors.NewBadRequest(fmt.Sprintf("Deployment %s has inconsistent schedule-writer role labels, selector, or container; preserve standard role identities or configure %s explicitly for every writer", deployment.Name, scheduleWritersEnv))
		}
		if len(candidates) != 1 {
			return invalid()
		}
		writer := candidates[0]
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if deployment.Spec.Template.Labels["app"] != writer.App || err != nil || deployment.Spec.Selector == nil || selector.Empty() || !selector.Matches(labels.Set(deployment.Spec.Template.Labels)) {
			return invalid()
		}
		found := false
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == writer.Container {
				found = true
			}
		}
		if !found {
			return invalid()
		}
		writer.Deployment = deployment.Name
		config = append(config, writer)
		deployments[deployment.Name] = deployment
		roles[writer.Role] = true
	}
	for _, standard := range defaultScheduleWriters() {
		if !roles[standard.Role] {
			return nil, nil, fmt.Errorf("managed writer Deployment %s role does not exist yet; waiting for installation or rollout, or configure %s if role labels changed", standard.Deployment, scheduleWritersEnv)
		}
	}
	sort.Slice(config, func(i, j int) bool { return config[i].Deployment < config[j].Deployment })
	return config, deployments, nil
}
