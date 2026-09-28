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
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { V2beta1Pipeline, V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
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
const newPipeline = {
  ...newMockPipeline(),
  pipeline_id: 'new-pipeline-id',
  display_name: 'new pipeline name',
};

function newMockPipeline(): V2beta1Pipeline {
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
      const response: V2beta1ListPipelinesResponse = {
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
    render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value={'ns'}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>,
    );
    await flushPromisesInAct();
    expect(screen.getByRole('tablist', { name: 'Pipeline visibility' })).toBeVisible();
    expect(screen.getByRole('tab', { name: 'Private' })).toHaveAttribute('aria-selected', 'true');
    expect(Apis.pipelineServiceApiV2.listPipelines).toHaveBeenLastCalledWith(
      'ns',
      '',
      10,
      'created_at desc',
      '',
    );
  });

  it('it renders correctly in single user mode', async () => {
    render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
          <NamespaceContext.Provider value={undefined}>
            <PrivateAndSharedPipelines {...generateProps()} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>,
    );
    await flushPromisesInAct();
    expect(screen.queryByRole('tablist', { name: 'Pipeline visibility' })).toBeNull();
    expect(screen.getByRole('list', { name: 'Pipelines' })).toBeVisible();
    expect(Apis.pipelineServiceApiV2.listPipelines).toHaveBeenLastCalledWith(
      undefined,
      '',
      10,
      'created_at desc',
      '',
    );
  });
  it('reloads private pipelines when the namespace changes and ignores the old namespace response', async () => {
    let finishOld: (value: V2beta1ListPipelinesResponse) => void = () => {};
    vi.mocked(Apis.pipelineServiceApiV2.listPipelines).mockImplementation((namespace) =>
      namespace === 'old'
        ? new Promise((resolve) => {
            finishOld = resolve;
          })
        : Promise.resolve({ pipelines: [newPipeline] }),
    );
    const props = generateProps();
    const view = (namespace: string) => (
      <MemoryRouter>
        <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
          <NamespaceContext.Provider value={namespace}>
            <PrivateAndSharedPipelines {...props} />
          </NamespaceContext.Provider>
        </BuildInfoContext.Provider>
      </MemoryRouter>
    );
    const rendered = render(view('old'));
    await waitFor(() =>
      expect(Apis.pipelineServiceApiV2.listPipelines).toHaveBeenCalledWith(
        'old',
        '',
        10,
        'created_at desc',
        '',
      ),
    );
    rendered.rerender(view('new'));
    await screen.findByRole('link', { name: 'new pipeline name' });
    finishOld({ pipelines: [oldPipeline] });
    await flushPromisesInAct();
    expect(screen.queryByRole('link', { name: 'mock pipeline name' })).toBeNull();
    expect(screen.getByRole('link', { name: 'new pipeline name' })).toBeVisible();
  });
});
