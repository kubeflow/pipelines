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
import { FormControl, InputLabel, MenuItem, Select } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { ArtifactArtifactType, V2beta1Artifact } from 'src/apisv2beta1/run';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import Banner from 'src/components/Banner';
import PlotCard from 'src/components/PlotCard';
import { color, padding } from 'src/Css';
import { queryKeys } from 'src/hooks/queryKeys';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import {
  getArtifactDisplayName,
  getArtifactIdentity,
  getScalarMetricEntries,
  isHtmlArtifact,
  isLegacyUiMetadataArtifact,
  isMarkdownArtifact,
  isScalarMetricArtifact,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { parseArtifactFileLocation, readArtifactFile } from 'src/lib/v2/ArtifactFileUtils';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import { HTMLViewerConfig } from './HTMLViewer';
import { MarkdownViewerConfig } from './MarkdownViewer';
import PagedTable from './PagedTable';
import ROCCurve, { ROCCurveConfig } from './ROCCurve';
import { buildRocCurveConfig, validateConfidenceMetrics } from './ROCCurveHelper';
import { PlotType, ViewerConfig } from './Viewer';
import { componentMap } from './ViewerContainer';

interface RuntimeMetricsVisualizationsProps {
  artifacts: V2beta1Artifact[];
  artifactKey?: string;
  namespace?: string;
  sourceFinished?: boolean;
}

export interface ClassificationVisualization {
  key: string;
  displayName: string;
  metadata?: { [key: string]: object };
  sourceArtifact: V2beta1Artifact;
}

interface LegacyUiMetadataVisualizationResult {
  configs: ViewerConfig[];
  errors: string[];
}

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate.';

export function RuntimeMetricsVisualizations({
  artifacts,
  artifactKey,
  namespace,
  sourceFinished,
}: RuntimeMetricsVisualizationsProps) {
  const { scalarMetrics, classificationMetrics, fileArtifacts, legacyUiMetadataArtifacts } =
    useMemo(() => {
      const files = artifacts.filter(
        (artifact) => isHtmlArtifact(artifact) || isMarkdownArtifact(artifact),
      );
      return {
        scalarMetrics: artifacts.filter(isScalarMetricArtifact),
        classificationMetrics: expandClassificationMetrics(artifacts),
        fileArtifacts: files,
        legacyUiMetadataArtifacts: artifacts.filter((artifact) =>
          isLegacyUiMetadataArtifact(artifact, artifactKey),
        ),
      };
    }, [artifactKey, artifacts]);

  const rocCurves = useMemo(() => buildRocCurves(classificationMetrics), [classificationMetrics]);
  const confusionMatrixResult = useMemo(
    () => buildConfusionMatrixResult(classificationMetrics),
    [classificationMetrics],
  );

  if (
    scalarMetrics.length === 0 &&
    rocCurves.configs.length === 0 &&
    !rocCurves.error &&
    confusionMatrixResult.matrices.length === 0 &&
    confusionMatrixResult.errors.length === 0 &&
    fileArtifacts.length === 0 &&
    legacyUiMetadataArtifacts.length === 0
  ) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
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
      {!!confusionMatrixResult.errors.length && (
        <Banner
          message='Invalid confusion matrix artifact.'
          mode='error'
          additionalInfo={confusionMatrixResult.errors.join('\n')}
        />
      )}
      {confusionMatrixResult.matrices.map(({ visualization, configs }) => (
        <div className={padding(40)} key={visualization.key}>
          <h3>Confusion Matrix: {visualization.displayName}</h3>
          <ConfusionMatrix configs={configs} />
        </div>
      ))}
      {!!scalarMetrics.length && (
        <div className={padding(40, 'lrt')}>
          <h3>Scalar Metrics</h3>
          <PagedTable
            configs={[
              {
                data: scalarMetrics
                  .flatMap(getScalarMetricEntries)
                  .map(({ name, value }) => [name, value]),
                labels: ['name', 'value'],
                type: PlotType.TABLE,
              },
            ]}
          />
        </div>
      )}
      <FileArtifactVisualization
        artifacts={fileArtifacts.filter(isHtmlArtifact)}
        kind='HTML'
        namespace={namespace}
        sourceFinished={sourceFinished}
      />
      <FileArtifactVisualization
        artifacts={fileArtifacts.filter(isMarkdownArtifact)}
        kind='Markdown'
        namespace={namespace}
        sourceFinished={sourceFinished}
      />
      {legacyUiMetadataArtifacts.map((artifact, index) => (
        <LegacyUiMetadataVisualization
          artifact={artifact}
          key={getArtifactIdentity(artifact) || `legacy-ui-metadata-${index}`}
          namespace={artifact.namespace || namespace}
          sourceFinished={sourceFinished}
        />
      ))}
    </>
  );
}

