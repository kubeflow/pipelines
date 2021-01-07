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
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import ErrorIcon from '@material-ui/icons/Error';
import WarningIcon from '@material-ui/icons/Warning';
import InfoIcon from '@material-ui/icons/Info';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, spacing } from '../Css';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export type Mode = 'error' | 'warning' | 'info';

export const css = stylesheet({
  banner: {
    border: `1px solid ${color.divider}`,
    boxSizing: 'border-box',
    justifyContent: 'space-between',
    margin: spacing.units(-1),
    minHeight: '50px',
    padding: spacing.units(-4),
  },
  button: {
    color: color.secondaryText,
    maxHeight: '32px',
    minWidth: '75px',
  },
  detailsButton: {
    backgroundColor: color.background,
  },
  icon: {
    height: '18px',
    padding: spacing.units(-4),
    width: '18px',
  },
  message: {
    alignItems: 'center',
    display: 'flex',
  },
  refreshButton: {
    backgroundColor: color.background,
  },
  troubleShootingLink: {
    alignItems: 'center',
    color: color.theme,
    fontWeight: 'bold',
    padding: spacing.units(-4),
  },
});

export interface BannerProps {
  additionalInfo?: string;
  message?: string;
  mode?: Mode;
  showTroubleshootingGuideLink?: boolean;
  refresh?: () => void;
  t: TFunction;
}

interface BannerState {
  dialogOpen: boolean;
}

class Banner extends React.Component<BannerProps, BannerState> {
  constructor(props: any) {
    super(props);

    this.state = {
      dialogOpen: false,
    };
  }

  public render(): JSX.Element {
    // Default to error styles.

    let bannerModeCss = stylesheet({
      mode: { backgroundColor: color.errorBg, color: color.errorText },
    });
    let bannerIcon = <ErrorIcon className={css.icon} />;
    let dialogTitle = 'An error occurred';
    let showTroubleshootingGuideLink = false;
    let showRefreshButton = true;

    switch (this.props.mode) {
      case 'error':
        bannerModeCss = stylesheet({
          mode: { backgroundColor: color.errorBg, color: color.errorText },
        });
        bannerIcon = <ErrorIcon className={css.icon} />;
        dialogTitle = this.props.t('common:errorOccurred');
        showTroubleshootingGuideLink = this.props.showTroubleshootingGuideLink || false;
        break;
      case 'warning':
        bannerModeCss = stylesheet({
          mode: { backgroundColor: color.warningBg, color: color.warningText },
        });
        bannerIcon = <WarningIcon className={css.icon} />;
        dialogTitle = 'Warning';
        break;
      case 'info':
        bannerModeCss = stylesheet({
          mode: { backgroundColor: color.infoBg, color: color.infoText },
        });
        bannerIcon = <InfoIcon className={css.icon} />;
        dialogTitle = 'Info';
        showRefreshButton = false;
        break;
      default:
        // Already set defaults above.
        break;
    }

    return (
      <div className={classes(commonCss.flex, css.banner, bannerModeCss.mode)}>
        <div className={css.message}>
          {bannerIcon}
          {this.props.message}
        </div>
        <div className={commonCss.flex}>
          {showTroubleshootingGuideLink && (
            <a
              className={css.troubleShootingLink}
              href='https://www.kubeflow.org/docs/pipelines/troubleshooting'
            >
              {this.props.t('common:troubleshooting')}
            </a>
          )}
          {this.props.additionalInfo && (
            <Button
              className={classes(css.button, css.detailsButton)}
              onClick={this._showAdditionalInfo.bind(this)}
            >
              {this.props.t('common:detailsLowercase')}
            </Button>
          )}
          {showRefreshButton && this.props.refresh && (
            <Button
              className={classes(css.button, css.refreshButton)}
              onClick={this._refresh.bind(this)}
            >
              {this.props.t('common:refresh')}
            </Button>
          )}
        </div>

        {this.props.additionalInfo && (
          <Dialog open={this.state.dialogOpen} onClose={this._dialogClosed.bind(this)}>
            <DialogTitle>{dialogTitle}</DialogTitle>
            <DialogContent className={commonCss.prewrap}>{this.props.additionalInfo}</DialogContent>
            <DialogActions>
              <Button id='dismissDialogBtn' onClick={this._dialogClosed.bind(this)}>
                Dismiss
              </Button>
            </DialogActions>
          </Dialog>
        )}
      </div>
    );
  }

  private _dialogClosed(): void {
    this.setState({ dialogOpen: false });
  }

  private _showAdditionalInfo(): void {
    this.setState({ dialogOpen: true });
  }

  private _refresh(): void {
    this.props.refresh!();
  }
}

export default withTranslation('common')(Banner);
