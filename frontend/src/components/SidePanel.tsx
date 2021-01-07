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
import CircularProgress from '@material-ui/core/CircularProgress';
import CloseIcon from '@material-ui/icons/Close';
import Resizable from 're-resizable';
import Slide from '@material-ui/core/Slide';
import { color, commonCss, zIndex } from '../Css';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  closeButton: {
    color: color.inactive,
    margin: 15,
    minHeight: 0,
    minWidth: 0,
    padding: 0,
  },
  nodeName: {
    flexGrow: 1,
    textAlign: 'center',
  },
  sidepane: {
    backgroundColor: color.background,
    borderLeft: 'solid 1px #ddd',
    bottom: 0,
    display: 'flex',
    flexFlow: 'column',
    position: 'absolute !important' as any,
    right: 0,
    top: 0,
    zIndex: zIndex.SIDE_PANEL,
  },
});

interface SidePanelProps {
  isBusy?: boolean;
  isOpen: boolean;
  onClose: () => void;
  title: string;
}

class SidePanel extends React.Component<SidePanelProps> {
  public render(): JSX.Element {
    const { isBusy, isOpen, onClose, title } = this.props;
    return (
      <Slide in={isOpen} direction='left'>
        <Resizable
          className={css.sidepane}
          defaultSize={{ width: '70%' }}
          maxWidth='90%'
          minWidth={100}
          enable={{
            bottom: false,
            bottomLeft: false,
            bottomRight: false,
            left: true,
            right: false,
            top: false,
            topLeft: false,
            topRight: false,
          }}
        >
          {isOpen && (
            <div className={commonCss.page}>
              <div className={commonCss.flex}>
                <Button className={css.closeButton} onClick={onClose}>
                  <CloseIcon />
                </Button>
                <div className={css.nodeName}>{title}</div>
              </div>
              <div className={commonCss.page}>
                {isBusy === true && (
                  <CircularProgress size={30} className={commonCss.absoluteCenter} />
                )}

                <div className={commonCss.page}>{this.props.children}</div>
              </div>
            </div>
          )}
        </Resizable>
      </Slide>
    );
  }
}
export default SidePanel;
