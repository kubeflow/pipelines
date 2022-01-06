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
import { mount, ReactWrapper } from 'enzyme';
import { format } from 'prettier';
import { object } from 'prop-types';
import * as React from 'react';
import { QueryClient } from 'react-query';
import { match } from 'react-router';
// @ts-ignore
import createRouterContext from 'react-router-test-context';
import snapshotDiff from 'snapshot-diff';
import { ToolbarActionConfig } from './components/Toolbar';
import { Feature } from './features';
import { logger } from './lib/Utils';
import { Page, PageProps } from './pages/Page';

export default class TestUtils {
  /**
   * Mounts the given component with a fake router and returns the mounted tree
   */
  // tslint:disable-next-line:variable-name
  public static mountWithRouter(component: React.ReactElement<any>): ReactWrapper {
    const childContextTypes = {
      router: object,
    };
    const context = createRouterContext();
    const tree = mount(component, { context, childContextTypes });
    return tree;
  }

  /**
   * Flushes all already queued promises and returns a promise. Note this will
   * only work if the promises have already been queued, so it cannot be used to
   * wait on a promise that hasn't been dispatched yet.
   */
  public static flushPromises(): Promise<void> {
    return new Promise(process.nextTick);
  }

  /**
   * Adds a one-time mock implementation to the provided spy that mimics an error
   * network response
   */
  public static makeErrorResponseOnce(
    spy: jest.MockInstance<unknown, any[]>,
    message: string,
  ): void {
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
  public static makeErrorResponse(spy: jest.MockInstance<unknown, any[]>, message: string): void {
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
    historyPushSpy: jest.SpyInstance<unknown> | null,
    updateBannerSpy: jest.SpyInstance<unknown> | null,
    updateDialogSpy: jest.SpyInstance<unknown> | null,
    updateToolbarSpy: jest.SpyInstance<unknown> | null,
    updateSnackbarSpy: jest.SpyInstance<unknown> | null,
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
    updateToolbarSpy: jest.SpyInstance<unknown>,
    buttonKey: string,
  ): ToolbarActionConfig {
    const lastCallIdx = updateToolbarSpy.mock.calls.length - 1;
    const lastCall = updateToolbarSpy.mock.calls[lastCallIdx][0];
    return lastCall.actions[buttonKey];
  }
}

/**
 * Generate diff text for two HTML strings.
 * Recommend providing base and update annotations to clarify context in the diff directly.
 */
export function diffHTML({
  base,
  update,
  baseAnnotation,
  updateAnnotation,
}: {
  base: string;
  baseAnnotation?: string;
  update: string;
  updateAnnotation?: string;
}) {
  return diff({
    base: formatHTML(base),
    update: formatHTML(update),
    baseAnnotation,
    updateAnnotation,
  });
}

export function diff({
  base,
  update,
  baseAnnotation,
  updateAnnotation,
}: {
  base: string;
  baseAnnotation?: string;
  update: string;
  updateAnnotation?: string;
}) {
  return snapshotDiff(base, update, {
    stablePatchmarks: true, // Avoid line numbers in diff, so that diffs are stable against irrelevant changes
    aAnnotation: baseAnnotation,
    bAnnotation: updateAnnotation,
  });
}

export function formatHTML(html: string): string {
  return format(html, { parser: 'html' });
}

export function expectWarnings() {
  const loggerWarningSpy = jest.spyOn(logger, 'warn');
  loggerWarningSpy.mockImplementation();
  return () => {
    expect(loggerWarningSpy).toHaveBeenCalled();
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
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });
}

export function forceSetFeatureFlag(features: Feature[]) {
  window.__FEATURE_FLAGS__ = JSON.stringify(features);
}

export function mockResizeObserver() {
  // Required by reactflow render.
  (window as any).ResizeObserver = jest.fn();
  (window as any).ResizeObserver.mockImplementation(() => ({
    disconnect: jest.fn(),
    observe: jest.fn(),
    unobserve: jest.fn(),
  }));
}
