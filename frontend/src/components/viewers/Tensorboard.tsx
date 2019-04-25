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
import BusyButton from '../../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Viewer, { ViewerConfig } from './Viewer';
import { Apis } from '../../lib/Apis';
import { commonCss, padding } from '../../Css';

export interface TensorboardViewerConfig extends ViewerConfig {
  url: string;
}

interface TensorboardViewerProps {
  configs: TensorboardViewerConfig[];
}

interface TensorboardViewerState {
  busy: boolean;
  podAddress: string;
}

class TensorboardViewer extends Viewer<TensorboardViewerProps, TensorboardViewerState> {
  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      podAddress: '',
    };
  }

  public getDisplayName(): string {
    return 'Tensorboard';
  }

  public isAggregatable(): boolean {
    return true;
  }

  public componentDidMount(): void {
    this._checkTensorboardApp();
  }

  public render(): JSX.Element {
    // Strip the protocol from the URL. This is a workaround for cloud shell
    // incorrectly decoding the address and replacing the protocol's // with /.
    const podAddress = encodeURIComponent(this.state.podAddress.replace(/(^\w+:|^)\/\//, ''));

    return <div>
      {this.state.podAddress && <div>
        <div className={padding(20, 'b')}>Tensorboard is running for this output.</div>
        <a href={'apis/v1beta1/_proxy/' + podAddress} target='_blank' className={commonCss.unstyled}>
          <Button className={commonCss.buttonAction} disabled={this.state.busy}>Open Tensorboard</Button>
        </a>
      </div>}

      {!this.state.podAddress &&
        <BusyButton className={commonCss.buttonAction} onClick={this._startTensorboard.bind(this)}
          busy={this.state.busy}
          title={`Start ${this.props.configs.length > 1 ? 'Combined ' : ''}Tensorboard`} />
      }
    </div>;
  }

  private _buildUrl(): string {
    const urls = this.props.configs.map(c => c.url).sort();
    return urls.length === 1 ? urls[0] : urls.map((c, i) => `Series${i + 1}:` + c).join(',');
  }

  private async _checkTensorboardApp(): Promise<void> {
    this.setState({ busy: true }, async () => {
      const podAddress = await Apis.getTensorboardApp(this._buildUrl());
      this.setState({ busy: false, podAddress });
    });
  }

  private async _startTensorboard(): Promise<void> {
    this.setState({ busy: true }, async () => {
      await Apis.startTensorboardApp(encodeURIComponent(this._buildUrl()));
      this.setState({ busy: false }, () => {
        this._checkTensorboardApp();
      });
    });
  }
}

export default TensorboardViewer;
