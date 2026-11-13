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

import * as React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import LogViewer from './LogViewer';

function renderRow(logLines: string[], index = 0): HTMLElement | null {
  const logViewer = new LogViewer({ logLines });
  const row = (logViewer as any)._rowRenderer({ index });
  const { container } = render(row);
  return container.firstChild as HTMLElement | null;
}

describe('LogViewer resizing and scrolling', () => {
  let notifyResize: () => void;
  let width = 640;
  let height = 150;

  beforeEach(() => {
    vi.useFakeTimers();
    width = 640;
    height = 150;
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
      () => new DOMRect(0, 0, width, height),
    );
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: ResizeObserverCallback) {
          notifyResize = () => callback([], this as unknown as ResizeObserver);
        }
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  function resize() {
    act(() => {
      notifyResize();
      vi.advanceTimersByTime(20);
    });
  }

  it('renders measured logs and follows new lines after the first resize', () => {
    const lines = Array.from({ length: 100 }, (_, index) => `log line ${index + 1}`);
    const { rerender } = render(<LogViewer logLines={lines} />);
    expect(screen.queryByRole('list')).not.toBeInTheDocument();

    resize();
    const list = screen.getByRole('list');
    expect(list).toHaveStyle({ width: '640px', height: '150px' });
    expect(screen.getByText('log line 100')).toBeInTheDocument();
    expect(list.scrollTop).toBe(1350);

    rerender(<LogViewer logLines={[...lines, 'new log line']} />);
    expect(list.scrollTop).toBe(1365);
    expect(screen.getByText('new log line')).toBeInTheDocument();

    width = 800;
    height = 300;
    resize();
    expect(list).toHaveStyle({ width: '800px', height: '300px' });
  });

  it.each([
    [0, 0],
    [0, 150],
    [640, 0],
  ])('follows the tail when a %ix%i viewport becomes visible', (initialWidth, initialHeight) => {
    width = initialWidth;
    height = initialHeight;
    const lines = Array.from({ length: 100 }, (_, index) => `log line ${index + 1}`);
    const { rerender } = render(<LogViewer logLines={lines} />);
    resize();

    width = 640;
    height = 150;
    resize();
    const list = screen.getByRole('list');
    expect(list.scrollTop).toBe(1350);
    expect(screen.getByText('log line 100')).toBeInTheDocument();

    rerender(<LogViewer logLines={[...lines, 'new log line']} />);
    expect(list.scrollTop).toBe(1365);
  });

  it('handles an empty stream before logs arrive and after they are cleared', () => {
    const { rerender } = render(<LogViewer logLines={[]} />);
    resize();
    const list = screen.getByRole('list');
    expect(list.scrollTop).toBe(0);
    expect(screen.queryAllByRole('listitem')).toHaveLength(0);

    const lines = Array.from({ length: 100 }, (_, index) => `log line ${index + 1}`);
    rerender(<LogViewer logLines={lines} />);
    expect(list.scrollTop).toBe(1350);

    rerender(<LogViewer logLines={[]} />);
    expect(list.scrollTop).toBe(0);
    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
  });

  it('preserves a scrolled-up position until the user returns to the bottom', () => {
    const lines = Array.from({ length: 1000 }, (_, index) => `log line ${index + 1}`);
    const { rerender } = render(<LogViewer logLines={lines} />);
    resize();
    const list = screen.getByRole('list');
    fireEvent.scroll(list, { target: { scrollTop: 7500 } });
    expect(screen.getByText('log line 101')).toBeInTheDocument();
    expect(screen.getByText('log line 901')).toBeInTheDocument();

    rerender(<LogViewer logLines={[...lines, 'new log line']} />);
    expect(list.scrollTop).toBe(7500);

    width = 0;
    height = 0;
    resize();
    width = 640;
    height = 150;
    resize();
    const visibleList = screen.getByRole('list');
    expect(visibleList.scrollTop).toBe(7500);

    fireEvent.scroll(visibleList, { target: { scrollTop: 14865 } });
    rerender(<LogViewer logLines={[...lines, 'new log line', 'another new log line']} />);
    expect(visibleList.scrollTop).toBe(14880);
  });
});

describe('LogViewer', () => {
  it('renders an empty container when no logs passed', () => {
    const { asFragment } = render(<LogViewer logLines={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders one log line', () => {
    const logLines = ['first line'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
  });

  it('renders two log lines', () => {
    const logLines = ['first line', 'second line'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
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
    const row = renderRow([line]);
    expect(row).toMatchSnapshot();
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
    const row = renderRow(line.split('\n'));
    expect(row).toMatchSnapshot();
  });

  it('linkifies standalone urls', () => {
    const logLines = ['this string: http://path.com is a url'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
  });

  it('linkifies standalone https urls', () => {
    const logLines = ['this string: https://path.com is a url'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
  });

  it('linkifies substring urls', () => {
    const logLines = ['this string:http://path.com is a url'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
  });

  it('does not linkify non http/https urls', () => {
    const logLines = ['this string: gs://path is a GCS path'];
    const row = renderRow(logLines);
    expect(row).toMatchSnapshot();
  });

  it('scrolls to end after rendering', async () => {
    const spy = vi.spyOn(LogViewer.prototype as any, '_scrollToEnd');
    const logs = 'this string: gs://path is a GCS path';
    const ref = React.createRef<LogViewer>();
    render(<LogViewer ref={ref} logLines={[logs]} />);
    await act(async () => {
      ref.current?.componentDidUpdate?.({}, {});
    });
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });

  it('renders a row with given index as line number', () => {
    const row = renderRow(['line1', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with error', () => {
    const row = renderRow(['line1 with error', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with upper case error', () => {
    const row = renderRow(['line1 with ERROR', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with error word as substring', () => {
    const row = renderRow(['line1 with errorWord', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with warning', () => {
    const row = renderRow(['line1 with warning', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with warn', () => {
    const row = renderRow(['line1 with warn', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with upper case warning', () => {
    const row = renderRow(['line1 with WARNING', 'line2']);
    expect(row).toMatchSnapshot();
  });

  it('renders a row with warning word as substring', () => {
    const row = renderRow(['line1 with warning:something', 'line2']);
    expect(row).toMatchSnapshot();
  });
});
