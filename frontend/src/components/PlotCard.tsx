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

import * as React from 'react';
import { Maximize2, X } from 'lucide-react';
import ViewerContainer, { componentMap } from './viewers/ViewerContainer';
import { ViewerConfig } from './viewers/Viewer';
import { Button } from './ui/button';
import { ModalDialog } from './ui/dialog';
import './viewers/ComparisonViewers.css';

export interface PlotCardProps {
  title: string;
  configs: ViewerConfig[];
  maxDimension?: number;
}

interface PlotCardState {
  fullscreenDialogOpen: boolean;
}

class PlotCard extends React.Component<PlotCardProps, PlotCardState> {
  public state = { fullscreenDialogOpen: false };

  public shouldComponentUpdate(nextProps: PlotCardProps, nextState: PlotCardState): boolean {
    return (
      JSON.stringify(nextProps) !== JSON.stringify(this.props) ||
      nextState.fullscreenDialogOpen !== this.state.fullscreenDialogOpen
    );
  }

  public render(): React.JSX.Element | null {
    const { title, configs, maxDimension } = this.props;
    if (!configs?.length) return null;
    return (
      <section className='kfp-plot-card plotCard'>
        <header className='kfp-plot-header'>
          <h3 title={title}>{title}</h3>
          <Button
            variant='ghost'
            size='icon'
            aria-label={`Expand ${title || 'visualization'}`}
            onClick={() => this.setState({ fullscreenDialogOpen: true })}
            className='popOutButton'
            data-testid='pop-out-button'
          >
            <Maximize2 size={16} aria-hidden='true' />
          </Button>
        </header>
        <div className='kfp-plot-content'>
          <ViewerContainer configs={configs} maxDimension={maxDimension} />
        </div>
        {this.state.fullscreenDialogOpen && (
          <ModalDialog
            open={true}
            size='full'
            title={
              <>
                {componentMap[configs[0].type].prototype.getDisplayName()}
                {title && <> · {title}</>}
              </>
            }
            onClose={() => this.setState({ fullscreenDialogOpen: false })}
            actions={
              <Button
                variant='secondary'
                onClick={() => this.setState({ fullscreenDialogOpen: false })}
                className='fullscreenCloseButton'
                data-testid='fullscreen-close-button'
              >
                <X size={16} aria-hidden='true' />
                Close
              </Button>
            }
          >
            <div className='kfp-plot-fullscreen'>
              <ViewerContainer configs={configs} />
            </div>
          </ModalDialog>
        )}
      </section>
    );
  }
}

export default PlotCard;
