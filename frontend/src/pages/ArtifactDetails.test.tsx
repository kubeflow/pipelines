/*
 * Copyright 2026 The Kubeflow Authors
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
import { createMemoryHistory } from 'history';
import { MemoryRouter, Route, Router } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, Mock } from 'vitest';
import { Api } from 'src/mlmd/library';
import { Artifact, ArtifactType, GetArtifactsByIDResponse, Value } from 'src/third_party/mlmd';
import { GetArtifactTypesByIDResponse } from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { RoutePage, RouteParams } from 'src/components/Router';
import { testBestPractices } from 'src/TestUtils';
import EnhancedArtifactDetails from 'src/pages/ArtifactDetails';
import { PageProps } from 'src/pages/Page';

vi.mock('src/mlmd/LineageView', () => ({
  LineageView: () => <div data-testid='lineage-view'>LineageView Mock</div>,
}));

testBestPractices();

describe('ArtifactDetails', () => {
  let updateBannerSpy: Mock;
  let updateToolbarSpy: Mock;
  let historyPushSpy: Mock;
  let getArtifactsByIDSpy: Mock;
  let getArtifactTypesByIDSpy: Mock;

  const TEST_ARTIFACT_ID = 42;

  function buildArtifact(id = TEST_ARTIFACT_ID, name = 'test-artifact'): Artifact {
    const artifact = new Artifact();
    artifact.setId(id);
    artifact.setTypeId(7);
    const nameValue = new Value();
    nameValue.setStringValue(name);
    artifact.getPropertiesMap().set('name', nameValue);
    return artifact;
  }

  function buildArtifactType(): ArtifactType {
    const artifactType = new ArtifactType();
    artifactType.setId(7);
    artifactType.setName('system/Dataset');
    return artifactType;
  }

  function buildGetArtifactsByIDResponse(artifacts: Artifact[]): GetArtifactsByIDResponse {
    const response = new GetArtifactsByIDResponse();
    response.setArtifactsList(artifacts);
    return response;
  }

  function buildGetArtifactTypesByIDResponse(types: ArtifactType[]): GetArtifactTypesByIDResponse {
    const response = new GetArtifactTypesByIDResponse();
    response.setArtifactTypesList(types);
    return response;
  }

  function generateProps(artifactId = TEST_ARTIFACT_ID): PageProps {
    const match = {
      isExact: true,
      path: RoutePage.ARTIFACT_DETAILS,
      url: `/artifacts/${artifactId}`,
      params: { [RouteParams.ID]: String(artifactId) },
    } as any;
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: `/artifacts/${artifactId}` } as any,
      match,
      toolbarProps: {
        actions: {},
        breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
        pageTitle: `Artifact #${artifactId}`,
      },
      updateBanner: updateBannerSpy,
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: updateToolbarSpy,
    } as any;
  }

  function renderWithRouter(props: PageProps, initialPath?: string) {
    const path = initialPath || `/artifacts/${props.match.params[RouteParams.ID]}`;
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[path]}>
          <Route path={['/artifacts/:id/lineage', '/artifacts/:id']}>
            {(routeProps: any) => (
              <EnhancedArtifactDetails
                {...routeProps}
                {...props}
                match={{ ...routeProps.match, ...props.match }}
              />
            )}
          </Route>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  }

  beforeEach(() => {
    updateBannerSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    getArtifactsByIDSpy = vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByID');
    getArtifactTypesByIDSpy = vi.spyOn(
      Api.getInstance().metadataStoreService,
      'getArtifactTypesByID',
    );
  });

  function mockSuccessfulLoad(artifact?: Artifact) {
    const a = artifact || buildArtifact();
    getArtifactsByIDSpy.mockResolvedValue(buildGetArtifactsByIDResponse([a]));
    getArtifactTypesByIDSpy.mockResolvedValue(
      buildGetArtifactTypesByIDResponse([buildArtifactType()]),
    );
  }

  it('shows CircularProgress spinner while artifact is loading', () => {
    getArtifactsByIDSpy.mockReturnValue(new Promise(() => {}));

    renderWithRouter(generateProps());

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('renders Overview tab with ResourceInfo after artifact loads', async () => {
    mockSuccessfulLoad();

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });
    expect(screen.getByText('Lineage Explorer')).toBeInTheDocument();
  });

  it('updates toolbar title with artifact name after load', async () => {
    mockSuccessfulLoad();

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          pageTitle: expect.stringContaining('test-artifact'),
        }),
      );
    });
  });

  it('shows page error banner when no artifact found for the given ID', async () => {
    getArtifactsByIDSpy.mockResolvedValue(buildGetArtifactsByIDResponse([]));

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining(`No artifact identified by id: ${TEST_ARTIFACT_ID}`),
          mode: 'error',
        }),
      );
    });
  });

  it('shows page error banner when multiple artifacts found for the given ID', async () => {
    getArtifactsByIDSpy.mockResolvedValue(
      buildGetArtifactsByIDResponse([buildArtifact(), buildArtifact()]),
    );

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining(`Found multiple artifacts with ID: ${TEST_ARTIFACT_ID}`),
          mode: 'error',
        }),
      );
    });
  });

  it('shows page error banner on service error', async () => {
    getArtifactsByIDSpy.mockRejectedValue({ message: 'Service unavailable' });

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining('Service unavailable'),
          mode: 'error',
        }),
      );
    });
  });

  it('shows fallback error message when a non-service error with no message is thrown', async () => {
    getArtifactsByIDSpy.mockRejectedValue(undefined);

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: 'Error: failed to load artifact.',
          mode: 'error',
        }),
      );
    });
  });

  it('shows extracted error message when a non-service error with text() is thrown', async () => {
    getArtifactsByIDSpy.mockRejectedValue({ text: () => 'detailed failure info' });

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          message: 'Error: detailed failure info',
          mode: 'error',
        }),
      );
    });
  });

  it('renders a new ArtifactDetails instance when artifact ID in URL changes', async () => {
    getArtifactsByIDSpy.mockImplementation((req) => {
      const id = req.getArtifactIdsList()[0];
      if (id === 1) {
        return Promise.resolve(buildGetArtifactsByIDResponse([buildArtifact(1, 'artifact-one')]));
      }
      if (id === 2) {
        return Promise.resolve(buildGetArtifactsByIDResponse([buildArtifact(2, 'artifact-two')]));
      }
      return Promise.resolve(buildGetArtifactsByIDResponse([]));
    });
    getArtifactTypesByIDSpy.mockResolvedValue(
      buildGetArtifactTypesByIDResponse([buildArtifactType()]),
    );

    const history = createMemoryHistory({ initialEntries: ['/artifacts/1'] });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <Router history={history}>
          <Route path={['/artifacts/:id/lineage', '/artifacts/:id']}>
            {(routeProps: any) => (
              <EnhancedArtifactDetails
                {...routeProps}
                updateBanner={updateBannerSpy}
                updateDialog={vi.fn()}
                updateSnackbar={vi.fn()}
                updateToolbar={updateToolbarSpy}
                toolbarProps={{
                  actions: {},
                  breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
                  pageTitle: '',
                }}
              />
            )}
          </Route>
        </Router>
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({ pageTitle: expect.stringContaining('artifact-one') }),
      );
    });

    updateToolbarSpy.mockClear();
    history.push('/artifacts/2');

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({ pageTitle: expect.stringContaining('artifact-two') }),
      );
    });
  });

  it('renders with empty type name when artifact type list is empty', async () => {
    getArtifactsByIDSpy.mockResolvedValue(buildGetArtifactsByIDResponse([buildArtifact()]));
    getArtifactTypesByIDSpy.mockResolvedValue(buildGetArtifactTypesByIDResponse([]));

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });
    expect(screen.queryByText('Dataset')).not.toBeInTheDocument();
  });

  it('includes version in toolbar title when artifact has a version property', async () => {
    const artifact = buildArtifact();
    const versionValue = new Value();
    versionValue.setStringValue('v1.2');
    artifact.getPropertiesMap().set('version', versionValue);

    mockSuccessfulLoad(artifact);

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          pageTitle: expect.stringContaining('(version: v1.2)'),
        }),
      );
    });
  });

  it('navigates to lineage URL when Lineage Explorer tab is clicked', async () => {
    mockSuccessfulLoad();

    renderWithRouter(generateProps());

    await waitFor(() => {
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Lineage Explorer'));

    expect(historyPushSpy).toHaveBeenCalledWith(expect.stringContaining('/lineage'));
  });

  it('navigates back to overview URL when Overview tab is clicked', async () => {
    mockSuccessfulLoad();

    renderWithRouter(generateProps(), `/artifacts/${TEST_ARTIFACT_ID}/lineage`);

    await waitFor(() => {
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Overview'));

    expect(historyPushSpy).toHaveBeenCalledWith(expect.stringMatching(/\/artifacts\/42$/));
  });
});
