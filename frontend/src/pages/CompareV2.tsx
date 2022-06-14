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
  filterLinkedArtifactsByType,
  getArtifactTypes,
  getExecutionsFromContext,
  getKfpV2RunContext,
  getOutputLinkedArtifactsInExecution,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import { PageProps } from './Page';

interface RunExecutions {
  run: ApiRunDetail;
  executions: Execution[];
}

interface ExecutionArtifacts {
  execution: Execution;
  executionName: string;
  linkedArtifacts: LinkedArtifact[];
  linkedArtifactNames: string[];
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

  // Retrieves MLMD states from the MLMD store.
  const { data: runExecutions } = useQuery<RunExecutions[], Error>(
    ['mlmd_package', { runIds, runs }],
    () => {
      if (runs) {
        return Promise.all(
          runIds.map(async (r, index) => {
            const context = await getKfpV2RunContext(r);
            const executions = await getExecutionsFromContext(context);

            return {
              run: runs[index],
              executions,
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

  // Retrieving a list of artifacts associated with each execution.
  const { data: runArtifacts } = useQuery<RunArtifacts[], Error>(
    ['execution_output_artifact', { runExecutions }],
    () => {
      if (runExecutions) {
        return Promise.all(
          runExecutions.map(async runExecution => {
            const executionArtifacts = Promise.all(
              runExecution.executions.map(async execution => {
                const executionName = execution
                  .getCustomPropertiesMap()
                  .get('display_name')
                  ?.getStringValue();
                const linkedArtifacts = await getOutputLinkedArtifactsInExecution(execution);
                // TODO: It seems I can also retrieve this through custom properties - which is preferable? Do I need the Event in the LinkedArtifact?
                const linkedArtifactNames = linkedArtifacts.map(
                  l =>
                    l.event
                      .getPath()
                      ?.getStepsList()[0]
                      .getKey() || 'na',
                );
                return {
                  execution,
                  executionName,
                  linkedArtifacts,
                  linkedArtifactNames,
                } as ExecutionArtifacts;
              }),
            );
            return {
              run: runExecution.run,
              executionArtifacts: await executionArtifacts,
            } as RunArtifacts;
          }),
        );
      }
      return [];
    },
    { staleTime: Infinity },
  );

  // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // so we can identify metrics artifact provided by system.
  const { data: artifactTypes } = useQuery<ArtifactType[], Error>(
    ['artifact_types', {}],
    () => getArtifactTypes(),
    {
      staleTime: Infinity,
    },
  );

  console.log(runArtifacts);
  console.log(artifactTypes);

  if (!runArtifacts) {
    return <></>;
  }

  if (!artifactTypes) {
    return <></>;
  }

  const scalarMetricsArtifacts: RunArtifacts[] = runArtifacts;
  const confusionMatrixArtifacts: RunArtifacts[] = runArtifacts;
  const rocCurveArtifacts: RunArtifacts[] = runArtifacts;
  const htmlArtifacts: RunArtifacts[] = runArtifacts;
  const markdownArtifacts: RunArtifacts[] = runArtifacts;

  let runIndex = 0;
  for (const runArtifact of runArtifacts) {
    let executionIndex = 0;
    for (const e of runArtifact.executionArtifacts) {
      scalarMetricsArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts = filterLinkedArtifactsByType(
        'system.Metrics',
        artifactTypes,
        e.linkedArtifacts,
      );
      scalarMetricsArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifactNames = scalarMetricsArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts.map(
        l =>
          l.event
            .getPath()
            ?.getStepsList()[0]
            .getKey() || 'na',
      );

      confusionMatrixArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts = filterLinkedArtifactsByType(
        'system.ClassificationMetrics',
        artifactTypes,
        e.linkedArtifacts,
      );
      confusionMatrixArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifactNames = confusionMatrixArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts.map(
        l =>
          l.event
            .getPath()
            ?.getStepsList()[0]
            .getKey() || 'na',
      );

      rocCurveArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts = filterLinkedArtifactsByType(
        'system.ClassificationMetrics',
        artifactTypes,
        e.linkedArtifacts,
      );
      rocCurveArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifactNames = rocCurveArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts.map(
        l =>
          l.event
            .getPath()
            ?.getStepsList()[0]
            .getKey() || 'na',
      );

      htmlArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts = filterLinkedArtifactsByType(
        'system.HTML',
        artifactTypes,
        e.linkedArtifacts,
      );
      htmlArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifactNames = htmlArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts.map(
        l =>
          l.event
            .getPath()
            ?.getStepsList()[0]
            .getKey() || 'na',
      );

      markdownArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts = filterLinkedArtifactsByType(
        'system.Markdown',
        artifactTypes,
        e.linkedArtifacts,
      );
      markdownArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifactNames = markdownArtifacts[runIndex].executionArtifacts[
        executionIndex
      ].linkedArtifacts.map(
        l =>
          l.event
            .getPath()
            ?.getStepsList()[0]
            .getKey() || 'na',
      );
      executionIndex++;
    }
    runIndex++;
  }

  console.log(scalarMetricsArtifacts);
  console.log(confusionMatrixArtifacts);
  console.log(rocCurveArtifacts);
  console.log(htmlArtifacts);
  console.log(markdownArtifacts);

  return (
    <div className={commonCss.page}>
      <p>This is the V2 Run Comparison page.</p>
    </div>
  );
}

export default CompareV2;
