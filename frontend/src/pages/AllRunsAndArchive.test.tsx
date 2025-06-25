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
import * as React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { Apis } from 'src/lib/Apis';
import TestUtils from 'src/TestUtils';

// Mock useNavigate and useLocation hooks
const mockNavigate = jest.fn();
const mockLocation = { pathname: '', search: '', hash: '', state: null, key: 'default' };
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useLocation: () => mockLocation,
}));

import AllRunsAndArchive, {
  AllRunsAndArchiveProps,
  AllRunsAndArchiveTab,
} from './AllRunsAndArchive';
import { PageProps } from './Page';

function generateProps(): AllRunsAndArchiveProps {
  return {
    ...generatePageProps(),
    view: AllRunsAndArchiveTab.RUNS,
  };
}

function generatePageProps(): PageProps {
  return {
    navigate: jest.fn(),
    location: { pathname: '', search: '', hash: '', state: null, key: 'default' },
    match: { params: {}, isExact: true, path: '', url: '' },
    toolbarProps: {} as any,
    updateBanner: jest.fn(),
    updateDialog: jest.fn(),
    updateSnackbar: jest.fn(),
    updateToolbar: jest.fn(),
  };
}

describe.skip('RunsAndArchive', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockNavigate.mockClear();
    // Mock the run list API calls to prevent actual network requests
    jest
      .spyOn(Apis.runServiceApi, 'listRuns')
      .mockImplementation(() => Promise.resolve({ runs: [], total_size: 0 }));
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('renders runs page', async () => {
    const { asFragment } = render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <AllRunsAndArchive {...generateProps()} />
      </MemoryRouter>,
    );
    await TestUtils.flushPromises();
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders archive page', async () => {
    const props = generateProps();
    props.view = AllRunsAndArchiveTab.ARCHIVE;
    const { asFragment } = render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <AllRunsAndArchive {...props} />
      </MemoryRouter>,
    );
    await TestUtils.flushPromises();
    expect(asFragment()).toMatchSnapshot();
  });

  it('switches to clicked page by navigating', async () => {
    const { rerender } = render(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <AllRunsAndArchive {...generateProps()} />
      </MemoryRouter>,
    );

    await TestUtils.flushPromises();

    // Start in RUNS (Active) view, click Archive tab
    const archiveTab = screen.getByRole('button', { name: /Archived/i });
    fireEvent.click(archiveTab);

    // Verify navigate was called with the archive route
    expect(mockNavigate).toHaveBeenCalledWith('/archive/runs');

    // Now simulate navigation by updating the view prop to ARCHIVE
    const updatedProps = { ...generateProps(), view: AllRunsAndArchiveTab.ARCHIVE };
    rerender(
      <MemoryRouter initialEntries={['/does-not-matter']}>
        <AllRunsAndArchive {...updatedProps} />
      </MemoryRouter>,
    );

    await TestUtils.flushPromises();

    // Click Active tab
    const runsTab = screen.getByRole('button', { name: /Active/i });
    fireEvent.click(runsTab);

    // Verify navigate was called with the runs route
    expect(mockNavigate).toHaveBeenCalledWith('/runs');
  });
});
