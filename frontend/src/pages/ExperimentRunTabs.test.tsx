/*
 * Copyright 2020 The Kubeflow Authors
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

import { render, screen, waitFor, within } from '@testing-library/react';
vi.mock('src/components/shell/ThemeProvider', async (importOriginal) => ({
  ...(await importOriginal<typeof import('src/components/shell/ThemeProvider')>()),
  useTheme: () => ({ themeClassName: 'kfp-theme' }),
}));

import { produce } from 'immer';
import ExperimentRunTabs, { ExperimentRunTabsProps } from './ExperimentRunTabs';
import { RouteParams } from 'src/components/Router';
import { V2Run, V2RunStorageState } from 'src/apisv2/run';
import { V2Experiment } from 'src/apisv2/experiment';
import { Apis } from 'src/lib/Apis';
import * as Utils from 'src/lib/Utils';
import { BrowserRouter } from 'react-router';
import { V2PredicateOperation } from 'src/apisv2/filter';

describe('RunListsRouter', () => {
  let navigateSpy: any;
  let runStorageState = V2RunStorageState.AVAILABLE;

  const onSelectionChangeMock = vi.fn();
  const listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns');
  const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
  const getPipelineSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline');
  const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');
  const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => null);

  const MOCK_EXPERIMENT = newMockExperiment();
  const archiveRunDisplayName = 'run with id: achiverunid';
  const activeRunDisplayName = 'run with id: activerunid';

  function newMockExperiment(): V2Experiment {
    return {
      description: 'mock experiment description',
      experiment_id: 'some-mock-experiment-id',
      display_name: 'some mock experiment name',
    };
  }

  function generateProps(): ExperimentRunTabsProps {
    const runListsRouterProps: ExperimentRunTabsProps = {
      onTabSwitch: vi.fn((newTab: number) => {
        // this.refresh();
        if (newTab === 1) {
          runStorageState = V2RunStorageState.ARCHIVED;
        } else {
          runStorageState = V2RunStorageState.AVAILABLE;
        }
      }),
      hideExperimentColumn: true,
      navigate: navigateSpy,
      location: '' as any,
      params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.experiment_id } as any,
      onSelectionChange: onSelectionChangeMock,
      selectedIds: [],
      storageState: runStorageState,
      refreshCount: 0,
      noFilterBox: false,
      disablePaging: false,
      disableSorting: true,
      disableSelection: false,
      onError: consoleErrorSpy,
    };
    return runListsRouterProps;
  }

  beforeEach(() => {
    getRunSpy.mockImplementation((id) =>
      Promise.resolve(
        produce({} as Partial<V2Run>, (draft) => {
          draft = draft || {};
          draft.run_id = id;
          draft.display_name = 'run with id: ' + id;
        }),
      ),
    );
    listRunsSpy.mockImplementation((pageToken, pageSize, sortBy, keyType, keyId, filter) => {
      let filterForArchive = JSON.parse(decodeURIComponent('{"predicates": []}'));
      filterForArchive = encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2PredicateOperation.EQUALS,
              string_value: V2RunStorageState.ARCHIVED.toString(),
            },
          ],
        }),
      );
      if (filter === filterForArchive) {
        return Promise.resolve({
          runs: [
            {
              run_id: 'achiverunid',
              display_name: archiveRunDisplayName,
            },
          ],
        });
      }
      return Promise.resolve({
        runs: [
          {
            run_id: 'activerunid',
            display_name: activeRunDisplayName,
          },
        ],
      });
    });
    getPipelineSpy.mockImplementation(() => ({ name: 'some pipeline' }));
    getExperimentSpy.mockImplementation(() => ({ name: 'some experiment' }));
    listExperimentsSpy.mockResolvedValue({ experiments: [] });
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('shows the inline Active and Archived scope without a tab row', async () => {
    render(
      <BrowserRouter>
        <ExperimentRunTabs {...generateProps()} />
      </BrowserRouter>,
    );

    const scope = within(screen.getByRole('group', { name: 'Experiment run scope' }));
    expect(scope.getByRole('button', { name: 'Active', exact: true })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(scope.getByRole('button', { name: 'Archived', exact: true })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
    await waitFor(() => screen.getByText(activeRunDisplayName));
  });
});
