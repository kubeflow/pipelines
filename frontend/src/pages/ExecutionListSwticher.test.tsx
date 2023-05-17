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
import { RoutePage } from 'src/components/Router';
import { testBestPractices } from 'src/TestUtils';
import ExecutionListSwitcher from 'src/pages/ExecutionListSwitcher';
import { PageProps } from 'src/pages/Page';
import { CommonTestWrapper } from 'src/TestWrapper';

testBestPractices();

describe('ExecutionListSwitcher', () => {
  let getExecutionsSpy: jest.Mock<{}>;
  let getExecutionTypesSpy: jest.Mock<{}>;
  const getExecutionsRequest = new GetExecutionsRequest();

  beforeEach(() => {
    getExecutionsSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getExecutions');
    getExecutionTypesSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getExecutionTypes');

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
    const pageProps: PageProps = {
      history: {} as any,
      location: {
        pathname: RoutePage.EXECUTIONS,
      } as any,
      match: {} as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: () => null,
      updateDialog: () => null,
      updateSnackbar: () => null,
      updateToolbar: () => null,
    };
    return pageProps;
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

  it('shows "Flat" and "Group" tabs', () => {
    render(
      <CommonTestWrapper>
        <ExecutionListSwitcher {...generateProps()} />
      </CommonTestWrapper>,
    );

    screen.getByText('Default');
    screen.getByText('Grouped');
  });

  it('disable pagination if switch to "Group" view', async () => {
    render(
      <CommonTestWrapper>
        <ExecutionListSwitcher {...generateProps()} />
      </CommonTestWrapper>,
    );

    const groupTab = screen.getByText('Grouped');
    fireEvent.click(groupTab);

    // "Group" view will call getExection() without list options
    getExecutionsRequest.setOptions(undefined);

    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(2); // once for flat, once for group
      expect(getExecutionsSpy).toHaveBeenCalledTimes(2); // once for flat, once for group
      expect(getExecutionsSpy).toHaveBeenLastCalledWith(getExecutionsRequest);
    });

    expect(screen.queryByText('Rows per page:')).toBeNull(); // no footer
    expect(screen.queryByText('10')).toBeNull(); // no footer
  });
});
