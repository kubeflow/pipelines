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

/* eslint-disable */
// Because this is test utils.

import 'src/build/tailwind.output.css';
import { QueryClient } from 'react-query';
import { match } from 'react-router';
import { beforeEach, expect, MockInstance } from 'vitest';
import { ToolbarActionConfig } from './components/Toolbar';
import { Feature } from './features';
import { logger } from './lib/Utils';
import { Page, PageProps } from './pages/Page';

export default class TestUtils {
  /**
   * Flushes all already queued promises and returns a promise. Note this will
   * only work if the promises have already been queued, so it cannot be used to
   * wait on a promise that hasn't been dispatched yet.
   */
  public static async flushPromises(): Promise<void> {
    const testApi = getTestApi();
    if (
      typeof testApi.isMockFunction === 'function' &&
      testApi.isMockFunction(setTimeout) &&
      typeof testApi.advanceTimersByTime === 'function'
    ) {
      // Fake timers won't advance unless we flush them explicitly.
      testApi.advanceTimersByTime(0);
      await Promise.resolve();
      testApi.advanceTimersByTime(0);
      await Promise.resolve();
      return;
    }
    await new Promise(resolve => setTimeout(resolve, 0));
    await new Promise(resolve => setTimeout(resolve, 0));
  }

  /**
   * Adds a one-time mock implementation to the provided spy that mimics an error
   * network response
   */
  public static makeErrorResponseOnce(spy: MockInstance, message: string): void {
    spy.mockImplementationOnce(() => {
      throw {
        text: () => Promise.resolve(message),
      };
    });
  }

  /**
   * Adds a mock implementation to the provided spy that mimics an error
   * network response
   */
  public static makeErrorResponse(spy: MockInstance, message: string): void {
    spy.mockImplementation(() => {
      throw {
        text: () => Promise.resolve(message),
      };
    });
  }

  /**
   * Generates a customizable PageProps object that can be passed to initialize
   * Page components, taking care of setting ToolbarProps properly, which have
   * to be set after component initialization.
   */
  // tslint:disable-next-line:variable-name
  public static generatePageProps(
    PageElement: new (_: PageProps) => Page<any, any>,
    location: Location,
    matchValue: match,
    historyPushSpy: MockInstance | null,
    updateBannerSpy: MockInstance | null,
    updateDialogSpy: MockInstance | null,
    updateToolbarSpy: MockInstance | null,
    updateSnackbarSpy: MockInstance | null,
  ): PageProps {
    const pageProps = {
      history: { push: historyPushSpy } as any,
      location: location as any,
      match: matchValue,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy as any,
      updateDialog: updateDialogSpy as any,
      updateSnackbar: updateSnackbarSpy as any,
      updateToolbar: updateToolbarSpy as any,
    } as PageProps;
    pageProps.toolbarProps = new PageElement(pageProps).getInitialToolbarState();
    // The toolbar spy gets called in the getInitialToolbarState method, reset it
    // in order to simplify tests
    if (updateToolbarSpy) {
      updateToolbarSpy.mockReset();
    }
    return pageProps;
  }

  public static getToolbarButton(
    updateToolbarSpy: MockInstance,
    buttonKey: string,
  ): ToolbarActionConfig {
    const lastCallIdx = updateToolbarSpy.mock.calls.length - 1;
    const lastCall = updateToolbarSpy.mock.calls[lastCallIdx][0];
    return lastCall.actions[buttonKey];
  }
}

function getTestApi() {
  const testApi = (globalThis as any).vi;
  if (!testApi) {
    throw new Error('Vitest API (vi) not found');
  }
  return testApi;
}

export function expectWarnings() {
  const testApi = getTestApi();
  const loggerWarningSpy = testApi.spyOn(console, 'warn');
  loggerWarningSpy.mockImplementation(() => null);
  return () => {
    expect(loggerWarningSpy).toHaveBeenCalled();
  };
}

export function expectErrors() {
  const testApi = getTestApi();
  const loggerErrorSpy = testApi.spyOn(console, 'error');
  loggerErrorSpy.mockImplementation(() => null);
  return () => {
    expect(loggerErrorSpy).toHaveBeenCalled();
  };
}

export const queryClientTest = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});
export function testBestPractices() {
  beforeEach(async () => {
    queryClientTest.clear();
    const testApi = getTestApi();
    testApi.resetAllMocks();
    testApi.restoreAllMocks();
  });
}

export function forceSetFeatureFlag(features: Feature[]) {
  window.__FEATURE_FLAGS__ = JSON.stringify(features);
}

export function mockResizeObserver(width = 800, height = 600) {
  const testApi = getTestApi();
  // Required by reactflow render.
  Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
    configurable: true,
    get: () => width,
  });
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
    configurable: true,
    get: () => height,
  });
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
    configurable: true,
    get: () => width,
  });
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get: () => height,
  });
  class ResizeObserverMock {
    private readonly callback: ResizeObserverCallback;
    public readonly disconnect: () => void;
    public readonly observe: (element: Element) => void;
    public readonly unobserve: () => void;

    constructor(callback: ResizeObserverCallback) {
      this.callback = callback;
      this.disconnect = testApi.fn();
      this.observe = testApi.fn((element: Element) => {
        this.callback(
          [
            {
              target: element,
              contentRect: { width, height } as DOMRectReadOnly,
            } as ResizeObserverEntry,
          ],
          (this as unknown) as ResizeObserver,
        );
      });
      this.unobserve = testApi.fn();
    }
  }

  (window as any).ResizeObserver = (ResizeObserverMock as unknown) as typeof ResizeObserver;
}