function FileArtifactVisualization({
  artifacts,
  kind,
  namespace,
  sourceFinished,
}: {
  artifacts: V2beta1Artifact[];
  kind: 'HTML' | 'Markdown';
  namespace?: string;
  sourceFinished?: boolean;
}) {
  const entries = useMemo(() => {
    const identityOccurrences = new Map<string, number>();
    return artifacts.map((artifact) => {
      const identity = artifact.artifact_id
        ? JSON.stringify(['id', artifact.artifact_id])
        : JSON.stringify([
            'artifact',
            artifact.uri || '',
            artifact.namespace || '',
            artifact.name || '',
            artifact.type || '',
          ]);
      const occurrence = identityOccurrences.get(identity) || 0;
      identityOccurrences.set(identity, occurrence + 1);
      return {
        artifact,
        key: JSON.stringify([identity, occurrence]),
      };
    });
  }, [artifacts]);
  const [selectedKey, setSelectedKey] = useState('');
  const selectedEntry =
    entries.find(({ key }) => key === selectedKey) ||
    (entries.length === 1 ? entries[0] : undefined);
  const activeSelectedKey = selectedEntry?.key || '';
  const selectedArtifact = selectedEntry?.artifact;

  if (!entries.length) {
    return null;
  }

  if (entries.length === 1 && selectedArtifact) {
    return (
      <div className={padding(20, 'lrt')}>
        <RuntimeArtifactVisualization
          artifact={selectedArtifact}
          namespace={namespace}
          sourceFinished={sourceFinished}
        />
      </div>
    );
  }

  return (
    <div className={padding(20, 'lrt')}>
      <FormControl variant='standard' style={{ minWidth: 240 }}>
        <InputLabel id={`${kind.toLowerCase()}-visualization-label`}>
          {kind} visualization
        </InputLabel>
        <Select
          labelId={`${kind.toLowerCase()}-visualization-label`}
          value={activeSelectedKey}
          onChange={(event) => setSelectedKey(event.target.value as string)}
          inputProps={{ 'aria-label': `${kind} visualization` }}
        >
          <MenuItem value=''>Choose an artifact</MenuItem>
          {entries.map(({ artifact, key }) => (
            <MenuItem key={key} value={key}>
              {getArtifactDisplayName(artifact)}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      {selectedArtifact && (
        <RuntimeArtifactVisualization
          artifact={selectedArtifact}
          namespace={namespace}
          sourceFinished={sourceFinished}
        />
      )}
    </div>
  );
}

export function RuntimeArtifactVisualization({
  artifact,
  namespace,
  sourceFinished,
  title,
}: {
  artifact: V2beta1Artifact;
  namespace?: string;
  sourceFinished?: boolean;
  title?: string;
}) {
  const artifactKey = getArtifactIdentity(artifact);
  const effectiveNamespace = artifact.namespace || namespace;
  const { data, error, isLoading } = useQuery<ViewerConfig, Error>({
    queryKey: queryKeys.runtimeArtifactVisualization(
      artifactKey,
      effectiveNamespace,
      sourceFinished,
    ),
    queryFn: () => downloadVisualization(artifact, effectiveNamespace),
    retry: false,
    staleTime: Infinity,
  });
  return (
    <>
      {error && (
        <Banner
          message='Unable to retrieve the selected visualization. Verify the artifact URI and refresh the page to retry.'
          mode='error'
          additionalInfo={error.message}
        />
      )}
      {isLoading && <Banner message='Visualization is loading.' mode='info' />}
      {data && (
        <PlotCard
          configs={[data]}
          title={title || (isHtmlArtifact(artifact) ? 'Static HTML' : 'Static Markdown')}
        />
      )}
    </>
  );
}

function LegacyUiMetadataVisualization({
  artifact,
  namespace,
  sourceFinished,
}: {
  artifact: V2beta1Artifact;
  namespace?: string;
  sourceFinished?: boolean;
}) {
  const artifactKey = getArtifactIdentity(artifact);
  const { data, error, isLoading } = useQuery<LegacyUiMetadataVisualizationResult, Error>({
    queryKey: queryKeys.legacyRuntimeUiMetadata(artifactKey, namespace, sourceFinished),
    queryFn: () => loadLegacyUiMetadataVisualization(artifact, namespace),
    retry: false,
    staleTime: Infinity,
  });
  const supportedConfigs = data?.configs.filter((config) => !!componentMap[config.type]);
  const containsUnsupportedConfig = supportedConfigs?.length !== data?.configs.length;
  return (
    <div className={padding(20, 'lrt')}>
      {error && (
        <Banner
          message='Unable to retrieve legacy UI visualizations. Verify the metadata artifact and its referenced sources, then refresh the page to retry.'
          mode='error'
          additionalInfo={error.message}
        />
      )}
      {isLoading && <Banner message='Legacy UI visualizations are loading.' mode='info' />}
      {!!data?.errors.length && (
        <Banner
          message='Some legacy UI visualizations could not be loaded.'
          mode='error'
          additionalInfo={data.errors.join('\n')}
        />
      )}
      {data?.configs.length === 0 && data.errors.length === 0 && (
        <Banner
          message='The legacy UI metadata artifact contains no supported visualizations.'
          mode='info'
        />
      )}
      {containsUnsupportedConfig && (
        <Banner
          message='The legacy UI metadata contains an unsupported visualization type. Update the metadata to use a supported viewer and refresh the page.'
          mode='error'
        />
      )}
      {supportedConfigs?.map((config, index) => (
        <PlotCard
          configs={[config]}
          key={`${config.type}-${index}`}
          title={componentMap[config.type].prototype.getDisplayName()}
        />
      ))}
    </div>
  );
}

async function loadLegacyUiMetadataVisualization(
  artifact: V2beta1Artifact,
  namespace?: string,
): Promise<LegacyUiMetadataVisualizationResult> {
  if (!artifact.uri) {
    throw new Error(
      `${getArtifactDisplayName(artifact)} has no URI. Verify that the component produced the UI metadata artifact at a valid location.`,
    );
  }
  const location = parseArtifactFileLocation(artifact.uri);
  return OutputArtifactLoader.loadResult(location.path, namespace, {
    throwOnError: true,
    artifactUriQuery: location.artifactUriQuery,
  });
}

export function buildRocCurves(visualizations: ClassificationVisualization[]): {
  configs: ROCCurveConfig[];
  error?: string;
} {
  const configs: ROCCurveConfig[] = [];
  const errors: string[] = [];
  for (const visualization of visualizations) {
    const confidenceMetrics = unwrapList(visualization.metadata?.confidenceMetrics);
    if (!confidenceMetrics) {
      continue;
    }
    const { error } = validateConfidenceMetrics(confidenceMetrics);
    if (error) {
      errors.push(`${visualization.displayName}: ${error}`);
      continue;
    }
    configs.push(
      buildRocCurveConfig(confidenceMetrics as Parameters<typeof buildRocCurveConfig>[0]),
    );
  }
  return { configs, error: errors.length ? errors.join('\n') : undefined };
}

export function expandClassificationMetrics(
  artifacts: V2beta1Artifact[],
): ClassificationVisualization[] {
  return artifacts.flatMap((artifact, artifactIndex) => {
    const sourceKey = getArtifactIdentity(artifact) || `classification-${artifactIndex}`;
    if (artifact.type === ArtifactArtifactType.ClassificationMetric) {
      return [
        {
          key: sourceKey,
          displayName: getArtifactDisplayName(artifact),
          metadata: artifact.metadata,
          sourceArtifact: artifact,
        },
      ];
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
          key: `${sourceKey}:slice:${index}`,
          displayName: `${getArtifactDisplayName(artifact)} · ${sliceName}`,
          metadata: sliceMetrics as { [key: string]: object },
          sourceArtifact: artifact,
        },
      ];
    });
  });
}

