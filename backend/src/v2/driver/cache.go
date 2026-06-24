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
	userCmdArgs = append(userCmdArgs, fmt.Sprintf("__kfp_pipeline_name=%s", opts.PipelineName))

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

// getFingerPrintsAndID computes the cache fingerprint for the current task and,
// when caching is enabled, returns the first successful PipelineTask in the
// same namespace whose stored fingerprint matches.
//
// A cache hit requires the task to resolve to the same effective execution
// shape: the same resolved inputs and declared outputs, the same output
// parameter types, the same user container image and command/args, and the
// same referenced PVC names. If caching is disabled, the task will not run, or
// the task opts out of caching, this returns no fingerprint and no match. When
// multiple successful tasks share the fingerprint, the first match found is
// reused.
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

	cachedTaskResponse, err := kfpAPI.FindCachedTask(ctx, &apiv2beta1.FindCachedTaskRequest{
		Namespace:        opts.Namespace,
		CacheFingerprint: fingerPrint,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failure while finding cached task: %w", err)
	}
	if cachedTaskResponse.GetTask() == nil {
		glog.Infof("No cached tasks found for task {%s}", opts.Task.GetTaskInfo().GetName())
		return fingerPrint, nil, nil
	}

	glog.V(4).Infof("Got a cache hit for task {%s}", opts.Task.GetTaskInfo().GetName())
	return fingerPrint, cachedTaskResponse.GetTask(), nil
}
