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
import AllRunsAndArchive, {
  AllRunsAndArchiveProps,
  AllRunsAndArchiveTab,
} from './AllRunsAndArchive';

describe('RunsAndArchive', () => {
  function generateProps(): AllRunsAndArchiveProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: {} as any,
      updateBanner: () => null,
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: () => null,
      view: AllRunsAndArchiveTab.RUNS,
    };
  }

  it('renders runs page', () => {
    const { asFragment } = render(<AllRunsAndArchive {...(generateProps() as any)} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders archive page', () => {
    const props = generateProps();
    props.view = AllRunsAndArchiveTab.ARCHIVE;
    const { asFragment } = render(<AllRunsAndArchive {...(props as any)} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('switches to clicked page by pushing to history', () => {
    const spy = vi.fn();
    const props = generateProps();
    props.history.push = spy;
    const { rerender } = render(<AllRunsAndArchive {...(props as any)} />);

    fireEvent.click(screen.getByRole('button', { name: 'Archived' }));
    expect(spy).toHaveBeenCalledWith('/archive/runs');

    rerender(<AllRunsAndArchive {...(props as any)} view={AllRunsAndArchiveTab.ARCHIVE} />);
    fireEvent.click(screen.getByRole('button', { name: 'Active' }));
    expect(spy).toHaveBeenCalledWith('/runs');
  });
});
