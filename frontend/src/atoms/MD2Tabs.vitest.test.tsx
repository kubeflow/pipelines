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
import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { logger } from '../lib/Utils';
import MD2Tabs from './MD2Tabs';

describe('MD2Tabs', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });
  it('renders with the right styles by default', () => {
    const { asFragment } = render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not try to call the onSwitch handler if it is not defined', () => {
    render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    fireEvent.click(screen.getByRole('button', { name: 'tab2' }));
  });

  it('calls the onSwitch function if an unselected button is clicked', () => {
    const switchHandler = vi.fn();
    render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} onSwitch={switchHandler} />);
    fireEvent.click(screen.getByRole('button', { name: 'tab2' }));
    expect(switchHandler).toHaveBeenCalledWith(1);
  });

  it('does not call the onSwitch function if the already selected button is clicked', () => {
    const switchHandler = vi.fn();
    render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={1} onSwitch={switchHandler} />);
    fireEvent.click(screen.getByRole('button', { name: 'tab2' }));
    expect(switchHandler).not.toHaveBeenCalled();
  });

  it('gracefully handles an out of bound selectedTab value', () => {
    const errorSpy = vi.spyOn(logger, 'error').mockImplementation(() => {});
    const { asFragment } = render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={100} />);
    expect(errorSpy).toHaveBeenCalledWith('Out of bound index passed for selected tab');
    expect(asFragment()).toMatchSnapshot();
  });

  it('recalculates indicator styles when props are updated', () => {
    vi.useFakeTimers();
    const updateSpy = vi.spyOn(MD2Tabs.prototype as any, '_updateIndicator');
    const { rerender } = render(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    vi.runOnlyPendingTimers();
    rerender(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={1} />);
    vi.runOnlyPendingTimers();
    expect(updateSpy.mock.calls.length).toBeGreaterThanOrEqual(2);
  });
});
