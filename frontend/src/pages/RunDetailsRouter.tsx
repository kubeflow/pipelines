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
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import EnhancedRunDetails, { RunDetailsProps } from 'src/pages/RunDetails';
import { RunDetailsV2 } from 'src/pages/RunDetailsV2';

// This is a router to determine whether to show V1 or V2 run detail page.
export default function RunDetailsRouter(props: RunDetailsProps) {
  const runId = props.match.params[RouteParams.runId];

  // Retrieves run detail.
  const { isSuccess, data } = useQuery<ApiRunDetail, Error>(
    ['run_detail', { id: runId }],
    () => Apis.runServiceApi.getRun(runId),
    {},
  );

  if (data === undefined) {
    return <></>;
  }

  if (
    isSuccess &&
    data &&
    data.run &&
    data.run.pipeline_spec &&
    data.run.pipeline_spec.pipeline_manifest
  ) {
    // TODO(zijianjoy): We need to switch to use pipeline_manifest for new API implementation.
    const isV2Pipeline = WorkflowUtils.isPipelineSpec(data.run.pipeline_spec.pipeline_manifest);
    if (isV2Pipeline) {
      return (
        <RunDetailsV2
          pipeline_job={data.run.pipeline_spec.pipeline_manifest}
          runDetail={data}
          {...props}
        />
      );
    }
  }

  return <EnhancedRunDetails {...props} />;
}
