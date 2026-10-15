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
import { MemoryRouter } from 'react-router';
import { ArtifactArtifactType, V2beta1Artifact } from 'src/apisv2beta1/artifact';
import { RoutePage } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { ArtifactList } from 'src/pages/ArtifactList';
import { PageProps } from 'src/pages/Page';
import TestUtils, { testBestPractices } from 'src/TestUtils';

testBestPractices();

describe('ArtifactList', () => {
  const updateBannerSpy = vi.fn();
  const navigateSpy = vi.fn();

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
      navigateSpy,
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

  it.each(['artifact-1', 'folder/item%2F?#'])(
    'links raw artifact ID %s to its details',
    async (id) => {
      vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
        artifacts: [{ ...generateArtifacts(1)[0], artifact_id: id }],
      });
      render(
        <MemoryRouter>
          <div className='kfp-theme'>
            <ArtifactList {...generateProps()} />
          </div>
        </MemoryRouter>,
      );

      const artifactLink = await screen.findByRole('link', { name: 'test artifact 1' });
      expect(artifactLink).toHaveAttribute('href', `/artifacts/${encodeURIComponent(id)}`);
      expect(screen.getByRole('link', { name: `Artifact ID ${id}` })).toHaveAttribute(
        'href',
        `/artifacts/${encodeURIComponent(id)}`,
      );
      screen.getByText('system.Dataset');
      screen.getByText('kubeflow');
    },
  );

  it('includes the artifact namespace and separates the stored query from the URI path', async () => {
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
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    const uriLink = await screen.findByRole('link', {
      name: 's3://reports/output.csv?endpoint=https%3A%2F%2Fceph.example%3A9443',
    });
    const [path, query] = (uriLink.getAttribute('href') || '').split('?');
    expect(path).toBe('artifacts/get');
    const params = new URLSearchParams(query);
    expect(params.get('source')).toBe('s3');
    expect(params.get('bucket')).toBe('reports');
    expect(params.get('key')).toBe('output.csv');
    expect(params.get('download')).toBe('true');
    expect(params.get('namespace')).toBe('team-a');
    expect(params.get('artifactUriQuery')).toBe('endpoint=https%3A%2F%2Fceph.example%3A9443');
    expect(params.has('providerInfo')).toBe(false);
  });

  it('compacts long IDs while preserving an accessible full identity and details link', async () => {
    const id = '12345678-1234-1234-1234-123456789abc';
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [{ ...generateArtifacts(1)[0], artifact_id: id }],
    });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    const idLink = await screen.findByRole('link', { name: `Artifact ID ${id}` });
    expect(idLink).toHaveTextContent('12345678…9abc');
    expect(idLink).toHaveAttribute('href', `/artifacts/${id}`);
    expect(idLink.closest('td')).toHaveTextContent('12345678…9abc');
    expect(screen.getByRole('table', { name: 'Artifacts' })).toContainElement(idLink);
    expect(screen.getByText('system.Dataset')).toHaveStyle({ whiteSpace: 'nowrap' });
    expect(screen.getByText('kubeflow')).toHaveStyle({ whiteSpace: 'normal' });
  });

  it('ellipsizes long types with a full tooltip and keeps date and time on deliberate lines', async () => {
    const createdAt = new Date('2026-08-01T12:34:56Z');
    const longName = 'a-very-long-artifact-name-that-remains-readable-without-clipping';
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [
        {
          ...generateArtifacts(1)[0],
          type: ArtifactArtifactType.ClassificationMetric,
          name: longName,
          namespace: 'a-long-namespace-that-can-wrap',
          created_at: createdAt,
        },
      ],
    });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    const type = await screen.findByText('system.ClassificationMetrics');
    expect(type).toHaveStyle({
      display: 'block',
      whiteSpace: 'nowrap',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
    });
    expect(type).toHaveAttribute('title', 'system.ClassificationMetrics');

    const date = screen.getByText(createdAt.toLocaleDateString());
    const time = screen.getByText(createdAt.toLocaleTimeString());
    expect(date).toHaveStyle({ display: 'block', whiteSpace: 'nowrap' });
    expect(time).toHaveStyle({ display: 'block', whiteSpace: 'nowrap' });
    expect(time.parentElement).toHaveAttribute('datetime', createdAt.toISOString());
    expect(time.parentElement).toHaveAttribute('title', createdAt.toLocaleString());
    expect(screen.getByRole('link', { name: longName })).toHaveStyle({
      whiteSpace: 'normal',
      overflowWrap: 'anywhere',
    });
    expect(screen.getByText('a-long-namespace-that-can-wrap')).toHaveStyle({
      whiteSpace: 'normal',
      overflowWrap: 'anywhere',
    });
  });

  it('uses the native API page token and page size', async () => {
    const artifactsSpy = vi.mocked(Apis.artifactServiceApiV2.artifacts);
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    await screen.findByRole('combobox', { name: 'Rows per page' });
    fireEvent.change(screen.getByRole('combobox', { name: 'Rows per page' }), {
      target: { value: '20' },
    });

    await waitFor(() =>
      expect(artifactsSpy).toHaveBeenLastCalledWith(undefined, '', 20, 'created_at desc', ''),
    );
  });

  it('scopes native artifacts to the selected namespace', async () => {
    const artifactsSpy = vi.mocked(Apis.artifactServiceApiV2.artifacts);
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} namespace='team-a' />
        </div>
      </MemoryRouter>,
    );

    await waitFor(() => expect(artifactsSpy).toHaveBeenCalled());
    expect(artifactsSpy.mock.calls.at(-1)?.[0]).toBe('team-a');
  });

  it('renders the empty state', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({ artifacts: [] });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
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
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({ additionalInfo: 'Artifact service unavailable', mode: 'error' }),
      ),
    );
  });

  it('skips an artifact without an ID while preserving valid rows', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [
        { ...generateArtifacts(1)[0], artifact_id: undefined, name: 'malformed artifact' },
        { ...generateArtifacts(1)[0], name: 'valid artifact' },
      ],
    });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          additionalInfo: expect.stringContaining(
            '1 artifact could not be displayed because the Artifact service returned no ID.',
          ),
          mode: 'error',
        }),
      ),
    );
    expect(screen.queryByRole('link', { name: 'malformed artifact' })).toBeNull();
    expect(screen.getByRole('link', { name: 'valid artifact' })).toBeVisible();
  });

  it('keeps matching rows visible when a refresh fails', async () => {
    const listRef = React.createRef<ArtifactList>();
    vi.mocked(Apis.artifactServiceApiV2.artifacts).mockResolvedValue({
      artifacts: [{ ...generateArtifacts(1)[0], name: 'last known artifact' }],
    });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList ref={listRef} {...generateProps()} />
        </div>
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
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
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
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );
    await waitFor(() => expect(Apis.artifactServiceApiV2.artifacts).toHaveBeenCalledTimes(2));

    second.resolve({ artifacts: [{ ...generateArtifacts(1)[0], name: 'new response' }] });
    await screen.findByText('new response');
    first.resolve({ artifacts: [{ ...generateArtifacts(1)[0], name: 'stale response' }] });
    await waitFor(() => expect(screen.queryByText('stale response')).toBeNull());
    screen.getByText('new response');
  });
  it('combines backend type and name filters, resets the token, and preserves sort when changing type', async () => {
    const api = vi
      .mocked(Apis.artifactServiceApiV2.artifacts)
      .mockResolvedValue({ artifacts: generateArtifacts(2), next_page_token: 'next' });
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} namespace='team-a' />
        </div>
      </MemoryRouter>,
    );
    await screen.findByRole('link', { name: 'test artifact 1' });
    fireEvent.change(screen.getByRole('searchbox', { name: 'Filter artifacts by name' }), {
      target: { value: 'test' },
    });
    await waitFor(() => expect(api.mock.lastCall?.[4]).toContain('test'));
    fireEvent.click(screen.getByRole('button', { name: 'Name', exact: true }));
    await waitFor(() => expect(api.mock.lastCall?.[3]).toBe('name'));
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    await waitFor(() => expect(api.mock.lastCall?.[1]).toBe('next'));
    fireEvent.click(screen.getByRole('button', { name: 'Metrics', exact: true }));
    await waitFor(() => expect(api.mock.lastCall?.[1]).toBeFalsy());
    expect(api.mock.lastCall?.[0]).toBe('team-a');
    expect(api.mock.lastCall?.[3]).toBe('name');
    expect(JSON.parse(decodeURIComponent(api.mock.lastCall?.[4] || ''))).toEqual({
      predicates: [
        { key: 'name', operation: 'IS_SUBSTRING', string_value: 'test' },
        { key: 'type', operation: 'IN', int_values: { values: [6, 7, 8] } },
      ],
    });
    expect(screen.getByRole('searchbox')).toHaveValue('test');
    expect(screen.getByRole('button', { name: 'Metrics', exact: true })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    fireEvent.click(screen.getByRole('button', { name: 'All', exact: true }));
    await waitFor(() =>
      expect(JSON.parse(decodeURIComponent(api.mock.lastCall?.[4] || '')).predicates).toHaveLength(
        1,
      ),
    );
  });

  it('does not allow an old type response to replace the new type and encodes artifact IDs', async () => {
    const old = deferred<{ artifacts: V2beta1Artifact[] }>();
    const api = vi
      .mocked(Apis.artifactServiceApiV2.artifacts)
      .mockImplementation(async (_namespace, _token, _size, _sort, filter) =>
        filter?.includes('int_values')
          ? {
              artifacts: [
                {
                  artifact_id: 'artifact/with space',
                  name: 'Model result',
                  type: ArtifactArtifactType.Model,
                },
              ],
            }
          : old.promise,
      );
    render(
      <MemoryRouter>
        <div className='kfp-theme'>
          <ArtifactList {...generateProps()} />
        </div>
      </MemoryRouter>,
    );
    await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
    fireEvent.click(screen.getByRole('button', { name: 'Model', exact: true }));
    const result = await screen.findByRole('link', { name: 'Model result' });
    expect(result).toHaveAttribute('href', '/artifacts/artifact%2Fwith%20space');
    await act(async () => old.resolve({ artifacts: generateArtifacts(1) }));
    expect(screen.queryByRole('link', { name: 'test artifact 1' })).not.toBeInTheDocument();
    expect(result).toBeVisible();
    fireEvent.click(result.closest('tr')!);
    expect(navigateSpy).toHaveBeenCalledWith('/artifacts/artifact%2Fwith%20space');
  });
});
