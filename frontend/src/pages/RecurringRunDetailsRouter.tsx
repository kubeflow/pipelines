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

import React from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from 'react-query';
import { V2beta1RecurringRun } from 'src/apisv2beta1/recurringrun';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { PageProps } from './Page';
import RecurringRunDetails from './RecurringRunDetails';
import RecurringRunDetailsV2 from './RecurringRunDetailsV2';
import { RecurringRunDetailsV2FC } from 'src/pages/functional_components/RecurringRunDetailsV2FC';
import { FeatureKey, isFeatureEnabled } from 'src/features';

// This is a router to determine whether to show V1 or V2 recurring run details page.
export default function RecurringRunDetailsRouter(props: PageProps) {
  const recurringRunId = props.match.params[RouteParams.recurringRunId];
  let pipelineManifest: string | undefined;

  const {
    isSuccess: getRecurringRunSuccess,
    isFetching: recurringRunIsFetching,
    data: v2RecurringRun,
  } = useQuery<V2beta1RecurringRun, Error>(
    ['v2_recurring_run_detail', { id: recurringRunId }],
    () => {
      if (!recurringRunId) {
        throw new Error('Recurring run ID is missing');
      }
      return Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    },
    { enabled: !!recurringRunId, staleTime: Infinity },
  );

  if (getRecurringRunSuccess && v2RecurringRun && v2RecurringRun.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(v2RecurringRun.pipeline_spec);
  }

  const pipelineId = v2RecurringRun?.pipeline_version_reference?.pipeline_id;
  const pipelineVersionId = v2RecurringRun?.pipeline_version_reference?.pipeline_version_id;

  const { isFetching: templateStrIsFetching, data: templateStrFromPipelineVersion } = useQuery<
    string,
    Error
  >(
    ['PipelineVersionTemplate', { pipelineId, pipelineVersionId }],
    async () => {
      if (!pipelineId || !pipelineVersionId) {
        return '';
      }
      const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
        pipelineId,
        pipelineVersionId,
      );
      const pipelineSpec = pipelineVersion.pipeline_spec;
      return pipelineSpec ? JsYaml.safeDump(pipelineSpec) : '';
    },
    { enabled: !!pipelineId && !!pipelineVersionId, staleTime: Infinity, cacheTime: Infinity },
  );

  const templateString = pipelineManifest ?? templateStrFromPipelineVersion;

  if (getRecurringRunSuccess && v2RecurringRun && templateString) {
    const isV2Pipeline = WorkflowUtils.isPipelineSpec(templateString);
    if (isV2Pipeline) {
      return isFeatureEnabled(FeatureKey.FUNCTIONAL_COMPONENT) ? (
        <RecurringRunDetailsV2FC {...props} />
      ) : (
        <RecurringRunDetailsV2 {...props} />
      );
    }
  }

  if (recurringRunIsFetching || templateStrIsFetching) {
    return <div>Currently loading recurring run information</div>;
  }

  return <RecurringRunDetails {...props} />;
}
