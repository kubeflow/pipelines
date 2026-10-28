/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { Apis } from 'src/lib/Apis';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { LatestPipelineVersion } from './LatestPipelineVersion';

const pipelineId = 'pipeline/one%';
const listVersions = vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
beforeEach(() => {
  listVersions.mockReset();
});

function setup(namespace = 'team-a') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const selectRow = vi.fn();
  const view = (scope: string) => (
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NamespaceContext.Provider value={scope}>
          <div onClick={selectRow}>
            <LatestPipelineVersion id={pipelineId} value={pipelineId} />
          </div>
        </NamespaceContext.Provider>
      </QueryClientProvider>
    </MemoryRouter>
  );
  const rendered = render(view(namespace));
  return {
    ...rendered,
    selectRow,
    changeNamespace: (scope: string) => rendered.rerender(view(scope)),
  };
}

it('requests only the newest version and encodes both identifiers in its link', async () => {
  listVersions.mockResolvedValue({
    pipeline_versions: [{ pipeline_version_id: 'version/two%', display_name: 'Latest release' }],
  });
  const { selectRow } = setup();
  const link = await screen.findByRole('link', { name: 'Latest release' });
  expect(listVersions).toHaveBeenCalledExactlyOnceWith(pipelineId, undefined, 1, 'created_at desc');
  expect(link).toHaveAttribute(
    'href',
    '/pipelines/details/pipeline%2Fone%25/version/version%2Ftwo%25',
  );
  await userEvent.click(link);
  expect(selectRow).not.toHaveBeenCalled();
});

it('distinguishes no versions from a failed lookup and retries without selecting the row', async () => {
  listVersions.mockRejectedValueOnce(new Error('unavailable'));
  const { selectRow } = setup();
  const retry = await screen.findByRole('button', { name: 'Retry version lookup' });
  expect(screen.queryByText('No versions')).not.toBeInTheDocument();
  listVersions.mockResolvedValue({ pipeline_versions: [] });
  await userEvent.click(retry);
  expect(await screen.findByText('No versions')).toBeVisible();
  expect(selectRow).not.toHaveBeenCalled();
  expect(listVersions).toHaveBeenCalledTimes(2);
});

it('does not display cached metadata from a different namespace while its lookup is pending', async () => {
  listVersions.mockResolvedValueOnce({
    pipeline_versions: [{ pipeline_version_id: 'private-a', display_name: 'Team A version' }],
  });
  const { changeNamespace } = setup();
  await screen.findByRole('link', { name: 'Team A version' });
  let finish!: (
    value: Awaited<ReturnType<typeof Apis.pipelineServiceApiV2.listPipelineVersions>>,
  ) => void;
  listVersions.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  changeNamespace('team-b');
  expect(screen.queryByRole('link', { name: 'Team A version' })).not.toBeInTheDocument();
  expect(screen.getByLabelText('Loading latest version')).toBeVisible();
  await waitFor(() => expect(listVersions).toHaveBeenCalledTimes(2));
  finish({
    pipeline_versions: [{ pipeline_version_id: 'private-b', display_name: 'Team B version' }],
  });
  expect(await screen.findByRole('link', { name: 'Team B version' })).toBeVisible();
  expect(screen.queryByRole('link', { name: 'Team A version' })).not.toBeInTheDocument();
});
