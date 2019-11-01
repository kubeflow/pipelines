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
import { mount, shallow } from 'enzyme';
import LogViewer from './LogViewer';

describe('LogViewer', () => {
  it('renders an empty container when no logs passed', () => {
    expect(shallow(<LogViewer logLines={[]} />)).toMatchSnapshot();
  });

  it('renders one log line', () => {
    const logLines = ['first line'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders two log lines', () => {
    const logLines = ['first line', 'second line'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders one long line without breaking', () => {
    const line =
      `Lorem Ipsum is simply dummy text of the printing and typesetting` +
      `industry. Lorem Ipsum has been the industry's standard dummy text ever` +
      `since the 1500s, when an unknown printer took a galley of type and` +
      `scrambled it to make a type specimen book. It has survived not only five` +
      `centuries, but also the leap into electronic typesetting, remaining` +
      `essentially unchanged. It was popularised in the 1960s with the release` +
      `of Letraset sheets containing Lorem Ipsum passages, and more recently` +
      `with desktop publishing software like Aldus PageMaker including versions` +
      `of Lorem Ipsum.`;
    const logViewer = new LogViewer({ logLines: [line] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a multi-line log', () => {
    const line = `Lorem Ipsum is simply dummy text of the printing and typesetting
    industry. Lorem Ipsum has been the industry's standard dummy text ever
    since the 1500s, when an unknown printer took a galley of type and
    scrambled it to make a type specimen book. It has survived not only five
    centuries, but also the leap into electronic typesetting, remaining
    essentially unchanged. It was popularised in the 1960s with the release
    of Letraset sheets containing Lorem Ipsum passages, and more recently
    with desktop publishing software like Aldus PageMaker including versions
    of Lorem Ipsum.`;
    const logViewer = new LogViewer({ logLines: line.split('\n') });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('linkifies standalone urls', () => {
    const logLines = ['this string: http://path.com is a url'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('linkifies standalone https urls', () => {
    const logLines = ['this string: https://path.com is a url'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('linkifies substring urls', () => {
    const logLines = ['this string:http://path.com is a url'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('does not linkify non http/https urls', () => {
    const logLines = ['this string: gs://path is a GCS path'];
    const logViewer = new LogViewer({ logLines });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('scrolls to end after rendering', () => {
    const spy = jest.spyOn(LogViewer.prototype as any, '_scrollToEnd');
    const logs = 'this string: gs://path is a GCS path';
    const tree = mount(<LogViewer logLines={[logs]} />);
    tree.instance().componentDidUpdate!({}, {});
    expect(spy).toHaveBeenCalled();
  });

  it('renders a row with given index as line number', () => {
    const logViewer = new LogViewer({ logLines: ['line1', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with error', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with error', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with upper case error', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with ERROR', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with error word as substring', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with errorWord', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with warning', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with warning', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with warn', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with warn', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with upper case warning', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with WARNING', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });

  it('renders a row with warning word as substring', () => {
    const logViewer = new LogViewer({ logLines: ['line1 with warning:something', 'line2'] });
    const tree = mount((logViewer as any)._rowRenderer({ index: 0 })).getDOMNode();
    expect(tree).toMatchSnapshot();
  });
});
