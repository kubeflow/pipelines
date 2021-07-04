/*
 * Copyright 2021 The Kubeflow Authors
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
import { Link } from 'react-router-dom';
import { classes, cssRaw } from 'typestyle';
import { LineageTypedResource } from './LineageTypes';
import { getResourceDescription, getResourceName } from './Utils';
import { Artifact } from 'src/third_party/mlmd';

cssRaw(`
.cardRow {
  align-items: center;
  border-bottom: 1px solid var(--grey-200);
  display: flex;
  height: 54px;
  padding: 6px 0px;
  position: relative;
}

.cardRow .noRadio {
  height: 16px;
  width: 16px;
}

.cardRow.clickTarget {
  cursor: pointer;
}

.cardRow .form-radio-container {
  position: relative;
}

.cardRow .form-radio {
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
  display: inline-block;
  position: relative;
  background-color: #fff;
  border: 1px solid var(--grey-400);
  color: var(--blue-500);
  top: 0px;
  height: 18px;
  width: 18px;
  border-radius: 50px;
  cursor: pointer;
  outline: none;
}

.cardRow .form-radio:checked {
  border: 2px solid var(--blue-500);
}

.cardRow .form-radio:checked::before {
  position: absolute;
  border-radius: 50%;
  top: 50%;
  left: 50%;
  content: '';
  transform: translate(-50%, -50%);
  padding: 5px;
  display: block;
  background: currentColor;
}

.cardRow .form-radio:hover {
  background-color: var(--grey-100);
}

.cardRow .form-radio:checked {
  background-color: #fff;
}

.cardRow .form-radio.hover-hint {
  color: var(--grey-400);
  left: 0;
  opacity: 0;
  position: absolute;
}

.cardRow.clickTarget:hover .form-radio.hover-hint {
  opacity: 1;
}

.cardRow.clickTarget .form-radio.hover-hint:checked::before {
  background: currentColor;
}

.cardRow.clickTarget .form-radio.hover-hint:checked {
  border: 2px solid var(--grey-400);
}

.cardRow div {
  display: inline-block;
  vertical-align: middle;
}

.cardRow div input {
  margin: 10px 10px 0 20px;
}

.cardRow .rowTitle {
  font-size: 12px;
  font-family: "PublicSans-SemiBold";
  color: var(--grey-900);
  letter-spacing: 0.2px;
  line-height: 24px;
  text-decoration: none;
  text-overflow: ellipsis;
  display: inline-block;
  white-space: nowrap;
  overflow: hidden;
}

.cardRow .rowTitle:hover {
  text-decoration: underline;
  color: var(--blue-600);
  cursor: pointer;
}

.cardRow .rowDesc {
  font-size: 11px;
  color: var(--grey-600);
  letter-spacing: 0.3px;
  line-height: 12px;
}
.cardRow footer {
  overflow: hidden;
}
.cardRow [class^='edge'] {
  width: 8px;
  height: 8px;
  background-color: var(--grey-700);
  border-radius: 2px;
  position: absolute;
  top: 50%;
  transform: translateY(-50%) translateX(-50%);
}
.cardRow .edgeRight {
  left: 100%;
}

.cardRow .edgeLeft {
  left: 0;
}

.cardRow.lastRow {
  border-bottom: 0px;
}

`);

const CLICK_TARGET_CSS_NAME = 'clickTarget';

interface LineageCardRowProps {
  leftAffordance: boolean;
  rightAffordance: boolean;
  hideRadio: boolean;
  isLastRow: boolean;
  isTarget?: boolean;
  typedResource: LineageTypedResource;
  resourceDetailsRoute: string;
  setLineageViewTarget?(artifact: Artifact): void;
}

export class LineageCardRow extends React.Component<LineageCardRowProps> {
  private rowContainerRef: React.RefObject<HTMLDivElement> = React.createRef();

  constructor(props: LineageCardRowProps) {
    super(props);
    this.handleClick = this.handleClick.bind(this);
  }

  public checkEdgeAffordances(): JSX.Element[] {
    const affItems = [];
    this.props.leftAffordance && affItems.push(<div className='edgeLeft' key={'edgeLeft'} />);
    this.props.rightAffordance && affItems.push(<div className='edgeRight' key={'edgeRight'} />);
    return affItems;
  }

  public render(): JSX.Element {
    const { isLastRow } = this.props;
    const cardRowClasses = classes(
      'cardRow',
      isLastRow && 'lastRow',
      this.canTarget() && 'clickTarget',
    );

    return (
      <div className={cardRowClasses} ref={this.rowContainerRef} onClick={this.handleClick}>
        {this.checkRadio()}
        <footer>
          <Link
            className={'rowTitle'}
            to={this.props.resourceDetailsRoute}
            onMouseEnter={this.handleMouseEnter}
            onMouseLeave={this.handleMouseLeave}
          >
            {getResourceName(this.props.typedResource)}
          </Link>
          <p className='rowDesc'>{getResourceDescription(this.props.typedResource)}</p>
        </footer>
        {this.checkEdgeAffordances()}
      </div>
    );
  }

  private checkRadio(): JSX.Element {
    if (this.props.hideRadio) {
      return <div className='noRadio' />;
    }

    return (
      <div className={'form-radio-container'}>
        <input
          type='radio'
          className='form-radio'
          name=''
          value=''
          onClick={this.handleClick}
          checked={this.props.isTarget}
          readOnly={true}
        />
        <input
          type='radio'
          className='form-radio hover-hint'
          name=''
          value=''
          onClick={this.handleClick}
          checked={true}
          readOnly={true}
        />
      </div>
    );
  }

  private handleClick() {
    if (!this.props.setLineageViewTarget || !(this.props.typedResource.type === 'artifact')) return;
    this.props.setLineageViewTarget(this.props.typedResource.resource as Artifact);
  }

  private handleMouseEnter = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (!e || !e.target || !this.canTarget()) return;

    const element = e.target as HTMLAnchorElement;
    if (element.className.match(/\browTitle\b/)) {
      this.showRadioHint(false);
    }
  };

  private handleMouseLeave = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (!e || !e.target || !this.canTarget()) return;

    const element = e.target as HTMLAnchorElement;
    if (element.className.match(/\browTitle\b/)) {
      this.showRadioHint(true);
    }
  };

  private showRadioHint = (show: boolean) => {
    if (this.props.isTarget || !this.rowContainerRef.current) return;

    const rowContainer = this.rowContainerRef.current;

    if (show) {
      if (!rowContainer.classList.contains(CLICK_TARGET_CSS_NAME)) {
        rowContainer.classList.add(CLICK_TARGET_CSS_NAME);
      }
    } else {
      if (rowContainer.classList.contains(CLICK_TARGET_CSS_NAME)) {
        rowContainer.classList.remove(CLICK_TARGET_CSS_NAME);
      }
    }
  };

  private canTarget = () => {
    const { isTarget, typedResource } = this.props;
    return !isTarget && typedResource.type === 'artifact';
  };
}
