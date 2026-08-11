// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import HelpIcon from '@mui/icons-material/Help';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { ArtifactArtifactType, V2beta1Artifact } from 'src/apisv2beta1/run';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import Banner from 'src/components/Banner';
import PlotCard from 'src/components/PlotCard';
import { color, padding } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { getArtifactDisplayName, getArtifactSessionInfo } from 'src/lib/v2/RuntimeArtifactUtils';
import WorkflowParser from 'src/lib/WorkflowParser';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import { HTMLViewerConfig } from './HTMLViewer';
import { MarkdownViewerConfig } from './MarkdownViewer';
import PagedTable from './PagedTable';
import ROCCurve, { ROCCurveConfig } from './ROCCurve';
import { buildRocCurveConfig, validateConfidenceMetrics } from './ROCCurveHelper';
import { PlotType } from './Viewer';

interface RuntimeMetricsVisualizationsProps {
  artifacts: V2beta1Artifact[];
  namespace?: string;
}

interface DownloadedVisualizations {
  html: HTMLViewerConfig[];
  markdown: MarkdownViewerConfig[];
}

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate.';

export function RuntimeMetricsVisualizations({
  artifacts,
  namespace,
}: RuntimeMetricsVisualizationsProps) {
  const scalarMetrics = artifacts.filter(
    (artifact) => artifact.type === ArtifactArtifactType.Metric,
  );
  const classificationMetrics = expandClassificationMetrics(artifacts);
  const fileArtifacts = artifacts.filter(
    (artifact) =>
      artifact.type === ArtifactArtifactType.HTML ||
      artifact.type === ArtifactArtifactType.Markdown,
  );
  const fileArtifactIds = fileArtifacts.map(
    (artifact) => artifact.artifact_id || artifact.uri || '',
  );

  const {
    data: downloaded,
    error: downloadError,
    isLoading,
  } = useQuery<DownloadedVisualizations, Error>({
    queryKey: queryKeys.runtimeArtifactVisualizations(fileArtifactIds, namespace),
    queryFn: () => downloadVisualizations(fileArtifacts, namespace),
    enabled: fileArtifacts.length > 0,
    staleTime: Infinity,
  });

  const rocCurves = useMemo(() => buildRocCurves(classificationMetrics), [classificationMetrics]);
  const confusionMatrices = useMemo(
    () => buildConfusionMatrices(classificationMetrics),
    [classificationMetrics],
  );

  if (
    scalarMetrics.length === 0 &&
    rocCurves.configs.length === 0 &&
    confusionMatrices.length === 0 &&
    fileArtifacts.length === 0
  ) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
      {downloadError && (
        <Banner
          message='Error in retrieving visualization information.'
          mode='error'
          additionalInfo={downloadError.message}
        />
      )}
      {isLoading && <Banner message='Visualization is loading.' mode='info' />}
      {rocCurves.error && (
        <Banner
          message='Invalid ROC curve artifact.'
          mode='error'
          additionalInfo={rocCurves.error}
        />
      )}
      {!!rocCurves.configs.length && (
        <div className={padding(40, 'lrt')}>
          <div className={padding(40, 'b')}>
            <h3>
              ROC Curve{' '}
              <IconWithTooltip
                Icon={HelpIcon}
                iconColor={color.weak}
                tooltip={ROC_CURVE_DEFINITION}
              />
            </h3>
          </div>
          <ROCCurve configs={rocCurves.configs} forceLegend={rocCurves.configs.length > 1} />
        </div>
      )}
      {confusionMatrices.map(({ artifact, configs }) => (
        <div className={padding(40)} key={artifact.artifact_id || artifact.name}>
          <h3>Confusion Matrix: {getArtifactDisplayName(artifact)}</h3>
          <ConfusionMatrix configs={configs} />
        </div>
      ))}
      {!!scalarMetrics.length && (
        <div className={padding(40, 'lrt')}>
          <h3>Scalar Metrics</h3>
          <PagedTable
            configs={[
              {
                data: scalarMetrics.map((artifact) => [
                  artifact.name || '-',
                  String(artifact.number_value ?? artifact.metadata?.[artifact.name || ''] ?? '-'),
                ]),
                labels: ['name', 'value'],
                type: PlotType.TABLE,
              },
            ]}
          />
        </div>
      )}
      {!!downloaded?.html.length && (
        <div className={padding(20, 'lrt')}>
          <PlotCard configs={downloaded.html} title='Static HTML' />
        </div>
      )}
      {!!downloaded?.markdown.length && (
        <div className={padding(20, 'lrt')}>
          <PlotCard configs={downloaded.markdown} title='Static Markdown' />
        </div>
      )}
    </>
  );
}

