/*
 * Copyright 2018 Google LLC
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

import { mount, ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import * as React from 'react';
import { MemoryRouter, RouterProps } from 'react-router';
import { GkeMetadataProvider } from 'src/lib/GkeMetadata';
import { Apis } from '../lib/Apis';
import { LocalStorage } from '../lib/LocalStorage';
import TestUtils, { diffHTML } from '../TestUtils';
import { RoutePage } from './Router';
import { css, SideNav } from './SideNav';
import EnhancedSideNav from './SideNav';

const wideWidth = 1000;
const narrowWidth = 200;
const isCollapsed = (tree: ShallowWrapper<any>) =>
  tree.find('WithStyles(IconButton)').hasClass(css.collapsedChevron);
const routerProps: RouterProps = { history: {} as any };
const defaultProps = { ...routerProps, gkeMetadata: {} };

describe('SideNav', () => {
  let tree: ReactWrapper | ShallowWrapper;
  const consoleErrorSpy = jest.spyOn(console, 'error');
  const buildInfoSpy = jest.spyOn(Apis, 'getBuildInfo');
  const checkHubSpy = jest.spyOn(Apis, 'isJupyterHubAvailable');
  const clusterNameSpy = jest.spyOn(Apis, 'getClusterName');
  const projectIdSpy = jest.spyOn(Apis, 'getProjectId');
  const localStorageHasKeySpy = jest.spyOn(LocalStorage, 'hasKey');
  const localStorageIsCollapsedSpy = jest.spyOn(LocalStorage, 'isNavbarCollapsed');

  beforeEach(() => {
    jest.clearAllMocks();

    consoleErrorSpy.mockImplementation(() => null);

    buildInfoSpy.mockImplementation(() => ({
      apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      apiServerReady: true,
      buildDate: 'Wed Jan 9 19:40:24 UTC 2019',
      frontendCommitHash: '8efb2fcff9f666ba5b101647e909dc9c6889cecb',
    }));
    checkHubSpy.mockImplementation(() => ({ ok: true }));
    clusterNameSpy.mockImplementation(() => Promise.reject('Error when fetching cluster name'));
    projectIdSpy.mockImplementation(() => Promise.reject('Error when fetching project ID'));

    localStorageHasKeySpy.mockImplementation(() => false);
    localStorageIsCollapsedSpy.mockImplementation(() => false);
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
    (window as any).innerWidth = wideWidth;
  });

  it('renders expanded state', () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders collapsed state', () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active page', () => {
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active when on PipelineDetails page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.PIPELINE_DETAILS} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.EXPERIMENTS} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active when on ExperimentDetails page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.EXPERIMENT_DETAILS} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewExperiment page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.NEW_EXPERIMENT} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on Compare page', () => {
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on AllRuns page', () => {
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.RUNS} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RunDetails page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.RUN_DETAILS} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RecurringRunDetails page', () => {
    tree = shallow(
      <SideNav t={(key: any) => key} page={RoutePage.RECURRING_RUN} {...defaultProps} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewRun page', () => {
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.NEW_RUN} {...defaultProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('show jupyterhub link if accessible', () => {
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    tree.setState({ jupyterHubAvailable: true });
    expect(tree).toMatchSnapshot();
  });

  it('collapses if collapse state is true localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => true);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if collapse state is false in localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window is too narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window is wide', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window goes from wide to narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window goes from narrow to wide', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);

    (window as any).innerWidth = wideWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('saves state in localStorage if chevron is clicked', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    const spy = jest.spyOn(LocalStorage, 'saveNavbarCollapsed');

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(true);

    tree.find('WithStyles(IconButton)').simulate('click');
    expect(spy).toHaveBeenCalledWith(false);
  });

  it('does not collapse if collapse state is saved in localStorage, and window resizes', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.COMPARE} {...defaultProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('populates the display build information using the response from the healthz endpoint', async () => {
    const buildInfo = {
      apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
      apiServerTagName: '1.0.0',
      apiServerReady: true,
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      frontendTagName: '1.0.0-rc1',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();

    expect(tree.state('displayBuildInfo')).toEqual({
      tagName: buildInfo.apiServerTagName,
      commitHash: buildInfo.apiServerCommitHash.substring(0, 7),
      commitUrl:
        'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.apiServerCommitHash,
      date: new Date(buildInfo.buildDate).toLocaleDateString(),
    });
  });

  it('populates the cluster information from context', async () => {
    const clusterName = 'some-cluster-name';
    const projectId = 'some-project-id';

    clusterNameSpy.mockImplementationOnce(() => Promise.resolve(clusterName));
    projectIdSpy.mockImplementationOnce(() => Promise.resolve(projectId));
    buildInfoSpy.mockImplementationOnce(() => Promise.reject('Error when fetching build info'));

    tree = mount(
      <GkeMetadataProvider>
        <MemoryRouter>
          <EnhancedSideNav page={RoutePage.PIPELINES} {...routerProps} />
        </MemoryRouter>
      </GkeMetadataProvider>,
    );
    const base = tree.html();
    await TestUtils.flushPromises();
    expect(
      diffHTML({
        base,
        baseAnnotation: 'base',
        update: tree.html(),
        updateAnnotation: 'after GKE metadata loaded',
      }),
    ).toMatchInlineSnapshot(`
      Snapshot Diff:
      - base
      + after GKE metadata loaded

      @@ --- --- @@
                  <path fill="none" d="M0 0h24v24H0z"></path></svg></span
              ><span class="MuiTouchRipple-root-53"></span>
            </button>
          </div>
          <div class="infoVisible">
      +     <div
      +       class="envMetadata"
      +       title="common:clusterName: some-cluster-name, common:projectId: some-project-id"
      +     >
      +       <span>common:clusterName: </span
      +       ><a
      +         href="https://console.cloud.google.com/kubernetes/list?project=some-project-id&amp;filter=name:some-cluster-name"
      +         class="link unstyled"
      +         rel="noopener"
      +         target="_blank"
      +         >some-cluster-name</a
      +       >
      +     </div>
            <div class="envMetadata" title="common:reportIssue">
              <a
                href="https://github.com/kubeflow/pipelines/issues/new/choose"
                class="link unstyled"
                rel="noopener"
    `);
  });
  it('displays the frontend tag name if the api server hash is not returned', async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash or apiServerTagName
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
      frontendTagName: '1.0.0',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitHash: buildInfo.frontendCommitHash.substring(0, 7),
        tagName: buildInfo.frontendTagName,
      }),
    );
  });

  it('uses the frontend commit hash for the link URL if the api server hash is not returned', async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitUrl:
          'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.frontendCommitHash,
      }),
    );
  });

  it("displays 'unknown' if the frontend and api server tag names/commit hashes are not returned", async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      // No frontendCommitHash
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitHash: 'unknown',
        tagName: 'unknown',
      }),
    );
  });

  it('links to the github repo root if the frontend and api server commit hashes are not returned', async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      // No frontendCommitHash
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitUrl: 'https://www.github.com/kubeflow/pipelines',
      }),
    );
  });

  it("displays 'unknown' if the date is not returned", async () => {
    const buildInfo = {
      apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
      apiServerReady: true,
      // No buildDate
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        date: 'unknown',
      }),
    );
  });

  it('logs an error if the call getBuildInfo fails', async () => {
    TestUtils.makeErrorResponseOnce(buildInfoSpy, 'Uh oh!');

    tree = shallow(<SideNav t={(key: any) => key} page={RoutePage.PIPELINES} {...defaultProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toBeUndefined();
    expect(consoleErrorSpy.mock.calls[0][0]).toBe('Failed to retrieve build info');
  });
});
