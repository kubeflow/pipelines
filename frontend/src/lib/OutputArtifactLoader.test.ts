/*
 * Copyright 2018 Google LLC
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

import { Apis } from '../lib/Apis';
import { ConfusionMatrixConfig } from '../components/viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../components/viewers/MarkdownViewer';
import { OutputArtifactLoader } from './OutputArtifactLoader';
import { PagedTableConfig } from '../components/viewers/PagedTable';
import { PlotType } from '../components/viewers/Viewer';
import { ROCCurveConfig } from '../components/viewers/ROCCurve';
import { StoragePath, StorageService } from './WorkflowParser';
import { TensorboardViewerConfig } from '../components/viewers/Tensorboard';

describe('OutputArtifactLoader', () => {
  const storagePath: StoragePath = { bucket: 'b', key: 'k', source: StorageService.GCS };
  const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => null);
  let fileToRead: string;
  jest.spyOn(Apis, 'readFile').mockImplementation(() => fileToRead);

  describe('loadOutputArtifacts', () => {
    it('handles bad API call', async () => {
      jest.spyOn(Apis, 'readFile').mockImplementation(() => {
        throw new Error('bad call');
      });
      expect(await OutputArtifactLoader.load(storagePath)).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      jest.spyOn(Apis, 'readFile').mockImplementation(() => fileToRead);
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
      expect(consoleSpy).toHaveBeenCalled();
    });
  });

  describe('buildConfusionMatrixConfig', () => {
    it('requires "source" metadata field', async () => {
      const metadata = { schema: 'schema', format: 'format' };
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('requires "labels" metadata field', () => {
      const metadata = { source: 'source', schema: 'schema' };
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "labels" is required.',
      );
    });

    it('requires "schema" metadata field', () => {
      const metadata = { source: 'source', labels: 'labels' };
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "schema" missing.',
      );
    });

    it('handles empty schema', () => {
      const metadata = { source: 'gs://source', labels: ['labels'], schema: 'schema' };
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        '"schema" must be an array of {"name": string, "type": string} objects',
      );
    });

    it('handles schema with bad field format', () => {
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
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        'Each item in the "schema" array must contain a "name" field',
      );
    });

    it('handles when too many labels are specified for the data', () => {
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
      expect(OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any)).rejects.toThrowError(
        'Data dimensions 0 do not match the number of labels passed 2',
      );
    });

    it('returns a confusion matrix config with basic metadata', async () => {
      const metadata = {
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
      fileToRead = `
      field1,field1,0
      field1,field2,0
      field2,field1,0
      field2,field2,0
      `;
      const result = await OutputArtifactLoader.buildConfusionMatrixConfig(metadata as any);
      expect(result).toEqual({
        axes: ['field1', 'field2'],
        data: [[0, 0], [0, 0]],
        labels: ['field1', 'field2'],
        type: PlotType.CONFUSION_MATRIX,
      } as ConfusionMatrixConfig);
    });
  });

  describe('buildPagedTableConfig', () => {
    it('requires "source" metadata field', () => {
      const metadata = { header: 'header', format: 'format' };
      expect(OutputArtifactLoader.buildPagedTableConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('requires "header" metadata field', () => {
      const metadata = { source: 'source', format: 'format' };
      expect(OutputArtifactLoader.buildPagedTableConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "header" is required.',
      );
    });

    it('requires "format" metadata field', () => {
      const metadata = { source: 'source', header: 'header' };
      expect(OutputArtifactLoader.buildPagedTableConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "format" is required.',
      );
    });

    it('requires only supports csv format', () => {
      const metadata = { source: 'source', header: 'header', format: 'json' };
      expect(OutputArtifactLoader.buildPagedTableConfig(metadata as any)).rejects.toThrowError(
        'Unsupported table format: json',
      );
    });

    it('returns a paged table config with basic metadata', async () => {
      const metadata = {
        format: 'csv',
        header: ['field1', 'field2'],
        source: 'gs://path',
      };
      fileToRead = `
      field1,field1,0
      field1,field2,0
      field2,field1,0
      field2,field2,0
      `;
      const result = await OutputArtifactLoader.buildPagedTableConfig(metadata as any);
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
  });

  describe('buildTensorboardConfig', () => {
    it('requires "source" metadata field', () => {
      const metadata = { header: 'header', format: 'format' };
      expect(OutputArtifactLoader.buildTensorboardConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('returns a tensorboard config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      expect(await OutputArtifactLoader.buildTensorboardConfig(metadata as any)).toEqual({
        type: PlotType.TENSORBOARD,
        url: 'gs://path',
      } as TensorboardViewerConfig);
    });
  });

  describe('buildHtmlViewerConfig', () => {
    it('requires "source" metadata field', () => {
      const metadata = { header: 'header', format: 'format' };
      expect(OutputArtifactLoader.buildHtmlViewerConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('returns an HTML viewer config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(await OutputArtifactLoader.buildHtmlViewerConfig(metadata as any)).toEqual({
        htmlContent: fileToRead,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig);
    });
  });

  describe('buildMarkdownViewerConfig', () => {
    it('requires "source" metadata field', () => {
      const metadata = { header: 'header', format: 'format' };
      expect(OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('returns a markdown viewer config with basic metadata for inline markdown', async () => {
      const metadata = { source: '# some markdown here', storage: 'inline' };
      expect(await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any)).toEqual({
        markdownContent: '# some markdown here',
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });

    it('returns a markdown viewer config with basic metadata for gcs path markdown', async () => {
      const metadata = { source: 'gs://path', storage: 'gcs' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any)).toEqual({
        markdownContent: fileToRead,
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });

    it('assumes remote path by default, and returns a markdown viewer config with basic metadata', async () => {
      const metadata = { source: 'gs://path' };
      fileToRead = `<html><body>
        Hello World!
      </body></html>`;
      expect(await OutputArtifactLoader.buildMarkdownViewerConfig(metadata as any)).toEqual({
        markdownContent: fileToRead,
        type: PlotType.MARKDOWN,
      } as MarkdownViewerConfig);
    });
  });

  describe('buildRocCurveConfig', () => {
    it('requires "source" metadata field', () => {
      const metadata = { header: 'header', schema: 'schema' };
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "source" is required.',
      );
    });

    it('requires "schema" metadata field', () => {
      const metadata = { source: 'header' };
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed metadata, property "schema" is required.',
      );
    });

    it('throws for a non-array schema', () => {
      const metadata = { source: 'gs://path', schema: { field1: 'string' } };
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed schema, must be an array of {"name": string, "type": string}',
      );
    });

    it('requires "fpr" field in the schema', () => {
      const metadata = { source: 'gs://path', schema: [{ name: 'field1', type: 'string' }] };
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed schema, expected to find a column named "fpr"',
      );
    });

    it('requires "tpr" field in the schema', () => {
      const metadata = { source: 'gs://path', schema: [{ name: 'fpr', type: 'string' }] };
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed schema, expected to find a column named "tpr"',
      );
    });

    it('requires "threshold" field in the schema', () => {
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
      expect(OutputArtifactLoader.buildRocCurveConfig(metadata as any)).rejects.toThrowError(
        'Malformed schema, expected to find a column named "threshold"',
      );
    });

    it('returns an ROC viewer config with basic metadata', async () => {
      const metadata = {
        schema: [{ name: 'fpr' }, { name: 'tpr' }, { name: 'threshold' }],
        source: 'gs://path',
      };
      fileToRead = `
        0,1,2
        3,4,5
        6,7,8
      `;
      expect(await OutputArtifactLoader.buildRocCurveConfig(metadata as any)).toEqual({
        data: [{ label: '2', x: 0, y: 1 }, { label: '5', x: 3, y: 4 }, { label: '8', x: 6, y: 7 }],
        type: PlotType.ROC,
      } as ROCCurveConfig);
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
      expect(await OutputArtifactLoader.buildRocCurveConfig(metadata as any)).toEqual({
        data: [{ label: '0', x: 2, y: 1 }, { label: '3', x: 5, y: 4 }, { label: '6', x: 8, y: 7 }],
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
      expect(await OutputArtifactLoader.buildRocCurveConfig(metadata as any)).toEqual({
        data: [{ label: '2', x: 0, y: 1 }, { label: '5', x: 3, y: 4 }, { label: '8', x: 6, y: 7 }],
        type: PlotType.ROC,
      } as ROCCurveConfig);
    });
  });
});
