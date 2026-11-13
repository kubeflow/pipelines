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

export interface LogArchiveContext {
  runId: string;
  namespace: string;
  workflowName: string;
}

export const DEFAULT_LOG_ARCHIVE_KEY_FORMAT =
  'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}';

function isDnsSubdomain(value: string): boolean {
  return (
    value.length > 0 &&
    value.length <= 253 &&
    !/[^a-z0-9.-]/.test(value) &&
    value.split('.').every((label) => /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/.test(label))
  );
}

function isLiteralSegment(value: string): boolean {
  return value.length > 0 && value !== '.' && value !== '..' && !/[^A-Za-z0-9_.-]/.test(value);
}

/** Validate object coordinates against an operator-owned format and authorized run context.
 * Never derive context or the format from mutable workflow status, and never decode keys here.
 */
export function validateLogArchiveKey(
  key: string,
  podName: string,
  context: LogArchiveContext,
  keyFormat: string,
): boolean {
  if (
    !context.runId ||
    !isDnsSubdomain(context.namespace) ||
    context.namespace.length > 63 ||
    context.namespace.includes('.') ||
    !isDnsSubdomain(context.workflowName) ||
    !isDnsSubdomain(podName)
  ) {
    return false;
  }
  const format = keyFormat.split('/');
  const segments = key.split('/');
  if (
    segments.length !== format.length + 1 ||
    segments[segments.length - 1] !== 'main.log' ||
    !segments.every(isLiteralSegment)
  ) {
    return false;
  }
  const identities: Record<string, string> = {
    '{{workflow.namespace}}': context.namespace,
    '{{workflow.name}}': context.workflowName,
  };
  if (![...Object.keys(identities), '{{pod.name}}'].every((token) => format.includes(token))) {
    return false;
  }
  const dates = new Map<string, number>();
  let archivedPod: string | undefined;
  for (let i = 0; i < format.length; i++) {
    const token = format[i];
    const segment = segments[i];
    if (Object.prototype.hasOwnProperty.call(identities, token)) {
      if (segment !== identities[token]) return false;
    } else if (token === '{{pod.name}}') {
      // A retry/group node may resolve to a child Pod within the authorized run.
      if (!isDnsSubdomain(segment) || (archivedPod !== undefined && archivedPod !== segment)) {
        return false;
      }
      archivedPod = segment;
    } else if (
      token === '{{workflow.creationTimestamp.Y}}' ||
      token === '{{workflow.creationTimestamp.m}}' ||
      token === '{{workflow.creationTimestamp.d}}'
    ) {
      const field = token.slice(-3, -2);
      const value = Number(segment);
      if (!(field === 'Y' ? /^\d{4}$/ : /^\d{2}$/).test(segment) || value < 1) {
        return false;
      }
      if ((field === 'm' && value > 12) || (field === 'd' && value > 31)) return false;
      if (dates.has(field) && dates.get(field) !== value) return false;
      dates.set(field, value);
    } else if (!isLiteralSegment(token) || token !== segment) {
      return false;
    }
  }
  const year = dates.get('Y');
  const month = dates.get('m');
  const day = dates.get('d');
  if (year !== undefined && month !== undefined && day !== undefined) {
    const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
    const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    if (day > days[month - 1]) return false;
  }
  return true;
}