export function buildConfusionMatrixResult(visualizations: ClassificationVisualization[]): {
  matrices: Array<{
    visualization: ClassificationVisualization;
    configs: ConfusionMatrixConfig[];
  }>;
  errors: string[];
} {
  const matrices: Array<{
    visualization: ClassificationVisualization;
    configs: ConfusionMatrixConfig[];
  }> = [];
  const errors: string[] = [];

  for (const visualization of visualizations) {
    const rawMatrix = visualization.metadata?.confusionMatrix;
    if (rawMatrix === undefined) {
      continue;
    }
    const matrix = unwrapStruct(rawMatrix);
    const error = validateConfusionMatrix(matrix);
    if (error) {
      errors.push(`${visualization.displayName}: ${error}`);
      continue;
    }
    const validMatrix = matrix as ConfusionMatrixValue;
    matrices.push({
      visualization,
      configs: [
        {
          type: PlotType.CONFUSION_MATRIX,
          axes: ['True label', 'Predicted label'],
          labels: validMatrix.annotationSpecs.map((annotation) => annotation.displayName),
          data: validMatrix.rows.map((row) => row.row),
        },
      ],
    });
  }
  return { errors, matrices };
}

async function downloadVisualization(
  artifact: V2beta1Artifact,
  namespace?: string,
): Promise<ViewerConfig> {
  if (!artifact.uri) {
    throw new Error(
      `${getArtifactDisplayName(artifact)} has no URI. Verify that the component produced a valid artifact location.`,
    );
  }
  const content = await readArtifactFile(artifact, namespace);
  if (isHtmlArtifact(artifact)) {
    return { htmlContent: content, type: PlotType.WEB_APP } as HTMLViewerConfig;
  }
  return { markdownContent: content, type: PlotType.MARKDOWN } as MarkdownViewerConfig;
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

function validateConfusionMatrix(value: unknown): string | undefined {
  if (!isRecord(value) || !Array.isArray(value.annotationSpecs) || !Array.isArray(value.rows)) {
    return 'confusionMatrix must contain annotationSpecs and rows arrays. Correct the logged metric data and rerun the pipeline.';
  }
  if (
    !value.annotationSpecs.every(
      (annotation) => isRecord(annotation) && typeof annotation.displayName === 'string',
    )
  ) {
    return 'every annotationSpec must have a string displayName. Correct the logged metric data and rerun the pipeline.';
  }
  if (value.annotationSpecs.length !== value.rows.length) {
    return `annotationSpecs has length ${value.annotationSpecs.length}, but rows has length ${value.rows.length}. Log one row per annotation and rerun the pipeline.`;
  }
  for (const row of value.rows) {
    if (!isRecord(row) || !Array.isArray(row.row)) {
      return 'every confusion matrix row must contain a row array. Correct the logged metric data and rerun the pipeline.';
    }
    if (row.row.length !== value.rows.length) {
      return `a confusion matrix row has length ${row.row.length}, but the matrix dimension is ${value.rows.length}. Log a square matrix and rerun the pipeline.`;
    }
    if (!row.row.every(Number.isFinite)) {
      return 'confusion matrix cells must be finite numbers. Correct the logged metric data and rerun the pipeline.';
    }
  }
  return undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export const TEST_ONLY = {
  downloadVisualization,
  loadLegacyUiMetadataVisualization,
};

export default RuntimeMetricsVisualizations;
