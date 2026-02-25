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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import PlotCard from './PlotCard';
import { ViewerConfig, PlotType } from './viewers/Viewer';

describe('PlotCard', () => {
  const config: ViewerConfig = {
    type: PlotType.CONFUSION_MATRIX,
    data: [[1]],
    axes: ['x', 'y'],
    labels: ['label'],
  } as any;

  it('renders nothing when there are no configs', () => {
    const { container } = render(<PlotCard title='' configs={[]} maxDimension={100} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders a confusion matrix plot card', () => {
    const { asFragment } = render(
      <PlotCard title='test title' configs={[config]} maxDimension={100} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('opens the fullscreen dialog', () => {
    const { container } = render(<PlotCard title='' configs={[config]} maxDimension={100} />);
    const popOutButton = container.querySelector('.popOutButton');
    if (!popOutButton) {
      throw new Error('Pop out button not found');
    }
    fireEvent.click(popOutButton);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Confusion matrix')).toBeInTheDocument();
  });

  it('closes the fullscreen dialog with the close button', async () => {
    const { container } = render(<PlotCard title='' configs={[config]} maxDimension={100} />);
    const popOutButton = container.querySelector('.popOutButton');
    if (!popOutButton) {
      throw new Error('Pop out button not found');
    }
    fireEvent.click(popOutButton);
    const closeButton = document.querySelector('.fullscreenCloseButton');
    if (!closeButton) {
      throw new Error('Close button not found');
    }
    fireEvent.click(closeButton);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the fullscreen dialog when the backdrop is clicked', async () => {
    const { container } = render(<PlotCard title='' configs={[config]} maxDimension={100} />);
    const popOutButton = container.querySelector('.popOutButton');
    if (!popOutButton) {
      throw new Error('Pop out button not found');
    }
    fireEvent.click(popOutButton);
    const backdrop = document.querySelector('[class*="MuiBackdrop-root"]');
    if (!backdrop) {
      throw new Error('Backdrop not found');
    }
    fireEvent.click(backdrop);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });
});
