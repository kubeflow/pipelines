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
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogContentText from '@material-ui/core/DialogContentText';
import DialogTitle from '@material-ui/core/DialogTitle';
import { classes, stylesheet } from 'typestyle';

export const css = stylesheet({
  button: {
    marginBottom: 20,
    width: 150,
  },
  formControl: {
    minWidth: 120,
  },
  select: {
    minHeight: 50,
  },
  shortButton: {
    width: 50,
  },
});

export interface TensorboardViewerConfig extends ViewerConfig {
  url: string;
}

interface TensorboardViewerProps {
  configs: TensorboardViewerConfig[];
}

interface TensorboardViewerState {
  busy: boolean;
  deleteDialogOpen: boolean;
  podAddress: string;
  tensorflowVersion: string;
}

// TODO(jingzhang36): we'll later parse Tensorboard version from mlpipeline-ui-metadata.json file.
const DEFAULT_TENSORBOARD_VERSION = '2.0.0';

class TensorboardViewer extends Viewer<TensorboardViewerProps, TensorboardViewerState> {
  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      deleteDialogOpen: false,
      podAddress: '',
      tensorflowVersion: DEFAULT_TENSORBOARD_VERSION,
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

  public handleVersionSelect = (e: React.ChangeEvent<{ name?: string; value: unknown }>): void => {
    if (typeof e.target.value !== 'string') {
      throw new Error('Invalid event value type, expected string');
    }
    this.setState({ tensorflowVersion: e.target.value });
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
            <div
              className={padding(20, 'b')}
            >{`Tensorboard ${this.state.tensorflowVersion} is running for this output.`}</div>
            <a
              href={'apis/v1beta1/_proxy/' + podAddress}
              target='_blank'
              className={commonCss.unstyled}
            >
              <Button
                className={classes(commonCss.buttonAction, css.button)}
                disabled={this.state.busy}
                color={'primary'}
              >
                Open Tensorboard
              </Button>
            </a>

            <div>
              <Button
                className={css.button}
                disabled={this.state.busy}
                id={'delete'}
                title={`stop tensorboard and delete its instance`}
                onClick={this._handleDeleteOpen}
                color={'default'}
              >
                Delete Tensorboard
              </Button>
              <Dialog
                open={this.state.deleteDialogOpen}
                onClose={this._handleDeleteClose}
                aria-labelledby='dialog-title'
              >
                <DialogTitle id='dialog-title'>
                  {`Stop Tensorboard ${this.state.tensorflowVersion}?`}
                </DialogTitle>
                <DialogContent>
                  <DialogContentText>
                    You can stop the current running tensorboard. The tensorboard viewer will also
                    be deleted from your workloads.
                  </DialogContentText>
                </DialogContent>
                <DialogActions>
                  <Button
                    className={css.shortButton}
                    id={'cancel'}
                    autoFocus={true}
                    onClick={this._handleDeleteClose}
                    color='primary'
                  >
                    Cancel
                  </Button>
                  <BusyButton
                    className={classes(commonCss.buttonAction, css.shortButton)}
                    onClick={this._deleteTensorboard}
                    busy={this.state.busy}
                    color='primary'
                    title={`Stop`}
                  />
                </DialogActions>
              </Dialog>
            </div>
          </div>
        )}

        {!this.state.podAddress && (
          <div>
            <div className={padding(30, 'b')}>
              <FormControl className={css.formControl}>
                <InputLabel htmlFor='grouped-select'>TF Version</InputLabel>
                <Select
                  className={css.select}
                  value={this.state.tensorflowVersion}
                  input={<Input id='grouped-select' />}
                  onChange={this.handleVersionSelect}
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
                disabled={!this.state.tensorflowVersion}
                onClick={this._startTensorboard}
                busy={this.state.busy}
                title={`Start ${this.props.configs.length > 1 ? 'Combined ' : ''}Tensorboard`}
              />
            </div>
          </div>
        )}
      </div>
    );
  }

  private _handleDeleteOpen = () => {
    this.setState({ deleteDialogOpen: true });
  };

  private _handleDeleteClose = () => {
    this.setState({ deleteDialogOpen: false });
  };

  private _buildUrl(): string {
    const urls = this.props.configs.map(c => c.url).sort();
    return urls.length === 1 ? urls[0] : urls.map((c, i) => `Series${i + 1}:` + c).join(',');
  }

  private async _checkTensorboardApp(): Promise<void> {
    this.setState({ busy: true }, async () => {
      const { podAddress, tfVersion } = await Apis.getTensorboardApp(this._buildUrl());
      if (podAddress) {
        this.setState({ busy: false, podAddress, tensorflowVersion: tfVersion });
      } else {
        // No existing pod
        this.setState({ busy: false });
      }
    });
  }

  private _startTensorboard = async () => {
    this.setState({ busy: true }, async () => {
      await Apis.startTensorboardApp(
        encodeURIComponent(this._buildUrl()),
        encodeURIComponent(this.state.tensorflowVersion),
      );
      this.setState({ busy: false }, () => {
        this._checkTensorboardApp();
      });
    });
  };

  private _deleteTensorboard = async () => {
    // delete the already opened Tensorboard, clear the podAddress recorded in frontend,
    // and return to the select & start tensorboard page
    this.setState({ busy: true }, async () => {
      await Apis.deleteTensorboardApp(encodeURIComponent(this._buildUrl()));
      this.setState({
        busy: false,
        deleteDialogOpen: false,
        podAddress: '',
        tensorflowVersion: DEFAULT_TENSORBOARD_VERSION,
      });
    });
  };
}

export default TensorboardViewer;
