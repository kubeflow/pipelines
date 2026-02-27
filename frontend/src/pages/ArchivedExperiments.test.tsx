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
import { ArchivedExperiments } from './ArchivedExperiments';
import TestUtils from 'src/TestUtils';
import { PageProps } from './Page';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { ButtonKeys } from 'src/lib/Buttons';

const refreshSpy = vi.fn();
let lastExperimentListProps: any = null;

vi.mock('src/components/ExperimentList', () => ({
  default: React.forwardRef((props: any, ref) => {
    lastExperimentListProps = props;
    React.useImperativeHandle(ref, () => ({ refresh: refreshSpy }));
    return <div data-testid='experiment-list' />;
  }),
}));

describe('ArchivedExperiments', () => {
  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;
  let renderResult: ReturnType<typeof render> | null = null;
  let archivedExperimentsRef: React.RefObject<ArchivedExperiments> | null = null;

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ArchivedExperiments,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function renderArchivedExperiments(propsPatch: Partial<PageProps & { namespace?: string }> = {}) {
    archivedExperimentsRef = React.createRef<ArchivedExperiments>();
    const props = { ...generateProps(), ...propsPatch } as PageProps;
    renderResult = render(<ArchivedExperiments ref={archivedExperimentsRef} {...props} />);
  }

  function getToolbarAction(key: string) {
    if (!archivedExperimentsRef?.current) {
      throw new Error(`ArchivedExperiments instance not available for ${key}`);
    }
    return archivedExperimentsRef.current.getInitialToolbarState().actions[key];
  }

  beforeEach(() => {
    updateBannerSpy = vi.fn();
    updateToolbarSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    historyPushSpy = vi.fn();
    refreshSpy.mockClear();
    lastExperimentListProps = null;
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    archivedExperimentsRef = null;
  });

  it('renders archived experiments', () => {
    renderArchivedExperiments();
    expect(lastExperimentListProps).toBeTruthy();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('removes error banner on unmount', () => {
    renderArchivedExperiments();
    renderResult!.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('refreshes the experiment list when refresh button is clicked', async () => {
    renderArchivedExperiments();
    const refreshAction = getToolbarAction(ButtonKeys.REFRESH);
    await act(async () => {
      await refreshAction.action();
    });
    expect(refreshSpy).toHaveBeenCalled();
  });

  it('shows a list of archived experiments', () => {
    renderArchivedExperiments();
    expect(lastExperimentListProps.storageState).toBe(
      V2beta1ExperimentStorageState.ARCHIVED.toString(),
    );
  });
});
