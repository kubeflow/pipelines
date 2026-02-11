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

import * as React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import EnhancedExperimentList, { ExperimentList } from './ExperimentList';
import TestUtils from 'src/TestUtils';
import * as Utils from 'src/lib/Utils';
import { logger } from 'src/lib/Utils';
import { V2beta1RunStorageState, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { ExpandState } from 'src/components/CustomTable';
import { PageProps } from './Page';
import { RoutePage, QUERY_PARAMS, RouteParams } from 'src/components/Router';
import { range } from 'lodash';
import { ButtonKeys } from 'src/lib/Buttons';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { MemoryRouter } from 'react-router-dom';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';

// Default arguments for Apis.experimentServiceApi.listExperiment.
const LIST_EXPERIMENT_DEFAULTS = [
  '', // page token
  10, // page size
  'created_at desc', // sort by
  encodeURIComponent(
    JSON.stringify({
      predicates: [
        {
          key: 'storage_state',
          operation: V2beta1PredicateOperation.NOTEQUALS,
          string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
        },
      ],
    } as V2beta1Filter),
  ), // filter
  undefined, // namespace
];
const LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE = LIST_EXPERIMENT_DEFAULTS.slice(0, 4);

describe('ExperimentList', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let experimentListRef: React.RefObject<ExperimentList> | null = null;

  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;

  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns');
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');

  vi.spyOn(console, 'log').mockImplementation(() => null);

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ExperimentList,
      { pathname: RoutePage.EXPERIMENTS } as any,
      '' as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function getInstance(): ExperimentList {
    if (!experimentListRef?.current) {
      throw new Error('ExperimentList instance not available');
    }
    return experimentListRef.current;
  }

  function mockListNExperiments(n: number = 1, includeDescription: boolean = false) {
    return () =>
      Promise.resolve({
        experiments: range(n).map(i => ({
          experiment_id: 'test-experiment-id' + i,
          display_name: 'test experiment name' + i,
          description: includeDescription ? 'test experiment description' + i : undefined,
        })),
      });
  }

  async function waitForDisplayExperiments(expectedCount: number): Promise<void> {
    await waitFor(() => {
      const displayExperiments = getInstance().state.displayExperiments || [];
      expect(displayExperiments.length).toBe(expectedCount);
    });
  }

  async function renderExperimentList(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ): Promise<PageProps> {
    experimentListRef = React.createRef<ExperimentList>();
    const props = { ...generateProps(), ...propsPatch } as PageProps;
    renderResult = render(
      <MemoryRouter>
        <ExperimentList ref={experimentListRef} {...props} />
      </MemoryRouter>,
    );
    await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalled());
    return props;
  }

  async function mountWithNExperiments(
    n: number,
    nRuns: number,
    { namespace }: { namespace?: string } = {},
  ): Promise<void> {
    listExperimentsSpy.mockImplementation(mockListNExperiments(n));
    listRunsSpy.mockImplementation(() => ({
      runs: range(nRuns).map(i => ({
        run_id: 'test-run-id' + i,
        display_name: 'test run name' + i,
      })),
    }));
    await renderExperimentList({ namespace });
    await waitForDisplayExperiments(n);
  }

  beforeEach(() => {
    vi.clearAllMocks();
    updateBannerSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    listExperimentsSpy.mockResolvedValue({ experiments: [] });
    listRunsSpy.mockResolvedValue({ runs: [] });
    formatDateStringSpy.mockImplementation(() => '1/2/2019, 12:34:56 PM');
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    experimentListRef = null;
  });

  it('renders an empty list with empty state message', async () => {
    await renderExperimentList();
    await waitFor(() =>
      expect(
        screen.getByText('No experiments found. Click "Create experiment" to start.'),
      ).toBeInTheDocument(),
    );
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one experiment', async () => {
    listExperimentsSpy.mockImplementation(mockListNExperiments(1, true));
    listRunsSpy.mockResolvedValue({
      runs: [{ run_id: 'test-run-id0', display_name: 'test run name0' }],
    });
    await renderExperimentList();
    await waitForDisplayExperiments(1);
    await waitFor(() => expect(screen.getAllByTestId('table-row')).toHaveLength(1));
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one experiment with no description', async () => {
    listExperimentsSpy.mockImplementation(mockListNExperiments(1));
    listRunsSpy.mockResolvedValue({
      runs: [{ run_id: 'test-run-id0', display_name: 'test run name0' }],
    });
    await renderExperimentList();
    await waitForDisplayExperiments(1);
    await waitFor(() => expect(screen.getAllByTestId('table-row')).toHaveLength(1));
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one experiment with error', async () => {
    listExperimentsSpy.mockImplementation(mockListNExperiments(1, true));
    TestUtils.makeErrorResponseOnce(listRunsSpy as any, 'bad stuff happened');
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    await renderExperimentList();
    await waitForDisplayExperiments(1);
    await waitFor(() => expect(screen.getAllByTestId('table-row')).toHaveLength(1));
    expect(loggerErrorSpy).toHaveBeenCalled();
    expect(renderResult!.asFragment()).toMatchSnapshot();
    loggerErrorSpy.mockRestore();
  });

  it('calls Apis to list experiments, sorted by creation time in descending order', async () => {
    await mountWithNExperiments(1, 1);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(listRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      'test-experiment-id0',
      undefined,
      5,
      'created_at desc',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.NOTEQUALS,
              string_value: V2beta1RunStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
    );
    expect(getInstance().state.displayExperiments).toEqual([
      {
        expandState: ExpandState.COLLAPSED,
        experiment_id: 'test-experiment-id0',
        last5Runs: [{ run_id: 'test-run-id0', display_name: 'test run name0' }],
        display_name: 'test experiment name0',
        description: undefined,
      },
    ]);
  });

  it('calls Apis to list experiments with namespace when available', async () => {
    await mountWithNExperiments(1, 1, { namespace: 'test-ns' });
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
      'test-ns',
    );
  });

  it('has a Refresh button, clicking it refreshes the experiment list', async () => {
    await mountWithNExperiments(1, 1);
    const refreshBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    expect(listExperimentsSpy.mock.calls.length).toBe(1);
    await act(async () => {
      await refreshBtn!.action();
    });
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('shows error banner when listing experiments fails', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    await renderExperimentList();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'bad stuff happened',
          message:
            'Error: failed to retrieve list of experiments. Click Details for more information.',
          mode: 'error',
        }),
      ),
    );
  });

  it('shows error next to experiment when listing its last 5 runs fails', async () => {
    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [{ display_name: 'exp1' }] }));
    TestUtils.makeErrorResponseOnce(listRunsSpy as any, 'bad stuff happened');
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    await renderExperimentList();
    await waitFor(() =>
      expect(getInstance().state.displayExperiments).toEqual([
        {
          error: 'Failed to load the last 5 runs of this experiment',
          expandState: 0,
          display_name: 'exp1',
        },
      ]),
    );
    expect(loggerErrorSpy).toHaveBeenCalled();
    loggerErrorSpy.mockRestore();
  });

  it('shows error banner when listing experiments fails after refresh', async () => {
    await renderExperimentList();
    const refreshBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    await act(async () => {
      await refreshBtn!.action();
    });
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad stuff happened',
        message:
          'Error: failed to retrieve list of experiments. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('hides error banner when listing experiments fails then succeeds', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    await renderExperimentList();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'bad stuff happened',
          message:
            'Error: failed to retrieve list of experiments. Click Details for more information.',
          mode: 'error',
        }),
      ),
    );
    updateBannerSpy.mockReset();

    const refreshBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.REFRESH];
    listExperimentsSpy.mockImplementationOnce(() => ({
      experiments: [{ display_name: 'experiment1' }],
    }));
    listRunsSpy.mockImplementationOnce(() => ({ runs: [{ display_name: 'run1' }] }));
    await act(async () => {
      await refreshBtn!.action();
    });
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('can expand an experiment to see its runs', async () => {
    await mountWithNExperiments(1, 1);
    const expandButtons = screen.getAllByRole('button', { name: 'Expand' });
    fireEvent.click(expandButtons[0]);
    await waitFor(() => {
      expect(getInstance().state.displayExperiments).toEqual([
        {
          expandState: ExpandState.EXPANDED,
          experiment_id: 'test-experiment-id0',
          last5Runs: [{ run_id: 'test-run-id0', display_name: 'test run name0' }],
          display_name: 'test experiment name0',
          description: undefined,
        },
      ]);
    });
  });

  it('renders a list of runs for given experiment', async () => {
    await renderExperimentList();
    await act(async () => {
      getInstance().setState({
        displayExperiments: [
          {
            display_name: 'experiment1',
            experiment_id: 'experiment1',
            last5Runs: [{ id: 'run1id' }, { id: 'run2id' }],
          },
        ],
      });
    });
    const runListTree = (getInstance() as any)._getExpandedExperimentComponent(0);
    expect(runListTree.props.experimentIdMask).toEqual('experiment1');
  });

  it('navigates to new experiment page when Create experiment button is clicked', async () => {
    await renderExperimentList();
    const createBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.NEW_EXPERIMENT];
    await act(async () => {
      await createBtn!.action();
    });
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_EXPERIMENT);
  });

  it('always has new experiment button enabled', async () => {
    await mountWithNExperiments(1, 1);
    const toolbarCall = updateToolbarSpy.mock.calls[0][0];
    expect(toolbarCall.actions[ButtonKeys.NEW_EXPERIMENT]).not.toHaveProperty('disabled');
  });

  it('enables clone button when one run is selected', async () => {
    await mountWithNExperiments(1, 1);
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1']);
    });
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    expect(updateToolbarSpy.mock.calls[0][0].actions[ButtonKeys.CLONE_RUN]).toHaveProperty(
      'disabled',
      true,
    );
    expect(updateToolbarSpy.mock.calls[1][0].actions[ButtonKeys.CLONE_RUN]).toHaveProperty(
      'disabled',
      false,
    );
  });

  it('disables clone button when more than one run is selected', async () => {
    await mountWithNExperiments(1, 1);
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1', 'run2']);
    });
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    expect(updateToolbarSpy.mock.calls[0][0].actions[ButtonKeys.CLONE_RUN]).toHaveProperty(
      'disabled',
      true,
    );
    expect(updateToolbarSpy.mock.calls[1][0].actions[ButtonKeys.CLONE_RUN]).toHaveProperty(
      'disabled',
      true,
    );
  });

  it('enables compare runs button only when more than one is selected', async () => {
    await mountWithNExperiments(1, 1);
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1']);
    });
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1', 'run2']);
    });
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1', 'run2', 'run3']);
    });
    expect(updateToolbarSpy).toHaveBeenCalledTimes(4);
    expect(updateToolbarSpy.mock.calls[0][0].actions[ButtonKeys.COMPARE]).toHaveProperty(
      'disabled',
      true,
    );
    expect(updateToolbarSpy.mock.calls[1][0].actions[ButtonKeys.COMPARE]).toHaveProperty(
      'disabled',
      false,
    );
    expect(updateToolbarSpy.mock.calls[2][0].actions[ButtonKeys.COMPARE]).toHaveProperty(
      'disabled',
      false,
    );
  });

  it('navigates to compare page with the selected run ids', async () => {
    await mountWithNExperiments(1, 1);
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1', 'run2', 'run3']);
    });
    const compareBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.COMPARE];
    await act(async () => {
      await compareBtn!.action();
    });
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.COMPARE}?${QUERY_PARAMS.runlist}=run1,run2,run3`,
    );
  });

  it('navigates to new run page with the selected run id for cloning', async () => {
    await mountWithNExperiments(1, 1);
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1']);
    });
    const cloneBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.CLONE_RUN];
    await act(async () => {
      await cloneBtn!.action();
    });
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.NEW_RUN}?${QUERY_PARAMS.cloneFromRun}=run1`,
    );
  });

  it('enables archive button when at least one run is selected', async () => {
    await mountWithNExperiments(1, 1);
    expect(TestUtils.getToolbarButton(updateToolbarSpy as any, ButtonKeys.ARCHIVE).disabled).toBe(
      true,
    );
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1']);
    });
    expect(TestUtils.getToolbarButton(updateToolbarSpy as any, ButtonKeys.ARCHIVE).disabled).toBe(
      false,
    );
    await act(async () => {
      (getInstance() as any)._selectionChanged(['run1', 'run2']);
    });
    expect(TestUtils.getToolbarButton(updateToolbarSpy as any, ButtonKeys.ARCHIVE).disabled).toBe(
      false,
    );
    await act(async () => {
      (getInstance() as any)._selectionChanged([]);
    });
    expect(TestUtils.getToolbarButton(updateToolbarSpy as any, ButtonKeys.ARCHIVE).disabled).toBe(
      true,
    );
  });

  it('renders experiment names as links to their details pages', async () => {
    await renderExperimentList();
    const nameRenderer = getInstance()._nameCustomRenderer({
      id: 'experiment-id',
      value: 'experiment name',
    } as any);
    const { getByTestId, asFragment, unmount } = render(
      <MemoryRouter>{nameRenderer}</MemoryRouter>,
    );
    const link = getByTestId('experiment-name-link');
    expect(link).toHaveAttribute('data-experiment-id', 'experiment-id');
    expect(link).toHaveAttribute('data-experiment-name', 'experiment name');
    expect(link.getAttribute('href')).toBe(
      RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, 'experiment-id'),
    );
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('renders last 5 runs statuses', async () => {
    await renderExperimentList();
    const statusRenderer = getInstance()._last5RunsCustomRenderer({
      experiment_id: 'experiment-id',
      value: [
        { state: V2beta1RuntimeState.SUCCEEDED },
        { state: V2beta1RuntimeState.PENDING },
        { state: V2beta1RuntimeState.FAILED },
        { state: V2beta1RuntimeState.RUNTIMESTATEUNSPECIFIED },
        { state: V2beta1RuntimeState.SUCCEEDED },
      ],
    } as any);
    const { asFragment, unmount } = render(<div>{statusRenderer}</div>);
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  describe('EnhancedExperimentList', () => {
    it('defaults to no namespace', async () => {
      render(
        <MemoryRouter>
          <EnhancedExperimentList {...generateProps()} />
        </MemoryRouter>,
      );
      await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalled());
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    });

    it('gets namespace from context', async () => {
      render(
        <MemoryRouter>
          <NamespaceContext.Provider value='test-ns'>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );
      await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalled());
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'test-ns',
      );
    });

    it('auto refreshes list when namespace changes', async () => {
      const { rerender } = render(
        <MemoryRouter>
          <NamespaceContext.Provider value='test-ns-1'>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );
      await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalledTimes(1));
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'test-ns-1',
      );
      rerender(
        <MemoryRouter>
          <NamespaceContext.Provider value='test-ns-2'>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );
      await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalledTimes(2));
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'test-ns-2',
      );
    });

    it("doesn't keep error message for request from previous namespace", async () => {
      listExperimentsSpy.mockImplementation(() => Promise.reject('namespace cannot be empty'));
      const { rerender } = render(
        <MemoryRouter>
          <NamespaceContext.Provider value={undefined}>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );

      listExperimentsSpy.mockImplementation(mockListNExperiments());
      rerender(
        <MemoryRouter>
          <NamespaceContext.Provider value='test-ns'>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );
      await act(TestUtils.flushPromises);
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        {}, // Empty object means banner has no error message
      );
    });
  });
});
