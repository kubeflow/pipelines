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
import InputLabel from '@material-ui/core/InputLabel';
import Input from '@material-ui/core/Input';
import MenuItem from '@material-ui/core/MenuItem';
import ListSubheader from '@material-ui/core/ListSubheader';
import FormControl from '@material-ui/core/FormControl';
import Select from '@material-ui/core/Select';

export interface TensorboardViewerConfig extends ViewerConfig {
  url: string;
}

interface TensorboardViewerProps {
  configs: TensorboardViewerConfig[];
}

interface TensorboardViewerState {
  busy: boolean;
  podAddress: string;
  tensorflowVersion: string;
}

class TensorboardViewer extends Viewer<TensorboardViewerProps, TensorboardViewerState> {
  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      podAddress: '',
      tensorflowVersion: '1.14.0',
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

  public onChangeFunc = (e: React.ChangeEvent<{ name?: string; value: unknown }>): void => {
    this.setState({ tensorflowVersion: e.target.value as string });
  };

  public render(): JSX.Element {
    // Strip the protocol from the URL. This is a workaround for cloud shell
    // incorrectly decoding the address and replacing the protocol's // with /.
    // Pod address (after stripping protocol) is of the format
    // <viewer_service_dns>.kubeflow.svc.cluster.local:6006/tensorboard/<viewer_name>/
    // We use this pod address without encoding since encoded pod address failed to open the
    // tensorboard instance on this pod.
    // TODO: figure out why the encoded pod address failed to open the tensorboard.
    const podAddress = this.state.podAddress.replace(/(^\w+:|^)\/\//, '');

    return (
      <div>
        {this.state.podAddress && (
          <div>
            <div className={padding(20, 'b')}>{`Tensorboard is running for this output.`}</div>
            <a
              href={'apis/v1beta1/_proxy/' + podAddress}
              target='_blank'
              className={commonCss.unstyled}
            >
              <Button 
              className={commonCss.buttonAction} disabled={this.state.busy}
              style = {{marginBottom: 20}}>
                Open Tensorboard
              </Button>
            </a>

            <div>
              <BusyButton
                className={commonCss.buttonAction}
                onClick={this._deleteTensorboard.bind(this)}
                busy={this.state.busy}
                title={`Stop Tensorboard`} //pop out dialog: this tensorboard would be deleted 
              />
            </div>

          </div>

        )}

        {!this.state.podAddress && (
          <div>
            <div className={padding(30, 'b')}>
              <FormControl className='Launch Tensorboard' style={{ minWidth: 120 }}>
                <InputLabel htmlFor='grouped-select'>TF Version</InputLabel>
                <Select
                  defaultValue={this.state.tensorflowVersion}
                  value={this.state.tensorflowVersion}
                  input={<Input id='grouped-select' />}
                  onChange={this.onChangeFunc}
                  style={{
                    minHeight: 40,
                  }}
                >
                  <ListSubheader>Tensoflow 1.x</ListSubheader>
                  <MenuItem value={'1.4.0'}>TensorFlow 1.4.0</MenuItem>
                  <MenuItem value={'1.5.0'}>TensorFlow 1.5.0</MenuItem>
                  <MenuItem value={'1.6.0'}>TensorFlow 1.6.0</MenuItem>
                  <MenuItem value={'1.7.0'}>TensorFlow 1.7.0</MenuItem>
                  <MenuItem value={'1.8.0'}>TensorFlow 1.8.0</MenuItem>
                  <MenuItem value={'1.9.0'}>TensorFlow 1.9.0</MenuItem>
                  <MenuItem value={'1.10.0'}>TensorFlow 1.10.0</MenuItem>
                  <MenuItem value={'1.11.0'}>TensorFlow 1.11.0</MenuItem>
                  <MenuItem value={'1.12.0'}>TensorFlow 1.12.0</MenuItem>
                  <MenuItem value={'1.13.2'}>TensorFlow 1.13.2</MenuItem>
                  <MenuItem value={'1.14.0'}>TensorFlow 1.14.0</MenuItem>
                  <MenuItem value={'1.15.0'}>TensorFlow 1.15.0</MenuItem>
                  <ListSubheader>TensorFlow 2.x</ListSubheader>
                  <MenuItem value={'2.0.0'}>TensorFlow 2.0.0</MenuItem>
                </Select>
              </FormControl>
            </div>
            <div>
              <BusyButton
                className={commonCss.buttonAction}
                onClick={this._startTensorboard.bind(this)}
                busy={this.state.busy}
                title={`Start ${this.props.configs.length > 1 ? 'Combined ' : ''}Tensorboard`}
              />
            </div>
      
          </div>
        )}
      </div>
    );
  }

  private _buildUrl(): string {
    const urls = this.props.configs.map(c => c.url).sort();
    return urls.length === 1 ? urls[0] : urls.map((c, i) => `Series${i + 1}:` + c).join(',');
  }

  private async _checkTensorboardApp(): Promise<void> {
    this.setState({ busy: true }, async () => {
      const podAddress = await Apis.getTensorboardApp(
        this._buildUrl(),
        this.state.tensorflowVersion,
      );
      this.setState({ busy: false, podAddress });
    });
  }

  private async _startTensorboard(): Promise<void> {
    this.setState({ busy: true }, async () => {
      await Apis.startTensorboardApp(
        encodeURIComponent(this._buildUrl()),
        encodeURIComponent(this.state.tensorflowVersion),
      );
      this.setState({ busy: false }, () => {
        this._checkTensorboardApp();
      });
    });
  }

  private async _deleteTensorboard(): Promise<void> {
    // delete the already opened Tensorboard, clear the podAddress recorded in frontend,
    // and return to the select & start tensorboard page
    this.setState({ busy: true }, async () => {
      await Apis.deleteTensorboardApp(this._buildUrl(), this.state.tensorflowVersion);
      this.setState({ busy: false, podAddress: '' });
    });
  }
}

export default TensorboardViewer;
