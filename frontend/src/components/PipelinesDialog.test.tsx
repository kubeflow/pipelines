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

import * as React from 'react';
import { render } from '@testing-library/react';
import PipelinesDialog, { PipelinesDialogProps } from './PipelinesDialog';
import { PageProps } from '../pages/Page';
import { Apis, PipelineSortKeys } from '../lib/Apis';
import { ApiListPipelinesResponse, ApiPipeline } from '../apis/pipeline';
import TestUtils from '../TestUtils';
import { BuildInfoContext } from '../lib/BuildInfo';

function generateProps(): PipelinesDialogProps {
  return {
    ...generatePageProps(),
    open: true,
    selectorDialog: '',
    onClose: jest.fn(),
    namespace: 'ns',
    pipelineSelectorColumns: [
      {
        flex: 1,
        label: 'Pipeline name',
        sortKey: PipelineSortKeys.NAME,
      },
      { label: 'Description', flex: 2 },
      { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
    ],
  };
}

function generatePageProps(): PageProps {
  return {
    history: {} as any,
    location: '' as any,
    match: {} as any,
    toolbarProps: {} as any,
    updateBanner: jest.fn(),
    updateDialog: jest.fn(),
    updateSnackbar: jest.fn(),
    updateToolbar: jest.fn(),
  };
}

const oldPipeline = newMockPipeline();
const newPipeline = newMockPipeline();

function newMockPipeline(): ApiPipeline {
  return {
    id: 'run-pipeline-id',
    name: 'mock pipeline name',
    parameters: [],
    default_version: {
      id: 'run-pipeline-version-id',
      name: 'mock pipeline version name',
    },
  };
}

describe('PipelinesDialog', () => {
  let listPipelineSpy: jest.SpyInstance<{}>;

  beforeEach(() => {
    jest.clearAllMocks();
    listPipelineSpy = jest
      .spyOn(Apis.pipelineServiceApi, 'listPipelines')
      .mockImplementation((...args) => {
        const response: ApiListPipelinesResponse = {
          pipelines: [oldPipeline, newPipeline],
          total_size: 2,
        };
        return Promise.resolve(response);
      });
  });

  afterEach(async () => {
    jest.resetAllMocks();
  });

  it('it renders correctly in multi user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
        <PipelinesDialog {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('it renders correctly in single user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
        <PipelinesDialog {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });
});
