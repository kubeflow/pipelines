/*
 * Copyright 2026 The Kubeflow Authors
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

import { ResponseError } from '../generated/openapi/runtime';
import { PaginationRestartRequired, throwIfPaginationRestartRequired } from './Pagination';

const detail = {
  '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
  domain: 'kubeflow.org',
  reason: 'PAGINATION_RESTART_REQUIRED',
};

describe('pagination restart errors', () => {
  it('recognizes the generated API runtime ResponseError wrapper', async () => {
    const response = new Response(JSON.stringify({ code: 9, details: [detail] }), { status: 400 });
    await expect(
      throwIfPaginationRestartRequired(new ResponseError(response)),
    ).rejects.toBeInstanceOf(PaginationRestartRequired);
    expect((await response.json()).code).toBe(9);
  });

  it('recognizes structured ErrorInfo even after a wrapped Status detail', async () => {
    const response = new Response(
      JSON.stringify({
        code: 9,
        details: [{ '@type': 'type.googleapis.com/google.rpc.Status' }, detail],
      }),
      { status: 400 },
    );
    await expect(throwIfPaginationRestartRequired(response)).rejects.toBeInstanceOf(
      PaginationRestartRequired,
    );
    expect((await response.json()).code).toBe(9);
  });

  it.each([
    { message: 'PAGINATION_RESTART_REQUIRED' },
    { details: [{ ...detail, domain: 'another.example' }] },
    { details: [{ ...detail, reason: 'PERMISSION_DENIED' }] },
    { details: [{ ...detail, '@type': 'another.Type' }] },
    { details: 'PAGINATION_RESTART_REQUIRED' },
    null,
  ])('preserves unrelated errors: %j', async (body) => {
    const response = new Response(JSON.stringify(body), { status: 400 });
    await expect(throwIfPaginationRestartRequired(response)).resolves.toBeUndefined();
    expect(await response.json()).toEqual(body);
  });

  it('does not recover from an old server generic token rejection or non-JSON response', async () => {
    const response = new Response('invalid token', { status: 400 });
    await expect(throwIfPaginationRestartRequired(response)).resolves.toBeUndefined();
    expect(await response.text()).toBe('invalid token');
    await expect(
      throwIfPaginationRestartRequired(new Error('PAGINATION_RESTART_REQUIRED')),
    ).resolves.toBeUndefined();
  });
});
