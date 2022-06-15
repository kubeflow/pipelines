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

// Include only the runs and executions which have artifacts of the specified type.
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

            const executionArtifacts = await Promise.all(
              executions.map(async execution => {
                const linkedArtifacts = await getOutputLinkedArtifactsInExecution(execution);
                return {
                  execution,
                  linkedArtifacts,
                } as ExecutionArtifacts;
              }),
            );

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

  // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // so we can identify metrics artifact provided by system.
  const { data: artifactTypes } = useQuery<ArtifactType[], Error>(
    ['artifact_types', {}],
    () => getArtifactTypes(),
    {
      staleTime: Infinity,
      onError: error =>
        props.updateBanner({
          message: 'Cannot get Artifact Types for MLMD.',
          additionalInfo: error.message,
          mode: 'error',
        }),
    },
  );

  if (!runArtifacts || !artifactTypes) {
    return (
      <div className={commonCss.page}>
        <p>This is the V2 Run Comparison page.</p>
      </div>
    );
  }

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

  console.log('Scalar Metrics');
  console.log(scalarMetricsArtifacts);
  console.log('Confusion Matrix');
  console.log(confusionMatrixArtifacts);
  console.log('ROC Curve');
  console.log(rocCurveArtifacts);
  console.log('HTML');
  console.log(htmlArtifacts);
  console.log('Markdown');
  console.log(markdownArtifacts);

  return (
    <div className={commonCss.page}>
      <p>This is the V2 Run Comparison page.</p>
    </div>
  );
}

export default CompareV2;
