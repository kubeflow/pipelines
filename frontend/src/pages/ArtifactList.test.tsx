/*
 * Copyright 2021 Google LLC
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
import { ArtifactList } from './ArtifactList';
import { PageProps } from './Page';
import { RoutePage } from '../components/Router';
import TestUtils from '../TestUtils';
import { render, screen } from '@testing-library/react';
import {
  Api,
  Artifact,
  ArtifactType,
  GetArtifactsResponse,
  GetArtifactTypesResponse,
  Value,
} from '@kubeflow/frontend';
import { MemoryRouter } from 'react-router-dom';

const pipelineName = 'pipeline1';
const artifactName = 'artifact1';
describe('ArtifactList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();

  const getArtifactsSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getArtifacts');
  const getArtifactTypesSpy = jest.spyOn(
    Api.getInstance().metadataStoreService,
    'getArtifactTypes',
  );

  beforeEach(() => {
    getArtifactTypesSpy.mockImplementation(() => {
      const map = new Map<number, ArtifactType>();
      const artifactType = new ArtifactType();
      artifactType.setId(6);
      artifactType.setName('String');
      const response = new GetArtifactTypesResponse();
      response.setArtifactTypesList([artifactType]);
      return Promise.resolve(response);
    });
    getArtifactsSpy.mockImplementation(() => {
      const artifact = new Artifact();
      const pipelineValue = new Value();
      pipelineValue.setStringValue(pipelineName);
      artifact.getPropertiesMap().set('pipeline_name', pipelineValue);
      const artifactValue = new Value();
      artifactValue.setStringValue(artifactName);
      artifact.getPropertiesMap().set('name', artifactValue);
      artifact.setName(artifactName);
      const response = new GetArtifactsResponse();
      response.setArtifactsList([artifact]);
      return Promise.resolve(response);
    });
  });

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ArtifactList,
      { pathname: RoutePage.ARTIFACTS } as any,
      '' as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  it('renders one artifact', async () => {
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );
    await TestUtils.flushPromises();

    expect(getArtifactTypesSpy).toHaveBeenCalledTimes(1);
    expect(getArtifactsSpy).toHaveBeenCalledTimes(1);

    screen.getByText(pipelineName);
    screen.getByText(artifactName);
  });

  it('found no artifact', async () => {
    getArtifactsSpy.mockClear();
    getArtifactsSpy.mockImplementation(() => {
      const response = new GetArtifactsResponse();
      response.setArtifactsList([]);
      return Promise.resolve(response);
    });
    render(
      <MemoryRouter>
        <ArtifactList {...generateProps()} />
      </MemoryRouter>,
    );
    await TestUtils.flushPromises();

    screen.getByText('No artifacts found.');
  });
});
