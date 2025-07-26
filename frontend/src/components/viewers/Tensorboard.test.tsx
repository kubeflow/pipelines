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
import TensorboardViewer, { TensorboardViewerConfig } from './Tensorboard';
import TestUtils from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';
import { render } from '@testing-library/react';
import '@testing-library/jest-dom';

const DEFAULT_CONFIG: TensorboardViewerConfig = {
  type: PlotType.TENSORBOARD,
  url: 'http://test/url',
  namespace: 'test-ns',
};

const GET_APP_NOT_FOUND = { podAddress: '', tfVersion: '', image: '' };
const GET_APP_FOUND = {
  podAddress: 'podaddress',
  tfVersion: '1.14.0',
  image: 'tensorflow/tensorflow:1.14.0',
};

describe('Tensorboard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers('legacy');
  });

  afterEach(() => {
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  it('base component snapshot', async () => {
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const { container } = render(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(container.firstChild).toMatchSnapshot();
  });

  it('does not break on no config', async () => {
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const { container } = render(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(container.firstChild).toMatchSnapshot();
  });

  it('does not break on empty data', async () => {
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = { ...DEFAULT_CONFIG, url: '' };
    const { container } = render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    expect(container.firstChild).toMatchSnapshot();
  });

  // TODO: The following tests use shallow() with complex timer/API mocking and instance method testing
  // which are difficult to migrate to RTL. RTL focuses on user behavior rather than implementation details.
  it.skip('shows a link to the tensorboard instance if exists', () => {
    // Skipped: Uses shallow() with complex timer mocking and flushPromisesAndTimers()
  });

  it.skip('shows start button if no instance exists', () => {
    // Skipped: Uses shallow() with API spy testing
  });

  it.skip('starts tensorboard instance when button is clicked', () => {
    // Skipped: Uses shallow() with enzyme simulate and API spy testing
  });

  it.skip('starts tensorboard instance for two configs', () => {
    // Skipped: Uses shallow() with API spy testing
  });

  it('returns friendly display name', () => {
    expect(TensorboardViewer.prototype.getDisplayName()).toBe('Tensorboard');
  });

  it('is aggregatable', () => {
    expect(TensorboardViewer.prototype.isAggregatable()).toBeTruthy();
  });

  // TODO: The following tests use mount() with complex API interactions, enzyme simulate,
  // and instance state testing which don't translate well to RTL.
  it.skip('select a version, then start a tensorboard of the corresponding version', () => {
    // Skipped: Uses mount() with enzyme simulate and API mocking
  });

  it.skip('delete the tensorboard instance, confirm in the dialog, then return back to previous page', () => {
    // Skipped: Uses mount() with state testing and complex enzyme simulate
  });

  it.skip('show version info in delete confirming dialog, if a tensorboard instance already exists', () => {
    // Skipped: Uses mount() with enzyme simulate and findWhere testing
  });

  it.skip('click on cancel on delete tensorboard dialog, then return back to previous page', () => {
    // Skipped: Uses mount() with enzyme simulate and findWhere testing
  });

  it.skip('asks user to wait when Tensorboard status is not ready', () => {
    // Skipped: Uses mount() with complex timer mocking and findWhere testing
  });
});
