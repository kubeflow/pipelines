/*
 * Copyright 2023 The Kubeflow Authors
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
import * as React from 'react';
import { Api } from 'src/mlmd/library';
import {
  Execution,
  ExecutionType,
  GetExecutionsRequest,
  GetExecutionsResponse,
  GetExecutionTypesResponse,
  Value,
} from 'src/third_party/mlmd';
import { ListOperationOptions } from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb';
import { RoutePage } from 'src/components/Router';
import TestUtils, { testBestPractices } from 'src/TestUtils';
import ExecutionList from 'src/pages/ExecutionList';
import { PageProps } from 'src/pages/Page';
import { CommonTestWrapper } from 'src/TestWrapper';

testBestPractices();

describe('ExecutionList ("Default" view)', () => {
  let updateBannerSpy: jest.Mock<{}>;
  let updateDialogSpy: jest.Mock<{}>;
  let updateSnackbarSpy: jest.Mock<{}>;
  let updateToolbarSpy: jest.Mock<{}>;
  let historyPushSpy: jest.Mock<{}>;
  let getExecutionsSpy: jest.Mock<{}>;
  let getExecutionTypesSpy: jest.Mock<{}>;

  const listOperationOpts = new ListOperationOptions();
  listOperationOpts.setMaxResultSize(10);
  const getExecutionsRequest = new GetExecutionsRequest();
  getExecutionsRequest.setOptions(listOperationOpts),
    beforeEach(() => {
      updateBannerSpy = jest.fn();
      updateDialogSpy = jest.fn();
      updateSnackbarSpy = jest.fn();
      updateToolbarSpy = jest.fn();
      historyPushSpy = jest.fn();
      getExecutionsSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getExecutions');
      getExecutionTypesSpy = jest.spyOn(
        Api.getInstance().metadataStoreService,
        'getExecutionTypes',
      );

      getExecutionTypesSpy.mockImplementation(() => {
        const executionType = new ExecutionType();
        executionType.setId(6);
        executionType.setName('String');
        const response = new GetExecutionTypesResponse();
        response.setExecutionTypesList([executionType]);
        return Promise.resolve(response);
      });

      getExecutionsSpy.mockImplementation(() => {
        const executions = mockNExecutions(5);
        const response = new GetExecutionsResponse();
        response.setExecutionsList(executions);
        return Promise.resolve(response);
      });
    });

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ExecutionList,
      { pathname: RoutePage.EXECUTIONS } as any,
      '' as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function mockNExecutions(n: number) {
    let executions: Execution[] = [];
    for (let i = 1; i <= n; i++) {
      const execution = new Execution();
      const pipelineValue = new Value();
      const pipelineName = `pipeline ${i}`;
      pipelineValue.setStringValue(pipelineName);
      execution.getCustomPropertiesMap().set('pipeline_name', pipelineValue);
      const executionValue = new Value();
      const executionName = `test execution ${i}`;
      executionValue.setStringValue(executionName);
      execution.getPropertiesMap().set('name', executionValue);
      execution.setName(executionName);
      execution.setId(i);
      executions.push(execution);
    }
    return executions;
  }

  it('renders one execution', async () => {
    getExecutionsSpy.mockImplementation(() => {
      const executions = mockNExecutions(1);
      const response = new GetExecutionsResponse();
      response.setExecutionsList(executions);
      return Promise.resolve(response);
    });
    render(
      <CommonTestWrapper>
        <ExecutionList {...generateProps()} isGroupView={false} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    screen.getByText('test execution 1');
  });

  it('displays footer with "10" as default value', async () => {
    render(
      <CommonTestWrapper>
        <ExecutionList {...generateProps()} isGroupView={false} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    screen.getByText('Rows per page:');
    screen.getByText('10');
  });

  it('shows 20th execution if page size is 20', async () => {
    render(
      <CommonTestWrapper>
        <ExecutionList {...generateProps()} isGroupView={false} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });
    expect(screen.queryByText('test execution 20')).toBeNull(); // Can not see the 20th execution initially

    getExecutionsSpy.mockImplementation(() => {
      const executions = mockNExecutions(20);
      const response = new GetExecutionsResponse();
      response.setExecutionsList(executions);
      return Promise.resolve(response);
    });

    const originalRowsPerPage = screen.getByText('10');
    fireEvent.click(originalRowsPerPage);
    const newRowsPerPage = screen.getByText('20'); // Change to render 20 rows per page.
    fireEvent.click(newRowsPerPage);

    listOperationOpts.setMaxResultSize(20);
    getExecutionsRequest.setOptions(listOperationOpts),
      await waitFor(() => {
        // API will be called again if "Rows per page" is changed
        expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
        expect(getExecutionsSpy).toHaveBeenLastCalledWith(getExecutionsRequest);
      });

    screen.getByText('test execution 20'); // The 20th execution appears.
  });

  it('finds no execution', async () => {
    getExecutionsSpy.mockClear();
    getExecutionsSpy.mockImplementation(() => {
      const response = new GetExecutionsResponse();
      response.setExecutionsList([]);
      return Promise.resolve(response);
    });
    render(
      <CommonTestWrapper>
        <ExecutionList {...generateProps()} isGroupView={false} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    screen.getByText('No executions found.');
  });
});
