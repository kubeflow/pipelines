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

import * as React from 'react';
import { ApiRun } from '../apis/run';
import { ApiTrigger } from '../apis/job';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { isFunction } from 'lodash';
import { hasFinished, NodePhase } from './StatusUtils';
import { ListRequest } from './Apis';
import { Row, Column, ExpandState } from '../components/CustomTable';
import { padding } from '../Css';
import { classes } from 'typestyle';
import { CustomTableRow, css } from '../components/CustomTableRow';

export const logger = {
  error: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.error(...args);
  },
  warn: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.warn(...args);
  },
  verbose: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.log(...args);
  },
};

export function extendError(err: any, extraMessage?: string): any {
  if (err.message && typeof err.message === 'string') {
    err.message = extraMessage + ': ' + err.message;
  }
  return err;
}

export function rethrow(err: any, extraMessage?: string): never {
  throw extendError(err, extraMessage);
}

export function formatDateString(date: Date | string | undefined): string {
  if (typeof date === 'string') {
    return new Date(date).toLocaleString();
  } else {
    return date ? date.toLocaleString() : '-';
  }
}

// TODO: add tests
export async function errorToMessage(error: any): Promise<string> {
  if (error instanceof Error) {
    return error.message;
  }

  if (error && error.text && isFunction(error.text)) {
    return await error.text();
  }

  return JSON.stringify(error) || '';
}

export function enabledDisplayString(trigger: ApiTrigger | undefined, enabled: boolean): string {
  if (trigger) {
    return enabled ? 'Yes' : 'No';
  }
  return '-';
}

function getDuration(start: Date, end: Date): string {
  let diff = end.getTime() - start.getTime();
  const sign = diff < 0 ? '-' : '';
  if (diff < 0) {
    diff *= -1;
  }
  const SECOND = 1000;
  const MINUTE = 60 * SECOND;
  const HOUR = 60 * MINUTE;
  const seconds = ('0' + Math.floor((diff / SECOND) % 60).toString()).slice(-2);
  const minutes = ('0' + Math.floor((diff / MINUTE) % 60).toString()).slice(-2);
  // Hours are the largest denomination, so we don't pad them
  const hours = Math.floor(diff / HOUR).toString();
  return `${sign}${hours}:${minutes}:${seconds}`;
}

export function getRunDuration(run?: ApiRun): string {
  if (!run || !run.created_at || !run.finished_at || !hasFinished(run.status as NodePhase)) {
    return '-';
  }

  // A bug in swagger-codegen causes the API to indicate that created_at and finished_at are Dates,
  // as they should be, when in reality they are transferred as strings.
  // See: https://github.com/swagger-api/swagger-codegen/issues/2776
  return getDuration(new Date(run.created_at), new Date(run.finished_at));
}

export function getRunDurationFromWorkflow(workflow?: Workflow): string {
  if (!workflow || !workflow.status || !workflow.status.startedAt || !workflow.status.finishedAt) {
    return '-';
  }

  return getDuration(new Date(workflow.status.startedAt), new Date(workflow.status.finishedAt));
}

export function s(items: any[] | number): string {
  const length = Array.isArray(items) ? items.length : items;
  return length === 1 ? '' : 's';
}

interface ServiceError {
  message: string;
  code?: number;
}

export function serviceErrorToString(error: ServiceError): string {
  return `Error: ${error.message}.${error.code ? ` Code: ${error.code}` : ''}`;
}

/**
 * Returns true if no filter is specified, or if the filter string matches any of the row's columns,
 * case insensitively.
 * @param request
 */
export function rowFilterFn(request: ListRequest): (r: Row) => boolean {
  // TODO: We are currently searching across all properties of all artifacts. We should figure
  // what the most useful fields are and limit filtering to those
  return r => {
    if (!request.filter) {
      return true;
    }

    const decodedFilter = decodeURIComponent(request.filter);
    try {
      const filter = JSON.parse(decodedFilter);
      if (!filter.predicates || filter.predicates.length === 0) {
        return true;
      }
      // TODO: Extend this to look at more than a single predicate
      const filterString =
        '' +
        (filter.predicates[0].int_value ||
          filter.predicates[0].long_value ||
          filter.predicates[0].string_value);
      return (
        r.otherFields
          .join('')
          .toLowerCase()
          .indexOf(filterString.toLowerCase()) > -1
      );
    } catch (err) {
      logger.error('Error parsing request filter!', err);
      return true;
    }
  };
}

export function rowCompareFn(
  request: ListRequest,
  columns: Column[],
): (r1: Row, r2: Row) => number {
  return (r1, r2) => {
    if (!request.sortBy) {
      return -1;
    }

    const descSuffix = ' desc';
    const cleanedSortBy = request.sortBy.endsWith(descSuffix)
      ? request.sortBy.substring(0, request.sortBy.length - descSuffix.length)
      : request.sortBy;

    const sortIndex = columns.findIndex(c => cleanedSortBy === c.sortKey);

    // Convert null to string to avoid null comparison behavior
    const compare = (r1.otherFields[sortIndex] || '') < (r2.otherFields[sortIndex] || '');
    if (request.orderAscending) {
      return compare ? -1 : 1;
    } else {
      return compare ? 1 : -1;
    }
  };
}

