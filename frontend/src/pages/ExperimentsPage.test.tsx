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

import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import ExperimentsPage, { ExperimentsPageProps, ExperimentsPageTab } from './ExperimentsPage';

describe('ExperimentsAndArchive', () => {
  function generateProps(): ExperimentsPageProps {
    return {
      navigate: vi.fn(),
      location: '' as any,
      params: {},
      toolbarProps: {} as any,
      updateBanner: () => null,
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: () => null,
      view: ExperimentsPageTab.EXPERIMENTS,
    };
  }

  it('renders experiments page', () => {
    const { asFragment } = render(<ExperimentsPage {...(generateProps() as any)} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders archive page', () => {
    const props = generateProps();
    props.view = ExperimentsPageTab.ARCHIVE;
    const { asFragment } = render(<ExperimentsPage {...(props as any)} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('switches to clicked page by pushing to history', () => {
    const spy = vi.fn();
    const props = generateProps();
    props.navigate = spy;
    const { rerender } = render(<ExperimentsPage {...(props as any)} />);

    fireEvent.click(screen.getByRole('tab', { name: 'Archived' }));
    expect(spy).toHaveBeenCalledWith('/archive/experiments');

    rerender(<ExperimentsPage {...(props as any)} view={ExperimentsPageTab.ARCHIVE} />);
    fireEvent.click(screen.getByRole('tab', { name: 'Active' }));
    expect(spy).toHaveBeenCalledWith('/experiments');
  });
});
