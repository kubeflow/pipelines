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

import React, { useState } from 'react';
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import { QUERY_PARAMS } from 'src/components/Router';
import { FeatureKey, isFeatureEnabled } from 'src/features';
import { Apis } from 'src/lib/Apis';
import { errorToMessage } from 'src/lib/Utils';
import { URLParser } from '../lib/URLParser';
import CompareV1, { CompareState } from './CompareV1';
import CompareV2 from './CompareV2';
import { PageProps } from './Page';

export type CompareProps = PageProps & CompareState;

enum CompareVersion {
  V1,
  V2,
  Mixed,
  TooFewRuns,
}

// This is a router to determine whether to show V1 or V2 compare page.
export default function Compare(props: CompareProps) {
  const [compareVersion, setCompareVersion] = useState<CompareVersion>(CompareVersion.TooFewRuns);
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details, set page version on success.
  const { isError } = useQuery<ApiRunDetail[], Error>(
    ['run_details', { ids: runIds }],
    () => Promise.all(runIds.map(async id => await Apis.runServiceApi.getRun(id))),
    {
      refetchOnMount: 'always',
      staleTime: Infinity,
      onError: async error => {
        const errorMessage = await errorToMessage(error);
        props.updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message: `Error: failed loading ${runIds.length} runs. Click Details for more information.`,
          mode: 'error',
        });
      },
      onSuccess: data => {
        // Set the version based on the runs included.
        let version: CompareVersion = CompareVersion.TooFewRuns;
        if (data && data.length > 1) {
          for (const run of data) {
            const runVersion = run.run?.pipeline_spec?.hasOwnProperty('pipeline_manifest')
              ? CompareVersion.V2
              : CompareVersion.V1;
            if (version === CompareVersion.TooFewRuns) {
              version = runVersion;
            } else if (version !== runVersion) {
              version = CompareVersion.Mixed;
            }
          }
        }

        // Update banner based on feature flag, run versions, and run count.
        if (isFeatureEnabled(FeatureKey.V2_ALPHA) && version === CompareVersion.TooFewRuns) {
          props.updateBanner({
            additionalInfo: 'At least two runs must be selected to view the Run Comparison page.',
            message:
              'Error: failed loading the Run Comparison page. Click Details for more information.',
            mode: 'error',
          });
        } else if (isFeatureEnabled(FeatureKey.V2_ALPHA) && version === CompareVersion.Mixed) {
          props.updateBanner({
            additionalInfo:
              'The selected runs are a mix of V1 and V2.' +
              ' Please select all V1 or all V2 runs to view the associated Run Comparison page.',
            message:
              'Error: failed loading the Run Comparison page. Click Details for more information.',
            mode: 'error',
          });
        } else if (!isFeatureEnabled(FeatureKey.V2_ALPHA) && version === CompareVersion.V2) {
          props.updateBanner({
            additionalInfo:
              'The selected runs are all V2, but the V2_ALPHA feature flag is disabled.' +
              ' The V1 page will not show any useful information for these runs.',
            message:
              'Info: enable the V2_ALPHA feature flag in order to view the updated Run Comparison page.',
            mode: 'info',
          });
        } else {
          props.updateBanner({});
        }

        setCompareVersion(version);
      },
    },
  );

  if (isError) {
    return <></>;
  }

  if (isFeatureEnabled(FeatureKey.V2_ALPHA) && compareVersion === CompareVersion.V2) {
    return <CompareV2 />;
  } else if (!isFeatureEnabled(FeatureKey.V2_ALPHA) || compareVersion === CompareVersion.V1) {
    return <CompareV1 {...props} />;
  }

  // Only the error banner shown for mixed run versions or less than two selected runs.
  return <></>;
}
