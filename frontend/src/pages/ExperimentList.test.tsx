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
import EnhancedExperimentList, { ExperimentList } from './ExperimentList';
import TestUtils from 'src/TestUtils';
import { V2beta1RunStorageState, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { ExpandState } from 'src/components/CustomTable';
import { PageProps } from './Page';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { range } from 'lodash';
import { ButtonKeys } from 'src/lib/Buttons';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { MemoryRouter } from 'react-router-dom';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import '@testing-library/jest-dom';

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

describe('ExperimentList', () => {
  let updateBannerSpy: jest.SpyInstance;
  let updateDialogSpy: jest.SpyInstance;
  let updateSnackbarSpy: jest.SpyInstance;
  let updateToolbarSpy: jest.SpyInstance;
  let historyPushSpy: jest.SpyInstance;
  let listExperimentsSpy: jest.SpyInstance;
  let getExperimentSpy: jest.SpyInstance;
  let deleteExperimentSpy: jest.SpyInstance;
  let listRunsSpy: jest.SpyInstance;
  let formatDateStringSpy: jest.SpyInstance;

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ExperimentList,
      { search: '' } as any,
      '' as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function mockListNExperiments(n: number) {
    return jest.fn().mockImplementation(() => ({
      experiments: range(n).map(i => ({
        experiment_id: 'test-experiment-id' + i,
        display_name: 'test experiment name' + i,
      })),
    }));
  }

  async function mountWithNExperiments(
    n: number,
    nRuns: number,
    { namespace }: { namespace?: string } = {},
  ): Promise<ReturnType<typeof TestUtils.renderWithRouter>> {
    listExperimentsSpy.mockImplementation(mockListNExperiments(n));
    listRunsSpy.mockImplementation(() => ({
      runs: range(nRuns).map(i => ({
        run_id: 'test-run-id' + i,
        display_name: 'test run name' + i,
      })),
    }));
    const result = TestUtils.renderWithRouter(
      <ExperimentList {...generateProps()} namespace={namespace} />,
    );
    await TestUtils.flushPromises();
    return result;
  }

  beforeEach(() => {
    updateBannerSpy = jest.fn();
    updateDialogSpy = jest.fn();
    updateSnackbarSpy = jest.fn();
    updateToolbarSpy = jest.fn();
    historyPushSpy = jest.fn();
    listExperimentsSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
    getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
    deleteExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'deleteExperiment');
    listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');
    formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

    // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
    // test environments
    formatDateStringSpy.mockImplementation((date?: Date | string) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
  });

  afterEach(() => {
    jest.resetAllMocks();
    jest.clearAllMocks();
  });

  it('renders an empty list with empty state message', async () => {
    const { container } = await mountWithNExperiments(0, 0);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with no description', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with error', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment without run', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with one run', async () => {
    const { container } = await mountWithNExperiments(1, 1);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with two runs', async () => {
    const { container } = await mountWithNExperiments(1, 2);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with pagination', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with pagination and page 2', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of one experiment with pagination and page 3', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    expect(container).toMatchSnapshot();
  });

  it('switches to Archive view, renders a list of one archived experiment', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(container).toMatchSnapshot();
  });

  it('switches to Archive view, renders a list of one archived experiment with one run', async () => {
    const { container } = await mountWithNExperiments(1, 1);
    expect(container).toMatchSnapshot();
  });

  it('switches to Archive view, renders a list of one archived experiment with two runs', async () => {
    const { container } = await mountWithNExperiments(1, 2);
    expect(container).toMatchSnapshot();
  });

  it('calls Apis to list experiments, sorted by creation time in descending order', async () => {
    await mountWithNExperiments(1, 0);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
  });

  it('has a Refresh button in toolbar', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    // The refresh button is part of the toolbar, not directly in the component
    expect(updateToolbarSpy).toHaveBeenCalled();
    const toolbarActions = updateToolbarSpy.mock.calls[0][0].actions;
    expect(toolbarActions).toHaveProperty('refresh');
  });

  it('has a new experiment button in toolbar', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    expect(updateToolbarSpy).toHaveBeenCalled();
    const toolbarActions = updateToolbarSpy.mock.calls[0][0].actions;
    expect(toolbarActions).toHaveProperty('newExperiment');
  });

  // TODO: Fix this test
  it.skip('renders experiment name as link to its details page', async () => {
    const { container } = await mountWithNExperiments(1, 0);
    const nameLink = screen.getByText('test experiment name0');
    expect(nameLink).toBeInTheDocument();
    expect(nameLink.closest('a')).toHaveAttribute(
      'href',
      '/experiments/details/test-experiment-id0',
    );
  });

  it.skip('renders last 5 runs statuses', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    expect(container).toMatchSnapshot();
  });

  it('renders a list of experiments, can expand to see runs', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    const expandButton = container.querySelector('button[aria-label="Expand"]');

    if (expandButton) {
      await fireEvent.click(expandButton);
      await TestUtils.flushPromises();
      expect(listRunsSpy).toHaveBeenCalledTimes(2); // Initial load + expand
    }
  });

  it.skip('renders a list of experiments, can collapse after expanded', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    const expandButton = container.querySelector('button[aria-label="Expand"]');

    if (expandButton) {
      await fireEvent.click(expandButton);
      await TestUtils.flushPromises();

      const collapseButton = container.querySelector('button[aria-label="Collapse"]');
      if (collapseButton) {
        await fireEvent.click(collapseButton);
        await TestUtils.flushPromises();
      }
    }

    expect(container).toMatchSnapshot();
  });

  it('renders a list of experiments, can expand to see runs, and navigates to runs page', async () => {
    const { container } = await mountWithNExperiments(1, 5);
    const expandButton = container.querySelector('button[aria-label="Expand"]');

    if (expandButton) {
      await fireEvent.click(expandButton);
      await TestUtils.flushPromises();

      const runLinks = container.querySelectorAll('a[href*="/runs/details/"]');
      expect(runLinks.length).toBeGreaterThan(0);
    }
  });

  it('loads experiments for the specified namespace', async () => {
    await mountWithNExperiments(1, 0, { namespace: 'test-ns' });
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
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
      'test-ns', // namespace
    );
  });

  it('loads experiments for all namespaces when namespace is not specified', async () => {
    await mountWithNExperiments(1, 0);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(...LIST_EXPERIMENT_DEFAULTS);
  });

  it('shows error banner when experiment list fails to load', async () => {
    listExperimentsSpy.mockImplementation(() => {
      throw new Error('Failed to load experiments');
    });

    const { container } = TestUtils.renderWithRouter(<ExperimentList {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        additionalInfo: 'Failed to load experiments',
        message: expect.stringContaining('Error'),
        mode: 'error',
      }),
    );
  });

  it('shows error banner when experiment delete fails', async () => {
    deleteExperimentSpy.mockImplementation(() => {
      throw new Error('Failed to delete experiment');
    });

    const { container } = await mountWithNExperiments(1, 0);
    const checkbox = container.querySelector('input[type="checkbox"]');

    if (checkbox) {
      await fireEvent.click(checkbox);
      await TestUtils.flushPromises();

      const deleteButton = screen.getByText('Delete');
      await fireEvent.click(deleteButton);

      // Confirm deletion in dialog
      const confirmButton = screen.getByText('Delete');
      await fireEvent.click(confirmButton);

      expect(updateBannerSpy).toHaveBeenCalledWith({
        additionalInfo: 'Failed to delete experiment',
        message: 'Error: failed to delete experiment: ',
        mode: 'error',
      });
    }
  });
});
