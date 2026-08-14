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

import { useCallback, useEffect, useMemo } from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from '@tanstack/react-query';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { errorToMessage } from 'src/lib/Utils';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { RouteComponentProps } from 'react-router-dom';
import EnhancedRunDetails, { RunDetailsProps } from 'src/pages/RunDetails';
import { RunDetailsV2, RunDetailsV2Params, RunDetailsV2Props } from 'src/pages/RunDetailsV2';
import { usePipelineVersionTemplate } from 'src/hooks/usePipelineVersionTemplate';
import { queryKeys } from 'src/hooks/queryKeys';
import { hasFinishedV2 } from 'src/lib/StatusUtils';

export const RUN_DETAILS_REFETCH_INTERVAL = 10000;

// This is a router to determine whether to show V1 or V2 run detail page.
export default function RunDetailsRouter(
  props: RunDetailsProps & RouteComponentProps<RunDetailsV2Params>,
) {
  const { updateBanner } = props;
  const runId = props.match.params[RouteParams.runId];
  let pipelineManifest: string | undefined;

  // Retrieves v2 run detail.
  const { isLoading: runIsLoading, data: v2Run } = useQuery<V2beta1Run, Error>({
    queryKey: queryKeys.v2RunDetail(runId),
    queryFn: () => Apis.runServiceApiV2.getRun(runId),
  });

  if (v2Run?.pipeline_spec) {
    pipelineManifest = JsYaml.dump(v2Run.pipeline_spec);
  }

  const pipelineId = v2Run?.pipeline_version_reference?.pipeline_id;
  const pipelineVersionId = v2Run?.pipeline_version_reference?.pipeline_version_id;

  const {
    isLoading: templateStrIsLoading,
    isError: templateStrIsError,
    error: templateStrError,
    data: templateStrFromPipelineVersion,
  } = usePipelineVersionTemplate(
    pipelineManifest ? undefined : pipelineId,
    pipelineManifest ? undefined : pipelineVersionId,
  );

  useEffect(() => {
    if (templateStrIsError && templateStrError) {
      let cancelled = false;
      errorToMessage(templateStrError).then((msg) => {
        if (!cancelled) {
          updateBanner({
            message:
              'Error: failed to retrieve pipeline version template. Click Details for more information.',
            mode: 'error',
            additionalInfo: msg,
          });
        }
      });
      return () => {
        cancelled = true;
      };
    }
    return undefined;
  }, [templateStrIsError, templateStrError, updateBanner]);

  const templateString = pipelineManifest ?? templateStrFromPipelineVersion;
  const pipelineSpec = useMemo(
    () =>
      templateString ? WorkflowUtils.tryConvertYamlToV2PipelineSpec(templateString) : undefined,
    [templateString],
  );

  if (v2Run && templateString && pipelineSpec) {
    return (
      <PolledRunDetailsV2
        key={runId}
        pipeline_job={templateString}
        parsedPipelineSpec={pipelineSpec}
        run={v2Run}
        {...props}
      />
    );
  }

  return <EnhancedRunDetails {...props} isLoading={runIsLoading || templateStrIsLoading} />;
}

function PolledRunDetailsV2(props: RunDetailsV2Props) {
  const runId = props.match.params[RouteParams.runId];
  const {
    data: refreshedRun,
    error: runRefreshError,
    isRefetchError,
    refetch: refetchRun,
  } = useQuery<V2beta1Run, Error>({
    queryKey: queryKeys.v2RunDetail(runId),
    queryFn: () => Apis.runServiceApiV2.getRun(runId),
    refetchInterval: (query) => {
      const state = query.state.data?.state;
      const runIsActive = state !== undefined && !hasFinishedV2(state);
      return runIsActive ? RUN_DETAILS_REFETCH_INTERVAL : false;
    },
    refetchOnMount: false,
  });
  const onRetryStarted = useCallback(() => {
    // The retry mutation has completed, so one explicit refresh is sufficient to discover an
    // active attempt. If the run already returned to a terminal state, do not poll forever waiting
    // for a RUNNING state that can no longer be observed.
    void refetchRun();
  }, [refetchRun]);

  return (
    <RunDetailsV2
      {...props}
      onRetryStarted={onRetryStarted}
      run={refreshedRun || props.run}
      runRefreshError={isRefetchError ? runRefreshError : undefined}
    />
  );
}
