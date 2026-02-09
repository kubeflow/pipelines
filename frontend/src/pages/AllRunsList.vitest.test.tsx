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
import { act, render } from '@testing-library/react';
import { vi } from 'vitest';
import { RoutePage } from 'src/components/Router';
import { ButtonKeys } from 'src/lib/Buttons';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { AllRunsList } from './AllRunsList';
import { PageProps } from './Page';
import { ToolbarProps } from 'src/components/Toolbar';

const refreshSpy = vi.fn();
let lastRunListProps: any = null;

vi.mock('./RunList', () => ({
  default: React.forwardRef((props: any, ref) => {
    lastRunListProps = props;
    React.useImperativeHandle(ref, () => ({ refresh: refreshSpy }));
    return <div data-testid='run-list' />;
  }),
}));

describe('AllRunsList', () => {
  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;
  let renderResult: ReturnType<typeof render> | null = null;
  let allRunsListRef: React.RefObject<AllRunsList> | null = null;
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

  function renderAllRunsList(propsPatch: Partial<PageProps & { namespace?: string }> = {}) {
    allRunsListRef = React.createRef<AllRunsList>();
    const props = { ...baseProps(), ...propsPatch } as PageProps;
    const { rerender, ...result } = render(<AllRunsList ref={allRunsListRef} {...props} />);
    if (!allRunsListRef.current) {
      throw new Error('AllRunsList instance not available');
    }
    toolbarProps = allRunsListRef.current.getInitialToolbarState();
    rerender(<AllRunsList ref={allRunsListRef} {...props} toolbarProps={toolbarProps} />);
    updateToolbarSpy.mockClear();
    renderResult = result as ReturnType<typeof render>;
  }

  beforeEach(() => {
    updateBannerSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    refreshSpy.mockClear();
    lastRunListProps = null;
    toolbarProps = null;
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    allRunsListRef = null;
    toolbarProps = null;
  });

  it('renders all runs', () => {
    renderAllRunsList();
    expect(lastRunListProps).toBeTruthy();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('lists all runs in namespace', () => {
    renderAllRunsList({ namespace: 'test-ns' });
    expect(lastRunListProps.namespaceMask).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    renderAllRunsList();
    renderResult!.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('only enables clone button when exactly one run is selected', async () => {
    renderAllRunsList();
    expect(toolbarProps!.actions[ButtonKeys.CLONE_RUN].disabled).toBeTruthy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1']);
    });
    expect(toolbarProps!.actions[ButtonKeys.CLONE_RUN].disabled).toBeFalsy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1', 'run2']);
    });
    expect(toolbarProps!.actions[ButtonKeys.CLONE_RUN].disabled).toBeTruthy();
  });

  it('enables archive button when at least one run is selected', async () => {
    renderAllRunsList();
    expect(toolbarProps!.actions[ButtonKeys.ARCHIVE].disabled).toBeTruthy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1']);
    });
    expect(toolbarProps!.actions[ButtonKeys.ARCHIVE].disabled).toBeFalsy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1', 'run2']);
    });
    expect(toolbarProps!.actions[ButtonKeys.ARCHIVE].disabled).toBeFalsy();
  });

  it('refreshes the run list when refresh button is clicked', async () => {
    renderAllRunsList();
    await act(async () => {
      await toolbarProps!.actions[ButtonKeys.REFRESH].action();
    });
    expect(refreshSpy).toHaveBeenCalled();
  });

  it('navigates to new run page when clone is clicked', async () => {
    renderAllRunsList();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1']);
    });
    toolbarProps!.actions[ButtonKeys.CLONE_RUN].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_RUN + '?cloneFromRun=run1');
  });

  it('navigates to compare page when compare button is clicked', async () => {
    renderAllRunsList();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1', 'run2', 'run3']);
    });
    toolbarProps!.actions[ButtonKeys.COMPARE].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.COMPARE + '?runlist=run1,run2,run3');
  });

  it('shows thrown error in error banner', async () => {
    renderAllRunsList();
    const errorMessage = 'test error message';
    const error = new Error('error object message');
    await act(async () => {
      await lastRunListProps.onError(errorMessage, error);
    });
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        message: expect.stringContaining(errorMessage),
      }),
    );
  });

  it('shows a list of available runs', () => {
    renderAllRunsList();
    expect(lastRunListProps.storageState).toBe(V2beta1RunStorageState.AVAILABLE.toString());
  });
});
