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

import { useId, useState } from 'react';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
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
  status?: V2beta1RecurringRunStatus;
  onUpdated: () => Promise<void>;
}) {
  const errorId = useId();
  const [state, setState] = useState({
    id,
    observedStatus: status,
    confirmedStatus: status,
    error: '',
  });
  // A successful mutation is authoritative until the parent supplies a changed server status.
  // Reconcile during render so stale responses cannot update a different schedule's control.
  if (state.id !== id || state.observedStatus !== status) {
    setState({ id, observedStatus: status, confirmedStatus: status, error: '' });
  }
  const { confirmedStatus, error } = state;
  const knownStatus =
    confirmedStatus === V2beta1RecurringRunStatus.ENABLED ||
    confirmedStatus === V2beta1RecurringRunStatus.DISABLED;
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
          ? { ...current, error: `Unable to update schedule: ${message}` }
          : current,
      );
      return;
    }
    setState((current) =>
      current.id === id
        ? {
            ...current,
            confirmedStatus: enabled
              ? V2beta1RecurringRunStatus.ENABLED
              : V2beta1RecurringRunStatus.DISABLED,
          }
        : current,
    );
    try {
      await onUpdated();
    } catch (cause) {
      const message = await errorToMessage(cause);
      setState((current) =>
        current.id === id
          ? {
              ...current,
              error: `Schedule updated, but the list could not refresh: ${message}. Refresh the list to try again.`,
            }
          : current,
      );
    }
  };
  return (
    <div className='kfp-schedule-control' onClick={(event) => event.stopPropagation()}>
      <ScheduleSwitch
        name={name}
        checked={confirmedStatus === V2beta1RecurringRunStatus.ENABLED}
        disabled={!knownStatus || !id}
        errorId={error ? errorId : undefined}
        status={confirmedStatus || '-'}
        onChange={changeEnabled}
      />
      {error && (
        <p role='alert' id={errorId}>
          {error}
        </p>
      )}
    </div>
  );
}
