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

import { Artifact, ArtifactType, Execution } from '@kubeflow/frontend';
import HelpIcon from '@material-ui/icons/Help';
import * as React from 'react';
import { useState } from 'react';
import { useQuery } from 'react-query';
import { Link } from 'react-router-dom';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import { color, commonCss, padding } from 'src/Css';
import {
  ExecutionHelpers,
  filterArtifactsByType,
  getArtifactTypes,
  getOutputArtifactsInExecution,
} from 'src/lib/MlmdUtils';
import Banner from '../Banner';
import { RoutePageFactory } from '../Router';
import ROCCurve, { ROCCurveConfig } from '../viewers/ROCCurve';

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate. ' +
  'A lower threshold results in a higher true positive rate (and a higher false positive rate), ' +
  'while a higher threshold results in a lower true positive rate (and a lower false positive rate)';

type ConfidenceMetric = {
  confidenceThreshold: string;
  falsePositiveRate: number;
  recall: number;
};

type MetricsTabProps = {
  execution: Execution;
};

interface MetricsSwitcherProps {
  artifacts: Artifact[];
  artifactTypes: ArtifactType[];
  refresh?: () => void;
}

/**
 * Metrics tab renders metrics for the artifact of given execution.
 * Some system metrics are: Confusion Matrix, ROC Curve, Scalar, etc.
 * Detail can be found in https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py
 * Note that these metrics are only available on KFP v2 mode.
 */
export function MetricsTab({ execution }: MetricsTabProps) {
  const [refreshCount, setRefreshCount] = useState(0);
  function refresh() {
    setRefreshCount(refreshCount + 1);
  }
  // Retrieving a list of artifacts associated with this execution,
  // so we can find the artifact for system metrics from there.
  const {
    isLoading: isLoadingArtifacts,
    isSuccess: isSuccessArtifacts,
    error: errorArtifacts,
    data: artifacts,
  } = useQuery<Artifact[], Error>(['execution_output_artifact', { id: execution.getId() }], () =>
    getOutputArtifactsInExecution(execution),
  );

  // artifactTypes allows us to map from artifactIds to artifactTypeNames,
  // so we can identify metrics artifact provided by system.
  const {
    isLoading: isLoadingArtifactTypes,
    isSuccess: isSuccessArtifactTypes,
    error: errorArtifactTypes,
    data: artifactTypes,
  } = useQuery<ArtifactType[], Error>(['artifact_types'], () => getArtifactTypes());

  // This react element produces banner message if query to MLMD is pending or has error.
  // Once query is completed, it shows actual content of metrics visualization in MetricsSwitcher.
  return (
    <div className={commonCss.page}>
      <div className={padding(20)}>
        <div>
          This step corresponds to execution{' '}
          <Link
            className={commonCss.link}
            to={RoutePageFactory.executionDetails(execution.getId())}
          >
            "{ExecutionHelpers.getName(execution)}".
          </Link>
        </div>
        {isLoadingArtifactTypes && isLoadingArtifacts && (
          <Banner message='Metrics is loading.' mode='info' />
        )}
        {errorArtifacts && (
          <Banner
            message='Error in retrieving metrics information.'
            mode='error'
            additionalInfo={errorArtifacts.message}
            refresh={refresh}
          />
        )}
        {!errorArtifacts && errorArtifactTypes && (
          <Banner
            message='Error in retrieving artifact types information.'
            mode='error'
            additionalInfo={errorArtifactTypes.message}
            refresh={refresh}
          />
        )}
        {isSuccessArtifacts && isSuccessArtifactTypes && artifacts && artifactTypes && (
          <MetricsSwitcher
            artifacts={artifacts}
            artifactTypes={artifactTypes}
            refresh={refresh}
          ></MetricsSwitcher>
        )}
      </div>
    </div>
  );
}

/**
 * Visualize system metrics based on artifact input.
 */
function MetricsSwitcher({ artifacts, artifactTypes, refresh }: MetricsSwitcherProps) {
  // system.ClassificationMetrics contains confusionMatrix or confidenceMetrics.
  // TODO: Visualize confusionMatrix using system.ClassificationMetrics artifacts.
  // https://github.com/kubeflow/pipelines/issues/5668
  let classificationMetricsArtifacts = filterArtifactsByType(
    'system.ClassificationMetrics',
    artifactTypes,
    artifacts,
  );

  const confidenceMetricsArtifacts = classificationMetricsArtifacts
    .map(artifact => ({
      name: artifact
        .getCustomPropertiesMap()
        .get('name')
        ?.getStringValue(),
      customProperties: artifact.getCustomPropertiesMap(),
    }))
    .filter(x => !!x.name)
    .filter(x => {
      const confidenceMetrics = x.customProperties
        .get('confidenceMetrics')
        ?.getStructValue()
        ?.toJavaScript();
      return !!confidenceMetrics;
    });

  return (
    <>
      {(!confidenceMetricsArtifacts || confidenceMetricsArtifacts.length === 0) && (
        <Banner
          message='There is no metrics artifact available in this step.'
          mode='info'
          refresh={refresh}
          enableRefresh={refresh ? true : false}
        />
      )}
      {classificationMetricsArtifacts &&
        classificationMetricsArtifacts.length > 0 &&
        classificationMetricsArtifacts.map(artifact => {
          const id = artifact.getId();
          const customProperties = artifact.getCustomPropertiesMap();
          const confidenceMetrics = customProperties
            .get('confidenceMetrics')
            ?.getStructValue()
            ?.toJavaScript();

          return (
            <>
              {confidenceMetrics && (confidenceMetrics as any).list && (
                <div key={'confidenceMetrics' + id}>
                  <div className={padding(40, 'lrt')}>
                    <h1>
                      {'ROC Curve: ' + customProperties.get('name')?.getStringValue()}{' '}
                      <IconWithTooltip
                        Icon={HelpIcon}
                        iconColor={color.weak}
                        tooltip={ROC_CURVE_DEFINITION}
                      ></IconWithTooltip>
                    </h1>
                  </div>
                  <ROCCurve configs={buildRocCurveConfig((confidenceMetrics as any).list)} />
                </div>
              )}
            </>
          );
        })}
    </>
  );
}

function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig[] {
  return [
    {
      data: confidenceMetricsArray.map(metric => ({
        label: metric.confidenceThreshold,
        x: metric.falsePositiveRate,
        y: metric.recall,
      })),
    },
  ] as ROCCurveConfig[];
}
