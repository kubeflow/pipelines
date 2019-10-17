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

import * as React from 'react';

import SideNav, { css } from './SideNav';
import TestUtils from '../TestUtils';
import { Apis } from '../lib/Apis';
import { LocalStorage } from '../lib/LocalStorage';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { RoutePage } from './Router';
import { RouterProps } from 'react-router';

const wideWidth = 1000;
const narrowWidth = 200;
const isCollapsed = (tree: ShallowWrapper<any>) =>
  tree.find('WithStyles(IconButton)').hasClass(css.collapsedChevron);
const routerProps: RouterProps = { history: {} as any };

describe('SideNav', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const consoleErrorSpy = jest.spyOn(console, 'error');
  const buildInfoSpy = jest.spyOn(Apis, 'getBuildInfo');
  const checkHubSpy = jest.spyOn(Apis, 'isJupyterHubAvailable');
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
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders collapsed state', () => {
    localStorageHasKeySpy.mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active page', () => {
    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active when on PipelineDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.PIPELINE_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page', () => {
    tree = shallow(<SideNav page={RoutePage.EXPERIMENTS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active when on ExperimentDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.EXPERIMENT_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewExperiment page', () => {
    tree = shallow(<SideNav page={RoutePage.NEW_EXPERIMENT} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on Compare page', () => {
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on AllRuns page', () => {
    tree = shallow(<SideNav page={RoutePage.RUNS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RunDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.RUN_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RecurringRunDetails page', () => {
    tree = shallow(<SideNav page={RoutePage.RECURRING_RUN} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewRun page', () => {
    tree = shallow(<SideNav page={RoutePage.NEW_RUN} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('show jupyterhub link if accessible', () => {
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    tree.setState({ jupyterHubAvailable: true });
    expect(tree).toMatchSnapshot();
  });

  it('collapses if collapse state is true localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => true);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if collapse state is false in localStorage', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window is too narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window is wide', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window goes from wide to narrow', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
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
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
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
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);

    tree.find('WithStyles(IconButton)').simulate('click');
    expect(spy).toHaveBeenCalledWith(false);
  });

  it('does not collapse if collapse state is saved in localStorage, and window resizes', () => {
    localStorageIsCollapsedSpy.mockImplementationOnce(() => false);
    localStorageHasKeySpy.mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('populates the display build information using the response from the healthz endpoint', async () => {
    const buildInfo = {
      apiServerCommitHash: '0a7b9e38f2b9bcdef4bbf3234d971e1635b50cd5',
      apiServerReady: true,
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();

    expect(tree.state('displayBuildInfo')).toEqual({
      commitHash: buildInfo.apiServerCommitHash.substring(0, 7),
      commitUrl:
        'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.apiServerCommitHash,
      date: new Date(buildInfo.buildDate).toLocaleDateString(),
    });
  });

  it('displays the frontend commit hash if the api server hash is not returned', async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      frontendCommitHash: '302e93ce99099173f387c7e0635476fe1b69ea98',
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitHash: buildInfo.frontendCommitHash.substring(0, 7),
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

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitUrl:
          'https://www.github.com/kubeflow/pipelines/commit/' + buildInfo.frontendCommitHash,
      }),
    );
  });

  it("displays 'unknown' if the frontend and api server commit hashes are not returned", async () => {
    const buildInfo = {
      apiServerReady: true,
      // No apiServerCommitHash
      buildDate: 'Tue Oct 23 14:23:53 UTC 2018',
      // No frontendCommitHash
    };
    buildInfoSpy.mockImplementationOnce(() => buildInfo);

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        commitHash: 'unknown',
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

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
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

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toEqual(
      expect.objectContaining({
        date: 'unknown',
      }),
    );
  });

  it('logs an error if the call getBuildInfo fails', async () => {
    TestUtils.makeErrorResponseOnce(buildInfoSpy, 'Uh oh!');

    tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    await TestUtils.flushPromises();

    expect(tree.state('displayBuildInfo')).toBeUndefined();
    expect(consoleErrorSpy.mock.calls[0][0]).toBe('Failed to retrieve build info');
  });
});
