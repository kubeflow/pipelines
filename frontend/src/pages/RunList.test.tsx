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
import * as Utils from 'src/lib/Utils';
import { act, render, screen, waitFor } from '@testing-library/react';
import RunList, { RunListProps } from './RunList';
import TestUtils from 'src/TestUtils';
import produce from 'immer';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import { V2beta1Run, V2beta1RunStorageState, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Apis, RunSortKeys, ListRequest } from 'src/lib/Apis';
import { range } from 'lodash';
import { CommonTestWrapper } from 'src/TestWrapper';
import { vi } from 'vitest';

class RunListTest extends RunList {
  public _loadRuns(request: ListRequest): Promise<string> {
    return super._loadRuns(request);
  }
}

describe('RunList', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let runListRef: React.RefObject<RunListTest> | null = null;

  let onErrorSpy: ReturnType<typeof vi.fn>;
  let listRunsSpy: ReturnType<typeof vi.spyOn>;
  let getRunSpy: ReturnType<typeof vi.spyOn>;
  let getPipelineVersionSpy: ReturnType<typeof vi.spyOn>;
  let listExperimentsSpy: ReturnType<typeof vi.spyOn>;
  let formatDateStringSpy: ReturnType<typeof vi.spyOn>;

  function generateProps(): RunListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function createRunListInstance(props?: RunListProps): RunListTest {
    return new RunListTest(props || generateProps());
  }

  function getRunListInstance(): RunListTest {
    if (!runListRef?.current) {
      throw new Error('RunList instance is not available');
    }
    return runListRef.current;
  }

  function getRunListState(): RunList['state'] | undefined {
    return runListRef?.current?.state;
  }

  async function renderRunList(props?: RunListProps): Promise<void> {
    runListRef = React.createRef<RunListTest>();
    renderResult = render(
      <CommonTestWrapper>
        <RunListTest ref={runListRef} {...(props || generateProps())} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
  }

  async function waitForRunListLoad(): Promise<void> {
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
    });
    await TestUtils.flushPromises();
  }

  async function waitForRunMaskLoad(expectedCount?: number): Promise<void> {
    await waitFor(() => {
      if (expectedCount !== undefined) {
        expect(getRunSpy).toHaveBeenCalledTimes(expectedCount);
      } else {
        expect(getRunSpy).toHaveBeenCalled();
      }
    });
    await TestUtils.flushPromises();
  }

  async function callLoadRuns(request: ListRequest): Promise<void> {
    await act(async () => {
      await getRunListInstance()._loadRuns(request);
    });
    await TestUtils.flushPromises();
  }

  function mockNRuns(n: number, runTemplate: Partial<V2beta1Run>): void {
    getRunSpy.mockImplementation(id => {
      const pipelineVersionRef = {
        pipeline_id: 'testpipeline' + id,
        pipeline_version_id: 'testversion' + id,
      };
      return Promise.resolve(
        produce(runTemplate, draft => {
          draft = draft || {};
          draft.run_id = id;
          draft.display_name = 'run with id: ' + id;
          draft.pipeline_version_reference = pipelineVersionRef;
          draft.state = draft.state || V2beta1RuntimeState.SUCCEEDED;
        }),
      );
    });

    listRunsSpy.mockImplementation(() =>
      Promise.resolve({
        runs: range(1, n + 1).map(i => {
          if (runTemplate) {
            const pipelineVersionRef = {
              pipeline_id: 'testpipeline' + i,
              pipeline_version_id: 'testversion' + i,
            };
            return produce(runTemplate as Partial<V2beta1Run>, draft => {
              draft.run_id = 'testrun' + i;
              draft.display_name = 'run with id: testrun' + i;
              draft.pipeline_version_reference = pipelineVersionRef;
              draft.state = draft.state || V2beta1RuntimeState.SUCCEEDED;
            });
          }
          return {
            run_id: 'testrun' + i,
            display_name: 'run with id: testrun' + i,
            pipeline_version_reference: {
              pipeline_id: 'testpipeline' + i,
              pipeline_version_id: 'testversion' + i,
            },
            state: V2beta1RuntimeState.SUCCEEDED,
          } as V2beta1Run;
        }),
      }),
    );

    getPipelineVersionSpy.mockImplementation(() => ({
      display_name: 'some pipeline version',
      pipeline_id: 'pipeline-id',
      pipeline_version_id: 'pipeline-version-id',
    }));
    listExperimentsSpy.mockImplementation(() => ({ experiments: [] }));
  }

  function renderRenderer(element: React.ReactElement) {
    const { asFragment } = render(<CommonTestWrapper>{element}</CommonTestWrapper>);
    return asFragment();
  }

  beforeEach(() => {
    onErrorSpy = vi.fn();
    listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns').mockResolvedValue({ runs: [] });
    getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun').mockResolvedValue({} as V2beta1Run);
    getPipelineVersionSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion')
      .mockResolvedValue({
        display_name: 'some pipeline version',
        pipeline_id: 'pipeline-id',
        pipeline_version_id: 'pipeline-version-id',
      } as any);
    listExperimentsSpy = vi
      .spyOn(Apis.experimentServiceApiV2, 'listExperiments')
      .mockResolvedValue({ experiments: [] } as any);
    formatDateStringSpy = vi.spyOn(Utils, 'formatDateString').mockImplementation(date => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
  });

  afterEach(() => {
    if (renderResult) {
      renderResult.unmount();
      renderResult = null;
    }
    runListRef = null;
    vi.resetAllMocks();
  });

  it('renders the empty experience', async () => {
    await renderRunList();
    await waitForRunListLoad();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  describe('in archived state', () => {
    it('renders the empty experience', async () => {
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      await renderRunList(props);
      await waitForRunListLoad();
      expect(renderResult!.asFragment()).toMatchSnapshot();
    });

    it('loads runs whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.AVAILABLE;
      await renderRunList(props);
      await waitForRunListLoad();
      listRunsSpy.mockClear();

      await callLoadRuns({});

      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
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
    });

    it('loads runs whose storage state is ARCHIVED when storage state equals ARCHIVED', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      await renderRunList(props);
      await waitForRunListLoad();
      listRunsSpy.mockClear();

      await callLoadRuns({});

      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'storage_state',
                operation: V2beta1PredicateOperation.EQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ],
          } as V2beta1Filter),
        ),
      );
    });

    it('augments request filter with storage state predicates', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      await renderRunList(props);
      await waitForRunListLoad();
      listRunsSpy.mockClear();

      await callLoadRuns({
        filter: encodeURIComponent(
          JSON.stringify({
            predicates: [{ key: 'k', op: 'op', string_value: 'val' }],
          }),
        ),
      });
      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'k',
                op: 'op',
                string_value: 'val',
              },
              {
                key: 'storage_state',
                operation: V2beta1PredicateOperation.EQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ],
          } as V2beta1Filter),
        ),
      );
    });
  });

  it('loads one run', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    await renderRunList(props);
    await waitForRunListLoad();

    expect(screen.getByText('run with id: testrun1')).toBeInTheDocument();
    expect(screen.queryByText('run with id: testrun2')).toBeNull();
  });

  it('reloads the run when refresh is called', async () => {
    mockNRuns(0, {});
    const props = generateProps();
    await renderRunList(props);
    await waitForRunListLoad();

    await act(async () => {
      await getRunListInstance().refresh();
    });
    await waitFor(() => {
      expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(2);
    });

    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      '',
      10,
      RunSortKeys.CREATED_AT + ' desc',
      '',
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('loads multiple runs', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    await renderRunList(props);
    await waitForRunListLoad();

    expect(screen.getByText('run with id: testrun1')).toBeInTheDocument();
    expect(screen.getByText('run with id: testrun2')).toBeInTheDocument();
    expect(screen.getByText('run with id: testrun3')).toBeInTheDocument();
    expect(screen.getByText('run with id: testrun4')).toBeInTheDocument();
    expect(screen.getByText('run with id: testrun5')).toBeInTheDocument();
  });

  it('calls error callback when loading runs fails', async () => {
    TestUtils.makeErrorResponseOnce(listRunsSpy as any, 'bad stuff happened');
    const props = generateProps();
    await renderRunList(props);
    await waitFor(() => {
      expect(props.onError).toHaveBeenLastCalledWith(
        'Error: failed to fetch runs.',
        new Error('bad stuff happened'),
      );
    });
  });

  it('displays error in run row if experiment could not be fetched', async () => {
    mockNRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    const props = generateProps();

    await renderRunList(props);
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(getRunListState()?.runs[0].error).toEqual(
        'Failed to get associated experiment: bad stuff happened',
      );
    });
  });

  it('displays error in run row if it failed to parse (run list mask)', async () => {
    TestUtils.makeErrorResponseOnce(getRunSpy as any, 'bad stuff happened');
    const props = generateProps();
    props.runIdListMask = ['testrun1'];
    await renderRunList(props);
    await waitFor(() => {
      // won't call listRuns if specific run id is provided
      expect(listRunsSpy).toHaveBeenCalledTimes(0);
      expect(getRunSpy).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(getRunListState()?.runs[0].error).toEqual('bad stuff happened');
    });
  });

  it('shows run time for each run', async () => {
    mockNRuns(1, {
      created_at: new Date(2018, 10, 10, 10, 10, 10),
      finished_at: new Date(2018, 10, 10, 11, 11, 11),
      state: V2beta1RuntimeState.SUCCEEDED,
    });
    const props = generateProps();
    await renderRunList(props);
    await waitForRunListLoad();

    await screen.findByText('1:01:01');
  });

  it('loads runs for a given experiment id', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    await renderRunList(props);
    await waitForRunListLoad();
    listRunsSpy.mockClear();

    await callLoadRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      'experiment1',
      undefined,
      undefined,
      undefined,
      undefined,
    );
  });

  it('loads runs for a given namespace', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.namespaceMask = 'namespace1';
    await renderRunList(props);
    await waitForRunListLoad();
    listRunsSpy.mockClear();

    await callLoadRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      'namespace1',
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
  });

  it('loads given list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    await renderRunList(props);
    await waitForRunMaskLoad(2);
    getRunSpy.mockClear();
    listRunsSpy.mockClear();

    await callLoadRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run1');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run2');
  });

  it('loads given and filtered list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['filterRun1', 'filterRun2', 'notincluded'];
    await renderRunList(props);
    await waitForRunMaskLoad(3);
    getRunSpy.mockClear();
    listRunsSpy.mockClear();

    await callLoadRuns({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'name',
              operation: V2beta1PredicateOperation.ISSUBSTRING,
              string_value: 'filterRun',
            },
          ],
        } as V2beta1Filter),
      ),
    });

    await waitFor(() => {
      expect(getRunListState()?.runs).toMatchObject([
        {
          run: { display_name: 'run with id: filterRun1', run_id: 'filterRun1' },
        },
        {
          run: { display_name: 'run with id: filterRun2', run_id: 'filterRun2' },
        },
      ]);
    });
  });

  it('loads given and filtered list of runs only through multiple filters', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['filterRun1', 'filterRun2', 'notincluded1'];
    await renderRunList(props);
    await waitForRunMaskLoad(3);
    getRunSpy.mockClear();
    listRunsSpy.mockClear();

    await callLoadRuns({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'name',
              operation: V2beta1PredicateOperation.ISSUBSTRING,
              string_value: 'filterRun',
            },
            { key: 'name', operation: V2beta1PredicateOperation.ISSUBSTRING, string_value: '1' },
          ],
        } as V2beta1Filter),
      ),
    });

    await waitFor(() => {
      expect(getRunListState()?.runs).toMatchObject([
        {
          run: { display_name: 'run with id: filterRun1', run_id: 'filterRun1' },
        },
      ]);
    });
  });

  it('shows pipeline version name', async () => {
    mockNRuns(1, {
      pipeline_version_reference: {
        pipeline_id: 'testpipeline1',
        pipeline_version_id: 'testversion1',
      },
    });
    const props = generateProps();
    await renderRunList(props);

    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(getPipelineVersionSpy).toHaveBeenCalled();
    });

    await screen.findByText('some pipeline version');
  });

  // TODO(jlyaoyuli): add back this test (show recurring run config)
  // after the recurring run v2 API integration

  it('shows experiment name', async () => {
    mockNRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockImplementationOnce(() => ({
      experiments: [
        {
          experiment_id: 'test-experiment-id',
          display_name: 'test experiment',
        },
      ],
    }));
    const props = generateProps();
    await renderRunList(props);
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalled();
    });

    expect(screen.getByText('test experiment')).toBeInTheDocument();
  });

  it('hides experiment name if instructed', async () => {
    mockNRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [] }));
    const props = generateProps();
    props.hideExperimentColumn = true;
    await renderRunList(props);
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalled();
    });

    expect(screen.queryByText('test experiment')).toBeNull();
  });

  it('renders run name as link to its details page', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._nameCustomRenderer({ value: 'test run', id: 'run-id' }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', pipelineId: 'pipeline-id', usePlaceholder: false },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('handles no pipeline id given', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', usePlaceholder: false },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('shows "View pipeline" button if pipeline is embedded in run', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', pipelineId: 'pipeline-id', usePlaceholder: true },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('handles no pipeline name', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { /* no displayName */ usePlaceholder: true },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._recurringRunCustomRenderer({
        id: 'run-id',
        value: { id: 'recurring-run-id' },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders experiment name as link to its details page', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._experimentCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test experiment', id: 'experiment-id' },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders no experiment name', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._experimentCustomRenderer({
        id: 'run-id',
        value: { /* no displayName */ id: 'experiment-id' },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders status as icon', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._statusCustomRenderer({
        value: V2beta1RuntimeState.SUCCEEDED,
        id: 'run-id',
      }),
    );
    expect(fragment).toMatchSnapshot();
  });

  it('renders pipeline version name as link to its details page', () => {
    const instance = createRunListInstance();
    const fragment = renderRenderer(
      instance._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: {
          displayName: 'test pipeline version',
          pipelineId: 'pipeline-id',
          usePlaceholder: false,
          versionId: 'version-id',
        },
      }),
    );
    expect(fragment).toMatchSnapshot();
  });
});