export interface CollapsedAndExpandedRows {
  // collapsedRows are the first row of each group, what the user sees before expanding a group.
  collapsedRows: Row[];
  // expandedRows is a map of grouping keys to a list of rows grouped by that key. This is what a
  // user sees when they expand one or more rows.
  expandedRows: Map<number, Row[]>;
}

/**
 * Groups the incoming rows by name and type pushing all but the first row
 * of each group to the expandedRows Map.
 * @param rows
 */
export function groupRows(rows: Row[]): CollapsedAndExpandedRows {
  const flattenedRows = rows.reduce((map, r) => {
    const stringKey = r.otherFields[0];
    const rowsForKey = map.get(stringKey);
    if (rowsForKey) {
      rowsForKey.push(r);
    } else {
      map.set(stringKey, [r]);
    }
    return map;
  }, new Map<string, Row[]>());

  const collapsedAndExpandedRows: CollapsedAndExpandedRows = {
    collapsedRows: [],
    expandedRows: new Map<number, Row[]>(),
  };
  // collapsedRows are the first row of each group, what the user sees before expanding a group.
  Array.from(flattenedRows.entries()) // entries() returns in insertion order
    .forEach((entry, index) => {
      // entry[0] is a grouping key, entry[1] is a list of rows
      const rowsInGroup = entry[1];

      // If there is only one row in the group, don't allow expansion.
      // Only the first row is displayed when collapsed
      if (rowsInGroup.length === 1) {
        rowsInGroup[0].expandState = ExpandState.NONE;
      }

      // Add the first row in this group to be displayed as collapsed row
      collapsedAndExpandedRows.collapsedRows.push(rowsInGroup[0]);

      // Remove the grouping column text for all but the first row in the group because it will be
      // redundant within an expanded group.
      const hiddenRows = rowsInGroup.slice(1);
      hiddenRows.forEach(row => (row.otherFields[0] = ''));

      // Add this group of rows sharing a pipeline to the list of grouped rows
      collapsedAndExpandedRows.expandedRows.set(index, hiddenRows);
    });

  return collapsedAndExpandedRows;
}

/**
 * Returns a fragment representing the expanded content for the given
 * row.
 * @param index
 */
export function getExpandedRow(
  expandedRows: Map<number, Row[]>,
  columns: Column[],
): (index: number) => React.ReactNode {
  return (index: number) => {
    const rows = expandedRows.get(index) || [];

    return (
      <div className={padding(65, 'l')}>
        {rows.map((r, rindex) => (
          <div className={classes('tableRow', css.row)} key={rindex}>
            <CustomTableRow row={r} columns={columns} />
          </div>
        ))}
      </div>
    );
  };
}

const GCS_CONSOLE_BASE = 'https://console.cloud.google.com/storage/browser/';
const GCS_URI_PREFIX = 'gs://';

/**
 * Generates a cloud console uri from gs:// uri
 *
 * @param gcsUri Gcs uri that starts with gs://, like gs://bucket/path/file
 * @returns A link user can open to visit cloud console page. Returns undefined when gcsUri is not valid.
 */
export function generateGcsConsoleUri(gcsUri: string): string | undefined {
  if (!gcsUri.startsWith(GCS_URI_PREFIX)) {
    return undefined;
  }

  return GCS_CONSOLE_BASE + gcsUri.substring(GCS_URI_PREFIX.length);
}

const MINIO_URI_PREFIX = 'minio://';

/**
 * Generates the path component of the url to retrieve an artifact.
 *
 * @param source source of the artifact. Can be "minio", "s3", "http", "https", or "gcs".
 * @param bucket bucket where the artifact is stored, value is assumed to be uri encoded.
 * @param key path to the artifact, value is assumed to be uri encoded.
 * @param peek number of characters or bytes to return. If not provided, the entire content of the artifact will be returned.
 */
export function generateArtifactUrl(
  source: string,
  bucket: string,
  key: string,
  peek?: number,
): string {
  return `artifacts/get${buildQuery({ source, bucket, key, peek })}`;
}

/**
 * Generates an HTTPS API URL from minio:// uri
 *
 * @param minioUri Minio uri that starts with minio://, like minio://ml-pipeline/path/file
 * @returns A URL that leads to the artifact data. Returns undefined when minioUri is not valid.
 */
export function generateMinioArtifactUrl(minioUri: string, peek?: number): string | undefined {
  if (!minioUri.startsWith(MINIO_URI_PREFIX)) {
    return undefined;
  }

  // eslint-disable-next-line no-useless-escape
  const matches = minioUri.match(/^minio:\/\/([^\/]+)\/(.+)$/);
  if (matches == null) {
    return undefined;
  }
  return generateArtifactUrl('minio', matches[1], matches[2], peek);
}

export function buildQuery(queriesMap: { [key: string]: string | number | undefined }): string {
  const queryContent = Object.entries(queriesMap)
    .filter((entry): entry is [string, string | number] => entry[1] != null)
    .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
    .join('&');
  if (!queryContent) {
    return '';
  }
  return `?${queryContent}`;
}
