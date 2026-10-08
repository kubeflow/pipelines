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

import { useRef, useState } from 'react';
import { Switch } from '../ui/switch';
import './ExperimentWorkflows.css';

/** Serialize a schedule mutation and its refresh; callers own error presentation. */
export function ScheduleSwitch({
  name,
  checked,
  disabled,
  status,
  errorId,
  onChange,
}: {
  name: string;
  checked: boolean;
  disabled?: boolean;
  status: string;
  errorId?: string;
  onChange: (enabled: boolean) => Promise<void>;
}) {
  const inFlight = useRef(false);
  const [pending, setPending] = useState(false);
  const change = async (enabled: boolean) => {
    if (inFlight.current || disabled) return;
    inFlight.current = true;
    setPending(true);
    try {
      await onChange(enabled);
    } finally {
      inFlight.current = false;
      setPending(false);
    }
  };
  return (
    <div className='kfp-schedule-switch-row' onClick={(event) => event.stopPropagation()}>
      <Switch
        aria-label={`Enable schedule ${name}`}
        checked={checked}
        disabled={pending || disabled}
        aria-busy={pending}
        aria-describedby={errorId}
        onCheckedChange={change}
      />
      <span>{pending ? 'Updating…' : status}</span>
    </div>
  );
}
