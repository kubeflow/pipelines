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
import { createMemoryHistory } from 'history';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { ApiListPipelinesResponse, ApiPipeline } from '../apis/pipeline';
import TestUtils from '../TestUtils';
import { BuildInfoContext } from '../lib/BuildInfo';
import PrivateAndSharedPipelines, {
  PrivateAndSharedProps,
  PrivateAndSharedTab,
} from './PrivateAndSharedPipelines';
import { Router } from 'react-router-dom';
import { NamespaceContext } from '../lib/KubeflowClient';

function generateProps(): PrivateAndSharedProps {
  return {
    ...generatePageProps(),
    view: PrivateAndSharedTab.PRIVATE,
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
    created_at: new Date('2022-09-21T13:53:59Z'),
    description: 'mock pipeline description',
  };
}

describe('PrivateAndSharedPipelines', () => {
  const history = createMemoryHistory({
    initialEntries: ['/does-not-matter'],
  });
  beforeEach(() => {
    jest.clearAllMocks();
    let listPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
    listPipelineSpy.mockImplementation((...args) => {
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
      <Router history={history}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value={'ns'}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </Router>,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('it renders correctly in single user mode', async () => {
    const tree = render(
      <Router history={history}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
          <NamespaceContext.Provider value={undefined}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </Router>,
    );
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });
});
