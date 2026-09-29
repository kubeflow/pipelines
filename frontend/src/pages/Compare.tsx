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
import { QUERY_PARAMS } from 'src/components/Router';
import { URLParser } from 'src/lib/URLParser';
import EnhancedCompareV2 from './CompareV2';
import { PageProps } from './Page';

export const OVERVIEW_SECTION_NAME = 'Run overview';
export const PARAMS_SECTION_NAME = 'Parameters';
export const METRICS_SECTION_NAME = 'Metrics';

export default function Compare(props: PageProps) {
  const runIds = new URLParser(props).get(QUERY_PARAMS.runlist)?.split(',') || [];
  const invalidRunCount = runIds.length < 2 || runIds.length > 10;
  const { updateBanner } = props;
  // External synchronization: the page shell owns the banner.
  useEffect(() => {
    if (invalidRunCount) {
      updateBanner({
        additionalInfo:
          'At least two runs and at most ten runs must be selected to view the Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      });
      return () => updateBanner({});
    }
    return undefined;
  }, [invalidRunCount, updateBanner]);
  return invalidRunCount ? null : <EnhancedCompareV2 {...props} />;
}
