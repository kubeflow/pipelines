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

import { describe, expect, it } from 'vitest';
import { DEFAULT_LOG_ARCHIVE_KEY_FORMAT, validateLogArchiveKey } from './log-archive-policy.js';

const context = { runId: 'run-uuid', namespace: 'team-a', workflowName: 'pipeline-abc' };
const pod = 'pipeline-abc-step-123';
const key = `private-artifacts/team-a/pipeline-abc/2026/11/12/${pod}/main.log`;
const validate = (candidate: string, format = DEFAULT_LOG_ARCHIVE_KEY_FORMAT) =>
  validateLogArchiveKey(candidate, pod, context, format);

describe('validateLogArchiveKey', () => {
  it('accepts standard historical log layouts without a SQL timestamp', () => {
    expect(validate(key)).toBe(true);
    expect(validate(key.replace(`/${pod}/`, '/retry-child-pod/'))).toBe(true);
    // Workflow creation can cross midnight after the run row was created.
    expect(validate(key.replace('/2026/11/12/', '/2026/11/13/'))).toBe(true);
    expect(validate(key.replace('/2026/11/12/', '/2024/02/29/'))).toBe(true);
  });

  it('accepts a custom operator-owned partitioned layout', () => {
    expect(
      validate(
        `custom_logs/team-a/pipeline-abc/${pod}/main.log`,
        'custom_logs/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
      ),
    ).toBe(true);
  });

  it.each([
    key.replace('/team-a/', '/team-b/'),
    key.replace('/pipeline-abc/', '/another-run/'),
    key.replace('/main.log', '/model.bin'),
    key.replace('/main.log', '/main.log/extra'),
    key.replace('/2026/11/12/', '/2026/02/29/'),
    key.replace('/2026/11/12/', '/2026/04/31/'),
    key.replace('/2026/11/12/', '/0000/11/12/'),
    key.replace('/2026/11/12/', '/2026/00/12/'),
    key.replace('/2026/11/12/', '/2026/11/00/'),
    key.replace('/2026/11/12/', '/2026/1/12/'),
    key.replace('/team-a/', '/team-a/../team-b/'),
    key.replace('/team-a/', '/team-a\\other/'),
    key.replace('/team-a/', '/team-a%2fother/'),
    key.replace('/team-a/', '/team-a%252fother/'),
    key.replace('/team-a/', '/%74eam-a/'),
    key.replace('/team-a/', '//'),
    key.replace('/team-a/', '/./'),
    key.replace('/team-a/', '/team-a\n/'),
    key.replace('/team-a/', '/team-a\u0000/'),
    `${key}?download=1`,
    `${key}#fragment`,
    `${key}\n`,
    `/${key}`,
  ])('rejects mismatched or ambiguous key %s', (candidate) => {
    expect(validate(candidate)).toBe(false);
  });

  it.each([
    'private-artifacts/team-a/{{workflow.name}}/{{pod.name}}',
    'private-artifacts/{{workflow.namespace}}/pipeline-abc/{{pod.name}}',
    'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/fixed-pod',
    'private-artifacts/{{workflow.namespace}}-{{workflow.name}}/{{pod.name}}',
    'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{unknown}}/{{pod.name}}',
    'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/.*/{{pod.name}}',
    'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/../{{pod.name}}',
  ])('rejects unsafe or unpartitioned format %s', (format) => {
    const candidate =
      format
        .replace('{{workflow.namespace}}', context.namespace)
        .replace('{{workflow.name}}', context.workflowName)
        .replace('{{pod.name}}', pod)
        .replace('{{unknown}}', 'other') + '/main.log';
    expect(validate(candidate, format)).toBe(false);
  });

  it('treats literal dots as literal characters rather than regex syntax', () => {
    const format = 'logs.v1/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}';
    expect(validate(`logs.v1/team-a/pipeline-abc/${pod}/main.log`, format)).toBe(true);
    expect(validate(`logsXv1/team-a/pipeline-abc/${pod}/main.log`, format)).toBe(false);
  });

  it.each(['', 'TEAM-A', '../team-a', 'team_a', 'team-a\n', 'a..b', '-team', 'a'.repeat(254)])(
    'rejects invalid Kubernetes identity %s',
    (invalid) => {
      expect(
        validateLogArchiveKey(
          key,
          pod,
          { ...context, namespace: invalid },
          DEFAULT_LOG_ARCHIVE_KEY_FORMAT,
        ),
      ).toBe(false);
      expect(
        validateLogArchiveKey(
          key,
          pod,
          { ...context, workflowName: invalid },
          DEFAULT_LOG_ARCHIVE_KEY_FORMAT,
        ),
      ).toBe(false);
      expect(validateLogArchiveKey(key, invalid, context, DEFAULT_LOG_ARCHIVE_KEY_FORMAT)).toBe(
        false,
      );
    },
  );

  it('requires an authorized run context', () => {
    expect(
      validateLogArchiveKey(key, pod, { ...context, runId: '' }, DEFAULT_LOG_ARCHIVE_KEY_FORMAT),
    ).toBe(false);
  });
});
