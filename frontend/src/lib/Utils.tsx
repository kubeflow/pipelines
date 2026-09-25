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

import { isFunction } from 'lodash';
import { V2beta1RecurringRunStatus, V2beta1Trigger } from 'src/apisv2beta1/recurringrun';
import { Column, Row } from 'src/components/CustomTable';
import { ListRequest } from './Apis';
import { hasFinishedV2 } from './StatusUtils';
import { V2beta1Run } from 'src/apisv2beta1/run';

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

/** Title cases a string by capitalizing the first letter of each word. */
export function titleCase(str: string): string {
  return str
    .split(/[\s_-]/)
    .map((w) => `${w.charAt(0).toUpperCase()}${w.slice(1)}`)
    .join(' ');
}

export async function errorToMessage(error: any): Promise<string> {
  if (error instanceof Error) {
    return error.message;
  }

  if (error && error.text && isFunction(error.text)) {
    return await error.text();
  }

  return JSON.stringify(error) || '';
}

export function enabledDisplayStringV2(
  trigger: V2beta1Trigger | undefined,
  status: V2beta1RecurringRunStatus,
): string {
  if (trigger) {
    switch (status) {
      case V2beta1RecurringRunStatus.ENABLED:
        return 'Yes';
      case V2beta1RecurringRunStatus.DISABLED:
        return 'No';
      case V2beta1RecurringRunStatus.STATUS_UNSPECIFIED:
        return 'Unknown';
      default:
        return '-';
    }
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

export function getRunDurationV2(run?: V2beta1Run): string {
  return !run || !run.created_at || !run.finished_at || !hasFinishedV2(run.state)
    ? '-'
    : getDuration(new Date(run.created_at), new Date(run.finished_at));
}

export function getRunDurationFromRunV2(run?: V2beta1Run): string {
  return run && run.created_at && run.finished_at
    ? getDuration(new Date(run.created_at), new Date(run.finished_at))
    : '-';
}

export function s(items: any[] | number): string {
  const length = Array.isArray(items) ? items.length : items;
  return length === 1 ? '' : 's';
}

export interface ServiceError {
  message: string;
  code?: number | string;
}

export function isServiceError(error: unknown): error is ServiceError {
  if (!error || typeof error !== 'object') {
    return false;
  }
  if (!('message' in error) || typeof (error as ServiceError).message !== 'string') {
    return false;
  }
  if (!('code' in error)) {
    return true;
  }
  const code = (error as ServiceError).code;
  return typeof code === 'number' || typeof code === 'string';
}

export function serviceErrorToString(error: ServiceError): string {
  return `Error: ${error.message}.${error.code ? ` Code: ${error.code}` : ''}`;
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

    const sortIndex = columns.findIndex((c) => cleanedSortBy === c.sortKey);

    // Convert null to string to avoid null comparison behavior
    const compare = (r1.otherFields[sortIndex] || '') < (r2.otherFields[sortIndex] || '');
    if (request.orderAscending) {
      return compare ? -1 : 1;
    } else {
      return compare ? 1 : -1;
    }
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

/**
 * Returns the given URL only when it uses a browsable, safe scheme (http/https),
 * otherwise undefined. Used to gate user-supplied values (for example a pipeline
 * version's code_source_url) before placing them in an anchor href, so that
 * untrusted schemes such as `javascript:` are never rendered as clickable links.
 *
 * @param url Candidate URL, typically user-supplied.
 * @returns The URL when it has an HTTP(S) scheme, otherwise undefined.
 */
export function sanitizeExternalHref(url?: string): string | undefined {
  if (!url) {
    return undefined;
  }
  try {
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol === 'http:' || parsedUrl.protocol === 'https:') {
      return url;
    }
  } catch {
    // Malformed and relative URLs are not safe external links.
  }
  return undefined;
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

declare global {
  interface Window {
    // Nonstandard, Safari-only; used below purely for browser detection.
    safari?: { pushNotification?: object };
  }
}

export function isSafari(): boolean {
  // Since react-ace Editor doesn't support in Safari when height or width is a percentage.
  // Fix the Yaml file cannot display issue via defining “width/height” does not not take percentage if it's Safari browser.
  // The code of detecting wether isSafari is from: https://stackoverflow.com/questions/9847580/how-to-detect-safari-chrome-ie-firefox-and-opera-browser/9851769#9851769
  const isSafari =
    /constructor/i.test(window.HTMLElement.toString()) ||
    (function (p: unknown) {
      return String(p) === '[object SafariRemoteNotification]';
    })(!window.safari || window.safari.pushNotification);
  return isSafari;
}

// For any String value Enum, use this approach to get the string of Enum Key.
export function getStringEnumKey(e: { [s: string]: any }, value: any): string {
  return Object.keys(e)[Object.values(e).indexOf(value)];
}

export function generateRandomString(length: number): string {
  let d = 0;
  function randomChar(): string {
    const r = Math.trunc((d + Math.random() * 16) % 16);
    d = Math.floor(d / 16);
    return r.toString(16);
  }
  let str = '';
  for (let i = 0; i < length; ++i) {
    str += randomChar();
  }
  return str;
}
