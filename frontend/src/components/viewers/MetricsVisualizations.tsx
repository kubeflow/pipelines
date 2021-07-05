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

import HelpIcon from '@material-ui/icons/Help';
import React from 'react';
import { useQuery } from 'react-query';
import { Array as ArrayRunType, Failure, Number, Record, String, ValidationError } from 'runtypes';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import { color, padding } from 'src/Css';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import WorkflowParser from 'src/lib/WorkflowParser';
import { getMetadataValue } from 'src/mlmd/library';
import {
  filterArtifactsByType,
  filterLinkedArtifactsByType,
  getArtifactName,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { Artifact, ArtifactType, Execution } from 'src/third_party/mlmd';
import Banner from '../Banner';
import PlotCard from '../PlotCard';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import PagedTable from './PagedTable';
import ROCCurve, { ROCCurveConfig } from './ROCCurve';
import { PlotType, ViewerConfig } from './Viewer';
import { componentMap } from './ViewerContainer';

interface MetricsVisualizationsProps {
  linkedArtifacts: LinkedArtifact[];
  artifactTypes: ArtifactType[];
  execution: Execution;
  namespace: string | undefined;
}

/**
 * Visualize system metrics based on artifact input. There can be multiple artifacts
 * and multiple visualizations associated with one artifact.
 */
export function MetricsVisualizations({
  linkedArtifacts,
  artifactTypes,
  execution,
  namespace,
}: MetricsVisualizationsProps) {
  // There can be multiple system.ClassificationMetrics or system.Metrics artifacts per execution.
  // Get scalar metrics, confidenceMetrics and confusionMatrix from artifact.
  // If there is no available metrics, show banner to notify users.
  // Otherwise, Visualize all available metrics per artifact.
  const artifacts = linkedArtifacts.map(x => x.artifact);
  const verifiedClassificationMetricsArtifacts = getVerifiedClassificationMetricsArtifacts(
    artifacts,
    artifactTypes,
  );
  const verifiedMetricsArtifacts = getVerifiedMetricsArtifacts(artifacts, artifactTypes);
  const v1VisualizationArtifact = getV1VisualizationArtifacts(linkedArtifacts, artifactTypes);

  const {
    isSuccess: isV1ViewerConfigsSuccess,
    error: v1ViewerConfigError,
    data: v1ViewerConfigs,
  } = useQuery<ViewerConfig[], Error>(
    [
      'viewconfig',
      {
        artifact: v1VisualizationArtifact?.artifact.getId(),
        state: execution.getLastKnownState(),
        namespace: namespace,
      },
    ],
    () => getViewConfig(v1VisualizationArtifact, namespace),
    { staleTime: Infinity },
  );

  if (
    verifiedClassificationMetricsArtifacts.length === 0 &&
    verifiedMetricsArtifacts.length === 0 &&
    !v1VisualizationArtifact
  ) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
      {v1ViewerConfigError && (
        <Banner
          message='Error in retrieving v1 metrics information.'
          mode='error'
          additionalInfo={v1ViewerConfigError.message}
        />
      )}
      {verifiedClassificationMetricsArtifacts.map(artifact => {
        return (
          <React.Fragment key={artifact.getId()}>
            <ConfidenceMetricsSection artifact={artifact} />
            <ConfusionMatrixSection artifact={artifact} />
          </React.Fragment>
        );
      })}
      {verifiedMetricsArtifacts.map(artifact => (
        <ScalarMetricsSection artifact={artifact} key={artifact.getId()} />
      ))}
      {isV1ViewerConfigsSuccess &&
        v1ViewerConfigs &&
        v1ViewerConfigs.map((config, i) => {
          const title = componentMap[config.type].prototype.getDisplayName();
          return (
            <div key={i} className={padding(20, 'lrt')}>
              <PlotCard configs={[config]} title={title} />
            </div>
          );
        })}
    </>
  );
}

