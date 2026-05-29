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

  it('recovers when the key prop changes (simulates navigation)', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    function Harness() {
      const [locationKey, setLocationKey] = useState('page-a');
      return (
        <>
          <ErrorBoundary key={locationKey}>
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
