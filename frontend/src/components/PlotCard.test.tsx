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

import { fireEvent, render as renderWithoutTheme, screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { ThemeProvider } from './modernization/ThemeProvider';
import userEvent from '@testing-library/user-event';
import PlotCard from './PlotCard';
import { ViewerConfig, PlotType } from './viewers/Viewer';

function render(element: ReactElement) {
  return renderWithoutTheme(element, {
    wrapper: ({ children }) => <ThemeProvider defaultTheme='light'>{children}</ThemeProvider>,
  });
}

beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal('matchMedia', (media: string) => ({
    matches: false,
    media,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe('PlotCard', () => {
  const config: ViewerConfig = {
    type: PlotType.CONFUSION_MATRIX,
    data: [[1]],
    axes: ['x', 'y'],
    labels: ['label'],
  } as any;

  it('renders nothing when there are no configs', () => {
    const { container } = render(<PlotCard title='' configs={[]} maxDimension={100} />);
    expect(container.querySelector('.kfp-plot-card')).toBeNull();
  });

  it('renders a confusion matrix plot card', () => {
    render(<PlotCard title='test title' configs={[config]} maxDimension={100} />);
    expect(screen.getByRole('heading', { name: 'test title' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Expand test title' })).toBeInTheDocument();
  });

  it('opens the fullscreen dialog', () => {
    render(<PlotCard title='' configs={[config]} maxDimension={100} />);
    fireEvent.click(screen.getByTestId('pop-out-button'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Confusion matrix')).toBeInTheDocument();
  });

  it('closes the fullscreen dialog with the close button', async () => {
    render(<PlotCard title='' configs={[config]} maxDimension={100} />);
    fireEvent.click(screen.getByTestId('pop-out-button'));
    fireEvent.click(screen.getByTestId('fullscreen-close-button'));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the fullscreen dialog with Escape and restores focus to its trigger', async () => {
    const user = userEvent.setup();
    render(<PlotCard title='test title' configs={[config]} maxDimension={100} />);
    const trigger = screen.getByRole('button', { name: 'Expand test title' });
    await user.click(trigger);
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    await user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    await waitFor(() => expect(trigger).toHaveFocus());
  });
});
