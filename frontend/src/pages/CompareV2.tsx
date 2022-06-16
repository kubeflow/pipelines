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
import { commonCss } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage } from 'src/lib/Utils';
import {
  getArtifactsFromContext,
  getEventsByExecutions,
  getExecutionsFromContext,
  getKfpV2RunContext,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { Event, Execution } from 'src/third_party/mlmd';
import { PageProps } from './Page';

interface ExecutionArtifacts {
  execution: Execution;
  linkedArtifacts: LinkedArtifact[];
}

interface RunArtifacts {
  run: ApiRunDetail;
  executionArtifacts: ExecutionArtifacts[];
}

function CompareV2(props: PageProps) {
  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details.
  const { data: runs } = useQuery<ApiRunDetail[], Error>(
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

  // Retrieves MLMD states (executions and linked artifacts) from the MLMD store.
  const { data: runArtifacts } = useQuery<RunArtifacts[], Error>(
    ['run_artifacts', { runIds, runs }],
    () => {
      if (runs) {
        return Promise.all(
          runIds.map(async (r, index) => {
            const context = await getKfpV2RunContext(r);
            const executions = await getExecutionsFromContext(context);
            const artifacts = await getArtifactsFromContext(context);
            const events = (await getEventsByExecutions(executions)).filter(
              e => e.getType() === Event.Type.OUTPUT,
            );

            // Match artifacts to executions.
            const artifactMap = new Map();
            artifacts.forEach(artifact => artifactMap.set(artifact.getId(), artifact));
            const executionArtifacts = executions.map(execution => {
              const executionEvents = events.filter(e => e.getExecutionId() === execution.getId());
              const linkedArtifacts = executionEvents.map(event => {
                return {
                  event,
                  artifact: artifactMap.get(event.getArtifactId()),
                } as LinkedArtifact;
              });
              return {
                execution,
                linkedArtifacts,
              } as ExecutionArtifacts;
            });
            return {
              run: runs[index],
              executionArtifacts,
            };
          }),
        );
      }
      return [];
    },
    {
      staleTime: Infinity,
      onError: error =>
        props.updateBanner({
          message: 'Cannot get MLMD objects from Metadata store.',
          additionalInfo: error.message,
          mode: 'error',
        }),
      onSuccess: () => props.updateBanner({}),
    },
  );

  console.log(runArtifacts);

  return (
    <div className={commonCss.page}>
      <p>This is the V2 Run Comparison page.</p>
    </div>
  );
}

export default CompareV2;
