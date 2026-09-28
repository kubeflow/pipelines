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

import { act, render as renderDOM, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from './modernization/ThemeProvider';
import { SpyInstance } from 'vitest';
import PipelinesDialogV2, { PipelinesDialogV2Props } from './PipelinesDialogV2';
import { PageProps } from 'src/pages/Page';
import { Apis, PipelineSortKeys } from 'src/lib/Apis';
import { V2beta1Pipeline, V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
import { flushPromisesInAct } from 'src/TestUtils';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { NameWithTooltip } from 'src/components/CustomTableNameColumn';

function render(element: ReactNode) {
  return renderDOM(<ThemeProvider>{element}</ThemeProvider>);
}

function generateProps(): PipelinesDialogV2Props {
  return {
    ...generatePageProps(),
    open: true,
    selectorDialog: '',
    onClose: vi.fn(),
    namespace: 'ns',
    pipelineSelectorColumns: [
      {
        customRenderer: NameWithTooltip,
        flex: 1,
        label: 'Pipeline name',
        sortKey: PipelineSortKeys.DISPLAY_NAME,
      },
      { label: 'Description', flex: 2 },
      { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
    ],
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

const oldPipeline: V2beta1Pipeline = {
  pipeline_id: 'old-run-pipeline-id',
  name: 'old-mock-pipeline-name',
  display_name: 'old mock pipeline name',
};

const newPipeline: V2beta1Pipeline = {
  pipeline_id: 'new-run-pipeline-id',
  name: 'new-mock-pipeline-name',
  display_name: 'new mock pipeline name',
};

describe('PipelinesDialog', () => {
  let listPipelineSpy: SpyInstance;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('matchMedia', (media: string) => ({
      media,
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    listPipelineSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'listPipelines')
      .mockImplementation((...args) => {
        const response: V2beta1ListPipelinesResponse = {
          pipelines: [oldPipeline, newPipeline],
          total_size: 2,
        };
        return Promise.resolve(response);
      });
  });

  afterEach(async () => {
    vi.resetAllMocks();
    vi.unstubAllGlobals();
  });

  it('it renders correctly in multi user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
        <PipelinesDialogV2 {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await flushPromisesInAct();

    expect(listPipelineSpy).toHaveBeenCalledWith('ns', '', 10, 'created_at desc', '');
    // Verify the display names are shown instead of the names
    screen.getByText('old mock pipeline name');
    screen.getByText('new mock pipeline name');
  });

  it('it renders correctly in single user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
        <PipelinesDialogV2 {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await flushPromisesInAct();

    expect(listPipelineSpy).toHaveBeenCalledWith(undefined, '', 10, 'created_at desc', '');
    // Verify the display names are shown instead of the names
    screen.getByText('old mock pipeline name');
    screen.getByText('new mock pipeline name');
  });
  it('keeps the latest pipeline choice when detail reads finish out of order', async () => {
    let finishOld!: (pipeline: V2beta1Pipeline) => void;
    vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline')
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishOld = resolve;
          }),
      )
      .mockResolvedValueOnce(newPipeline);
    const props = generateProps();
    render(<PipelinesDialogV2 {...props} />);
    await userEvent.click(
      await screen.findByRole('radio', { name: 'Select resource old mock pipeline name' }),
    );
    await userEvent.click(
      screen.getByRole('radio', { name: 'Select resource new mock pipeline name' }),
    );
    const useButton = screen.getByRole('button', { name: 'Use this pipeline' });
    await waitFor(() => expect(useButton).toBeEnabled());
    await act(async () => {
      finishOld(oldPipeline);
    });
    await userEvent.click(useButton);
    expect(props.onClose).toHaveBeenCalledWith(true, newPipeline);
  });

  it('invalidates pending detail reads when switching from private to shared pipelines', async () => {
    let finish!: (pipeline: V2beta1Pipeline) => void;
    vi.spyOn(Apis.pipelineServiceApiV2, 'getPipeline').mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
        <PipelinesDialogV2 {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await userEvent.click(
      await screen.findByRole('radio', { name: 'Select resource old mock pipeline name' }),
    );
    await userEvent.click(screen.getByRole('tab', { name: 'Shared' }));
    await waitFor(() =>
      expect(listPipelineSpy).toHaveBeenLastCalledWith(undefined, '', 10, 'created_at desc', ''),
    );
    await act(async () => {
      finish(oldPipeline);
    });
    expect(screen.getByRole('button', { name: 'Use this pipeline' })).toBeDisabled();
    expect(
      screen.getByRole('radio', { name: 'Select resource old mock pipeline name' }),
    ).not.toBeChecked();
  });
});