function buildRocCurves(artifacts: V2beta1Artifact[]): {
  configs: ROCCurveConfig[];
  error?: string;
} {
  const configs: ROCCurveConfig[] = [];
  for (const artifact of artifacts) {
    const confidenceMetrics = unwrapList(artifact.metadata?.confidenceMetrics);
    if (!confidenceMetrics) {
      continue;
    }
    const { error } = validateConfidenceMetrics(confidenceMetrics);
    if (error) {
      return { configs, error: `${getArtifactDisplayName(artifact)}: ${error}` };
    }
    configs.push(
      buildRocCurveConfig(confidenceMetrics as Parameters<typeof buildRocCurveConfig>[0]),
    );
  }
  return { configs };
}

function expandClassificationMetrics(artifacts: V2beta1Artifact[]): V2beta1Artifact[] {
  return artifacts.flatMap((artifact) => {
    if (artifact.type === ArtifactArtifactType.ClassificationMetric) {
      return [artifact];
    }
    if (artifact.type !== ArtifactArtifactType.SlicedClassificationMetric) {
      return [];
    }
    const slices = unwrapList(artifact.metadata?.evaluationSlices) || [];
    return slices.flatMap((rawSlice, index) => {
      const slice = unwrapStruct(rawSlice);
      if (!isRecord(slice)) {
        return [];
      }
      const sliceMetrics = unwrapStruct(slice.sliceClassificationMetrics);
      if (!isRecord(sliceMetrics)) {
        return [];
      }
      const sliceName = typeof slice.slice === 'string' ? slice.slice : `Slice ${index + 1}`;
      return [
        {
          ...artifact,
          artifact_id: `${artifact.artifact_id || artifact.name || 'sliced-metric'}:${sliceName}`,
          name: `${getArtifactDisplayName(artifact)} · ${sliceName}`,
          type: ArtifactArtifactType.ClassificationMetric,
          metadata: sliceMetrics as { [key: string]: object },
        },
      ];
    });
  });
}

function buildConfusionMatrices(
  artifacts: V2beta1Artifact[],
): Array<{ artifact: V2beta1Artifact; configs: ConfusionMatrixConfig[] }> {
  return artifacts.flatMap((artifact) => {
    const matrix = unwrapStruct(artifact.metadata?.confusionMatrix);
    if (!isConfusionMatrix(matrix)) {
      return [];
    }
    return [
      {
        artifact,
        configs: [
          {
            type: PlotType.CONFUSION_MATRIX,
            axes: ['True label', 'Predicted label'],
            labels: matrix.annotationSpecs.map((annotation) => annotation.displayName),
            data: matrix.rows.map((row) => row.row),
          },
        ],
      },
    ];
  });
}

async function downloadVisualizations(
  artifacts: V2beta1Artifact[],
  namespace?: string,
): Promise<DownloadedVisualizations> {
  const downloaded = await Promise.all(
    artifacts.map(async (artifact) => {
      if (!artifact.uri) {
        throw new Error(`${getArtifactDisplayName(artifact)} has no URI.`);
      }
      const content = await Apis.readFile({
        path: WorkflowParser.parseStoragePath(artifact.uri),
        providerInfo: getArtifactSessionInfo(artifact),
        namespace,
      });
      return { artifact, content };
    }),
  );
  return {
    html: downloaded
      .filter(({ artifact }) => artifact.type === ArtifactArtifactType.HTML)
      .map(({ content }) => ({ htmlContent: content, type: PlotType.WEB_APP })),
    markdown: downloaded
      .filter(({ artifact }) => artifact.type === ArtifactArtifactType.Markdown)
      .map(({ content }) => ({ markdownContent: content, type: PlotType.MARKDOWN })),
  };
}

function unwrapList(value: object | undefined): unknown[] | undefined {
  if (Array.isArray(value)) {
    return value;
  }
  if (isRecord(value) && Array.isArray(value.list)) {
    return value.list;
  }
  return undefined;
}

function unwrapStruct(value: unknown): unknown {
  return isRecord(value) && 'struct' in value ? value.struct : value;
}

interface ConfusionMatrixValue {
  annotationSpecs: Array<{ displayName: string }>;
  rows: Array<{ row: number[] }>;
}

function isConfusionMatrix(value: unknown): value is ConfusionMatrixValue {
  if (!isRecord(value) || !Array.isArray(value.annotationSpecs) || !Array.isArray(value.rows)) {
    return false;
  }
  return (
    value.annotationSpecs.every(
      (annotation) => isRecord(annotation) && typeof annotation.displayName === 'string',
    ) &&
    value.rows.every(
      (row) => isRecord(row) && Array.isArray(row.row) && row.row.every(Number.isFinite),
    )
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export const TEST_ONLY = {
  buildConfusionMatrices,
  buildRocCurves,
  downloadVisualizations,
  expandClassificationMetrics,
};

export default RuntimeMetricsVisualizations;
