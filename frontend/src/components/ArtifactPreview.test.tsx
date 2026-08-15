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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { CommonTestWrapper } from 'src/TestWrapper';
import { Apis } from '../lib/Apis';
import { expectErrors, testBestPractices } from '../TestUtils';
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

  it('handles unsupported path artifact', () => {
    const expectError = expectErrors();
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'i am random path'} />
      </CommonTestWrapper>,
    );
    screen.getByText('Can not retrieve storage path from artifact uri: i am random path');
    expectError();
  });

  it('handles invalid artifact: no bucket', async () => {
    vi.spyOn(Apis, 'readFile').mockRejectedValue(new Error('server error: no bucket'));

    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'minio://'} namespace={'kubeflow'} />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));
    await waitFor(() => screen.getByText('Error in retrieving artifact preview.'));
  });

  it('allows a failed lazy preview to be retried', async () => {
    const readFileSpy = vi
      .spyOn(Apis, 'readFile')
      .mockRejectedValueOnce(new Error('temporary storage failure'))
      .mockResolvedValueOnce('recovered preview');

    render(
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/key' namespace='kubeflow' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));
    await screen.findByText('Error in retrieving artifact preview.');

    fireEvent.click(screen.getByRole('button', { name: 'Retry preview' }));

    expect(await screen.findByText('recovered preview')).toBeVisible();
    expect(readFileSpy).toHaveBeenCalledTimes(2);
  });

  it('shows progress while a lazy preview is loading', async () => {
    let resolvePreview!: (value: string) => void;
    vi.spyOn(Apis, 'readFile').mockReturnValue(
      new Promise((resolve) => {
        resolvePreview = resolve;
      }),
    );

    render(
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/key' namespace='kubeflow' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));

    expect(
      await screen.findByRole('progressbar', { name: 'Loading artifact preview' }),
    ).toBeVisible();

    resolvePreview('loaded preview');
    expect(await screen.findByText('loaded preview')).toBeVisible();
  });

  it('renders an explicit state for an empty artifact preview', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('');

    render(
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/key' namespace='kubeflow' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));

    expect(await screen.findByText('Artifact preview is empty.')).toBeVisible();
  });

  it('handles gcs artifact', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('gcs preview');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'gs://bucket/key'} />
      </CommonTestWrapper>,
    );
    await waitFor(() => screen.getByText('gs://bucket/key'));
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));
    await waitFor(() => screen.getByText('gcs preview'));
  });

  it('handles minio artifact with namespace', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValueOnce('minio content');
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
    vi.spyOn(Apis, 'readFile').mockResolvedValueOnce(data);
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
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));
    await waitFor(() => screen.getByText(`012 345 ...`));
  });

  it('handles artifact that previews with maxbytes', async () => {
    const data = `012\n345\n678\n910`;
    vi.spyOn(Apis, 'readFile').mockResolvedValueOnce(data);
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
    fireEvent.click(screen.getByRole('button', { name: 'Load preview' }));
    await waitFor(() => screen.getByText(`012 345 67 ...`));
  });
});
