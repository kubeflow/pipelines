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

import { ArtifactArtifactType, V2beta1Artifact } from 'src/apisv2beta1/run';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import { StorageService } from 'src/lib/WorkflowParser';
import { CommonTestWrapper } from 'src/TestWrapper';
import { PlotType, ViewerConfig } from './Viewer';
import {
  buildConfusionMatrices,
  buildConfusionMatrixResult,
  buildRocCurves,
  expandClassificationMetrics,
  RuntimeMetricsVisualizations,
  TEST_ONLY,
} from './RuntimeMetricsVisualizations';

describe('RuntimeMetricsVisualizations', () => {
  it('builds ROC curve configurations from native artifact metadata', () => {
    const artifact: V2beta1Artifact = {
      artifact_id: 'roc-1',
      name: 'roc',
      type: ArtifactArtifactType.ClassificationMetric,
      metadata: {
        confidenceMetrics: {
          list: [
            { confidenceThreshold: 0.8, falsePositiveRate: 0.1, recall: 0.9 },
            { confidenceThreshold: 0.5, falsePositiveRate: 0.3, recall: 1 },
          ],
        },
      },
    };

    expect(buildRocCurves(expandClassificationMetrics([artifact]))).toEqual({
      configs: [
        {
          type: PlotType.ROC,
          data: [
            { label: 0.8, x: 0.1, y: 0.9 },
            { label: 0.5, x: 0.3, y: 1 },
          ],
        },
      ],
    });
  });

  it('builds confusion matrices from wrapped and direct native metadata', () => {
    const matrix = {
      annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
      rows: [{ row: [3, 1] }, { row: [0, 4] }],
    };
    const artifacts: V2beta1Artifact[] = [
      {
        artifact_id: 'matrix-1',
        name: 'wrapped',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: { confusionMatrix: { struct: matrix } },
      },
      {
        artifact_id: 'matrix-2',
        name: 'direct',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: { confusionMatrix: matrix },
      },
    ];

    const visualizations = expandClassificationMetrics(artifacts);
    expect(buildConfusionMatrices(visualizations)).toEqual([
      {
        visualization: {
          key: 'matrix-1',
          displayName: 'wrapped',
          metadata: artifacts[0].metadata,
          sourceArtifact: artifacts[0],
        },
        configs: [
          {
            type: PlotType.CONFUSION_MATRIX,
            axes: ['True label', 'Predicted label'],
            labels: ['cat', 'dog'],
            data: [
              [3, 1],
              [0, 4],
            ],
          },
        ],
      },
      {
        visualization: {
          key: 'matrix-2',
          displayName: 'direct',
          metadata: artifacts[1].metadata,
          sourceArtifact: artifacts[1],
        },
        configs: [
          {
            type: PlotType.CONFUSION_MATRIX,
            axes: ['True label', 'Predicted label'],
            labels: ['cat', 'dog'],
            data: [
              [3, 1],
              [0, 4],
            ],
          },
        ],
      },
    ]);
  });

  it('reports dimension and cell errors in malformed confusion matrices', () => {
    const artifacts: V2beta1Artifact[] = [
      {
        artifact_id: 'matrix-dimension',
        name: 'wrong dimensions',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: {
          confusionMatrix: {
            annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
            rows: [{ row: [1, 0] }],
          },
        },
      },
      {
        artifact_id: 'matrix-cell',
        name: 'wrong cells',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: {
          confusionMatrix: {
            annotationSpecs: [{ displayName: 'cat' }],
            rows: [{ row: ['many'] }],
          },
        },
      },
    ];

    expect(buildConfusionMatrixResult(expandClassificationMetrics(artifacts))).toEqual({
      errors: [
        'wrong dimensions: annotationSpecs has length 2, but rows has length 1. Log one row per annotation and rerun the pipeline.',
        'wrong cells: confusion matrix cells must be finite numbers. Correct the logged metric data and rerun the pipeline.',
      ],
      matrices: [],
    });
  });

  it('renders an explicit error for an invalid confusion matrix', () => {
    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              name: 'broken matrix',
              type: ArtifactArtifactType.ClassificationMetric,
              metadata: {
                confusionMatrix: {
                  annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
                  rows: [{ row: [1, 0] }],
                },
              },
            },
          ]}
        />
      </CommonTestWrapper>,
    );

    screen.getByText('Invalid confusion matrix artifact.');
    fireEvent.click(screen.getByText('Details'));
    screen.getByText(/annotationSpecs has length 2, but rows has length 1/);
    expect(screen.queryByText('There is no metrics artifact available in this step.')).toBeNull();
  });

  it('expands sliced classification metrics into one visualization artifact per slice', () => {
    const artifact: V2beta1Artifact = {
      artifact_id: 'sliced-1',
      name: 'evaluation',
      type: ArtifactArtifactType.SlicedClassificationMetric,
      metadata: {
        evaluationSlices: [
          {
            slice: 'country=US',
            sliceClassificationMetrics: {
              confidenceMetrics: [
                { confidenceThreshold: 0.5, falsePositiveRate: 0.1, recall: 0.9 },
              ],
            },
          },
          {
            slice: 'country=CA',
            sliceClassificationMetrics: {
              confusionMatrix: {
                annotationSpecs: [{ displayName: 'yes' }, { displayName: 'no' }],
                rows: [{ row: [1, 0] }, { row: [0, 1] }],
              },
            },
          },
        ],
      },
    };

    expect(expandClassificationMetrics([artifact])).toEqual([
      {
        key: 'sliced-1:slice:0',
        displayName: 'evaluation · country=US',
        metadata: {
          confidenceMetrics: [{ confidenceThreshold: 0.5, falsePositiveRate: 0.1, recall: 0.9 }],
        },
        sourceArtifact: artifact,
      },
      {
        key: 'sliced-1:slice:1',
        displayName: 'evaluation · country=CA',
        metadata: {
          confusionMatrix: {
            annotationSpecs: [{ displayName: 'yes' }, { displayName: 'no' }],
            rows: [{ row: [1, 0] }, { row: [0, 1] }],
          },
        },
        sourceArtifact: artifact,
      },
    ]);
  });

  it('does not download multiple files of one type until one is selected', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile');
    const artifacts: V2beta1Artifact[] = [
      {
        artifact_id: 'html-1',
        name: 'report',
        type: ArtifactArtifactType.HTML,
        uri: 's3://reports/output.html',
        metadata: { store_session_info: 'stale-session' } as any,
      },
      {
        artifact_id: 'html-2',
        name: 'dashboard',
        type: ArtifactArtifactType.HTML,
        uri: 'gs://reports/dashboard.html',
      },
    ];

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations artifacts={artifacts} namespace='team-a' />
      </CommonTestWrapper>,
    );

    expect(readFileSpy).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'HTML visualization' }));
    fireEvent.click(screen.getByRole('option', { name: 'report' }));

    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));
    expect(readFileSpy).toHaveBeenCalledWith({
      path: { bucket: 'reports', key: 'output.html', source: StorageService.S3 },
      namespace: 'team-a',
    });
  });

  it('automatically renders a single file artifact', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Report</h1>');
    readFileSpy.mockClear();

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              artifact_id: 'html-1',
              name: 'report',
              type: ArtifactArtifactType.HTML,
              uri: 's3://reports/output.html',
            },
          ]}
          namespace='team-a'
        />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText('Static HTML')).toBeVisible();
    expect(readFileSpy).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('combobox', { name: 'HTML visualization' })).toBeNull();
  });

  it('renders one HTML and one Markdown artifact together', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('content');
    readFileSpy.mockClear();

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              artifact_id: 'html-1',
              name: 'report',
              type: ArtifactArtifactType.HTML,
              uri: 's3://reports/output.html',
            },
            {
              artifact_id: 'markdown-1',
              name: 'summary',
              type: ArtifactArtifactType.Markdown,
              uri: 'gs://reports/output.md',
            },
          ]}
          namespace='team-a'
        />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText('Static HTML')).toBeVisible();
    expect(await screen.findByText('Static Markdown')).toBeVisible();
    expect(readFileSpy).toHaveBeenCalledTimes(2);
  });

  it('downloads one selected visualization with native storage metadata', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('# Summary');
    const artifact: V2beta1Artifact = {
      artifact_id: 'markdown-1',
      name: 'summary',
      type: ArtifactArtifactType.Markdown,
      uri: 'gs://reports/output.md',
    };

    await expect(TEST_ONLY.downloadVisualization(artifact, 'team-a')).resolves.toEqual({
      markdownContent: '# Summary',
      type: PlotType.MARKDOWN,
    });
  });

  it('rejects file artifacts without a URI', async () => {
    await expect(
      TEST_ONLY.downloadVisualization({ name: 'missing-file', type: ArtifactArtifactType.HTML }),
    ).rejects.toThrow(
      'missing-file has no URI. Verify that the component produced a valid artifact location.',
    );
  });

  it('renders legacy UI metadata viewers from a native artifact and output key', async () => {
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'load').mockResolvedValue([
      {
        data: [['restored']],
        labels: ['value'],
        type: PlotType.TABLE,
      },
    ]);
    const artifact: V2beta1Artifact = {
      artifact_id: 'legacy-metadata-1',
      name: 'legacy-output',
      type: ArtifactArtifactType.Artifact,
      uri: 's3://reports/mlpipeline-ui-metadata.json',
      metadata: { store_session_info: 'stale-session' } as any,
    };

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[artifact]}
          artifactKey='mlpipeline-ui-metadata'
          namespace='team-a'
        />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText('restored')).toBeVisible();
    expect(loadSpy).toHaveBeenCalledWith(
      { bucket: 'reports', key: 'mlpipeline-ui-metadata.json', source: StorageService.S3 },
      'team-a',
      { throwOnError: true },
    );
  });

  it('isolates a legacy UI metadata loading failure behind an actionable banner', async () => {
    vi.spyOn(OutputArtifactLoader, 'load').mockRejectedValue(new Error('metadata unavailable'));

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              artifact_id: 'legacy-metadata-1',
              name: 'mlpipeline-ui-metadata',
              uri: 'gs://reports/metadata.json',
            },
          ]}
          namespace='team-a'
        />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText(/Unable to retrieve legacy UI visualizations/)).toBeVisible();
    screen.getByText(/Verify the metadata artifact and its referenced sources/);
  });

  it('reports an unsupported legacy viewer type without crashing the visualization tab', async () => {
    vi.spyOn(OutputArtifactLoader, 'load').mockResolvedValue([
      { type: 'future-viewer' } as unknown as ViewerConfig,
    ]);

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              artifact_id: 'legacy-metadata-1',
              name: 'mlpipeline-ui-metadata',
              uri: 'gs://reports/metadata.json',
            },
          ]}
          namespace='team-a'
        />
      </CommonTestWrapper>,
    );

    expect(await screen.findByText(/contains an unsupported visualization type/)).toBeVisible();
  });
});
