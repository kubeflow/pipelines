// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

// getFingerPrint generates a fingerprint for caching. The PVC names are included in the fingerprint since it's assumed
// PVCs have side effects (e.g. files written for tasks later on in the run) on the execution. If the PVC names are
// different, the execution shouldn't be reused for the cache.
func getFingerPrint(opts common.Options, executorInput *pipelinespec.ExecutorInput, pvcNames []string) (string, error) {
	outputParametersTypeMap := make(map[string]string)
	for outputParamName, outputParamSpec := range opts.Component.GetOutputDefinitions().GetParameters() {
		outputParametersTypeMap[outputParamName] = outputParamSpec.GetParameterType().String()
	}
	userCmdArgs := make([]string, 0, len(opts.Container.Command)+len(opts.Container.Args))
	userCmdArgs = append(userCmdArgs, opts.Container.Command...)
	userCmdArgs = append(userCmdArgs, opts.Container.Args...)

	// Deduplicate PVC names and sort them to ensure consistent fingerprint generation.
	pvcNamesMap := map[string]struct{}{}
	for _, pvcName := range pvcNames {
		pvcNamesMap[pvcName] = struct{}{}
	}

	sortedPVCNames := make([]string, 0, len(pvcNamesMap))
	for pvcName := range pvcNamesMap {
		sortedPVCNames = append(sortedPVCNames, pvcName)
	}
	sort.Strings(sortedPVCNames)

	cacheKey, err := cacheutils.GenerateCacheKey(
		executorInput.GetInputs(),
		executorInput.GetOutputs(),
		outputParametersTypeMap,
		userCmdArgs,
		opts.Container.Image,
		sortedPVCNames,
	)
	if err != nil {
		return "", fmt.Errorf("failure while generating CacheKey: %w", err)
	}
	fingerPrint, err := cacheutils.GenerateFingerPrint(cacheKey)
	return fingerPrint, err
}

func getFingerPrintsAndID(
	ctx context.Context,
	execution *Execution,
	kfpAPI kfpapi.API,
	opts *common.Options,
	pvcNames []string) (fingerprint string, task *apiv2beta1.PipelineTask, err error) {

	if opts.CacheDisabled || !execution.WillTrigger() || !opts.Task.GetCachingOptions().GetEnableCache() {
		return "", nil, nil
	}

	glog.Infof("Task {%s} enables cache", opts.Task.GetTaskInfo().GetName())
	fingerPrint, err := getFingerPrint(*opts, execution.ExecutorInput, pvcNames)
	if err != nil {
		return "", nil, fmt.Errorf("failure while getting fingerPrint: %w", err)
	}

	fullView := apiv2beta1.ListRunsRequest_FULL
	pageToken := ""
	var matchedTasks []*apiv2beta1.PipelineTask

	for {
		runs, err := kfpAPI.ListRuns(ctx, &apiv2beta1.ListRunsRequest{
			Namespace: opts.Namespace,
			PageSize:  50,
			PageToken: pageToken,
			View:      &fullView,
		})
		if err != nil {
			return "", nil, fmt.Errorf("failure while listing runs for cache lookup: %w", err)
		}

		for _, run := range runs.GetRuns() {
			for _, candidateTask := range run.GetTasks() {
				if candidateTask.GetCacheFingerprint() == fingerPrint &&
					candidateTask.GetState() == apiv2beta1.PipelineTask_SUCCEEDED {
					matchedTasks = append(matchedTasks, candidateTask)
				}
			}
		}

		pageToken = runs.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}

	if len(matchedTasks) == 0 {
		glog.Infof("No cached tasks found for task {%s}", opts.Task.GetTaskInfo().GetName())
		return fingerPrint, nil, nil
	} else if len(matchedTasks) > 1 {
		glog.Infof("Found multiple cached tasks for task %s with fingerprint %s, the first one found will be used.", opts.Task.GetTaskInfo().GetName(), fingerPrint)
	}

	glog.V(4).Infof("Got a cache hit for task {%s}", opts.Task.GetTaskInfo().GetName())
	return fingerPrint, matchedTasks[0], nil
}
