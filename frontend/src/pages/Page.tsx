/*
 * Copyright 2018 The Kubeflow Authors
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
import { Location, NavigateFunction, Params, useLocation, useNavigate, useParams } from 'react-router-dom';
import { ToolbarProps } from '../components/Toolbar';
import { BannerProps } from '../components/Banner';
import { SnackbarProps } from '@mui/material/Snackbar';
import { DialogProps } from '../components/Router';
import { errorToMessage } from '../lib/Utils';

// React Router v6 compatible route props
export interface RouteComponentProps {
  location: Location;
  navigate: NavigateFunction;
  params: Params;
  // Compatibility shim for v4 patterns
  match: {
    params: Params;
    path: string;
    url: string;
  };
  history: {
    push: (path: string) => void;
    replace: (path: string) => void;
    goBack: () => void;
  };
}

export interface PageProps extends RouteComponentProps {
  toolbarProps: ToolbarProps;
  updateBanner: (bannerProps: BannerProps) => void;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  updateToolbar: (toolbarProps: Partial<ToolbarProps>) => void;
}

export type PageErrorHandler = (
  message: string,
  error?: unknown,
  mode?: 'error' | 'warning',
  refresh?: () => Promise<void>,
) => Promise<void>;

export abstract class Page<P, S> extends React.Component<P & PageProps, S> {
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
    if (!this._isMounted) {
      return;
    }
    this.props.updateBanner({});
  }

  public showPageError: PageErrorHandler = async (message, error, mode, refresh): Promise<void> => {
    const errorMessage = await errorToMessage(error);
    if (!this._isMounted) {
      return;
    }
    this.props.updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
      mode: mode || 'error',
      refresh: refresh || this.refresh.bind(this),
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

/**
 * HOC that provides React Router v6 hooks as props for class components.
 * This is a compatibility shim for migrating from React Router v4 to v6.
 */
export function withRouter<P extends RouteComponentProps>(
  Component: React.ComponentType<P>,
): React.FC<Omit<P, keyof RouteComponentProps>> {
  return function WithRouterWrapper(props: Omit<P, keyof RouteComponentProps>) {
    const location = useLocation();
    const navigate = useNavigate();
    const params = useParams();

    const routeProps: RouteComponentProps = {
      location,
      navigate,
      params,
      match: {
        params,
        path: location.pathname,
        url: location.pathname,
      },
      history: {
        push: navigate,
        replace: (path: string) => navigate(path, { replace: true }),
        goBack: () => navigate(-1),
      },
    };

    return <Component {...(props as P)} {...routeProps} />;
  };
}
