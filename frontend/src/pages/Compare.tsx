/*
 * Copyright 2022 The Kubeflow Authors
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

import { useEffect } from 'react';
import { useQueries } from '@tanstack/react-query';
import { CircularProgress } from '@mui/material';
import { ApiRunDetail } from 'src/apis/run';
import { QUERY_PARAMS } from 'src/components/Router';
import { queryKeys } from 'src/hooks/queryKeys';
import { FeatureKey, isFeatureEnabled } from 'src/features';
import { Apis } from 'src/lib/Apis';
import { errorToMessage } from 'src/lib/Utils';
import { URLParser } from '../lib/URLParser';
import EnhancedCompareV1 from './CompareV1';
import EnhancedCompareV2 from './CompareV2';
import { PageProps } from './Page';

enum CompareVersion {
  V1,
  V2,
  Mixed,
  InvalidRunCount,
  Unknown,
}

export const OVERVIEW_SECTION_NAME = 'Run overview';
export const PARAMS_SECTION_NAME = 'Parameters';
export const METRICS_SECTION_NAME = 'Metrics';

// This is a router to determine whether to show V1 or V2 compare page.
export default function Compare(props: PageProps) {
  const { updateBanner } = props;
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Route each run independently so one rejection is not cached as successful aggregate data.
  const runLoadQueries = useQueries({
    queries: runIds.map((id) => ({
      queryKey: queryKeys.runDetailForComparisonRouting(id),
      queryFn: () => Apis.runServiceApi.getRun(id),
      retry: 1,
      retryDelay: 0,
      staleTime: Infinity,
    })),
  });

  const isLoading = runLoadQueries.some((query) => query.isPending);
  const successfulRuns = runLoadQueries.flatMap((query) =>
    query.data ? [query.data as ApiRunDetail] : [],
  );
  const failedRunLoad = runLoadQueries.find((query) => query.isError);
  const compareVersion = isLoading
    ? CompareVersion.Unknown
    : runIds.length < 2 || runIds.length > 10
      ? CompareVersion.InvalidRunCount
      : !successfulRuns?.length
        ? CompareVersion.Unknown
        : (() => {
            const v2runs = successfulRuns.filter(
              (run) => 'pipeline_manifest' in (run.run?.pipeline_spec ?? {}),
            );
            if (v2runs.length === 0) {
              return CompareVersion.V1;
            }
            if (v2runs.length === successfulRuns.length) {
              return CompareVersion.V2;
            }
            return CompareVersion.Mixed;
          })();

  useEffect(() => {
    if (isLoading) {
      return;
    }

    // Update banner based on error, feature flag, run versions, and run count.
    const routeCannotRenderPartialResults =
      !!failedRunLoad &&
      (!isFeatureEnabled(FeatureKey.V2_ALPHA) || compareVersion !== CompareVersion.V2);
    if (routeCannotRenderPartialResults) {
      (async function () {
        const errorMessage = await errorToMessage(failedRunLoad?.error);
        updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message: `Error: failed loading ${runIds.length} runs. Click Details for more information.`,
          mode: 'error',
        });
      })();
    } else if (
      isFeatureEnabled(FeatureKey.V2_ALPHA) &&
      compareVersion === CompareVersion.InvalidRunCount
    ) {
      updateBanner({
        additionalInfo:
          'At least two runs and at most ten runs must be selected to view the Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      });
    } else if (isFeatureEnabled(FeatureKey.V2_ALPHA) && compareVersion === CompareVersion.Mixed) {
      updateBanner({
        additionalInfo:
          'The selected runs are a mix of V1 and V2.' +
          ' Please select all V1 or all V2 runs to view the associated Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      });
    } else if (
      isFeatureEnabled(FeatureKey.V2_ALPHA) &&
      compareVersion !== CompareVersion.V1 &&
      compareVersion !== CompareVersion.V2
    ) {
      // Clear the banner unless the V1 page is shown, as that page handles its own banner state.
      updateBanner({});
    }
  }, [compareVersion, failedRunLoad, isLoading, updateBanner, runIds.length]);

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 40 }}>
        <CircularProgress />
      </div>
    );
  }

  if (
    !!failedRunLoad &&
    (!isFeatureEnabled(FeatureKey.V2_ALPHA) || compareVersion !== CompareVersion.V2)
  ) {
    return <></>;
  }

  if (!isFeatureEnabled(FeatureKey.V2_ALPHA) || compareVersion === CompareVersion.V1) {
    return <EnhancedCompareV1 {...props} />;
  }

  if (compareVersion === CompareVersion.V2) {
    return <EnhancedCompareV2 {...props} />;
  }

  return <></>;
}
