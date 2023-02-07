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

import { render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { Api } from 'src/mlmd/library';
import {
  Execution,
  ExecutionType,
  GetExecutionsResponse,
  GetExecutionTypesResponse,
  Value,
} from 'src/third_party/mlmd';
import { RoutePage } from '../components/Router';
import TestUtils from '../TestUtils';
import ExecutionList from './ExecutionList';
import { PageProps } from './Page';

const pipelineName = 'test_pipeline';
const executionName = 'test_execution';
const executionId = 2023;

describe('ExecutionList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();

  const getExecutionsSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getExecutions');
  const getExecutionTypesSpy = jest.spyOn(
    Api.getInstance().metadataStoreService,
    'getExecutionTypes',
  );

  beforeEach(() => {
    getExecutionTypesSpy.mockImplementation(() => {
      const executionType = new ExecutionType();
      executionType.setId(6);
      executionType.setName('String');
      const response = new GetExecutionTypesResponse();
      response.setExecutionTypesList([executionType]);
      return Promise.resolve(response);
    });

    getExecutionsSpy.mockImplementation(() => {
      const execution = new Execution();
      const pipelineValue = new Value();
      pipelineValue.setStringValue(pipelineName);
      execution.getPropertiesMap().set('pipeline_name', pipelineValue);
      const executionValue = new Value();
      executionValue.setStringValue(executionName);
      execution.getPropertiesMap().set('Name', executionValue);
      execution.setName(executionName);
      execution.setId(executionId);
      const response = new GetExecutionsResponse();
      response.setExecutionsList([execution]);
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

  it('renders one execution', async () => {
    render(<ExecutionList {...generateProps()} />);
    await waitFor(() => {
      expect(getExecutionTypesSpy).toHaveBeenCalledTimes(1);
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    await screen.queryByText(executionName);
  });

  /*
  it('renders at most 10 execution in one page by default.', async () => {
    // getExecutionsSpy.mockClear();
    getExecutionsSpy.mockImplementation(() => {
      const executionList: Execution[] = [];
      for (let i = 1; i <= 15; i++) {
        const execution = new Execution();
        const pipelineValue = new Value();
        pipelineValue.setStringValue(pipelineName + '_' + i);
        execution.getPropertiesMap().set('pipeline_name', pipelineValue);
        const executionValue = new Value();
        executionValue.setStringValue(executionName + '_' + i);
        execution.getPropertiesMap().set('name', executionValue);
        execution.setName(executionName);
        executionList.push(execution);
      }
      const response = new GetExecutionsResponse();
        response.setExecutionsList(executionList);
      return Promise.resolve(response);
    });

    render(<ExecutionList {...generateProps()} />);
    await waitFor(() => {
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    screen.getByText(pipelineName);
  });
*/
  it('found no execution', async () => {
    getExecutionsSpy.mockClear();
    getExecutionsSpy.mockImplementation(() => {
      const response = new GetExecutionsResponse();
      response.setExecutionsList([]);
      return Promise.resolve(response);
    });
    render(<ExecutionList {...generateProps()} />);
    await waitFor(() => {
      expect(getExecutionsSpy).toHaveBeenCalledTimes(1);
    });

    screen.getByText('No executions found.');
  });
});
