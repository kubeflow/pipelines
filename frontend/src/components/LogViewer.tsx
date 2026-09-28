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
import { List, AutoSizer, ListRowProps } from 'react-virtualized';
import './LogViewer.css';
import { OverscanIndicesGetter } from 'react-virtualized/dist/es/Grid';

const listContainerStyleOverride = {
  overflow: 'visible',
};

interface LogViewerProps {
  logLines: string[];
}

// Use the same amount of overscan above and below visible rows.
//
// Why:
// * Default behavior is that when we scroll to one direction, content off
// screen on the other direction is unmounted from browser immediately. This
// caused a bug when selecting lines + scrolling.
// * With new behavior implemented below: we are now overscanning on both
// directions disregard of which direction user is scrolling to, we can ensure
// lines not exceeding maximum overscanRowCount lines off screen are still
// selectable.
const overscanOnBothDirections: OverscanIndicesGetter = ({
  direction: _direction, // One of "horizontal" or "vertical"
  cellCount, // Number of rows or columns in the current axis
  scrollDirection: _scrollDirection, // 1 (forwards) or -1 (backwards)
  overscanCellsCount, // Maximum number of cells to over-render in either direction
  startIndex, // Begin of range of visible cells
  stopIndex, // End of range of visible cells
}) => {
  return {
    overscanStartIndex: Math.max(0, startIndex - overscanCellsCount),
    overscanStopIndex: Math.min(cellCount - 1, stopIndex + overscanCellsCount),
  };
};

interface LogViewerState {
  followNewLogs: boolean;
}

class LogViewer extends React.Component<LogViewerProps, LogViewerState> {
  public state = {
    followNewLogs: true,
  };

  private _rootRef = React.createRef<List>();

  public componentDidMount(): void {
    // Wait until the next frame to scroll to bottom, because doms haven't been
    // rendered when running this.
    setTimeout(() => {
      this._scrollToEnd();
    });
  }

  public componentDidUpdate(): void {
    if (this.state.followNewLogs) {
      this._scrollToEnd();
    }
  }

  public render(): React.JSX.Element {
    return (
      <AutoSizer>
        {({ height, width }) => (
          <List
            id='logViewer'
            containerStyle={listContainerStyleOverride}
            width={width}
            height={height}
            rowCount={this.props.logLines.length}
            rowHeight={15}
            className='kfp-log-viewer'
            aria-label='Task logs'
            ref={this._rootRef}
            overscanIndicesGetter={overscanOnBothDirections}
            overscanRowCount={
              400 /* make this large, so selecting maximum 400 lines is supported */
            }
            rowRenderer={this._rowRenderer.bind(this)}
            onScroll={this.handleScroll}
          />
        )}
      </AutoSizer>
    );
  }

  private handleScroll = (info: {
    clientHeight: number;
    scrollHeight: number;
    scrollTop: number;
  }) => {
    const offsetTolerance = 20; // pixels
    const isScrolledToBottom =
      info.scrollHeight - info.scrollTop - info.clientHeight <= offsetTolerance;
    if (isScrolledToBottom !== this.state.followNewLogs) {
      this.setState({
        followNewLogs: isScrolledToBottom,
      });
    }
  };

  private _scrollToEnd(): void {
    const root = this._rootRef.current;
    if (root) {
      root.scrollToRow(this.props.logLines.length + 1);
    }
  }

  private _rowRenderer(props: ListRowProps): React.ReactNode {
    const { style, key, index } = props;
    const line = this.props.logLines[index];
    return (
      <div key={key} className='kfp-log-line' data-level={getLineLevel(line)} style={style}>
        <MemoedLogLine index={index} line={line} />
      </div>
    );
  }
}

const LogLine: React.FC<{ index: number; line: string }> = ({ index, line }) => (
  <>
    <span className='kfp-log-number'>{index + 1}</span>
    <span className='kfp-log-text'>
      {parseLine(line).map((piece, p) => (
        <span key={p}>{piece}</span>
      ))}
    </span>
  </>
);
// improve performance when rerendering, because we render a lot of logs
const MemoedLogLine = React.memo(LogLine);

function getLineLevel(line: string): 'error' | 'warning' | undefined {
  const lower = line.toLowerCase();
  if (lower.includes('error') || lower.includes('fail')) return 'error';
  if (lower.includes('warn')) return 'warning';
  return undefined;
}

function parseLine(line: string): React.ReactNode[] {
  // Linkify URLs starting with http:// or https://
  // eslint-disable-next-line no-useless-escape
  const urlPattern = /(\b(https?):\/\/[-A-Z0-9+&@#\/%?=~_|!:,.;]*[-A-Z0-9+&@#\/%=~_|])/gim;
  let lastMatch = 0;
  let match = urlPattern.exec(line);
  const nodes = [];
  while (match) {
    // Append all text before URL match
    nodes.push(<span>{line.substr(lastMatch, match.index)}</span>);
    // Append URL via an anchor element
    nodes.push(
      <a href={match[0]} target='_blank' rel='noopener noreferrer' className='kfp-log-link'>
        {match[0]}
      </a>,
    );

    lastMatch = match.index + match[0].length;
    match = urlPattern.exec(line);
  }
  // Append all text after final URL
  nodes.push(<span>{line.substr(lastMatch)}</span>);
  return nodes;
}

export default LogViewer;
