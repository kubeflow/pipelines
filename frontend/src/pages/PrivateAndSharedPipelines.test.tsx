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

import { render, waitFor } from '@testing-library/react';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { V2Pipeline, V2ListPipelinesResponse } from 'src/apisv2/pipeline';
import { flushPromisesInAct } from 'src/TestUtils';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import PrivateAndSharedPipelines, {
  PrivateAndSharedProps,
  PrivateAndSharedTab,
} from './PrivateAndSharedPipelines';
import { MemoryRouter } from 'react-router';
import { NamespaceContext } from 'src/lib/KubeflowClient';

function generateProps(): PrivateAndSharedProps {
  return {
    ...generatePageProps(),
    view: PrivateAndSharedTab.PRIVATE,
  };
}

function generatePageProps(): PageProps {
  return {
    navigate: vi.fn(),
    location: '' as any,
    params: {},
    toolbarProps: {} as any,
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
}

const oldPipeline = newMockPipeline();
const newPipeline = newMockPipeline();

function newMockPipeline(): V2Pipeline {
  return {
    pipeline_id: 'run-pipeline-id',
    display_name: 'mock pipeline name',
    name: 'mock-pipeline-name',
    created_at: new Date('2022-09-21T13:53:59Z'),
    description: 'mock pipeline description',
  };
}

// This test is related to pipeline list where we intergrate with v2 API
// Thus, change to mock v2 API behavior and return values.
describe('PrivateAndSharedPipelines', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    let listPipelineSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelines');
    listPipelineSpy.mockImplementation((...args) => {
      const response: V2ListPipelinesResponse = {
        pipelines: [oldPipeline, newPipeline],
        total_size: 2,
      };
      return Promise.resolve(response);
    });
  });

  afterEach(async () => {
    vi.resetAllMocks();
  });

  it('it renders correctly in multi user mode', async () => {
    const tree = render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value={'ns'}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>,
    );
    await flushPromisesInAct();
    expect(tree).toMatchSnapshot();
  });

  it('it renders correctly in single user mode', async () => {
    const tree = render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
          <NamespaceContext.Provider value={undefined}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>,
    );
    await flushPromisesInAct();
    expect(tree).toMatchSnapshot();
  });

  it('reloads private pipelines when namespace changes', async () => {
    const listPipelineSpy = vi.mocked(Apis.pipelineServiceApiV2.listPipelines);
    const renderWithNamespace = (namespace: string) => (
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value={namespace}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>
    );
    const { rerender } = render(renderWithNamespace('test-ns-1'));
    await waitFor(() => expect(listPipelineSpy).toHaveBeenCalled());
    expect(listPipelineSpy.mock.lastCall?.[0]).toBe('test-ns-1');
    listPipelineSpy.mockClear();

    rerender(renderWithNamespace('test-ns-2'));
    await waitFor(() => expect(listPipelineSpy).toHaveBeenCalled());
    await flushPromisesInAct();
    const requestedNamespaces = listPipelineSpy.mock.calls.map(([namespace]) => namespace);
    expect(requestedNamespaces).toContain('test-ns-2');
    expect(requestedNamespaces).not.toContain('test-ns-1');
  });
});
