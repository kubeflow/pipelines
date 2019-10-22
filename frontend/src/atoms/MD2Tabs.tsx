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
import Button from '@material-ui/core/Button';
import Separator from './Separator';
import { color, fontsize } from '../Css';
import { classes, stylesheet } from 'typestyle';
import { logger } from '../lib/Utils';

interface MD2TabsProps {
  onSwitch?: (tab: number) => void;
  selectedTab: number;
  tabs: string[];
}

const css = stylesheet({
  active: {
    color: `${color.theme} !important`,
    fontWeight: 'bold',
  },
  button: {
    $nest: {
      '&:hover': {
        backgroundColor: 'initial',
      },
    },
    borderRadius: '4px 4px 0 0',
    color: color.inactive,
    fontSize: fontsize.base,
    height: 33,
    minHeight: 0,
    padding: '0 15px',
    textTransform: 'none',
  },
  indicator: {
    backgroundColor: 'initial',
    borderBottom: '3px solid ' + color.theme,
    borderLeft: '3px solid transparent',
    borderRight: '3px solid transparent',
    bottom: 0,
    left: 35,
    position: 'absolute',
    transition: 'width 0.3s, left 0.3s',
    width: 50,
  },
  tabs: {
    borderBottom: '1px solid ' + color.separator,
    height: 33,
    position: 'relative',
    whiteSpace: 'nowrap',
  },
});

class MD2Tabs extends React.Component<MD2TabsProps, any> {
  private _rootRef = React.createRef<any>();
  private _indicatorRef = React.createRef<any>();
  private _tabRefs = this.props.tabs.map(t => React.createRef<HTMLSpanElement>());
  private _timeoutHandle = 0;

  public render(): JSX.Element {
    const selected = this._getSelectedIndex();
    const switchHandler = this.props.onSwitch || (() => null);
    return (
      <div className={css.tabs} ref={this._rootRef}>
        <div className={css.indicator} ref={this._indicatorRef} />
        <Separator units={20} />
        {this.props.tabs.map((tab, i) => (
          <Button
            className={classes(css.button, i === selected ? css.active : '')}
            key={i}
            onClick={() => {
              if (i !== selected) {
                switchHandler(i);
              }
            }}
          >
            <span ref={this._tabRefs[i]}>{tab}</span>
          </Button>
        ))}
      </div>
    );
  }

  public componentDidMount(): void {
    this._timeoutHandle = setTimeout(this._updateIndicator.bind(this));
  }

  public componentDidUpdate(): void {
    this._timeoutHandle = setTimeout(this._updateIndicator.bind(this));
  }

  public componentWillUnmount(): void {
    clearTimeout(this._timeoutHandle);
  }

  private _getSelectedIndex(): number {
    let selected = this.props.selectedTab;
    if (this.props.tabs[selected] === undefined) {
      logger.error('Out of bound index passed for selected tab');
      selected = 0;
    }
    return selected;
  }

  private _updateIndicator(): void {
    const selected = this._getSelectedIndex();
    const activeLabelElement = this._tabRefs[selected].current as HTMLSpanElement;
    if (!activeLabelElement) {
      return;
    }
    const leftOffset =
      activeLabelElement.getBoundingClientRect().left -
      this._rootRef.current.getBoundingClientRect().left;

    const tabIndicator = this._indicatorRef.current;
    tabIndicator.style.left = leftOffset - 5 + 'px';
    tabIndicator.style.width = activeLabelElement.getBoundingClientRect().width + 5 + 'px';
    tabIndicator.style.display = 'block';
  }
}

export default MD2Tabs;
