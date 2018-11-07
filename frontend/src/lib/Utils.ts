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

import { Workflow } from '../../third_party/argo-ui/argo_template';
import { ApiTrigger } from '../apis/job';
import { isFunction } from 'lodash';

export const logger = {
  error: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.error(...args);
  },
  verbose: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.log(...args);
  },
};

export function formatDateString(date: Date | string | undefined): string {
  if (typeof date === 'string') {
    return new Date(date).toLocaleString();
  } else {
    return date ? date.toLocaleString() : '-';
  }
}

export async function errorToMessage(error: any): Promise<string> {
  if (error instanceof Error) {
    return error.message;
  }

  if (error && error.text && isFunction(error.text)) {
    return await error.text();
  }

  return JSON.stringify(error || '');
}

export function enabledDisplayString(trigger: ApiTrigger | undefined, enabled: boolean): string {
  if (trigger) {
    return enabled ? 'Yes' : 'No';
  }
  return '-';
}

export function getRunTime(workflow?: Workflow): string {
  if (!workflow
    || !workflow.status
    || !workflow.status.phase
    || !workflow.status.startedAt
    || !workflow.status.finishedAt) {
    return '-';
  }

  let diff = new Date(workflow.status.finishedAt).getTime()
    - new Date(workflow.status.startedAt).getTime();
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
  const hours = Math.floor((diff / HOUR) % 24).toString();
  return `${sign}${hours}:${minutes}:${seconds}`;
}

export function s(items: any[] | number): string {
  const length = Array.isArray(items) ? items.length : items;
  return length === 1 ? '' : 's';
}
