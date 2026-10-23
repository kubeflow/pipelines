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

export const ARTIFACT_PREVIEW_METADATA_KEY = '_kfp_preview';
export const MAX_PREVIEW_BYTES = 64 * 1024;
export const MAX_PREVIEW_ROWS = 10;
export const MAX_PREVIEW_COLUMNS = 20;
export const MAX_PREVIEW_TEXT_LENGTH = 256;
export const MAX_MODEL_PREVIEW_ENTRIES = 100;
const MAX_MODEL_VALUE_DEPTH = 4;

export type PreviewKind = 'dataset' | 'model';
export type PreviewScalar = string | number | boolean | null;
export type ModelPreviewValue =
  | PreviewScalar
  | ModelPreviewValue[]
  | { [key: string]: ModelPreviewValue };

export interface ArtifactPreviewDescriptor {
  version: 1;
  path: 'preview.json';
}

export interface DatasetPreview {
  version: 1;
  kind: 'dataset';
  format?: string;
  columns: Array<{ name: string; dtype: string }>;
  rows: PreviewScalar[][];
  total_rows?: number;
  total_columns?: number;
  sample: {
    method: string;
    rows_truncated: boolean;
    columns_truncated: boolean;
    values_truncated: boolean;
    redacted: boolean;
    rows_omitted: boolean;
  };
}

export interface ModelPreview {
  version: 1;
  kind: 'model';
  framework?: { name: string; version?: string };
  estimator_class?: string;
  hyperparameters: Record<string, ModelPreviewValue>;
  structure: Record<string, ModelPreviewValue>;
  notes: string[];
}

export type ArtifactContentPreview = DatasetPreview | ModelPreview;

function invalid(message: string): never {
  throw new Error(`${message} Regenerate preview.json using the documented v1 preview format.`);
}

function record(value: unknown, field: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return invalid(`${field} must be an object.`);
  }
  return value as Record<string, unknown>;
}

function text(value: unknown, field: string): string {
  if (typeof value !== 'string' || [...value].length > MAX_PREVIEW_TEXT_LENGTH) {
    return invalid(`${field} must be a string of at most ${MAX_PREVIEW_TEXT_LENGTH} characters.`);
  }
  return value;
}

function optionalText(value: unknown, field: string): string | undefined {
  return value === undefined ? undefined : text(value, field);
}

function boolean(value: unknown, field: string): boolean {
  if (typeof value !== 'boolean') {
    return invalid(`${field} must be a boolean.`);
  }
  return value;
}

function array(value: unknown, field: string, limit: number): unknown[] {
  if (!Array.isArray(value) || value.length > limit) {
    return invalid(`${field} must be an array with at most ${limit} entries.`);
  }
  return value;
}

function scalar(value: unknown, field: string): PreviewScalar {
  if (value === null || typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'number') {
    if (!Number.isFinite(value) || (Number.isInteger(value) && !Number.isSafeInteger(value))) {
      return invalid(`${field} must use a string for non-finite values or unsafe integers.`);
    }
    return value;
  }
  return text(value, field);
}

function total(value: unknown, field: string, displayed: number): number | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < displayed) {
    return invalid(`${field} must be a safe nonnegative integer no smaller than the sample.`);
  }
  return value;
}

/** Reads only an explicit descriptor; malformed declarations must not look like absent previews. */
export function getArtifactPreviewDescriptor(
  metadata: Record<string, unknown> | undefined,
): ArtifactPreviewDescriptor | undefined {
  if (!metadata || !Object.prototype.hasOwnProperty.call(metadata, ARTIFACT_PREVIEW_METADATA_KEY)) {
    return undefined;
  }
  const descriptor = record(metadata[ARTIFACT_PREVIEW_METADATA_KEY], 'Preview descriptor');
  if (descriptor.version !== 1 || descriptor.path !== 'preview.json') {
    return invalid('The preview descriptor must specify version 1 and path "preview.json".');
  }
  return { version: 1, path: 'preview.json' };
}

