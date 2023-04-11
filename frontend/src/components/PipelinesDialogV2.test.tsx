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
import { render, screen } from '@testing-library/react';
import PipelinesDialogV2, { PipelinesDialogV2Props } from './PipelinesDialogV2';
import { PageProps } from 'src/pages/Page';
import { Apis, PipelineSortKeys } from 'src/lib/Apis';
import { V2beta1Pipeline, V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
import TestUtils from 'src/TestUtils';
import { BuildInfoContext } from 'src/lib/BuildInfo';

function generateProps(): PipelinesDialogV2Props {
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

const oldPipeline: V2beta1Pipeline = {
  pipeline_id: 'old-run-pipeline-id',
  display_name: 'old mock pipeline name',
};

const newPipeline: V2beta1Pipeline = {
  pipeline_id: 'new-run-pipeline-id',
  display_name: 'new mock pipeline name',
};

describe('PipelinesDialog', () => {
  let listPipelineSpy: jest.SpyInstance<{}>;

  beforeEach(() => {
    jest.clearAllMocks();
    listPipelineSpy = jest
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
    jest.resetAllMocks();
  });

  it('it renders correctly in multi user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
        <PipelinesDialogV2 {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await TestUtils.flushPromises();

    expect(listPipelineSpy).toHaveBeenCalledWith('ns', '', 10, 'created_at desc', '');
    screen.getByText('old mock pipeline name');
    screen.getByText('new mock pipeline name');
  });

  it('it renders correctly in single user mode', async () => {
    const tree = render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: false }}>
        <PipelinesDialogV2 {...generateProps()} />
      </BuildInfoContext.Provider>,
    );
    await TestUtils.flushPromises();

    expect(listPipelineSpy).toHaveBeenCalledWith(undefined, '', 10, 'created_at desc', '');
    screen.getByText('old mock pipeline name');
    screen.getByText('new mock pipeline name');
  });
});
