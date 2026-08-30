// Copyright 2018 The Kubeflow Authors
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

package resource

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type podDeletionTarget struct {
	name string
	uid  types.UID
}

// snapshotPodsForDeletion resolves the object identity before RetryRun claims
// the database row. Pod names are deterministic across retry generations, so
// deleting by name after the claim could remove a replacement created by a
// concurrent retry.
func snapshotPodsForDeletion(
	ctx context.Context,
	k8sCoreClient client.KubernetesCoreInterface,
	podNames []string,
	namespace string,
) ([]podDeletionTarget, error) {
	targets := make([]podDeletionTarget, 0, len(podNames))
	podClient := k8sCoreClient.PodClient(namespace)
	for _, podName := range podNames {
		pod, err := podClient.Get(ctx, podName, metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to read pods before retry cleanup")
		}
		if pod == nil || pod.UID == "" {
			return nil, util.NewInternalServerError(
				fmt.Errorf("pod %q has no UID", podName),
				"Failed to identify pods before retry cleanup",
			)
		}
		targets = append(targets, podDeletionTarget{name: podName, uid: pod.UID})
	}
	return targets, nil
}

func deletePods(ctx context.Context, k8sCoreClient client.KubernetesCoreInterface, targets []podDeletionTarget, namespace string) error {
	podClient := k8sCoreClient.PodClient(namespace)
	for _, target := range targets {
		uid := target.uid
		err := podClient.Delete(ctx, target.name, metav1.DeleteOptions{
			Preconditions: &metav1.Preconditions{UID: &uid},
		})
		// A UID conflict proves that this name now belongs to a replacement;
		// the stale retry must leave it alone. A missing pod is already clean.
		if err != nil && !apierr.IsNotFound(err) && !apierr.IsConflict(err) {
			return util.NewInternalServerError(err, "Failed to delete pods")
		}
	}
	return nil
}
