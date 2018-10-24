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
    marginRight: 10,
    textAlign: 'right',
    userSelect: 'none',
  },
  root: {
    $nest: {
      '& .ReactVirtualized__Grid__innerScrollContainer': {
        overflow: 'initial !important',
      },
    },
    backgroundColor: '#222',
    color: '#fff',
    fontFamily: fonts.code,
    fontSize: fontsize.small,
    padding: 10,
    whiteSpace: 'pre',
  },
});

interface LogViewerProps {
  classes?: string;
  logLines: string[];
}

class LogViewer extends React.Component<LogViewerProps> {
  private _rootRef = React.createRef<List>();

  public componentDidMount() {
    this._scrollToEnd();
  }

  public componentDidUpdate() {
    this._scrollToEnd();
  }

  public render() {
    return <AutoSizer>
      {({ height, width }) => (
        <List width={width} height={height} rowCount={this.props.logLines.length}
          rowHeight={15} className={css.root} ref={this._rootRef}
          rowRenderer={this._rowRenderer.bind(this)} />
      )}
    </AutoSizer>;
  }

  private _scrollToEnd() {
    const root = this._rootRef.current;
    if (root) {
      root.scrollToRow(this.props.logLines.length + 1);
    }
  }

  private _rowRenderer(props: ListRowProps): React.ReactNode {
    const { style, key, index } = props;
    return (
      <div key={key} className={css.line}
        style={this._getLineNumberStyle(this.props.logLines[index], style)}>
        <span className={css.number}>{index + 1}</span>
        {this._parseLine(this.props.logLines[index]).map((piece, p) => (
          <span key={p}>{piece}</span>
        ))}
      </div>
    );
  }

  private _getLineNumberStyle(line: string, oldStyle: React.CSSProperties): React.CSSProperties {
    const lineLowerCase = line.toLowerCase();
    const style = { ...oldStyle };
    if (lineLowerCase.indexOf('error') > -1 || lineLowerCase.indexOf('fail') > -1) {
      style.backgroundColor = '#700000';
      style.color = 'white';
    } else if (lineLowerCase.indexOf('warn') > -1) {
      style.backgroundColor = '#545400';
      style.color = 'white';
    }
    return style;
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
