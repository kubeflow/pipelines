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
// @ts-ignore
import createRouterContext from 'react-router-test-context';
import { mount, ReactWrapper } from 'enzyme';
import { object } from 'prop-types';

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
    return new Promise(resolve => setImmediate(resolve));
  }

  /**
   * Adds a one-time mock implementation to the provided spy that mimics an error
   * network response
   */
  public static makeErrorResponseOnce(spy: jest.SpyInstance, message: string): void {
    spy.mockImplementationOnce(() => {
      throw {
        text: () => Promise.resolve(message),
      };
    });
  }
}
