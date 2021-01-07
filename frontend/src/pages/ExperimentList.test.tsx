/*
 * Copyright 2018 Google LLC
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
import * as Utils from '../lib/Utils';
import EnhancedExperimentList, { ExperimentList } from './ExperimentList';
import TestUtils, { defaultToolbarProps } from '../TestUtils';
import { ApiFilter, PredicateOp } from '../apis/filter';
import { RunStorageState } from '../apis/run';
import { Apis } from '../lib/Apis';
import { ExpandState } from '../components/CustomTable';
import { NodePhase } from '../lib/StatusUtils';
import { PageProps } from './Page';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { range } from 'lodash';
import { ButtonKeys } from '../lib/Buttons';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { render, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ExperimentStorageState } from '../apis/experiment';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: ((key: string) => key) as any };
    return Component;
  },

  useTranslation: () => {
    return {
      t: (key: string) => key as any,
    };
  },
}));
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
          op: PredicateOp.NOTEQUALS,
          string_value: ExperimentStorageState.ARCHIVED.toString(),
        },
      ],
    } as ApiFilter),
  ), // filter
  undefined, // resource_reference_key_type
  undefined, // resource_reference_key_id
];
const LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE = LIST_EXPERIMENT_DEFAULTS.slice(0, 4);

describe('ExperimentList', () => {
  let tree: ShallowWrapper | ReactWrapper;
  let t: TFunction = (key: string) => key;
  jest.spyOn(console, 'log').mockImplementation(() => null);

  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
  const listRunsSpy = jest.spyOn(Apis.runServiceApi, 'listRuns');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  jest.spyOn(Utils, 'formatDateString').mockImplementation(() => '1/2/2019, 12:34:56 PM');

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
      { t },
      t,
    );
  }

  function mockListNExpperiments(n: number = 1) {
    return () =>
      Promise.resolve({
        experiments: range(n).map(i => ({
          id: 'test-experiment-id' + i,
          name: 'test experiment name' + i,
        })),
      });
  }

  async function mountWithNExperiments(
    n: number,
    nRuns: number,
    { namespace }: { namespace?: string } = {},
  ): Promise<void> {
    listExperimentsSpy.mockImplementation(mockListNExpperiments(n));
    listRunsSpy.mockImplementation(() => ({
      runs: range(nRuns).map(i => ({ id: 'test-run-id' + i, name: 'test run name' + i })),
    }));
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} namespace={namespace} />);
    await listExperimentsSpy;
    await listRunsSpy;
    await TestUtils.flushPromises();
    tree.update(); // Make sure the tree is updated before returning it
  }

  afterEach(() => {
    jest.resetAllMocks();
    jest.clearAllMocks();
    if (tree.exists()) {
      tree.unmount();
    }
  });

  it('renders an empty list with empty state message', () => {
    tree = shallow(<ExperimentList {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one experiment', async () => {
    tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      displayExperiments: [
        {
          description: 'test experiment description',
          expandState: ExpandState.COLLAPSED,
          name: 'test experiment name',
        },
      ],
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one experiment with no description', async () => {
    tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      experiments: [
        {
          expandState: ExpandState.COLLAPSED,
          name: 'test experiment name',
        },
      ],
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one experiment with error', async () => {
    tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      experiments: [
        {
          description: 'test experiment description',
          error: 'oops! could not load experiment',
          expandState: ExpandState.COLLAPSED,
          name: 'test experiment name',
        },
      ],
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('calls Apis to list experiments, sorted by creation time in descending order', async () => {
    await mountWithNExperiments(1, 1);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(listRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      5,
      'created_at desc',
      'EXPERIMENT',
      'test-experiment-id0',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              op: PredicateOp.NOTEQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
    );
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.COLLAPSED,
        id: 'test-experiment-id0',
        last5Runs: [{ id: 'test-run-id0', name: 'test run name0' }],
        name: 'test experiment name0',
      },
    ]);
  });

  it('calls Apis to list experiments with namespace when available', async () => {
    await mountWithNExperiments(1, 1, { namespace: 'test-ns' });
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
      'NAMESPACE',
      'test-ns',
    );
  });

  it('has a Refresh button, clicking it refreshes the experiment list', async () => {
    await mountWithNExperiments(1, 1);
    const instance = tree.instance() as ExperimentList;
    expect(listExperimentsSpy.mock.calls.length).toBe(1);
    const refreshBtn = instance.getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({ t });
  });

  it('shows error banner when listing experiments fails', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad stuff happened',
        message: 'experimentListError common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('shows error next to experiment when listing its last 5 runs fails', async () => {
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [{ name: 'exp1' }] }));
    TestUtils.makeErrorResponseOnce(listRunsSpy, 'bad stuff happened');
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        error: 'last5RunsFailed',
        expandState: 0,
        name: 'exp1',
      },
    ]);
  });

  it('shows error banner when listing experiments fails after refresh', async () => {
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const instance = tree.instance() as ExperimentList;
    const refreshBtn = instance.getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad stuff happened',
        message: 'experimentListError common:clickDetails',
        mode: 'error',
      }),
    );
  });

  it('hides error banner when listing experiments fails then succeeds', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const instance = tree.instance() as ExperimentList;
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'bad stuff happened',
        message: 'experimentListError common:clickDetails',
        mode: 'error',
      }),
    );
    updateBannerSpy.mockReset();

    const refreshBtn = instance.getInitialToolbarState().actions[ButtonKeys.REFRESH];
    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [{ name: 'experiment1' }] }));
    listRunsSpy.mockImplementationOnce(() => ({ runs: [{ name: 'run1' }] }));
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({ t });
  });

  it('can expand an experiment to see its runs', async () => {
    await mountWithNExperiments(1, 1);
    tree
      .find('.tableRow button')
      .at(0)
      .simulate('click');
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.EXPANDED,
        id: 'test-experiment-id0',
        last5Runs: [{ id: 'test-run-id0', name: 'test run name0' }],
        name: 'test experiment name0',
      },
    ]);
  });

  it('renders a list of runs for given experiment', async () => {
    tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      displayExperiments: [{ id: 'experiment1', last5Runs: [{ id: 'run1id' }, { id: 'run2id' }] }],
    });
    const runListTree = (tree.instance() as any)._getExpandedExperimentComponent(0);
    expect(runListTree.props.experimentIdMask).toEqual('experiment1');
  });

  it('navigates to new experiment page when Create experiment button is clicked', async () => {
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const createBtn = (tree.instance() as ExperimentList).getInitialToolbarState().actions[
      ButtonKeys.NEW_EXPERIMENT
    ];
    await createBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_EXPERIMENT);
  });

  it('always has new experiment button enabled', async () => {
    await mountWithNExperiments(1, 1);
    const calls = updateToolbarSpy.mock.calls[0];
    expect(calls[0].actions[ButtonKeys.NEW_EXPERIMENT]).not.toHaveProperty('disabled');
  });

  it('enables clone button when one run is selected', async () => {
    await mountWithNExperiments(1, 1);
    (tree.instance() as any)._selectionChanged(['run1']);
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
    (tree.instance() as any)._selectionChanged(['run1', 'run2']);
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
    (tree.instance() as any)._selectionChanged(['run1']);
    (tree.instance() as any)._selectionChanged(['run1', 'run2']);
    (tree.instance() as any)._selectionChanged(['run1', 'run2', 'run3']);
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
    (tree.instance() as any)._selectionChanged(['run1', 'run2', 'run3']);
    const compareBtn = (tree.instance() as ExperimentList).getInitialToolbarState().actions[
      ButtonKeys.COMPARE
    ];
    await compareBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.COMPARE}?${QUERY_PARAMS.runlist}=run1,run2,run3`,
    );
  });

  it('navigates to new run page with the selected run id for cloning', async () => {
    await mountWithNExperiments(1, 1);
    (tree.instance() as any)._selectionChanged(['run1']);
    const cloneBtn = (tree.instance() as ExperimentList).getInitialToolbarState().actions[
      ButtonKeys.CLONE_RUN
    ];
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.NEW_RUN}?${QUERY_PARAMS.cloneFromRun}=run1`,
    );
  });

  it('enables archive button when at least one run is selected', async () => {
    await mountWithNExperiments(1, 1);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE).disabled).toBeTruthy();
    (tree.instance() as any)._selectionChanged(['run1']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE).disabled).toBeFalsy();
    (tree.instance() as any)._selectionChanged(['run1', 'run2']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE).disabled).toBeFalsy();
    (tree.instance() as any)._selectionChanged([]);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ARCHIVE).disabled).toBeTruthy();
  });

  it('renders experiment names as links to their details pages', async () => {
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    expect(
      (tree.instance() as ExperimentList)._nameCustomRenderer({
        id: 'experiment-id',
        value: 'experiment name',
      }),
    ).toMatchSnapshot();
  });

  it('renders last 5 runs statuses', async () => {
    tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    expect(
      (tree.instance() as ExperimentList)._last5RunsCustomRenderer({
        id: 'experiment-id',
        value: [
          { status: NodePhase.SUCCEEDED },
          { status: NodePhase.PENDING },
          { status: NodePhase.FAILED },
          { status: NodePhase.UNKNOWN },
          { status: NodePhase.SUCCEEDED },
        ],
      }),
    ).toMatchSnapshot();
  });

  describe('EnhancedExperimentList', () => {
    it('defaults to no namespace', () => {
      render(<EnhancedExperimentList {...generateProps()} />);
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
    });

    it('gets namespace from context', () => {
      render(
        <NamespaceContext.Provider value='test-ns'>
          <EnhancedExperimentList {...generateProps()} />
        </NamespaceContext.Provider>,
      );
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'NAMESPACE',
        'test-ns',
      );
    });

    it('auto refreshes list when namespace changes', () => {
      const { rerender } = render(
        <NamespaceContext.Provider value='test-ns-1'>
          <EnhancedExperimentList {...generateProps()} />
        </NamespaceContext.Provider>,
      );
      expect(listExperimentsSpy).toHaveBeenCalledTimes(1);
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'NAMESPACE',
        'test-ns-1',
      );
      rerender(
        <NamespaceContext.Provider value='test-ns-2'>
          <EnhancedExperimentList {...generateProps()} />
        </NamespaceContext.Provider>,
      );
      expect(listExperimentsSpy).toHaveBeenCalledTimes(2);
      expect(listExperimentsSpy).toHaveBeenLastCalledWith(
        ...LIST_EXPERIMENT_DEFAULTS_WITHOUT_RESOURCE_REFERENCE,
        'NAMESPACE',
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

      listExperimentsSpy.mockImplementation(mockListNExpperiments());
      rerender(
        <MemoryRouter>
          <NamespaceContext.Provider value={'test-ns'}>
            <EnhancedExperimentList {...generateProps()} />
          </NamespaceContext.Provider>
        </MemoryRouter>,
      );
      await act(TestUtils.flushPromises);
      expect(updateBannerSpy.mock.calls.length).toBe(4);
      expect(updateBannerSpy.mock.calls[0]).toEqual(expect.objectContaining({}));
      expect(updateBannerSpy.mock.calls[1]).toEqual(expect.objectContaining({}));
      expect(updateBannerSpy.mock.calls[3]).toEqual(expect.objectContaining({}));
      expect(updateBannerSpy.mock.calls[4]).toEqual(expect.objectContaining({}));
    });
  });
});
