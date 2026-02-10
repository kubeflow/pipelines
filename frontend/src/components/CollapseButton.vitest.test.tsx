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
import CollapseButton from './CollapseButton';

describe('CollapseButton', () => {
  const compareComponent = {
    collapseSectionsUpdate: vi.fn(),
    state: {
      collapseSections: {},
    },
  } as any;

  afterEach(() => {
    compareComponent.state.collapseSections = {};
    compareComponent.collapseSectionsUpdate.mockClear();
  });

  it('initial render', () => {
    const { asFragment } = render(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        collapseSectionsUpdate={compareComponent.collapseSectionsUpdate}
        sectionName='testSection'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders the button collapsed if in collapsedSections', () => {
    compareComponent.state.collapseSections.testSection = true;
    const { asFragment } = render(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        collapseSectionsUpdate={compareComponent.collapseSectionsUpdate}
        sectionName='testSection'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('collapses given section when clicked', () => {
    render(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        collapseSectionsUpdate={compareComponent.collapseSectionsUpdate}
        sectionName='testSection'
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Expand/Collapse this section' }));
    expect(compareComponent.collapseSectionsUpdate).toHaveBeenCalledWith({ testSection: true });
  });

  it('expands given section when clicked if it is collapsed', () => {
    compareComponent.state.collapseSections.testSection = true;
    render(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        collapseSectionsUpdate={compareComponent.collapseSectionsUpdate}
        sectionName='testSection'
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Expand/Collapse this section' }));
    expect(compareComponent.collapseSectionsUpdate).toHaveBeenCalledWith({ testSection: false });
  });
});
