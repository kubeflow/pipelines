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
import TestUtils, { diff } from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';
import { ReactWrapper, ShallowWrapper, shallow, mount } from 'enzyme';

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

describe.only('Tensorboard', () => {
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
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    tree = shallow(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            className=""
          >
            <WithStyles(FormControl)
              className="formControl"
            >
              <WithStyles(WithFormControlContext(InputLabel))
                htmlFor="viewer-tb-image-select"
              >
                TF Image
              </WithStyles(WithFormControlContext(InputLabel))>
              <WithStyles(WithFormControlContext(Select))
                className="select"
                input={
                  <WithStyles(Input)
                    id="viewer-tb-image-select"
                  />
                }
                onChange={[Function]}
                value="tensorflow/tensorflow:2.2.2"
              >
                <WithStyles(ListSubheader)>
                  Tensoflow 1.x
                </WithStyles(ListSubheader)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.7.1"
                >
                  TensorFlow 1.7.1
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.8.0"
                >
                  TensorFlow 1.8.0
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.9.0"
                >
                  TensorFlow 1.9.0
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.10.1"
                >
                  TensorFlow 1.10.1
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.11.0"
                >
                  TensorFlow 1.11.0
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.12.3"
                >
                  TensorFlow 1.12.3
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.13.2"
                >
                  TensorFlow 1.13.2
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.14.0"
                >
                  TensorFlow 1.14.0
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:1.15.5"
                >
                  TensorFlow 1.15.5
                </WithStyles(MenuItem)>
                <WithStyles(ListSubheader)>
                  TensorFlow 2.x
                </WithStyles(ListSubheader)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:2.0.4"
                >
                  TensorFlow 2.0.4
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:2.1.2"
                >
                  TensorFlow 2.1.2
                </WithStyles(MenuItem)>
                <WithStyles(MenuItem)
                  value="tensorflow/tensorflow:2.2.2"
                >
                  TensorFlow 2.2.2
                </WithStyles(MenuItem)>
              </WithStyles(WithFormControlContext(Select))>
            </WithStyles(FormControl)>
          </div>
          <div>
            <BusyButton
              busy={false}
              className="buttonAction"
              disabled={false}
              onClick={[Function]}
              title="Start Tensorboard"
            />
          </div>
        </div>
      </div>
    `);
  });

  it('does not break on no config', async () => {
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
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
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="Start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="Start Tensorboard" />
            </div>
          </div>
        </div>
    `);
  });

  it('does not break on empty data', async () => {
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
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
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="Start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="Start Tensorboard" />
            </div>
          </div>
        </div>
    `);
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const getAppMock = () =>
      Promise.resolve({
        ...GET_APP_FOUND,
        podAddress: 'test/address',
      });
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'isTensorboardPodReady').mockImplementation(() => Promise.resolve(true));
    tree = shallow(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    await flushPromisesAndTimers();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledTimes(1);
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apis/v1beta1/_proxy/test/address');
    expect(tree.debug()).toMatchInlineSnapshot(`
      "<div>
        <div>
          <div className=\\"\\">
            Tensorboard tensorflow/tensorflow:1.14.0 is running for this output.
          </div>
          <a href=\\"apis/v1beta1/_proxy/test/address\\" target=\\"_blank\\" rel=\\"noopener noreferrer\\" className=\\"unstyled\\">
            <WithStyles(Button) className=\\"buttonAction button\\" disabled={false} color=\\"primary\\">
              Open Tensorboard
            </WithStyles(Button)>
          </a>
          <div>
            <WithStyles(Button) className=\\"button\\" disabled={false} id=\\"delete\\" title=\\"stop tensorboard and delete its instance\\" onClick={[Function]} color=\\"default\\">
              Stop Tensorboard
            </WithStyles(Button)>
            <WithStyles(Dialog) open={false} onClose={[Function]} aria-labelledby=\\"dialog-title\\">
              <WithStyles(DialogTitle) id=\\"dialog-title\\">
                Stop Tensorboard?
              </WithStyles(DialogTitle)>
              <WithStyles(DialogContent)>
                <WithStyles(DialogContentText)>
                  You can stop the current running tensorboard. The tensorboard viewer will also be deleted from your workloads.
                </WithStyles(DialogContentText)>
              </WithStyles(DialogContent)>
              <WithStyles(DialogActions)>
                <WithStyles(Button) className=\\"shortButton\\" id=\\"cancel\\" autoFocus={true} onClick={[Function]} color=\\"primary\\">
                  Cancel
                </WithStyles(Button)>
                <BusyButton className=\\"buttonAction shortButton\\" onClick={[Function]} busy={false} color=\\"primary\\" title=\\"Stop\\" />
              </WithStyles(DialogActions)>
            </WithStyles(Dialog)>
          </div>
        </div>
      </div>"
    `);
  });

  it('shows start button if no instance exists', async () => {
    const config = DEFAULT_CONFIG;
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
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
      -       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={true} title="Start Tensorboard" />
      +       <BusyButton className="buttonAction" disabled={false} onClick={[Function]} busy={false} title="Start Tensorboard" />
            </div>
          </div>
        </div>
    `);
    expect(getTensorboardSpy).toHaveBeenCalledWith(config.url, config.namespace);
  });

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { ...DEFAULT_CONFIG };
    const getAppMock = () => Promise.resolve(GET_APP_NOT_FOUND);
    const startAppMock = jest.fn(() => Promise.resolve(''));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    jest.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    tree = shallow(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.find('BusyButton').simulate('click');
    expect(startAppMock).toHaveBeenCalledWith({
      logdir: config.url,
      namespace: config.namespace,
      image: expect.stringContaining('tensorflow/tensorflow:'), // default image
    });
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const config2 = { ...DEFAULT_CONFIG, url: 'http://test/url2' };
    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_NOT_FOUND));
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
    expect(startAppMock).toHaveBeenCalledWith({
      logdir: expectedUrl,
      image: expect.stringContaining('tensorflow/tensorflow:'), // default image
      namespace: config.namespace,
    });
  });

  it('returns friendly display name', () => {
    expect(TensorboardViewer.prototype.getDisplayName()).toBe('Tensorboard');
  });

  it('is aggregatable', () => {
    expect(TensorboardViewer.prototype.isAggregatable()).toBeTruthy();
  });

  it('select a version, then start a tensorboard of the corresponding version', async () => {
    const config = { ...DEFAULT_CONFIG };

    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_NOT_FOUND));
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
      .findWhere(el => el.text().startsWith('TensorFlow 1.15'))
      .hostNodes()
      .simulate('click');
    tree.find('BusyButton').simulate('click');
    expect(startAppSpy).toHaveBeenCalledWith({
      logdir: config.url,
      image: 'tensorflow/tensorflow:1.15.5',
      namespace: config.namespace,
    });
  });

  it('delete the tensorboard instance, confirm in the dialog,\
    then return back to previous page', async () => {
    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_FOUND));
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
    expect(tree.findWhere(el => el.text() === 'Start Tensorboard').exists()).toBeTruthy();
  });

  it('show version info in delete confirming dialog, \
    if a tensorboard instance already exists', async () => {
    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_FOUND));
    jest.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    const config = DEFAULT_CONFIG;
    tree = mount(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    tree.update();
    tree
      .find('#delete')
      .find('Button')
      .simulate('click');
    expect(tree.findWhere(el => el.text() === 'Stop Tensorboard?').exists()).toBeTruthy();
  });

  it('click on cancel on delete tensorboard dialog, then return back to previous page', async () => {
    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_FOUND));
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

    expect(tree.findWhere(el => el.text() === 'Open Tensorboard').exists()).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'Stop Tensorboard').exists()).toBeTruthy();
  });

  it('asks user to wait when Tensorboard status is not ready', async () => {
    const getAppMock = jest.fn(() => Promise.resolve(GET_APP_FOUND));
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
    expect(tree.findWhere(el => el.text() === 'Open Tensorboard').exists()).toBeTruthy();
    expect(
      tree
        .findWhere(
          el =>
            el.text() === 'Tensorboard is starting, and you may need to wait for a few minutes.',
        )
        .exists(),
    ).toBeTruthy();
    expect(tree.findWhere(el => el.text() === 'Stop Tensorboard').exists()).toBeTruthy();

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
