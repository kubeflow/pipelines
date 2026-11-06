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
import {
  act,
  fireEvent,
  render as rtlRender,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { vi } from 'vitest';
import TensorboardViewer, { TensorboardViewerConfig } from './Tensorboard';
import TestUtils, { flushPromisesInAct, invokeAndFlush } from '../../TestUtils';
import { Apis } from '../../lib/Apis';
import { PlotType } from './Viewer';
import { ThemeProvider } from '../shell/ThemeProvider';

const render = (ui: React.ReactElement) =>
  rtlRender(ui, {
    wrapper: ({ children }) => <ThemeProvider defaultTheme='light'>{children}</ThemeProvider>,
  });

const DEFAULT_CONFIG: TensorboardViewerConfig = {
  type: PlotType.TENSORBOARD,
  url: 'http://test/url',
  namespace: 'test-ns',
};

const GET_APP_NOT_FOUND = { proxyPath: '', tfVersion: '', image: '' };
const GET_APP_FOUND = {
  proxyPath: 'apps/tensorboard/proxy/test-token/',
  tfVersion: '1.14.0',
  image: 'tensorflow/tensorflow:1.14.0',
};

describe('Tensorboard', () => {
  let intervalCallback: (() => void) | null = null;
  let setIntervalSpy: ReturnType<typeof vi.spyOn>;
  let clearIntervalSpy: ReturnType<typeof vi.spyOn>;

  const flushPromisesAndInterval = async () => {
    await act(async () => {
      if (intervalCallback) {
        intervalCallback();
      }
      await TestUtils.flushPromises();
    });
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })),
    );

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
    vi.unstubAllGlobals();
  });

  it('base component snapshot', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const { asFragment } = render(<TensorboardViewer configs={[]} />);
    await flushPromisesInAct();
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not break on no config', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    render(<TensorboardViewer configs={[]} />);
    await flushPromisesInAct();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('does not break on empty data', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const config = { ...DEFAULT_CONFIG, url: '' };
    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('shows a link to the tensorboard instance if exists', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue({
      ...GET_APP_FOUND,
      proxyPath: 'apps/tensorboard/proxy/existing-token/',
    });
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(true);
    render(<TensorboardViewer configs={[config]} />);

    await flushPromisesInAct();
    await flushPromisesAndInterval();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith(
      'apps/tensorboard/proxy/existing-token/',
    );
    expect(
      screen.getByText('Tensorboard tensorflow/tensorflow:1.14.0 is running for this output.'),
    ).toBeInTheDocument();
    const link = screen.getByRole('link', { name: 'Open Tensorboard' });
    expect(link).toHaveAttribute('href', 'apps/tensorboard/proxy/existing-token/');
  });

  it('shows start button if no instance exists', async () => {
    const config = DEFAULT_CONFIG;
    const getTensorboardSpy = vi
      .spyOn(Apis, 'getTensorboardApp')
      .mockResolvedValue(GET_APP_NOT_FOUND);
    render(<TensorboardViewer configs={[DEFAULT_CONFIG]} />);
    await flushPromisesInAct();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
    expect(getTensorboardSpy).toHaveBeenCalledWith(config.url, config.namespace);
  });

  it.each(['lookup', 'start'] as const)(
    'shows the server namespace error on authenticated %s requests',
    async (operation) => {
      const fetchSpy = vi.spyOn(global, 'fetch').mockImplementation(async (_url, init) => {
        if (operation === 'start' && init?.method !== 'POST') {
          return new Response(JSON.stringify(GET_APP_NOT_FOUND));
        }
        return new Response('namespace argument is required', { status: 400 });
      });
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      render(<TensorboardViewer configs={[{ ...DEFAULT_CONFIG, namespace: '' }]} />);
      await flushPromisesInAct();
      if (operation === 'start') {
        fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
      }
      expect(await screen.findByText('namespace argument is required')).toBeInTheDocument();
      expect(screen.queryByText('Unknown error')).not.toBeInTheDocument();
      expect(fetchSpy).toHaveBeenLastCalledWith(
        expect.stringContaining('apps/tensorboard'),
        expect.anything(),
      );
      consoleSpy.mockRestore();
    },
  );

  it('starts tensorboard instance when button is clicked', async () => {
    const config = { ...DEFAULT_CONFIG };
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const startAppMock = vi.fn(() => Promise.resolve(''));
    vi.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();
    await invokeAndFlush(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    });
    expect(startAppMock).toHaveBeenCalledWith({
      logdir: config.url,
      namespace: config.namespace,
      image: expect.stringContaining('tensorflow/tensorflow:'),
      podTemplateSpec: undefined,
    });
  });

  it('shows the open link immediately when start returns a pod address', async () => {
    const config = { ...DEFAULT_CONFIG };
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const startAppSpy = vi
      .spyOn(Apis, 'startTensorboardApp')
      .mockResolvedValue('apps/tensorboard/proxy/new-token/');
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(false);

    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();

    fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));

    await flushPromisesInAct();
    await waitFor(() => expect(screen.getByRole('link', { name: 'Open Tensorboard' })));
    expect(screen.getByRole('link', { name: 'Open Tensorboard' })).toHaveAttribute(
      'href',
      'apps/tensorboard/proxy/new-token/',
    );
    await flushPromisesAndInterval();

    expect(startAppSpy).toHaveBeenCalledWith({
      logdir: config.url,
      namespace: config.namespace,
      image: expect.stringContaining('tensorflow/tensorflow:'),
      podTemplateSpec: undefined,
    });
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apps/tensorboard/proxy/new-token/');
  });

  it('starts tensorboard instance for two configs', async () => {
    const config = { ...DEFAULT_CONFIG, url: 'http://test/url' };
    const config2 = { ...DEFAULT_CONFIG, url: 'http://test/url2' };
    const getAppMock = vi.fn(() => Promise.resolve(GET_APP_NOT_FOUND));
    const startAppMock = vi.fn(() => Promise.resolve(''));
    vi.spyOn(Apis, 'getTensorboardApp').mockImplementation(getAppMock);
    vi.spyOn(Apis, 'startTensorboardApp').mockImplementationOnce(startAppMock);
    render(<TensorboardViewer configs={[config, config2]} />);
    await flushPromisesInAct();
    expect(getAppMock).toHaveBeenCalledWith(
      `Series1:${config.url},Series2:${config2.url}`,
      config.namespace,
    );
    await invokeAndFlush(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Start Combined Tensorboard' }));
    });
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

    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();
    fireEvent.change(screen.getByRole('combobox', { name: 'TF Image' }), {
      target: { value: 'tensorflow/tensorflow:1.15.5' },
    });

    await invokeAndFlush(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    });
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
    await flushPromisesInAct();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }));

    expect(deleteAppSpy).toHaveBeenCalledWith(config.url, config.namespace);
    await flushPromisesInAct();
    expect(screen.getByRole('button', { name: 'Start Tensorboard' })).toBeInTheDocument();
  });

  it('shows version info in delete confirming dialog if a tensorboard instance exists', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    expect(screen.getByText('Stop Tensorboard?')).toBeInTheDocument();
  });

  it('click on cancel on delete tensorboard dialog, then return back', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();

    fireEvent.click(screen.getByText('Stop Tensorboard'));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await flushPromisesInAct();
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.getByRole('link', { name: 'Open Tensorboard' })).toBeInTheDocument();
    expect(screen.getByText('Stop Tensorboard')).toBeInTheDocument();
  });

  it('asks user to wait when Tensorboard status is not ready', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    vi.spyOn(Apis, 'isTensorboardPodReady').mockResolvedValue(false);
    vi.spyOn(Apis, 'deleteTensorboardApp').mockImplementation(vi.fn(() => Promise.resolve('')));
    const config = DEFAULT_CONFIG;
    render(<TensorboardViewer configs={[config]} />);

    await flushPromisesInAct();
    await flushPromisesAndInterval();
    expect(Apis.isTensorboardPodReady).toHaveBeenCalledWith('apps/tensorboard/proxy/test-token/');
    expect(screen.getByRole('link', { name: 'Open Tensorboard' })).toBeInTheDocument();
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
  it('retains the selected custom image and pod template through a failed start and retry', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_NOT_FOUND);
    const start = vi
      .spyOn(Apis, 'startTensorboardApp')
      .mockRejectedValueOnce(new Error('Start denied'))
      .mockResolvedValueOnce('apps/tensorboard/proxy/recovered/');
    const config = {
      ...DEFAULT_CONFIG,
      image: 'registry.example/tensorboard:custom',
      podTemplateSpec: { spec: { serviceAccountName: 'viewer' } },
    };
    render(<TensorboardViewer configs={[config]} />);
    await flushPromisesInAct();
    expect(screen.getByRole('combobox', { name: 'TF Image' })).toHaveValue(config.image);
    fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Start denied');
    fireEvent.click(screen.getByRole('button', { name: 'Start Tensorboard' }));
    await screen.findByRole('link', { name: 'Open Tensorboard' });
    expect(screen.queryByText('Start denied')).not.toBeInTheDocument();
    expect(start).toHaveBeenCalledTimes(2);
    expect(start).toHaveBeenLastCalledWith({
      logdir: config.url,
      namespace: config.namespace,
      image: config.image,
      podTemplateSpec: config.podTemplateSpec,
    });
  });

  it('cancels without deleting and retains the running instance after a failed stop', async () => {
    vi.spyOn(Apis, 'getTensorboardApp').mockResolvedValue(GET_APP_FOUND);
    const remove = vi
      .spyOn(Apis, 'deleteTensorboardApp')
      .mockRejectedValueOnce(new Error('Stop denied'))
      .mockResolvedValueOnce('');
    render(<TensorboardViewer configs={[DEFAULT_CONFIG]} />);
    await flushPromisesInAct();
    fireEvent.click(screen.getByRole('button', { name: 'Stop Tensorboard' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(remove).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Stop Tensorboard' }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Stop' }));
    await waitFor(() =>
      expect(
        within(screen.getByRole('dialog')).getByRole('button', { name: 'Stop' }),
      ).toBeEnabled(),
    );
    expect(screen.getByText('Stop denied')).toBeInTheDocument();
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Stop' }));
    await screen.findByRole('button', { name: 'Start Tensorboard' });
    expect(remove).toHaveBeenCalledTimes(2);
    expect(remove).toHaveBeenLastCalledWith(DEFAULT_CONFIG.url, DEFAULT_CONFIG.namespace);
    expect(screen.queryByText('Stop denied')).not.toBeInTheDocument();
  });
});
