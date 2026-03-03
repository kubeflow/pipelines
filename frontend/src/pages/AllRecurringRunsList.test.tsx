/*
 * Copyright 2021 Arrikto Inc.
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
import { render } from '@testing-library/react';
import { vi } from 'vitest';
import { RoutePage } from 'src/components/Router';
import { ButtonKeys } from 'src/lib/Buttons';
import { AllRecurringRunsList } from './AllRecurringRunsList';
import { PageProps } from './Page';
import { ToolbarProps } from 'src/components/Toolbar';

let lastRecurringRunListProps: any = null;

vi.mock('./RecurringRunList', () => ({
  default: (props: any) => {
    lastRecurringRunListProps = props;
    return <div data-testid='recurring-run-list' />;
  },
}));

describe('AllRecurringRunsList', () => {
  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;
  let renderResult: ReturnType<typeof render> | null = null;
  let allRecurringRunsListRef: React.RefObject<AllRecurringRunsList> | null = null;
  let toolbarProps: ToolbarProps | null = null;

  function baseProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  function renderAllRecurringRunsList(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ) {
    allRecurringRunsListRef = React.createRef<AllRecurringRunsList>();
    const props = { ...baseProps(), ...propsPatch } as PageProps;
    const { rerender, ...result } = render(
      <AllRecurringRunsList ref={allRecurringRunsListRef} {...props} />,
    );
    if (!allRecurringRunsListRef.current) {
      throw new Error('AllRecurringRunsList instance not available');
    }
    toolbarProps = allRecurringRunsListRef.current.getInitialToolbarState();
    rerender(
      <AllRecurringRunsList ref={allRecurringRunsListRef} {...props} toolbarProps={toolbarProps} />,
    );
    updateToolbarSpy.mockClear();
    renderResult = result as ReturnType<typeof render>;
  }

  beforeEach(() => {
    updateBannerSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    lastRecurringRunListProps = null;
    toolbarProps = null;
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    allRecurringRunsListRef = null;
    toolbarProps = null;
  });

  it('renders all recurring runs', () => {
    renderAllRecurringRunsList();
    expect(lastRecurringRunListProps).toBeTruthy();
    expect(lastRecurringRunListProps.refreshCount).toBe(0);
    expect(lastRecurringRunListProps.selectedIds).toEqual([]);
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('lists all recurring runs in namespace', () => {
    renderAllRecurringRunsList({ namespace: 'test-ns' });
    expect(lastRecurringRunListProps.namespaceMask).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    renderAllRecurringRunsList();
    renderResult!.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('navigates to new run page when new run is clicked', () => {
    renderAllRecurringRunsList();
    toolbarProps!.actions[ButtonKeys.NEW_RECURRING_RUN].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + '?experimentId=&recurring=1',
    );
  });
});
