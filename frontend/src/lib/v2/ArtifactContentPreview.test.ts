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

import {
  getArtifactPreviewDescriptor,
  MAX_PREVIEW_BYTES,
  parseArtifactContentPreview,
} from './ArtifactContentPreview';

const dataset = {
  version: 1,
  kind: 'dataset',
  format: 'parquet',
  columns: [{ name: 'value', dtype: 'string' }],
  rows: [['first'], ['second']],
  total_rows: 100,
  total_columns: 1,
  sample: {
    method: 'head',
    rows_truncated: true,
    columns_truncated: false,
    values_truncated: false,
  },
};
const parse = (value: unknown) => parseArtifactContentPreview(JSON.stringify(value), 'dataset');

describe('artifact preview contract', () => {
  it('distinguishes absent and invalid descriptors', () => {
    expect(getArtifactPreviewDescriptor(undefined)).toBeUndefined();
    expect(getArtifactPreviewDescriptor({ framework: 'xgboost' })).toBeUndefined();
    expect(
      getArtifactPreviewDescriptor({ _kfp_preview: { version: 1, path: '_kfp_preview.json' } }),
    ).toEqual({ version: 1, path: '_kfp_preview.json' });
    expect(
      getArtifactPreviewDescriptor({ _kfp_preview: { version: 2, storage: 'sibling', extra: 1 } }),
    ).toEqual({ version: 2, storage: 'sibling' });
    for (const descriptor of [
      null,
      true,
      {},
      { version: 2, path: '_kfp_preview.json' },
      { version: 2, storage: 'object' },
      { version: 2, storage: 'inline', content: '{}' },
      { version: 1, path: '../_kfp_preview.json' },
    ]) {
      expect(() => getArtifactPreviewDescriptor({ _kfp_preview: descriptor })).toThrow();
    }
  });

  it('preserves sample order, duplicate column names and JSON scalars', () => {
    const values = [
      null,
      '',
      false,
      0,
      1.5,
      '9007199254740993',
      'NaN',
      '<script>alert(1)</script>',
    ];
    const result = parse({
      ...dataset,
      columns: values.map(() => ({ name: 'value', dtype: 'mixed' })),
      total_columns: values.length,
      rows: [values],
    });
    expect(result.kind === 'dataset' && result.rows).toEqual([values]);
  });

  it('accepts exactly ten rows and twenty columns', () => {
    const value = {
      ...dataset,
      columns: Array.from({ length: 20 }, (_, index) => ({ name: String(index), dtype: 'int64' })),
      total_columns: 20,
      rows: Array.from({ length: 10 }, () => Array(20).fill(0)),
    };
    expect(parse(value).kind).toBe('dataset');
    expect(() => parse({ ...value, rows: [...value.rows, value.rows[0]] })).toThrow('10');
    expect(() => parse({ ...value, columns: [...value.columns, value.columns[0]] })).toThrow('20');
  });

  it('distinguishes unknown totals, empty datasets and intentionally omitted rows', () => {
    expect(parse({ ...dataset, total_rows: null, total_columns: undefined })).toMatchObject({
      total_rows: undefined,
      total_columns: undefined,
    });
    expect(parse({ ...dataset, rows: [], total_rows: 0 })).toMatchObject({
      rows: [],
      total_rows: 0,
    });
    expect(
      parse({ ...dataset, rows: [], sample: { ...dataset.sample, rows_omitted: true } }),
    ).toMatchObject({ sample: { rows_omitted: true } });
    expect(() => parse({ ...dataset, sample: { ...dataset.sample, rows_omitted: true } })).toThrow(
      'schema-only',
    );
  });

  it.each([
    { rows: [['one', 'two']] },
    { total_rows: -1 },
    { total_rows: 1 },
    { total_columns: 0 },
    { total_columns: 5 },
    { rows: [[9007199254740992]] },
    { rows: [[{}]] },
    { rows: [['a'.repeat(257)]] },
    { sample: { ...dataset.sample, rows_truncated: false } },
    { sample: { ...dataset.sample, redacted: 'yes' } },
    { version: 2 },
    { kind: 'model' },
  ])('rejects invalid dataset fields: %j', (fields) => {
    expect(() => parse({ ...dataset, ...fields })).toThrow();
  });

  it('counts UTF-8 bytes and Unicode characters, not UTF-16 units', () => {
    expect(parse({ ...dataset, rows: [['\u{1f600}'.repeat(256)]] }).kind).toBe('dataset');
    const content = JSON.stringify(dataset);
    expect(parseArtifactContentPreview(content.padEnd(MAX_PREVIEW_BYTES), 'dataset').kind).toBe(
      'dataset',
    );
    expect(() =>
      parseArtifactContentPreview(content.padEnd(MAX_PREVIEW_BYTES + 1), 'dataset'),
    ).toThrow('UTF-8');
    expect(() => parse({ ...dataset, unused: '\u{1f600}'.repeat(MAX_PREVIEW_BYTES / 4) })).toThrow(
      'UTF-8',
    );
  });

  it('rejects malformed JSON and unsupported root shapes', () => {
    for (const content of [
      '{',
      'null',
      '[]',
      'true',
      '{"version":1,"kind":"dataset","columns":[]}',
    ]) {
      expect(() => parseArtifactContentPreview(content, 'dataset')).toThrow();
    }
  });

  it('keeps recorded model parameters separate from structural facts', () => {
    const model = {
      version: 1,
      kind: 'model',
      framework: { name: 'xgboost', version: '3.1.0' },
      estimator_class: 'XGBClassifier',
      hyperparameters: { max_depth: 3, missing: 'NaN', class_weights: [1, 2] },
      structure: { boosting_rounds: 5, features: 4 },
      notes: ['Training configuration recorded by the producer.'],
    };
    expect(parseArtifactContentPreview(JSON.stringify(model), 'model')).toEqual(model);
    expect(parseArtifactContentPreview('{"version":1,"kind":"model"}', 'model')).toMatchObject({
      hyperparameters: {},
      structure: {},
      notes: [],
    });
  });

  it('bounds model summaries and nested values', () => {
    const model = {
      version: 1,
      kind: 'model',
      hyperparameters: Object.fromEntries(Array.from({ length: 100 }, (_, i) => [String(i), i])),
    };
    expect(parseArtifactContentPreview(JSON.stringify(model), 'model').kind).toBe('model');
    expect(() =>
      parseArtifactContentPreview(JSON.stringify({ ...model, structure: { trees: 1 } }), 'model'),
    ).toThrow('combined entries');
    expect(() =>
      parseArtifactContentPreview(
        JSON.stringify({ ...model, hyperparameters: { x: [[[[[0]]]]] } }),
        'model',
      ),
    ).toThrow('nested');
    expect(() =>
      parseArtifactContentPreview(
        JSON.stringify({ ...model, structure: { x: 'x'.repeat(257) } }),
        'model',
      ),
    ).toThrow('characters');
  });
});
