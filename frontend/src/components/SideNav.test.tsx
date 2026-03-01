/*
 * Copyright 2018 The Kubeflow Authors
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
import { act, fireEvent, render, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { createMemoryHistory } from 'history';
import { vi } from 'vitest';
import { LocalStorage, LocalStorageKey } from '../lib/LocalStorage';
import { RoutePage } from './Router';
import { css, SideNav } from './SideNav';
import { GkeMetadata } from '../lib/GkeMetadata';

const wideWidth = 1000;
const narrowWidth = 200;

const defaultProps = {
  gkeMetadata: {},
  buildInfo: {
    apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
    apiServerTagName: '1.0.0',
    apiServerReady: true,
    buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
    frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    frontendTagName: '1.0.0-rc1',
  },
};

function isCollapsed(container: HTMLElement): boolean {
  const root = container.querySelector('#sideNav');
  return !!root && root.classList.contains(css.collapsedRoot);
}

function renderSideNav(
  page: RoutePage,
  overrides: Partial<React.ComponentProps<typeof SideNav>> = {},
) {
  const history = createMemoryHistory();
  const ref = React.createRef<SideNav>();
  const props = {
    ...defaultProps,
    ...overrides,
    history,
    page,
  } as React.ComponentProps<typeof SideNav>;
  const renderResult = render(
    <MemoryRouter>
      <SideNav ref={ref} {...props} />
    </MemoryRouter>,
  );
  if (!ref.current) {
    throw new Error('SideNav instance not available');
  }
  return { renderResult, instance: ref.current };
}

describe('SideNav', () => {
  let localStorageHasKeySpy: ReturnType<typeof vi.spyOn>;
  let localStorageIsCollapsedSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    localStorageHasKeySpy = vi.spyOn(LocalStorage, 'hasKey').mockImplementation(() => false);
    localStorageIsCollapsedSpy = vi
      .spyOn(LocalStorage, 'isNavbarCollapsed')
      .mockImplementation(() => false);
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    (window as any).innerWidth = wideWidth;
  });

  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
    (window as any).innerWidth = wideWidth;
  });

  it('renders expanded state', async () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    const { renderResult } = renderSideNav(RoutePage.PIPELINES);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders collapsed state', async () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    const { renderResult } = renderSideNav(RoutePage.PIPELINES);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders Pipelines as active page', async () => {
    const { renderResult } = renderSideNav(RoutePage.PIPELINES);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders Pipelines as active when on PipelineDetails page', async () => {
    const { renderResult } = renderSideNav(RoutePage.PIPELINE_DETAILS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page', async () => {
    const { renderResult } = renderSideNav(RoutePage.EXPERIMENTS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active when on ExperimentDetails page', async () => {
    const { renderResult } = renderSideNav(RoutePage.EXPERIMENT_DETAILS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewExperiment page', async () => {
    const { renderResult } = renderSideNav(RoutePage.NEW_EXPERIMENT);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on Compare page', async () => {
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on AllRuns page', async () => {
    const { renderResult } = renderSideNav(RoutePage.RUNS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on RunDetails page', async () => {
    const { renderResult } = renderSideNav(RoutePage.RUN_DETAILS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on RecurringRunDetails page', async () => {
    const { renderResult } = renderSideNav(RoutePage.RECURRING_RUN_DETAILS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewRun page', async () => {
    const { renderResult } = renderSideNav(RoutePage.NEW_RUN);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders recurring runs as active page', async () => {
    const { renderResult } = renderSideNav(RoutePage.RECURRING_RUNS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('renders jobs as active page when on JobDetails page', async () => {
    const { renderResult } = renderSideNav(RoutePage.RECURRING_RUN_DETAILS);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('show jupyterhub link if accessible', () => {
    const { renderResult, instance } = renderSideNav(RoutePage.COMPARE);
    act(() => {
      instance.setState({ jupyterHubAvailable: true });
    });
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('collapses if collapse state is true localStorage', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => true);
    localStorageHasKeySpy.mockImplementationOnce(() => true);
    (window as any).innerWidth = wideWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));
  });

  it('expands if collapse state is false in localStorage', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
  });

  it('collapses if no collapse state in localStorage, and window is too narrow', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));
  });

  it('expands if no collapse state in localStorage, and window is wide', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
  });

  it('collapses if no collapse state in localStorage, and window goes from wide to narrow', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));

    act(() => {
      (window as any).innerWidth = narrowWidth;
      window.dispatchEvent(new Event('resize'));
    });
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));
  });

  it('expands if no collapse state in localStorage, and window goes from narrow to wide', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));

    act(() => {
      (window as any).innerWidth = wideWidth;
      window.dispatchEvent(new Event('resize'));
    });
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
  });

  it('saves state in localStorage if chevron is clicked', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    const spy = vi.spyOn(LocalStorage, 'saveNavbarCollapsed');

    (window as any).innerWidth = narrowWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));

    const chevron = renderResult.container.querySelector(`.${css.chevron}`);
    if (!chevron) {
      throw new Error('Chevron button not found');
    }
    fireEvent.click(chevron);
    expect(spy).toHaveBeenCalledWith(false);
  });

  it('does not collapse if collapse state is saved in localStorage, and window resizes', async () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));

    act(() => {
      (window as any).innerWidth = narrowWidth;
      window.dispatchEvent(new Event('resize'));
    });
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(false));
  });

  it('populates the display build information using the default props', () => {
    const { instance, renderResult } = renderSideNav(RoutePage.PIPELINES);
    expect(renderResult.asFragment()).toMatchSnapshot();

    const buildInfo = defaultProps.buildInfo;
    expect((instance as any)._getBuildInfo()).toEqual({
      tagName: buildInfo.apiServerTagName,
      commitHash: buildInfo.apiServerCommitHash.substring(0, 7),
      commitUrl:
        'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.apiServerCommitHash,
      date: new Date(buildInfo.buildDate).toLocaleDateString(),
    });
  });

  it('display the correct GKE metadata', () => {
    const clusterName = 'some-cluster-name';
    const projectId = 'some-project-id';
    const gkeMetadata: GkeMetadata = { clusterName, projectId };

    const newProps = {
      ...defaultProps,
      gkeMetadata,
      buildInfo: {},
    };

    const { renderResult } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('displays the frontend tag name if the api server hash is not returned', () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
        frontendTagName: '1.0.0',
      },
    };

    const { instance } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect((instance as any)._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitHash: newProps.buildInfo.frontendCommitHash.substring(0, 7),
        tagName: newProps.buildInfo.frontendTagName,
      }),
    );
  });

  it('uses the frontend commit hash for the link URL if the api server hash is not returned', () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      },
    };

    const { instance } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect((instance as any)._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitUrl:
          'https://www.github.com/kubeflow/pipelines/commit/' +
          newProps.buildInfo.frontendCommitHash,
      }),
    );
  });

  it("displays 'unknown' if the frontend and api server tag names/commit hashes are not returned", () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      },
    };

    const { instance } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect((instance as any)._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitHash: 'unknown',
        tagName: 'unknown',
      }),
    );
  });

  it('links to the github repo root if the frontend and api server commit hashes are not returned', () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerReady: true,
        buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      },
    };

    const { instance } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect((instance as any)._getBuildInfo()).toEqual(
      expect.objectContaining({
        commitUrl: 'https://www.github.com/kubeflow/pipelines',
      }),
    );
  });

  it("displays 'unknown' if the date is not returned", () => {
    const newProps = {
      ...defaultProps,
      buildInfo: {
        apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
        apiServerReady: true,
        frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      },
    };

    const { instance } = renderSideNav(RoutePage.PIPELINES, newProps);
    expect((instance as any)._getBuildInfo()).toEqual(
      expect.objectContaining({
        date: 'unknown',
      }),
    );
  });

  it('auto-collapses on first load when localStorage key is missing and window is narrow', async () => {
    localStorageHasKeySpy.mockRestore();
    localStorageIsCollapsedSpy.mockRestore();
    localStorage.removeItem(LocalStorageKey.navbarCollapsed);
    (window as any).innerWidth = narrowWidth;

    const { renderResult } = renderSideNav(RoutePage.COMPARE);
    await waitFor(() => expect(isCollapsed(renderResult.container)).toBe(true));
  });
});
