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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { ArtifactArtifactType, V2beta1Artifact, V2beta1IOType } from 'src/apisv2beta1/artifact';
import { RoutePage, RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import EnhancedArtifactDetails from 'src/pages/ArtifactDetails';
import { PageProps } from 'src/pages/Page';
import { testBestPractices } from 'src/TestUtils';

vi.mock('src/components/ArtifactPreview', () => ({ default: () => <div>Artifact preview</div> }));

testBestPractices();

describe('ArtifactDetails', () => {
  const TEST_ARTIFACT_ID = 'artifact-42';
  const updateBannerSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
  const historyPushSpy = vi.fn();
  const artifact: V2beta1Artifact = {
    artifact_id: TEST_ARTIFACT_ID,
    name: 'test-artifact',
    description: 'A native artifact',
    type: ArtifactArtifactType.Dataset,
    uri: 's3://pipeline-root/dataset',
    namespace: 'kubeflow',
    metadata: { accuracy: 0.9 },
    created_at: new Date('2026-08-11T12:00:00Z'),
  };

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: `/artifacts/${TEST_ARTIFACT_ID}` } as any,
      match: {
        isExact: true,
        path: RoutePage.ARTIFACT_DETAILS,
        url: `/artifacts/${TEST_ARTIFACT_ID}`,
        params: { [RouteParams.ID]: TEST_ARTIFACT_ID },
      } as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: updateToolbarSpy,
    };
  }

  function renderPage(initialPath = `/artifacts/${TEST_ARTIFACT_ID}`) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialPath]}>
          <EnhancedArtifactDetails {...generateProps()} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
  }

  beforeEach(() => {
    vi.spyOn(Apis.artifactServiceApiV2, 'artifact_1').mockResolvedValue(artifact);
    vi.spyOn(Apis.artifactServiceApiV2, 'artifactTasks').mockResolvedValue({
      artifact_tasks: [
        {
          id: 'relationship-1',
          artifact_id: TEST_ARTIFACT_ID,
          run_id: 'run-1',
          task_id: 'task-1',
          key: 'dataset',
          type: V2beta1IOType.OUTPUT,
        },
      ],
    });
  });

  it('shows a spinner while the artifact is loading', () => {
    vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockReturnValue(new Promise(() => {}));
    renderPage();

    screen.getByRole('progressbar');
  });

  it('renders native artifact details and updates the toolbar', async () => {
    renderPage();

    await screen.findByText('Artifact details');
    expect(screen.getAllByText('test-artifact')).toHaveLength(2);
    screen.getByText('system.Dataset');
    screen.getByText('A native artifact');
    screen.getByText('Artifact preview');
    expect(updateToolbarSpy).toHaveBeenCalledWith({ pageTitle: 'test-artifact' });
    expect(Apis.artifactServiceApiV2.artifactTasks).not.toHaveBeenCalled();
  });

  it('renders native producer and consumer relationships with run links', async () => {
    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('Producing and consuming tasks');
    const runLink = screen.getByRole('link', { name: 'Run run-1 · Task task-1' });
    expect(runLink).toHaveAttribute('href', '/runs/details/run-1');
    screen.getByText('Produced as dataset');
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(1);
  });

  it.each([
    V2beta1IOType.ITERATOR_OUTPUT,
    V2beta1IOType.ONE_OF_OUTPUT,
    V2beta1IOType.TASK_FINAL_STATUS_OUTPUT,
  ])('labels %s relationships as produced', async (type) => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockResolvedValue({
      artifact_tasks: [{ id: type, key: type, type }],
    });

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText(`Produced as ${type}`);
  });

  it('keeps the old lineage bookmark path but labels it as related tasks', async () => {
    renderPage();
    await screen.findByText('Related tasks');

    fireEvent.click(screen.getByText('Related tasks'));

    expect(historyPushSpy).toHaveBeenCalledWith(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);
  });

  it('shows a page error when the native service fails', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockRejectedValue(
      new Error('Artifact not found'),
    );
    renderPage();

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({ additionalInfo: 'Artifact not found', mode: 'error' }),
      ),
    );
    expect(screen.queryByRole('progressbar')).toBeNull();
  });
});
