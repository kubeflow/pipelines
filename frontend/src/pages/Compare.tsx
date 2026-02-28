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

import React, { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import { QUERY_PARAMS } from 'src/components/Router';
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
  const [compareVersion, setCompareVersion] = useState<CompareVersion>(CompareVersion.Unknown);
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details, set page version on success.
  const { isLoading, isError, error, data } = useQuery<ApiRunDetail[], Error>(
    ['run_details', { ids: runIds }],
    () => Promise.all(runIds.map(async id => await Apis.runServiceApi.getRun(id))),
    {
      staleTime: Infinity,
    },
  );

  useEffect(() => {
    // Set the version based on the runs included.
    if (data) {
      if (data.length < 2 || data.length > 10) {
        setCompareVersion(CompareVersion.InvalidRunCount);
      } else {
        const v2runs = data.filter(run => 'pipeline_manifest' in (run.run?.pipeline_spec ?? {}));
        if (v2runs.length === 0) {
          setCompareVersion(CompareVersion.V1);
        } else if (v2runs.length === data.length) {
          setCompareVersion(CompareVersion.V2);
        } else {
          setCompareVersion(CompareVersion.Mixed);
        }
      }
    }
  }, [data]);

  useEffect(() => {
    if (isLoading) {
      return;
    }

    // Update banner based on error, feature flag, run versions, and run count.
    if (isError) {
      (async function() {
        const errorMessage = await errorToMessage(error);
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
    } else if (isFeatureEnabled(FeatureKey.V2_ALPHA) && compareVersion !== CompareVersion.V1) {
      // Clear the banner unless the V1 page is shown, as that page handles its own banner state.
      updateBanner({});
    }
  }, [compareVersion, isError, error, isLoading, updateBanner, runIds.length]);

  if (isError || isLoading) {
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
