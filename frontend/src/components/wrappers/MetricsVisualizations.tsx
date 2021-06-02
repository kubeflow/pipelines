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

import { Artifact, ArtifactType } from '@kubeflow/frontend';
import HelpIcon from '@material-ui/icons/Help';
import React from 'react';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import { color, padding } from 'src/Css';
import { filterArtifactsByType } from 'src/lib/MlmdUtils';
import Banner from '../Banner';
import ConfusionMatrix, { ConfusionMatrixConfig } from '../viewers/ConfusionMatrix';
import ROCCurve, { ROCCurveConfig } from '../viewers/ROCCurve';
import { PlotType } from '../viewers/Viewer';

interface MetricsVisualizationsProps {
  artifacts: Artifact[];
  artifactTypes: ArtifactType[];
}

/**
 * Visualize system metrics based on artifact input. There can be multiple artifacts
 * and multiple visualizations associated with one artifact.
 */
export function MetricsVisualizations({ artifacts, artifactTypes }: MetricsVisualizationsProps) {
  // system.ClassificationMetrics contains confusionMatrix or confidenceMetrics.
  // TODO: Visualize confusionMatrix using system.ClassificationMetrics artifacts.
  // https://github.com/kubeflow/pipelines/issues/5668
  let classificationMetricsArtifacts = filterArtifactsByType(
    'system.ClassificationMetrics',
    artifactTypes,
    artifacts,
  );

  // There can be multiple system.ClassificationMetrics artifacts per execution.
  // Get confidenceMetrics and confusionMatrix from artifact.
  // If there is no available metrics, show banner to notify users.
  // Otherwise, Visualize all available metrics per artifact.
  const metricsAvailableArtifacts = getMetricsAvailableArtifacts(classificationMetricsArtifacts);

  if (metricsAvailableArtifacts.length === 0) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
      {metricsAvailableArtifacts.map(artifact => {
        return (
          <React.Fragment key={artifact.getId()}>
            <ConfidenceMetricsSection artifact={artifact} />
            <ConfusionMatrixSection artifact={artifact} />
          </React.Fragment>
        );
      })}
    </>
  );
}

function getMetricsAvailableArtifacts(artifacts: Artifact[]): Artifact[] {
  if (!artifacts) {
    return [];
  }
  return artifacts
    .map(artifact => ({
      name: artifact
        .getCustomPropertiesMap()
        .get('name')
        ?.getStringValue(),
      customProperties: artifact.getCustomPropertiesMap(),
      artifact: artifact,
    }))
    .filter(x => !!x.name)
    .filter(x => {
      const confidenceMetrics = x.customProperties
        .get('confidenceMetrics')
        ?.getStructValue()
        ?.toJavaScript();

      const confusionMatrix = x.customProperties
        .get('confusionMatrix')
        ?.getStructValue()
        ?.toJavaScript();
      return !!confidenceMetrics || !!confusionMatrix;
    })
    .map(x => x.artifact);
}

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate. ' +
  'A lower threshold results in a higher true positive rate (and a higher false positive rate), ' +
  'while a higher threshold results in a lower true positive rate (and a lower false positive rate)';

type ConfidenceMetric = {
  confidenceThreshold: string;
  falsePositiveRate: number;
  recall: number;
};

interface ConfidenceMetricsSectionProps {
  artifact: Artifact;
}

function ConfidenceMetricsSection({ artifact }: ConfidenceMetricsSectionProps) {
  const customProperties = artifact.getCustomPropertiesMap();
  const name = customProperties.get('name')?.getStringValue();

  const confidenceMetrics = customProperties
    .get('confidenceMetrics')
    ?.getStructValue()
    ?.toJavaScript();
  if (confidenceMetrics === undefined) {
    return <></>;
  }

  const { error } = validateConfidenceMetrics((confidenceMetrics as any).list);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confidenceMetrics data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }
  return (
    <>
      {
        <div className={padding(40, 'lrt')}>
          <div className={padding(40, 'b')}>
            <h1>
              {'ROC Curve: ' + name}{' '}
              <IconWithTooltip
                Icon={HelpIcon}
                iconColor={color.weak}
                tooltip={ROC_CURVE_DEFINITION}
              ></IconWithTooltip>
            </h1>
          </div>
          <ROCCurve configs={buildRocCurveConfig((confidenceMetrics as any).list)} />
        </div>
      }
    </>
  );
}

