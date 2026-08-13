/*
 * Copyright 2021 The Kubeflow Authors
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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { ArtifactArtifactType, V2beta1Artifact } from 'src/apisv2beta1/artifact';
import { RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { ArtifactList } from 'src/pages/ArtifactList';
import { PageProps } from 'src/pages/Page';
import TestUtils, { testBestPractices } from 'src/TestUtils';

testBestPractices();

describe('ArtifactList', () => {
  const updateBannerSpy = vi.fn();
  const historyPushSpy = vi.fn();

  function generateArtifacts(count: number): V2beta1Artifact[] {
    return Array.from({ length: count }, (_, index) => ({
      artifact_id: `artifact-${index + 1}`,
      name: `test artifact ${index + 1}`,
      type: ArtifactArtifactType.Dataset,
      uri: `s3://pipeline-root/artifact-${index + 1}`,
      namespace: 'kubeflow',
      created_at: new Date(`2026-08-${String(index + 1).padStart(2, '0')}T12:00:00Z`),
    }));
  }

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ArtifactList,
      { pathname: RoutePage.ARTIFACTS } as any,
      '' as any,
      historyPushSpy,
      updateBannerSpy,
      vi.fn(),
      vi.fn(),
      vi.fn(),
    );
  }

  function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>((resolvePromise) => {
      resolve = resolvePromise;
    });
    return { promise, resolve };
  }

  beforeEach(() => {
    vi.spyOn(Apis.artifactServiceApiV2, 'artifacts');
    vi.mocked(Apis.artifactServiceApiV2.artifacts)
      .mockReset()
      .mockResolvedValue({
        artifacts: generateArtifacts(5),
      });
  });

  it('renders native artifacts and links to their details', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: generateArtifacts(1),
    });
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );

    const artifactLink = await screen.findByRole('link', { name: 'test artifact 1' });
    expect(artifactLink).toHaveAttribute('href', '/artifacts/artifact-1');
    screen.getByText('system.Dataset');
    screen.getByText('kubeflow');
  });

  it('includes the artifact namespace and separates provider query from the URI path', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [
        {
          ...generateArtifacts(1)[0],
          namespace: 'team-a',
          uri: 's3://reports/output.csv?endpoint=https%3A%2F%2Fceph.example%3A9443',
        },
      ],
    });
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );

    const uriLink = await screen.findByRole('link', {
      name: 's3://reports/output.csv?endpoint=https%3A%2F%2Fceph.example%3A9443',
    });
    const [path, query] = (uriLink.getAttribute('href') || '').split('?');
    expect(path).toBe('artifacts/s3/reports/output.csv');
    const params = new URLSearchParams(query);
    expect(params.get('namespace')).toBe('team-a');
    expect(JSON.parse(params.get('providerInfo') || '')).toEqual({
      Provider: 's3',
      Params: {
        endpoint: 'https://ceph.example:9443',
        fromEnv: 'true',
      },
    });
  });

  it('uses the native API page token and page size', async () => {
    const artifactsSpy = vi.mocked(Apis.artifactServiceApiV2.artifacts);
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );

    await screen.findByText('Rows per page:');
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByRole('option', { name: '20' }));

    await waitFor(() =>
      expect(artifactsSpy).toHaveBeenLastCalledWith(undefined, '', 20, 'created_at desc', ''),
    );
  });

  it('scopes native artifacts to the selected namespace', async () => {
    const artifactsSpy = vi.mocked(Apis.artifactServiceApiV2.artifacts);
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} namespace='team-a' />
      </MemoryRouter>,
    );

    await waitFor(() => expect(artifactsSpy).toHaveBeenCalled());
    expect(artifactsSpy.mock.calls.at(-1)?.[0]).toBe('team-a');
  });

  it('renders the empty state', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({ artifacts: [] });
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );

    await screen.findByText('No artifacts found.');
  });

  it('shows a page error when the native API fails', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockRejectedValue(
      new Error('Artifact service unavailable'),
    );
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({ additionalInfo: 'Artifact service unavailable', mode: 'error' }),
      ),
    );
  });

  it('keeps matching rows visible when a refresh fails', async () => {
    const listRef = React.createRef<ArtifactList>();
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [{ ...generateArtifacts(1)[0], name: 'last known artifact' }],
    });
    render(
      <MemoryRouter>
        <ArtifactList ref={listRef} {...generateProps()} />
      </MemoryRouter>,
    );
    await screen.findByText('last known artifact');
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockRejectedValue(
      new Error('Artifact service unavailable'),
    );

    await act(async () => listRef.current?.refresh());

    screen.getByText('last known artifact');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({ additionalInfo: 'Artifact service unavailable', mode: 'error' }),
    );
  });

  it('stops pagination when the service repeats the current page token', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockImplementation(
      async (_namespace, pageToken) => ({
        artifacts: [{ ...generateArtifacts(1)[0], name: pageToken || 'first page' }],
        next_page_token: pageToken || 'repeated-page',
      }),
    );
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );
    await screen.findByText('first page');

    fireEvent.click(screen.getByTestId('next-page-btn'));

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining(
            'Artifact service returned a repeated page token: repeated-page',
          ),
        }),
      ),
    );
    expect(screen.getByTestId('next-page-btn')).toBeDisabled();
  });

  it('ignores an older response when reload requests overlap', async () => {
    const first = deferred<{ artifacts: V2beta1Artifact[] }>();
    const second = deferred<{ artifacts: V2beta1Artifact[] }>();
    vi.mocked(Apis.artifactServiceApiV2.artifacts)
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);

    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );
    await waitFor(() => expect(Apis.artifactServiceApiV2.artifacts).toHaveBeenCalledTimes(2));

    second.resolve({ artifacts: [{ ...generateArtifacts(1)[0], name: 'new response' }] });
    await screen.findByText('new response');
    first.resolve({ artifacts: [{ ...generateArtifacts(1)[0], name: 'stale response' }] });
    await waitFor(() => expect(screen.queryByText('stale response')).toBeNull());
    screen.getByText('new response');
  });
});
