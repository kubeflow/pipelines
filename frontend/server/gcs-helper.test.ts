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

import { vi, describe, it, expect, beforeEach, Mock } from 'vitest';

vi.mock('google-auth-library');

import { GoogleAuth } from 'google-auth-library';
import { PassThrough } from 'stream';
import { downloadGCSObjectStream, getGCSClient, listGCSObjectNames } from './gcs-helper.js';

describe('gcs-helper', () => {
  const MockedGoogleAuth: Mock = GoogleAuth as any;

  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('creates a GCS auth client with the provided credentials and scope', async () => {
    const mockClient = { request: vi.fn() };
    const mockedGetClient = vi.fn().mockResolvedValue(mockClient);
    MockedGoogleAuth.mockImplementation(function () {
      return { getClient: mockedGetClient };
    });

    const credentials = {
      client_email: 'test@example.com',
      private_key: 'test-private-key',
    } as any;

    const client = await getGCSClient(credentials);

    expect(client).toBe(mockClient);
    expect(MockedGoogleAuth).toHaveBeenCalledWith({
      credentials,
      scopes: 'https://www.googleapis.com/auth/devstorage.read_write',
    });
    expect(mockedGetClient).toHaveBeenCalledTimes(1);
  });

  it('lists object names across pages and filters empty names when a client is provided', async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({
        data: {
          items: [{ name: 'hello/world 1.txt' }, { name: '' }, {}],
          nextPageToken: 'page-2',
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [{ name: 'hello/world?.txt' }],
        },
      });
    const client = { request } as any;

    const result = await listGCSObjectNames({
      bucket: 'bucket/name',
      client,
      prefix: 'hello/world prefix/',
    });

    expect(result).toEqual(['hello/world 1.txt', 'hello/world?.txt']);
    expect(request).toHaveBeenNthCalledWith(1, {
      url: 'https://storage.googleapis.com/storage/v1/b/bucket%2Fname/o?prefix=hello%2Fworld+prefix%2F',
    });
    expect(request).toHaveBeenNthCalledWith(2, {
      url: 'https://storage.googleapis.com/storage/v1/b/bucket%2Fname/o?prefix=hello%2Fworld+prefix%2F&pageToken=page-2',
    });
    expect(MockedGoogleAuth).not.toHaveBeenCalled();
  });

  it('downloads an object stream with the encoded object URL when a client is provided', async () => {
    const stream = new PassThrough();
    stream.end('hello world');
    const request = vi.fn().mockResolvedValue({ data: stream });
    const client = { request } as any;

    const result = await downloadGCSObjectStream({
      bucket: 'bucket/name',
      client,
      objectName: 'hello/world #1.txt',
    });

    expect(result).toBe(stream);
    expect(request).toHaveBeenCalledWith({
      responseType: 'stream',
      url: 'https://storage.googleapis.com/storage/v1/b/bucket%2Fname/o/hello%2Fworld%20%231.txt?alt=media',
    });
    expect(MockedGoogleAuth).not.toHaveBeenCalled();
  });
});
