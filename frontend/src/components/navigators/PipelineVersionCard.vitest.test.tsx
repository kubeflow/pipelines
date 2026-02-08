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
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { testBestPractices } from 'src/TestUtils';
import { PipelineVersionCard } from './PipelineVersionCard';

const OLD_VERSION_NAME = 'old version';
const NEW_VERSION_NAME = 'new version';
const TEST_PIPELINE_ID = 'pipeline-id';

const OLD_TEST_PIPELINE_VERSION_ID = 'old-version-id';
const OLD_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
  created_at: new Date('2021-11-24T20:58:23.000Z'),
  description: 'This is old version description.',
  display_name: OLD_VERSION_NAME,
  pipeline_id: TEST_PIPELINE_ID,
  pipeline_version_id: OLD_TEST_PIPELINE_VERSION_ID,
};

const NEW_TEST_PIPELINE_VERSION_ID = 'new-version-id';
const NEW_TEST_PIPELINE_VERSION: V2beta1PipelineVersion = {
  created_at: new Date('2021-12-24T20:58:23.000Z'),
  description: 'This is new version description.',
  display_name: NEW_VERSION_NAME,
  pipeline_id: TEST_PIPELINE_ID,
  pipeline_version_id: NEW_TEST_PIPELINE_VERSION_ID,
};

const TEST_PIPELINE_VERSIONS_LIST: V2beta1PipelineVersion[] = [
  OLD_TEST_PIPELINE_VERSION,
  NEW_TEST_PIPELINE_VERSION,
];

const TEST_PIPELINE: V2beta1Pipeline = {
  created_at: new Date('2021-11-24T20:58:23.000Z'),
  description: 'This is pipeline level description.',
  display_name: 'v2_lightweight_python_functions_pipeline',
  pipeline_id: TEST_PIPELINE_ID,
};
testBestPractices();
describe('PipelineVersionCard', () => {
  it('makes Show Summary button visible by default', async () => {
    render(
      <PipelineVersionCard
        pipeline={TEST_PIPELINE}
        selectedVersion={OLD_TEST_PIPELINE_VERSION}
        versions={TEST_PIPELINE_VERSIONS_LIST}
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
        pipeline={TEST_PIPELINE}
        selectedVersion={OLD_TEST_PIPELINE_VERSION}
        versions={TEST_PIPELINE_VERSIONS_LIST}
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
        pipeline={TEST_PIPELINE}
        selectedVersion={OLD_TEST_PIPELINE_VERSION}
        versions={TEST_PIPELINE_VERSIONS_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    userEvent.click(screen.getByText('Show Summary'));

    screen.getByText('Pipeline ID');
    screen.getByText(TEST_PIPELINE_ID);
    screen.getByText('Version');
    screen.getByText(OLD_VERSION_NAME);
    screen.getByText('Version source');
    screen.getByText('Uploaded on');
    screen.getByText('Pipeline Description');
    screen.getByText('This is pipeline level description.');
    screen.getByText('This is old version description.');
  });

  it('shows version list', async () => {
    const { getByRole } = render(
      <PipelineVersionCard
        pipeline={TEST_PIPELINE}
        selectedVersion={OLD_TEST_PIPELINE_VERSION}
        versions={TEST_PIPELINE_VERSIONS_LIST}
        handleVersionSelected={versionId => {
          return Promise.resolve();
        }}
      ></PipelineVersionCard>,
    );

    userEvent.click(screen.getByText('Show Summary'));

    fireEvent.click(getByRole('button', { name: OLD_VERSION_NAME }));
    fireEvent.click(getByRole('listbox'));
    getByRole('option', { name: NEW_VERSION_NAME });
  });
});
