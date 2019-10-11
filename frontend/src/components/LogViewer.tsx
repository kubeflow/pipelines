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
    backgroundColor: '#222',
    color: '#fff',
    fontFamily: fonts.code,
    fontSize: fontsize.small,
    padding: '10px 0',
    whiteSpace: 'pre',
  },
});

interface LogViewerProps {
  classes?: string;
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
  direction,          // One of "horizontal" or "vertical"
  cellCount,          // Number of rows or columns in the current axis
  scrollDirection,    // 1 (forwards) or -1 (backwards)
  overscanCellsCount, // Maximum number of cells to over-render in either direction
  startIndex,         // Begin of range of visible cells
  stopIndex           // End of range of visible cells
}) => {
  return ({
    overscanStartIndex: Math.max(0, startIndex - overscanCellsCount),
    overscanStopIndex: Math.min(cellCount - 1, stopIndex + overscanCellsCount)
  });
};

class LogViewer extends React.Component<LogViewerProps> {
  private _rootRef = React.createRef<List>();

  public componentDidMount(): void {
    this._scrollToEnd();
  }

  public componentDidUpdate(): void {
    this._scrollToEnd();
  }

  public render(): JSX.Element {
    return <AutoSizer>
      {({ height, width }) => (
        <List
          id='logViewer'
          width={width} height={height} rowCount={this.props.logLines.length}
          rowHeight={15} className={css.root} ref={this._rootRef}
          overscanIndicesGetter={overscanOnBothDirections}
          overscanRowCount={1000 /* make this large, so selecting maximum 1000 lines is supported */}
          rowRenderer={this._rowRenderer.bind(this)}
        />
      )}
    </AutoSizer>;
  }

  private _scrollToEnd(): void {
    const root = this._rootRef.current;
    if (root) {
      root.scrollToRow(this.props.logLines.length + 1);
    }
  }

  private _rowRenderer(props: ListRowProps): React.ReactNode {
    const { style, key, index } = props;
    return (
      <div key={key} className={css.line} style={style}>
        <span className={css.number} style={this._getLineStyle(index)}>{index + 1}</span>

        <span className={css.line} style={this._getLineStyle(index)}>
          {this._parseLine(this.props.logLines[index]).map((piece, p) => (
            <span key={p}>{piece}</span>
          ))}
        </span>
      </div>
    );
  }

  private _getLineStyle(index: number): React.CSSProperties {
    const line = this.props.logLines[index];
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

  private _parseLine(line: string): React.ReactNode[] {
    // Linkify URLs starting with http:// or https://
    const urlPattern = /(\b(https?):\/\/[-A-Z0-9+&@#\/%?=~_|!:,.;]*[-A-Z0-9+&@#\/%=~_|])/gim;
    let lastMatch = 0;
    let match = urlPattern.exec(line);
    const nodes = [];
    while (match) {
      // Append all text before URL match
      nodes.push(<span>{line.substr(lastMatch, match.index)}</span>);
      // Append URL via an anchor element
      nodes.push(<a href={match[0]} target='_blank' className={css.a}>{match[0]}</a>);

      lastMatch = match.index + match[0].length;
      match = urlPattern.exec(line);
    }
    // Append all text after final URL
    nodes.push(<span>{line.substr(lastMatch)}</span>);
    return nodes;
  }
}

export default LogViewer;
