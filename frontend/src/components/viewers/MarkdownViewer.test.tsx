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
import MarkdownViewer, { MarkdownViewerConfig } from './MarkdownViewer';
import { PlotType } from './Viewer';
import { TFunction } from 'i18next';
import { componentMap } from './ViewerContainer';

let mockValue = '';
jest.mock('i18next', () => ({ t: () => mockValue }));

describe('MarkdownViewer', () => {
  let t: TFunction = (key: string) => key;
  it('does not break on empty data', () => {
    const tree = mount(<MarkdownViewer configs={[]} />).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders some basic markdown', () => {
    const markdown = '# Title\n[some link here](http://example.com)';
    const config: MarkdownViewerConfig = {
      markdownContent: markdown,
      type: PlotType.MARKDOWN,
    };

    const tree = mount(<MarkdownViewer configs={[config]} />).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('sanitizes the markdown to remove XSS', () => {
    const markdown = `
      lower[click me](javascript&#x3a;...)lower
      upper[click me](javascript&#X3a;...)upper
    `;
    const config: MarkdownViewerConfig = {
      markdownContent: markdown,
      type: PlotType.MARKDOWN,
    };

    const tree = mount(<MarkdownViewer configs={[config]} />).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('returns a user friendly display name', () => {
    mockValue = 'common:markdown';
    expect((componentMap[PlotType.MARKDOWN].displayNameKey = 'common:markdown'));
  });
});
