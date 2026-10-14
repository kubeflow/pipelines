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
import { retryScheduleHandoff } from './ScheduleHandoff';

const detail = {
  '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
  domain: 'kubeflow.org',
  reason: 'SCHEDULE_HANDOFF_IN_PROGRESS',
};
const rejection = () =>
  new ResponseError(new Response(JSON.stringify({ code: 14, details: [detail] }), { status: 503 }));

afterEach(() => vi.useRealTimers());

describe('schedule handoff retry', () => {
  it('submits once on success', async () => {
    const submit = vi.fn().mockResolvedValue({ id: 'schedule' });
    await expect(retryScheduleHandoff('create', submit)).resolves.toEqual({ id: 'schedule' });
    expect(submit).toHaveBeenCalledTimes(1);
  });

  it.each(['create', 'enable', 'disable'] as const)(
    'recovers %s with bounded backoff',
    async (operation) => {
      vi.useFakeTimers();
      const submit = vi
        .fn()
        .mockRejectedValueOnce(rejection())
        .mockRejectedValueOnce(rejection())
        .mockResolvedValue('ok');
      const result = retryScheduleHandoff(operation, submit);
      await vi.advanceTimersByTimeAsync(999);
      expect(submit).toHaveBeenCalledTimes(1);
      await vi.advanceTimersByTimeAsync(1);
      expect(submit).toHaveBeenCalledTimes(2);
      await vi.advanceTimersByTimeAsync(1999);
      expect(submit).toHaveBeenCalledTimes(2);
      await vi.advanceTimersByTimeAsync(1);
      await expect(result).resolves.toBe('ok');
      expect(submit).toHaveBeenCalledTimes(3);
    },
  );

  it.each([
    ['create', 'was not created'],
    ['disable', 'an enabled schedule remains enabled and may still run'],
    ['enable', 'retains its previous state'],
  ] as const)(
    'stops after five rejected %s attempts with actionable feedback',
    async (operation, message) => {
      vi.useFakeTimers();
      const submit = vi.fn().mockRejectedValue(rejection());
      const result = expect(retryScheduleHandoff(operation, submit)).rejects.toThrow(message);
      await vi.advanceTimersByTimeAsync(15000);
      await result;
      expect(submit).toHaveBeenCalledTimes(5);
      expect(vi.getTimerCount()).toBe(0);
    },
  );

  it.each([
    { code: 14, message: 'SCHEDULE_HANDOFF_IN_PROGRESS' },
    { code: 14, details: [{ ...detail, domain: 'another.example' }] },
    { code: 14, details: [{ ...detail, '@type': 'another.Type' }] },
    { code: 14, details: [{ ...detail, reason: 'OTHER' }] },
    { code: 9, details: [detail] },
    { code: 14, details: 'SCHEDULE_HANDOFF_IN_PROGRESS' },
    null,
  ])('does not repeat an unrecognized 503 response: %j', async (body) => {
    const response = new Response(JSON.stringify(body), { status: 503 });
    const error = new ResponseError(response);
    const submit = vi.fn().mockRejectedValue(error);
    await expect(retryScheduleHandoff('create', submit)).rejects.toBe(error);
    expect(submit).toHaveBeenCalledTimes(1);
    expect(await response.json()).toEqual(body);
  });

  it('does not repeat wrong-status, non-JSON, or ambiguous transport errors', async () => {
    for (const error of [
      new Response(JSON.stringify({ code: 14, details: [detail] }), { status: 500 }),
      new Response('upstream unavailable', { status: 503 }),
      new TypeError('Failed to fetch'),
      new Error('SCHEDULE_HANDOFF_IN_PROGRESS'),
    ]) {
      const submit = vi.fn().mockRejectedValue(error);
      await expect(retryScheduleHandoff('create', submit)).rejects.toBe(error);
      expect(submit).toHaveBeenCalledTimes(1);
    }
  });

  it('stops if a later attempt has an ambiguous failure', async () => {
    vi.useFakeTimers();
    const error = new TypeError('Failed to fetch');
    const submit = vi.fn().mockRejectedValueOnce(rejection()).mockRejectedValue(error);
    const result = expect(retryScheduleHandoff('create', submit)).rejects.toBe(error);
    await vi.advanceTimersByTimeAsync(1000);
    await result;
    expect(submit).toHaveBeenCalledTimes(2);
  });
});
