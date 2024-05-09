/*
 * Copyright 2018 The Kubeflow Authors
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

import { ConfusionMatrixConfig } from '../components/viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../components/viewers/MarkdownViewer';
import { PagedTableConfig } from '../components/viewers/PagedTable';
import { ROCCurveConfig } from '../components/viewers/ROCCurve';
import { TensorboardViewerConfig } from '../components/viewers/Tensorboard';
import { PlotType } from '../components/viewers/Viewer';
import { Apis } from '../lib/Apis';
import { OutputArtifactLoader, TEST_ONLY } from './OutputArtifactLoader';
import { StoragePath, StorageService } from './WorkflowParser';

beforeEach(async () => {
  jest.resetAllMocks();
});

describe('OutputArtifactLoader', () => {
  const storagePath: StoragePath = { bucket: 'b', key: 'k', source: StorageService.GCS };
  const consoleSpy = jest.spyOn(console, 'error');
  let fileToRead: string;
  const readFileSpy = jest.spyOn(Apis, 'readFile');
  let getSourceContent: jest.Mock;
  beforeEach(() => {
    consoleSpy.mockImplementation(() => null);
    readFileSpy.mockImplementation(() => Promise.resolve(fileToRead));
    // Mocked in tests, because we test namespace separately.
    getSourceContent = jest.fn(async (source, storage) =>
      TEST_ONLY.readSourceContent(source, storage, /* namespace */ undefined),
    );
  });

  describe('loadOutputArtifacts', () => {
    it('handles bad API call', async () => {
      readFileSpy.mockImplementation(() => {
        throw new Error('bad call');
      });
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();
      readFileSpy.mockImplementation(() => fileToRead);
    });

    it('handles an empty file', async () => {
      fileToRead = '';
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
    });

    it('handles bad json', async () => {
      fileToRead = 'bad json';
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();
    });

    it('handles an empty json', async () => {
      fileToRead = '{}';
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();
    });

    it('handles json with no "outputs" field', async () => {
      fileToRead = '{}';
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();
    });

    it('handles json with empty "outputs" array', async () => {
      fileToRead = '{"outputs": []}';
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).not.toHaveBeenCalled();
    });

    it('passes namespace to readFile api call', async () => {
      const metadata = { type: 'markdown', source: 'gs://bucket/object/key', storage: 'gcs' };
      fileToRead = JSON.stringify({ outputs: [metadata] });
      await OutputArtifactLoader.load(storagePath, 'ns1');
      expect(readFileSpy).toHaveBeenCalledTimes(2);
      expect(readFileSpy.mock.calls.map(([{ path, namespace }]) => namespace))
        .toMatchInlineSnapshot(`
        Array [
          "ns1",
          "ns1",
        ]
      `);
    });
  });

  describe('buildConfusionMatrixConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { schema: 'schema', format: 'format' };
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('requires "labels" metadata field', async () => {
      const metadata = { source: 'source', schema: 'schema' };
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "labels" is required.');
    });

    it('requires "schema" metadata field', async () => {
      const metadata = { source: 'source', labels: 'labels' };
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "schema" missing.');
    });

    it('handles empty schema', async () => {
      const metadata = { source: 'gs://source', labels: ['labels'], schema: 'schema' };
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError(
        '"schema" must be an array of {"name": string, "type": string} objects',
      );
    });

    it('handles schema with bad field format', async () => {
      const metadata = {
        labels: ['field1', 'field2'],
        schema: [
          {
            name: 'field1',
          },
          {
            type: 'field2 type',
          },
        ],
        source: 'gs://source',
      };
      fileToRead = `
      field1,field1,0
      field1,field2,0
      field2,field1,0
      field2,field2,0
      `;
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Each item in the "schema" array must contain a "name" field');
    });

    it('handles when too many labels are specified for the data', async () => {
      const metadata = {
        labels: ['labels', 'labels'],
        schema: [
          {
            name: 'field1',
          },
          {
            name: 'field2',
            type: 'field2 type',
          },
        ],
        source: 'gs://path',
      };
      fileToRead = '';
      await expect(
        OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Data dimensions 0 do not match the number of labels passed 2');
    });

    const basicMetadata = {
      labels: ['field1', 'field2'],
      schema: [
        {
          name: 'field1',
        },
        {
          name: 'field2',
          type: 'field2 type',
        },
      ],
      source: 'gs://path',
    };
    it('returns a confusion matrix config with basic metadata', async () => {
      fileToRead = `
      field1,field1,0
      field1,field2,0
      field2,field1,0
      field2,field2,0
      `;
      const result = await OutputArtifactLoader.buildConfusionMatrixConfig(
        basicMetadata as any,
        getSourceContent,
      );
      expect(result).toEqual({
        axes: ['field1', 'field2'],
        data: [
          [0, 0],
          [0, 0],
        ],
        labels: ['field1', 'field2'],
        type: PlotType.CONFUSION_MATRIX,
      } as ConfusionMatrixConfig);
    });
    it('supports inline confusion matrix data', async () => {
      fileToRead = '';

      const source = `
      label1,label1,1
      label1,label2,2
      label2,label1,3
      label2,label2,4
      `;
      const expectedResult: ConfusionMatrixConfig = {
        axes: ['field1', 'field2'],
        data: [
          // Note, the data matrix's layout does not match how we show it in UI.
          // field1 is x-axis, field2 is y-axis
          [1 /* field1=label1, field2=label1 */, 2 /* field1=label1, field2=label2 */],
          [3 /* field1=label2, field2=label1 */, 4 /* field1=label2, field2=label2 */],
        ],
        labels: ['label1', 'label2'],
        type: PlotType.CONFUSION_MATRIX,
      };

      const result = await OutputArtifactLoader.buildConfusionMatrixConfig(
        {
          ...basicMetadata,
          labels: ['label1', 'label2'],
          storage: 'inline',
          source,
        } as any,
        getSourceContent,
      );
      expect(result).toEqual(expectedResult);
    });
  });

  describe('buildPagedTableConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { header: 'header', format: 'format' };
      await expect(
        OutputArtifactLoader.buildPagedTableConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('requires "header" metadata field', async () => {
      const metadata = { source: 'source', format: 'format' };
      await expect(
        OutputArtifactLoader.buildPagedTableConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "header" is required.');
    });

    it('requires "format" metadata field', async () => {
      const metadata = { source: 'source', header: 'header' };
      await expect(
        OutputArtifactLoader.buildPagedTableConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "format" is required.');
    });

    it('requires only supports csv format', async () => {
      const metadata = { source: 'http://source', header: 'header', format: 'json' };
      await expect(
        OutputArtifactLoader.buildPagedTableConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Unsupported table format: json');
    });

    const basicMetadata = {
      format: 'csv',
      header: ['field1', 'field2'],
      source: 'gs://path',
    };

    it('returns a paged table config with basic metadata', async () => {
      fileToRead = `
      field1,field1,0
      field1,field2,0
      field2,field1,0
      field2,field2,0
      `;
      const result = await OutputArtifactLoader.buildPagedTableConfig(
        basicMetadata as any,
        getSourceContent,
      );
      expect(result).toEqual({
        data: [
          ['field1', 'field1', '0'],
          ['field1', 'field2', '0'],
          ['field2', 'field1', '0'],
          ['field2', 'field2', '0'],
        ],
        labels: ['field1', 'field2'],
        type: PlotType.TABLE,
      } as PagedTableConfig);
    });

    it('returns a paged table config with inline metadata', async () => {
      fileToRead = '';
      const source = `
      field1,field1,1
      field1,field2,2
      field2,field1,3
      field2,field2,4
      `;
      const metadata = { ...basicMetadata, storage: 'inline', source };
      const result = await OutputArtifactLoader.buildPagedTableConfig(
        metadata as any,
        getSourceContent,
      );
      const expectedResult: PagedTableConfig = {
        data: [
          ['field1', 'field1', '1'],
          ['field1', 'field2', '2'],
          ['field2', 'field1', '3'],
          ['field2', 'field2', '4'],
        ],
        labels: ['field1', 'field2'],
        type: PlotType.TABLE,
      };
      expect(result).toEqual(expectedResult);
    });
  });

  describe('buildTensorboardConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { header: 'header', format: 'format' };
      await expect(
        OutputArtifactLoader.buildTensorboardConfig(metadata as any, 'test-ns'),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('returns a tensorboard config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      expect(await OutputArtifactLoader.buildTensorboardConfig(metadata as any, 'test-ns')).toEqual(
        {
          namespace: 'test-ns',
          type: PlotType.TENSORBOARD,
          url: 'gs://path',
        } as TensorboardViewerConfig,
      );
    });
  });

  describe('buildHtmlViewerConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { header: 'header', format: 'format' };
      await expect(
        OutputArtifactLoader.buildHtmlViewerConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('returns an HTML viewer config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(
        await OutputArtifactLoader.buildHtmlViewerConfig(metadata as any, getSourceContent),
      ).toEqual({
        htmlContent: fileToRead,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig);
    });

    it('returns source as html content when storage type is inline', async () => {
      const metadata = {
        source: `<html><body>
        Hello World!
      </body></html>`,
        storage: 'inline',
      };
      expect(
        await OutputArtifactLoader.buildHtmlViewerConfig(metadata as any, getSourceContent),
      ).toEqual({
        htmlContent: metadata.source,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig);
    });
  });

  describe('buildMarkdownViewerConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { header: 'header', format: 'format' };
      await expect(
        OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('returns a markdown viewer config with basic metadata for inline markdown', async () => {
      const metadata = { source: '# some markdown here', storage: 'inline' };
      expect(
        await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any, getSourceContent),
      ).toEqual({
        markdownContent: '# some markdown here',
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });

    it('returns a markdown viewer config with basic metadata for gcs path markdown', async () => {
      const metadata = { source: 'gs://path', storage: 'gcs' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(
        await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any, getSourceContent),
      ).toEqual({
        markdownContent: fileToRead,
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });

    it('assumes remote path by default, and returns a markdown viewer config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(
        await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any, getSourceContent),
      ).toEqual({
        markdownContent: fileToRead,
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });
  });

  describe('buildRocCurveConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { header: 'header', schema: 'schema' };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "source" is required.');
    });

    it('requires "schema" metadata field', async () => {
      const metadata = { source: 'header' };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed metadata, property "schema" is required.');
    });

    it('throws for a non-array schema', async () => {
      const metadata = { source: 'gs://path', schema: { field1: 'string' } };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError(
        'Malformed schema, must be an array of {"name": string, "type": string}',
      );
    });

    it('requires "fpr" field in the schema', async () => {
      const metadata = { source: 'gs://path', schema: [{ name: 'field1', type: 'string' }] };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed schema, expected to find a column named "fpr"');
    });

    it('requires "tpr" field in the schema', async () => {
      const metadata = { source: 'gs://path', schema: [{ name: 'fpr', type: 'string' }] };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed schema, expected to find a column named "tpr"');
    });

    it('requires "threshold" field in the schema', async () => {
      const metadata = {
        schema: [
          {
            name: 'fpr',
            type: 'string',
          },
          {
            name: 'tpr',
            type: 'string',
          },
        ],
        source: 'gs://path',
      };
      await expect(
        OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).rejects.toThrowError('Malformed schema, expected to find a column named "threshold"');
    });

    const basicMetadata = {
      schema: [{ name: 'fpr' }, { name: 'tpr' }, { name: 'threshold' }],
      source: 'gs://path',
    };

    it('returns an ROC viewer config with basic metadata', async () => {
      const metadata = basicMetadata;
      fileToRead = `
        0,1,2
        3,4,5
        6,7,8
      `;
      expect(
        await OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).toEqual({
        data: [
          { label: '2', x: 0, y: 1 },
          { label: '5', x: 3, y: 4 },
          { label: '8', x: 6, y: 7 },
        ],
        type: PlotType.ROC,
      } as ROCCurveConfig);
    });

    it('returns an ROC viewer config with basic metadata', async () => {
      const source = `
        9,1,2
        3,4,5
        6,7,8
      `;
      const metadata = { ...basicMetadata, source, storage: 'inline' };
      fileToRead = '';
      const expectedResult: ROCCurveConfig = {
        data: [
          { label: '2', x: 9, y: 1 },
          { label: '5', x: 3, y: 4 },
          { label: '8', x: 6, y: 7 },
        ],
        type: PlotType.ROC,
      };
      expect(
        await OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).toEqual(expectedResult);
    });

    it('returns an ROC viewer config with fields out of order', async () => {
      const metadata = {
        schema: [{ name: 'threshold' }, { name: 'tpr' }, { name: 'fpr' }],
        source: 'gs://path',
      };
      fileToRead = `
        0,1,2
        3,4,5
        6,7,8
      `;
      expect(
        await OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).toEqual({
        data: [
          { label: '0', x: 2, y: 1 },
          { label: '3', x: 5, y: 4 },
          { label: '6', x: 8, y: 7 },
        ],
        type: PlotType.ROC,
      } as ROCCurveConfig);
    });

    it('returns an ROC viewer config with "thresholds" column instead of "threshold', async () => {
      const metadata = {
        schema: [{ name: 'fpr' }, { name: 'tpr' }, { name: 'thresholds' }],
        source: 'gs://path',
      };
      fileToRead = `
        0,1,2
        3,4,5
        6,7,8
      `;
      expect(
        await OutputArtifactLoader.buildRocCurveConfig(metadata as any, getSourceContent),
      ).toEqual({
        data: [
          { label: '2', x: 0, y: 1 },
          { label: '5', x: 3, y: 4 },
          { label: '8', x: 6, y: 7 },
        ],
        type: PlotType.ROC,
      } as ROCCurveConfig);
    });
  });
});
