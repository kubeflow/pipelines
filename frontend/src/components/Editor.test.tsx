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
import { mount } from 'enzyme';
import Editor from './Editor';

/*
  These tests mimic https://github.com/securingsincity/react-ace/blob/master/tests/src/ace.spec.js
  to ensure that editor properties (placeholder and value) can be properly
  tested.
*/

describe('Editor', () => {
  it('renders without a placeholder and value', () => {
    const tree = mount(<Editor />);
    expect(tree.html()).toMatchSnapshot();
  });

  it('renders with a placeholder', () => {
    const placeholder = 'I am a placeholder.';
    const tree = mount(<Editor placeholder={placeholder} />);
    expect(tree.html()).toMatchSnapshot();
  });

  it('renders a placeholder that contains HTML', () => {
    const placeholder = 'I am a placeholder with <strong>HTML</strong>.';
    const tree = mount(<Editor placeholder={placeholder} />);
    expect(tree.html()).toMatchSnapshot();
  });

  it('has its value set to the provided value', () => {
    const value = 'I am a value.';
    const tree = mount(<Editor value={value} />);
    expect(tree).not.toBeNull();
    const editor = (tree.instance() as any).editor;
    expect(editor.getValue()).toBe(value);
  });
});
