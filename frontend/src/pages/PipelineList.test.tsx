/*
 * Copyright 2018 The Kubeflow Authors
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

import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import * as React from 'react';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { range } from 'lodash';
import { RoutePage, RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { ButtonKeys } from 'src/lib/Buttons';
import TestUtils, { flushPromisesInAct } from 'src/TestUtils';
import { PageProps } from './Page';
import PipelineList from './PipelineList';
import { CommonTestWrapper } from 'src/TestWrapper';
import { vi } from 'vitest';

beforeEach(() => {
  vi.stubGlobal('matchMedia', (media: string) => ({
    media,
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe('PipelineList', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let pipelineListRef: React.RefObject<PipelineList> | null = null;

  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let listPipelinesSpy: ReturnType<typeof vi.spyOn>;
  let listPipelineVersionsSpy: ReturnType<typeof vi.spyOn>;
  let deletePipelineSpy: ReturnType<typeof vi.spyOn>;

  function spyInit() {
    updateBannerSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    listPipelinesSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'listPipelines')
      .mockResolvedValue({ pipelines: [] });
    listPipelineVersionsSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions')
      .mockResolvedValue({ pipeline_versions: [] });
    deletePipelineSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'deletePipeline')
      .mockResolvedValue(undefined as any);
    vi.spyOn(Apis.pipelineServiceApiV2, 'deletePipelineVersion').mockResolvedValue(
      undefined as any,
    );
  }

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      PipelineList,
      '' as any,
      '' as any,
      null,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function getPipelineListState(): PipelineList['state'] | undefined {
    return pipelineListRef?.current?.state;
  }

  function getToolbarAction(key: string, callIndex?: number) {
    const calls = updateToolbarSpy.mock.calls;
    if (!calls.length) {
      throw new Error(`Toolbar action not available: ${key}`);
    }
    if (callIndex !== undefined) {
      return calls[callIndex][0].actions[key];
    }
    for (let i = calls.length - 1; i >= 0; i--) {
      const action = calls[i][0]?.actions?.[key];
      if (action && action.disabled !== true) {
        return action;
      }
    }
    return calls[calls.length - 1][0].actions[key];
  }

  function getToolbarActionFromInstance(key: string) {
    if (!pipelineListRef?.current) {
      throw new Error(`PipelineList instance not available for ${key}`);
    }
    return pipelineListRef.current.getInitialToolbarState().actions[key];
  }

  function getRowById(id: string): HTMLElement {
    const rows = [
      ...screen.queryAllByTestId('pipeline-card'),
      ...screen.queryAllByTestId('table-row'),
    ];
    const row = rows.find((element) => element.getAttribute('data-row-id') === id);
    if (!row) {
      throw new Error(`Row not found: ${id}`);
    }
    return row;
  }

  async function waitForRowById(id: string): Promise<HTMLElement> {
    let row: HTMLElement | null = null;
    await waitFor(() => {
      row = getRowById(id);
      expect(row).toBeTruthy();
    });
    return row as HTMLElement;
  }

  async function waitForSelectedPipelines(ids: string[]): Promise<void> {
    await waitFor(() => {
      const selected = getPipelineListState()?.selectedIds || [];
      expect(selected.slice().sort()).toEqual(ids.slice().sort());
    });
  }

  async function selectPipeline(id: string): Promise<void> {
    const row = await waitForRowById(id);
    await userEvent.click(within(row).getByRole('checkbox', { name: /^Select pipeline / }));
    await waitForSelectedPipelines([id]);
  }

  async function selectPipelines(ids: string[]): Promise<void> {
    for (const id of ids) {
      const row = await waitForRowById(id);
      await userEvent.click(within(row).getByRole('checkbox', { name: /^Select pipeline / }));
    }
    await waitForSelectedPipelines(ids);
  }

  async function deselectPipeline(id: string): Promise<void> {
    const row = await waitForRowById(id);
    await userEvent.click(within(row).getByRole('checkbox', { name: /^Select pipeline / }));
    await waitForSelectedPipelines([]);
  }

  async function openDeleteDialog(): Promise<any> {
    const deleteBtn = getToolbarActionFromInstance(ButtonKeys.DELETE_RUN);
    await act(async () => {
      await deleteBtn.action();
    });
    await waitFor(() => {
      expect(updateDialogSpy).toHaveBeenCalled();
    });
    return updateDialogSpy.mock.calls[updateDialogSpy.mock.calls.length - 1][0];
  }

  async function renderPipelineList(options?: {
    namespace?: string;
    props?: Partial<PageProps>;
  }): Promise<void> {
    pipelineListRef = React.createRef<PipelineList>();
    const props = { ...generateProps(), ...options?.props } as PageProps;
    renderResult = render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineList ref={pipelineListRef} {...props} namespace={options?.namespace} />
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    await flushPromisesInAct();
    await waitForPipelinesLoad();
  }

  async function waitForPipelinesLoad(): Promise<void> {
    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
    });
    await flushPromisesInAct();
  }

  async function mountWithNPipelines(n: number, options?: { namespace?: string }) {
    listPipelinesSpy.mockResolvedValue({
      pipelines: range(n).map((i) => ({
        id: 'test-pipeline-id' + i,
        pipeline_id: 'test-pipeline-id' + i,
        display_name: 'test pipeline name' + i,
        name: 'test-pipeline-name' + i,
      })),
    });
    await renderPipelineList({ namespace: options?.namespace });
    await waitForPipelinesLoad();
  }

  beforeEach(() => {
    spyInit();
  });

  afterEach(() => {
    pipelineListRef = null;
    vi.resetAllMocks();
  });

  it('applies uploaded dates server-side and preserves the filter across views', async () => {
    await renderPipelineList();
    await userEvent.click(screen.getByRole('button', { name: 'Filter uploaded date' }));
    await userEvent.click(await screen.findByRole('menuitemradio', { name: 'Last 30 days' }));
    await waitFor(() => {
      const call = listPipelinesSpy.mock.calls.at(-1)!;
      const predicates = JSON.parse(decodeURIComponent(call[4] as string)).predicates;
      expect(predicates).toEqual([
        {
          key: 'created_at',
          operation: 'GREATER_THAN_EQUALS',
          timestamp_value: expect.any(String),
        },
      ]);
      expect(call[1]).toBe('');
    });
    const filter = listPipelinesSpy.mock.calls.at(-1)![4];
    expect(listPipelinesSpy.mock.calls.at(-1)![4]).toBe(filter);
    await userEvent.click(screen.getByRole('button', { name: 'Clear uploaded date filter' }));
    await waitFor(() => expect(listPipelinesSpy.mock.calls.at(-1)![4]).toBe(''));
  });

  it('does not place workspace actions beside search', async () => {
    await renderPipelineList();
    expect(screen.queryByRole('button', { name: 'More actions' })).not.toBeInTheDocument();
  });

  it('renders an empty list with empty state message', async () => {
    await renderPipelineList();
    await waitForPipelinesLoad();
    expect(
      screen.getByText('No pipelines found. Click "Upload pipeline" to start.'),
    ).toBeInTheDocument();
  });

  it('renders a list of one pipeline', async () => {
    listPipelinesSpy.mockResolvedValue({
      pipelines: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          description: 'test pipeline description',
          display_name: 'pipeline1',
          name: 'pipeline1',
          parameters: [],
        },
      ],
    });
    await renderPipelineList();
    await waitForPipelinesLoad();
    expect(screen.getByText('pipeline1')).toBeInTheDocument();
  });

  it('renders a list of one pipeline with no description or created date', async () => {
    listPipelinesSpy.mockResolvedValue({
      pipelines: [
        {
          display_name: 'pipeline1',
          name: 'pipeline1',
          parameters: [],
        },
      ],
    });
    await renderPipelineList();
    await waitForPipelinesLoad();
    expect(screen.getByText('pipeline1')).toBeInTheDocument();
  });

  it('renders a list of one pipeline with a display name that is not the same as the name', async () => {
    listPipelinesSpy.mockResolvedValue({
      pipelines: [
        {
          display_name: 'Pipeline One',
          name: 'pipeline1',
          parameters: [],
        },
      ],
    });
    await renderPipelineList();
    await waitForPipelinesLoad();
    expect(screen.getByText('Pipeline One')).toBeInTheDocument();
  });

  it('renders a list of one pipeline with error', async () => {
    listPipelinesSpy.mockResolvedValue({
      pipelines: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          description: 'test pipeline description',
          error: 'oops! could not load pipeline',
          display_name: 'pipeline1',
          name: 'pipeline1',
          parameters: [],
        } as any,
      ],
    });
    await renderPipelineList();
    await waitForPipelinesLoad();
    expect(screen.getByText('pipeline1')).toBeInTheDocument();
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', async () => {
    listPipelinesSpy.mockResolvedValue({
      pipelines: [{ display_name: 'pipeline1', name: 'pipeline1' }],
    });
    await renderPipelineList({ namespace: 'test-ns' });
    await waitForPipelinesLoad();
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('test-ns', '', 20, 'created_at desc', '');
    expect(getPipelineListState()).toHaveProperty('displayPipelines', [
      { display_name: 'pipeline1', name: 'pipeline1' },
    ]);
  });

  it('has a Refresh button, clicking it refreshes the pipeline list', async () => {
    await mountWithNPipelines(1, { namespace: 'test-ns' });
    listPipelinesSpy.mockClear();
    const refreshBtn = getToolbarActionFromInstance(ButtonKeys.REFRESH);
    expect(refreshBtn).toBeDefined();
    await act(async () => {
      await refreshBtn.action();
    });
    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    });
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('test-ns', '', 20, 'created_at desc', '');
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('shows error banner when listing pipelines fails', async () => {
    listPipelinesSpy.mockRejectedValue(new Error('bad stuff happened'));
    await renderPipelineList();
    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'bad stuff happened',
          message:
            'Error: failed to retrieve list of pipelines. Click Details for more information.',
          mode: 'error',
        }),
      );
    });
  });

  it('shows error banner when listing pipelines fails after refresh', async () => {
    await renderPipelineList();
    await waitForPipelinesLoad();
    listPipelinesSpy.mockClear();
    const refreshBtn = getToolbarActionFromInstance(ButtonKeys.REFRESH);
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listPipelinesSpy as any, 'bad stuff happened');
    await act(async () => {
      await refreshBtn.action();
    });
    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    });
    expect(listPipelinesSpy).toHaveBeenLastCalledWith(undefined, '', 20, 'created_at desc', '');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad stuff happened',
        message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('retains a list error when an in-flight version load succeeds', async () => {
    let finishVersions!: (value: { pipeline_versions: [] }) => void;
    listPipelineVersionsSpy.mockImplementation(
      () =>
        new Promise((resolve) => {
          finishVersions = resolve;
        }),
    );
    await mountWithNPipelines(1);
    await waitFor(() => expect(listPipelineVersionsSpy).toHaveBeenCalled());
    listPipelinesSpy.mockRejectedValueOnce(new Error('list failed'));
    await act(async () => {
      await getToolbarActionFromInstance(ButtonKeys.REFRESH).action();
    });
    const errorBanner = expect.objectContaining({ additionalInfo: 'list failed', mode: 'error' });
    expect(updateBannerSpy).toHaveBeenLastCalledWith(errorBanner);
    await act(async () => {
      finishVersions({ pipeline_versions: [] });
    });
    expect(updateBannerSpy).toHaveBeenLastCalledWith(errorBanner);
  });

  it('keeps latest version errors local and allows retry', async () => {
    listPipelineVersionsSpy.mockRejectedValue(new Error('versions failed'));
    await mountWithNPipelines(1);
    expect(await screen.findByRole('button', { name: 'Retry version lookup' })).toBeInTheDocument();
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await userEvent.click(screen.getByRole('button', { name: 'Retry version lookup' }));
    expect(await screen.findByText('No versions')).toBeInTheDocument();
  });

  it('hides error banner when listing pipelines fails then succeeds', async () => {
    listPipelinesSpy.mockRejectedValue(new Error('bad stuff happened'));
    await renderPipelineList();
    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'bad stuff happened',
          message:
            'Error: failed to retrieve list of pipelines. Click Details for more information.',
          mode: 'error',
        }),
      );
    });
    updateBannerSpy.mockReset();
    listPipelinesSpy.mockClear();

    const refreshBtn = getToolbarActionFromInstance(ButtonKeys.REFRESH);
    listPipelinesSpy.mockResolvedValueOnce({
      pipelines: [{ display_name: 'pipeline1', name: 'pipeline1' }],
    });
    await act(async () => {
      await refreshBtn.action();
    });
    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    });
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('renders pipeline names as links to their details pages', async () => {
    await mountWithNPipelines(1);
    const row = await waitForRowById('test-pipeline-id0');
    const link = within(row).getByRole('link');
    expect(link).toHaveTextContent('test pipeline name0');
    expect(link.getAttribute('href')).toBe(
      RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(
        ':' + RouteParams.pipelineId + '?',
        'test-pipeline-id0',
      ),
    );
  });

  it('always has upload pipeline button enabled', async () => {
    await mountWithNPipelines(1);
    expect(getToolbarAction(ButtonKeys.NEW_PIPELINE_VERSION).disabled).not.toBe(true);
  });

  it('enables delete button when one pipeline is selected', async () => {
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    await waitFor(() => {
      expect(getToolbarAction(ButtonKeys.DELETE_RUN).disabled).toBe(false);
    });
  });

  it('enables delete button when two pipelines are selected', async () => {
    await mountWithNPipelines(2);
    await selectPipelines(['test-pipeline-id0', 'test-pipeline-id1']);
    await waitFor(() => {
      expect(getToolbarAction(ButtonKeys.DELETE_RUN).disabled).toBe(false);
    });
  });

  it('re-disables delete button pipelines are unselected', async () => {
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    await deselectPipeline('test-pipeline-id0');
    await waitFor(() => {
      expect(getToolbarAction(ButtonKeys.DELETE_RUN).disabled).toBe(true);
    });
  });

  it('shows delete dialog when delete button is clicked', async () => {
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    const call = await openDeleteDialog();
    expect(call).toHaveProperty('title', 'Delete 1 pipeline?');
  });

  it('shows delete dialog when delete button is clicked, indicating several pipelines to delete', async () => {
    await mountWithNPipelines(5);
    await selectPipelines(['test-pipeline-id0', 'test-pipeline-id2', 'test-pipeline-id3']);
    const call = await openDeleteDialog();
    expect(call).toHaveProperty('title', 'Delete 3 pipelines?');
  });

  it('does not call delete API for selected pipeline when delete dialog is canceled', async () => {
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    const call = await openDeleteDialog();
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await act(async () => {
      await cancelBtn.onClick();
    });
    expect(deletePipelineSpy).not.toHaveBeenCalled();
  });

  it('calls delete API for selected pipeline after delete dialog is confirmed', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitFor(() => {
      expect(deletePipelineSpy).toHaveBeenLastCalledWith('test-pipeline-id0', false);
    });
  });

  it('updates the selected indices after a pipeline is deleted', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(5);
    await selectPipeline('test-pipeline-id0');
    deletePipelineSpy.mockResolvedValue(undefined as any);
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitForSelectedPipelines([]);
  });

  it('updates the selected indices after multiple pipelines are deleted', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(5);
    await selectPipelines(['test-pipeline-id0', 'test-pipeline-id3']);
    deletePipelineSpy.mockResolvedValue(undefined as any);
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitForSelectedPipelines([]);
  });

  it('calls delete API for all selected pipelines after delete dialog is confirmed', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(5);
    await selectPipelines(['test-pipeline-id0', 'test-pipeline-id1', 'test-pipeline-id4']);
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitFor(() => {
      expect(deletePipelineSpy).toHaveBeenCalledTimes(3);
    });
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id0', false);
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id1', false);
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id4', false);
  });

  it('shows snackbar confirmation after pipeline is deleted', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    deletePipelineSpy.mockResolvedValue(undefined as any);
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitFor(() => {
      expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
        message: 'Deletion succeeded for 1 pipeline',
        open: true,
      });
    });
  });

  it('shows error dialog when pipeline deletion fails', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');
    TestUtils.makeErrorResponseOnce(deletePipelineSpy as any, 'woops, failed');
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitFor(() => {
      expect(updateDialogSpy.mock.calls.length).toBeGreaterThan(1);
    });
    const lastCall = updateDialogSpy.mock.calls[updateDialogSpy.mock.calls.length - 1][0];
    expect(lastCall).toMatchObject({
      content: 'Failed to delete pipeline: test-pipeline-id0 with error: "woops, failed"',
      title: 'Failed to delete some pipelines and/or some pipeline versions',
    });
  });

  it('shows error dialog when multiple pipeline deletions fail', async () => {
    listPipelineVersionsSpy.mockResolvedValue({ pipeline_versions: [] });
    await mountWithNPipelines(5);
    await selectPipelines([
      'test-pipeline-id0',
      'test-pipeline-id2',
      'test-pipeline-id1',
      'test-pipeline-id3',
    ]);
    deletePipelineSpy.mockImplementation((id) => {
      if (id.indexOf(3) === -1 && id.indexOf(2) === -1) {
        throw {
          text: () => Promise.resolve('woops, failed!'),
        };
      }
      return Promise.resolve();
    });
    const call = await openDeleteDialog();
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await waitFor(() => {
      expect(updateDialogSpy).toHaveBeenCalledTimes(2);
    });
    const lastCall = updateDialogSpy.mock.calls[updateDialogSpy.mock.calls.length - 1][0];
    expect(lastCall).toMatchObject({
      content:
        'Failed to delete pipeline: test-pipeline-id0 with error: "woops, failed!"\n\n' +
        'Failed to delete pipeline: test-pipeline-id1 with error: "woops, failed!"',
      title: 'Failed to delete some pipelines and/or some pipeline versions',
    });

    await waitFor(() => {
      expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
        message: 'Deletion succeeded for 2 pipelines',
        open: true,
      });
    });
  });

  it('shows "Delete All" dialog when pipelines have versions and calls API with cascade=true', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          pipeline_version_id: 'test-version-id-1',
          display_name: 'test version 1',
          name: 'test-version-1',
        },
        {
          pipeline_version_id: 'test-version-id-2',
          display_name: 'test version 2',
          name: 'test-version-2',
        },
      ],
    });

    deletePipelineSpy.mockResolvedValue(undefined as any);
    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');

    const call = await openDeleteDialog();

    expect(call.title).toBe('Delete 1 pipeline?');
    expect(call.content).toContain('pipeline has existing versions');
    expect(call.content).toContain('Deleting this pipeline will also delete all its versions');
    expect(call.content).toContain('This action cannot be undone');

    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete All');
    expect(confirmBtn).toBeDefined();

    await act(async () => {
      await confirmBtn.onClick();
    });
    expect(deletePipelineSpy).toHaveBeenLastCalledWith('test-pipeline-id0', true);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'Deletion succeeded for 1 pipeline',
      open: true,
    });
  });

  it('shows "Delete All" dialog for multiple pipelines with versions', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          pipeline_version_id: 'test-version-id-1',
          display_name: 'test version 1',
          name: 'test-version-1',
        },
      ],
    });

    deletePipelineSpy.mockResolvedValue(undefined as any);
    await mountWithNPipelines(3);

    await selectPipelines(['test-pipeline-id0', 'test-pipeline-id1']);

    const call = await openDeleteDialog();

    expect(call.title).toBe('Delete 2 pipelines?');
    expect(call.content).toContain('pipelines have existing versions');
    expect(call.content).toContain('Deleting these pipelines will also delete all their versions');

    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete All');
    expect(confirmBtn).toBeDefined();
    await act(async () => {
      await confirmBtn.onClick();
    });
    expect(deletePipelineSpy).toHaveBeenCalledTimes(2);
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id0', true);
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id1', true);
  });

  it('does not call delete API when "Delete All" dialog is canceled', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          pipeline_version_id: 'test-version-id-1',
          display_name: 'test version 1',
          name: 'test-version-1',
        },
      ],
    });

    await mountWithNPipelines(1);
    await selectPipeline('test-pipeline-id0');

    const call = await openDeleteDialog();
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await act(async () => {
      await cancelBtn.onClick();
    });
    expect(deletePipelineSpy).not.toHaveBeenCalled();
  });
});