function getVerifiedClassificationMetricsArtifacts(
  artifacts: Artifact[],
  artifactTypes: ArtifactType[],
): Artifact[] {
  if (!artifacts || !artifactTypes) {
    return [];
  }
  // Reference: https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py#L124
  // system.ClassificationMetrics contains confusionMatrix or confidenceMetrics.
  const classificationMetricsArtifacts = filterArtifactsByType(
    'system.ClassificationMetrics',
    artifactTypes,
    artifacts,
  );

  return classificationMetricsArtifacts
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

function getVerifiedMetricsArtifacts(
  artifacts: Artifact[],
  artifactTypes: ArtifactType[],
): Artifact[] {
  if (!artifacts || !artifactTypes) {
    return [];
  }
  // Reference: https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py#L104
  // system.Metrics contains scalar metrics.
  const metricsArtifacts = filterArtifactsByType('system.Metrics', artifactTypes, artifacts);

  return metricsArtifacts.filter(x =>
    x
      .getCustomPropertiesMap()
      .get('name')
      ?.getStringValue(),
  );
}

function getV1VisualizationArtifacts(
  linkedArtifacts: LinkedArtifact[],
  artifactTypes: ArtifactType[],
): LinkedArtifact | undefined {
  const systemArtifacts = filterLinkedArtifactsByType(
    'system.Artifact',
    artifactTypes,
    linkedArtifacts,
  );

  const v1VisualizationArtifacts = systemArtifacts.filter(x => {
    if (!x) {
      return false;
    }
    const artifactName = getArtifactName(x);
    // This is a hack to find mlpipeline-ui-metadata artifact for visualization.
    const updatedName = artifactName?.replace(/[\W_]/g, '-').toLowerCase();
    return updatedName === 'mlpipeline-ui-metadata';
  });

  if (v1VisualizationArtifacts.length > 1) {
    throw new Error(
      'There are more than 1 mlpipeline-ui-metadata artifact: ' +
        JSON.stringify(v1VisualizationArtifacts),
    );
  }
  return v1VisualizationArtifacts.length === 0 ? undefined : v1VisualizationArtifacts[0];
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
    return null;
  }

  const { error } = validateConfidenceMetrics((confidenceMetrics as any).list);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confidenceMetrics data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }
  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>
          {'ROC Curve: ' + name}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={ROC_CURVE_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      <ROCCurve configs={buildRocCurveConfig((confidenceMetrics as any).list)} />
    </div>
  );
}

const ConfidenceMetricRunType = Record({
  confidenceThreshold: Number,
  falsePositiveRate: Number,
  recall: Number,
});
const ConfidenceMetricArrayRunType = ArrayRunType(ConfidenceMetricRunType);
function validateConfidenceMetrics(inputs: any): { error?: string } {
  try {
    ConfidenceMetricArrayRunType.check(inputs);
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(inputs) };
    }
  }
  return {};
}

function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig[] {
  const arraytypesCheck = ConfidenceMetricArrayRunType.check(confidenceMetricsArray);
  return [
    {
      type: PlotType.ROC,
      data: arraytypesCheck.map(metric => ({
        label: (metric.confidenceThreshold as unknown) as string,
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
    return null;
  }

  const { error } = validateConfusionMatrix(confusionMatrix.struct as any);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confusionMatrix data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }
  return (
    <div className={padding(40)}>
      <div className={padding(40, 'b')}>
        <h3>
          {'Confusion Matrix: ' + name}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={CONFUSION_MATRIX_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      <ConfusionMatrix configs={buildConfusionMatrixConfig(confusionMatrix.struct as any)} />
    </div>
  );
}

const ConfusionMatrixInputRunType = Record({
  annotationSpecs: ArrayRunType(
    Record({
      displayName: String,
    }),
  ),
  rows: ArrayRunType(Record({ row: ArrayRunType(Number) })),
});
function validateConfusionMatrix(input: any): { error?: string } {
  if (!input) return { error: 'confusionMatrix does not exist.' };
  try {
    const matrix = ConfusionMatrixInputRunType.check(input);
    const height = matrix.rows.length;
    const annotationLen = matrix.annotationSpecs.length;
    if (annotationLen !== height) {
      throw new ValidationError({
        message:
          'annotationSpecs has different length ' + annotationLen + ' than rows length ' + height,
      } as Failure);
    }
    for (let x of matrix.rows) {
      if (x.row.length !== height)
        throw new ValidationError({
          message: 'row: ' + JSON.stringify(x) + ' has different length of columns from rows.',
        } as Failure);
    }
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(input) };
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

interface ScalarMetricsSectionProps {
  artifact: Artifact;
}
function ScalarMetricsSection({ artifact }: ScalarMetricsSectionProps) {
  const customProperties = artifact.getCustomPropertiesMap();
  const name = customProperties.get('name')?.getStringValue();
  const data = customProperties
    .getEntryList()
    .map(([key]) => ({
      key,
      value: JSON.stringify(getMetadataValue(customProperties.get(key))),
    }))
    .filter(metric => metric.key !== 'name');

  if (data.length === 0) {
    return null;
  }
  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>{'Scalar Metrics: ' + name}</h3>
      </div>
      <PagedTable
        configs={[
          {
            data: data.map(d => [d.key, d.value]),
            labels: ['name', 'value'],
            type: PlotType.TABLE,
          },
        ]}
      />
    </div>
  );
}
async function getViewConfig(
  v1VisualizationArtifact: LinkedArtifact | undefined,
  namespace: string | undefined,
): Promise<ViewerConfig[]> {
  if (v1VisualizationArtifact) {
    return OutputArtifactLoader.load(
      WorkflowParser.parseStoragePath(v1VisualizationArtifact.artifact.getUri()),
      namespace,
    );
  }
  return [];
}
