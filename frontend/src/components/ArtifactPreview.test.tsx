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
import userEvent from '@testing-library/user-event';
import { CommonTestWrapper } from 'src/TestWrapper';
import { Apis } from '../lib/Apis';
import { expectErrors, testBestPractices } from '../TestUtils';
import ArtifactPreview from './ArtifactPreview';

testBestPractices();
describe('ArtifactPreview', () => {
  afterEach(() => vi.unstubAllGlobals());
  it('keeps the exact URI available in a selectable disclosure without loading a preview', () => {
    const uri = 's3://bucket/a-long-object-prefix/output.html?region=us-west-2';
    const readFile = vi.spyOn(Apis, 'readFile');
    const { container } = render(
      <CommonTestWrapper>
        <ArtifactPreview value={uri} />
      </CommonTestWrapper>,
    );
    expect(screen.getByRole('link')).toHaveAttribute('title', uri);
    expect(container.querySelector('details')).not.toHaveAttribute('open');
    fireEvent.click(screen.getByText('Full URI'));
    expect(screen.getByText(uri, { selector: 'code' })).toBeVisible();
    expect(screen.getByText(uri, { selector: 'code' })).not.toHaveAttribute('aria-label');
    expect(screen.getByText(uri, { selector: 'code' })).not.toHaveAttribute('tabindex');
    expect(readFile).not.toHaveBeenCalled();
  });

  it('tabs directly from the URI disclosure to keyboard copying without fetching contents', async () => {
    const uri = 's3://bucket/caf%C3%A9/output?region=us-west-2';
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue(undefined);
    const readFile = vi.spyOn(Apis, 'readFile');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={uri} />
      </CommonTestWrapper>,
    );
    await user.click(screen.getByText('Full URI'));
    await user.tab();
    expect(screen.getByRole('button', { name: 'Copy URI' })).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(await screen.findByText('URI copied.')).toBeVisible();
    expect(writeText).toHaveBeenCalledWith(uri);
    expect(readFile).not.toHaveBeenCalled();
  });

  it('keeps the URI readable when clipboard access fails and resets feedback for another artifact', async () => {
    vi.stubGlobal('navigator', {
      userAgent: navigator.userAgent,
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error('Denied')) },
    });
    const { rerender } = render(
      <CommonTestWrapper>
        <ArtifactPreview value='s3://bucket/first' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByText('Full URI'));
    fireEvent.click(screen.getByRole('button', { name: 'Copy URI' }));
    expect(
      await screen.findByText('Could not copy. Select the URI above and copy it manually.'),
    ).toBeVisible();
    expect(screen.getByText('s3://bucket/first', { selector: 'code' })).toBeVisible();
    rerender(
      <CommonTestWrapper>
        <ArtifactPreview value='s3://bucket/second' />
      </CommonTestWrapper>,
    );
    expect(screen.queryByText(/Could not copy/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Full URI'));
    expect(screen.getByText('s3://bucket/second', { selector: 'code' })).toBeVisible();
  });

  it('loads bounded inline previews when requested by the containing surface', async () => {
    const readFile = vi.spyOn(Apis, 'readFile').mockResolvedValue('inline preview');
    const { rerender } = render(
      <CommonTestWrapper>
        <ArtifactPreview value={{ uri: 's3://bucket/first' }} namespace='team-a' autoLoad />
      </CommonTestWrapper>,
    );
    expect(await screen.findByText('inline preview')).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Preview file contents' })).not.toBeInTheDocument();
    expect(readFile).toHaveBeenCalledWith(
      expect.objectContaining({ namespace: 'team-a', peek: 256 }),
    );
    readFile.mockResolvedValue('second preview');
    rerender(
      <CommonTestWrapper>
        <ArtifactPreview value={{ uri: 's3://bucket/second' }} namespace='team-a' autoLoad />
      </CommonTestWrapper>,
    );
    expect(await screen.findByText('second preview')).toBeVisible();
    expect(screen.queryByText('inline preview')).not.toBeInTheDocument();
    expect(readFile).toHaveBeenCalledTimes(2);
  });

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
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
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
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
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
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));

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
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));

    expect(await screen.findByText('Empty file')).toHaveAttribute('role', 'status');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });

  it('does not display cached preview state before a remounted row requests it', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('cached preview');
    const preview = (
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/key' namespace='kubeflow' />
      </CommonTestWrapper>
    );
    const { rerender } = render(preview);
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    expect(await screen.findByText('cached preview')).toBeVisible();

    rerender(
      <CommonTestWrapper>
        <div>row removed</div>
      </CommonTestWrapper>,
    );
    rerender(preview);

    expect(screen.getByRole('button', { name: 'Preview file contents' })).toBeVisible();
    expect(screen.queryByText('cached preview')).toBeNull();
    expect(readFileSpy).toHaveBeenCalledTimes(1);
  });

  it('does not display a cached preview error before a remounted row requests it', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockRejectedValue(new Error('cached failure'));
    const preview = (
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/key' namespace='kubeflow' />
      </CommonTestWrapper>
    );
    const { rerender } = render(preview);
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    expect(await screen.findByText('Error in retrieving artifact preview.')).toBeVisible();

    rerender(
      <CommonTestWrapper>
        <div>row removed</div>
      </CommonTestWrapper>,
    );
    rerender(preview);

    expect(screen.getByRole('button', { name: 'Preview file contents' })).toBeVisible();
    expect(screen.queryByText('Error in retrieving artifact preview.')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry preview' })).toBeNull();
    expect(readFileSpy).toHaveBeenCalledTimes(1);
  });

  it('requires fresh preview consent when the artifact changes in place', async () => {
    const readFileSpy = vi
      .spyOn(Apis, 'readFile')
      .mockResolvedValueOnce('artifact A preview')
      .mockResolvedValueOnce('artifact B preview');
    const { rerender } = render(
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/artifact-a' namespace='kubeflow' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    expect(await screen.findByText('artifact A preview')).toBeVisible();

    rerender(
      <CommonTestWrapper>
        <ArtifactPreview value='minio://bucket/artifact-b' namespace='kubeflow' />
      </CommonTestWrapper>,
    );

    expect(screen.getByRole('button', { name: 'Preview file contents' })).toBeVisible();
    expect(screen.queryByText('artifact A preview')).toBeNull();
    expect(readFileSpy).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    expect(await screen.findByText('artifact B preview')).toBeVisible();
    expect(readFileSpy).toHaveBeenCalledTimes(2);
  });

  it('handles gcs artifact', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('gcs preview');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'gs://bucket/key'} />
      </CommonTestWrapper>,
    );
    await screen.findByRole('link', { name: 'gs://bucket/key' });
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    await waitFor(() => screen.getByText('gcs preview'));
  });

  it('handles minio artifact with namespace', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValueOnce('minio content');
    render(
      <CommonTestWrapper>
        <ArtifactPreview value={'minio://bucket/key'} namespace={'kubeflow'} />
      </CommonTestWrapper>,
    );
    await screen.findByRole('link', { name: 'minio://bucket/key' });
    const downloadLink = screen.getByRole('link', { name: 'minio://bucket/key' });
    expect(downloadLink).toHaveAttribute(
      'href',
      'artifacts/get?source=minio&namespace=kubeflow&bucket=bucket&key=key&keyEncoding=storage&download=true',
    );
    expect(downloadLink).toHaveAttribute('download');
    expect(screen.getAllByRole('link')).toHaveLength(1);
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
    await screen.findByRole('link', { name: 'minio://bucket/key' });
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
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
    await screen.findByRole('link', { name: 'minio://bucket/key' });
    fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
    await waitFor(() => screen.getByText(`012 345 67 ...`));
  });
});
