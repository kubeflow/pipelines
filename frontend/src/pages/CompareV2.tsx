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
import { stylesheet } from 'typestyle';
import { PageProps } from './Page';

interface RunExecutions {
  run: ApiRunDetail;
  executions: Execution[];
}

interface ExecutionArtifacts {
  execution: Execution;
  linkedArtifacts: LinkedArtifact[];
}

interface RunArtifacts {
  run: ApiRunDetail;
  executionArtifacts: ExecutionArtifacts[];
}

enum MetricsType {
  SCALAR_METRICS,
  CONFUSION_MATRIX,
  ROC_CURVE,
  HTML,
  MARKDOWN,
}

const css = stylesheet({
  smallIndent: {
    marginLeft: '0.5rem',
  },
  mediumIndent: {
    marginLeft: '1rem',
  },
  largeIndent: {
    marginLeft: '2rem',
  },
});

function filterRunArtifactsByType(
  runArtifacts: RunArtifacts[],
  artifactTypes: ArtifactType[],
  metricsType: MetricsType,
): RunArtifacts[] {
  const metricsFilter =
    metricsType === MetricsType.SCALAR_METRICS
      ? 'system.Metrics'
      : metricsType === MetricsType.CONFUSION_MATRIX || metricsType === MetricsType.ROC_CURVE
      ? 'system.ClassificationMetrics'
      : metricsType === MetricsType.HTML
      ? 'system.HTML'
      : metricsType === MetricsType.MARKDOWN
      ? 'system.Markdown'
      : '';
  const typeRuns: RunArtifacts[] = [];
  for (const runArtifact of runArtifacts) {
    const typeExecutions: ExecutionArtifacts[] = [];
    for (const e of runArtifact.executionArtifacts) {
      let typeArtifacts: LinkedArtifact[] = filterLinkedArtifactsByType(
        metricsFilter,
        artifactTypes,
        e.linkedArtifacts,
      );
      if (metricsType === MetricsType.CONFUSION_MATRIX) {
        typeArtifacts = typeArtifacts.filter(x =>
          x.artifact.getCustomPropertiesMap().has('confusionMatrix'),
        );
      } else if (metricsType === MetricsType.ROC_CURVE) {
        typeArtifacts = typeArtifacts.filter(x =>
          x.artifact.getCustomPropertiesMap().has('confidenceMetrics'),
        );
      }
      if (typeArtifacts.length > 0) {
        typeExecutions.push({
          execution: e.execution,
          linkedArtifacts: typeArtifacts,
        } as ExecutionArtifacts);
      }
    }
    if (typeExecutions.length > 0) {
      typeRuns.push({
        run: runArtifact.run,
        executionArtifacts: typeExecutions,
      } as RunArtifacts);
    }
  }
  return typeRuns;
}

function displayMetricsFromRunArtifacts(runArtifacts: RunArtifacts[]) {
  return runArtifacts.length === 0 ? (
    <p>There are no artifacts of this type.</p>
  ) : (
    runArtifacts.map(x => {
      return (
        <div className={css.smallIndent}>
          <p>{x.run?.run?.name}</p>
          {x.executionArtifacts.map(y => {
            return (
              <div className={css.mediumIndent}>
                <p>
                  {y.execution
                    .getCustomPropertiesMap()
                    .get('display_name')
                    ?.getStringValue()}
                </p>
                <div className={css.largeIndent}>
                  {y.linkedArtifacts.map(z => {
                    return (
                      <p>
                        {z.event
                          .getPath()
                          ?.getStepsList()[0]
                          .getKey()}
                      </p>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      );
    })
  );
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
                  linkedArtifacts,
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

  // TODO: I still need to determine how to differentiate Confusion Matrices and ROC Curves.
  const scalarMetricsArtifacts = filterRunArtifactsByType(
    runArtifacts,
    artifactTypes,
    MetricsType.SCALAR_METRICS,
  );
  const confusionMatrixArtifacts = filterRunArtifactsByType(
    runArtifacts,
    artifactTypes,
    MetricsType.CONFUSION_MATRIX,
  );
  const rocCurveArtifacts = filterRunArtifactsByType(
    runArtifacts,
    artifactTypes,
    MetricsType.ROC_CURVE,
  );
  const htmlArtifacts = filterRunArtifactsByType(runArtifacts, artifactTypes, MetricsType.HTML);
  const markdownArtifacts = filterRunArtifactsByType(
    runArtifacts,
    artifactTypes,
    MetricsType.MARKDOWN,
  );

  console.log(scalarMetricsArtifacts);
  console.log(confusionMatrixArtifacts);
  console.log(rocCurveArtifacts);
  console.log(htmlArtifacts);
  console.log(markdownArtifacts);

  return (
    <div className={commonCss.page}>
      <p>This is the V2 Run Comparison page.</p>
      <div>Scalar Metrics</div>
      {displayMetricsFromRunArtifacts(scalarMetricsArtifacts)}
      <br />
      <div>Confusion Matrices</div>
      {displayMetricsFromRunArtifacts(confusionMatrixArtifacts)}
      <br />
      <div>ROC Curves</div>
      {displayMetricsFromRunArtifacts(rocCurveArtifacts)}
      <br />
      <div>HTML</div>
      {displayMetricsFromRunArtifacts(htmlArtifacts)}
      <br />
      <div>Markdown</div>
      {displayMetricsFromRunArtifacts(markdownArtifacts)}
    </div>
  );
}

export default CompareV2;
