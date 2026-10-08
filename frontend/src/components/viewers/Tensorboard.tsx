/*
 * Copyright 2018 The Kubeflow Authors
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

import type * as React from 'react';
import BusyButton from '../../atoms/BusyButton';
import Viewer, { ViewerConfig } from './Viewer';
import { Apis } from '../../lib/Apis';
import { Button, buttonVariants } from '../ui/button';
import { ModalDialog } from '../ui/dialog';
import { Alert } from '../ui/alert';
import './RichViewers.css';

export interface TensorboardViewerConfig extends ViewerConfig {
  url: string;
  namespace: string;
  podTemplateSpec?: any; // JSON object of pod template spec
  image?: string;
}

interface TensorboardViewerProps {
  configs: TensorboardViewerConfig[];
  // Interval in ms. If not specified, default to 5000.
  intervalOfCheckingTensorboardPodStatus?: number;
}

interface TensorboardViewerState {
  busy: boolean;
  deleteDialogOpen: boolean;
  proxyPath: string;
  tfImage: string;
  // When proxyPath is not null, we need to further tell whether the TensorBoard pod is accessible or not
  tensorboardReady: boolean;
  errorMessage?: string;
}

// TODO: bump default version when https://github.com/kubeflow/pipelines/issues/5521
// is resolved.
const DEFAULT_TF_IMAGE = 'tensorflow/tensorflow:2.2.2';

class TensorboardViewer extends Viewer<TensorboardViewerProps, TensorboardViewerState> {
  timerID: NodeJS.Timeout;
  private _isMounted = true;

  constructor(props: TensorboardViewerProps) {
    super(props);

    this.state = {
      busy: false,
      deleteDialogOpen: false,
      proxyPath: '',
      tfImage: this._image() || DEFAULT_TF_IMAGE,
      tensorboardReady: false,
      errorMessage: undefined,
    };
  }

  public getDisplayName(): string {
    return 'Tensorboard';
  }

  public isAggregatable(): boolean {
    return true;
  }

  public componentDidMount(): void {
    this._isMounted = true;
    this._checkTensorboardApp();
    this.timerID = setInterval(
      () => this._checkTensorboardPodStatus(),
      this.props.intervalOfCheckingTensorboardPodStatus || 5000,
    );
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
    clearInterval(this.timerID);
  }

  public handleImageSelect = (e: React.ChangeEvent<HTMLSelectElement>): void => {
    if (typeof e.target.value !== 'string') {
      throw new Error('Invalid event value type, expected string');
    }
    this.setStateSafe({ tfImage: e.target.value });
  };

  public render(): React.JSX.Element {
    const { busy, proxyPath, tfImage, tensorboardReady, errorMessage } = this.state;
    return (
      <div className='kfp-tensorboard-viewer'>
        {errorMessage && !this.state.deleteDialogOpen && (
          <Alert variant='error'>{errorMessage}</Alert>
        )}
        {proxyPath ? (
          <>
            <p>{`Tensorboard ${tfImage} is running for this output.`}</p>
            {!tensorboardReady && (
              <Alert variant='warning'>
                Tensorboard is starting, and you may need to wait for a few minutes.
              </Alert>
            )}
            <div className='kfp-tensorboard-controls'>
              <a
                href={proxyPath}
                target='_blank'
                rel='noopener noreferrer'
                className={buttonVariants()}
                aria-disabled={busy || undefined}
                tabIndex={busy ? -1 : undefined}
                onClick={(event) => {
                  if (busy) event.preventDefault();
                }}
              >
                Open Tensorboard
              </a>
              <Button
                variant='secondary'
                disabled={busy}
                id='delete'
                title='stop tensorboard and delete its instance'
                onClick={this._handleDeleteOpen}
              >
                Stop Tensorboard
              </Button>
            </div>
            <ModalDialog
              open={this.state.deleteDialogOpen}
              onClose={this._handleDeleteClose}
              title='Stop Tensorboard?'
              actions={
                <>
                  <Button variant='secondary' id='cancel' onClick={this._handleDeleteClose}>
                    Cancel
                  </Button>
                  <BusyButton
                    onClick={this._deleteTensorboard}
                    busy={busy}
                    variant='destructive'
                    title='Stop'
                  />
                </>
              }
            >
              {errorMessage && <Alert variant='error'>{errorMessage}</Alert>}
              You can stop the current running tensorboard. The tensorboard viewer will also be
              deleted from your workloads.
            </ModalDialog>
          </>
        ) : (
          <>
            <label className='kfp-tensorboard-image' htmlFor='viewer-tb-image-select'>
              <span>TF Image</span>
              <select
                id='viewer-tb-image-select'
                className='kfp-viewer-select'
                value={tfImage}
                disabled={busy}
                onChange={this.handleImageSelect}
              >
                {this._image() && <option value={this._image()}>{this._image()}</option>}
                <optgroup label='TensorFlow 1.x'>
                  {[
                    '1.7.1',
                    '1.8.0',
                    '1.9.0',
                    '1.10.1',
                    '1.11.0',
                    '1.12.3',
                    '1.13.2',
                    '1.14.0',
                    '1.15.5',
                  ].map((version) => (
                    <option key={version} value={`tensorflow/tensorflow:${version}`}>
                      TensorFlow {version}
                    </option>
                  ))}
                </optgroup>
                <optgroup label='TensorFlow 2.x'>
                  {['2.0.4', '2.1.2', '2.2.2'].map((version) => (
                    <option key={version} value={`tensorflow/tensorflow:${version}`}>
                      TensorFlow {version}
                    </option>
                  ))}
                </optgroup>
              </select>
            </label>
            <div>
              <BusyButton
                disabled={!tfImage}
                onClick={this._startTensorboard}
                busy={busy}
                title={`Start ${this.props.configs.length > 1 ? 'Combined ' : ''}Tensorboard`}
              />
            </div>
          </>
        )}
      </div>
    );
  }

  private _handleDeleteOpen = () => {
    this.setStateSafe({ deleteDialogOpen: true });
  };

  private _handleDeleteClose = () => {
    this.setStateSafe({ deleteDialogOpen: false });
  };

  private _getNamespace(): string {
    // TODO: We should probably check if all configs have the same namespace.
    return this.props.configs[0]?.namespace || '';
  }

  private _buildUrl(): string {
    const urls = this.props.configs.map((c) => c.url).sort();
    return urls.length === 1 ? urls[0] : urls.map((c, i) => `Series${i + 1}:` + c).join(',');
  }

  private _podTemplateSpec(): any | undefined {
    const podTemplateSpec = this.props.configs[0]?.podTemplateSpec;
    // TODO: how to handle multiple config with different pod template specs?
    return podTemplateSpec || undefined;
  }

  private _image(): string | undefined {
    return this.props.configs[0]?.image || undefined;
  }

  private async _checkTensorboardPodStatus(): Promise<void> {
    // If the proxied TensorBoard is not ready yet, poll the scoped proxy path again.
    if (this.state.proxyPath && !this.state.tensorboardReady) {
      Apis.isTensorboardPodReady(this.state.proxyPath).then((ready) => {
        this.setStateSafe(({ tensorboardReady }) => ({
          tensorboardReady: tensorboardReady || ready,
        }));
      });
    }
  }

  private async _checkTensorboardApp(): Promise<void> {
    this.setStateSafe({ busy: true }, async () => {
      try {
        // TODO: parse tfImage here
        const { proxyPath, image } = await Apis.getTensorboardApp(
          this._buildUrl(),
          this._getNamespace(),
        );
        if (proxyPath) {
          this.setStateSafe({ busy: false, proxyPath, tfImage: image });
        } else {
          // No existing pod
          this.setStateSafe({ busy: false });
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        this.setStateSafe({ busy: false, errorMessage });
      }
    });
  }

  private _startTensorboard = async () => {
    this.setStateSafe({ busy: true, errorMessage: undefined }, async () => {
      try {
        const proxyPath = await Apis.startTensorboardApp({
          logdir: this._buildUrl(),
          namespace: this._getNamespace(),
          image: this.state.tfImage,
          podTemplateSpec: this._podTemplateSpec(),
        });
        this.setStateSafe({ busy: false, proxyPath, tensorboardReady: false }, () => {
          if (proxyPath) {
            this._checkTensorboardPodStatus();
          } else {
            this._checkTensorboardApp();
          }
        });
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        this.setStateSafe({ busy: false, errorMessage });
      }
    });
  };

  private _deleteTensorboard = async () => {
    // delete the already opened Tensorboard, clear the proxy path recorded in frontend,
    // and return to the select & start tensorboard page
    this.setStateSafe({ busy: true, errorMessage: undefined }, async () => {
      try {
        await Apis.deleteTensorboardApp(this._buildUrl(), this._getNamespace());
        this.setStateSafe({
          busy: false,
          deleteDialogOpen: false,
          proxyPath: '',
          tensorboardReady: false,
        });
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        this.setStateSafe({ busy: false, errorMessage });
      }
    });
  };

  private setStateSafe(
    newState:
      | Partial<TensorboardViewerState>
      | ((prevState: TensorboardViewerState) => Partial<TensorboardViewerState>),
    cb?: () => void,
  ): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }
}

export default TensorboardViewer;
