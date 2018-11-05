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
import { mount } from 'enzyme';
import { object } from 'prop-types';

export default class TestUtils {
  // tslint:disable-next-line:variable-name
  public static mountWithRouter(ComponentClass: React.ComponentClass, props: any) {
    const childContextTypes = {
      router: object,
    };
    const context = createRouterContext();
    const tree = mount(<ComponentClass {...props} />, { context, childContextTypes });
    return tree;
  }

  public static flushPromises() {
    return new Promise(resolve => setImmediate(resolve));
  }
}
