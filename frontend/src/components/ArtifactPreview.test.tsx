/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import { Apis } from '../lib/Apis';
import { testBestPractices } from '../TestUtils';
import ArtifactPreview from './ArtifactPreview';

testBestPractices();
describe('ArtifactPreview', () => {
  it('handles undefined artifact', () => {
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={undefined} />
      </CommonTestWrapper>,
    );
    screen.getByText('Can not retrieve storage path from artifact uri: undefined');
  });

  it('handles null artifact', () => {
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={null as any} />
      </CommonTestWrapper>,
    );
    screen.getByText('Can not retrieve storage path from artifact uri: null');
  });

  it('handles empty artifact', () => {
    expect(() => {
      render(
        <CommonTestWrapper>
          <ArtifactPreview value={'i am random path'} />
        </CommonTestWrapper>,
      );
    }).toThrow(new Error('Unsupported storage path: i am random path'));
  });

  it('handles invalid artifact: no bucket', async () => {
    jest.spyOn(Apis, 'readFile').mockRejectedValue(new Error('server error: no bucket'));

    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'minio://'} namespace={'kubeflow'} />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('Error in retrieving artifact preview.'));
  });

  it('handles gcs artifact', async () => {
    jest.spyOn(Apis, 'readFile').mockResolvedValue('gcs preview');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'gs://bucket/key'} />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('gcs://bucket/key'));
    await waitFor(() => screen.getByText('gcs preview'));
  });

  it('handles minio artifact with namespace', async () => {
    jest.spyOn(Apis, 'readFile').mockResolvedValueOnce('minio content');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'minio://bucket/key'} namespace={'kubeflow'} />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('minio://bucket/key'));
    await waitFor(() =>
      expect(screen.getByText('View All').getAttribute('href')).toEqual(
        'artifacts/get?source=minio&namespace=kubeflow&bucket=bucket&key=key',
      ),
    );
  });

  it('handles artifact that previews with maxlines', async () => {
    const data = `012\n345\n678\n910`;
    jest.spyOn(Apis, 'readFile').mockResolvedValueOnce(data);
    render(
      <CommonTestWrapper>
        <ArtifactPreview
          value={'minio://bucket/key'}
          namespace={'kubeflow'}
          maxbytes={data.length}
          maxlines={2}
        />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('minio://bucket/key'));
    await waitFor(() => screen.getByText(`012 345 ...`));
  });

  it('handles artifact that previews with maxbytes', async () => {
    const data = `012\n345\n678\n910`;
    jest.spyOn(Apis, 'readFile').mockResolvedValueOnce(data);
    render(
      <CommonTestWrapper>
        <ArtifactPreview
          value={'minio://bucket/key'}
          namespace={'kubeflow'}
          maxbytes={data.length - 5}
        />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('minio://bucket/key'));
    await waitFor(() => screen.getByText(`012 345 67 ...`));
  });
});
