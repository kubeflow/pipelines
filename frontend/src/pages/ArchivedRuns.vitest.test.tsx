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
import { ArchivedRuns } from './ArchivedRuns';
import { PageProps } from './Page';
import { ToolbarProps } from 'src/components/Toolbar';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { ButtonKeys } from 'src/lib/Buttons';
import { Apis } from 'src/lib/Apis';

const refreshSpy = vi.fn();
let lastRunListProps: any = null;

vi.mock('./RunList', () => ({
  default: React.forwardRef((props: any, ref) => {
    lastRunListProps = props;
    React.useImperativeHandle(ref, () => ({ refresh: refreshSpy }));
    return <div data-testid='run-list' />;
  }),
}));

describe('ArchivedRuns', () => {
  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;
  let renderResult: ReturnType<typeof render> | null = null;
  let archivedRunsRef: React.RefObject<ArchivedRuns> | null = null;
  let toolbarProps: ToolbarProps | null = null;
  let deleteRunSpy: ReturnType<typeof vi.spyOn>;

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

  function renderArchivedRuns(propsPatch: Partial<PageProps & { namespace?: string }> = {}) {
    archivedRunsRef = React.createRef<ArchivedRuns>();
    const props = { ...baseProps(), ...propsPatch } as PageProps;
    const { rerender, ...result } = render(<ArchivedRuns ref={archivedRunsRef} {...props} />);
    if (!archivedRunsRef.current) {
      throw new Error('ArchivedRuns instance not available');
    }
    toolbarProps = archivedRunsRef.current.getInitialToolbarState();
    rerender(<ArchivedRuns ref={archivedRunsRef} {...props} toolbarProps={toolbarProps} />);
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
    deleteRunSpy = vi.spyOn(Apis.runServiceApi, 'deleteRun');
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    archivedRunsRef = null;
    toolbarProps = null;
    deleteRunSpy.mockRestore();
  });

  it('renders archived runs', () => {
    renderArchivedRuns();
    expect(lastRunListProps).toBeTruthy();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('lists archived runs in namespace', () => {
    renderArchivedRuns({ namespace: 'test-ns' });
    expect(lastRunListProps.namespaceMask).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    renderArchivedRuns();
    renderResult!.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('enables restore and delete button when at least one run is selected', async () => {
    renderArchivedRuns();
    expect(toolbarProps!.actions[ButtonKeys.RESTORE].disabled).toBeTruthy();
    expect(toolbarProps!.actions[ButtonKeys.DELETE_RUN].disabled).toBeTruthy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1']);
    });
    expect(toolbarProps!.actions[ButtonKeys.RESTORE].disabled).toBeFalsy();
    expect(toolbarProps!.actions[ButtonKeys.DELETE_RUN].disabled).toBeFalsy();
    await act(async () => {
      await lastRunListProps.onSelectionChange(['run1', 'run2']);
    });
    expect(toolbarProps!.actions[ButtonKeys.RESTORE].disabled).toBeFalsy();
    expect(toolbarProps!.actions[ButtonKeys.DELETE_RUN].disabled).toBeFalsy();
    await act(async () => {
      await lastRunListProps.onSelectionChange([]);
    });
    expect(toolbarProps!.actions[ButtonKeys.RESTORE].disabled).toBeTruthy();
    expect(toolbarProps!.actions[ButtonKeys.DELETE_RUN].disabled).toBeTruthy();
  });

  it('refreshes the run list when refresh button is clicked', async () => {
    renderArchivedRuns();
    await act(async () => {
      await toolbarProps!.actions[ButtonKeys.REFRESH].action();
    });
    expect(refreshSpy).toHaveBeenCalled();
  });

  it('shows a list of available runs', () => {
    renderArchivedRuns();
    expect(lastRunListProps.storageState).toBe(V2beta1RunStorageState.ARCHIVED.toString());
  });

  it('cancels deletion when Cancel is clicked', async () => {
    renderArchivedRuns();

    await act(async () => {
      await toolbarProps!.actions[ButtonKeys.DELETE_RUN].action();
    });

    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Do you want to delete the selected runs? This action cannot be undone.',
      }),
    );

    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((button: any) => button.text === 'Cancel');
    await act(async () => {
      await cancelBtn.onClick();
    });
    expect(deleteRunSpy).not.toHaveBeenCalled();
  });

  it('deletes selected ids when Confirm is clicked', async () => {
    renderArchivedRuns();

    await act(async () => {
      await lastRunListProps.onSelectionChange(['id1', 'id2', 'id3']);
    });

    deleteRunSpy.mockImplementationOnce(() => {
      throw { text: () => Promise.resolve('woops') };
    });
    deleteRunSpy.mockResolvedValue(undefined as any);

    await act(async () => {
      await toolbarProps!.actions[ButtonKeys.DELETE_RUN].action();
    });

    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Do you want to delete the selected runs? This action cannot be undone.',
      }),
    );

    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((button: any) => button.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });

    expect(deleteRunSpy).toHaveBeenCalledTimes(3);
    expect(deleteRunSpy).toHaveBeenCalledWith('id1');
    expect(deleteRunSpy).toHaveBeenCalledWith('id2');
    expect(deleteRunSpy).toHaveBeenCalledWith('id3');
    expect(archivedRunsRef!.current!.state.selectedIds).toEqual(['id1']);
  });
});
