/*
 * Copyright 2021 The Kubeflow Authors
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
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { LineageActionBar } from './LineageActionBar';
import { ArtifactHelpers } from './MlmdUtils';
import { buildTestModel, testModel } from './TestUtils';

describe('LineageActionBar', () => {
  const user = userEvent.setup();
  const setLineageViewTarget = vi.fn();
  const label = ArtifactHelpers.getName(testModel);

  afterEach(() => {
    setLineageViewTarget.mockReset();
  });

  it('renders the initial breadcrumb as active and shows the Reset button', () => {
    const { asFragment } = render(
      <LineageActionBar initialTarget={testModel} setLineageViewTarget={vi.fn()} />,
    );
    const breadcrumb = screen.getByRole('button', { name: label });
    expect(breadcrumb).toBeDisabled();
    expect(screen.getByRole('button', { name: /Reset/i })).toBeInTheDocument();
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not call setLineageViewTarget when the active breadcrumb is clicked', async () => {
    render(
      <LineageActionBar initialTarget={testModel} setLineageViewTarget={setLineageViewTarget} />,
    );
    const breadcrumb = screen.getByRole('button', { name: label });
    expect(breadcrumb).toBeDisabled();
    await user.click(breadcrumb);
    expect(setLineageViewTarget).not.toHaveBeenCalled();
  });

  it('calls setLineageViewTarget when an inactive breadcrumb is clicked', async () => {
    const ref = React.createRef<LineageActionBar>();
    const { asFragment } = render(
      <LineageActionBar
        ref={ref}
        initialTarget={testModel}
        setLineageViewTarget={setLineageViewTarget}
      />,
    );

    // pushHistory is a public API called by parent components.
    // act() is required because it triggers setState outside of Testing Library.
    act(() => {
      ref.current!.pushHistory(buildTestModel());
    });

    const breadcrumbs = screen.getAllByRole('button', { name: label });
    expect(breadcrumbs).toHaveLength(2);
    expect(breadcrumbs[0]).not.toBeDisabled();
    expect(breadcrumbs[1]).toBeDisabled();

    await user.click(breadcrumbs[0]);
    expect(setLineageViewTarget).toHaveBeenCalledTimes(1);
    expect(asFragment()).toMatchSnapshot();
  });

  it('adds a breadcrumb when pushHistory() is called', () => {
    const ref = React.createRef<LineageActionBar>();
    const { asFragment } = render(
      <LineageActionBar
        ref={ref}
        initialTarget={testModel}
        setLineageViewTarget={setLineageViewTarget}
      />,
    );

    expect(screen.getAllByRole('button', { name: label })).toHaveLength(1);

    act(() => {
      ref.current!.pushHistory(buildTestModel());
    });

    const breadcrumbs = screen.getAllByRole('button', { name: label });
    expect(breadcrumbs).toHaveLength(2);
    expect(breadcrumbs[0]).not.toBeDisabled();
    expect(breadcrumbs[1]).toBeDisabled();
    expect(asFragment()).toMatchSnapshot();
  });

  it('resets to initial breadcrumb when Reset is clicked', async () => {
    const ref = React.createRef<LineageActionBar>();
    render(
      <LineageActionBar
        ref={ref}
        initialTarget={testModel}
        setLineageViewTarget={setLineageViewTarget}
      />,
    );

    act(() => {
      ref.current!.pushHistory(buildTestModel());
    });
    act(() => {
      ref.current!.pushHistory(buildTestModel());
    });

    expect(screen.getAllByRole('button', { name: label })).toHaveLength(3);

    await user.click(screen.getByRole('button', { name: /Reset/i }));

    const breadcrumbs = screen.getAllByRole('button', { name: label });
    expect(breadcrumbs).toHaveLength(1);
    expect(breadcrumbs[0]).toBeDisabled();
  });
});
