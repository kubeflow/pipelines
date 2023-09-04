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
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils from 'src/TestUtils';
import { PipelineListFC } from 'src/pages/functional_components/PipelineListFC';
import { V2beta1Pipeline } from 'src/apisv2beta1/pipeline';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { RoutePage } from 'src/components/Router';
import { range } from 'lodash';

describe('PipelineList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelines');
  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
  const deletePipelineSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'deletePipeline');
  const deletePipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'deletePipelineVersion');

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: RoutePage.PIPELINES } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Pipelines' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  function mockNPipelines(n: number): void {
    listPipelinesSpy.mockImplementation(() => ({
      pipelines: range(1, n + 1).map(
        i =>
          ({
            display_name: 'test pipeline name' + i,
            pipeline_id: 'test-pipeline-id' + i,
            created_at: new Date(2023, 9, 3, 16, 5, 48),
            description: 'test pipeline description',
          } as V2beta1Pipeline),
      ),
    }));
  }

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders an empty list with empty state message', async () => {
    listPipelinesSpy.mockImplementationOnce(() => {});
    render(
      <CommonTestWrapper>
        <PipelineListFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
    });

    screen.getByText('No pipelines found. Click "Upload pipeline" to start.');
  });

  it('renders a list of one pipeline', async () => {
    mockNPipelines(1);
    render(
      <CommonTestWrapper>
        <PipelineListFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
    });

    screen.getByText('test pipeline name1');
    screen.getByText('test pipeline description');
  });

  it('renders a list of one pipeline with no description or created date', async () => {
    listPipelinesSpy.mockImplementationOnce(() => ({
      pipelines: [{ display_name: 'test pipeline name1', pipeline_id: 'test-pipeline-id1' }],
    }));

    render(
      <CommonTestWrapper>
        <PipelineListFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
    });

    screen.getByText('test pipeline name1');
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', async () => {
    mockNPipelines(10);
    render(
      <CommonTestWrapper>
        <PipelineListFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
      expect(listPipelinesSpy).toHaveBeenLastCalledWith(undefined, '', 10, 'created_at desc', '');
    });
  });

  it('shows error banner when listing pipelines fails', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'There was something wrong!');
    render(
      <CommonTestWrapper>
        <PipelineListFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listPipelinesSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledWith({
      additionalInfo: 'There was something wrong!',
      message: `Error: failed to retrieve list of pipelines. Click Details for more information.`,
      mode: 'error',
    });
  });
});
