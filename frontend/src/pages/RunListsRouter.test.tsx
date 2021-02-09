/*
 * Copyright 2020 Google LLC
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

import { render, screen } from '@testing-library/react';
import produce from 'immer';
import RunListsRouter, { RunListsRouterProps } from './RunListsRouter';
import React from 'react';
import { RouteParams } from 'src/components/Router';
import { ApiRunDetail, RunStorageState } from 'src/apis/run';
import { ApiExperiment } from 'src/apis/experiment';
import { Apis } from 'src/lib/Apis';
import * as Utils from '../lib/Utils';
import { BrowserRouter } from 'react-router-dom';
import { PredicateOp } from 'src/apis/filter';

describe('RunListsRouter', () => {
  let historyPushSpy: any;
  let runStorageState = RunStorageState.AVAILABLE;

  const onSelectionChangeMock = jest.fn();
  const listRunsSpy = jest.spyOn(Apis.runServiceApi, 'listRuns');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');
  const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => null);

  const MOCK_EXPERIMENT = newMockExperiment();
  const archiveRunDisplayName = 'run with id: achiverunid';
  const activeRunDisplayName = 'run with id: activerunid';

  function newMockExperiment(): ApiExperiment {
    return {
      description: 'mock experiment description',
      id: 'some-mock-experiment-id',
      name: 'some mock experiment name',
    };
  }

  function generateProps(): RunListsRouterProps {
    const runListsRouterProps: RunListsRouterProps = {
      onTabSwitch: jest.fn((newTab: number) => {
        // this.refresh();
        if (newTab === 1) {
          runStorageState = RunStorageState.ARCHIVED;
        } else {
          runStorageState = RunStorageState.AVAILABLE;
        }
      }),
      hideExperimentColumn: true,
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.id } } as any,
      onSelectionChange: onSelectionChangeMock,
      selectedIds: [],
      storageState: runStorageState,
      refreshCount: 0,
      noFilterBox: false,
      disablePaging: false,
      disableSorting: true,
      disableSelection: false,
      hideMetricMetadata: false,
      onError: consoleErrorSpy,
    };
    return runListsRouterProps;
  }

  beforeEach(() => {
    getRunSpy.mockImplementation(id =>
      Promise.resolve(
        produce({} as Partial<ApiRunDetail>, draft => {
          draft.run = draft.run || {};
          draft.run.id = id;
          draft.run.name = 'run with id: ' + id;
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
              op: PredicateOp.EQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            },
          ],
        }),
      );
      if (filter === filterForArchive) {
        return Promise.resolve({
          runs: [
            {
              id: 'achiverunid',
              name: archiveRunDisplayName,
            },
          ],
        });
      }
      return Promise.resolve({
        runs: [
          {
            id: 'activerunid',
            name: activeRunDisplayName,
          },
        ],
      });
    });
    getPipelineSpy.mockImplementation(() => ({ name: 'some pipeline' }));
    getExperimentSpy.mockImplementation(() => ({ name: 'some experiment' }));
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('shows Active and Archive tabs', () => {
    render(
      <BrowserRouter>
        <RunListsRouter {...generateProps()} />
      </BrowserRouter>,
    );

    screen.getByText('Active');
    screen.getByText('Archived');
  });
});
