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
import { Apis } from 'src/lib/Apis';
import { StorageService } from 'src/lib/WorkflowParser';
import { PlotType } from './Viewer';
import { TEST_ONLY } from './RuntimeMetricsVisualizations';

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

    expect(TEST_ONLY.buildRocCurves([artifact])).toEqual({
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

    expect(TEST_ONLY.buildConfusionMatrices(artifacts)).toEqual([
      {
        artifact: artifacts[0],
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
        artifact: artifacts[1],
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

    expect(TEST_ONLY.expandClassificationMetrics([artifact])).toEqual([
      expect.objectContaining({
        artifact_id: 'sliced-1:country=US',
        name: 'evaluation · country=US',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: {
          confidenceMetrics: [{ confidenceThreshold: 0.5, falsePositiveRate: 0.1, recall: 0.9 }],
        },
      }),
      expect.objectContaining({
        artifact_id: 'sliced-1:country=CA',
        name: 'evaluation · country=CA',
        type: ArtifactArtifactType.ClassificationMetric,
        metadata: {
          confusionMatrix: {
            annotationSpecs: [{ displayName: 'yes' }, { displayName: 'no' }],
            rows: [{ row: [1, 0] }, { row: [0, 1] }],
          },
        },
      }),
    ]);
  });

  it('downloads HTML and Markdown artifacts concurrently with native storage metadata', async () => {
    let resolveHtml!: (value: string) => void;
    let resolveMarkdown!: (value: string) => void;
    const readFileSpy = vi.spyOn(Apis, 'readFile');
    readFileSpy
      .mockReturnValueOnce(new Promise((resolve) => (resolveHtml = resolve)))
      .mockReturnValueOnce(new Promise((resolve) => (resolveMarkdown = resolve)));
    const artifacts: V2beta1Artifact[] = [
      {
        artifact_id: 'html-1',
        name: 'report',
        type: ArtifactArtifactType.HTML,
        uri: 's3://reports/output.html',
        metadata: { store_session_info: 'session' } as any,
      },
      {
        artifact_id: 'markdown-1',
        name: 'summary',
        type: ArtifactArtifactType.Markdown,
        uri: 'gs://reports/output.md',
      },
    ];

    const resultPromise = TEST_ONLY.downloadVisualizations(artifacts, 'team-a');

    expect(readFileSpy).toHaveBeenCalledTimes(2);
    expect(readFileSpy).toHaveBeenNthCalledWith(1, {
      path: { bucket: 'reports', key: 'output.html', source: StorageService.S3 },
      providerInfo: 'session',
      namespace: 'team-a',
    });
    expect(readFileSpy).toHaveBeenNthCalledWith(2, {
      path: { bucket: 'reports', key: 'output.md', source: StorageService.GCS },
      providerInfo: undefined,
      namespace: 'team-a',
    });

    resolveHtml('<h1>Report</h1>');
    resolveMarkdown('# Summary');

    await expect(resultPromise).resolves.toEqual({
      html: [{ htmlContent: '<h1>Report</h1>', type: PlotType.WEB_APP }],
      markdown: [{ markdownContent: '# Summary', type: PlotType.MARKDOWN }],
    });
  });

  it('rejects file artifacts without a URI', async () => {
    await expect(
      TEST_ONLY.downloadVisualizations([{ name: 'missing-file', type: ArtifactArtifactType.HTML }]),
    ).rejects.toThrow('missing-file has no URI.');
  });
});
