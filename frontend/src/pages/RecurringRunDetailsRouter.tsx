/*
 * Copyright 2023 The Kubeflow Authors
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
import { useQuery } from '@tanstack/react-query';
import { CircularProgress } from '@mui/material';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { errorToMessage } from 'src/lib/Utils';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import RecurringRunDetailsV2 from './RecurringRunDetailsV2';
import { RecurringRunDetailsV2FC } from 'src/pages/functional_components/RecurringRunDetailsV2FC';
import { FeatureKey, isFeatureEnabled } from 'src/features';
import { queryKeys } from 'src/hooks/queryKeys';

export default function RecurringRunDetailsRouter(props: PageProps) {
  const { updateBanner } = props;
  const recurringRunId = props.params[RouteParams.recurringRunId];

  const {
    isLoading: recurringRunIsLoading,
    error: recurringRunError,
    data: recurringRun,
  } = useQuery<V2beta1RecurringRun, Error>({
    queryKey: queryKeys.v2RecurringRunDetail(recurringRunId),
    queryFn: () => {
      if (!recurringRunId) {
        throw new Error('Recurring run ID is missing');
      }
      return Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    },
    enabled: !!recurringRunId,
    staleTime: Infinity,
  });

  useEffect(() => {
    if (recurringRunError) {
      let cancelled = false;
      errorToMessage(recurringRunError).then((msg) => {
        if (!cancelled) {
          updateBanner({
            message: `Error: failed to retrieve recurring run: ${recurringRunId}. Click Details for more information.`,
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
  }, [recurringRunError, recurringRunId, updateBanner]);

  // Metadata and actions do not require a template: latest-version schedules have
  // no pinned version, and an existing schedule's version may have been deleted.
  if (recurringRun) {
    return isFeatureEnabled(FeatureKey.FUNCTIONAL_COMPONENT) ? (
      <RecurringRunDetailsV2FC {...props} />
    ) : (
      <RecurringRunDetailsV2 {...props} />
    );
  }

  if (recurringRunIsLoading) {
    return (
      <div style={{ textAlign: 'center', paddingTop: 40 }}>
        <CircularProgress />
        <div>Currently loading recurring run information</div>
      </div>
    );
  }

  return <div role='alert'>Unable to load recurring run details. Refresh this page to retry.</div>;
}
