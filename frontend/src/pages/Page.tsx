/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from 'react';
import { RouteComponentProps } from 'react-router';
import { ToolbarProps } from '../components/Toolbar';
import { BannerProps } from '../components/Banner';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { DialogProps } from '../components/Router';
import { errorToMessage } from '../lib/Utils';
import { TFunction } from 'i18next';

export interface PageProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  updateToolbar: (toolbarProps: Partial<ToolbarProps>) => void;
  t: TFunction;
}

export type PageErrorHandler = (
  message: string,
  error?: Error,
  mode?: 'error' | 'warning',
  refresh?: () => Promise<void>,
) => Promise<void>;

export abstract class Page<P, S> extends React.Component<P & PageProps, S> {
  [x: string]: any;
  protected _isMounted = true;

  constructor(props: any) {
    super(props);
    this.props.updateToolbar(this.getInitialToolbarState());
  }

  public abstract render(): JSX.Element;

  public abstract getInitialToolbarState(): ToolbarProps;

  public abstract refresh(): Promise<void>;

  public componentWillUnmount(): void {
    this.clearBanner();
    this._isMounted = false;
  }

  public componentDidMount(): void {
    this.clearBanner();
  }

  public clearBanner(): void {
    const { t } = this.props;
    if (!this._isMounted) {
      return;
    }
    this.props.updateBanner({ t });
  }

  public showPageError: PageErrorHandler = async (message, error, mode, refresh): Promise<void> => {
    const { t } = this.props;
    const errorMessage = await errorToMessage(error);
    if (!this._isMounted) {
      return;
    }
    this.props.updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + ' ' + (errorMessage ? t('common:clickDetails') : ''),
      mode: mode || 'error',
      refresh: refresh || this.refresh.bind(this),
      t,
    });
  };

  public showErrorDialog(title: string, content: string): void {
    if (!this._isMounted) {
      return;
    }
    this.props.updateDialog({
      buttons: [{ text: 'Dismiss' }],
      content,
      title,
    });
  }

  protected setStateSafe(newState: Partial<S>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }
}
