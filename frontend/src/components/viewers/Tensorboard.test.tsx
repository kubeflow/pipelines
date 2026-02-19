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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import TensorboardViewer, { TensorboardViewerConfig } from './Tensorboard';
import TestUtils from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';

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
  let intervalCallback: (() => void) | null = null;
  let setIntervalSpy: ReturnType<typeof vi.spyOn>;
  let clearIntervalSpy: ReturnType<typeof vi.spyOn>;

  const flushPromisesAndInterval = async () => {
    if (intervalCallback) {
      intervalCallback();
    }
    await TestUtils.flushPromises();
  };

  beforeEach(() => {
    vi.clearAllMocks();
    intervalCallback = null;
    setIntervalSpy = vi.spyOn(global, 'setInterval').mockImplementation((callback: any) => {
      intervalCallback = callback;
      return 0 as any;
    });
    clearIntervalSpy = vi.spyOn(global, 'clearInterval').mockImplementation(() => {});
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(false);
  });

  afterEach(() => {
    setIntervalSpy.mockRestore();
    clearIntervalSpy.mockRestore();
    vi.restoreAllMocks();
  });

  it('base component snapshot', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const { asFragment } = render(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not break on no config', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    render(<TensorboardViewer configs={[]} />);
    await TestUtils.flushPromises();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('does not break on empty data', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const config = { ...DEFAULT_CONFIG, url: '' };
    render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue({
      ...GET_APP_FOUND,
      podAddress: 'test/address',
    });
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(true);
    render(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    await flushPromisesAndInterval();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apis/v1beta1/_proxy/test/address');
    expect(
      screen.getByText('Tensorboard tensorflow/tensorflow:1.14.0 is running for this output.'),
    ).toBeInTheDocument();
    const link = screen.getByRole('link', { name: 'Open Tensorboard' });
    expect(link).toHaveAttribute('href', 'apis/v1beta1/_proxy/test/address');
  });

  it('shows start button if no instance exists', async () => {
    const config = DEFAULT_CONFIG;
    const getTensorboardSpy = vi
      .spyOn(Apis, 'getTensorboardApp')
      .mockResolvedValue(GET_APP_NOT_FOUND);
    render(<TensorboardViewer configs={[DEFAULT_CONFIG]} />);
    await TestUtils.flushPromises();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
    expect(getTensorboardSpy).toHaveBeenCalledWith(config.url, config.namespace);
  });

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { ...DEFAULT_CONFIG };
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const startAppMock = vi.fn(() => Promise.resolve(''));
    vi.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();
    fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    expect(startAppMock).toHaveBeenCalledWith({
      logdir: config.url,
      namespace: config.namespace,
      image: expect.stringContaining('tensorflow/tensorflow:'),
      podTemplateSpec: undefined,
    });
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const config2 = { ...DEFAULT_CONFIG, url: 'http://test/url2' };
    const getAppMock = vi.fn(() => Promise.resolve(GET_APP_NOT_FOUND));
    const startAppMock = vi.fn(() => Promise.resolve(''));
    vi.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    vi.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    render(<TensorboardViewer configs={[config, config2]} />);
    await TestUtils.flushPromises();
    expect(getAppMock).toHaveBeenCalledWith(
      `Series1:${config.url},Series2:${config2.url}`,
      config.namespace,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Start Combined Tensorboard' }));
    const expectedUrl = `Series1:${config.url},Series2:${config2.url}`;
    expect(startAppMock).toHaveBeenCalledWith({
      logdir: expectedUrl,
      image: expect.stringContaining('tensorflow/tensorflow:'),
      namespace: config.namespace,
      podTemplateSpec: undefined,
    });
  });

  it('returns friendly display name', () => {
    expect(TensorboardViewer.prototype.getDisplayName()).toBe('Tensorboard');
  });

  it('is aggregatable', () => {
    expect(TensorboardViewer.prototype.isAggregatable()).toBeTruthy();
  });

  it('selects a version, then starts a tensorboard of the corresponding version', async () => {
    const config = { ...DEFAULT_CONFIG };

    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const startAppMock = vi.fn(() => Promise.resolve(''));
    const startAppSpy = vi.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);

    const ref = React.createRef<TensorboardViewer>();
    render(<TensorboardViewer ref={ref} configs={[config]} />);
    await TestUtils.flushPromises();

    act(() => {
      ref.current?.handleImageSelect({
        target: { value: 'tensorflow/tensorflow:1.15.5' },
      } as React.ChangeEvent<{ name?: string; value: unknown }>);
    });

    fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    expect(startAppSpy).toHaveBeenCalledWith({
      logdir: config.url,
      image: 'tensorflow/tensorflow:1.15.5',
      namespace: config.namespace,
      podTemplateSpec: undefined,
    });
  });

  it('deletes the tensorboard instance, confirm in the dialog, then returns back', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const deleteAppMock = vi.fn(() => Promise.resolve(''));
    const deleteAppSpy = vi.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(deleteAppMock);
    const config = { ...DEFAULT_CONFIG };

    render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }));

    expect(deleteAppSpy).toHaveBeenCalledWith(config.url, config.namespace);
    await TestUtils.flushPromises();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('shows version info in delete confirming dialog if a tensorboard instance exists', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    expect(screen.getByText('Stop Tensorboard?')).toBeInTheDocument();
  });

  it('click on cancel on delete tensorboard dialog, then return back', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await TestUtils.flushPromises();
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Open Tensorboard' })).toBeInTheDocument();
    expect(screen.getByText('Stop Tensorboard')).toBeInTheDocument();
  });

  it('asks user to wait when Tensorboard status is not ready', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(false);
    vi.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(vi.fn(() => Promise.resolve('')));
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);

    await TestUtils.flushPromises();
    await flushPromisesAndInterval();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apis/v1beta1/_proxy/podaddress');
    expect(screen.getByRole('button', { name: 'Open Tensorboard' })).toBeInTheDocument();
    expect(
      screen.getByText('Tensorboard is starting, and you may need to wait for a few minutes.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Stop Tensorboard')).toBeInTheDocument();

    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(true);
    await flushPromisesAndInterval();
    expect(
      screen.queryByText('Tensorboard is starting, and you may need to wait for a few minutes.'),
    ).toBeNull();
  });
});
