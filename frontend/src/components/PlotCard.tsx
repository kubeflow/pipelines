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
import CloseIcon from '@material-ui/icons/Close';
import Dialog from '@material-ui/core/Dialog';
import Paper from '@material-ui/core/Paper';
import PopOutIcon from '@material-ui/icons/Launch';
import Separator from '../atoms/Separator';
import Tooltip from '@material-ui/core/Tooltip';
import ViewerContainer, { componentMap } from '../components/viewers/ViewerContainer';
import { ViewerConfig } from '../components/viewers/Viewer';
import { color, fontsize } from '../Css';
import { stylesheet, classes } from 'typestyle';
import { isEqual } from 'lodash';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

const css = stylesheet({
  dialogTitle: {
    color: color.strong,
    fontSize: fontsize.large,
    width: '100%',
  },
  fullscreenCloseBtn: {
    minHeight: 0,
    minWidth: 0,
    padding: 3,
  },
  fullscreenDialog: {
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: '80%',
    minWidth: '80%',
    padding: 20,
  },
  fullscreenViewerContainer: {
    alignItems: 'center',
    boxSizing: 'border-box',
    display: 'flex',
    flexFlow: 'column',
    flexGrow: 1,
    height: '100%',
    justifyContent: 'center',
    margin: 20,
    overflow: 'auto',
    width: '100%',
  },
  plotCard: {
    flexShrink: 0,
    margin: 20,
    minWidth: 250,
    padding: 20,
    width: 'min-content',
  },
  plotHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    overflow: 'hidden',
    paddingBottom: 10,
  },
  plotTitle: {
    color: color.strong,
    fontSize: 12,
    fontWeight: 'bold',
  },
  popoutIcon: {
    fontSize: 18,
  },
});

export interface PlotCardProps {
  title: string;
  configs: ViewerConfig[];
  maxDimension: number;
  t: TFunction;
}

interface PlotCardState {
  fullscreenDialogOpen: boolean;
}

class PlotCard extends React.Component<PlotCardProps, PlotCardState> {
  constructor(props: any) {
    super(props);

    this.state = {
      fullscreenDialogOpen: false,
    };
  }

  public shouldComponentUpdate(nextProps: PlotCardProps, nextState: PlotCardState): boolean {
    return (
      !isEqual(nextProps, this.props) ||
      nextState.fullscreenDialogOpen !== this.state.fullscreenDialogOpen
    );
  }

  public render(): JSX.Element | null {
    const { title, configs, maxDimension, t, ...otherProps } = this.props;

    if (!configs || !configs.length) {
      return null;
    }

    return (
      <div>
        <Paper {...otherProps} className={classes(css.plotCard, 'plotCard')}>
          <div className={css.plotHeader}>
            <div className={css.plotTitle} title={title}>
              {title}
            </div>
            <div>
              <Button
                onClick={() => this.setState({ fullscreenDialogOpen: true })}
                style={{ padding: 4, minHeight: 0, minWidth: 0 }}
                className='popOutButton'
              >
                <Tooltip title={t('popOut')}>
                  <PopOutIcon classes={{ root: css.popoutIcon }} />
                </Tooltip>
              </Button>
            </div>
          </div>
          <ViewerContainer configs={configs} maxDimension={maxDimension} />
        </Paper>

        <Dialog
          open={!!this.state.fullscreenDialogOpen}
          classes={{ paper: css.fullscreenDialog }}
          onClose={() => this.setState({ fullscreenDialogOpen: false })}
        >
          <div className={css.dialogTitle}>
            <Button
              onClick={() => this.setState({ fullscreenDialogOpen: false })}
              className={classes(css.fullscreenCloseBtn, 'fullscreenCloseButton')}
            >
              <CloseIcon />
            </Button>
            {t(componentMap[configs[0].type].displayNameKey)}
            <Separator />
            <span style={{ color: color.inactive }}>({title})</span>
          </div>
          <div className={css.fullscreenViewerContainer}>
            <ViewerContainer configs={configs} />
          </div>
        </Dialog>
      </div>
    );
  }
}
export default withTranslation('common')(PlotCard);
