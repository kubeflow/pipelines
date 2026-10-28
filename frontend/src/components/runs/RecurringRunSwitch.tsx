/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useId, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { PageNotification } from '../shell/PageChrome';
import { useTheme } from '../shell/ThemeProvider';
import { Button } from '../ui/button';
import { NoticeDetails } from '../ui/notice-details';
import { V2RecurringRunStatus } from 'src/apisv2/recurringrun';
import { Apis } from 'src/lib/Apis';
import { errorToMessage } from 'src/lib/Utils';
import { ScheduleSwitch } from './ScheduleSwitch';
import './ExperimentWorkflows.css';

export function RecurringRunSwitch({
  id,
  name,
  status,
  onUpdated,
}: {
  id: string;
  name: string;
  status?: V2RecurringRunStatus;
  onUpdated: () => Promise<void>;
}) {
  const errorId = useId();
  const { themeClassName } = useTheme();
  const inFlight = useRef(false);
  const [retrying, setRetrying] = useState(false);
  const [state, setState] = useState({
    id,
    observedStatus: status,
    confirmedStatus: status,
    error: '',
    refreshFailed: false,
  });
  // A successful mutation is authoritative until the parent supplies a changed server status.
  // Reconcile during render so stale responses cannot update a different schedule's control.
  if (state.id !== id || state.observedStatus !== status) {
    setState({
      id,
      observedStatus: status,
      confirmedStatus: status,
      error: '',
      refreshFailed: false,
    });
  }
  const { confirmedStatus, error } = state;
  const knownStatus =
    confirmedStatus === V2RecurringRunStatus.ENABLED ||
    confirmedStatus === V2RecurringRunStatus.DISABLED;
  const refresh = async () => {
    try {
      await onUpdated();
    } catch (cause) {
      const message = await errorToMessage(cause);
      setState((current) =>
        current.id === id
          ? {
              ...current,
              error: `Schedule updated, but the list could not refresh: ${message}. Refresh the list to try again.`,
              refreshFailed: true,
            }
          : current,
      );
    }
  };
  const changeEnabled = async (enabled: boolean) => {
    setState((current) => ({ ...current, error: '' }));
    try {
      await (enabled
        ? Apis.recurringRunServiceApi.enableRecurringRun(id)
        : Apis.recurringRunServiceApi.disableRecurringRun(id));
    } catch (cause) {
      const message = await errorToMessage(cause);
      setState((current) =>
        current.id === id
          ? { ...current, error: `Unable to update schedule: ${message}`, refreshFailed: false }
          : current,
      );
      return;
    }
    setState((current) =>
      current.id === id
        ? {
            ...current,
            confirmedStatus: enabled ? V2RecurringRunStatus.ENABLED : V2RecurringRunStatus.DISABLED,
          }
        : current,
    );
    await refresh();
  };
  const retry = async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setRetrying(true);
    const refreshOnly = state.refreshFailed;
    setState((current) => ({ ...current, error: '' }));
    try {
      if (refreshOnly) await refresh();
      else await changeEnabled(confirmedStatus !== V2RecurringRunStatus.ENABLED);
    } finally {
      inFlight.current = false;
      setRetrying(false);
    }
  };
  return (
    <div className='kfp-schedule-control' onClick={(event) => event.stopPropagation()}>
      <ScheduleSwitch
        name={name}
        checked={confirmedStatus === V2RecurringRunStatus.ENABLED}
        disabled={!knownStatus || !id || retrying}
        errorId={error ? errorId : undefined}
        status={
          knownStatus
            ? confirmedStatus === V2RecurringRunStatus.ENABLED
              ? 'Enabled'
              : 'Disabled'
            : 'Unknown'
        }
        onChange={changeEnabled}
      />
      {error &&
        createPortal(
          <div className={themeClassName}>
            <PageNotification
              duration={null}
              onClose={() => setState((current) => ({ ...current, error: '' }))}
              message={
                <span role='alert' id={errorId}>
                  {state.refreshFailed
                    ? `Schedule “${name}” updated, but the list could not refresh.`
                    : `Could not update schedule “${name}”. Its previous setting is unchanged.`}{' '}
                  <Button variant='secondary' size='sm' onClick={retry} disabled={retrying}>
                    Retry
                  </Button>{' '}
                  <NoticeDetails>{error}</NoticeDetails>
                </span>
              }
            />
          </div>,
          document.body,
        )}
    </div>
  );
}
