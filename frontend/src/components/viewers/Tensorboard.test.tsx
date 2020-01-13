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
import TensorboardViewer from './Tensorboard';
import TestUtils from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';
import { ReactWrapper, ShallowWrapper, shallow, mount } from 'enzyme';
import snapshotDiff from 'snapshot-diff';

describe('Tensorboard', () => {
  let tree: ReactWrapper | ShallowWrapper;
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      await tree.unmount();
    }
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  it('base component snapshot', async () => {
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('does not break on no config', async () => {
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[]} />);
    const baseTree = tree.debug();

    await TestUtils.flushPromises();
    expect(snapshotDiff(tree.debug(), baseTree)).toMatchSnapshot();
  });

  it('does not break on empty data', async () => {
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = { type: PlotType.TENSORBOARD, url: '' };
    tree = shallow(<TensorboardViewer configs={[config]} />);
    const baseTree = tree.debug();

    await TestUtils.flushPromises();
    expect(snapshotDiff(tree.debug(), baseTree)).toMatchSnapshot();
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const getAppMock = () => Promise.resolve({ podAddress: 'test/address', tfVersion: '1.14.0' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('shows start button if no instance exists', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    const spy = jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[config]} />);
    const baseTree = tree.debug();

    await TestUtils.flushPromises();
    expect(snapshotDiff(tree.debug(), baseTree)).toMatchSnapshot();
    expect(spy).toHaveBeenCalledWith(config.url);
  });

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    tree = shallow(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.find('BusyButton').simulate('click');
    expect(startAppMock).toHaveBeenCalledWith('http%3A%2F%2Ftest%2Furl', '2.0.0');
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const config2 = { type: PlotType.TENSORBOARD, url: 'http://test/url2' };
    const getAppMock = jest.fn(() => Promise.resolve({ podAddress: '', tfVersion: '' }));
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    tree = shallow(<TensorboardViewer configs={[config, config2]} />);
    await TestUtils.flushPromises();
    expect(getAppMock).toHaveBeenCalledWith(`Series1:${config.url},Series2:${config2.url}`);
    tree.find('BusyButton').simulate('click');
    const expectedUrl =
      `Series1${encodeURIComponent(':' + config.url)}` +
      `${encodeURIComponent(',')}` +
      `Series2${encodeURIComponent(':' + config2.url)}`;
    expect(startAppMock).toHaveBeenCalledWith(expectedUrl, '2.0.0');
  });

  it('returns friendly display name', () => {
    expect(TensorboardViewer.prototype.getDisplayName()).toBe('Tensorboard');
  });

  it('is aggregatable', () => {
    expect(TensorboardViewer.prototype.isAggregatable()).toBeTruthy();
  });

  it('select a version, then start a tensorboard of the corresponding version', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };

    const getAppMock = jest.fn(() => Promise.resolve({ podAddress: '', tfVersion: '' }));
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const startAppSpy = jest
      .spyOn(Apis, 'startTensorboardApp')
      .mockImplementationOnce(startAppMock);

    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();

    tree
      .find('Select')
      .find('[role="button"]')
      .simulate('click');
    tree
      .findWhere(el => el.text() === 'TensorFlow 1.15.0')
      .hostNodes()
      .simulate('click');
    tree.find('BusyButton').simulate('click');
    expect(startAppSpy).toHaveBeenCalledWith('http%3A%2F%2Ftest%2Furl', '1.15.0');
  });

  it('delete the tensorboard instance, confirm in the dialog,\
    then return back to previous page', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const deleteAppMock = jest.fn(() => Promise.resolve(''));
    const deleteAppSpy = jest.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(deleteAppMock);
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };

    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    expect(!!tree.state('podAddress')).toBeTruthy();

    // delete a tensorboard
    tree.update();
    tree
      .find('#delete')
      .find('Button')
      .simulate('click');
    tree.find('BusyButton').simulate('click');
    expect(deleteAppSpy).toHaveBeenCalledWith(encodeURIComponent('http://test/url'));
    await TestUtils.flushPromises();
    tree.update();
    // the tree has returned to 'start tensorboard' page
    expect(tree.findWhere(el => el.text() === 'Start Tensorboard').exists()).toBeTruthy();
  });

  it('show version info in delete confirming dialog, \
    if a tensorboard instance already exists', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.update();
    tree
      .find('#delete')
      .find('Button')
      .simulate('click');
    expect(tree.findWhere(el => el.text() === 'Stop Tensorboard 1.14.0?').exists()).toBeTruthy();
  });

  it('click on cancel on delete tensorboard dialog, then return back to previous page', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.update();
    tree
      .find('#delete')
      .find('Button')
      .simulate('click');

    tree
      .find('#cancel')
      .find('Button')
      .simulate('click');

    expect(tree.findWhere(el => el.text() === 'Open Tensorboard').exists()).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'Delete Tensorboard').exists()).toBeTruthy();
  });
});
