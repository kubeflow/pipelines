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
import TensorboardViewer, { TensorboardViewerConfig } from './Tensorboard';
import TestUtils, { diff } from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';
import { ReactWrapper, ShallowWrapper, shallow, mount } from 'enzyme';
import { TFunction } from 'i18next';
import { componentMap } from './ViewerContainer';

const DEFAULT_CONFIG: TensorboardViewerConfig = {
  type: PlotType.TENSORBOARD,
  url: 'http://test/url',
  namespace: 'test-ns',
};

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

describe('Tensorboard', () => {
  let t: TFunction = (key: string) => key;
  let tree: ReactWrapper | ShallowWrapper;
  const flushPromisesAndTimers = async () => {
    jest.runOnlyPendingTimers();
    await TestUtils.flushPromises();
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
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
    const base = tree.debug();

    await TestUtils.flushPromises();
    expect(diff({ base, update: tree.debug() })).toMatchInlineSnapshot(`
      Snapshot Diff:
      - Expected
      + Received

      @@ --- --- @@
                  </WithStyles(MenuItem)>
                </WithStyles(WithFormControlContext(Select))>
              </WithStyles(FormControl)>
            </div>
            <div>
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="common:start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="common:start Tensorboard" />
            </div>
          </div>
        </div>
    `);
  });

  it('does not break on empty data', async () => {
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = { ...DEFAULT_CONFIG, url: '' };
    tree = shallow(<TensorboardViewer configs={[config]} />);
    const base = tree.debug();

    await TestUtils.flushPromises();
    expect(diff({ base, update: tree.debug() })).toMatchInlineSnapshot(`
      Snapshot Diff:
      - Expected
      + Received

      @@ --- --- @@
                  </WithStyles(MenuItem)>
                </WithStyles(WithFormControlContext(Select))>
              </WithStyles(FormControl)>
            </div>
            <div>
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="common:start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="common:start Tensorboard" />
            </div>
          </div>
        </div>
    `);
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const getAppMock = () => Promise.resolve({ podAddress: 'test/address', tfVersion: '1.14.0' });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'isTensorboardPodReady').mockImplementation(() => Promise.resolve(true));
    tree = shallow(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    await flushPromisesAndTimers();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledTimes(1);
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apis/v1beta1/_proxy/test/address');
    expect(tree).toMatchSnapshot();
  });

  it('shows start button if no instance exists', async () => {
    const config = DEFAULT_CONFIG;
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    const getTensorboardSpy = jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[DEFAULT_CONFIG]} />);
    const base = tree.debug();

    await TestUtils.flushPromises();
    expect(
      diff({
        base,
        update: tree.debug(),
        baseAnnotation: 'initial',
        updateAnnotation: 'no instance exists',
      }),
    ).toMatchInlineSnapshot(`
      Snapshot Diff:
      - initial
      + no instance exists

      @@ --- --- @@
                  </WithStyles(MenuItem)>
                </WithStyles(WithFormControlContext(Select))>
              </WithStyles(FormControl)>
            </div>
            <div>
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="common:start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="common:start Tensorboard" />
            </div>
          </div>
        </div>
    `);
    expect(getTensorboardSpy).toHaveBeenCalledWith(config.url, config.namespace);
  });

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { ...DEFAULT_CONFIG };
    const getAppMock = () => Promise.resolve({ podAddress: '', tfVersion: '' });
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    tree = shallow(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.find('BusyButton').simulate('click');
    expect(startAppMock).toHaveBeenCalledWith(config.url, '2.0.0', config.namespace);
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const config2 = { ...DEFAULT_CONFIG, url: 'http://test/url2' };
    const getAppMock = jest.fn(() => Promise.resolve({ podAddress: '', tfVersion: '' }));
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    tree = shallow(<TensorboardViewer configs={[config, config2]} />);
    await TestUtils.flushPromises();
    expect(getAppMock).toHaveBeenCalledWith(
      `Series1:${config.url},Series2:${config2.url}`,
      config.namespace,
    );
    tree.find('BusyButton').simulate('click');
    const expectedUrl = `Series1:${config.url},Series2:${config2.url}`;
    expect(startAppMock).toHaveBeenCalledWith(expectedUrl, '2.0.0', config.namespace);
  });

  it('returns friendly display name', () => {
    expect((componentMap[PlotType.TENSORBOARD].displayNameKey = 'common:tensorboard'));
  });

  it('is aggregatable', () => {
    expect((componentMap[PlotType.TENSORBOARD].isAggregatable = true));
  });

  it('select a version, then start a tensorboard of the corresponding version', async () => {
    const config = { ...DEFAULT_CONFIG };
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
    expect(startAppSpy).toHaveBeenCalledWith(config.url, '1.15.0', config.namespace);
  });

  it('delete the tensorboard instance, confirm in the dialog,\
    then return back to previous page', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const deleteAppMock = jest.fn(() => Promise.resolve(''));
    const deleteAppSpy = jest.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(deleteAppMock);
    const config = { ...DEFAULT_CONFIG };

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
    expect(deleteAppSpy).toHaveBeenCalledWith(config.url, config.namespace);
    await TestUtils.flushPromises();
    tree.update();
    // the tree has returned to 'start tensorboard' page
    expect(tree.findWhere(el => el.text() === 'common:start Tensorboard').exists()).toBeTruthy();
  });

  it('show version info in delete confirming dialog, \
    if a tensorboard instance already exists', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = DEFAULT_CONFIG;
    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.update();
    tree
      .find('#delete')
      .find('Button')
      .simulate('click');
    expect(
      tree.findWhere(el => el.text() === 'common:stopTensorboard 1.14.0?').exists(),
    ).toBeTruthy();
  });

  it('click on cancel on delete tensorboard dialog, then return back to previous page', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = DEFAULT_CONFIG;
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

    expect(tree.findWhere(el => el.text() === 'common:openTensorboard').exists()).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'common:deleteTensorboard').exists()).toBeTruthy();
  });

  it('asks user to wait when Tensorboard status is not ready', async () => {
    const getAppMock = jest.fn(() =>
      Promise.resolve({ podAddress: 'podaddress', tfVersion: '1.14.0' }),
    );
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'isTensorboardPodReady').mockImplementation(() => Promise.resolve(false));
    jest.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(jest.fn(() => Promise.resolve('')));
    const config = DEFAULT_CONFIG;
    tree = mount(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    await flushPromisesAndTimers();
    tree.update();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledTimes(1);
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apis/v1beta1/_proxy/podaddress');
    expect(tree.findWhere(el => el.text() === 'common:openTensorboard').exists()).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'common:tensorboardStarting').exists()).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'common:deleteTensorboard').exists()).toBeTruthy();

    // After a while, it is ready and wait message is not shwon any more
    jest.spyOn(Apis, 'isTensorboardPodReady').mockImplementation(() => Promise.resolve(true));
    await flushPromisesAndTimers();
    tree.update();
    expect(
      tree
        .findWhere(
          el =>
            el.text() === `Tensorboard is starting, and you may need to wait for a few minutes.`,
        )
        .exists(),
    ).toEqual(false);
  });
});
