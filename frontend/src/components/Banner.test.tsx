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
import { vi } from 'vitest';
import Banner from './Banner';

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
    render(<Banner message={'Some message'} mode={'info'} refresh={() => undefined} />);
    expect(screen.queryByRole('button', { name: 'Refresh' })).toBeNull();
  });

  it('shows troubleshooting link instructed by prop', () => {
    render(<Banner message='Some message' mode='error' showTroubleshootingGuideLink={true} />);
    expect(screen.getByRole('link', { name: 'Troubleshooting guide' })).toHaveAttribute(
      'href',
      'https://www.kubeflow.org/docs/pipelines/troubleshooting',
    );
  });

  it('does not show troubleshooting link if warning', () => {
    render(<Banner message='Some message' mode='warning' showTroubleshootingGuideLink={true} />);
    expect(screen.queryByRole('link', { name: 'Troubleshooting guide' })).toBeNull();
  });

  it('opens details dialog when button is clicked', () => {
    render(<Banner message='hello' additionalInfo='world' />);
    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('closes details dialog when cancel button is clicked', async () => {
    render(<Banner message='hello' additionalInfo='world' />);
    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    const dismissButton = screen.getByRole('button', { name: 'Dismiss' });
    fireEvent.click(dismissButton);
    await waitFor(
      () => {
        expect(screen.queryByRole('dialog')).toBeNull();
      },
      { timeout: 10000 },
    );
  }, 15000);

  it('calls refresh callback', () => {
    const spy = vi.fn();
    render(<Banner message='hello' refresh={spy} />);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(spy).toHaveBeenCalled();
  });
});
