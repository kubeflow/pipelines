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
import Page404 from './404';
import { PageProps } from './Page';
import { shallow } from 'enzyme';

describe('404', () => {
  function generateProps(): PageProps {
    return {
      history: {} as any,
      location: { pathname: 'some bad page' } as any,
      match: {} as any,
      toolbarProps: {} as any,
      updateBanner: jest.fn(),
      updateDialog: jest.fn(),
      updateSnackbar: jest.fn(),
      updateToolbar: jest.fn(),
    };
  }

  it('renders a 404 page', () => {
    expect(shallow(<Page404 {...generateProps()} />)).toMatchSnapshot();
  });
});
