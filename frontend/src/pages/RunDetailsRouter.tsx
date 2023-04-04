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

import React from 'react';
import * as JsYaml from 'js-yaml';
import { useQuery } from 'react-query';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import EnhancedRunDetails, { RunDetailsProps } from 'src/pages/RunDetails';
import { RunDetailsV2 } from 'src/pages/RunDetailsV2';

// This is a router to determine whether to show V1 or V2 run detail page.
export default function RunDetailsRouter(props: RunDetailsProps) {
  const runId = props.match.params[RouteParams.runId];
  let pipelineManifest: string | undefined;

  // Retrieves v2 run detail.
  const { isSuccess: getV2RunSuccess, data: v2Run } = useQuery<V2beta1Run, Error>(
    ['v2_run_detail', { id: runId }],
    () => Apis.runServiceApiV2.getRun(runId),
    {},
  );

  if (getV2RunSuccess && v2Run && v2Run.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(v2Run.pipeline_spec);
  }

  const pipelineVersionId = v2Run?.pipeline_version_reference?.pipeline_version_id;

  const { data: templateStrFromVersionId } = useQuery<string, Error>(
    ['PipelineVersionTemplate', pipelineVersionId],
    async () => {
      if (!pipelineVersionId) {
        return '';
      }
      // TODO(jlyaoyuli): temporarily use v1 API here, need to change in pipeline API integration.
      const template = await Apis.pipelineServiceApi.getPipelineVersionTemplate(pipelineVersionId);
      return template?.template || '';
    },
    { enabled: !!pipelineVersionId, staleTime: Infinity, cacheTime: Infinity },
  );

  const templateString = pipelineManifest ?? templateStrFromVersionId;

  if (getV2RunSuccess && v2Run && templateString) {
    // TODO(zijianjoy): We need to switch to use pipeline_manifest for new API implementation.
    const isV2Pipeline = WorkflowUtils.isPipelineSpec(templateString);
    if (isV2Pipeline) {
      return <RunDetailsV2 pipeline_job={templateString} run={v2Run} {...props} />;
    }
  }

  return <EnhancedRunDetails {...props} />;
}
