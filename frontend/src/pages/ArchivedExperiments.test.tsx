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
import { ArchivedExperiments } from './ArchivedExperiments';
import TestUtils from '../TestUtils';
import { PageProps } from './Page';
import { ExperimentStorageState } from '../apis/experiment';
import { ShallowWrapper, shallow } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';

describe('ArchivedExperiemnts', () => {
  const updateBannerSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  let tree: ShallowWrapper;

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ArchivedExperiments,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => tree.unmount());

  it('renders archived experiments', () => {
    tree = shallow(<ArchivedExperiments {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('removes error banner on unmount', () => {
    tree = shallow(<ArchivedExperiments {...generateProps()} />);
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('refreshes the experiment list when refresh button is clicked', async () => {
    tree = shallow(<ArchivedExperiments {...generateProps()} />);
    const spy = jest.fn();
    (tree.instance() as any)._experimentlistRef = { current: { refresh: spy } };
    await TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.REFRESH).action();
    expect(spy).toHaveBeenLastCalledWith();
  });

  it('shows a list of archived experiments', () => {
    tree = shallow(<ArchivedExperiments {...generateProps()} />);
    expect(tree.find('ExperimentList').prop('storageState')).toBe(
      ExperimentStorageState.ARCHIVED.toString(),
    );
  });
});
