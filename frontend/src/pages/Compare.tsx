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

import React from 'react';
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

// This is a router to determine whether to show V1 or V2 compare page.
export default function Compare(props: CompareProps) {
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details.
  const { isSuccess, isError, data } = useQuery<ApiRunDetail[], Error>(
    ['run_details', { ids: runIds }],
    () => Promise.all(runIds.map(async id => await Apis.runServiceApi.getRun(id))),
    {
      staleTime: Infinity,
      onError: async error => {
        const errorMessage = await errorToMessage(error);
        props.updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message: `Error: failed loading ${runIds.length} runs. Click Details for more information.`,
          mode: 'error',
        });
      },
      onSuccess: () => props.updateBanner({}),
    },
  );

  if (isError || data === undefined) {
    return <></>;
  }

  // Display the V2 Compare page if all selected runs are V2.
  if (
    isFeatureEnabled(FeatureKey.V2_ALPHA) &&
    isSuccess &&
    data &&
    data.every(run => run.run?.pipeline_spec?.hasOwnProperty('pipeline_manifest'))
  ) {
    return <CompareV2 />;
  } else {
    return <CompareV1 {...props} />;
  }
}
