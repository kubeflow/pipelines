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
import { shallow, ShallowWrapper } from 'enzyme';
import { RoutePage } from './Router';
import { LocalStorage } from '../lib/LocalStorage';
import { RouterProps } from 'react-router';

const wideWidth = 1000;
const narrowWidth = 200;
const isCollapsed = (tree: ShallowWrapper<any>) =>
  tree.find('WithStyles(IconButton)').hasClass(css.collapsedChevron);
const routerProps: RouterProps = { history: {} as any };

describe('SideNav', () => {
  it('renders expanded state', () => {
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);
    (window as any).innerWidth = wideWidth;
    const tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders collapsed state', () => {
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);
    (window as any).innerWidth = narrowWidth;
    const tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active page', () => {
    const tree = shallow(<SideNav page={RoutePage.PIPELINES} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders Pipelines as active when on PipelineDetails page', () => {
    const tree = shallow(<SideNav page={RoutePage.PIPELINE_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page', () => {
    const tree = shallow(<SideNav page={RoutePage.EXPERIMENTS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active when on ExperimentDetails page', () => {
    const tree = shallow(<SideNav page={RoutePage.EXPERIMENT_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewExperiment page', () => {
    const tree = shallow(<SideNav page={RoutePage.NEW_EXPERIMENT} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on Compare page', () => {
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on AllRuns page', () => {
    const tree = shallow(<SideNav page={RoutePage.RUNS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RunDetails page', () => {
    const tree = shallow(<SideNav page={RoutePage.RUN_DETAILS} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on RecurringRunDetails page', () => {
    const tree = shallow(<SideNav page={RoutePage.RECURRING_RUN} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders experiments as active page when on NewRun page', () => {
    const tree = shallow(<SideNav page={RoutePage.NEW_RUN} {...routerProps} />);
    expect(tree).toMatchSnapshot();
  });

  it('show jupyterhub link if accessible', () => {
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    tree.setState({ jupyterHubAvailable: true });
    expect(tree).toMatchSnapshot();
  });

  it('collapses if collapse state is true localStorage', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => true);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if collapse state is false in localStorage', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => true);

    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window is too narrow', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window is wide', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('collapses if no collapse state in localStorage, and window goes from wide to narrow', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);

    (window as any).innerWidth = wideWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(true);
  });

  it('expands if no collapse state in localStorage, and window goes from narrow to wide', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);

    (window as any).innerWidth = narrowWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);

    (window as any).innerWidth = wideWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });

  it('saves state in localStorage if chevron is clicked', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => false);
    const spy = jest.spyOn(LocalStorage, 'saveNavbarCollapsed');

    (window as any).innerWidth = narrowWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(true);

    tree.find('WithStyles(IconButton)').simulate('click');
    expect(spy).toHaveBeenCalledWith(false);
  });

  it('does not collapse if collapse state is saved in localStorage, and window resizes', () => {
    jest.spyOn(LocalStorage, 'isNavbarCollapsed').mockImplementationOnce(() => false);
    jest.spyOn(LocalStorage, 'hasKey').mockImplementationOnce(() => true);

    (window as any).innerWidth = wideWidth;
    const tree = shallow(<SideNav page={RoutePage.COMPARE} {...routerProps} />);
    expect(isCollapsed(tree)).toBe(false);

    (window as any).innerWidth = narrowWidth;
    const resizeEvent = new Event('resize');
    window.dispatchEvent(resizeEvent);
    expect(isCollapsed(tree)).toBe(false);
  });
});
