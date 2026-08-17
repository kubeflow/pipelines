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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { ArtifactArtifactType, V2beta1Artifact, V2beta1IOType } from 'src/apisv2beta1/artifact';
import { RoutePage, RouteParams } from 'src/components/Router';
import { PlotType } from 'src/components/viewers/Viewer';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
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
    localStorage.clear();
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

  it('uses the producing output key to render legacy UI metadata', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockResolvedValue({
      artifact_id: TEST_ARTIFACT_ID,
      name: 'legacy-output',
      uri: 's3://reports/metadata.json',
      namespace: 'kubeflow',
    });
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, type) =>
        type === V2beta1IOType.OUTPUT
          ? {
              artifact_tasks: [
                {
                  artifact_id: TEST_ARTIFACT_ID,
                  key: 'mlpipeline-ui-metadata',
                  type: V2beta1IOType.OUTPUT,
                },
              ],
            }
          : { artifact_tasks: [] },
    );
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ data: [['restored']], labels: ['value'], type: PlotType.TABLE }],
      errors: [],
    });

    renderPage();

    expect(await screen.findByText('restored')).toBeVisible();
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(4);
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledWith(
      undefined,
      undefined,
      [TEST_ARTIFACT_ID],
      V2beta1IOType.OUTPUT,
      undefined,
      1,
      'id asc',
      expect.stringContaining('mlpipeline-ui-metadata'),
    );
    expect(
      vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mock.calls.map((call) => call[3]),
    ).toEqual([
      V2beta1IOType.OUTPUT,
      V2beta1IOType.ITERATOR_OUTPUT,
      V2beta1IOType.ONE_OF_OUTPUT,
      V2beta1IOType.TASK_FINAL_STATUS_OUTPUT,
    ]);
    expect(loadSpy).toHaveBeenCalledTimes(1);
  });

  it('does not treat a consumed mlpipeline-ui-metadata key as viewer metadata', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockResolvedValue({
      artifact_id: TEST_ARTIFACT_ID,
      name: 'ordinary-input',
      uri: 's3://reports/data.json',
      namespace: 'kubeflow',
    });
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockResolvedValue({
      artifact_tasks: [
        {
          artifact_id: TEST_ARTIFACT_ID,
          key: 'mlpipeline-ui-metadata',
          type: V2beta1IOType.TASK_OUTPUT_INPUT,
        },
      ],
    });
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'load');

    renderPage();

    await waitFor(() => expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(4));
    expect(loadSpy).not.toHaveBeenCalled();
  });

  it('keeps a confirmed legacy visualization when another relationship lookup fails', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifact_1).mockResolvedValue({
      artifact_id: TEST_ARTIFACT_ID,
      name: 'legacy-output',
      uri: 's3://reports/metadata.json',
      namespace: 'kubeflow',
    });
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, type) => {
        if (type === V2beta1IOType.OUTPUT) {
          return {
            artifact_tasks: [
              {
                artifact_id: TEST_ARTIFACT_ID,
                key: 'mlpipeline-ui-metadata',
                type: V2beta1IOType.OUTPUT,
              },
            ],
          };
        }
        throw new Error(`${type} unavailable`);
      },
    );
    vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ data: [['restored']], labels: ['value'], type: PlotType.TABLE }],
      errors: [],
    });

    renderPage();

    expect(await screen.findByText('restored')).toBeVisible();
    expect(screen.getByText(/Some artifact relationships could not be checked/)).toBeVisible();
  });

  it('renders native producer and consumer relationships with run links', async () => {
    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('Producing and consuming tasks');
    const runLink = await screen.findByRole('link', { name: 'Run run-1 · Task task-1' });
    expect(runLink).toHaveAttribute('href', '/runs/details/run-1?task=task-1');
    screen.getByText('Produced as dataset');
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(1);
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledWith(
      undefined,
      undefined,
      [TEST_ARTIFACT_ID],
      undefined,
      '',
      10,
      'id asc',
    );
  });

  it('requests and renders one relationship page at a time', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, _type, pageToken) =>
        pageToken === 'next-page'
          ? {
              artifact_tasks: [
                {
                  id: 'relationship-2',
                  run_id: 'run-2',
                  task_id: 'task-2',
                  key: 'consumer-input',
                  type: V2beta1IOType.TASK_OUTPUT_INPUT,
                },
              ],
            }
          : {
              artifact_tasks: [
                {
                  id: 'relationship-1',
                  run_id: 'run-1',
                  task_id: 'task-1',
                  key: 'producer-output',
                  type: V2beta1IOType.OUTPUT,
                },
              ],
              next_page_token: 'next-page',
            },
    );

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('Produced as producer-output');
    expect(screen.queryByText('Consumed as consumer-input')).not.toBeInTheDocument();
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId('next-page-btn'));

    await screen.findByText('Consumed as consumer-input');
    expect(screen.queryByText('Produced as producer-output')).not.toBeInTheDocument();
    expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      [TEST_ARTIFACT_ID],
      undefined,
      'next-page',
      10,
      'id asc',
    );

    fireEvent.click(screen.getByTestId('prev-page-btn'));

    await screen.findByText('Produced as producer-output');
    expect(screen.queryByText('Consumed as consumer-input')).not.toBeInTheDocument();
  });

  it('renders an empty related-task page', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockResolvedValue({ artifact_tasks: [] });

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('No related tasks found.');
  });

  it('shows an actionable error when a relationship page fails', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockRejectedValue(
      new Error('Artifact service unavailable'),
    );

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('Unable to load related tasks. Refresh the page to try again.');
    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    await screen.findByText('Artifact service unavailable');
  });

  it('keeps the cached relationship page visible when returning to it fails', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, _type, pageToken) =>
        pageToken === 'next-page'
          ? {
              artifact_tasks: [
                {
                  id: 'relationship-2',
                  key: 'second-page',
                  type: V2beta1IOType.TASK_OUTPUT_INPUT,
                },
              ],
            }
          : {
              artifact_tasks: [
                {
                  id: 'relationship-1',
                  key: 'cached-first-page',
                  type: V2beta1IOType.TASK_OUTPUT_INPUT,
                },
              ],
              next_page_token: 'next-page',
            },
    );
    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);
    await screen.findByText('Consumed as cached-first-page');
    fireEvent.click(screen.getByTestId('next-page-btn'));
    await screen.findByText('Consumed as second-page');
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockRejectedValue(
      new Error('Artifact service unavailable'),
    );

    fireEvent.click(screen.getByTestId('prev-page-btn'));

    await screen.findByText('Consumed as cached-first-page');
    screen.getByText('Unable to load related tasks. Refresh the page to try again.');
  });

  it('stops pagination when the service repeats the current page token', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, _type, pageToken) => ({
        artifact_tasks: [
          {
            id: pageToken || 'first',
            key: pageToken || 'first',
            type: V2beta1IOType.TASK_OUTPUT_INPUT,
          },
        ],
        next_page_token: pageToken || 'repeated-page',
      }),
    );

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);
    await screen.findByText('Consumed as first');

    fireEvent.click(screen.getByTestId('next-page-btn'));

    await screen.findByText('Unable to load related tasks. Refresh the page to try again.');
    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    await screen.findByText('Artifact service returned a repeated page token: repeated-page');
    expect(screen.getByTestId('next-page-btn')).toBeDisabled();
  });

  it('does not replace a newer page-size result when an older request finishes later', async () => {
    let resolveFirstPage!: (value: {
      artifact_tasks: Array<{ id: string; key: string; type?: V2beta1IOType }>;
      next_page_token: string;
    }) => void;
    const firstPage = new Promise<{
      artifact_tasks: Array<{ id: string; key: string; type?: V2beta1IOType }>;
      next_page_token: string;
    }>((resolve) => {
      resolveFirstPage = resolve;
    });
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockImplementation(
      async (_taskIds, _runIds, _artifactIds, _type, _pageToken, pageSize) =>
        pageSize === 10
          ? firstPage
          : {
              artifact_tasks: [
                {
                  id: 'newer',
                  key: 'newer-page-size',
                  type: V2beta1IOType.TASK_OUTPUT_INPUT,
                },
              ],
            },
    );

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);
    await waitFor(() => expect(Apis.artifactServiceApiV2.artifactTasks).toHaveBeenCalledTimes(1));

    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByRole('option', { name: '20' }));
    await screen.findByText('Consumed as newer-page-size');

    await act(async () => {
      resolveFirstPage({
        artifact_tasks: [
          {
            id: 'older',
            key: 'older-page-size',
            type: V2beta1IOType.TASK_OUTPUT_INPUT,
          },
        ],
        next_page_token: 'older-next-page',
      });
      await firstPage;
    });

    expect(screen.queryByText('Consumed as older-page-size')).not.toBeInTheDocument();
    screen.getByText('Consumed as newer-page-size');
    expect(screen.getByTestId('next-page-btn')).toBeDisabled();
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

  it.each([undefined, V2beta1IOType.UNSPECIFIED])(
    'labels %s relationships as unknown instead of consumed',
    async (type) => {
      vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockResolvedValue({
        artifact_tasks: [{ id: 'unknown', key: 'dataset', type }],
      });

      renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

      await screen.findByText('Related as unknown: dataset');
      expect(screen.queryByText('Consumed as dataset')).not.toBeInTheDocument();
    },
  );

  it('preserves a future relationship type instead of classifying it as consumed', async () => {
    vi.mocked(Apis.artifactServiceApiV2.artifactTasks).mockResolvedValue({
      artifact_tasks: [
        { id: 'future', key: 'dataset', type: 'FUTURE_RELATIONSHIP' as V2beta1IOType },
      ],
    });

    renderPage(`/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await screen.findByText('Related as FUTURE_RELATIONSHIP: dataset');
    expect(screen.queryByText('Consumed as dataset')).not.toBeInTheDocument();
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