function datasetPreview(value: Record<string, unknown>): DatasetPreview {
  const columns = array(value.columns, 'columns', MAX_PREVIEW_COLUMNS).map((column) => {
    const definition = record(column, 'Column');
    return { name: text(definition.name, 'Column name'), dtype: text(definition.dtype, 'dtype') };
  });
  const rows = array(value.rows, 'rows', MAX_PREVIEW_ROWS).map((row) => {
    const cells = array(row, 'Row', MAX_PREVIEW_COLUMNS);
    if (cells.length !== columns.length) {
      return invalid('Each row must have exactly one cell per preview column.');
    }
    return cells.map((cell) => scalar(cell, 'Cell'));
  });
  const sampling = record(value.sample, 'sample');
  const sample = {
    method: text(sampling.method, 'Sampling method'),
    rows_truncated: boolean(sampling.rows_truncated, 'rows_truncated'),
    columns_truncated: boolean(sampling.columns_truncated, 'columns_truncated'),
    values_truncated: boolean(sampling.values_truncated, 'values_truncated'),
    redacted: boolean(sampling.redacted === undefined ? false : sampling.redacted, 'redacted'),
    rows_omitted: boolean(
      sampling.rows_omitted === undefined ? false : sampling.rows_omitted,
      'rows_omitted',
    ),
  };
  const total_rows = total(value.total_rows, 'total_rows', rows.length);
  const total_columns = total(value.total_columns, 'total_columns', columns.length);
  if (sample.rows_omitted && rows.length) {
    return invalid('A schema-only preview must not contain rows.');
  }
  if (rows.length && !columns.length) {
    return invalid('A row sample must have at least one column.');
  }
  if (
    total_rows !== undefined &&
    total_rows > rows.length &&
    !sample.rows_omitted &&
    !sample.rows_truncated
  ) {
    return invalid('rows_truncated must identify an incomplete row sample.');
  }
  if (total_columns !== undefined && total_columns > columns.length && !sample.columns_truncated) {
    return invalid('columns_truncated must identify an incomplete column sample.');
  }
  return {
    version: 1,
    kind: 'dataset',
    format: optionalText(value.format, 'format'),
    columns,
    rows,
    total_rows,
    total_columns,
    sample,
  };
}

function modelValue(value: unknown, depth: number): ModelPreviewValue {
  if (value === null || typeof value !== 'object') {
    return scalar(value, 'Model value');
  }
  if (depth >= MAX_MODEL_VALUE_DEPTH) {
    return invalid(`Model values must not exceed ${MAX_MODEL_VALUE_DEPTH} nested containers.`);
  }
  if (Array.isArray(value)) {
    return array(value, 'Model value', MAX_MODEL_PREVIEW_ENTRIES).map((item) =>
      modelValue(item, depth + 1),
    );
  }
  return modelEntries(value, depth + 1);
}

function modelEntries(value: unknown, depth = 0): Record<string, ModelPreviewValue> {
  const entries = Object.entries(record(value, 'Model entries'));
  if (entries.length > MAX_MODEL_PREVIEW_ENTRIES) {
    return invalid(`Model sections must not exceed ${MAX_MODEL_PREVIEW_ENTRIES} entries.`);
  }
  return Object.fromEntries(
    entries.map(([key, entry]) => [text(key, 'Model key'), modelValue(entry, depth)]),
  );
}

function modelPreview(value: Record<string, unknown>): ModelPreview {
  const framework =
    value.framework === undefined ? undefined : record(value.framework, 'framework');
  const hyperparameters = modelEntries(
    value.hyperparameters === undefined ? {} : value.hyperparameters,
  );
  const structure = modelEntries(value.structure === undefined ? {} : value.structure);
  if (
    Object.keys(hyperparameters).length + Object.keys(structure).length >
    MAX_MODEL_PREVIEW_ENTRIES
  ) {
    return invalid(
      `Model summaries must not exceed ${MAX_MODEL_PREVIEW_ENTRIES} combined entries.`,
    );
  }
  return {
    version: 1,
    kind: 'model',
    framework: framework
      ? {
          name: text(framework.name, 'Framework name'),
          version: optionalText(framework.version, 'Framework version'),
        }
      : undefined,
    estimator_class: optionalText(value.estimator_class, 'estimator_class'),
    hyperparameters,
    structure,
    notes: array(value.notes === undefined ? [] : value.notes, 'notes', 10).map((note) =>
      text(note, 'Note'),
    ),
  };
}

/** Validates untrusted preview JSON before any values reach a viewer. */
export function parseArtifactContentPreview(
  content: string,
  kind: PreviewKind,
): ArtifactContentPreview {
  if (new TextEncoder().encode(content).byteLength > MAX_PREVIEW_BYTES) {
    return invalid(`Preview exceeds ${MAX_PREVIEW_BYTES} UTF-8 bytes.`);
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(content);
  } catch {
    return invalid('Preview is not valid JSON.');
  }
  const value = record(parsed, 'Preview');
  if (value.version !== 1 || value.kind !== kind) {
    return invalid(`Expected a version 1 ${kind} preview.`);
  }
  return kind === 'dataset' ? datasetPreview(value) : modelPreview(value);
}
