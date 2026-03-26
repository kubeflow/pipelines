/*
 * Copyright 2026 The Kubeflow Authors
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
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import SidePanel from './SidePanel';

// Mock @mui/material Slide to avoid react-transition-group issues in jsdom
vi.mock('@mui/material', async () => {
  const actual = await vi.importActual('@mui/material');
  return {
    ...actual,
    Slide: ({ children, in: isIn }: { children: React.ReactElement; in: boolean }) =>
      isIn ? children : null,
  };
});

// Mock re-resizable to avoid rendering issues in jsdom
vi.mock('re-resizable', () => ({
  __esModule: true,
  Resizable: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <div className={className}>{children}</div>
  ),
}));

describe('SidePanel (interaction)', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    title: 'Test Panel',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the title when open', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByText('Test Panel')).toBeInTheDocument();
  });

  it('renders children when open', () => {
    render(
      <SidePanel {...defaultProps}>
        <div>Child Content</div>
      </SidePanel>,
    );
    expect(screen.getByText('Child Content')).toBeInTheDocument();
  });

  it('renders close button when open', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByRole('button', { name: 'close' })).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);
    fireEvent.click(screen.getByRole('button', { name: 'close' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when Escape key is pressed', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not call onClose on Escape when panel is closed', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} isOpen={false} onClose={onClose} />);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });

  it('shows loading spinner when isBusy is true', () => {
    render(<SidePanel {...defaultProps} isBusy={true} />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('does not show loading spinner when isBusy is false', () => {
    render(<SidePanel {...defaultProps} isBusy={false} />);
    expect(screen.queryByRole('progressbar')).toBeNull();
  });

  it('removes keydown listener on unmount', () => {
    const onClose = vi.fn();
    const { unmount } = render(<SidePanel {...defaultProps} onClose={onClose} />);
    unmount();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });
});
