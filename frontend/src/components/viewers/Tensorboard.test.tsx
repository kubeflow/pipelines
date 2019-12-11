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
import { shallow, mount } from 'enzyme';

beforeEach(() => {
  jest.restoreAllMocks();
  jest.clearAllMocks();
});

describe('Tensorboard', () => {
  it('does not break on no config', () => {
    const tree = shallow(<TensorboardViewer configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const config = { type: PlotType.TENSORBOARD, url: '' };
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const mockGetApp = () => Promise.resolve('test/address');
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(mockGetApp);
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    await mockGetApp;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('shows start button if no instance exists', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const defaultVersion = '1.14.0';
    const getAppMock = () => Promise.resolve('');
    const spy = jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(getAppMock);
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    await getAppMock;
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
    expect(spy).toHaveBeenCalledWith(config.url, defaultVersion);
  });

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const getAppMock = () => Promise.resolve('');
    const defaultVersion = '1.14.0';
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    jest.spyOn(tree.instance(), 'setState').mockImplementation((_, cb) => cb && cb());
    await getAppMock;
    tree.find('BusyButton').simulate('click');
    await startAppMock;
    expect(startAppMock).toHaveBeenCalledWith('http%3A%2F%2Ftest%2Furl', defaultVersion);
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const config2 = { type: PlotType.TENSORBOARD, url: 'http://test/url2' };
    const defaultVersion = '1.14.0';
    const getAppMock = jest.fn(() => Promise.resolve(''));
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    const tree = shallow(<TensorboardViewer configs={[config, config2]} />);
    jest.spyOn(tree.instance(), 'setState').mockImplementation((_, cb) => cb && cb());
    await getAppMock;
    expect(getAppMock).toHaveBeenCalledWith(
      `Series1:${config.url},Series2:${config2.url}`,
      defaultVersion,
    );
    tree.find('BusyButton').simulate('click');
    await startAppMock;
    const expectedUrl =
      `Series1${encodeURIComponent(':' + config.url)}` +
      `${encodeURIComponent(',')}` +
      `Series2${encodeURIComponent(':' + config2.url)}`;
    expect(startAppMock).toHaveBeenCalledWith(expectedUrl, defaultVersion);
  });

  it('returns friendly display name', () => {
    expect(TensorboardViewer.prototype.getDisplayName()).toBe('Tensorboard');
  });

  it('is aggregatable', () => {
    expect(TensorboardViewer.prototype.isAggregatable()).toBeTruthy();
  });

  it('set state', async () => {
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const tree = shallow(<TensorboardViewer configs={[config]} />);
    const event = { target: { value: '2.0.0' } } as React.ChangeEvent<{
      name?: string;
      value: unknown;
    }>;

    const onChangeMock = jest.fn(event => {
      tree.setState({ tensorflowVersion: event.target.value });
    });
    jest.spyOn(TensorboardViewer.prototype, 'onChangeFunc').mockImplementation(onChangeMock);
    TensorboardViewer.prototype.onChangeFunc(event);
    expect(onChangeMock).toHaveBeenCalled();
    expect(tree.state('tensorflowVersion')).toEqual('2.0.0');

    // then call the remote api with new state version
    const getAppMock = () => Promise.resolve('');
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    jest.spyOn(tree.instance(), 'setState').mockImplementation((_, cb) => cb && cb());
    await getAppMock;
    tree.find('BusyButton').simulate('click');
    await startAppMock;
    expect(startAppMock).toHaveBeenCalledWith(
      'http%3A%2F%2Ftest%2Furl',
      tree.state('tensorflowVersion'),
    );
  });

  it.only('simulate select', async () => {
    const getAppMock = () => Promise.resolve('');
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementationOnce(getAppMock);
    const config = { type: PlotType.TENSORBOARD, url: 'http://test/url' };
    const spy = jest.spyOn(Apis, 'startTensorboardApp');

    const tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();

    tree
      .find('Select')
      .find('[role="button"]')
      .simulate('click');
    tree
      .findWhere(el => el.text() === 'TensorFlow 2.0.0')
      .hostNodes()
      .simulate('click');
    tree.find('BusyButton').simulate('click');
    expect(spy).toHaveBeenCalledWith('http%3A%2F%2Ftest%2Furl', '2.0.0');
  });
});
