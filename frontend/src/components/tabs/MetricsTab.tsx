/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from 'react';
import { useQuery } from 'react-query';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { commonCss, padding } from 'src/Css';
import {
  getArtifactTypes,
  getOutputLinkedArtifactsInExecution,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import Banner from '../Banner';
import { MetricsVisualizations } from '../viewers/MetricsVisualizations';
import { ExecutionTitle } from './ExecutionTitle';

type MetricsTabProps = {
  execution: Execution;
  namespace: string | undefined;
};

/**
 * Metrics tab renders metrics for the artifact of given execution.
 * Some system metrics are: Confusion Matrix, ROC Curve, Scalar, etc.
 * Detail can be found in https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py
 * Note that these metrics are only available on KFP v2 mode.
 */
export function MetricsTab({ execution, namespace }: MetricsTabProps) {
  let executionCompleted = false;
  const executionState = execution.getLastKnownState();
  if (
    !(
      executionState === Execution.State.NEW ||
      executionState === Execution.State.UNKNOWN ||
      executionState === Execution.State.RUNNING
    )
  ) {
    executionCompleted = true;
  }

  const executionId = execution.getId();
  // Retrieving a list of artifacts associated with this execution,
  // so we can find the artifact for system metrics from there.
  const {
    isLoading: isLoadingArtifacts,
    isSuccess: isSuccessArtifacts,
    error: errorArtifacts,
    data: artifacts,
  } = useQuery<LinkedArtifact[], Error>(
    ['execution_output_artifact', { id: executionId, state: executionState }],
    () => getOutputLinkedArtifactsInExecution(execution),
    { enabled: executionCompleted, staleTime: Infinity },
  );

  // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // so we can identify metrics artifact provided by system.
  const {
    isLoading: isLoadingArtifactTypes,
    isSuccess: isSuccessArtifactTypes,
    error: errorArtifactTypes,
    data: artifactTypes,
  } = useQuery<ArtifactType[], Error>(
    ['artifact_types', { id: executionId, state: executionState }],
    () => getArtifactTypes(),
    {
      enabled: executionCompleted,
      staleTime: Infinity,
    },
  );

  let executionStateUnknown = executionState === Execution.State.UNKNOWN;
  // This react element produces banner message if query to MLMD is pending or has error.
  // Once query is completed, it shows actual content of metrics visualization in MetricsSwitcher.
  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <ExecutionTitle execution={execution} />
          {executionStateUnknown && <Banner message='Task is in unknown state.' mode='info' />}
          {!executionStateUnknown && !executionCompleted && (
            <Banner message='Task has not completed.' mode='info' />
          )}
          {(isLoadingArtifactTypes || isLoadingArtifacts) && (
            <Banner message='Metrics is loading.' mode='info' />
          )}
          {errorArtifacts && (
            <Banner
              message='Error in retrieving metrics information.'
              mode='error'
              additionalInfo={errorArtifacts.message}
            />
          )}
          {!errorArtifacts && errorArtifactTypes && (
            <Banner
              message='Error in retrieving artifact types information.'
              mode='error'
              additionalInfo={errorArtifactTypes.message}
            />
          )}
          {isSuccessArtifacts && isSuccessArtifactTypes && artifacts && artifactTypes && (
            <MetricsVisualizations
              linkedArtifacts={artifacts}
              artifactTypes={artifactTypes}
              execution={execution}
              namespace={namespace}
            ></MetricsVisualizations>
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}
