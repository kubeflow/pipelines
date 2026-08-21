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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import { StorageService } from 'src/lib/WorkflowParser';
import { CommonTestWrapper } from 'src/TestWrapper';
import { PlotType, ViewerConfig } from './Viewer';
import {
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

  it('keeps valid ROC curves after an invalid artifact', () => {
    const invalid: V2beta1Artifact = {
      name: 'invalid ROC',
      type: ArtifactArtifactType.ClassificationMetric,
      metadata: {
        confidenceMetrics: [{ confidenceThreshold: 0.8, falsePositiveRate: 0.1 }],
      },
    };
    const valid: V2beta1Artifact = {
      name: 'valid ROC',
      type: ArtifactArtifactType.ClassificationMetric,
      metadata: {
        confidenceMetrics: [{ confidenceThreshold: 0.5, falsePositiveRate: 0.2, recall: 0.9 }],
      },
    };

    const result = buildRocCurves(expandClassificationMetrics([invalid, valid]));

    expect(result.configs).toHaveLength(1);
    expect(result.configs[0].data).toEqual([{ label: 0.5, x: 0.2, y: 0.9 }]);
    expect(result.error).toContain('invalid ROC');
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
    expect(buildConfusionMatrixResult(visualizations).matrices).toEqual([
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

  it('renders an explicit error when an invalid ROC curve is the only visualization', () => {
    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              name: 'broken ROC',
              type: ArtifactArtifactType.ClassificationMetric,
              metadata: {
                confidenceMetrics: [{ confidenceThreshold: 0.8, falsePositiveRate: 0.1 }],
              },
            },
          ]}
        />
      </CommonTestWrapper>,
    );

    screen.getByText('Invalid ROC curve artifact.');
    fireEvent.click(screen.getByText('Details'));
    screen.getByText(/broken ROC/);
    expect(screen.queryByText('There is no metrics artifact available in this step.')).toBeNull();
  });

  it('renders every value from one multi-key scalar metric artifact', () => {
    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              name: 'metrics',
              type: ArtifactArtifactType.Metric,
              metadata: { accuracy: 0.9, loss: 0.1 },
            },
          ]}
        />
      </CommonTestWrapper>,
    );

    screen.getByText('Scalar Metrics');
    screen.getByText('accuracy');
    screen.getByText('0.9');
    screen.getByText('loss');
    screen.getByText('0.1');
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

  it('assigns distinct keys to classification artifacts with the same URI', () => {
    const artifacts: V2beta1Artifact[] = [
      {
        name: 'first',
        type: ArtifactArtifactType.ClassificationMetric,
        uri: 's3://metrics/shared.json',
        metadata: { confidenceMetrics: [] },
      },
      {
        name: 'second',
        type: ArtifactArtifactType.ClassificationMetric,
        uri: 's3://metrics/shared.json',
        metadata: { confusionMatrix: {} },
      },
    ];

    const visualizations = expandClassificationMetrics(artifacts);

    expect(visualizations.map(({ key }) => key)).toHaveLength(2);
    expect(new Set(visualizations.map(({ key }) => key)).size).toBe(2);
    expect(visualizations.map(({ sourceArtifact }) => sourceArtifact)).toEqual(artifacts);
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
      path: {
        bucket: 'reports',
        key: 'output.html',
        keyEncoding: 'storage',
        source: StorageService.S3,
      },
      namespace: 'team-a',
      artifactUriQuery: undefined,
    });
  });

  it('selects artifacts independently when they share the same URI', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Dashboard</h1>');
    readFileSpy.mockClear();
    const sharedUri = 's3://reports/shared.html';

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              name: 'report',
              namespace: 'team-a',
              type: ArtifactArtifactType.HTML,
              uri: sharedUri,
            },
            {
              name: 'dashboard',
              namespace: 'team-b',
              type: ArtifactArtifactType.HTML,
              uri: sharedUri,
            },
          ]}
        />
      </CommonTestWrapper>,
    );

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'HTML visualization' }));
    fireEvent.click(screen.getByRole('option', { name: 'report' }));
    await waitFor(() =>
      expect(readFileSpy).toHaveBeenCalledWith(expect.objectContaining({ namespace: 'team-a' })),
    );

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'HTML visualization' }));
    fireEvent.click(screen.getByRole('option', { name: 'dashboard' }));

    await waitFor(() =>
      expect(readFileSpy).toHaveBeenCalledWith(expect.objectContaining({ namespace: 'team-b' })),
    );
    expect(readFileSpy).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('combobox', { name: 'HTML visualization' })).toHaveTextContent(
      'dashboard',
    );
  });

  it('preserves file selection when polling inserts or reorders other artifacts', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Dashboard</h1>');
    const report: V2beta1Artifact = {
      name: 'report',
      type: ArtifactArtifactType.HTML,
      uri: 's3://reports/report.html',
    };
    const dashboard: V2beta1Artifact = {
      name: 'dashboard',
      type: ArtifactArtifactType.HTML,
      uri: 's3://reports/dashboard.html',
    };
    const wrapper = ({ artifacts }: { artifacts: V2beta1Artifact[] }) => (
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations artifacts={artifacts} />
      </CommonTestWrapper>
    );
    const { rerender } = render(wrapper({ artifacts: [report, dashboard] }));

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'HTML visualization' }));
    fireEvent.click(screen.getByRole('option', { name: 'dashboard' }));
    expect(screen.getByRole('combobox', { name: 'HTML visualization' })).toHaveTextContent(
      'dashboard',
    );

    rerender(
      wrapper({
        artifacts: [
          {
            name: 'new report',
            type: ArtifactArtifactType.HTML,
            uri: 's3://reports/new.html',
          },
          dashboard,
          report,
        ],
      }),
    );

    expect(screen.getByRole('combobox', { name: 'HTML visualization' })).toHaveTextContent(
      'dashboard',
    );
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

  it('refetches a file visualization once when its source task finishes', async () => {
    const readFileSpy = vi
      .spyOn(Apis, 'readFile')
      .mockResolvedValueOnce('<h1>Running</h1>')
      .mockResolvedValueOnce('<h1>Complete</h1>');
    readFileSpy.mockClear();
    const artifact: V2beta1Artifact = {
      artifact_id: 'live-html',
      name: 'report',
      type: ArtifactArtifactType.HTML,
      uri: 's3://reports/output.html',
    };
    const view = (sourceFinished: boolean) => (
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[artifact]}
          namespace='team-a'
          sourceFinished={sourceFinished}
        />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(false));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

    rerender(view(true));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));

    rerender(view(true));
    expect(readFileSpy).toHaveBeenCalledTimes(2);
  });

  it('renders same-URI HTML and Markdown artifacts with separate cached configurations', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('content');
    readFileSpy.mockClear();

    render(
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[
            {
              name: 'report',
              type: ArtifactArtifactType.HTML,
              uri: 's3://reports/shared-output',
            },
            {
              name: 'summary',
              type: ArtifactArtifactType.Markdown,
              uri: 's3://reports/shared-output',
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
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [
        {
          data: [['restored']],
          labels: ['value'],
          type: PlotType.TABLE,
        },
      ],
      errors: [],
    });
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
      {
        bucket: 'reports',
        key: 'mlpipeline-ui-metadata.json',
        keyEncoding: 'storage',
        source: StorageService.S3,
      },
      'team-a',
      { artifactUriQuery: undefined, throwOnError: true },
    );
  });

  it('refetches legacy UI metadata once when its source task finishes', async () => {
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ data: [['updated']], labels: ['value'], type: PlotType.TABLE }],
      errors: [],
    });
    loadSpy.mockClear();
    const artifact: V2beta1Artifact = {
      artifact_id: 'live-legacy-metadata',
      name: 'mlpipeline-ui-metadata',
      uri: 's3://reports/metadata.json',
    };
    const view = (sourceFinished: boolean) => (
      <CommonTestWrapper>
        <RuntimeMetricsVisualizations
          artifacts={[artifact]}
          namespace='team-a'
          sourceFinished={sourceFinished}
        />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(false));
    await waitFor(() => expect(loadSpy).toHaveBeenCalledTimes(1));

    rerender(view(true));
    await waitFor(() => expect(loadSpy).toHaveBeenCalledTimes(2));

    rerender(view(true));
    expect(loadSpy).toHaveBeenCalledTimes(2);
  });

  it('isolates a legacy UI metadata loading failure behind an actionable banner', async () => {
    vi.spyOn(OutputArtifactLoader, 'loadResult').mockRejectedValue(
      new Error('metadata unavailable'),
    );

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
    vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ type: 'future-viewer' } as unknown as ViewerConfig],
      errors: [],
    });

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

  it('renders valid legacy viewers when a sibling viewer fails to load', async () => {
    vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ data: [['valid']], labels: ['value'], type: PlotType.TABLE }],
      errors: ['missing HTML source'],
    });

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

    expect(await screen.findByText('valid')).toBeVisible();
    expect(screen.getByText('Some legacy UI visualizations could not be loaded.')).toBeVisible();
    fireEvent.click(screen.getByText('Details'));
    expect(screen.getByText('missing HTML source')).toBeVisible();
  });

  it('retains valid legacy viewers while retrying transient sibling failures', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult');
      loadSpy.mockReset();
      loadSpy
        .mockResolvedValueOnce({
          configs: [{ data: [['valid']], labels: ['value'], type: PlotType.TABLE }],
          errors: ['temporary HTML failure'],
        })
        .mockResolvedValue({
          configs: [
            { data: [['valid']], labels: ['value'], type: PlotType.TABLE },
            { data: [['recovered']], labels: ['value'], type: PlotType.TABLE },
          ],
          errors: [],
        });

      render(
        <CommonTestWrapper>
          <RuntimeMetricsVisualizations
            artifacts={[
              {
                artifact_id: 'legacy-metadata-retry',
                name: 'mlpipeline-ui-metadata',
                uri: 'gs://reports/metadata.json',
              },
            ]}
            namespace='team-a'
          />
        </CommonTestWrapper>,
      );

      expect(await screen.findByText('valid')).toBeVisible();
      expect(screen.getByText('Some legacy UI visualizations could not be loaded.')).toBeVisible();

      await act(async () => vi.advanceTimersByTimeAsync(10_000));

      await waitFor(() => expect(loadSpy).toHaveBeenCalledTimes(2));
      expect(screen.queryByText('Some legacy UI visualizations could not be loaded.')).toBeNull();
      expect(screen.getByText('recovered')).toBeVisible();
    } finally {
      vi.useRealTimers();
    }
  });

  it('bounds retries when partial legacy data is followed by rejected refetches', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult');
      loadSpy.mockReset();
      loadSpy
        .mockResolvedValueOnce({
          configs: [{ data: [['valid']], labels: ['value'], type: PlotType.TABLE }],
          errors: ['temporary HTML failure'],
        })
        .mockRejectedValue(new Error('metadata service unavailable'));

      render(
        <CommonTestWrapper>
          <RuntimeMetricsVisualizations
            artifacts={[
              {
                artifact_id: 'legacy-metadata-bounded-retry',
                name: 'mlpipeline-ui-metadata',
                uri: 'gs://reports/metadata.json',
              },
            ]}
            namespace='team-a'
          />
        </CommonTestWrapper>,
      );

      expect(await screen.findByText('valid')).toBeVisible();

      await act(async () => vi.advanceTimersByTimeAsync(100_000));
      expect(loadSpy).toHaveBeenCalledTimes(4);

      await act(async () => vi.advanceTimersByTimeAsync(100_000));
      expect(loadSpy).toHaveBeenCalledTimes(4);
      expect(screen.getByText('valid')).toBeVisible();
    } finally {
      vi.useRealTimers();
    }
  });
});
