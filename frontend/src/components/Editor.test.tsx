/*
 * Copyright 2019 Google LLC
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
import 'brace';
import { mount } from 'enzyme';
import Editor from './Editor';
import 'brace/ext/language_tools';
import 'brace/mode/json';
import 'brace/mode/python';
import 'brace/theme/github';

/*
  tree.html() is used alongside mount because it allows for the HTML of the
  Editor component to be rendered and compared to a snapshot. This is required
  because the react-ace library utilizes brace which is based on vanilla HTML
  and JavaScript, without mount and tree.html() the component would not render
  the brace editor.
*/

describe('Editor', () => {
  it('renders without a placeholder', () => {
    const tree = mount(<Editor />);
    expect(tree.html()).toMatchSnapshot();
  });

  it('renders with a placeholder', () => {
    const tree = mount(<Editor placeholder='I am a placeholder.' />);
    expect(tree.html()).toMatchSnapshot();
  });

  it ('renders a placeholder that contains HTML', () => {
    const placeholder = 'I am a placeholder with <strong>HTML</strong>.';
    const tree = mount(<Editor placeholder={placeholder} />);
    expect(tree.html()).toMatchSnapshot();
  });
});