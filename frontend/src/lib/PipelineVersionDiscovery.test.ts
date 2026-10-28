/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {
  discoverUploadedPipelineVersion,
  VERSION_DISCOVERY_TIMEOUT_MS,
} from './PipelineVersionDiscovery';
import { V2ListPipelineVersionsResponse } from '../apisv2/pipeline';

const version = { pipeline_id: 'uploaded', pipeline_version_id: 'version' };

describe('discoverUploadedPipelineVersion', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('retries missing and empty results, then returns the scoped version', async () => {
    const read = vi
      .fn()
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce({ pipeline_versions: [] })
      .mockResolvedValueOnce({ pipeline_versions: [version] });
    const result = discoverUploadedPipelineVersion('uploaded', read, new AbortController().signal);
    await vi.advanceTimersByTimeAsync(1000);
    await expect(result).resolves.toEqual(version);
    expect(read).toHaveBeenCalledTimes(3);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('bounds a stalled request and aborts its signal', async () => {
    const read = vi.fn(
      (_signal: AbortSignal) => new Promise<V2ListPipelineVersionsResponse>(() => {}),
    );
    const result = discoverUploadedPipelineVersion('uploaded', read, new AbortController().signal);
    const rejected = expect(result).rejects.toThrow('not available yet');
    await vi.advanceTimersByTimeAsync(VERSION_DISCOVERY_TIMEOUT_MS);
    await rejected;
    expect(read).toHaveBeenCalledTimes(1);
    expect(read.mock.calls[0][0].aborted).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('surfaces API errors immediately without retrying', async () => {
    const error = new Error('Forbidden');
    const read = vi.fn().mockRejectedValue(error);
    await expect(
      discoverUploadedPipelineVersion('uploaded', read, new AbortController().signal),
    ).rejects.toBe(error);
    expect(read).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each([
    { pipeline_id: 'uploaded' },
    { pipeline_id: 'different', pipeline_version_id: 'version' },
  ])('rejects invalid nonempty records without retrying: %j', async (record) => {
    const read = vi.fn().mockResolvedValue({ pipeline_versions: [record] });
    await expect(
      discoverUploadedPipelineVersion('uploaded', read, new AbortController().signal),
    ).rejects.toThrow('invalid pipeline version');
    expect(read).toHaveBeenCalledTimes(1);
  });

  it('uses the scoped pipeline ID when a version omits the redundant ID', async () => {
    const read = vi
      .fn()
      .mockResolvedValue({ pipeline_versions: [{ pipeline_version_id: 'version' }] });
    await expect(
      discoverUploadedPipelineVersion('uploaded', read, new AbortController().signal),
    ).resolves.toEqual(version);
  });

  it('cancels while waiting and does not issue another read', async () => {
    const controller = new AbortController();
    const read = vi.fn().mockResolvedValue({ pipeline_versions: [] });
    const result = discoverUploadedPipelineVersion('uploaded', read, controller.signal);
    const rejected = expect(result).rejects.toMatchObject({ name: 'AbortError' });
    await vi.advanceTimersByTimeAsync(1);
    controller.abort();
    await rejected;
    await vi.advanceTimersByTimeAsync(VERSION_DISCOVERY_TIMEOUT_MS);
    expect(read).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('never starts a read for an already cancelled request', async () => {
    const controller = new AbortController();
    controller.abort();
    const read = vi.fn();
    await expect(
      discoverUploadedPipelineVersion('uploaded', read, controller.signal),
    ).rejects.toMatchObject({ name: 'AbortError' });
    expect(read).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });
});
