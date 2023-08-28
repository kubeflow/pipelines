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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { Api } from 'src/mlmd/library';
import {
  Artifact,
  ArtifactType,
  GetArtifactsRequest,
  GetArtifactsResponse,
  GetArtifactTypesResponse,
  Value,
} from 'src/third_party/mlmd';
import { RoutePage } from 'src/components/Router';
import { testBestPractices } from 'src/TestUtils';
import ArtifactListSwitcher from 'src/pages/ArtifactListSwitcher';
import { PageProps } from 'src/pages/Page';
import { CommonTestWrapper } from 'src/TestWrapper';

testBestPractices();

describe('ArtifactListSwitcher', () => {
  let getArtifactsSpy: jest.Mock<{}>;
  let getArtifactTypesSpy: jest.Mock<{}>;
  const getArtifactsRequest = new GetArtifactsRequest();

  beforeEach(() => {
    getArtifactsSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getArtifacts');
    getArtifactTypesSpy = jest.spyOn(Api.getInstance().metadataStoreService, 'getArtifactTypes');

    getArtifactTypesSpy.mockImplementation(() => {
      const artifactType = new ArtifactType();
      artifactType.setId(6);
      artifactType.setName('String');
      const response = new GetArtifactTypesResponse();
      response.setArtifactTypesList([artifactType]);
      return Promise.resolve(response);
    });

    getArtifactsSpy.mockImplementation(() => {
      const artifacts = mockNArtifacts(5);
      const response = new GetArtifactsResponse();
      response.setArtifactsList(artifacts);
      return Promise.resolve(response);
    });
  });

  function generateProps(): PageProps {
    const pageProps: PageProps = {
      history: {} as any,
      location: {
        pathname: RoutePage.EXECUTIONS,
      } as any,
      match: {} as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: () => null,
      updateDialog: () => null,
      updateSnackbar: () => null,
      updateToolbar: () => null,
    };
    return pageProps;
  }

  function mockNArtifacts(n: number) {
    let artifacts: Artifact[] = [];
    for (let i = 1; i <= n; i++) {
      const artifact = new Artifact();
      const pipelineValue = new Value();
      const pipelineName = `pipeline ${i}`;
      pipelineValue.setStringValue(pipelineName);
      artifact.getPropertiesMap().set('pipeline_name', pipelineValue);
      const artifactValue = new Value();
      const artifactName = `test artifact ${i}`;
      artifactValue.setStringValue(artifactName);
      artifact.getPropertiesMap().set('name', artifactValue);
      artifact.setName(artifactName);
      artifacts.push(artifact);
    }
    return artifacts;
  }

  it('shows "Flat" and "Group" tabs', () => {
    render(
      <CommonTestWrapper>
        <ArtifactListSwitcher {...generateProps()} />
      </CommonTestWrapper>,
    );

    screen.getByText('Default');
    screen.getByText('Grouped');
  });

  it('disable pagination if switch to "Group" view', async () => {
    render(
      <CommonTestWrapper>
        <ArtifactListSwitcher {...generateProps()} />
      </CommonTestWrapper>,
    );

    const groupTab = screen.getByText('Grouped');
    fireEvent.click(groupTab);

    // "Group" view will call getExection() without list options
    getArtifactsRequest.setOptions(undefined);

    await waitFor(() => {
      expect(getArtifactTypesSpy).toHaveBeenCalledTimes(2); // once for flat, once for group
      expect(getArtifactsSpy).toHaveBeenCalledTimes(2); // once for flat, once for group
      expect(getArtifactsSpy).toHaveBeenLastCalledWith(getArtifactsRequest);
    });

    expect(screen.queryByText('Rows per page:')).toBeNull(); // no footer
    expect(screen.queryByText('10')).toBeNull(); // no footer
  });
});
