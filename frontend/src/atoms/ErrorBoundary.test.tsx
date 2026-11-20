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

import { fireEvent, render, screen } from '@testing-library/react';
import React, { useState } from 'react';
import { testBestPractices } from 'src/TestUtils';
import { ErrorBoundary } from './ErrorBoundary';

function ThrowingChild(): JSX.Element {
  throw new Error('test render crash');
}

testBestPractices();
describe('ErrorBoundary', () => {
  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <div>child content</div>
      </ErrorBoundary>,
    );
    expect(screen.getByText('child content')).toBeInTheDocument();
  });

  it('renders a Banner with error details when a child throws', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <ThrowingChild />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Something went wrong.')).toBeInTheDocument();
    expect(screen.getByText('Details')).toBeInTheDocument();

    consoleSpy.mockRestore();
  });

  it('preserves healthy child state when the navigation reset key changes', () => {
    function StatefulChild() {
      const [selection, setSelection] = useState('initial');
      return <button onClick={() => setSelection('selected')}>{selection}</button>;
    }

    const view = render(
      <ErrorBoundary resetKey='page-a'>
        <StatefulChild />
      </ErrorBoundary>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'initial' }));

    view.rerender(
      <ErrorBoundary resetKey='page-b'>
        <StatefulChild />
      </ErrorBoundary>,
    );

    expect(screen.getByRole('button', { name: 'selected' })).toBeInTheDocument();
  });

  it('retains a captured error until the navigation reset key changes', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const view = render(
      <ErrorBoundary resetKey='page-a'>
        <ThrowingChild />
      </ErrorBoundary>,
    );

    view.rerender(
      <ErrorBoundary resetKey='page-a'>
        <div>recovered content</div>
      </ErrorBoundary>,
    );

    expect(screen.getByText('Something went wrong.')).toBeInTheDocument();
    expect(screen.queryByText('recovered content')).not.toBeInTheDocument();
    consoleSpy.mockRestore();
  });

  it('recovers when the navigation reset key changes', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    function Harness() {
      const [locationKey, setLocationKey] = useState('page-a');
      return (
        <>
          <ErrorBoundary resetKey={locationKey}>
            {locationKey === 'page-a' ? <ThrowingChild /> : <div>recovered content</div>}
          </ErrorBoundary>
          <button onClick={() => setLocationKey('page-b')}>navigate</button>
        </>
      );
    }

    render(<Harness />);
    expect(screen.getByText('Something went wrong.')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'navigate' }));
    expect(screen.queryByText('Something went wrong.')).not.toBeInTheDocument();
    expect(screen.getByText('recovered content')).toBeInTheDocument();

    consoleSpy.mockRestore();
  });
});
