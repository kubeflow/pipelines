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
import { Description } from './Description';

describe('Description', () => {
  describe('When in normal mode', () => {
    it('renders empty string', () => {
      const tree = mount(<Description description='' />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders pure text', () => {
      const tree = mount(<Description description='this is a line of pure text' />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders raw link', () => {
      const description = 'https://www.google.com';
      const tree = mount(<Description description={description} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders markdown link', () => {
      const description = '[google](https://www.google.com)';
      const tree = mount(<Description description={description} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders paragraphs', () => {
      const description = 'Paragraph 1\n' + '\n' + 'Paragraph 2';
      const tree = mount(<Description description={description} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders markdown list as list', () => {
      const description = `
* abc
* def`;
      const tree = mount(<Description description={description} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });
  });

  describe('When in inline mode', () => {
    it('renders paragraphs separated by space', () => {
      const description = `
Paragraph 1

Paragraph 2
`;
      const tree = mount(<Description description={description} forceInline={true} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders pure text', () => {
      const tree = mount(
        <Description description='this is a line of pure text' forceInline={true} />,
      ).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders raw link', () => {
      const description = 'https://www.google.com';
      const tree = mount(<Description description={description} forceInline={true} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders markdown link', () => {
      const description = '[google](https://www.google.com)';
      const tree = mount(<Description description={description} forceInline={true} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });

    it('renders markdown list as pure text', () => {
      const description = `
* abc
* def`;
      const tree = mount(<Description description={description} forceInline={true} />).getDOMNode();
      expect(tree).toMatchSnapshot();
    });
  });
});
