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

import * as React from 'react';
import { mount, ReactWrapper, shallow, ShallowWrapper } from 'enzyme';
import { LineageActionBar, LineageActionBarProps, LineageActionBarState } from './LineageActionBar';
import { buildTestModel, testModel } from './TestUtils';
import { Artifact } from 'src/third_party/mlmd';

describe('LineageActionBar', () => {
  let tree: ShallowWrapper;
  const setLineageViewTarget = jest.fn();

  const mountActionBar = (): ReactWrapper<
    LineageActionBarProps,
    LineageActionBarState,
    LineageActionBar
  > =>
    mount(
      <LineageActionBar initialTarget={testModel} setLineageViewTarget={setLineageViewTarget} />,
    );

  afterEach(() => {
    tree.unmount();
    setLineageViewTarget.mockReset();
  });

  it('Renders correctly for a given initial target', () => {
    tree = shallow(<LineageActionBar initialTarget={testModel} setLineageViewTarget={jest.fn()} />);
    expect(tree).toMatchSnapshot();
  });

  it('Does not update the LineageView target when the current breadcrumb is clicked', () => {
    const tree = mountActionBar();

    // There is one disabled button for the initially selected target.
    expect(tree.state('history').length).toBe(1);

    expect(setLineageViewTarget.mock.calls.length).toBe(0);
    tree
      .find('button')
      .first()
      .simulate('click');
    expect(setLineageViewTarget.mock.calls.length).toBe(0);
    expect(tree.state('history').length).toBe(1);
  });

  it('Updates the LineageView target model when an inactive breadcrumb is clicked', () => {
    const tree = mountActionBar();

    // Add a second artifact to the history state to make the first breadcrumb enabled.
    tree.setState({
      history: [testModel, buildTestModel()],
    });
    expect(tree.state('history').length).toBe(2);
    tree.update();

    expect(setLineageViewTarget.mock.calls.length).toBe(0);
    tree
      .find('button')
      .first()
      .simulate('click');

    expect(setLineageViewTarget.mock.calls.length).toBe(1);
    expect(tree).toMatchSnapshot();
  });

  it('Adds the artifact to the history state and DOM when pushHistory() is called', () => {
    const tree = mountActionBar();
    expect(tree.state('history').length).toBe(1);
    tree.instance().pushHistory(new Artifact());
    expect(tree.state('history').length).toBe(2);
    tree.update();

    expect(tree).toMatchSnapshot();
  });

  it('Sets history to the initial prop when the reset button is clicked', () => {
    const tree = mountActionBar();

    expect(tree.state('history').length).toBe(1);
    tree.instance().pushHistory(buildTestModel());
    tree.instance().pushHistory(buildTestModel());
    expect(tree.state('history').length).toBe(3);

    // Flush state to the DOM
    tree.update();
    const buttonWrappers = tree.find('button');
    expect(buttonWrappers.length).toBe(4);

    // The reset button is the last button on the action bar.
    tree
      .find('button')
      .last()
      .simulate('click');

    expect(tree.state('history').length).toBe(1);
    expect(tree).toMatchSnapshot();
  });
});