function validateConfidenceMetrics(inputs: any): { error?: string } {
  if (!Array.isArray(inputs)) return { error: 'ConfidenceMetrics is not an array.' };
  for (let i = 0; i < inputs.length; i++) {
    const metric = inputs[i] as ConfidenceMetric;
    if (metric.confidenceThreshold == null)
      return { error: 'confidenceThreshold not found for item with index ' + i };
    if (metric.falsePositiveRate == null)
      return { error: 'falsePositiveRate not found for item with index ' + i };
    if (metric.recall == null) return { error: 'recall not found for item with index ' + i };
  }
  return {};
}

function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig[] {
  return [
    {
      type: PlotType.ROC,
      data: confidenceMetricsArray.map(metric => ({
        label: metric.confidenceThreshold,
        x: metric.falsePositiveRate,
        y: metric.recall,
      })),
    },
  ];
}

type AnnotationSpec = {
  displayName: string;
};
type Row = {
  row: number[];
};
type ConfusionMatrixInput = {
  annotationSpecs: AnnotationSpec[];
  rows: Row[];
};

interface ConfusionMatrixProps {
  artifact: Artifact;
}

const CONFUSION_MATRIX_DEFINITION =
  'The number of correct and incorrect predictions are ' +
  'summarized with count values and broken down by each class. ' +
  'The higher value on cell where Predicted label matches True label, ' +
  'the better prediction performance of this model is.';

function ConfusionMatrixSection({ artifact }: ConfusionMatrixProps) {
  const customProperties = artifact.getCustomPropertiesMap();
  const name = customProperties.get('name')?.getStringValue();

  const confusionMatrix = customProperties
    .get('confusionMatrix')
    ?.getStructValue()
    ?.toJavaScript();
  if (confusionMatrix === undefined) {
    return <></>;
  }

  const { error } = validateConfusionMatrix(confusionMatrix.struct as any);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confusionMatrix data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }
  return (
    <>
      {
        <div className={padding(40, 'lrt')}>
          <div className={padding(40, 'b')}>
            <h1>
              {'Confusion Matrix: ' + name}{' '}
              <IconWithTooltip
                Icon={HelpIcon}
                iconColor={color.weak}
                tooltip={CONFUSION_MATRIX_DEFINITION}
              ></IconWithTooltip>
            </h1>
          </div>
          <ConfusionMatrix configs={buildConfusionMatrixConfig(confusionMatrix.struct as any)} />
        </div>
      }
    </>
  );
}

function validateConfusionMatrix(input: any): { error?: string } {
  if (!input) return { error: 'confusionMatrix does not exist.' };
  const metric = input as ConfusionMatrixInput;

  if (!Array.isArray(metric.annotationSpecs)) return { error: 'annotationSpecs is not array.' };
  for (let i = 0; i < metric.annotationSpecs.length; i++) {
    if (metric.annotationSpecs[i].displayName == null) {
      return { error: 'displayName not found for item at index ' + i + ' of annotationSpecs' };
    }
  }

  if (!Array.isArray(metric.rows)) return { error: 'Data field rows is not array.' };
  const height = metric.rows.length;
  if (metric.annotationSpecs.length !== height)
    return {
      error:
        'annotationSpecs has different length ' +
        metric.annotationSpecs.length +
        ' than rows length ' +
        height,
    };
  for (let i = 0; i < height; i++) {
    if (!Array.isArray(metric.rows[i].row))
      return { error: 'row at index ' + i + ' is not array.' };
    const width = metric.rows[i].row.length;
    if (width !== height)
      return {
        error:
          'row at index ' + i + ' has ' + width + ' columns, but there are ' + height + ' rows.',
      };
    for (let j = 0; j < width; j++) {
      if (metric.rows[i].row[j] == null) {
        return { error: 'Data at index ' + i + ' row, ' + j + " column doesn't exist." };
      }
    }
  }

  return {};
}

function buildConfusionMatrixConfig(
  confusionMatrix: ConfusionMatrixInput,
): ConfusionMatrixConfig[] {
  return [
    {
      type: PlotType.CONFUSION_MATRIX,
      axes: ['True label', 'Predicted label'],
      labels: confusionMatrix.annotationSpecs.map(annotation => annotation.displayName),
      data: confusionMatrix.rows.map(x => x.row),
    },
  ];
}
