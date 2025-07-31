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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import * as React from 'react';

import Banner, { css } from './Banner';

describe('Banner', () => {
  it('defaults to error mode', () => {
    const { asFragment } = render(<Banner message={'Some message'} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('uses error mode when instructed', () => {
    const { asFragment } = render(<Banner message={'Some message'} mode={'error'} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('uses warning mode when instructed', () => {
    const { asFragment } = render(<Banner message={'Some message'} mode={'warning'} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('uses info mode when instructed', () => {
    const { asFragment } = render(<Banner message={'Some message'} mode={'info'} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('shows "Details" button and has dialog when there is additional info', () => {
    const { asFragment } = render(<Banner message={'Some message'} additionalInfo={'More info'} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('shows "Refresh" button when passed a refresh function', () => {
    const { asFragment } = render(
      <Banner
        message={'Some message'}
        refresh={() => {
          /* do nothing */
        }}
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not show "Refresh" button if mode is "info"', () => {
    render(
      <Banner
        message={'Some message'}
        mode={'info'}
        refresh={() => {
          /* do nothing */
        }}
      />,
    );
    expect(screen.queryByText('Refresh')).not.toBeInTheDocument();
  });

  it('shows troubleshooting link instructed by prop', () => {
    const { asFragment } = render(
      <Banner message='Some message' mode='error' showTroubleshootingGuideLink={true} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not show troubleshooting link if warning', () => {
    render(
      <Banner message='Some message' mode='warning' showTroubleshootingGuideLink={true} />,
    );
    expect(screen.queryByText('Troubleshooting guide')).not.toBeInTheDocument();
  });

  it('opens details dialog when button is clicked', () => {
    render(<Banner message='hello' additionalInfo='world' />);
    
    // Click the Details button
    const detailsButton = screen.getByRole('button', { name: /details/i });
    fireEvent.click(detailsButton);
    
    // Check that dialog is open by looking for dialog content
    expect(screen.getByText('world')).toBeInTheDocument();
  });

  it('closes details dialog when cancel button is clicked', async () => {
    render(<Banner message='hello' additionalInfo='world' />);
    
    // Open the dialog
    const detailsButton = screen.getByRole('button', { name: /details/i });
    fireEvent.click(detailsButton);
    expect(screen.getByText('world')).toBeInTheDocument();
    
    // Close the dialog
    const dismissButton = screen.getByRole('button', { name: /dismiss/i });
    fireEvent.click(dismissButton);
    
    // Check that dialog is closed (wait for animation)
    await waitFor(() => {
      expect(screen.queryByText('world')).not.toBeInTheDocument();
    });
  });

  it('calls refresh callback', () => {
    const spy = jest.fn();
    render(<Banner message='hello' refresh={spy} />);
    
    const refreshButton = screen.getByRole('button', { name: /refresh/i });
    fireEvent.click(refreshButton);
    
    expect(spy).toHaveBeenCalled();
  });
});
