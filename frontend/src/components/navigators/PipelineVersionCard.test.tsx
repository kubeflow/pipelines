/*
 * Copyright 2021 The Kubeflow Authors
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

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline/api';
import { testBestPractices } from 'src/TestUtils';
import { PipelineVersionCard } from './PipelineVersionCard';

const DEFAULT_VERSION_NAME = 'default version';
const REVISION_NAME = 'revision';

const PIPELINE_ID_V2_PYTHON_TWO_STEPS = '8fbe3bd6-a01f-11e8-98d0-529269fb1460';
const PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT: ApiPipelineVersion = {
  created_at: new Date('2021-11-24T20:58:23.000Z'),
  id: PIPELINE_ID_V2_PYTHON_TWO_STEPS,
  name: DEFAULT_VERSION_NAME,
  description: 'This is default version description.',
  parameters: [
    {
      name: 'message',
    },
  ],
};
const PIPELINE_ID_V2_PYTHON_TWO_STEPS_REV = '9fbe3bd6-a01f-11e8-98d0-529269fb1460';
const PIPELINE_V2_PYTHON_TWO_STEPS_REV: ApiPipelineVersion = {
  created_at: new Date('2021-12-24T20:58:23.000Z'),
  id: PIPELINE_ID_V2_PYTHON_TWO_STEPS_REV,
  name: REVISION_NAME,
  description: 'This is version description.',
  parameters: [
    {
      name: 'revision-message',
    },
  ],
};

const V2_TWO_STEPS_VERSION_LIST: ApiPipelineVersion[] = [
  PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT,
  PIPELINE_V2_PYTHON_TWO_STEPS_REV,
];
const PIPELINE_V2_PYTHON_TWO_STEPS: ApiPipeline = {
  ...PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT,
  description: 'This is pipeline level description.',
  name: 'v2_lightweight_python_functions_pipeline',
  default_version: PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT,
};
testBestPractices();
describe('PipelineVersionCard', () => {
  it('makes Show Summary button visible by default', async () => {
    render(
      <PipelineVersionCard
        apiPipeline={PIPELINE_V2_PYTHON_TWO_STEPS}
        selectedVersion={PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT}
        versions={V2_TWO_STEPS_VERSION_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    screen.getByText('Show Summary');
    expect(screen.queryByText('Hide')).toBeNull();
  });

  it('clicks to open and hide Summary', async () => {
    render(
      <PipelineVersionCard
        apiPipeline={PIPELINE_V2_PYTHON_TWO_STEPS}
        selectedVersion={PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT}
        versions={V2_TWO_STEPS_VERSION_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    userEvent.click(screen.getByText('Show Summary'));
    expect(screen.queryByText('Show Summary')).toBeNull();
    userEvent.click(screen.getByText('Hide'));
    screen.getByText('Show Summary');
  });

  it('shows Summary and checks detail', async () => {
    render(
      <PipelineVersionCard
        apiPipeline={PIPELINE_V2_PYTHON_TWO_STEPS}
        selectedVersion={PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT}
        versions={V2_TWO_STEPS_VERSION_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    userEvent.click(screen.getByText('Show Summary'));

    screen.getByText('Pipeline ID');
    screen.getByText(PIPELINE_ID_V2_PYTHON_TWO_STEPS);
    screen.getByText('Version');
    screen.getByText(DEFAULT_VERSION_NAME);
    screen.getByText('Version source');
    screen.getByText('Uploaded on');
    screen.getByText('Pipeline Description');
    screen.getByText('This is pipeline level description.');
    screen.getByText('Default Version Description');
    screen.getByText('This is default version description.');
  });

  it('shows version list', async () => {
    const { getByRole } = render(
      <PipelineVersionCard
        apiPipeline={PIPELINE_V2_PYTHON_TWO_STEPS}
        selectedVersion={PIPELINE_V2_PYTHON_TWO_STEPS_DEFAULT}
        versions={V2_TWO_STEPS_VERSION_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    userEvent.click(screen.getByText('Show Summary'));

    fireEvent.click(getByRole('button', { name: DEFAULT_VERSION_NAME }));
    fireEvent.click(getByRole('listbox'));
    getByRole('option', { name: REVISION_NAME });
  });
});
