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

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import * as JsYaml from 'js-yaml';
import { isEqual } from 'lodash';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
import { QUERY_PARAMS, RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { errorToMessage } from 'src/lib/Utils';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { RouteComponentProps } from 'react-router-dom';
import EnhancedRunDetails, { RunDetailsProps } from 'src/pages/RunDetails';
import { RunDetailsV2, RunDetailsV2Params, RunDetailsV2Props } from 'src/pages/RunDetailsV2';
import { usePipelineVersionTemplate } from 'src/hooks/usePipelineVersionTemplate';
import { queryKeys } from 'src/hooks/queryKeys';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { BannerProps } from 'src/components/Banner';

export const RUN_DETAILS_REFETCH_INTERVAL = 10000;
const MAX_POST_RETRY_DISCOVERY_ATTEMPTS = 3;
let latestRetryRefreshVersion = 0;
const LEGACY_TASK_LINK_WARNING: BannerProps = {
  message:
    'This task link cannot be opened in the legacy Run Details view. Locate the task from the run graph instead.',
  mode: 'warning',
};

function nextRetryRefreshVersion(): number {
  latestRetryRefreshVersion += 1;
  return latestRetryRefreshVersion;
}

function preserveDeepEqualData<T>(previous: T | undefined, next: T): T {
  return previous !== undefined && isEqual(previous, next) ? previous : next;
}

interface PendingRetryDiscovery {
  baseline: V2beta1Run;
  preRetryTasks?: V2beta1PipelineTask[];
  remainingAttempts: number;
}

// This is a router to determine whether to show V1 or V2 run detail page.
export default function RunDetailsRouter(
  props: RunDetailsProps & RouteComponentProps<RunDetailsV2Params>,
) {
  const { updateBanner } = props;
  const currentPageBanner = useRef<BannerProps>({});
  const updatePageBanner = useCallback(
    (banner: BannerProps) => {
      currentPageBanner.current = banner;
      updateBanner(banner);
    },
    [updateBanner],
  );
  const runId = props.match.params[RouteParams.runId];

  // Retrieves v2 run detail.
  const { isLoading: runIsLoading, data: v2Run } = useQuery<V2beta1Run, Error>({
    queryKey: queryKeys.v2RunDetail(runId),
    queryFn: () => Apis.runServiceApiV2.getRun(runId),
    structuralSharing: preserveDeepEqualData,
  });

  const pipelineManifest = useMemo(
    () => (v2Run?.pipeline_spec ? JsYaml.dump(v2Run.pipeline_spec) : undefined),
    [v2Run],
  );

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
          updatePageBanner({
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
  }, [templateStrIsError, templateStrError, updatePageBanner]);

  const templateString = pipelineManifest ?? templateStrFromPipelineVersion;
  const pipelineSpec = useMemo(
    () =>
      templateString ? WorkflowUtils.tryConvertYamlToV2PipelineSpec(templateString) : undefined,
    [templateString],
  );
  const linkedTaskId = new URLSearchParams(props.location.search).get(QUERY_PARAMS.taskId);
  const renderV2Details = !!(v2Run && templateString && pipelineSpec);
  const runDetailsIsLoading = runIsLoading || templateStrIsLoading;

  useEffect(() => {
    if (linkedTaskId && !runDetailsIsLoading && !renderV2Details && !templateStrIsError) {
      updatePageBanner(LEGACY_TASK_LINK_WARNING);
      return () => {
        if (currentPageBanner.current === LEGACY_TASK_LINK_WARNING) {
          updatePageBanner({});
        }
      };
    }
    return undefined;
  }, [linkedTaskId, renderV2Details, runDetailsIsLoading, templateStrIsError, updatePageBanner]);

  if (v2Run && templateString && pipelineSpec) {
    return (
      <PolledRunDetailsV2
        key={runId}
        pipeline_job={templateString}
        parsedPipelineSpec={pipelineSpec}
        run={v2Run}
        {...props}
        updateBanner={updatePageBanner}
      />
    );
  }

  return (
    <EnhancedRunDetails
      {...props}
      isLoading={runDetailsIsLoading}
      updateBanner={updatePageBanner}
    />
  );
}

function PolledRunDetailsV2(props: RunDetailsV2Props) {
  const runId = props.match.params[RouteParams.runId];
  const queryClient = useQueryClient();
  const runQueryKey = useMemo(() => queryKeys.v2RunDetail(runId), [runId]);
  const retryDiscoveryQueryKey = useMemo(() => queryKeys.runRetryDiscovery(runId), [runId]);
  const retryRefreshVersionQueryKey = useMemo(
    () => queryKeys.runRetryRefreshVersion(runId),
    [runId],
  );
  const [retryRefreshVersion, setRetryRefreshVersion] = useState(
    () => queryClient.getQueryData<number>(retryRefreshVersionQueryKey) || 0,
  );
  const [, refreshRetryPolling] = useState(0);
  const loadRun = useCallback(() => Apis.runServiceApiV2.getRun(runId), [runId]);
  const retryDiscoveryPending =
    queryClient.getQueryData<PendingRetryDiscovery>(retryDiscoveryQueryKey) !== undefined;
  const {
    data: refreshedRun,
    error: runRefreshError,
    isRefetchError,
    refetch: refetchRun,
  } = useQuery<V2beta1Run, Error>({
    queryKey: runQueryKey,
    queryFn: loadRun,
    retry: false,
    structuralSharing: preserveDeepEqualData,
    refetchInterval: (query) => {
      const state = query.state.data?.state;
      const runIsActive = state !== undefined && !hasFinishedV2(state);
      const retryPending =
        queryClient.getQueryData<PendingRetryDiscovery>(retryDiscoveryQueryKey) !== undefined;
      return retryPending || runIsActive ? RUN_DETAILS_REFETCH_INTERVAL : false;
    },
    refetchOnMount: retryDiscoveryPending ? 'always' : false,
  });

  useLayoutEffect(() => {
    const queryCache = queryClient.getQueryCache();
    const runQuery = queryCache.find({ exact: true, queryKey: runQueryKey });
    if (!runQuery) {
      return undefined;
    }
    // Success and error events represent completed attempts TanStack accepted. Query functions can
    // still resolve after cancellation, but those discarded results never consume this budget.
    return queryCache.subscribe((event) => {
      if (
        event.type !== 'updated' ||
        event.query !== runQuery ||
        (event.action.type !== 'success' && event.action.type !== 'error')
      ) {
        return;
      }
      const pending = queryClient.getQueryData<PendingRetryDiscovery>(retryDiscoveryQueryKey);
      if (!pending) {
        return;
      }
      const acceptedRun =
        event.action.type === 'success'
          ? (event.query.state.data as V2beta1Run | undefined)
          : undefined;
      if (acceptedRun && !isEqual(acceptedRun, pending.baseline)) {
        // Query cache entries can outlive this component. Use a process-unique generation so a
        // remounted details page cannot reuse task snapshots from an earlier retry.
        const nextVersion = nextRetryRefreshVersion();
        queryClient.setQueryData(retryRefreshVersionQueryKey, nextVersion);
        if (pending.preRetryTasks) {
          queryClient.setQueryData(
            queryKeys.runTaskRetryBaseline(runId, nextVersion),
            pending.preRetryTasks,
          );
        }
        queryClient.removeQueries({ exact: true, queryKey: retryDiscoveryQueryKey });
        setRetryRefreshVersion(nextVersion);
        return;
      }
      const remainingAttempts = pending.remainingAttempts - 1;
      if (remainingAttempts <= 0) {
        queryClient.removeQueries({ exact: true, queryKey: retryDiscoveryQueryKey });
        // QueryObserver computes its next interval before QueryCache subscribers run. Rerender
        // once so the exhausted budget cancels the timer it just scheduled.
        refreshRetryPolling((revision) => revision + 1);
      } else {
        queryClient.setQueryData(retryDiscoveryQueryKey, { ...pending, remainingAttempts });
      }
    });
  }, [queryClient, retryDiscoveryQueryKey, retryRefreshVersionQueryKey, runId, runQueryKey]);
  const onRetryStarted = useCallback(() => {
    // Retry persistence can lag the mutation response. Continue through a few unchanged terminal
    // snapshots, but keep the recovery bounded when a fast retry remains terminal throughout.
    const currentTaskQueryKey = queryKeys.runTasks(runId, retryRefreshVersion || undefined);
    queryClient.setQueryData<PendingRetryDiscovery>(retryDiscoveryQueryKey, {
      baseline: refreshedRun || props.run,
      preRetryTasks: queryClient.getQueryData<V2beta1PipelineTask[]>(currentTaskQueryKey),
      remainingAttempts: MAX_POST_RETRY_DISCOVERY_ATTEMPTS,
    });
    refreshRetryPolling((revision) => revision + 1);
    void refetchRun();
  }, [
    props.run,
    queryClient,
    refreshedRun,
    refetchRun,
    retryDiscoveryQueryKey,
    retryRefreshVersion,
    runId,
  ]);

  return (
    <RunDetailsV2
      {...props}
      onRetryStarted={onRetryStarted}
      retryRefreshVersion={retryRefreshVersion}
      run={refreshedRun || props.run}
      runRefreshError={isRefetchError ? runRefreshError : undefined}
    />
  );
}
