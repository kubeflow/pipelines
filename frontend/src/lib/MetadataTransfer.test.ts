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

import { afterEach, describe, expect, it, vi } from 'vitest';
import { exportMetadata, importMetadata, MAX_TRANSFER_BYTES } from './MetadataTransfer';

afterEach(() => vi.unstubAllGlobals());

describe('metadata transfer client', () => {
  it('uploads original archive bytes with namespace and same-origin credentials', async () => {
    const result = { counts: {}, imported: 1, skipped: 0, dry_run: true, warnings: [] };
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => result });
    vi.stubGlobal('fetch', fetchMock);
    const file = new File(['{"integer":9007199254740993}'], 'archive.json');
    expect(await importMetadata('team/a', file, 'copy & ', true)).toEqual(result);
    expect(fetchMock).toHaveBeenCalledWith(
      'apis/v2beta1/transfer/import?namespace=team%2Fa&dry_run=true&name_prefix=copy+%26+',
      expect.objectContaining({
        method: 'POST',
        body: file,
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });
  it('rejects oversized archives without a request', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const file = new File([], 'large.json');
    Object.defineProperty(file, 'size', { value: MAX_TRANSFER_BYTES + 1 });
    await expect(importMetadata('', file, '', true)).rejects.toThrow('256 MiB');
    expect(fetchMock).not.toHaveBeenCalled();
  });
  it('exports date bounds without parsing archive JSON', async () => {
    const blob = new Blob(['{"integer":9007199254740993}']);
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, blob: async () => blob });
    vi.stubGlobal('fetch', fetchMock);
    expect(await exportMetadata('team', { completed_after: 100 })).toBe(blob);
    expect(fetchMock).toHaveBeenCalledWith(
      'apis/v2beta1/transfer/export?namespace=team',
      expect.objectContaining({ body: '{"completed_after":100}', credentials: 'same-origin' }),
    );
  });
  it('surfaces backend validation errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        text: async () => '{"error":"A pipeline already exists."}',
      }),
    );
    await expect(importMetadata('team', new File([], 'archive.json'), '', true)).rejects.toThrow(
      'A pipeline already exists.',
    );
  });
});
