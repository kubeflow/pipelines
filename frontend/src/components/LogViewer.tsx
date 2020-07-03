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
import { List, AutoSizer, ListRowProps } from 'react-virtualized';
import { fontsize, fonts } from '../Css';
import { stylesheet } from 'typestyle';
import { OverscanIndicesGetter } from 'react-virtualized/dist/es/Grid';

const css = stylesheet({
  a: {
    $nest: {
      '&:hover': {
        color: '#bcdeff',
      },
      '&:visited': {
        color: '#bc9fff',
      },
    },
    color: '#96cbfe',
  },
  line: {
    $nest: {
      '&:hover': {
        backgroundColor: '#333',
      },
    },
    display: 'flex',
  },
  number: {
    color: '#999',
    flex: '40px 0 0',
    paddingRight: 10,
    textAlign: 'right',
    userSelect: 'none',
  },
  root: {
    // We cannot easily add padding here without breaking react-virtualized size calculation, for
    // details: https://github.com/bvaughn/react-virtualized/issues/992
    // Specifically, a complex solution was proposed in https://github.com/bvaughn/react-virtualized/issues/992#issuecomment-371145943.
    // We may consider that later.
    backgroundColor: '#222',
    color: '#fff',
    fontFamily: fonts.code,
    fontSize: fontsize.small,
    // This override and listContainerStyleOverride workarounds to allow horizontal scroll.
    // Reference: https://github.com/bvaughn/react-virtualized/issues/1248
    overflow: 'auto !important',
    whiteSpace: 'pre',
  },
});

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
  direction, // One of "horizontal" or "vertical"
  cellCount, // Number of rows or columns in the current axis
  scrollDirection, // 1 (forwards) or -1 (backwards)
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

  public render(): JSX.Element {
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
            className={css.root}
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
      <div key={key} className={css.line} style={style}>
        <MemoedLogLine index={index} line={line} />
      </div>
    );
  }
}

const LogLine: React.FC<{ index: number; line: string }> = ({ index, line }) => (
  <>
    <span className={css.number} style={getLineStyle(line)}>
      {index + 1}
    </span>
    <span className={css.line} style={getLineStyle(line)}>
      {parseLine(line).map((piece, p) => (
        <span key={p}>{piece}</span>
      ))}
    </span>
  </>
);
// improve performance when rerendering, because we render a lot of logs
const MemoedLogLine = React.memo(LogLine);

function getLineStyle(line: string): React.CSSProperties {
  const lineLowerCase = line.toLowerCase();
  if (lineLowerCase.indexOf('error') > -1 || lineLowerCase.indexOf('fail') > -1) {
    return {
      backgroundColor: '#700000',
      color: 'white',
    };
  } else if (lineLowerCase.indexOf('warn') > -1) {
    return {
      backgroundColor: '#545400',
      color: 'white',
    };
  } else {
    return {};
  }
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
      <a href={match[0]} target='_blank' rel='noopener noreferrer' className={css.a}>
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
